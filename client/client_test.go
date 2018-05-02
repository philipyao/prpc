package client

import (
    //"time"
    "testing"
    "github.com/philipyao/prpc/registry"
    //"fmt"
)

//func TestCreateClient(t *testing.T) {
//    config := &registry.RegConfigZooKeeper{ZKAddr: "localhost:2181"}
//    client := New(config)
//    if client == nil {
//        t.Fatal("error new client")
//    }
//    time.Sleep(60 * time.Second)
//}

//func TestClientCallRPC(t *testing.T) {
//    config := &registry.RegConfigZooKeeper{ZKAddr: "localhost:2181"}
//    client := New(config)
//    if client == nil {
//        t.Fatal("error new client")
//    }
//
//    rpcCli := client.Get("zone1001.gamesvr", 2)
//    if rpcCli == nil {
//        t.Fatal("error find rpc client")
//    }
//    var args Args
//    args.A = 2
//    args.B = 3
//    var reply int
//    err := rpcCli.Call("Arith.Multiply", &args, &reply)
//    if err != nil {
//        t.Fatalf("error call %v", err)
//    }
//    t.Logf("reply: %v", reply)
//}

//func TestClientSelect(t *testing.T) {
//    config := &registry.RegConfigZooKeeper{ZKAddr: "localhost:2181"}
//    client := New(config)
//    if client == nil {
//        t.Fatal("error new client")
//    }
//
//    status := make(map[int]int)
//    for i := 0; i < 1000; i++ {
//        rpcCli := client.Select("zone1001.gamesvr")
//        if rpcCli == nil {
//            t.Fatal("error find rpc client")
//        }
//        if _, exist := status[rpcCli.svc.ID.Index]; !exist {
//            status[rpcCli.svc.ID.Index] = 0
//        }
//        status[rpcCli.svc.ID.Index]++
//    }
//    for index, count := range status {
//        fmt.Printf("index: %v, count %v\n", index, count)
//    }
//
//    rpcCli2 := client.Select("invalid_group")
//    if rpcCli2 != nil {
//        t.Fatal("error find rpc client")
//    }
//}

func TestGetService(t *testing.T) {
    config := &registry.RegConfigZooKeeper{ZKAddr: "localhost:2181"}
    client := New(config)
    if client == nil {
        t.Fatal("error new client")
    }

    svc := client.Service("Game", "zone1001")
    svc2 := client.Service("Rank", "world1000", WithIndex(1))
    if svc == svc2 {
        t.Fatal("service be considered the same")
    }
    svc3 := client.Service("Rank", "world1000", WithIndex(1))
    if svc2 != svc3 {
        t.Fatal("service be considered different")
    }
    svc4 := client.Service("Game", "zone1002", WithVersion("v1.1"), WithSelectType(SelectTypeRandom))
    svc5 := client.Service("Game", "zone1002", WithVersion("v1.1"), WithSelectType(SelectTypeRandom))
    if svc4 != svc5 {
        t.Fatal("service be considered different")
    }
}
