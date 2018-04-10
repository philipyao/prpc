package client

import (
    "errors"
    "net"
    "log"
    "sync"
    "strings"
    "strconv"

    "github.com/philipyao/prpc/codec"
    "github.com/philipyao/toolbox/zkcli"
)

const (
    DefaultServiceWeight        = 1
    NodeValNumber               = 3
)

var ErrShutdown = errors.New("connection is shut down")
type CBFn func(a interface{}, r interface{}, e error)

// Call represents an active RPC.
type Call struct {
    ServiceMethod string      // The name of the service and method to call.
    Args          interface{} // The argument to the function (*struct).
    Reply         interface{} // The reply from the function (*struct).
    Error         error       // After completion, the error status.
    Done          chan *Call  // Strobes when call is complete.
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
    //todo data watcher, service数据发生改变

    //service 数据
    path    string
    nodeVal string
    zk      *zkcli.Conn
    addr    string
    weight  int
    styp    codec.SerializeType

    conn    *net.Conn
    serializer codec.Serializer

    mutex    sync.Mutex // protects following
    seq      uint64
    pending  map[uint64]*Call
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
    styp, err := strconv.Atoi(slices[1])
    if err != nil {
        log.Printf("invalid styp: %v %v", slices[1], err)
        return nil
    }
    serializer := codec.GetSerializer(codec.SerializeType(styp))
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
        pending: make(map[uint64]*Call),
    }

    go client.input()
    go client.watch()
    go client.heartbeat()
    return client
}

func (rc *RPCClient) NodeVal() string {
    return rc.nodeVal
}

func (rc *RPCClient) Call(serviceMethod string, args interface{}, reply interface{}) {

}

func (rc *RPCClient) Go(serviceMethod string, args interface{}, reply interface{}, fn CBFn) error {
    //call := new(Call)
    //call.ServiceMethod = serviceMethod
    //call.Args = args
    //call.Reply = reply
    //if done == nil {
    //    done = make(chan *Call, 10) // buffered.
    //} else {
    //    // If caller passes done != nil, it must arrange that
    //    // done has enough buffer for the number of simultaneous
    //    // RPCs that will be using that channel. If the channel
    //    // is totally unbuffered, it's best not to run at all.
    //    if cap(done) == 0 {
    //        log.Panic("rpc: done channel is unbuffered")
    //    }
    //}
    //call.Done = done
    //client.send(call)
    //return call

    return nil
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
    //stop监听

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
    rc.pending[seq] = call
    rc.mutex.Unlock()

    // Encode and send the request.
    //rc.request.Seq = seq
    //rc.request.ServiceMethod = call.ServiceMethod
    //err := rc.codec.WriteRequest(&rc.request, call.Args)
    //if err != nil {
    //    rc.mutex.Lock()
    //    call = rc.pending[seq]
    //    delete(rc.pending, seq)
    //    rc.mutex.Unlock()
    //    if call != nil {
    //        call.Error = err
    //        call.done()
    //    }
    //}
}

func (rc *RPCClient) watch() {

}

func (rc *RPCClient) input() {
    //var err error
    //var response Response
    //for err == nil {
    //    response = Response{}
    //    err = rc.codec.ReadResponseHeader(&response)
    //    if err != nil {
    //        break
    //    }
    //    seq := response.Seq
    //    rc.mutextex.Lock()
    //    call := rc.pending[seq]
    //    delete(rc.pending, seq)
    //    rc.mutextex.Unlock()
    //
    //    switch {
    //    case call == nil:
    //        // We've got no pending call. That usually means that
    //        // WriteRequest partially failed, and call was already
    //        // removed; response is a server telling us about an
    //        // error reading request body. We should still attempt
    //        // to read error body, but there's no one to give it to.
    //        err = rc.codec.ReadResponseBody(nil)
    //        if err != nil {
    //            err = errors.New("reading error body: " + err.Error())
    //        }
    //    case response.Error != "":
    //        // We've got an error response. Give this to the request;
    //        // any subsequent requests will get the ReadResponseBody
    //        // error if there is one.
    //        call.Error = ServerError(response.Error)
    //        err = rc.codec.ReadResponseBody(nil)
    //        if err != nil {
    //            err = errors.New("reading error body: " + err.Error())
    //        }
    //        call.done()
    //    default:
    //        err = rc.codec.ReadResponseBody(call.Reply)
    //        if err != nil {
    //            call.Error = errors.New("reading body " + err.Error())
    //        }
    //        call.done()
    //    }
    //}
    // Terminate pending calls.
    //rc.reqMutex.Lock()
    //rc.mutextex.Lock()
    //rc.shutdown = true
    //closing := rc.closing
    //if err == io.EOF {
    //    if closing {
    //        err = ErrShutdown
    //    } else {
    //        err = io.ErrUnexpectedEOF
    //    }
    //}
    //for _, call := range rc.pending {
    //    call.Error = err
    //    call.done()
    //}
    //rc.mutextex.Unlock()
    //rc.reqMutex.Unlock()
    //if debugLog && err != io.EOF && !closing {
    //    log.Println("rpc: client protocol error:", err)
    //}
}

func (rc *RPCClient) heartbeat() {

}
