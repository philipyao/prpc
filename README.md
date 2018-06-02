# PRPC
prpc is a rpc framework for game development

## Feature
* tcp transport
* service grouped by group id
* service with weight and version
* multiple encoding support, such as json, messagepack
* service discovery by zookeeper
* client selecting with select algorithm or specify concrete service by index
* cucuit breaker support

## Installation

```bash
go get -u -v github.com/philipyao/prpc/...
```


## Example
### client

```golang

import(
    "sys"
    "log"
    "github.com/philipyao/prpc/registry"
    "github.com/philipyao/prpc/client"
)

func main() {
    config := &registry.RegConfigZooKeeper{ZKAddr: "localhost:2181"}
    cli := client.New(config)
    if cli == nil {
        log.Println("error new client")
        sys.Exit(-1)
    }
    svc := cli.Service(
        "Arith",
        "zone1001"
        client.WithVersion("v1.1"),
    )
    args := Args{
        A: 2,
        B: 3,
    }
    var reply int
    err := svc.Call("Multiply", &args, &reply)
    if err != nil {
        log.Printf("error call %v", err)
    }
}

```

### server

```golang

import (
    "github.com/philipyao/prpc/registry"
    "github.com/philipyao/prpc/server"
)

func main() {
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

    wg.Add(1)
    go srv.Serve(&wg, addr, &registry.RegConfigZooKeeper{ZKAddr: ZKAddr})
}

```
