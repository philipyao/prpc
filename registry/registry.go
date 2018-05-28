// Package registry is an interface for service discovery
package registry

import (
    "errors"
    "fmt"
    "github.com/philipyao/prpc/codec"
    "path/filepath"
    "sync"
    "log"
)

const (
    DefaultGroup   = "default"
    DefaultVersion = "v1.0"

    defaultNodeWeight = 10
)

type Listener interface {
    OnServiceChange(map[string]*Node, []string)
    OnNodeChange(string, *Node)
}

type Registry struct {
    rt remote
    fb failback
    c  cache

    listener     Listener
    svcWatcher   ServiceWatcher
    lock sync.Mutex       //protect nodeWatchers
    nodeWatchers []NodeWatcher

    exit chan struct{}
    wg   sync.WaitGroup
}

func New(zkAddr string) *Registry {
    r := &Registry{
        rt:   newRemoteZooKeeper(zkAddr),
        exit: make(chan struct{}),
    }
    err := r.rt.Connect()
    if err != nil {
        fmt.Printf("connect to remote error: %v\n", err)
        return nil
    }
    return r
}

//服务提供方（server）在注册中心注册服务节点
func (r *Registry) Register(service, group string, index int, addr string, opts ...fnOptionNode) error {
    //todo check args
    node := &Node{
        ID: ID{
            Group: group,
            Index: index,
        },
        Addr: addr,
        NodeOption: &NodeOption{
            Weight:  defaultNodeWeight,               //默认权重
            Styp:    int(codec.SerializeTypeMsgpack), //默认序列化方法
            Version: DefaultVersion,                  //缺省版本号
        },
    }
    //修饰
    err := node.decorate(opts...)
    if err != nil {
        return err
    }
    nodeData, err := node.encode()
    if err != nil {
        fmt.Printf("encode node<%+v> err %v\n", node, err)
        return err
    }
    log.Printf("====== register service: %v, %v", node.key(), string(nodeData))

    //todo 检查cache是否存在，否则报错


    err = r.rt.CreateServiceNode(makeServiceKey(service, group), node.key(), nodeData)
    if err != nil {
        if err == ErrRemoteNodeExist {
            //可能进程异常退出，导致 remote 还缓存有节点信息（此时心跳尚未过期，否则节点会被删除）
            data, terr := r.rt.GetServiceNode(makeServiceKey(service, group), node.key())
            if terr == nil {
                var preNode Node
                terr = preNode.decode(data)
                if terr == nil && node.Addr == preNode.Addr {
                    log.Println("find node with the same addr already exist, clean it up")
                    r.Unregister(service, group, index)
                    return r.rt.CreateServiceNode(makeServiceKey(service, group), node.key(), nodeData)
                }
            }
        }
        fmt.Printf("remote create node<%+v> err %v\n", node, err)
        return err
    }

    //todo add node to cache

    return nil
}

//服务提供方（server）在注册中心注销服务节点
func (r *Registry) Unregister(service, group string, index int) error {
    return r.rt.DeleteServiceNode(makeServiceKey(service, group), fmt.Sprintf("%v.%v", group, index))
}

//client来订阅特定service，如果服务节点有增删或者节点数据变化，会有通知；
//返回所有节点，供客户端初始化
func (r *Registry) Subscribe(service, group string, listener Listener) ([]*Node, error) {
    //todo check args
    if listener == nil {
        return nil, errors.New("no listener specified")
    }
    r.listener = listener

    var nodes []*Node
    nodeMap, err := r.rt.ListServiceNode(makeServiceKey(service, group))
    if err != nil {
        return nil, err
    }
    for k, v := range nodeMap {
        node := new(Node)
        node.Path = k
        err := node.decode(v)
        if err != nil {
            fmt.Printf("decode node err: %v, %v\n", err, string(v))
            continue
        }
        if filepath.Base(k) != node.ID.Dump() {
            fmt.Printf("node id mismatch: %v, %v\n", k, node.ID.Dump())
            continue
        }
        nodes = append(nodes, node)

        r.wg.Add(1)
        go r.watchNode(k)
    }

    r.wg.Add(1)
    go r.watchService(service, group)

    return nodes, nil
}

func (r *Registry) Close() {
    select {
    case <-r.exit: //防止重复关闭channel
        return
    default:
    }
    fmt.Println("registry Close()")
    if r.svcWatcher != nil {
        r.svcWatcher.Stop()
    }
    r.lock.Lock()
    for _, w := range r.nodeWatchers {
        w.Stop()
    }
    r.lock.Unlock()
    r.rt.Close()  //todo 是不是rt断了，所有watch循环就会返回错误？从而结束？
    close(r.exit) //通知所有goroutine停止运行
    r.wg.Wait()   //等所有的goroutine结束
}

///====================================================================

func (r *Registry) watchService(service, group string) {
    defer r.wg.Done()

    watcher := r.rt.WatchService(makeServiceKey(service, group))
    r.svcWatcher = watcher
    var event *ServiceEvent

    for {
        event = watcher.Accept()
        if event == nil {
            break
        }
        if event.Err != nil {
            fmt.Printf("Accept error: %v, break\n", event.Err)
            break
        }
        adds := make(map[string]*Node)
        var (
            node *Node
            err  error
        )
        for k, v := range event.Adds {
            node = new(Node)
            node.Path = k
            err = node.decode([]byte(v))
            if err != nil {
                //todo
                fmt.Printf("decode add node<%v %v> err: %v\n", k, v, err)
                continue
            }
            adds[k] = node

            r.wg.Add(1)
            go r.watchNode(k)
        }
        if len(adds) > 0 || len(event.Dels) > 0 {
            r.listener.OnServiceChange(adds, event.Dels)
        }
    }
}

func (r *Registry) watchNode(nodePath string) {
    defer r.wg.Done()

    w := r.rt.WatchNode(nodePath)
    r.lock.Lock()
    r.nodeWatchers = append(r.nodeWatchers, w)
    r.lock.Unlock()

    var (
        err error
        nev *NodeEvent
    )

    for {
        nev = w.Accept()
        if nev == nil {
            break
        }
        if nev.Err != nil {
            fmt.Printf("watch node error: %v, break\n", nev.Err)
            break
        }
        if nev.Path != nodePath {
            fmt.Printf("node path mismatch: %v %+v, break\n", nodePath, nev)
            break
        }
        tnode := new(Node)
        tnode.Path = nev.Path
        err = tnode.decode([]byte(nev.Value))
        if err != nil {
            //todo
            fmt.Printf("decode node err: %v, %+v\n", nodePath, nev)
            continue
        }
        if filepath.Base(nev.Path) != tnode.ID.Dump() {
            //todo
            fmt.Printf("node id mismatch: %+v %+v\n", nev, tnode)
            break
        }
        r.listener.OnNodeChange(nev.Path, tnode)
    }
}

func makeServiceKey(service, group string) string {
    return fmt.Sprintf("%v@%v", service, group)
}
