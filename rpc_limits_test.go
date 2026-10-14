package sproto

import (
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
