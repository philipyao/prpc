package registry

import (
    "time"
    "testing"
    //"github.com/philipyao/prpc/codec"
    "fmt"
    "sync"
)

//func TestRegister(t *testing.T) {
//    registry := New("localhost:2181")
//    if registry == nil {
//        t.Fatal("new registry error")
//    }
//
//    var err error
//    err = registry.Register(
//        "Game",
//        "Zone1001",
//        1,
//        "127.0.0.1:1234",
//    )
//    if err != nil {
//        t.Fatalf("Register error: %v", err)
//    }
//    err = registry.Register(
//        "Game",
//        "Zone1001",
//        2,
//        "127.0.0.1:1235",
//        WithWeight(20),
//    )
//    if err != nil {
//        t.Fatalf("Register error: %v", err)
//    }
//    err = registry.Register(
//        "Game",
//        "Zone1001",
//        3,
//        "127.0.0.1:1236",
//        WithWeight(5),
//        WithVersion("v1.1"),
//        WithSerialize(codec.SerializeTypeJson),
//    )
//    if err != nil {
//        t.Fatalf("Register error: %v", err)
//    }
//    err = registry.Register(
//        "Game",
//        "Zone1002",
//        1,
//        "127.0.0.1:2231",
//        WithWeight(20),
//    )
//    if err != nil {
//        t.Fatalf("Register error: %v", err)
//    }
//    err = registry.Register(
//        "Rank",
//        "World1000",
//        1,
//        "127.0.0.1:3231",
//    )
//    if err != nil {
//        t.Fatalf("Register error: %v", err)
//    }
//
//    time.Sleep(1 * time.Second)
//}
//
//func TestRegisterDuplicated(t *testing.T) {
//    reg := New("localhost:2181")
//    if reg == nil {
//        t.Fatal("new registry error")
//    }
//
//    var err error
//    err = reg.Register(
//        "Game",
//        "Zone1001",
//        11,
//        "127.0.0.1:8001",
//        WithVersion(DefaultVersion),
//    )
//    if err != nil {
//        t.Fatalf("Register error: %v", err)
//    }
//    err = reg.Register(
//        "Game",
//        "Zone1001",
//        11,
//        "127.0.0.1:8002",
//        WithVersion(DefaultVersion),
//    )
//    if err == nil {
//        t.Fatal("duplicated with no error")
//    }
//    fmt.Println(err)
//    time.Sleep(1 * time.Second)
//}
//
//func TestRegisterDecorateWrong(t *testing.T) {
//    reg := New("localhost:2181")
//    if reg == nil {
//        t.Fatal("new registry error")
//    }
//
//    var err error
//    err = reg.Register(
//        "Game",
//        "Zone1001",
//        1,
//        "127.0.0.1:8001",
//        WithWeight(-10),
//    )
//    if err == nil {
//        t.Fatal("decorate with no error")
//    }
//    fmt.Println(err)
//    err = reg.Register(
//        "Game",
//        "Zone1001",
//        2,
//        "127.0.0.1:8002",
//        WithWeight(maxWeight + 1),
//    )
//    if err == nil {
//        t.Fatal("decorate with no error")
//    }
//    fmt.Println(err)
//    err = reg.Register(
//        "Game",
//        "Zone1001",
//        3,
//        "127.0.0.1:8003",
//        WithVersion(""),
//    )
//    if err == nil {
//        t.Fatal("decorate with no error")
//    }
//    fmt.Println(err)
//    time.Sleep(1 * time.Second)
//}

func TestSubscribe(t *testing.T) {
    reg := New("localhost:2181")
    if reg == nil {
        t.Fatal("new registry error")
    }

    var err error

    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        reg2 := New("localhost:2181")
        if reg2 == nil {
            t.Fatal("new registry error")
        }
        err = reg2.Register(
            "Example",
            "ForSubscribe",
            1,
            "127.0.0.1:8001",
        )
        if err != nil {
            t.Fatal("register with error")
        }
        time.Sleep(2 * time.Second)
        err = reg2.Register(
            "Example",
            "ForSubscribe",
            2,
            "127.0.0.1:8002",
        )
        if err != nil {
            t.Fatal("register with error")
        }
        time.Sleep(1 * time.Second)
    }()

    time.Sleep(1 * time.Second)
    nodes, err := reg.Subscribe("Example", "ForSubscribe", new(tmpListener))
    if err != nil {
        t.Fatal(err)
    }
    fmt.Printf("subscribe, nodes: %+v\n", nodes)

    wg.Wait()
    reg.Close()
}

type tmpListener struct {}
func (tl *tmpListener) OnServiceChange(adds map[string]*Node, dels []string) {
    fmt.Printf("OnServiceChange: adds %+v, dels %+v\n", adds, dels)
}
func (tl *tmpListener) OnNodeChange(path string, node *Node) {
    fmt.Printf("OnNodeChange: path %v, node %+v\n", path, node)
}