package registry

import (
    "github.com/philipyao/prpc/codec"
    "log"
)

const (
    maxWeight = 10000
)

//服务注册修饰项
type fnOptionNode func(node *Node) error

func WithWeight(weight int) fnOptionNode {
    if weight < 0 || weight > maxWeight {
        log.Println("[registry] invalid weight value")
        return nil
    }
    return func(node *Node) error {
        node.Weight = weight
        return nil
    }
}
func WithSerialize(styp codec.SerializeType) fnOptionNode {
    return func(node *Node) error {
        node.Styp = int(styp)
        return nil
    }
}
func WithVersion(version string) fnOptionNode {
    if version == "" {
        log.Println("[registry] empty version not allowed")
        return nil
    }
    return func(node *Node) error {
        node.Version = version
        return nil
    }
}
