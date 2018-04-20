// Package registry is an interface for service discovery
package registry

import (
    "fmt"
    "strings"
    "strconv"
)

const (
    DefaultServiceWeight        = 10
    DefaultServiceBranch       = "master"   //默认分支
)

//服务id
type SvcID struct {
    Group       string
    Index       int
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
    ID          SvcID      //服务id，唯一

    Weight      int         //权重, 默认10
    Styp        int         //序列化
    Version     string      //版本, 默认版本号1.0

    Addr        string      //ip
    Port        int         //port
}

type Registry struct {
    rt remote
    fb failback
    c cache

    //(group+version) -> service list
    // group + version 唯一标记一组服务
    serviceMap    map[string][]Service
}

func New() *Registry {
    return nil
}

func (r *Registry) Register(group string, index int) {

}

func (r *Registry) Subscribe() {

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
