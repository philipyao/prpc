package client

import (
    "fmt"
    "sync"

    "github.com/philipyao/prpc/registry"
    //"log"
)

type Client struct {
    registry *registry.Registry
    mu sync.Mutex //protect following

    //group.service -> svcClient
    services map[string]*svcClient

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
    //c.clientMap = make(map[string]*RPCClient)
	//
    //svcs, err := reg.Lookup("")
    //if err != nil {
    //    fmt.Printf("registry lookup error: %v", err)
    //    return nil
    //}
    //c.addClients(svcs)

    // 开始监听registry的事件
    //reg.Subscribe("", c)

    return c
}

func (c *Client) Service(group string, service string, opts ...fnOptionService) *svcClient {
    //GetOption可以指定index/version等
    //todo 如果clientmap里没有，则应该是新的svcClient
    //则尝试新建svcClient
    //如果成功，则加入缓存

    //新建svcClient的时候，从registry里获取所有endpoints，结合选择策略，生成selector必报

    id := fmt.Sprintf("%v.%v", group, service)
    //todo id这里不行，要唯一，每次调用Service的id是否一致，要算所有参数的hash
    //如果hash一致才算同一个svcClient，否则即使group和service一样，都不是一个svcClient
    return c.services[id]
}