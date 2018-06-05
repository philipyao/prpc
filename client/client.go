package client

import (
    "sync"

    "github.com/philipyao/prpc/registry"
    "log"
)

func init() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type Client struct {
    registry *registry.Registry

    mu       sync.Mutex            //protect following
    services map[string]*SvcClient //id -> SvcClient

    //todo middleware
}

func New(regConfig interface{}) *Client {
    var reg *registry.Registry
    switch regConfig.(type) {
    case *registry.RegConfigZooKeeper:
        reg = registry.New(regConfig.(*registry.RegConfigZooKeeper).ZKAddr)
    default:
    }
    if reg == nil {
        log.Printf("[prpc][ERROR] make registry error: %+v", regConfig)
        return nil
    }

    c := new(Client)
    c.services = make(map[string]*SvcClient)
    c.registry = reg
    return c
}

func (c *Client) Service(service, group string, opts ...fnOptionService) *SvcClient {
    svc := newSvcClient(service, group, c.registry, opts...)
    id, err := svc.hashCode()
    if err != nil {
        log.Printf("[prpc][ERROR] system error: %v", err)
        return nil
    }
    c.mu.Lock()
    defer c.mu.Unlock()
    sc, exist := c.services[id]
    if exist {
        log.Printf("[prpc] return existed service: id<%v>, sc<%p>", id, sc)
        return sc
    }

    //prepare to create new service
    err = svc.Subscribe()
    if err != nil {
        log.Printf("[prpc][ERROR] subscribe new service err: %v, sc %p", err, svc)
        return nil
    }
    log.Printf("[prpc] new service: id<%v>, sc<%#v>", id, svc)
    c.services[id] = svc
    return svc
}
