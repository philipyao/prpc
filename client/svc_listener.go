package client

import (
	"log"
	"github.com/philipyao/prpc/registry"
	//"path/filepath"
)

func (sc *svcClient) OnServiceChange(adds map[string]*registry.Node, dels []string) {
	log.Printf("OnServiceChange: adds %+v, dels %+v", adds, dels)
	var nodes []*registry.Node
	for _, v := range adds {
		nodes = append(nodes, v)
	}
	if len(nodes) > 0 {
		sc.addEndpoint(nodes)
	}
	if len(dels) > 0 {
		sc.delEndpoint(dels)
	}
}

func (sc *svcClient) OnNodeChange(key string, node *registry.Node) {
	log.Printf("OnNodeChange: key %s, svc %+v", key, node)
	sc.updateEndpoint(node)
}
