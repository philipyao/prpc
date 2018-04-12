package client

import (
    "fmt"
    "sync"
    "log"
    "path/filepath"

    "github.com/philipyao/toolbox/zkcli"
)

var (
    DefaultZKPath       = "/__RPC__"
)

type Client struct {
    zk *zkcli.Conn
    //dir watcher, rpc新增或者删除
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
        return nil
    }

    c := new(Client)
    c.clientMap = make(map[string]*RPCClient)
    c.zk = zkConn
    //获取数据
    nodes, err := zkConn.GetChildren(DefaultZKPath)
    if err != nil {
        log.Printf("zk GetChildren error: %v\n", err)
        return nil
    }
    c.update(nodes)

    c.watcher = newWatcher(zkConn, DefaultZKPath)
    err = c.watcher.WatchChildren(func(p string, children []string, e error){
        log.Printf("watch trigger: %+v, %v", children, e)
        if e != nil {
            return
        }
        nodes := make(map[string][]byte)
        for _, cp := range children {
            path := c.watcher.Path() + "/" + cp
            val, err := zkConn.Get(path)
            if err != nil {
                log.Printf("get child %v error %v", cp, err)
                continue
            }
            nodes[path] = val
        }
        c.update(nodes)
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

func (c *Client) update(nodes map[string][]byte) {
    log.Printf("update: nodes %+v\n", nodes)
    for id, rpc := range c.clientMap {
        val, exist := nodes[rpc.Path()]
        if !exist {
            log.Printf("rpc not exist in nodes: id %v, delete it\n", id)
            //关闭rpc
            rpc.Close()
            //safe delete
            delete(c.clientMap, id)
            continue
        }
        if string(val) != rpc.NodeVal() {
            //this is not supposed to happen
            panic("child val")
        }
    }
    for path, val := range nodes {
        id := filepath.Base(path)
        log.Printf("check nodes: id %v\n", id)
        if _, exist := c.clientMap[id]; !exist {
            //有新rpc server加入
            rpc := newRPC(c.zk, path, string(val))
            if rpc != nil {
                c.clientMap[id] = rpc
                log.Printf("find new client: id %v, rpc %+v\n", id, rpc)
            }
        }
    }
}
