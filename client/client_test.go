package client

import (
    "time"
    "testing"
)

//func TestClientCallRPC(t *testing.T) {
//    zkAddr := "10.1.164.20:2181,10.1.164.20:2182"
//    client := New(zkAddr)
//    if client == nil {
//        t.Fatal("error new client")
//    }
//
//    rpcCli := client.Get("global.platsvr", 1)
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
//    rpcCli.Close()
//}

func TestClientWatch(t *testing.T) {
    zkAddr := "10.1.164.20:2181,10.1.164.20:2182"
    client := New(zkAddr)
    if client == nil {
        t.Fatal("error new client")
    }
    time.Sleep(60 * time.Second)
}
