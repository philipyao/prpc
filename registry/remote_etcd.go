package registry

type ServiceWatcherETCD struct {
}

func (swe *ServiceWatcherETCD) Accept() *ServiceEvent {
    return nil
}
func (swe *ServiceWatcherETCD) Stop() {
}

type NodeWatcherETCD struct {
}

func (nwe *NodeWatcherETCD) Accept() *NodeEvent {
    return &NodeEvent{}
}
func (nwe *NodeWatcherETCD) Stop() {
}

type remoteETCD struct {
}

func (re *remoteETCD) Connect() {
    //todo重试
}

func (re *remoteETCD) CreateService() (string, error) {
    return "", nil
}

// 拉取特定branch下的所有service
func (re *remoteETCD) ListService(branch string) (map[string][]byte, error) {
    return nil, nil
}

//监听分支：服务新增或者减少
func (re *remoteETCD) WatchBranch(branch string) ServiceWatcher {
    watcher := &ServiceWatcherETCD{}

    return watcher
}

//监听服务：数据变化
func (re *remoteETCD) WatchService(svcPath string) NodeWatcher {
    watcher := &NodeWatcherETCD{}

    return watcher
}

func (re *remoteETCD) Close() {

}
