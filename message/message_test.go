package message

import (
    "testing"
    "bytes"
    //"bufio"
    "github.com/philipyao/prpc/codec"
)

func TestRpcJson(t *testing.T) {
    type MyRPC struct {
        Text    string          `json:"text"`
        Integer    int          `json:"integer"`
    }
    rpc := &MyRPC{
        Text:  "hello",
        Integer: 666,
    }

    msg := NewRequest(MsgKindDefault)
    s := codec.GetSerializer(codec.SerializeTypeJson)
    data, err := msg.Pack("Demo.func", rpc, s)
    if err != nil {
        t.Errorf("pack error: %v", err)
    }
    t.Logf("pack ok, data len %v", len(data))

    r := bytes.NewReader(data)
    rmsg := NewResponse(r)
    if rmsg.IsHeartbeat() {
        t.Logf("heartbeart received!")
    } else {
        var rrpc MyRPC
        err = rmsg.Unpack(s, &rrpc)
        if err != nil {
            t.Errorf("Unpack error: %v", err)
        }
        t.Logf("rrpc received: %+v!", rrpc)
    }
}

func TestRpcMsgpack(t *testing.T) {
    //msgpack依然使用json的tag
    type MyRPC struct {
        Text    string          `json:"text"`
        Integer    int          `json:"integer"`
    }
    rpc := &MyRPC{
        Text:  "world",
        Integer: 888,
    }

    msg := NewRequest(MsgKindDefault)
    s := codec.GetSerializer(codec.SerializeTypeMsgpack)
    data, err := msg.Pack("Demo.func", rpc, s)
    if err != nil {
        t.Errorf("pack error: %v", err)
    }
    t.Logf("pack ok, data len %v", len(data))

    r := bytes.NewReader(data)
    rmsg := NewResponse(r)
    if rmsg.IsHeartbeat() {
        t.Logf("heartbeart received!")
    } else {
        var rrpc MyRPC
        err = rmsg.Unpack(s, &rrpc)
        if err != nil {
            t.Errorf("Unpack error: %v", err)
        }
        t.Logf("rrpc received: %+v!", rrpc)
    }
}

func TestRpcCompress(t *testing.T) {
    type MyRPC struct {
        Text       []string       `json:"text"`
        Integer    []int          `json:"integer"`
    }
    rpc := &MyRPC{}
    for i := 0; i < 200; i++ {
        rpc.Text = append(rpc.Text, "lslfdfs")
        rpc.Integer = append(rpc.Integer, i)
    }

    msg := NewRequest(MsgKindDefault)
    s := codec.GetSerializer(codec.SerializeTypeJson)
    data, err := msg.Pack("Demo.func", rpc, s)
    if err != nil {
        t.Errorf("pack error: %v", err)
    }
    t.Logf("pack ok, data len %v", len(data))

    r := bytes.NewReader(data)
    rmsg := NewResponse(r)
    if rmsg.IsHeartbeat() {
        t.Logf("heartbeart received!")
    } else {
        var rrpc MyRPC
        err = rmsg.Unpack(s, &rrpc)
        if err != nil {
            t.Errorf("Unpack error: %v", err)
        }
        t.Logf("rrpc received: %+v!", rrpc)
    }
}