package registry

type BranchEvent struct {
    Err         error
    Adds        map[string]string       //新增service path/value
    Dels        []string                //删除service path
}
type BranchWatcher interface {
    Accept() *BranchEvent
    Stop()
}

type ServiceEvent struct {
    Err         error
    Path        string
    Value       string
    OldVal      string
}
type ServiceWatcher interface {
    Accept() *ServiceEvent
    Stop()
}

type remote interface {
    Connect()

    //创建服务
    CreateService(string, string, []byte) (string, error)

    // 拉取特定branch下的所有服务
    ListService(string) (map[string][]byte, error)

    //监听分支：服务新增或者减少
    WatchBranch(branch string) BranchWatcher

    //监听服务：数据变化
    WatchService(svcPath string) ServiceWatcher

    Close()
}
