package main

import (
    "log"
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
    flag.Parse()
    if *group == "" {
        panic("no group specified")
    }
    if *index <= 0 {
        panic("invalid index specified")
    }
    addr := fmt.Sprintf("%v:%v", *ip, *port)
    opts := []server.FnOptionServer{}
    if *weight >= 0 {
        opts = append(opts, server.WithWeight(*weight))
    }
    if *version != "" {
        opts = append(opts, server.WithVersion(*version))
    }

    srv := server.New(
        *group,
        *index,
        opts...
    )
    if srv == nil {
        log.Fatal("server.New error, exit")
    }
    err := srv.Handle(new(Arith), "Arith")
    if err != nil {
        log.Fatalf("register error %v\n", err)
    }

    err = srv.Serve(addr, &registry.RegConfigZooKeeper{ZKAddr: ZKAddr})
    if err != nil {
        log.Fatal(err)
    }

    waiter := make(chan struct{})
    go func(){
        time.Sleep(10 * time.Second)
        srv.Stop()
        srv.Fini()
        close(waiter)
    }()
    <- waiter
    log.Println("stopped.")
}
