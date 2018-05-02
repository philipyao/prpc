package client

import (
    //"time"
    "testing"
    "github.com/philipyao/prpc/registry"
    //"fmt"
)

func TestClientCallRPC(t *testing.T) {
    config := &registry.RegConfigZooKeeper{ZKAddr: "localhost:2181"}
    client := New(config)
    if client == nil {
        t.Fatal("error new client")
    }

    svc := client.Service("Arith", "zone1001")
    if svc == nil {
        t.Fatal("error find rpc client")
    }
    var args Args
    args.A = 2
    args.B = 3
    var reply int
    err := svc.Call("Multiply", &args, &reply)
    if err != nil {
        t.Fatalf("error call %v", err)
    }
    t.Logf("reply: %v", reply)
}

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
