package client

import (
	"fmt"
	"github.com/philipyao/prpc/registry"
	"testing"
)

func TestCallRPCVersion(t *testing.T) {
	config := &registry.RegConfigZooKeeper{ZKAddr: "localhost:2181"}
	client := New(config)
	if client == nil {
		t.Fatal("error new client")
	}

	var args Args
	args.A = 2
	args.B = 3
	var reply int

	var err error

	//match default version(v1.0)
	svc := client.Service("Arith", "zone1001")
	if svc == nil {
		t.Fatal("error find rpc client")
	}
	err = svc.Call("Multiply", &args, &reply)
	if err != nil {
		t.Fatalf("error call %v", err)
	}
	//match version v1.1
	svc2 := client.Service("Arith", "zone1001", WithVersion("v1.1"))
	if svc2 == nil {
		t.Fatal("error find rpc client")
	}
	err = svc2.Call("Multiply", &args, &reply)
	if err != nil {
		t.Fatalf("error call %v", err)
	}
	//match all version
	svc3 := client.Service("Arith", "zone1001", WithVersionAll())
	if svc3 == nil {
		t.Fatal("error find rpc client")
	}
	err = svc3.Call("Multiply", &args, &reply)
	if err != nil {
		t.Fatalf("error call %v", err)
	}
	//match invalid version
	svc4 := client.Service("Arith", "zone1001", WithVersion("unknown"))
	if svc4 == nil {
		t.Fatal("error find rpc client")
	}
	err = svc4.Call("Multiply", &args, &reply)
	if err != nil {
		t.Fatalf("error call %v", err)
	}
}

func TestCallRPCIndex(t *testing.T) {
	config := &registry.RegConfigZooKeeper{ZKAddr: "localhost:2181"}
	client := New(config)
	if client == nil {
		t.Fatal("error new client")
	}

	var args Args
	args.A = 2
	args.B = 3
	var reply int

	var err error

	//normal index
	svc := client.Service("Arith", "zone1001", WithIndex(1))
	if svc == nil {
		t.Fatal("error find rpc client")
	}
	err = svc.Call("Multiply", &args, &reply)
	if err != nil {
		t.Fatalf("error call %v", err)
	}
	//invalid index
	svc2 := client.Service("Arith", "zone1001", WithIndex(1000))
	if svc2 == nil {
		t.Fatal("error find rpc client")
	}
	err = svc2.Call("Multiply", &args, &reply)
	if err != nil {
		fmt.Println("error call ", err)
	}
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

func TestSelect(t *testing.T) {
	config := &registry.RegConfigZooKeeper{ZKAddr: "localhost:2181"}
	client := New(config)
	if client == nil {
		t.Fatal("error new client")
	}

	var args Args
	args.A = 2
	args.B = 3
	var reply int

	var err error
	for _, st := range []selectType{SelectTypeRandom, SelectTypeRoundRobin, SelectTypeWeightedRandom} {
		svc := client.Service("Arith", "zone1001", WithSelectType(st))
		for i := 0; i < 100; i++ {
			err = svc.Call("Multiply", &args, &reply)
			if err != nil {
				t.Fatalf("error call %v", err)
			}
		}
		svc.dumpMetrics()
	}
}
