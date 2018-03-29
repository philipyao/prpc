package message

import (
    "fmt"
    "io"
    //"bufio"
    "errors"
    "encoding/binary"
    "encoding/hex"
    "github.com/philipyao/prpc/codec"
)

type MsgKind byte
type CompressKind byte
const (
    MsgKindDefault MsgKind   = iota  //默认rpc包
    MsgKindHeartbeat                 //心跳包

    CompressKindNone CompressKind  = iota
    CompressKindGzip
)

const (
    magicNumber         = 9527
    msgVersion          = 0xA1
)

var (
    ErrMagic            = errors.New("magic mismatch")
    ErrVersion          = errors.New("version mismatch")
    ErrUnpackHeartbeat  = errors.New("unpack heartbeart to rpc")
    ErrInvLength        = errors.New("invalid total msg length")
)

//magic(2) + ver(1) + len(2) + (msgkind+compresskind)(1) + seq(2)
type head [8]byte
func (h *head) SetMeta(mk MsgKind, seq uint16) {
    binary.BigEndian.PutUint16(h[0:], uint16(magicNumber))
    h[2] = byte(msgVersion)
    h[5] = (byte(mk)<<7)&0x80
    binary.BigEndian.PutUint16(h[6:], seq)
}
func (h *head) setLength(length int) {
    //65535，最大65k数据
    binary.BigEndian.PutUint16(h[3:], uint16(length))
}
func (h *head) setCompressed() {
    ck := CompressKindGzip
    h[5] |= ((byte(ck)<<3)&0x08)
}

func (h *head) IsHeartbeat() bool {
    mk := MsgKind((h[5]&0x80)>>7)
    return mk == MsgKindHeartbeat
}
func (h *head) isCompressed() bool {
    ck := CompressKind((h[5]&0x08)>>3)
    return ck == CompressKindGzip
}
func (h *head) magic() int {
    return int(binary.BigEndian.Uint16(h[0:]))
}
func (h *head) version() int {
    return int(h[2])
}
func (h *head) Seq() uint16 {
    return binary.BigEndian.Uint16(h[6:])
}
func (h *head) length() int {
    return int(binary.BigEndian.Uint16(h[3:]))
}

type Message struct {
    head
    rpc         *msgRPC
    hbt         *msgHeartbeat

    data        []byte
}

type msgHeartbeat struct {
    Seqno       uint        `json:"seqno"`
}

type msgRPC struct {
    ServiceMethod   string          `json:"service_method"`
    V               interface{}     `json:"v"`
}

//将v序列化为payload，并添加head后打包成二进制
func (m *Message) Pack(serviceMethod string, v interface{}, s codec.Serializer) ([]byte, error) {
    m.rpc = &msgRPC{
        ServiceMethod:  serviceMethod,
        V:  v,
    }
    payload, err := s.Encode(m.rpc)
    if err != nil {
        return nil, err
    }
    fmt.Printf("payload: %v\n", string(payload))
    if len(payload) > 2048 {
        m.setCompressed()
        //todo compress
    }
    hlen := len(m.head)
    dlen := hlen + len(payload)
    m.data = make([]byte, dlen)
    //pack head len
    m.setLength(dlen)
    fmt.Printf("hlen %v dlen %v\n", hlen, dlen)
    fmt.Printf("head %s\n", hex.EncodeToString((m.head[:])))
    fmt.Printf("payload %s\n", hex.Dump(payload))
    copy(m.data[0:], m.head[:])
    copy(m.data[hlen:], payload)
    return m.data, nil
}

func (m *Message) UnpackHead(r io.Reader) error {
    _, err := io.ReadFull(r, m.head[:])
    if err != nil {
        return err
    }
    fmt.Printf("head %s\n", hex.EncodeToString(m.head[:]))
    if m.magic() != magicNumber {
        return ErrMagic
    }
    if m.version() != msgVersion {
        return ErrVersion
    }
    return nil
}

//从reader中读取head和payload，并把payload反序列化出来
func (m *Message) Unpack(r io.Reader, s codec.Serializer, v interface{}) error {
    if m.IsHeartbeat() {
        return ErrUnpackHeartbeat
    }
    length := m.length()
    fmt.Printf("length: %d, hlen %v\n", length, len(m.head))
    if length <= len(m.head) {
        return ErrInvLength
    }
    lenPayload := length - len(m.head)
    m.data = make([]byte, lenPayload)
    _, err := io.ReadFull(r, m.data)
    if err != nil {
        return err
    }
    rpc := &msgRPC{
        V: v,
    }
    err = s.Decode(m.data, rpc)
    if err != nil {
        return err
    }
    return nil
}

