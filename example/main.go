package main

import (
    "log"
    "sync"
    "time"
    "github.com/philipyao/prpc/server"
)

const (
    ZKAddr = "10.1.164.20:2181,10.1.164.20:2182"
    DefaultSrvWeight    = 1
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

    group := "global.platsvr"
    addr := "127.0.0.1:7045"
    srv := server.New(
        ZKAddr,
        group,
        DefaultSrvWeight,
        addr,
    )
    if srv == nil {
        log.Println("server.New error")
        return
    }

    err := srv.Register(new(Arith), "Arith")
    if err != nil {
        log.Printf("register error %v\n", err)
        return
    }

    log.Printf("begin to serve\n", err)
    wg.Add(1)
    go srv.Serve(&wg)
    go func(){
       time.Sleep(5 * time.Second)

        srv.Stop()
    }()

    wg.Wait()
    srv.Fini()
    log.Println("stopped.")
}
