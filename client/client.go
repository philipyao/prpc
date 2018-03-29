package client

import (
    "sync"
)

type client struct {
    //todo dir watcher, service新增或者删除

    mu sync.Mutex //protect following

    clientMap map[string]*RPCClient

    //todo selector

    //todo middleware
}

func (c *client) GetClient(id string) *RPCClient {
    return nil
}
