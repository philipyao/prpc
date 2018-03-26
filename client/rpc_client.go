package client

import (
    "net"
)


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

    ip      string
    conn    *net.Conn

    //todo codec

    mutex    sync.Mutex // protects following
    seq      uint64
    pending  map[uint64]*Call
    closing  bool // user has called Close
    shutdown bool // server has told us to stop
}

func New() *RPCClient {
    return &RPCClient{}
}

func (rc *RPCClient) Call() {

}

func (rc *RPCClient) Go() {

}

//==========================================================================

func (rc *RPCClient) send() {

}

func (rc *RPCClient) input() {

}
