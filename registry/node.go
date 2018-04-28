package registry

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

//节点id
type ID struct {
	Group 		string			`json:"group"`
	Index       int             `json:"index"`
}
func (id *ID) Dump() string {
	return fmt.Sprintf("%v.%v", id.Group, id.Index)
}
func (id *ID) Load(s string) error {
	strs := strings.Split(s, ".")
	if len(strs) != 2 {
		return fmt.Errorf("illformed node id %v", s)
	}
	id.Group = strs[0]
	idx, err := strconv.Atoi(strs[1])
	if err != nil {
		return fmt.Errorf("invalid index from string %v", s)
	}
	id.Index = idx
	return nil
}

// service可选配置项
type NodeOption struct {
	Weight      int         `json:"weight"` //权重, 默认10
	Styp        int         `json:"styp"`   //序列化, 默认messagepack
	Version 	string 		`json:"version"`//灰度版本，默认为空
}

// service node定义
type Node struct {
	ID
	Addr        string      `json:"addr"` //ip:port

	*NodeOption
}
func (s *Node) Encode() ([]byte, error) {
	return json.Marshal(s)
}
func (s *Node) Decode(data []byte) error {
	return json.Unmarshal(data, s)
}
func (s *Node) Key() string {
	return s.ID.Dump()
}

