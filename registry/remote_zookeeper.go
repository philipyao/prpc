package registry

import (
    "fmt"
    "github.com/philipyao/toolbox/zkcli"
)

const (
    DefaultZKRootPath       = "/__PRPC__"
)

type BranchWatcherZK struct {
    exit chan struct{}
    events chan *zkcli.EventDataChild
}
func (bwzk *BranchWatcherZK) Accept() *BranchEvent {
    zkev := <- bwzk.events
    return &BranchEvent{
        Err: zkev.Err,
        Adds: zkev.Adds,
        Dels: zkev.Dels,
    }
}
func (bwzk *BranchWatcherZK) Stop() {
    close(bwzk.exit)
}

type ServiceWatcherZK struct {
    exit chan struct{}
    events chan *zkcli.EventDataNode
}
func (swzk *ServiceWatcherZK) Accept() *ServiceEvent {
    zkev := <- swzk.events
    return &ServiceEvent{
        Err: zkev.Err,
        Path: zkev.Path,
        Value: zkev.Value,
        OldVal: zkev.OldVal,
    }
}
func (swzk *ServiceWatcherZK) Stop() {
    close(swzk.exit)
}


type remoteZooKeeper struct {
    client      *zkcli.Conn
}

func (rz *remoteZooKeeper) Connect() {
    //todo重试
}

// @param branch, 分支, 可以做灰度版本控制
// @param: sid, 服务id
// @param: data, 服务数据
// 返回svcPath
func (rz *remoteZooKeeper) CreateService(branch, sid string, data []byte) (string, error) {
    var err error
    branchPath := joinPath(DefaultZKRootPath, branch)
    err = rz.client.MakeDirP(branchPath)
    if err != nil {
        return "", err
    }

    key := joinPath(branchPath, sid)
    //判断key是否存在
    exist, err := rz.client.Exists(key)
    if err != nil {
        return "", err
    }
    if exist {
        return "", fmt.Errorf("service already exist: sid<%v>", sid)
    }

    err = rz.client.Create(key, []byte(data))
    if err != nil {
        return "", err
    }
    return key, nil
}

// 拉取特定branch下的所有service
func (rz *remoteZooKeeper) ListService(branch string) (map[string][]byte, error) {
    return rz.client.GetChildren(joinPath(DefaultZKRootPath, branch))
}

//监听分支：服务新增或者减少
func (rz *remoteZooKeeper) WatchBranch(branch string) BranchWatcher {
    watcher := &BranchWatcherZK{
        exit: make(chan struct{}),
        events: make(chan *zkcli.EventDataChild, 10),
    }

    zkPath := joinPath(DefaultZKRootPath, branch)
    rz.client.WatchDir(zkPath, watcher.events, watcher.exit)
    return watcher
}

//监听服务：数据变化
func (rz *remoteZooKeeper) WatchService(svcPath string) ServiceWatcher {
    watcher := &ServiceWatcherZK{
        exit: make(chan struct{}),
        events: make(chan *zkcli.EventDataNode, 10),
    }

    rz.client.WatchNode(svcPath, watcher.events, watcher.exit)
    return watcher
}

func (rz *remoteZooKeeper) Close() {

}

/////////////////////////////////////////////////////////
func joinPath(a, b string) string {
    return a + "/" + b
}