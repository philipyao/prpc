package server

import (
    "fmt"
    "errors"
    "sync"
    "net"
    "time"
    "io"
    "log"
    "reflect"
    "unicode"
    "strings"
    "unicode/utf8"
    "github.com/philipyao/prpc/codec"
    "github.com/philipyao/prpc/message"
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

func (s *service) call(server *Server, conn io.ReadWriteCloser, wg *sync.WaitGroup, mtype *methodType, reqmsg *message.Message, argv, replyv reflect.Value) {
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
    _ = errmsg
    //
    server.sendResponse(conn, reqmsg, replyv.Interface(),errmsg)
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
    rune, _ := utf8.DecodeRuneInString(name)
    return unicode.IsUpper(rune)
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
    group   string
    index   int

    serializer codec.Serializer

    serviceMap map[string]*service

    //todo registry

    listener    *net.TCPListener
}

func New(group string, index int, addr string) *Server {
    laddr, err := net.ResolveTCPAddr("tcp", addr)
    if err != nil {
        fmt.Printf("[rpc] ResolveTCPAddr() error: addr %v, err %v\n", addr, err)
        return nil
    }

    l, err := net.ListenTCP("tcp", laddr)
    if err != nil {
        fmt.Printf("[rpc ] rpc listen on %v, %v\n", laddr, err)
        return nil
    }

    return &Server{
        group: group,
        index: index,
        serializer: codec.GetSerializer(codec.SerializeTypeMsgpack),
        serviceMap: make(map[string]*service),
        listener: l,
    }
}

//设置打解包方法，默认msgpack
func (s *Server) SetCodec(styp codec.SerializeType) {
    //todo
}

//注册rpc处理
func (s *Server) Register(rcvr interface{}, name string) error {
    return s.register(rcvr, name)
}

func (s *Server) Serve(done chan struct{}, wg *sync.WaitGroup) {
    s.doServe(done, wg)
}

//========================================================================

func (server *Server) register(rcvr interface{}, name string) error {
    s := new(service)
    s.typ = reflect.TypeOf(rcvr)
    s.rcvr = reflect.ValueOf(rcvr)
    sname := name
    if sname == "" {
        s := "rpc.Register: no service name for type " + s.typ.String()
        log.Print(s)
        return errors.New(s)
    }
    if !isExported(sname) {
        s := "rpc.Register: type " + sname + " is not exported"
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
            str = "rpc.Register: type " + sname + " has no exported methods of suitable type (hint: pass a pointer to value of that type)"
        } else {
            str = "rpc.Register: type " + sname + " has no exported methods of suitable type"
        }
        log.Print(str)
        return errors.New(str)
    }

    if _, dup := server.serviceMap[sname]; dup {
        return errors.New("rpc: service already defined: " + sname)
    }
    return nil
}

func (s *Server) doServe(done chan struct{}, wg *sync.WaitGroup) {
    if wg != nil {
        defer wg.Done()
    }
    defer s.listener.Close()

    for {
        select {
        case <-done:
            log.Printf("[rpc] stop listening on %v...", s.listener.Addr())
            return
        default:
        }
        s.listener.SetDeadline(time.Now().Add(1e9))
        conn, err := s.listener.AcceptTCP()
        if err != nil {
            if opErr, ok := err.(*net.OpError); ok && opErr.Timeout() {
                continue
            }
            log.Printf("[rpc] Error: accept connection, %v", err.Error())
        }
        log.Printf("[rpc] accept connection: %v", conn)
        go s.serveConn(conn)
    }
}

func (s *Server) serveConn(conn io.ReadWriteCloser) {
    wg := new(sync.WaitGroup)
    for {
        reqmsg, err := message.NewResponse(conn)
        if err != nil {
            log.Printf("[rpc] Error: NewResponse %v", err)
            break
        }
        service, mtype, argv, replyv, err := s.unpackRequest(reqmsg)
        if err != nil {
            if err != io.EOF {
                log.Println("rpc:", err)
            }
            continue
        }
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
        err = errors.New("rpc: can't find service " + serviceName)
        return
    }
    mtype = service.method[methodName]
    if mtype == nil {
        err = errors.New("rpc: can't find method " + methodName)
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
    log.Printf("sendResponse: conn %v, reply %+v, seqno %v, method %v", conn, reply, reqmsg.Seqno(), reqmsg.ServiceMethod())
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
        log.Printf("pack error %v", err)
        return
    }
    log.Printf("pack ok, data len %v", len(data))
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