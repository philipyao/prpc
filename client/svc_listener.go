package client

import (
    "github.com/philipyao/prpc/registry"
    "log"
    //"path/filepath"
)

func (sc *SvcClient) OnServiceChange(adds map[string]*registry.Node, dels []string) {
    log.Printf("[prpc] OnServiceChange: adds %+v, dels %+v", adds, dels)
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

func (sc *SvcClient) OnNodeChange(key string, node *registry.Node) {
    log.Printf("[prpc] OnNodeChange: key %s, svc %+v", key, node)
    sc.updateEndpoint(node)
}
