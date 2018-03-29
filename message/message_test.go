package message

import (
    "testing"
    "bytes"
    //"bufio"
    "github.com/philipyao/prpc/codec"
)

func TestMessageRpc(t *testing.T) {
    type MyRPC struct {
        Text    string          `json:"text"`
        Integer    int         `json:"integer"`
    }
    rpc := &MyRPC{
        Text:  "hello",
        Integer: 666,
    }

    msg := new(Message)
    var seq uint16 = 1097
    msg.SetMeta(MsgKindDefault, seq)
    s := codec.GetSerializer(codec.SerializeTypeJson)
    data, err := msg.Pack("Demo.func", rpc, s)
    if err != nil {
        t.Errorf("pack error: %v", err)
    }
    t.Logf("pack ok, data len %v", len(data))

    rmsg := new(Message)
    r := bytes.NewReader(data)
    //bufr := bufio.NewReader(r)
    err = rmsg.UnpackHead(r)
    if err != nil {
        t.Errorf("UnpackHead error: %v", err)
    }
    if rmsg.IsHeartbeat() {
        t.Logf("heartbeart received!")
    } else {
        var rrpc MyRPC
        err = rmsg.Unpack(r, s, &rrpc)
        if err != nil {
            t.Errorf("Unpack error: %v", err)
        }
        t.Logf("rrpc received: %+v!", rrpc)
    }
}
