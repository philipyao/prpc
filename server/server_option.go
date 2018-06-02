package server

import (
    "github.com/philipyao/prpc/codec"
    "log"
)

//服务注册修饰项
type FnOptionServer func(srv *Server) error

func WithWeight(weight int) FnOptionServer {
    if weight < 0 {
        log.Println("invalid weight value")
        return nil
    }
    return func(srv *Server) error {
        srv.weight = weight
        return nil
    }
}
func WithSerialize(styp codec.SerializeType) FnOptionServer {
    return func(srv *Server) error {
        srv.styp = styp
        return nil
    }
}
func WithVersion(version string) FnOptionServer {
    if version == "" {
        log.Println("empty version not allowed")
        return nil
    }
    return func(srv *Server) error {
        srv.version = version
        return nil
    }
}

