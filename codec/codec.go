package codec

import (
    "bytes"
    "encoding/json"
    "github.com/vmihailenco/msgpack"
)

type SerializeType byte
const (
    SerializeTypeNone       SerializeType    = iota
    SerializeTypeMsgpack
    SerializeTypeJson
)

var (
    serializers = map[SerializeType]Serializer{
        SerializeTypeMsgpack: &msgpackSerializer{},
        SerializeTypeJson: &jsonSerializer{},
    }
)

type Serializer interface {
    Encode(v interface{}) ([]byte, error)
    Decode(data []byte, v interface{}) error
}

type jsonSerializer struct {}
func (js jsonSerializer) Encode(v interface{}) ([]byte, error) {
    return json.Marshal(v)
}
func (js jsonSerializer) Decode(data []byte, v interface{}) error {
    return json.Unmarshal(data, v)
}

type msgpackSerializer struct {}
func (ms msgpackSerializer) Encode(v interface{}) ([]byte, error) {
    var buf bytes.Buffer
    enc := msgpack.NewEncoder(&buf)
    //UseJSONTag causes the Decoder to use json struct tag as fallback option if there is no msgpack tag.
    enc.UseJSONTag(true)
    err := enc.Encode(v)
    if err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}
func (ms msgpackSerializer) Decode(data []byte, v interface{}) error {
    buf := bytes.NewBuffer(data)
    dec := msgpack.NewDecoder(buf)
    //UseJSONTag causes the Decoder to use json struct tag as fallback option if there is no msgpack tag.
    dec.UseJSONTag(true)
    return dec.Decode(v)
}

func GetSerializer(styp SerializeType) Serializer {
    return serializers[styp]
}