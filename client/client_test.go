package client

import (
    //"time"
    "testing"
    "github.com/philipyao/prpc/registry"
)

//func TestCreateClient(t *testing.T) {
//    config := &registry.RegConfigZooKeeper{ZKAddr: "localhost:2181"}
//    client := New(config)
//    if client == nil {
//        t.Fatal("error new client")
//    }
//    time.Sleep(60 * time.Second)
//}

func TestClientCallRPC(t *testing.T) {
    config := &registry.RegConfigZooKeeper{ZKAddr: "localhost:2181"}
    client := New(config)
    if client == nil {
        t.Fatal("error new client")
    }

    rpcCli := client.Get("zone1001.gamesvr", 2)
    if rpcCli == nil {
        t.Fatal("error find rpc client")
    }
    var args Args
    args.A = 2
    args.B = 3
    var reply int
    err := rpcCli.Call("Arith.Multiply", &args, &reply)
    if err != nil {
        t.Fatalf("error call %v", err)
    }
    t.Logf("reply: %v", reply)
}
