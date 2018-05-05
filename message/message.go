package message

import (
    "encoding/binary"
    "errors"
    "fmt"
    "io"
    //"encoding/hex"

    "github.com/philipyao/prpc/codec"
    "github.com/philipyao/toolbox/util"
)

type MsgKind byte

const (
    MsgKindDefault   MsgKind = iota //默认rpc包
    MsgKindHeartbeat                //心跳包
)

type CompressKind byte

const (
    CompressKindNone CompressKind = iota
    CompressKindGzip
)

const (
    magicNumber = 9527
    msgVersion  = 0xA1

    DataCompressLen = 2048
)

var (
    ErrMagic           = errors.New("magic mismatch")
    ErrVersion         = errors.New("version mismatch")
    ErrUnpackHeartbeat = errors.New("unpack heartbeart to rpc")
    ErrInvLength       = errors.New("invalid total msg length")
)

var (
    seqno        = 0
    defaultCodec = codec.GetSerializer(codec.SerializeTypeMsgpack)
)

//magic(2) + ver(1) + len(2) + (msgkind+compresskind)(1) + seq(2)
type head [8]byte

func (h *head) initHead(mk MsgKind, seq uint16) {
    binary.BigEndian.PutUint16(h[0:], uint16(magicNumber))
    h[2] = byte(msgVersion)
    h[5] = (byte(mk) << 7) & 0x80
    binary.BigEndian.PutUint16(h[6:], seq)
}
func (h *head) setLength(length int) {
    //65535，最大65k数据
    binary.BigEndian.PutUint16(h[3:], uint16(length))
}
func (h *head) setCompressed() {
    ck := CompressKindGzip
    h[5] |= ((byte(ck) << 3) & 0x08)
}

func (h *head) IsHeartbeat() bool {
    mk := MsgKind((h[5] & 0x80) >> 7)
    return mk == MsgKindHeartbeat
}

func (h *head) IsDefault() bool {
    mk := MsgKind((h[5] & 0x80) >> 7)
    return mk == MsgKindDefault
}

func (h *head) isCompressed() bool {
    ck := CompressKind((h[5] & 0x08) >> 3)
    return ck == CompressKindGzip
}
func (h *head) magic() int {
    return int(binary.BigEndian.Uint16(h[0:]))
}
func (h *head) version() int {
    return int(h[2])
}
func (h *head) Seqno() uint16 {
    return binary.BigEndian.Uint16(h[6:])
}
func (h *head) length() int {
    return int(binary.BigEndian.Uint16(h[3:]))
}

type response struct {
    r   io.Reader
    rpc *msgRPC
}

type Message struct {
    head
    *response

    data []byte
}

type msgHeartbeat struct {
    Seqno uint `json:"seqno"`
}

type msgRPC struct {
    ServiceMethod string `json:"service_method"`
    Payload       []byte `json:"payload"` //rpc实际数据
}

func NewRequest(msgKind MsgKind, seqno uint16) *Message {
    msg := new(Message)
    msg.initHead(msgKind, seqno)
    return msg
}

func NewResponse(r io.Reader) (*Message, error) {
    msg := new(Message)
    msg.response = &response{r: r}
    err := msg.read()
    if err != nil {
        return nil, err
    }
    if msg.IsDefault() {
        //如果是默认（rpc）包，解析出method
        msg.response.rpc = new(msgRPC)
        err = defaultCodec.Decode(msg.data, msg.response.rpc)
        if err != nil {
            return nil, fmt.Errorf("decode body error %v", err)
        }

    }
    return msg, nil
}

//将v序列化为payload，并添加head后打包成二进制
func (m *Message) Pack(serviceMethod string, v interface{}, s codec.Serializer) ([]byte, error) {
    payload, err := s.Encode(v)
    if err != nil {
        return nil, err
    }

    rpc := &msgRPC{
        ServiceMethod: serviceMethod,
        Payload:       payload,
    }
    body, err := defaultCodec.Encode(rpc)
    if err != nil {
        return nil, err
    }
    //fmt.Printf("body len: %v\n", len(body))
    if len(body) > DataCompressLen {
        m.setCompressed()
        body = util.Compress(body)
        //fmt.Printf("pack after compress: body len %v\n", len(body))
    }
    hlen := len(m.head)
    dlen := hlen + len(body)
    m.data = make([]byte, dlen)
    //pack head len
    m.setLength(dlen)
    //fmt.Printf("head %s\n", hex.EncodeToString((m.head[:])))
    //fmt.Printf("body %s\n", hex.EncodeToString(body))
    copy(m.data[0:], m.head[:])
    copy(m.data[hlen:], body)
    return m.data, nil
}

func (m *Message) unpackHead() error {
    _, err := io.ReadFull(m.response.r, m.head[:])
    if err != nil {
        return err
    }
    //fmt.Printf("unpackHead: %s\n", hex.EncodeToString(m.head[:]))
    if m.magic() != magicNumber {
        return ErrMagic
    }
    if m.version() != msgVersion {
        return ErrVersion
    }
    length := m.length()
    //fmt.Printf("length: %d, hlen %v\n", length, len(m.head))
    if length <= len(m.head) {
        return ErrInvLength
    }
    return nil
}

func (m *Message) read() error {
    var err error
    //read head
    err = m.unpackHead()
    if err != nil {
        return err
    }
    //read body
    lenBody := m.length() - len(m.head)
    m.data = make([]byte, lenBody)
    _, err = io.ReadFull(m.response.r, m.data)
    if err != nil {
        return err
    }
    //fmt.Printf("unpack: body len %v\n", lenBody)
    if m.isCompressed() {
        m.data = util.Decompress(m.data)
        //fmt.Printf("unpack after decompress: body len %v\n", len(m.data))
    }
    return nil
}

//获取rpc的ServiceMethod
func (m *Message) ServiceMethod() string {
    if m.IsHeartbeat() {
        return ""
    }

    if m.response != nil && m.response.rpc != nil {
        return m.response.rpc.ServiceMethod
    }
    return ""
}

//把payload反序列化出来
func (m *Message) Unpack(s codec.Serializer, v interface{}) error {
    if m.IsHeartbeat() {
        return ErrUnpackHeartbeat
    }
    if m.response == nil || m.response.rpc == nil {
        return errors.New("no rpc found")
    }
    if len(m.response.rpc.Payload) == 0 {
        return errors.New("empty rpc payload")
    }
    return s.Decode(m.response.rpc.Payload, v)
}
