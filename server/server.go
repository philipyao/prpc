package server

import (
    "errors"
    "fmt"
    "io"
    "log"
    "net"
    "reflect"
    "strings"
    "sync"
    "time"
    "unicode"
    "unicode/utf8"

    "github.com/philipyao/prpc/codec"
    "github.com/philipyao/prpc/message"
    "github.com/philipyao/prpc/registry"
    "runtime"
    "bufio"
)

const (
    DefaultSrvIndexWeight = 10
    DefaultMsgPack        = codec.SerializeTypeMsgpack
    MaxReadSize           = 65535   //64k
)

// Precompute the reflect type for error. Can't use error directly
// because Typeof takes an empty interface value. This is annoying.
var typeOfError = reflect.TypeOf((*error)(nil)).Elem()

type methodType struct {
    sync.Mutex // protects counters
    method     reflect.Method
    ArgType    reflect.Type
    ReplyType  reflect.Type
    numCalls   uint
}

type service struct {
    name   string                 // name of service
    rcvr   reflect.Value          // receiver of methods for the service
    typ    reflect.Type           // type of the receiver
    method map[string]*methodType // registered methods
}

func init() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func (s *service) call(server *Server, conn io.ReadWriteCloser, wg *sync.WaitGroup,
    mtype *methodType, reqmsg *message.Message, argv, replyv reflect.Value) {
    if wg != nil {
        defer wg.Done()
    }
    mtype.Lock()
    mtype.numCalls++
    mtype.Unlock()
    function := mtype.method.Func
    // Invoke the method, providing a new value for the reply.
    returnValues := function.Call([]reflect.Value{s.rcvr, argv, replyv})
    // The return value for the method is an error.
    errInter := returnValues[0].Interface()
    errmsg := ""
    if errInter != nil {
        errmsg = errInter.(error).Error()
    }
    //
    server.sendResponse(conn, reqmsg, replyv.Interface(), errmsg)
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
    r, _ := utf8.DecodeRuneInString(name)
    return unicode.IsUpper(r)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
    for t.Kind() == reflect.Ptr {
        t = t.Elem()
    }
    // PkgPath will be non-empty even for an exported type,
    // so we need to check the type name as well.
    return isExported(t.Name()) || t.PkgPath() == ""
}

type Server struct {
    group string
    index int

    weight     int
    version    string
    styp       codec.SerializeType
    serializer codec.Serializer

    serviceMap map[string]*service

    //registry
    registry *registry.Registry

    listener *net.TCPListener

    done chan struct{}
    wg sync.WaitGroup
}

func New(group string, index int, opts ...FnOptionServer) *Server {
    srv := &Server{
        group:      group,
        index:      index,
        weight:     DefaultSrvIndexWeight,
        styp:       DefaultMsgPack,
        version:    registry.DefaultVersion,
        serviceMap: make(map[string]*service),
        done:       make(chan struct{}),
    }
    for _, opt := range opts {
        err := opt(srv)
        if err != nil {
            log.Printf("[prpc] err: decorate server: %v", err)
            return nil
        }
    }
    srv.serializer = codec.GetSerializer(srv.styp)
    if srv.serializer == nil {
        log.Printf("[prpc] err: unsupported styp %v", srv.styp)
        return nil
    }
    return srv
}

//注册rpc处理
func (s *Server) Handle(rcvr interface{}, name string) error {
    return s.handle(rcvr, name)
}

func (s *Server) Serve(addr string, regConfig interface{}) error {
    var reg *registry.Registry
    switch regConfig.(type) {
    case *registry.RegConfigZooKeeper:
        reg = registry.New(regConfig.(*registry.RegConfigZooKeeper).ZKAddr)
    default:
    }

    if reg == nil {
        return errors.New("[registry] invalid registry provided")
    }
    s.registry = reg

    laddr, err := net.ResolveTCPAddr("tcp", addr)
    if err != nil {
        return fmt.Errorf("[rpc] ResolveTCPAddr(): %v", err)
    }

    l, err := net.ListenTCP("tcp", laddr)
    if err != nil {
        return fmt.Errorf("[rpc] err: listen on %v, %v", laddr, err)
    }
    s.listener = l

    log.Println("[rpc] >> rpc service start to serve")
    log.Printf("[rpc] >> [args] group: <%v>", s.group)
    log.Printf("[rpc] >> [args] index: <%v>", s.index)
    log.Printf("[rpc] >> [args] addr:  <%v>", s.listener.Addr().String())
    for sname := range s.serviceMap {
        err := reg.Register(
            sname,
            s.group,
            s.index,
            s.listener.Addr().String(),
            registry.WithWeight(s.weight),
            registry.WithVersion(s.version),
            registry.WithSerialize(s.styp),
        )
        if err != nil {
            return fmt.Errorf("register %v err %v", sname, err)
        }
    }
    s.wg.Add(1)
    go s.doServe()

    return nil
}

func (s *Server) Fini() {
    log.Println("[rpc] try to stop service.")
    close(s.done)
    s.wg.Wait()

    log.Println("[rpc] finilize service...")
    for sname := range s.serviceMap {
        //注销服务
        log.Printf("[rpc] unregister service %v: %v.%v\n", sname, s.group, s.index)
        err := s.registry.Unregister(sname, s.group, s.index)
        if err != nil {
            log.Printf("[rpc][error] unregister err: %v\n", err)
        }
    }
    s.registry.Close()
}

//========================================================================

func (server *Server) handle(rcvr interface{}, name string) error {
    s := new(service)
    s.typ = reflect.TypeOf(rcvr)
    s.rcvr = reflect.ValueOf(rcvr)
    sname := name
    if sname == "" {
        s := "[rpc][error] rpc.Register: no service name for type " + s.typ.String()
        log.Print(s)
        return errors.New(s)
    }
    if !isExported(sname) {
        s := "[rpc][error] rpc.Register: type " + sname + " is not exported"
        log.Print(s)
        return errors.New(s)
    }
    s.name = sname

    // Install the methods
    s.method = suitableMethods(s.typ)

    if len(s.method) == 0 {
        str := ""

        // To help the user, see if a pointer receiver would work.
        method := suitableMethods(reflect.PtrTo(s.typ))
        if len(method) != 0 {
            str = "[rpc][error] rpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
        } else {
            str = "[rpc][error] rpc.Register: type " + sname + " has no exported methods of suitable type"
        }
        log.Print(str)
        return errors.New(str)
    }

    if _, dup := server.serviceMap[sname]; dup {
        return errors.New("[rpc][error] rpc: service already defined: " + sname)
    }
    server.serviceMap[sname] = s
    log.Printf("[rpc] handle with %v", sname)
    return nil
}

func (s *Server) doServe() {
    defer s.wg.Done()
    defer s.listener.Close()

    for {
        select {
        case <-s.done:
            log.Printf("[rpc] stop listening on %v...", s.listener.Addr())
            return
        default:
        }
        s.listener.SetDeadline(time.Now().Add(time.Second))
        conn, err := s.listener.AcceptTCP()
        if err != nil {
            if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
                continue
            }
            log.Printf("[rpc][error] accept connection, %v", err.Error())
            break
        }
        log.Printf("[rpc] accept new connection: %p", conn)
        go s.serveConn(conn)
    }
}

func (s *Server) serveConn(conn io.ReadWriteCloser) {
    defer func() {
        if err := recover(); err != nil {
            const size = 64 << 10
            buf := make([]byte, size)
            ss := runtime.Stack(buf, false)
            if ss > size {
                ss = size
            }
            buf = buf[:ss]
            log.Printf("panic: %v", buf)
        }
    }()

    wg := new(sync.WaitGroup)
    reader := bufio.NewReaderSize(conn, MaxReadSize)
    for {
        reqmsg, err := message.NewResponse(reader)
        if err != nil {
            if err != io.EOF {
                log.Printf("[rpc] err: NewResponse %v", err)
            }
            break
        }

        service, mtype, argv, replyv, err := s.unpackRequest(reqmsg)
        if err != nil {
            //if err != io.EOF {
                log.Printf("[rpc][error] unpackRequest: %v", err)
            //}
            continue
        }
        log.Printf("conn %p receive msg %v", conn, mtype)
        wg.Add(1)
        go service.call(s, conn, wg, mtype, reqmsg, argv, replyv)
    }
    // We've seen that there are no more requests.
    // Wait for responses to be sent before closing codec.
    wg.Wait()
    log.Printf("[rpc] conn %v end", conn)
    conn.Close()
}

func (s *Server) unpackRequest(msg *message.Message) (service *service, mtype *methodType, argv, replyv reflect.Value, err error) {
    if msg.IsHeartbeat() {
        //todo
        panic("heartbeat")
    }
    serviceMethod := msg.ServiceMethod()
    dot := strings.LastIndex(serviceMethod, ".")
    if dot < 0 {
        err = errors.New("rpc: service/method request ill-formed: " + serviceMethod)
        return
    }
    serviceName := serviceMethod[:dot]
    methodName := serviceMethod[dot+1:]

    // Look up the request.
    service, ok := s.serviceMap[serviceName]
    if !ok {
        err = errors.New("[rpc] can't find service " + serviceName)
        return
    }
    mtype = service.method[methodName]
    if mtype == nil {
        err = errors.New("[rpc] can't find method " + methodName)
        return
    }

    // Decode the argument value.
    argIsValue := false // if true, need to indirect before calling.
    if mtype.ArgType.Kind() == reflect.Ptr {
        argv = reflect.New(mtype.ArgType.Elem())
    } else {
        argv = reflect.New(mtype.ArgType)
        argIsValue = true
    }
    // argv guaranteed to be a pointer now.
    err = msg.Unpack(s.serializer, argv.Interface())
    if err != nil {
        log.Printf("[rpc][error] Unpack: %v", err)
        return
    }
    if argIsValue {
        argv = argv.Elem()
    }

    replyv = reflect.New(mtype.ReplyType.Elem())

    switch mtype.ReplyType.Elem().Kind() {
    case reflect.Map:
        replyv.Elem().Set(reflect.MakeMap(mtype.ReplyType.Elem()))
    case reflect.Slice:
        replyv.Elem().Set(reflect.MakeSlice(mtype.ReplyType.Elem(), 0, 0))
    }
    return
}

func (s *Server) sendResponse(conn io.ReadWriteCloser, reqmsg *message.Message, reply interface{}, errmsg string) {
    log.Printf("[rpc] sendResponse: conn %v, reply %+v, seqno %v, method %v",
        conn, reply, reqmsg.Seqno(), reqmsg.ServiceMethod())
    pkg := message.NewRequest(message.MsgKindDefault, reqmsg.Seqno())
    // Encode the response header
    if errmsg != "" {
        //todo
        panic("errmsg")
        reply = struct{}{}
    }
    data, err := pkg.Pack(reqmsg.ServiceMethod(), reply, s.serializer)
    if err != nil {
        //todo
        log.Printf("[rpc] pack error: %v", err)
        return
    }
    //todo write timeout
    conn.Write(data)
}

// suitableMethods returns suitable Rpc methods of typ
func suitableMethods(typ reflect.Type) map[string]*methodType {
    methods := make(map[string]*methodType)
    for m := 0; m < typ.NumMethod(); m++ {
        method := typ.Method(m)
        mtype := method.Type
        mname := method.Name
        // Method must be exported.
        if method.PkgPath != "" {
            continue
        }
        // Method needs three ins: receiver, *args, *reply.
        if mtype.NumIn() != 3 {
            log.Printf("rpc.Register: method %q has %d input parameters; needs exactly three\n", mname, mtype.NumIn())
            continue
        }
        // First arg need not be a pointer.
        argType := mtype.In(1)
        if !isExportedOrBuiltinType(argType) {
            log.Printf("rpc.Register: argument type of method %q is not exported: %q\n", mname, argType)
            continue
        }
        // Second arg must be a pointer.
        replyType := mtype.In(2)
        if replyType.Kind() != reflect.Ptr {
            log.Printf("rpc.Register: reply type of method %q is not a pointer: %q\n", mname, replyType)
            continue
        }
        // Reply type must be exported.
        if !isExportedOrBuiltinType(replyType) {
            log.Printf("rpc.Register: reply type of method %q is not exported: %q\n", mname, replyType)
            continue
        }
        // Method needs one out.
        if mtype.NumOut() != 1 {
            log.Printf("rpc.Register: method %q has %d output parameters; needs exactly one\n", mname, mtype.NumOut())
            continue
        }
        // The return type of the method must be error.
        if returnType := mtype.Out(0); returnType != typeOfError {
            log.Printf("rpc.Register: return type of method %q is %q, must be error\n", mname, returnType)
            continue
        }
        methods[mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType}
    }
    return methods
}
