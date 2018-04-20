// Package registry is an interface for service discovery
package registry

import (
    "fmt"
    "strings"
    "strconv"
    "encoding/json"
    "path/filepath"

    "github.com/philipyao/prpc/codec"
)

const (
    DefaultServiceWeight        = 10
    DefaultServiceBranch       = "master"   //默认分支
)

//服务id
type SvcID struct {
    Group       string          `json:"group"`
    Index       int             `json:"index"`
}
func (sid *SvcID) Dump() string {
    return fmt.Sprintf("%v.%v", sid.Group, sid.Index)
}
func (sid *SvcID) Load(s string) error {
    slices := strings.Split(s, ".")
    if len(slices) != 2 {
        return fmt.Errorf("invalid string: %v", s)
    }
    sid.Group = slices[0]
    index, err := strconv.Atoi(slices[1])
    if err != nil {
        return fmt.Errorf("invalid index from string %v", s)
    }
    sid.Index = index
    return nil
}

type Service struct {
    ID          SvcID       `json:"id"` //服务id，唯一

    Addr        string      `json:"addr"` //ip
    Port        int         `json:"port"` //port

    *ServiceOption
}
func (s *Service) Encode() ([]byte, error) {
    return json.Marshal(s)
}
func (s *Service) Decode(data []byte) error {
    return json.Unmarshal(data, s)
}

type ServiceOption struct {
    Weight      int         `json:"weight"` //权重, 默认10
    Styp        int         `json:"styp"`   //序列化, 默认messagepack
}

type OptionBuilder ServiceOption
func (ob *OptionBuilder) SetWeight(w int) *OptionBuilder {
    ob.Weight = w
    return ob
}
func (ob *OptionBuilder) SetStyp(s int) *OptionBuilder {
    ob.Styp = s
    return ob
}
func (ob *OptionBuilder) Build() *ServiceOption {
    so := &ServiceOption{
        Weight: DefaultServiceWeight,
        Styp: int(codec.SerializeTypeMsgpack),
    }
    if ob.Weight > 0 {
        so.Weight = ob.Weight
    }
    if ob.Styp > 0 {
        so.Styp = ob.Styp
    }
    return so
}

type fnBranch func(map[string]*Service, []string)
type fnService func(string, *Service)

type Registry struct {
    rt remote
    fb failback
    c cache

    fnb fnBranch
    fns fnService

    //(group+version) -> service list
    // group + version 唯一标记一组服务
    serviceMap    map[string][]Service

    stop bool
}

func New(zkAddr string) *Registry {
    r := &Registry{
        rt: newRemoteZooKeeper(zkAddr),
        serviceMap: make(map[string][]Service),
    }
    err := r.rt.Connect()
    if err != nil {
        //todo log
        return nil
    }
    return r
}

func (r *Registry) Register(branch string, id SvcID, addr string, port int, opt *ServiceOption) error {
    if branch == "" {
        branch = DefaultServiceBranch
    }

    svc := &Service{
        ID: id,
        Addr: addr,
        Port: port,
    }
    if opt == nil {
        opt = new(OptionBuilder).Build()
    }
    svc.ServiceOption = opt

    svcData, err := svc.Encode()
    if err != nil {
        return err
    }
    fmt.Printf("======: %v\n", string(svcData))
    err = r.rt.CreateService(branch, id.Dump(), svcData)
    if err != nil {
        return err
    }

    //todo add svc

    return nil
}

//client来订阅，指定branch下的所有服务
func (r *Registry) Subscribe(branch string, fnb fnBranch, fns fnService) {
    if branch == "" {
        branch = DefaultServiceBranch
    }

    r.fnb = fnb
    r.fns = fns

    go r.watchBranch(branch)
}

func (r *Registry) Unsubscribe(watcher BranchWatcher) {
    watcher.Stop()
}

func (r *Registry) Lookup(group string, version string) []Service {
    if version == "" {
        version = DefaultServiceBranch
    }
    key := group + "-" + version
    services, ok := r.serviceMap[key]
    if !ok {
        return nil
    }
    return services
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
        r.fnb(adds, event.Dels)
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
        r.fns(sev.Path, tsvc)
    }
}
