package client

import (
    "errors"
    "net"
    "log"
    "io"
    "fmt"
    "sync"
    "strings"
    "strconv"

    "github.com/philipyao/prpc/codec"
    "github.com/philipyao/prpc/message"
    "github.com/philipyao/toolbox/zkcli"
)

const (
    DefaultServiceWeight        = 1
    NodeValNumber               = 3
)

var ErrShutdown = errors.New("connection is shut down")
var ErrNetClosing = errors.New("use of closed network connection")
type CBFn func(a interface{}, r interface{}, e error)

// Call represents an active RPC.
type Call struct {
    ServiceMethod string      // The name of the service and method to call.
    Args          interface{} // The argument to the function (*struct).
    Reply         interface{} // The reply from the function (*struct).
    Error         error       // After completion, the error status.
    Done          chan *Call  // Strobes when call is complete.
    fn            CBFn
}
func (call *Call) done() {
    select {
    case call.Done <- call:
        // ok
    default:
        // We don't want to block here. It is the caller's responsibility to make
        // sure the channel has enough buffer space. See comment in Go().
    }
}


type RPCClient struct {
    //data watcher, service数据发生改变
    zk      *zkcli.Conn
    path    string
    watcher *watcher

    //service 数据
    nodeVal string
    addr    string
    weight  int
    styp    codec.SerializeType

    conn    net.Conn
    serializer codec.Serializer

    mutex    sync.Mutex // protects following
    seq      uint16
    pending  map[uint16]*Call
    closing  bool // user has called Close
    shutdown bool // server has told us to stop

    //todo inservice 检测本rpc依赖的dependency是否ok
}

func newRPC(zk *zkcli.Conn, path, nodeVal string) *RPCClient {
    //nodeVal: ip | styp | weight
    if len(nodeVal) == 0 {
        log.Println("empty nodeVal")
        return nil
    }
    slices := strings.Split(nodeVal, "|")
    if len(slices) != NodeValNumber {
        log.Printf("mismatch nodeVal: %v", nodeVal)
        return nil
    }
    addr := strings.TrimSpace(slices[0])
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        log.Printf("conn to server<%v> error: %v", addr, err)
        return nil
    }
    slices[1] = strings.TrimSpace(slices[1])
    tp, err := strconv.Atoi(slices[1])
    if err != nil {
        log.Printf("invalid styp: %v %v", slices[1], err)
        return nil
    }
    styp := codec.SerializeType(tp)
    serializer := codec.GetSerializer(styp)
    if serializer == nil {
        log.Printf("styp %v not support", styp)
        return nil
    }
    slices[2] = strings.TrimSpace(slices[2])
    weight, err := strconv.Atoi(slices[2])
    if err != nil {
        log.Printf("invalid weight: %v %v", slices[2], err)
        return nil
    }
    if weight < 0 {
        log.Printf("nagtive weight: %v", weight)
        return nil
    }
    if weight == 0 {
        log.Printf("0 weight: %v", weight)
        return nil
    }

    client := &RPCClient{
        zk: zk,
        path: path,
        nodeVal: nodeVal,
        addr: addr,
        conn: conn,
        styp: styp,
        weight: weight,
        serializer: serializer,
        pending: make(map[uint16]*Call),
    }

    go client.input()
    go client.watch()
    go client.heartbeat()
    return client
}

func (rc *RPCClient) NodeVal() string {
    return rc.nodeVal
}

func (rc *RPCClient) Path() string {
    return rc.path
}

func (rc *RPCClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
    call := <- rc.doCall(serviceMethod, args, reply)
    return call.Error
}

func (rc *RPCClient) doCall(serviceMethod string, args interface{}, reply interface{}) chan *Call {
    call := new(Call)
    call.ServiceMethod = serviceMethod
    call.Args = args
    call.Reply = reply
    call.Done = make(chan *Call, 10)
    rc.send(call)

    return call.Done
}

func (rc *RPCClient) Go(serviceMethod string, args interface{}, reply interface{}, fn CBFn) {
    call := <- rc.doCall(serviceMethod, args, reply)
    fn(call.Args, call.Reply, call.Error)
}

func (rc *RPCClient) Close() error {
    rc.mutex.Lock()
    if rc.closing {
        rc.mutex.Unlock()
        return ErrShutdown
    }
    rc.closing = true
    rc.mutex.Unlock()
    //return rc.codec.Close()

    //关闭tcpconn
    rc.conn.Close()

    //stop监听
    if rc.watcher != nil {
        rc.watcher.Stop()
    }

    return nil
}

//==========================================================================

func (rc *RPCClient) send(call *Call) {
    // Register this call.
    rc.mutex.Lock()
    if rc.shutdown || rc.closing {
        call.Error = ErrShutdown
        rc.mutex.Unlock()
        call.done()
        return
    }
    seq := rc.seq
    rc.seq++
    if rc.seq == 0 {
        rc.seq = 1
    }
    rc.pending[seq] = call
    rc.mutex.Unlock()

    // Encode and send the request.
    pkg := message.NewRequest(message.MsgKindDefault, seq)
    data, err := pkg.Pack(call.ServiceMethod, call.Args, rc.serializer)
    if err != nil {
        //todo
        log.Printf("pack error %v", err)
        rc.mutex.Lock()
        call = rc.pending[seq]
        delete(rc.pending, seq)
        rc.mutex.Unlock()
        if call != nil {
            call.Error = err
            call.done()
        }
        return
    }
    log.Printf("pack ok, data len %v", len(data))
    rc.conn.Write(data)
}

func (rc *RPCClient) watch() {
    if rc.watcher == nil {
        rc.watcher = newWatcher(rc.zk, rc.path)
        err := rc.watcher.Watch(func(p string, d []byte, e error){
            //todo
        })
        if err != nil {
            log.Printf("watch error %v", err)
        }
    }
}

func (rc *RPCClient) input() {
    var err error
    //var response Response
    for err == nil {
        rmsg, err := message.NewResponse(rc.conn)
        if err != nil {
            if err != ErrNetClosing {
                log.Printf("NewResponse error: %v", err)
            }
            break
        }
        if rmsg.IsHeartbeat() {
            //todo 处理rpc心跳
            log.Println("heartbeat received")
            continue
        }

        //开始处理rpc返回
        seq := rmsg.Seqno()
        //rc.mutextex.Lock()
        call := rc.pending[seq]
        delete(rc.pending, seq)
        //rc.mutextex.Unlock()

        switch {
        case call == nil:
            // We've got no pending call.
            log.Printf("rpc request seqno<%v> not found", seq)
            continue

        //case response.Error != "":
        //    // We've got an error response. Give this to the request;
        //    // any subsequent requests will get the ReadResponseBody
        //    // error if there is one.
        //    call.Error = ServerError(response.Error)
        //    err = rc.codec.ReadResponseBody(nil)
        //    if err != nil {
        //        err = errors.New("reading error body: " + err.Error())
        //    }
        //    call.done()

        default:
            if rmsg.ServiceMethod() != call.ServiceMethod {
                call.Error = fmt.Errorf("response method mismatch: %v %v" + rmsg.ServiceMethod(), call.ServiceMethod)
                call.done()
                continue
            }
            err = rmsg.Unpack(rc.serializer, call.Reply)
            if err != nil {
                call.Error = errors.New("unpacking body " + err.Error())
            }
            call.done()
        }
    }

    // Terminate pending calls.
    //rc.reqMutex.Lock()
    //rc.mutextex.Lock()
    rc.shutdown = true
    closing := rc.closing
    if err == io.EOF {
        if closing {
            err = ErrShutdown
        } else {
            err = io.ErrUnexpectedEOF
        }
    }
    for _, call := range rc.pending {
        call.Error = err
        call.done()
    }
    //rc.mutextex.Unlock()
    //rc.reqMutex.Unlock()
    if err != io.EOF && !closing {
        log.Println("rpc: client protocol error:", err)
    }
}

func (rc *RPCClient) heartbeat() {

}
