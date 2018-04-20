package registry


type BranchWatcherETCD struct {
}
func (bwe *BranchWatcherETCD) Accept() *BranchEvent {
    return nil
}
func (bwe *BranchWatcherETCD) Stop() {
}

type ServiceWatcherETCD struct {
}
func (swe *ServiceWatcherETCD) Accept() *ServiceEvent {
    return &ServiceEvent{
    }
}
func (swe *ServiceWatcherETCD) Stop() {
}


type remoteETCD struct {
}

func (re *remoteETCD) Connect() {
    //todo重试
}

// @param branch, 分支, 可以做灰度版本控制
// @param: sid, 服务id
// @param: data, 服务数据
// 返回svcPath
func (re *remoteETCD) CreateService(branch, sid string, data []byte) (string, error) {
    return "", nil
}

// 拉取特定branch下的所有service
func (re *remoteETCD) ListService(branch string) (map[string][]byte, error) {
    return nil, nil
}

//监听分支：服务新增或者减少
func (re *remoteETCD) WatchBranch(branch string) BranchWatcher {
    watcher := &BranchWatcherETCD{

    }

    return watcher
}

//监听服务：数据变化
func (re *remoteETCD) WatchService(svcPath string) ServiceWatcher {
    watcher := &ServiceWatcherETCD{
    }

    return watcher
}

func (re *remoteETCD) Close() {

}

