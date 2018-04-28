package client

import (
	"log"
	"github.com/philipyao/prpc/registry"
	//"path/filepath"
)

func (sc *svcClient) OnBranchChange(adds map[string]*registry.Service, dels []string) {
	log.Printf("OnBranchChange: adds %+v, dels %+v", adds, dels)
	var svcs []*registry.Service
	for _, svc := range adds {
		svcs = append(svcs, svc)
	}
	if len(svcs) > 0 {
		//sc.addClients(svcs)
	}
	//for _, key := range dels {
		//id := filepath.Base(key)
		//sc.delClient(id)
	//}
}

func (sc *svcClient) OnServiceChange(key string, svc *registry.Service) {
	log.Printf("OnServiceChange: key %s, svc %+v", key, svc)
}
