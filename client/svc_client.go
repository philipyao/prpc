package client

import (
    "bytes"
    "crypto/sha256"
    "encoding/binary"
    "errors"
    "fmt"
    "github.com/philipyao/prpc/codec"
    "github.com/philipyao/prpc/registry"
    "github.com/afex/hystrix-go/hystrix"
    "log"
    "sync"
    "context"
    "reflect"
)

const (
    noSpecifiedVersion = ""
    noSpecifiedIndex   = -1
)

type Args struct {
    A, B int
}

type endPoint struct {
    key string

    index   int
    weight  int
    version string
    styp    codec.SerializeType
    addr    string
    conn    *RPCClient

    lock sync.Mutex         //protect following
    callTimes uint32
    //failtimes
}

//检查注册中心的数据发生变化后，可变数据(可在线更新的)是否发生改变
//不检查不可变数据，如index，addr等
//如果服务提供方的styp addr要修改，肯定会停掉服务，修改然后重新注册的，不会走到这里
func (ep *endPoint) equalTo(node *registry.Node) bool {
    return ep.weight == node.Weight &&
        ep.version == node.Version
}

//更新可变数据(可在线更新的)
func (ep *endPoint) update(node *registry.Node) {
    ep.weight = node.Weight
    ep.version = node.Version
}

type SvcClient struct {
    group   string
    service string

    //options
    version    string     //选择特定版本
    index      int        //选择特定index的endpoint
    selectType selectType //选取算法

    selector  selector //选择器
    endPoints []*endPoint

    //断路器是否初始化: command -> isInit
    breakerLock sync.RWMutex
    breakers map[string]bool

    statLock sync.Mutex
    reqTimes uint64
    succTimes uint64

    registry *registry.Registry
}

func (sc *SvcClient) Subscribe() error {
    //获取endpoints, watch endpoints的变化
    nodes, err := sc.registry.Subscribe(sc.service, sc.group, sc)
    if err != nil {
        log.Printf("[prpc][ERROR] suscribe err: %v", err)
        return nil
    }
    if len(nodes) > 0 {
        sc.addEndpoint(nodes)
    }
    return nil
}
func (sc *SvcClient) Call(serviceMethod string, args interface{}, reply interface{}) error {
    val := reflect.ValueOf(reply)
    if val.Kind() != reflect.Ptr || val.IsNil(){
        return errors.New("reply should be pointer and not nil")
    }

    //todo 过滤机制
    defer func() {
       if r := recover(); r != nil {
           log.Printf("[prpc][ERROR] recover: %v\n", r)
       }
    }()

    sc.statLock.Lock()
    sc.reqTimes++
    sc.statLock.Unlock()

    //初始化熔断器
    breakerName := fmt.Sprintf("%v-%v-%v", sc.group, sc.service, serviceMethod)
    sc.breakerLock.RLock()
    _, exist := sc.breakers[breakerName];
    sc.breakerLock.RUnlock()
    if  !exist {
        hystrix.ConfigureCommand(breakerName, hystrix.CommandConfig{
            Timeout:                2000,       //函数执行2s超时
            MaxConcurrentRequests:  50000,        //QPS
            SleepWindow:            5000,       //5s
            RequestVolumeThreshold: 10,
            ErrorPercentThreshold:  20,         //20%
        })
        sc.breakerLock.Lock()
        sc.breakers[breakerName] = true
        sc.breakerLock.Unlock()
    }

    // 加断路器来调用run函数：控制超时，错误熔断，提供过载保护
    output := make(chan error, 1)
    errors := hystrix.Go(breakerName, func() error {
        output <- sc.doCall(serviceMethod, args, reply)
        return nil
    }, func(e error) error {
        log.Printf("[prpc][ERROR] In fallback function for breaker %v, error: %v", breakerName, e.Error())
        circuit, _, _ := hystrix.GetCircuit(breakerName)
        log.Printf("[prpc][ERROR] Circuit state is: %v", circuit.IsOpen())
        return e
    })
    // Response and error handling. If the call was successful, the output channel gets the response. Otherwise,
    // the errors channel gives us the error.
    // blocking wait here
    select {
    case out := <-output:
        sc.statLock.Lock()
        sc.succTimes++
        sc.statLock.Unlock()
        return out
    case err := <-errors:
        return err
    }
}

func (sc *SvcClient) doCall(serviceMethod string, args interface{}, reply interface{}) error {
    var ep *endPoint
    if sc.index >= 0 {
        //指定固定的index
        for _, e := range sc.endPoints {
            if e.index == sc.index {
                ep = e
                break
            }
        }
        if ep == nil {
            return fmt.Errorf("specified index %v not exist", sc.index)
        }
    } else {
        //selector选取算法来选择节点
        var eps []*endPoint

        //1）如果指定了版本，先根据版本过滤可用的节点
        if sc.version != noSpecifiedVersion {
            for _, v := range sc.endPoints {
                if sc.version != v.version {
                    continue
                }
                eps = append(eps, v)
            }
        } else {
            eps = sc.endPoints
        }
        //2)selector选取节点
        if len(eps) > 0 {
            ep = sc.selector(eps)
        }
    }
    if ep == nil {
        //todo
        return errors.New("no available rpc servers")
    }
    //todo failover机制
    retry := 1
    var err error
    for retry > 0 {
        ep.lock.Lock()
        ep.callTimes++
        ep.lock.Unlock()
        smethod := fmt.Sprintf("%v.%v", sc.service, serviceMethod)
        err = ep.conn.Call(context.Background(), smethod, args, reply)
        if err == nil {
            break
        }
        retry--
    }
    return err
}

func (sc *SvcClient) setVersion(v string) {
    sc.version = v
}

func (sc *SvcClient) setIndex(index int) error {
    if index < 0 {
        return errors.New("negtive index not allowed")
    }
    sc.index = index
    return nil
}

func (sc *SvcClient) setSelectType(styp selectType) error {
    if int(styp) < 0 || int(styp) > len(selectors) {
        return fmt.Errorf("select type %v not support", styp)
    }
    sc.selectType = styp
    return nil
}

func (sc *SvcClient) decorate(opts ...fnOptionService) error {
    for n, fnOpt := range opts {
        if fnOpt == nil {
            return fmt.Errorf("err: decrator service client, nil option no.%v", n+1)
        }
        err := fnOpt(sc)
        if err != nil {
            return err
        }
    }
    return nil
}

func (sc *SvcClient) addEndpoint(nodes []*registry.Node) {
    for _, node := range nodes {
        ep := &endPoint{
            key:     node.Path,
            index:   node.ID.Index,
            weight:  node.Weight,
            version: node.Version,
            styp:    codec.SerializeType(node.Styp),
            addr:    node.Addr,
        }
        rpc := newRPCClient(ep.addr, ep.styp)
        if rpc == nil {
            continue
        }
        ep.conn = rpc
        sc.endPoints = append(sc.endPoints, ep)
        log.Printf("[prpc] service<%v> add endpoint: %+v, total %v\n", sc.service, ep, len(sc.endPoints))
    }
}

func (sc *SvcClient) delEndpoint(dels []string) {
    for _, del := range dels {
        for i, ep := range sc.endPoints {
            if del == ep.key {
                sc.endPoints = append(sc.endPoints[:i], sc.endPoints[i+1:]...)
                log.Printf("[prpc] delete endpoint %+v, total %v\n", ep, len(sc.endPoints))
                break
            }
        }
    }
}

func (sc *SvcClient) updateEndpoint(node *registry.Node) {
    for _, ep := range sc.endPoints {
        if ep.key == node.Path {
            //关心的数据确实发生变化
            if !ep.equalTo(node) {
                log.Printf("[prpc] update endpoint<%+v> by node<%+v>", ep, node)
                ep.update(node)
            }
            return
        }
    }
    log.Printf("[prpc][ERROR] node<%+v> update, found no corresponding endpoint", node)
}

func (sc *SvcClient) hashCode() (string, error) {
    var err error
    endian := binary.LittleEndian
    buf := new(bytes.Buffer)
    for _, v := range []interface{}{int32(sc.index), int32(sc.selectType)} {
        err = binary.Write(buf, endian, v)
        if err != nil {
            log.Println("[prpc][ERROR] binary.Write failed:", err)
            return "", err
        }
    }
    for _, v := range []string{sc.service, sc.group, sc.version} {
        buf.Write([]byte(v))
    }
    hash := sha256.New()
    hash.Write(buf.Bytes())
    if err != nil {
        return "", err
    }
    return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (sc *SvcClient) dumpMetrics() {
    log.Println("******* dumpMetrics *******")
    sc.statLock.Lock()
    log.Printf("reqTimes<%v> succTimes<%v>", sc.reqTimes, sc.succTimes)
    sc.statLock.Unlock()

    for _, ep := range sc.endPoints {
        ep.lock.Lock()
        log.Printf("endpoint: index<%v> weight<%v> callTimes<%v>", ep.index, ep.weight, ep.callTimes)
        ep.lock.Unlock()
    }
}

//===========================================================

func newSvcClient(service, group string, reg *registry.Registry, opts ...fnOptionService) *SvcClient {
    sc := &SvcClient{
        group:      group,
        service:    service,
        registry:   reg,
        version:    registry.DefaultVersion,  //默认匹配缺省版本
        index:      noSpecifiedIndex,         //默认不指定index
        selectType: SelectTypeWeightedRandom, //默认按照权重随机获得endpoint
        breakers:   make(map[string]bool),
    }
    //修饰svcClient
    err := sc.decorate(opts...)
    if err != nil {
        log.Println(err)
        return nil
    }
    //设置selector
    cs := configSelect{
        typ:   sc.selectType,
        index: sc.index,
    }
    slt, err := createSelector(cs)
    if err != nil {
        //handle err
        log.Printf("createSelector err: %v\n", err)
        return nil
    }
    sc.selector = slt
    return sc
}
