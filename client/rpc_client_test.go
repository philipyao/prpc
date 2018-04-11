package client

import (
    "testing"
    "github.com/philipyao/toolbox/zkcli"
)

func TestCreateRPCClient(t *testing.T) {
    zkAddr := "10.1.164.20:2181,10.1.164.20:2182"
    zkConn, err := zkcli.Connect(zkAddr)
    if err != nil {
        t.Fatalf("zk connect returned error: %v", err)
    }
    defer zkConn.Close()

    path := DefaultZKPath + "/" + "testnode1"
    //addr + styp + weight
    nodeVal := "10.1.164.45:8607" + "|" + "2" + "|" + "1"
    cli := newRPC(zkConn, path, nodeVal)
    if cli != nil {
        t.Logf("cli: %+v", cli)
        cli.Close()
    } else {
        t.Error("create rpc client error")
    }
}

func TestCreateRPCClient2(t *testing.T) {
    zkAddr := "10.1.164.20:2181,10.1.164.20:2182"
    zkConn, err := zkcli.Connect(zkAddr)
    if err != nil {
        t.Fatalf("zk connect returned error: %v", err)
    }
    defer zkConn.Close()

    path := DefaultZKPath + "/" + "testnode2"
    //addr + styp + weight
    nodeVal := "10.1.164.45:8607" + "2" + "|" + "1"
    cli := newRPC(zkConn, path, nodeVal)
    if cli != nil {
        t.Logf("cli: %+v", cli)
        cli.Close()
    } else {
        t.Error("create rpc client error")
    }
}

func TestCreateRPCClient3(t *testing.T) {
    zkAddr := "10.1.164.20:2181,10.1.164.20:2182"
    zkConn, err := zkcli.Connect(zkAddr)
    if err != nil {
        t.Fatalf("zk connect returned error: %v", err)
    }
    defer zkConn.Close()

    path := DefaultZKPath + "/" + "testnode3"
    //addr + styp + weight
    nodeVal := " 12344" + "|" + " 2" + "|" + "1 "
    cli := newRPC(zkConn, path, nodeVal)
    if cli != nil {
        t.Logf("cli: %+v", cli)
        cli.Close()
    } else {
        t.Error("create rpc client error")
    }
}

func TestCreateRPCClient4(t *testing.T) {
    zkAddr := "10.1.164.20:2181,10.1.164.20:2182"
    zkConn, err := zkcli.Connect(zkAddr)
    if err != nil {
        t.Fatalf("zk connect returned error: %v", err)
    }
    defer zkConn.Close()

    path := DefaultZKPath + "/" + "testnode3"
    //addr + styp + weight
    nodeVal := "10.1.164.45:8607" + "|" + " 8" + "|" + "1 "
    cli := newRPC(zkConn, path, nodeVal)
    if cli != nil {
        t.Logf("cli: %+v", cli)
        cli.Close()
    } else {
        t.Error("create rpc client error")
    }
}

func TestCreateRPCClient5(t *testing.T) {
    zkAddr := "10.1.164.20:2181,10.1.164.20:2182"
    zkConn, err := zkcli.Connect(zkAddr)
    if err != nil {
        t.Fatalf("zk connect returned error: %v", err)
    }
    defer zkConn.Close()

    path := DefaultZKPath + "/" + "testnode3"
    //addr + styp + weight
    nodeVal := "10.1.164.45:8607" + "|" + " 1" + "|" + "02d "
    cli := newRPC(zkConn, path, nodeVal)
    if cli != nil {
        t.Logf("cli: %+v", cli)
        cli.Close()
    } else {
        t.Error("create rpc client error")
    }
}

type Args struct {
    A, B int
}

func TestCallRPC(t *testing.T) {
    zkAddr := "10.1.164.20:2181,10.1.164.20:2182"
    zkConn, err := zkcli.Connect(zkAddr)
    if err != nil {
        t.Fatalf("zk connect returned error: %v", err)
    }
    defer zkConn.Close()

    path := DefaultZKPath + "/" + "global.platsvr.1"
    //addr + styp + weight
    nodeVal := "10.1.164.99:7045|1|1"
    cli := newRPC(zkConn, path, nodeVal)
    if cli != nil {
        t.Logf("cli: %+v", cli)

        var args Args
        args.A = 2
        args.B = 3
        var reply int
        cli.Call("Arith.Multiply", &args, &reply)
        t.Logf("reply: %+v", reply)
        cli.Close()
    } else {
        t.Error("create rpc client error")
    }
}