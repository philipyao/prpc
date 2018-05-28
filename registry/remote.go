package registry

type ServiceEvent struct {
    Err  error
    Adds map[string]string //新增nodes: path/value
    Dels []string          //删除nodes: path
}
type ServiceWatcher interface {
    Accept() *ServiceEvent
    Stop()
}

type NodeEvent struct {
    Err   error
    Path  string
    Value string
}
type NodeWatcher interface {
    Accept() *NodeEvent
    Stop()
}

type remote interface {
    Connect() error

    //创建服务节点
    CreateServiceNode(string, string, []byte) error

    //获取节点
    GetServiceNode(string, string) ([]byte, error)

    //注销服务节点
    DeleteServiceNode(string, string) error

    //拉取特定service下的所有节点
    ListServiceNode(string) (map[string][]byte, error)

    //监听服务：节点新增或者减少
    WatchService(string) ServiceWatcher

    //监听服务节点：数据变化
    WatchNode(string) NodeWatcher

    Close()
}
