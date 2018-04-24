package main

import (
    "log"
    "sync"
    //"time"
    "github.com/philipyao/prpc/server"
    "github.com/philipyao/prpc/registry"
    "flag"
    "fmt"
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

var (
    group = flag.String("g", "", "server group")
    index = flag.Int("i", 1, "server index")
    ip = flag.String("I", "localhost", "server listening ip")
    port = flag.Int("p", 0, "server listening port")
    weight = flag.Int("w", registry.DefaultServiceWeight, "server weight")
)

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
        log.Println("server.New error")
        return
    }
    if *weight >= 0 {
        srv.SetWeight(*weight)
    }
    err := srv.Handle(new(Arith), "Arith")
    if err != nil {
        log.Printf("register error %v\n", err)
        return
    }

    log.Println("begin to serve")
    wg.Add(1)
    go srv.Serve(&wg, &registry.RegConfigZooKeeper{ZKAddr: ZKAddr})
    //go func(){
    //   time.Sleep(50 * time.Second)
	//
    //    srv.Stop()
    //}()

    wg.Wait()
    srv.Fini()
    log.Println("stopped.")
}
