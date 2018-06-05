package registry

import (
    "encoding/json"
    "fmt"
    "strconv"
    "strings"
)

//节点id
type ID struct {
    Group string `json:"group"`
    Index int    `json:"index"`
}

func (id *ID) Dump() string {
    return fmt.Sprintf("%v.%v", id.Group, id.Index)
}
func (id *ID) Load(s string) error {
    strs := strings.Split(s, ".")
    if len(strs) != 2 {
        return fmt.Errorf("[registry] illformed node id %v", s)
    }
    id.Group = strs[0]
    idx, err := strconv.Atoi(strs[1])
    if err != nil {
        return fmt.Errorf("[registry] invalid index from string %v", s)
    }
    id.Index = idx
    return nil
}

// service可选配置项
type NodeOption struct {
    Weight  int    `json:"weight"`  //权重, 默认10
    Styp    int    `json:"styp"`    //序列化, 默认messagepack
    Version string `json:"version"` //灰度版本，默认为空
}

// service node定义
type Node struct {
    Path string
    ID
    Addr string `json:"addr"` //ip:port

    *NodeOption
}

func (node *Node) encode() ([]byte, error) {
    return json.Marshal(node)
}
func (node *Node) decode(data []byte) error {
    return json.Unmarshal(data, node)
}
func (node *Node) key() string {
    return node.ID.Dump()
}
func (node *Node) decorate(opts ...fnOptionNode) error {
    for n, fnOpt := range opts {
        if fnOpt == nil {
            return fmt.Errorf("[registry] err: decorate node, nil option no.%v", n+1)
        }
        err := fnOpt(node)
        if err != nil {
            return err
        }
    }
    return nil
}
