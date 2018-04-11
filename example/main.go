package main

import (
    "log"
    "sync"
    "github.com/philipyao/prpc/server"
)

type Args struct {
    A, B int
}

type Quotient struct {
    Quo, Rem int
}

type Arith int
func (t *Arith) Multiply(args *Args, reply *int) error {
    *reply = args.A * args.B
    return nil
}

func main() {
    var wg sync.WaitGroup

    zkAddr := "10.1.164.20:2181,10.1.164.20:2182"
    srv := server.New(zkAddr, "global.platsvr", 1, "10.1.164.99:7045")
    if srv != nil {
        err := srv.Register(new(Arith), "Arith")
        if err != nil {
            log.Printf("register error %v\n", err)
        } else {
            log.Printf("begin to serve\n", err)
            done := make(chan struct{}, 1)
            wg.Add(1)
            srv.Serve(done, &wg)
        }
    } else {
        log.Println("server.New error")
    }

    wg.Wait()
    log.Println("to stop")
}
