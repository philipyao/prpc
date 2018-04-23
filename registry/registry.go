// Package registry is an interface for service discovery
package registry

import (
    "fmt"
    "path/filepath"
)

const (
    DefaultServiceWeight        = 10
    DefaultServiceBranch       = "master"   //默认分支
)

type Listener interface {
    OnBranchChange(map[string]*Service, []string)
    OnServiceChange(string, *Service)
}

type Registry struct {
    rt remote
    fb failback
    c cache

    listener Listener
}

func New(zkAddr string) *Registry {
    r := &Registry{
        rt: newRemoteZooKeeper(zkAddr),
    }
    err := r.rt.Connect()
    if err != nil {
        fmt.Printf("connect to remote error: %v\n", err)
        return nil
    }
    return r
}

func (r *Registry) Register(branch string, id SvcID, addr string, opt *ServiceOption) error {
    if branch == "" {
        branch = DefaultServiceBranch
    }

    //todo check args

    svc := &Service{
        ID: id,
        Addr: addr,
    }
    if opt == nil {
        opt = new(OptionBuilder).Build()
    }
    svc.ServiceOption = opt

    svcData, err := svc.Encode()
    if err != nil {
        fmt.Printf("encode service<%v> err %v\n", svc, err)
        return err
    }
    fmt.Printf("====== register service: %v\n", string(svcData))
    err = r.rt.CreateService(branch, id.Dump(), svcData)
    if err != nil {
        fmt.Printf("remote create service<%v> err %v\n", svc, err)
        return err
    }

    //todo add svc

    return nil
}

//client来订阅，指定branch下的所有服务
func (r *Registry) Subscribe(branch string, listener Listener) {
    if branch == "" {
        branch = DefaultServiceBranch
    }
    if listener == nil {
        panic("no listener specified")
    }
    r.listener = listener

    go r.watchBranch(branch)
}

//func (r *Registry) Unsubscribe(watcher BranchWatcher) {
//    watcher.Stop()
//}

//拉取branch下的所有服务
func (r *Registry) Lookup(branch string) ([]*Service, error) {
    if branch == "" {
        branch = DefaultServiceBranch
    }

    var svcs []*Service
    svcMap, err := r.rt.ListService(branch)
    if err != nil {
        return nil, err
    }
    for k, v := range svcMap {
        svc := new(Service)
        err := svc.Decode(v)
        if err != nil {
            fmt.Printf("decode service err: %v, %v\n", err, string(v))
            continue
        }
        if filepath.Base(k) != svc.ID.Dump() {
            fmt.Printf("service id mismatch: %v, %v\n", k, svc.ID.Dump())
            continue
        }
        svcs = append(svcs, svc)
    }
    return svcs, nil
}

func (r *Registry) Close() {
    r.rt.Close()
}

///====================================================================

func (r *Registry) watchBranch(branch string) {
    watcher := r.rt.WatchBranch(branch)
    var event *BranchEvent
    for {
        event = watcher.Accept()
        if event.Err != nil {
            fmt.Printf("subscribe watch error %v, break", event.Err)
            break
        }
        adds := make(map[string]*Service)
        var (
            svc *Service
            err error
        )
        for k, v := range event.Adds {
            svc = new(Service)
            err = svc.Decode([]byte(v))
            if err != nil {
                //todo
            }
            adds[k] = svc

            go r.watchService(k)
        }
        r.listener.OnBranchChange(adds, event.Dels)
    }
}

func (r *Registry) watchService(spath string) {
    w := r.rt.WatchService(spath)
    var (
        err error
        sev *ServiceEvent
    )
    for {
        sev = w.Accept()
        if sev.Err != nil {
            fmt.Printf("watch service error %v, break", sev.Err)
            break
        }
        tsvc := new(Service)
        err = tsvc.Decode([]byte(sev.Value))
        if err != nil {
            //todo
        }
        if filepath.Base(sev.Path) != tsvc.ID.Dump() {
            //todo
        }
        r.listener.OnServiceChange(sev.Path, tsvc)
    }
}
