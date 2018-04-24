package client

import (
	"log"
	"github.com/philipyao/prpc/registry"
)

type clientListener struct {}

func (cl *clientListener) OnBranchChange(adds map[string]*registry.Service, dels []string) {
	log.Printf("OnBranchChange: adds %+v, dels %+v", adds, dels)
}

func (cl *clientListener) OnServiceChange(key string, svc *registry.Service) {
	log.Printf("OnServiceChange: key %s, svc %+v", key, svc)
}
