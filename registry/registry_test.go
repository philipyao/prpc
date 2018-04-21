package registry

import (
    "time"
    "testing"
)

func TestRegistry(t *testing.T) {
	registry := New("localhost:2181")
    if registry == nil {
        t.Fatal("new registry error")
    }

    var err error
    err = registry.Register("", SvcID{Group: "group11", Index: 1}, "127.0.0.1", 1234, nil)
    if err != nil {
        t.Fatalf("Register error: %v", err)
    }

    builder := new(OptionBuilder).SetWeight(5)
    opt := builder.Build()
    err = registry.Register("", SvcID{Group: "group21", Index: 2}, "127.0.0.1", 5678, opt)
    if err != nil {
        t.Fatalf("Register error: %v", err)
    }

    builder = new(OptionBuilder).SetWeight(5).SetStyp(2)
    opt = builder.Build()
    err = registry.Register("", SvcID{Group: "group31", Index: 3}, "127.0.0.1", 9111, opt)
    if err != nil {
        t.Fatalf("Register error: %v", err)
    }

    time.Sleep(10 * time.Second)
}
