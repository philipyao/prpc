package client

import (
//"testing"
//"github.com/philipyao/prpc/registry"
//"github.com/philipyao/prpc/codec"
)
import (
    "testing"
    "github.com/philipyao/prpc/codec"
    "context"
    "fmt"
    "time"
)

func TestRPCCall(t *testing.T) {
    if true {
        return
    }
    cli := newRPCClient("localhost:7881", codec.SerializeTypeMsgpack)
    if cli == nil {
        t.Fatal("create rpc client error")
    }
    defer cli.Close()
    fmt.Printf("create rpc client ok: %v\n", cli)

    var args Args
    args.A = 2
    args.B = 3
    var reply int
    cli.Call(context.Background(), "Arith.Multiply", &args, &reply)
    fmt.Printf("reply: %+v\n", reply)
}

func TestRPCGo(t *testing.T) {
    if true {
        return
    }
    cli := newRPCClient("localhost:7881", codec.SerializeTypeMsgpack)
    if cli == nil {
        t.Fatal("create rpc client error")
    }
    defer cli.Close()
    fmt.Printf("create rpc client ok: %v\n", cli)

    var args Args
    args.A = 2
    args.B = 3
    var reply int
    exit := make(chan struct{})
    callback := func(a interface{}, r interface{}, e error) {
        if e == nil {
            fmt.Printf("reply: %+v\n", r.(*int))
        } else {
            fmt.Printf("err %v\n", e)
        }
        close(exit)
    }
    cli.Go(context.Background(), "Arith.Multiply", &args, &reply, callback)
    <- exit
}

func TestRPCGoCancell(t *testing.T) {
    if true {
        return
    }
    cli := newRPCClient("localhost:7881", codec.SerializeTypeMsgpack)
    if cli == nil {
        t.Fatal("create rpc client error")
    }
    defer cli.Close()
    fmt.Printf("create rpc client ok: %v\n", cli)

    var args Args
    args.A = 2
    args.B = 3
    var reply int
    callback := func(a interface{}, r interface{}, e error) {
        time.Sleep(1 * time.Second)
        if e == nil {
            fmt.Printf("reply: %+v\n", r.(*int))
        } else {
            fmt.Printf("err %v\n", e)
        }
    }
    ctx, cancel := context.WithCancel(context.Background())
    cli.Go(ctx, "Arith.Multiply", &args, &reply, callback)
    cancel()
    time.Sleep(time.Second)
}

func TestRPCCallCancell(t *testing.T) {
    if true {
        return
    }
    cli := newRPCClient("localhost:7881", codec.SerializeTypeMsgpack)
    if cli == nil {
        t.Fatal("create rpc client error")
    }
    defer cli.Close()
    fmt.Printf("create rpc client ok: %v\n", cli)

    var args Args
    args.A = 2
    args.B = 3
    var reply int
    ctx, cancel := context.WithCancel(context.Background())
    go func() {
        time.Sleep(time.Second)
        err := cli.Call(ctx, "Arith.Multiply", &args, &reply)
        if err != nil {
            fmt.Println(err)
        }
    }()
    cancel()

    time.Sleep(2 * time.Second)
}
