package sproto

import (
	"bytes"
	"errors"
	"testing"
)

// 验证单连接 RPC 等待有界，超时或断线清理后可接纳新的请求。
func TestRpcPendingAdmissionAndCleanup(t *testing.T) {
	rpc, err := NewRpc(protocols)
	if err != nil {
		t.Fatal(err)
	}
	request := &FoobarRequest{What: String("request")}
	if _, err := rpc.RequestEncode("test.foobar", 0, request); !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("zero session: %v", err)
	}
	for session := int32(1); session <= 1024; session++ {
		if _, err := rpc.RequestEncode("test.foobar", session, request); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := rpc.RequestEncode("test.foobar", 1025, request); !errors.Is(err, ErrLimit) {
		t.Fatalf("pending limit: %v", err)
	}
	rpc.CancelSession(1)
	if _, err := rpc.RequestEncode("test.foobar", 1025, request); err != nil {
		t.Fatal(err)
	}
	rpc.ResetSessions()
	if _, err := rpc.RequestEncode("test.foobar", 1026, request); err != nil {
		t.Fatal(err)
	}
}

// 验证 nil 定义和重复协议返回可判断的错误，非法请求/响应类型不占用 session。
func TestRpcInvalidDefinitionsAndMessages(t *testing.T) {
	if _, err := NewRpc([]*Protocol{nil}); !errors.Is(err, ErrUnknownProtocol) {
		t.Fatalf("nil protocol: %v", err)
	}
	for _, field := range []string{"type", "name"} {
		first := *protocols[0]
		second := *protocols[1]
		switch field {
		case "type":
			second.Type = first.Type
		case "name":
			second.Name = first.Name
		}
		if _, err := NewRpc([]*Protocol{&first, &second}); !errors.Is(err, ErrRepeatedRpc) {
			t.Fatalf("duplicate %s: %v", field, err)
		}
	}
	rpc, err := NewRpc(protocols)
	if err != nil {
		t.Fatal(err)
	}
	request := &FoobarRequest{What: String("valid")}
	if _, err := rpc.RequestEncode("missing", 1, request); !errors.Is(err, ErrUnknownProtocol) {
		t.Fatalf("unknown request: %v", err)
	}
	for _, session := range []int32{0, -1} {
		if output, err := rpc.RequestEncode("test.foobar", session, request); !errors.Is(err, ErrUnknownSession) || output != nil {
			t.Fatalf("invalid session: %v", err)
		}
		if _, err := rpc.ResponseEncode("test.foobar", session, &FoobarResponse{}); !errors.Is(err, ErrUnknownSession) {
			t.Fatalf("invalid response session: %v", err)
		}
	}
	if _, err := rpc.RequestEncode("test.foobar", 1, &FooResponse{}); !errors.Is(err, ErrEncode) {
		t.Fatalf("invalid request payload: %v", err)
	}
	if _, err := rpc.RequestEncode("test.foobar", 1, request); err != nil {
		t.Fatalf("bad input consumed session: %v", err)
	}
	encoded, err := Encode(request)
	if err != nil {
		t.Fatal(err)
	}
	if body, err := rpc.RequestEncode("test.foobar", 1, request); err == nil || !bytes.Equal(body, encoded) || cap(body) != len(encoded) {
		t.Fatal("duplicate session must preserve the existing encoded-body error result")
	}
	if _, err := rpc.RequestEncode("test.foobar", 0, &FooResponse{}); !errors.Is(err, ErrEncode) {
		t.Fatalf("body error must precede request session validation: %v", err)
	}
	// body 错误必须先于 session 检查，且不能返回预备工作区的部分内容。
	var nilRequest *FoobarRequest
	for _, session := range []int32{0, -1, 1} {
		if output, err := rpc.RequestEncode("test.foobar", session, nilRequest); !errors.Is(err, ErrNil) || output != nil {
			t.Fatalf("typed nil request session=%d output=%d err=%v", session, len(output), err)
		}
	}
	if _, err := rpc.ResponseEncode("test.foobar", 1, &FooResponse{}); !errors.Is(err, ErrEncode) {
		t.Fatalf("invalid response payload: %v", err)
	}
	// 已声明 body 的 typed nil 是 ErrNil，不能因为头已准备就返回成功或部分帧。
	var empty *FoobarResponse
	for _, session := range []int32{1, 32767, MaxInt32} {
		output, err := rpc.ResponseEncode("test.foobar", session, empty)
		if !errors.Is(err, ErrNil) || output != nil {
			t.Fatalf("typed nil response session=%d output=%d err=%v", session, len(output), err)
		}
	}
	if _, err := rpc.ResponseEncode("missing", 1, &FoobarResponse{}); !errors.Is(err, ErrUnknownProtocol) {
		t.Fatalf("unknown response: %v", err)
	}
	unknown, err := Encode(&rpcHeader{Type: Int32(99)})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, err := rpc.Dispatch(Pack(unknown)); !errors.Is(err, ErrUnknownProtocol) {
		t.Fatalf("unknown incoming type: %v", err)
	}
	for _, malformed := range [][]byte{{255}, Pack([]byte{1}), Pack([]byte{1, 0, 0, 0, 0xff, 0xff, 0xff, 0xff})} {
		if _, _, _, _, err := rpc.Dispatch(malformed); err == nil {
			t.Fatalf("malformed RPC accepted: %x", malformed)
		}
	}
}

// 验证相同 session 在不同 Rpc 中隔离，取消、重置和重复回复不破坏其它连接。
func TestRpcSessionIsolationAndLateResponses(t *testing.T) {
	first, err := NewRpc(protocols)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewRpc(protocols)
	if err != nil {
		t.Fatal(err)
	}
	request := &FoobarRequest{What: String("request")}
	for _, rpc := range []*Rpc{first, second} {
		if _, err := rpc.RequestEncode("test.foobar", 1, request); err != nil {
			t.Fatal(err)
		}
	}
	reply, err := first.ResponseEncode("test.foobar", 1, &FoobarResponse{Ok: Bool(true)})
	if err != nil {
		t.Fatal(err)
	}
	first.CancelSession(1)
	if _, _, _, _, err := first.Dispatch(reply); !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("canceled session: %v", err)
	}
	mode, name, session, response, err := second.Dispatch(reply)
	if err != nil || mode != RpcResponseMode || name != "test.foobar" || session != 1 || !*response.(*FoobarResponse).Ok {
		t.Fatalf("other connection affected: %v %s %d %v", mode, name, session, err)
	}
	if _, _, _, _, err := second.Dispatch(reply); !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("duplicate response: %v", err)
	}
	if _, err := first.RequestEncode("test.foobar", 2, request); err != nil {
		t.Fatal(err)
	}
	late, err := first.ResponseEncode("test.foobar", 2, &FoobarResponse{})
	if err != nil {
		t.Fatal(err)
	}
	first.ResetSessions()
	if _, _, _, _, err := first.Dispatch(late); !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("response after close: %v", err)
	}
	if _, err := first.RequestEncode("test.bar", 0, nil); err != nil {
		t.Fatalf("notify: %v", err)
	}
}
