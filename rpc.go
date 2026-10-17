package sproto

import (
	"errors"
	"fmt"
	"reflect"
	"sync"
)

var (
	ErrRepeatedRpc     = errors.New("sproto rpc: repeated rpc")
	ErrUnknownProtocol = errors.New("sproto rpc: unknown protocol")
	ErrUnknownSession  = errors.New("sproto rpc: unknown session")
)

type RpcMode int

const (
	RpcRequestMode RpcMode = iota
	RpcResponseMode
)

type rpcHeader struct {
	Type    *int32 `sproto:"integer,0,name=type"`
	Session *int32 `sproto:"integer,1,name=session"`
}

type Protocol struct {
	Type     int32
	Name     string
	Request  reflect.Type
	Response reflect.Type
}

func (p *Protocol) HasRequest() bool {
	return p.Request != nil
}

func (p *Protocol) HasResponse() bool {
	return p.Response != nil
}

type Rpc struct {
	protocols    []*Protocol
	idMap        map[int32]int
	nameMap      map[string]int
	sessionMutex sync.Mutex
	sessions     map[int32]int
}

func getRpcSprotoType(typ reflect.Type) (*SprotoType, error) {
	if typ == nil {
		return nil, nil
	}

	if typ.Kind() != reflect.Ptr {
		return nil, ErrNonPtr
	}

	return GetSprotoType(typ.Elem())
}

func (rpc *Rpc) Dispatch(packed []byte) (mode RpcMode, name string, session int32, sp interface{}, err error) {
	var unpacked []byte
	if unpacked, err = Unpack(packed); err != nil {
		return
	}

	var used int
	header := rpcHeader{}
	if used, err = Decode(unpacked, &header); err != nil {
		return
	}

	var proto *Protocol
	if header.Type != nil {
		index, ok := rpc.idMap[*header.Type]
		if !ok {
			err = ErrUnknownProtocol
			return
		}
		proto = rpc.protocols[index]
		if proto.Request != nil {
			sp = reflect.New(proto.Request.Elem()).Interface()
			if _, err = Decode(unpacked[used:], sp); err != nil {
				return
			}
		}
		mode = RpcRequestMode
		if header.Session != nil {
			session = *header.Session
		}
	} else {
		if header.Session == nil {
			err = ErrUnknownSession
			return
		}
		session = *header.Session
		rpc.sessionMutex.Lock()
		defer rpc.sessionMutex.Unlock()
		index, ok := rpc.sessions[session]
		if !ok {
			err = ErrUnknownSession
			return
		}
		delete(rpc.sessions, session)

		proto = rpc.protocols[index]
		if proto.Response != nil {
			sp = reflect.New(proto.Response.Elem()).Interface()
			if _, err = Decode(unpacked[used:], sp); err != nil {
				return
			}
		}
		mode = RpcResponseMode
	}
	name = proto.Name
	return
}

// encodeRpcHeader 编码库内固定的两个 int32 字段，返回实际写出长度。
// tag 0/1 最多占六字节头部和两个八字节非 inline 段，总长不超过 22 字节。
// 栈内小缓冲只属于本次调用；整数与 tag 跳跃仍复用普通编码器的 wire 函数。
func encodeRpcHeader(header *rpcHeader, encodedHeader *[22]byte) (int, error) {
	limits, err := normalizeLimits(Limits{})
	if err != nil {
		return 0, err
	}
	budget := &decodeBudget{limits: limits}
	count := 0
	if header.Type != nil {
		count = 1
	}
	if header.Session != nil {
		count = 2
	}
	writeUint16(encodedHeader[:], uint16(count))
	headerOffset, bodyOffset, previous := 2, 2+count*2, -1
	for tag, number := range [2]*int32{header.Type, header.Session} {
		if number == nil {
			continue
		}
		if err := budget.add(1); err != nil {
			return 0, err
		}
		if skip := skipTag(previous, tag); skip > 0 {
			writeUint16(encodedHeader[headerOffset:], skip)
			headerOffset += 2
		}
		fieldHeader, width := encodeIntegerInto(uint64(int64(*number)), encodedHeader[bodyOffset:])
		writeUint16(encodedHeader[headerOffset:], fieldHeader)
		headerOffset += 2
		if width > 0 {
			writeUint32(encodedHeader[bodyOffset:], uint32(width))
			bodyOffset += 4 + width
		}
		previous = tag
	}
	// 私有 header 无嵌套，深度固定为一；normalizeLimits 已保证 MaxDepth 至少为一。
	// header 与 body 各自使用默认预算，不将二者改为共享额度。
	if bodyOffset > limits.MaxBytes {
		return 0, ErrLimit
	}
	return bodyOffset, nil
}

// packRpcFrame 将固定头与已编码 body 拼成独占帧，请求仍保留独立 body 的错误返回契约。
func packRpcFrame(header *rpcHeader, body []byte) ([]byte, error) {
	var encodedHeader [22]byte
	headerSize, err := encodeRpcHeader(header, &encodedHeader)
	if err != nil {
		return nil, err
	}
	size := headerSize + len(body)
	output := make([]byte, packedBufferSize(size))
	start := len(output) - size
	copy(output[start:], encodedHeader[:headerSize])
	copy(output[start+headerSize:], body)
	return packInto(output, output[start:]), nil
}

// ResponseEncode 将响应编码为独占的 Sproto RPC 帧，session 必须为正数。
func (rpc *Rpc) ResponseEncode(name string, session int32, response interface{}) (data []byte, err error) {
	if session <= 0 {
		return nil, ErrUnknownSession
	}
	index, ok := rpc.nameMap[name]
	if !ok {
		err = ErrUnknownProtocol
		return
	}

	protocol := rpc.protocols[index]
	if protocol.Response != nil {
		if reflect.TypeOf(response) != protocol.Response {
			return nil, ErrEncode
		}
		var encodedHeader [22]byte
		headerSize, err := encodeRpcHeader(&rpcHeader{Session: &session}, &encodedHeader)
		if err != nil {
			return nil, err
		}
		return encodeWithLimits(response, Limits{}, true, encodedHeader[:headerSize])
	}

	return packRpcFrame(&rpcHeader{Session: &session}, nil)
}

// RequestEncode 编码独占 RPC 请求帧；有响应的协议要求正 session，并登记有界等待。
func (rpc *Rpc) RequestEncode(name string, session int32, req interface{}) (data []byte, err error) {
	index, ok := rpc.nameMap[name]
	if !ok {
		err = ErrUnknownProtocol
		return
	}

	protocol := rpc.protocols[index]
	if protocol.Request != nil {
		if reflect.TypeOf(req) != protocol.Request {
			return nil, ErrEncode
		}
		if !protocol.HasResponse() {
			var encodedHeader [22]byte
			headerSize, err := encodeRpcHeader(&rpcHeader{Type: &protocol.Type}, &encodedHeader)
			if err != nil {
				return nil, err
			}
			return encodeWithLimits(req, Limits{}, true, encodedHeader[:headerSize])
		}
		if data, err = Encode(req); err != nil {
			return
		}
	}

	header := rpcHeader{
		Type: &protocol.Type,
	}

	if protocol.HasResponse() {
		if session <= 0 {
			return nil, ErrUnknownSession
		}
		rpc.sessionMutex.Lock()
		defer rpc.sessionMutex.Unlock()
		if _, ok := rpc.sessions[session]; ok {
			err = fmt.Errorf("sproto: repeated session:%d", session)
			return
		}
		// 仅有响应的请求需要独占 session 存储，避免让通知路径的参数一起逃逸。
		requestSession := session
		header.Session = &requestSession
		if len(rpc.sessions) >= 1024 {
			return nil, ErrLimit
		}
		rpc.sessions[session] = index
	}

	data, err = packRpcFrame(&header, data)
	return
}

// get protocol by name
func (rpc *Rpc) GetProtocolByName(name string) *Protocol {
	if index, ok := rpc.nameMap[name]; ok {
		return rpc.protocols[index]
	}
	return nil
}

func NewRpc(protocols []*Protocol) (*Rpc, error) {
	idMap := make(map[int32]int)
	nameMap := make(map[string]int)
	for i, protocol := range protocols {
		if protocol == nil {
			return nil, fmt.Errorf("%w: nil definition at index %d", ErrUnknownProtocol, i)
		}
		if _, err := getRpcSprotoType(protocol.Request); err != nil {
			return nil, err
		}
		if _, err := getRpcSprotoType(protocol.Response); err != nil {
			return nil, err
		}
		if _, ok := idMap[protocol.Type]; ok {
			return nil, ErrRepeatedRpc
		}
		if _, ok := nameMap[protocol.Name]; ok {
			return nil, ErrRepeatedRpc
		}
		idMap[protocol.Type] = i
		nameMap[protocol.Name] = i
	}
	rpc := &Rpc{
		protocols: protocols,
		idMap:     idMap,
		nameMap:   nameMap,
		sessions:  make(map[int32]int),
	}
	return rpc, nil
}

// CancelSession 撤销调用方不再等待的请求，发送失败、超时或断线时由会话 owner 调用。
func (rpc *Rpc) CancelSession(session int32) {
	rpc.sessionMutex.Lock()
	defer rpc.sessionMutex.Unlock()
	delete(rpc.sessions, session)
}

// ResetSessions 清理单连接的全部等待请求；不改变协议定义。
func (rpc *Rpc) ResetSessions() {
	rpc.sessionMutex.Lock()
	defer rpc.sessionMutex.Unlock()
	clear(rpc.sessions)
}
