package client

import (
    "errors"
    "fmt"
    "io"
    "log"
    "net"
    "strings"
    "sync"

    "github.com/philipyao/prpc/codec"
    "github.com/philipyao/prpc/message"
    "bufio"
)

var ErrShutdown = errors.New("connection is shut down")
var ErrBeClosed = errors.New("connection closed by peer")
//var ErrNetClosing = errors.New("use of closed network connection")

const (
    BuffSizeReader      = 16 * 1024     //16K
    BuffSizeWriter      = 16 * 1024     //16k
)

type FnCallback func(a interface{}, r interface{}, e error)

// Call represents an active RPC.
type Call struct {
    ServiceMethod string      // The name of the service and method to call.
    Args          interface{} // The argument to the function (*struct).
    Reply         interface{} // The reply from the function (*struct).
    Error         error       // After completion, the error status.
    Done          chan *Call  // Strobes when call is complete.
    fn            FnCallback
}

func (call *Call) done() {
    select {
    case call.Done <- call:
        // ok
    default:
        // We don't want to block here. It is the caller's responsibility to make
        // sure the channel has enough buffer space. See comment in Go().
        log.Println("call done capcity out of needs")
    }
}

type RPCClient struct {
    conn       net.Conn
    reader     *bufio.Reader
    serializer codec.Serializer

    mutex    sync.Mutex // protects following
    seq      uint16
    pending  map[uint16]*Call
    closing  bool // user has called Close
    shutdown chan struct{} // server has told us to stop

    wg sync.WaitGroup

    //todo inservice 检测本rpc依赖的dependency是否ok
}

func newRPC(ep *endPoint) *RPCClient {
    styp := ep.styp
    serializer := codec.GetSerializer(styp)
    if serializer == nil {
        log.Printf("styp %v not support", styp)
        return nil
    }
    addr := strings.TrimSpace(ep.addr)
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        log.Printf("conn to rpc server<%v> error %v", addr, err)
        return nil
    }
    client := &RPCClient{
        conn:       conn,
        reader:     bufio.NewReaderSize(conn, BuffSizeReader),
        serializer: serializer,
        pending:    make(map[uint16]*Call),
        shutdown:   make(chan struct{}),
    }

    client.wg.Add(1)
    go client.input()

    client.wg.Add(1)
    go client.heartbeat()

    return client
}

//同步阻塞调用
func (rc *RPCClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
    call := <-rc.doCall(serviceMethod, args, reply)
    return call.Error
}

//异步非阻塞调用
func (rc *RPCClient) Go(serviceMethod string, args interface{}, reply interface{}, fn FnCallback) {
    done := rc.doCall(serviceMethod, args, reply)
    rc.wg.Add(1)
    go func() {
        defer rc.wg.Done()

        select {
        case <- rc.shutdown:
            return
        case call := <- done:
            fn(call.Args, call.Reply, call.Error)
            return
        }
    }()
}

func (rc *RPCClient) Close() error {
    rc.mutex.Lock()
    if rc.closing {
        rc.mutex.Unlock()
        return ErrShutdown
    }
    rc.closing = true
    rc.mutex.Unlock()

    close(rc.shutdown)
    rc.wg.Wait()

    //关闭tcpconn
    rc.conn.Close()

    return nil
}

//==========================================================================

func (rc *RPCClient) doCall(serviceMethod string, args interface{}, reply interface{}) chan *Call {
    call := new(Call)
    call.ServiceMethod = serviceMethod
    call.Args = args
    call.Reply = reply
    call.Done = make(chan *Call, 10)
    rc.send(call)

    return call.Done
}

func (rc *RPCClient) send(call *Call) {
    // Register this call.
    rc.mutex.Lock()
    if rc.closing {
        call.Error = ErrShutdown
        rc.mutex.Unlock()
        call.done()
        return
    }
    rc.seq++
    if rc.seq == 0 {
        rc.seq = 1
    }
    seq := rc.seq
    rc.pending[seq] = call
    rc.mutex.Unlock()

    // Encode and send the request.
    //todo msg pool
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
    //log.Printf("pack ok, data len %v", len(data))

    //todo write timeout SetWriteDeadline
    _, err = rc.conn.Write(data)
    if err != nil {
        log.Printf("conn write error %v", err)
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

    return
}

// 处理收包逻辑
func (rc *RPCClient) input() {
    defer rc.wg.Done()

    var err error
    //var response Response
    var rmsg *message.Message
    for err == nil {
        select {
        case <- rc.shutdown:
            return
        default:
        }
        //todo read timeout SetReadDeadline
        rmsg, err = message.NewResponse(rc.reader)
        if err != nil {
            if /*err != ErrNetClosing &&*/ err != io.EOF {
                log.Printf("read response msg error: %v", err)
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
        rc.mutex.Lock()
        call := rc.pending[seq]
        delete(rc.pending, seq)
        rc.mutex.Unlock()

        switch {
        case call == nil:
            // We've got no pending call.
            log.Printf("rpc request seqno<%v> not found", seq)
            continue
        default:
            if rmsg.ServiceMethod() != call.ServiceMethod {
                call.Error = fmt.Errorf("response method mismatch: %v %v"+rmsg.ServiceMethod(), call.ServiceMethod)
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
    log.Printf("stop rpc_client input(): err %v", err)
    rc.mutex.Lock()
    closing := rc.closing
    rc.mutex.Unlock()
    if err == io.EOF {
        if closing {
            //主动关闭
            err = ErrShutdown
        } else {
            //对端关闭
            err = ErrBeClosed
        }
    }
    //log.Printf("stop rpc_client input: err %v", err)
    rc.mutex.Lock()
    for _, call := range rc.pending {
        call.Error = err
        call.done()
    }
    rc.mutex.Unlock()
    if err != io.EOF && !closing {
        log.Println("rpc: client protocol error:", err)
    }
}

func (rc *RPCClient) heartbeat() {
    //todo
    defer rc.wg.Done()
}
