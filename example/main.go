package main

import (
    "log"
    "sync"
    "time"
    "github.com/philipyao/prpc/server"
    "github.com/philipyao/prpc/registry"
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
    return nil
}

func main() {
    var wg sync.WaitGroup

    group := "zone1001.gamesvr"
    index := 2
    addr := "localhost:7045"
    srv := server.New(
        group,
        index,
        addr,
    )
    if srv == nil {
        log.Println("server.New error")
        return
    }

    err := srv.Handle(new(Arith), "Arith")
    if err != nil {
        log.Printf("register error %v\n", err)
        return
    }

    log.Println("begin to serve")
    wg.Add(1)
    go srv.Serve(&wg, &registry.RegConfigZooKeeper{ZKAddr: ZKAddr})
    go func(){
       time.Sleep(50 * time.Second)

        srv.Stop()
    }()

    wg.Wait()
    srv.Fini()
    log.Println("stopped.")
}
