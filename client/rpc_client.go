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
    "context"
    "time"
)

var ErrShutdown = errors.New("shut down")
var ErrBeClosed = errors.New("connection closed by peer")
var ErrNetClosing = errors.New("use of closed network connection")

const (
    BuffSizeReader      = 64 * 1024     //64K
    //BuffSizeWriter      = 64 * 1024     //64k

    ReadTimeout         = 5 * time.Second
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
    Seq           uint16
}

func (call *Call) done() {
    select {
    case call.Done <- call:
        // ok
    default:
        // We don't want to block here. It is the caller's responsibility to make
        // sure the channel has enough buffer space. See comment in Go().
        log.Println("[prpc][ERROR] call done capcity out of needs")
    }
}

type RPCClient struct {
    conn       net.Conn
    reader     *bufio.Reader    //带缓冲读取
    serializer codec.Serializer

    mutex    sync.Mutex // protects following
    seq      uint16
    pending  map[uint16]*Call
    closing  bool // user has called Close
    shutdown chan struct{} // server has told us to stop

    wg sync.WaitGroup

    //todo inservice 检测本rpc依赖的dependency是否ok
}

func newRPCClient(addr string, styp codec.SerializeType) *RPCClient {
    serializer := codec.GetSerializer(styp)
    if serializer == nil {
        log.Printf("[prpc][ERROR] styp %v not support", styp)
        return nil
    }
    addr = strings.TrimSpace(addr)
    conn, err := net.Dial("tcp", addr)
    if err != nil {
        log.Printf("[prpc][ERROR] conn to rpc server<%v> error %v", addr, err)
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
func (rc *RPCClient) Call(ctx context.Context, serviceMethod string, args interface{}, reply interface{}) error {
    rc.mutex.Lock()
    rc.seq++
    if rc.seq == 0 {
        rc.seq = 1
    }
    seq := rc.seq
    rc.mutex.Unlock()
    done := rc.doCall(seq, serviceMethod, args, reply)
    select {
    case <- rc.shutdown:
        log.Println("[prpc] call encounter shutdown")
        return ErrShutdown
    case <- ctx.Done():     //cancelled by caller
        rc.mutex.Lock()
        delete(rc.pending, seq)
        rc.mutex.Unlock()
        log.Printf("[prpc] call canceled by context: %v", ctx.Err())
        return ctx.Err()
    case call := <- done:
        log.Printf("[prpc] call rsp: err %v", call.Error)
        return call.Error
    }
}

//异步非阻塞调用
func (rc *RPCClient) Go(ctx context.Context, serviceMethod string, args interface{}, reply interface{}, fn FnCallback) {
    rc.mutex.Lock()
    rc.seq++
    if rc.seq == 0 {
        rc.seq = 1
    }
    seq := rc.seq
    rc.mutex.Unlock()
    done := rc.doCall(seq, serviceMethod, args, reply)
    rc.wg.Add(1)
    go func() {
        defer rc.wg.Done()

        select {
        case <- rc.shutdown:
            log.Println("[prpc] Go() encounter shutdown")
            return
        case <- ctx.Done():             //cancelled by caller
            log.Println("[prpc] Go() canceled by caller")
            rc.mutex.Lock()
            delete(rc.pending, seq)
            rc.mutex.Unlock()
            fn(args, reply, ctx.Err())
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
    //关闭tcpconn
    rc.conn.Close()

    log.Println("[prpc] call shutdown, wait")
    rc.wg.Wait()
    log.Println("[prpc] wait ok")


    return nil
}

//==========================================================================

func (rc *RPCClient) doCall(seq uint16, serviceMethod string, args interface{}, reply interface{}) chan *Call {
    call := new(Call)
    call.ServiceMethod = serviceMethod
    call.Args = args
    call.Reply = reply
    call.Done = make(chan *Call, 10)
    call.Seq = seq
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

    //call.Seq = seq
    rc.pending[call.Seq] = call
    rc.mutex.Unlock()

    // Encode and send the request.
    //todo msg pool
    pkg := message.NewRequest(message.MsgKindDefault, call.Seq)
    data, err := pkg.Pack(call.ServiceMethod, call.Args, rc.serializer)
    if err != nil {
        //todo
        log.Printf("[prpc][ERROR] pack error %v", err)
        rc.mutex.Lock()
        call = rc.pending[call.Seq]
        delete(rc.pending, call.Seq)
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
        log.Printf("[prpc][ERROR] conn write error %v", err)
        rc.mutex.Lock()
        call = rc.pending[call.Seq]
        delete(rc.pending, call.Seq)
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
    var rmsg *message.Message
    for err == nil {
        select {
        case <- rc.shutdown:
            log.Println("[prpc] input() encounter shutdown, return")
            return
        default:
        }
        // read timeout SetReadDeadline
        rc.conn.SetReadDeadline(time.Now().Add(ReadTimeout))
        rmsg, err = message.NewResponse(rc.reader)
        if err != nil {
            if err != io.EOF {
                if strings.Contains(err.Error(), ErrNetClosing.Error()) {
                    err = ErrShutdown
                } else {
                    if ne, ok := err.(net.Error); ok {
                        if ne.Timeout() == true {
                            //log.Println("read timeout, continue")
                            err = nil
                            continue
                        }
                    }
                    log.Printf("[prpc][ERROR] read response msg error: %v", err)
                }
            }
            break
        }
        if rmsg.IsHeartbeat() {
            //todo 处理rpc心跳
            log.Println("[prpc] heartbeat received")
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
            log.Printf("[prpc] rpc request seqno<%v> not found", seq)
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
    log.Printf("[prpc] stop rpc_client input(): %v", err)
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
        log.Println("[prpc][ERROR] rpc: client protocol error:", err)
    }
}

func (rc *RPCClient) heartbeat() {
    defer rc.wg.Done()

    //todo
}
