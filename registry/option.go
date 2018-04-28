package registry

import (
	"github.com/philipyao/prpc/codec"
	"errors"
)

const (
	maxWeight		= 10000
)

type fnOptionNode func(node *Node) error
func WithWeight(weight int) (fnOptionNode, error) {
	if weight < 0 || weight > maxWeight {
		return nil, errors.New("invalid weight value")
	}
	return func(node *Node) error {
		node.Weight = weight
		return nil
	}, nil
}
func WithSerialize(styp codec.SerializeType) (fnOptionNode, error) {
	return func(node *Node) error {
		node.Styp = int(styp)
		return nil
	}, nil
}
func WithVersion(version string) (fnOptionNode, error) {
	if version == "" {
		return nil, errors.New("empty version not allowed")
	}
	return func(node *Node) error {
		node.Version = version
		return nil
	}, nil
}
