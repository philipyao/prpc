package client

import (
//"testing"
//"github.com/philipyao/prpc/registry"
//"github.com/philipyao/prpc/codec"
)


//func TestCallRPC(t *testing.T) {
//    svc := &registry.Service{
//        ID: registry.SvcID{
//            Group: "zone1001.gamesvr",
//            Index: 2,
//        },
//        Addr: "localhost:7045",
//        ServiceOption: &registry.ServiceOption{
//            Weight: 1,
//            Styp: int(codec.SerializeTypeMsgpack),
//        },
//    }
//    cli := newRPC(svc)
//    if cli == nil {
//        t.Fatal("create rpc client error")
//    }
//    defer cli.Close()
//    t.Logf("create rpc client ok: %+v", cli)
//
//    var args Args
//    args.A = 2
//    args.B = 3
//    var reply int
//    cli.Call("Arith.Multiply", &args, &reply)
//    t.Logf("reply: %+v", reply)
//}
