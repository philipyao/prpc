package client

import (
    "testing"
    "github.com/philipyao/prpc/registry"
    "github.com/philipyao/prpc/codec"
)

type Args struct {
    A, B int
}

func TestCallRPC(t *testing.T) {
    svc := &registry.Service{
        ID: registry.SvcID{
            Group: "global.platsvr",
            Index: 1,
        },
        Addr: "localhost:8607",
        ServiceOption: &registry.ServiceOption{
            Weight: 1,
            Styp: int(codec.SerializeTypeMsgpack),
        },
    }
    cli := newRPC(svc)
    if cli == nil {
        t.Error("create rpc client error")
    }
    defer cli.Close()
    t.Logf("create rpc client ok: %+v", cli)

    var args Args
    args.A = 2
    args.B = 3
    var reply int
    cli.Call("Arith.Multiply", &args, &reply)
    t.Logf("reply: %+v", reply)
}