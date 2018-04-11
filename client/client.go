package client

import (
    "fmt"
    "sync"
    "log"

    "github.com/philipyao/toolbox/zkcli"
)

var (
    DefaultZKPath       = "/__RPC__"
)

type Client struct {
    //todo dir watcher, service新增或者删除
    zk *zkcli.Conn
    watcher *watcher

    mu sync.Mutex //protect following

    clientMap map[string]*RPCClient

    //todo selector

    //todo middleware
}

func New(zkAddr string) *Client {
    zkConn, err := zkcli.Connect(zkAddr)
    if err != nil {
        log.Printf("zk connect returned error: %+v", err)
    }

    c := new(Client)
    c.clientMap = make(map[string]*RPCClient)
    c.zk = zkConn
    //获取数据
    nodes, err := zkConn.GetChildren(DefaultZKPath)
    for k, v := range nodes {
        rpc := newRPC(zkConn, DefaultZKPath + "/" + k, string(v))
        if rpc != nil {
            c.clientMap[k] = rpc
        }
    }

    c.watcher = newWatcher(zkConn, DefaultZKPath)
    err = c.watcher.Watch(func(p string, c []string, e error){
        log.Printf("watch trigger: %+v, %v", c, e)
    })
    if err != nil {
        log.Printf("watch error %v", err)
        return nil
    }
    return c
}

func (c *Client) Get(group string, index int) *RPCClient {
    id := fmt.Sprintf("%v.%v", group, index)
    return c.clientMap[id]
}
