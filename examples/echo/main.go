// Echo 展示原生 Sproto Request/Response 的字节往返，不包含连接或调度实现。
package main

import (
	"fmt"
	"reflect"

	sproto "github.com/cnooljaw/gosproto"
)

type echoRequest struct {
	Text string `sproto:"string,0"`
}

type echoResponse struct {
	Text string `sproto:"string,0"`
}

func exchange() (string, error) {
	definitions := []*sproto.Protocol{{
		Type: 1, Name: "echo",
		Request: reflect.TypeOf((*echoRequest)(nil)), Response: reflect.TypeOf((*echoResponse)(nil)),
	}}
	client, err := sproto.NewRpc(definitions)
	if err != nil {
		return "", err
	}
	server, err := sproto.NewRpc(definitions)
	if err != nil {
		return "", err
	}
	request, err := client.RequestEncode("echo", 1, &echoRequest{Text: "中文🙂"})
	if err != nil {
		return "", err
	}
	defer client.CancelSession(1)
	mode, name, session, payload, err := server.Dispatch(request)
	if err != nil {
		return "", err
	}
	if mode != sproto.RpcRequestMode || name != "echo" || session != 1 {
		return "", fmt.Errorf("unexpected request: mode=%v name=%s session=%d", mode, name, session)
	}
	reply, err := server.ResponseEncode(name, session, &echoResponse{Text: payload.(*echoRequest).Text})
	if err != nil {
		return "", err
	}
	mode, name, session, payload, err = client.Dispatch(reply)
	if err != nil {
		return "", err
	}
	if mode != sproto.RpcResponseMode || name != "echo" || session != 1 {
		return "", fmt.Errorf("unexpected response: mode=%v name=%s session=%d", mode, name, session)
	}
	return payload.(*echoResponse).Text, nil
}

func main() {
	text, err := exchange()
	if err != nil {
		panic(err)
	}
	fmt.Println(text)
}
