package main

import (
    "log"
    "sync"
    //"time"
    "flag"
    "fmt"
    "github.com/philipyao/prpc/registry"
    "github.com/philipyao/prpc/server"
    "time"
)

const (
    ZKAddr = "localhost:2181"
)

type Args struct {
    A, B int
}

type Arith int

func (t *Arith) Multiply(args *Args, reply *int) error {
    *reply = args.A * args.B
    log.Printf("args %+v, reply %v\n", args, *reply)

    if *index == 2 {
        //simulate timeout
        time.Sleep(3 * time.Second)
    }
    return nil
}

var (
    group   = flag.String("g", "", "server group")
    index   = flag.Int("i", 1, "server index")
    ip      = flag.String("I", "localhost", "server listening ip")
    port    = flag.Int("p", 0, "server listening port")
    weight  = flag.Int("w", 10, "server weight")
    version = flag.String("v", "v1.0", "server version")
)

func init() {
    log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
    var wg sync.WaitGroup

    flag.Parse()
    if *group == "" {
        panic("no group specified")
    }
    if *index <= 0 {
        panic("invalid index specified")
    }
    addr := fmt.Sprintf("%v:%v", *ip, *port)
    srv := server.New(
        *group,
        *index,
        addr,
    )
    if srv == nil {
        log.Fatal("server.New error, exit")
    }
    if *weight >= 0 {
        srv.SetWeight(*weight)
    }
    if *version != "" {
        srv.SetVersion(*version)
    }
    err := srv.Handle(new(Arith), "Arith")
    if err != nil {
        log.Fatalf("register error %v\n", err)
    }

    wg.Add(1)
    go srv.Serve(&wg, &registry.RegConfigZooKeeper{ZKAddr: ZKAddr})
    go func(){
       //time.Sleep(10 * time.Second)
       //
       // srv.Stop()
    }()

    wg.Wait()
    srv.Fini()
    log.Println("stopped.")
}
