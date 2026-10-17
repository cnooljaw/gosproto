package sproto

import (
	"bytes"
	"errors"
	"math"
	"reflect"
	"testing"
)

type FoobarRequest struct {
	What *string `sproto:"string,0,name=what"`
}

type FoobarResponse struct {
	Ok   *bool   `sproto:"boolean,0,name=ok"`
	What *string `sproto:"string,1,name=what"`
}

type FooResponse struct {
	Ok *bool `sproto:"boolean,0,name=ok"`
}

var protocols []*Protocol = []*Protocol{
	&Protocol{
		Type:     1,
		Name:     "test.foobar",
		Request:  reflect.TypeOf(&FoobarRequest{}),
		Response: reflect.TypeOf(&FoobarResponse{}),
	},
	&Protocol{
		Type:     2,
		Name:     "test.foo",
		Response: reflect.TypeOf(&FooResponse{}),
	},
	&Protocol{
		Type: 3,
		Name: "test.bar",
	},
}

func checkRequest(rpc *Rpc, name string, session int32, sp interface{}) (interface{}, error) {
	chunk, err := rpc.RequestEncode(name, session, sp)
	if err != nil {
		return nil, err
	}

	mode, name1, session1, sp1, err := rpc.Dispatch(chunk)
	if err != nil {
		return nil, err
	}
	if mode != RpcRequestMode || name != name1 || session != session1 {
		return nil, errors.New("dipatch failed: unmatch meta info")
	}
	return sp1, nil
}

func TestRpcRequest(t *testing.T) {
	rpc, err := NewRpc(protocols)
	if err != nil {
		t.Fatalf("new rpc failed with error:%s", err)
	}

	// request & dispatch request
	what := "hello"
	sp, err := checkRequest(rpc, "test.foobar", 1, &FoobarRequest{What: &what})
	if err != nil {
		t.Fatalf("check request failed:%s", err)
	}
	req := sp.(*FoobarRequest)
	if *req.What != what {
		t.Fatalf("check failed: unmatch data")
	}

	// nil request
	sp, err = checkRequest(rpc, "test.foo", 2, nil)
	if err != nil {
		t.Fatalf("check request failed:%s", err)
	}
	if sp != nil {
		t.Fatalf("check failed: unmatch data")
	}

	// nil request
	sp, err = checkRequest(rpc, "test.bar", 0, nil)
	if err != nil {
		t.Fatalf("check request failed:%s", err)
	}
	if sp != nil {
		t.Fatalf("check failed: unmatch data")
	}
}

func TestRpcResponse(t *testing.T) {
	rpc, err := NewRpc(protocols)
	if err != nil {
		t.Fatalf("new rpc failed with error:%s", err)
	}

	// request & dispatch request
	what := "hello"
	_, err = rpc.RequestEncode("test.foobar", 18, &FoobarRequest{What: &what})
	if err != nil {
		t.Fatalf("request encode failed:%s", err)
	}

	// check response
	chunk, err := rpc.ResponseEncode("test.foobar", 18, &FoobarResponse{Ok: Bool(true)})
	if err != nil {
		t.Fatalf("response encode failed:%s", err)
	}

	mode, name, session, sp, err := rpc.Dispatch(chunk)
	if err != nil {
		t.Fatalf("dispatch failed:%s", err)
	}
	if mode != RpcResponseMode || name != "test.foobar" || session != 18 {
		t.Fatalf("dispatch failed:unmatch meta info")
	}
	response := sp.(*FoobarResponse)
	if !*response.Ok {
		t.Fatalf("dispatch failed:unmatch data")
	}
}

// RPC 组帧在 header 整数边界、FF 长段和无 body 协议上保持独立编码的 wire/cap 与输出独占。
func TestRpcFrameAssembly(t *testing.T) {
	for _, code := range []int32{-1, 0, 1, 32766, 32767, math.MaxInt32} {
		for _, size := range []int{0, 1, 7, 8, 31, 2047, 2048, 2049, 65537} {
			input := &packedBufferMessage{Payload: bytes.Repeat([]byte{0xa5}, size)}
			body, err := Encode(input)
			if err != nil {
				t.Fatal(err)
			}
			bodyType := reflect.TypeOf(input)
			for _, role := range []struct{ request, response reflect.Type }{
				{bodyType, bodyType}, {bodyType, nil}, {nil, bodyType}, {},
			} {
				rpc, err := NewRpc([]*Protocol{{Type: code, Name: "frame", Request: role.request, Response: role.response}})
				if err != nil {
					t.Fatal(err)
				}
				for _, session := range []int32{1, 32766, 32767, math.MaxInt32} {
					header := &rpcHeader{Type: &code}
					if role.response != nil {
						header.Session = &session
					}
					head, err := Encode(header)
					if err != nil {
						t.Fatal(err)
					}
					raw := bytes.Clone(head)
					if role.request != nil {
						raw = append(raw, body...)
					}
					want := Pack(raw)
					request, err := rpc.RequestEncode("frame", session, input)
					if err != nil || !bytes.Equal(request, want) || cap(request) != cap(want) {
						t.Fatalf("request code=%d session=%d size=%d err=%v", code, session, size, err)
					}
					rpc.CancelSession(session)
					snapshot := bytes.Clone(request)
					head, err = Encode(&rpcHeader{Session: &session})
					if err != nil {
						t.Fatal(err)
					}
					raw = bytes.Clone(head)
					if role.response != nil {
						raw = append(raw, body...)
					}
					want = Pack(raw)
					response, err := rpc.ResponseEncode("frame", session, input)
					if err != nil || !bytes.Equal(response, want) || cap(response) != cap(want) || !bytes.Equal(request, snapshot) ||
						!bytes.Equal(input.Payload, bytes.Repeat([]byte{0xa5}, size)) {
						t.Fatalf("response/ownership code=%d session=%d size=%d err=%v", code, session, size, err)
					}
				}
			}
		}
	}
}

// RPC 的 header 与 body 保留各自默认预算，body 恰到上限时不能因新增 header 而被拒绝。
func TestRpcFrameBodyBudget(t *testing.T) {
	input := &packedBufferMessage{Payload: make([]byte, (8<<20)-8)}
	rpc, err := NewRpc([]*Protocol{{Type: 1, Name: "budget", Request: reflect.TypeOf(input), Response: reflect.TypeOf(input)}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rpc.RequestEncode("budget", 1, input); err != nil {
		t.Fatal(err)
	}
	if _, err := rpc.ResponseEncode("budget", 1, input); err != nil {
		t.Fatal(err)
	}
	rpc.CancelSession(1)
	input.Payload = append(input.Payload, 0)
	if output, err := rpc.RequestEncode("budget", 1, input); !errors.Is(err, ErrLimit) || output != nil {
		t.Fatalf("request over budget: output=%d error=%v", len(output), err)
	}
	if output, err := rpc.ResponseEncode("budget", 1, input); !errors.Is(err, ErrLimit) || output != nil {
		t.Fatalf("response over budget: output=%d error=%v", len(output), err)
	}
}

// RPC 组合工作区比分步 header 编码、Append 和 Pack 少分配，避免重新引入拼接缓冲。
func TestRpcFrameAllocation(t *testing.T) {
	input := &packedBufferMessage{Payload: bytes.Repeat([]byte{0x91}, 256)}
	code := int32(1)
	rpc, err := NewRpc([]*Protocol{{Type: code, Name: "notify", Request: reflect.TypeOf(input)}})
	if err != nil {
		t.Fatal(err)
	}
	var output []byte
	assembled := testing.AllocsPerRun(100, func() {
		var err error
		output, err = rpc.RequestEncode("notify", 0, input)
		if err != nil {
			t.Fatal(err)
		}
	})
	separate := testing.AllocsPerRun(100, func() {
		body, err := Encode(input)
		if err != nil {
			t.Fatal(err)
		}
		header, err := Encode(&rpcHeader{Type: &code})
		if err != nil {
			t.Fatal(err)
		}
		output = Pack(Append(header, body))
	})
	if len(output) == 0 || assembled >= separate {
		t.Fatalf("RPC=%v separate=%v, want fewer allocations", assembled, separate)
	}
}
