package server

import (
    //"sync"
    "net"
    "time"
    "io"
    "github.com/philipyao/prpc/codec"
)

type service struct {
    dummy int
}

type Server struct {
    group   string
    index   int

    serializer codec.Serializer

    services map[string]service

    //todo registry

    listener    *net.TCPListener
}

func New(group, index, addr string) *Server {
    //laddr, err := net.ResolveTCPAddr("tcp", addr)
    //if err != nil {
    //    errMsg = fmt.Sprintf("[rpc] ResolveTCPAddr() error: addr %v, err %v", addr, err)
    //    return nil
    //}
    //
    //l, err := net.ListenTCP("tcp", laddr)
    //if err != nil {
    //    errMsg = fmt.Sprintf("[rpc ] rpc listen on %v, %v", laddr, err)
    //    return nil
    //}

    return nil
}

//设置打解包方法，默认msgpack
func (s *Server) SetCodec(styp codec.SerializeType) {

}

//注册rpc处理
func (s *Server) Register(name string, rcvr interface{}) error {
    return nil
}

func (s *Server) Serve(done chan struct{}, wg *sync.WaitGroup) {
    s.doServe(done, wg)
}

//========================================================================

func (s *Server) doServe(done chan struct{}, wg *sync.WaitGroup) {
    if wg != nil {
        defer wg.Done()
    }
    defer s.listener.Close()

    for {
        select {
        case <-done:
            //if s.logFunc != nil {
            //    s.logFunc("[rpc] stop listening on %v...", s.listener.Addr())
            //}
            return
        default:
        }
        s.listener.SetDeadline(time.Now().Add(1e9))
        conn, err := s.listener.AcceptTCP()
        if err != nil {
            if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
                continue
            }
            //if s.logFunc != nil {
            //    s.logFunc("[rpc] Error: accept connection, %v", err.Error())
            //}
        }
        go s.serveConn(conn)
    }
}

func (s *Server) serveConn(conn io.ReadWriteCloser) {

}