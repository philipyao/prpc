package client

import (
    "fmt"
    "sync"

    "github.com/philipyao/prpc/registry"
)

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
        fmt.Printf("make registry error: %+v\n", regConfig)
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
        fmt.Printf("system error: %v\n", err)
        return nil
    }
    c.mu.Lock()
    defer c.mu.Unlock()
    sc, exist := c.services[id]
    if exist {
        fmt.Printf("return existed service: id<%v>, sc<%#v>\n", id, sc)
        return sc
    }

    //prepare to create new service
    err = svc.Subscribe()
    if err != nil {
        fmt.Printf("subscribe new service err: %v, sc %#v\n", err, svc)
        return nil
    }
    fmt.Printf("new service: id<%v>, sc<%#v>\n", id, svc)
    c.services[id] = svc
    return svc
}
