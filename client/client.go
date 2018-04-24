package client

import (
    "fmt"
    "sync"

    "github.com/philipyao/prpc/registry"
    "log"
)

type Client struct {
    registry *registry.Registry
    mu sync.Mutex //protect following

    clientMap map[string]*RPCClient

    //todo selector

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
    c.clientMap = make(map[string]*RPCClient)

    svcs, err := reg.Lookup("")
    if err != nil {
        fmt.Printf("registry lookup error: %v", err)
        return nil
    }
    c.addClients(svcs)

    // 开始监听registry的事件
    reg.Subscribe("", c)

    return c
}

func (c *Client) Get(group string, index int) *RPCClient {
    id := fmt.Sprintf("%v.%v", group, index)
    return c.clientMap[id]
}

func (c *Client) Select(group string) *RPCClient {
    return c.selectByRandom(group)
}

func (c *Client) addClients(svcs []*registry.Service) {
    for _, svc := range svcs {
        id := svc.ID.Dump()
        if _, exist := c.clientMap[id]; exist {
            fmt.Printf("error addClients: exist %v\n", id)
            continue
        }
        rpc := newRPC(svc)
        if rpc == nil {
            fmt.Printf("create rpc error: svr<%+v>", svc)
            continue
        }
        c.clientMap[id] = rpc
        fmt.Printf("add rpc: %v, svc: %+v\n", rpc, rpc.svc)
    }
}

func (c *Client) delClient(id string) {
    rpc, exist := c.clientMap[id]
    if !exist {
        return
    }
    log.Printf("delClient: id %v", id)
    rpc.Close()
    delete(c.clientMap, id)
}