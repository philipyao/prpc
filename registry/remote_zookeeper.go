package registry

import (
    "fmt"
    "github.com/philipyao/toolbox/zkcli"
    "sync"
    "log"
    "errors"
)

const (
    defaultZKRootPath = "/__PRPC__"
)

var (
    ErrRemoteNodeExist      = errors.New("node already exist")
)

type ServiceWatcherZK struct {
    exit   chan struct{}
    events chan *zkcli.EventDataChild
}

func (swzk *ServiceWatcherZK) Accept() *ServiceEvent {
    zkev := <-swzk.events
    if zkev == nil {
        fmt.Println("service events closed!")
        return nil
    }
    return &ServiceEvent{
        Err:  zkev.Err,
        Adds: zkev.Adds,
        Dels: zkev.Dels,
    }
}
func (swzk *ServiceWatcherZK) Stop() {
    select {
    case <-swzk.exit: //防止重复关闭channel
        return
    default:
    }
    close(swzk.exit)
}

type NodeWatcherZK struct {
    exit   chan struct{}
    events chan *zkcli.EventDataNode
}

func (nwzk *NodeWatcherZK) Accept() *NodeEvent {
    zkev := <-nwzk.events
    if zkev == nil {
        fmt.Println("node events closed!")
        return nil
    }
    return &NodeEvent{
        Err:   zkev.Err,
        Path:  zkev.Path,
        Value: zkev.Value,
    }
}
func (nwzk *NodeWatcherZK) Stop() {
    select {
    case <-nwzk.exit: //防止重复关闭channel
        return
    default:
    }
    close(nwzk.exit)
}

type remoteZooKeeper struct {
    zkAddr string
    client *zkcli.Conn

    once sync.Once
}

func newRemoteZooKeeper(zkAddr string) remote {
    return &remoteZooKeeper{
        zkAddr: zkAddr,
    }
}

func (rz *remoteZooKeeper) Connect() error {
    client, err := zkcli.Connect(rz.zkAddr)
    if err != nil {
        //todo重试
        return err
    }
    //todo 重连
    rz.client = client
    return nil
}

func (rz *remoteZooKeeper) CreateServiceNode(service, key string, data []byte) error {
    log.Printf("CreateServiceNode service<%v> key<%v>\n", service, key)
    var err error
    servicePath := makePath(defaultZKRootPath, service)
    err = rz.client.MakeDirP(servicePath)
    if err != nil {
        return err
    }

    nodePath := makePath(servicePath, key)
    //判断key是否存在
    exist, err := rz.client.Exists(nodePath)
    if err != nil {
        return err
    }
    if exist {
        return ErrRemoteNodeExist
    }
    err = rz.client.CreateEphemeral(nodePath, []byte(data))
    if err != nil {
        return err
    }
    log.Printf("create service node %v ok\n", nodePath)
    return nil
}

func (rz *remoteZooKeeper) GetServiceNode(service, key string) ([]byte, error) {
    var err error
    servicePath := makePath(defaultZKRootPath, service)
    err = rz.client.MakeDirP(servicePath)
    if err != nil {
        return nil, err
    }

    nodePath := makePath(servicePath, key)
    return rz.client.Get(nodePath)
}

func (rz *remoteZooKeeper) DeleteServiceNode(service, key string) error {
    var err error
    servicePath := makePath(defaultZKRootPath, service)
    err = rz.client.MakeDirP(servicePath)
    if err != nil {
        return err
    }

    nodePath := makePath(servicePath, key)
    log.Printf("delete service node %v\n", nodePath)
    return rz.client.Delete(nodePath)
}

func (rz *remoteZooKeeper) ListServiceNode(service string) (map[string][]byte, error) {
    //如果service目录不存在，则创建
    servicePath := makePath(defaultZKRootPath, service)
    err := rz.client.MakeDirP(servicePath)
    if err != nil {
        return nil, err
    }
    return rz.client.GetChildren(servicePath)
}

func (rz *remoteZooKeeper) WatchService(service string) ServiceWatcher {
    watcher := &ServiceWatcherZK{
        exit:   make(chan struct{}),
        events: make(chan *zkcli.EventDataChild, 10),
    }

    //如果service目录不存在，则创建
    servicePath := makePath(defaultZKRootPath, service)
    err := rz.client.MakeDirP(servicePath)
    if err != nil {
        go func() {
            watcher.events <- &zkcli.EventDataChild{Err: fmt.Errorf("MakeDirP error: %v, %v", service, err)}
        }()
        return watcher
    }
    rz.client.WatchDir(servicePath, watcher.events, watcher.exit)
    return watcher
}

func (rz *remoteZooKeeper) WatchNode(nodePath string) NodeWatcher {
    watcher := &NodeWatcherZK{
        exit:   make(chan struct{}),
        events: make(chan *zkcli.EventDataNode, 10),
    }

    rz.client.WatchNode(nodePath, watcher.events, watcher.exit)
    return watcher
}

func (rz *remoteZooKeeper) Close() {
    rz.once.Do(func() {
        rz.client.Close()
        rz.client = nil
    })
}

/////////////////////////////////////////////////////////
func makePath(prefix string, entries ...string) string {
    path := prefix
    for _, entry := range entries {
        if entry == "" {
            continue
        }
        path += "/" + entry
    }
    return path
}
