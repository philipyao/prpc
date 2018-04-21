package registry

import (
	"encoding/json"
	"github.com/philipyao/prpc/codec"
	"fmt"
	"strings"
	"strconv"
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

// service可选配置项
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

// service定义
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

