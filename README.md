# gosproto

Go 实现的 [Sproto](https://github.com/cloudwu/sproto) 编解码库，支持结构体编码、0 packing 和原生 `type/session` RPC。运行时使用纯 Go，不需要 C 或 Lua。库只处理协议字节、类型与 session 关联，不管理连接、网络分帧、业务 Handler、goroutine 或阻塞调用。

## 安装

需要 Go 1.24 或更高版本。

```sh
go get github.com/cnooljaw/gosproto@v0.4.0
```

## 快速开始

用 Go struct tag 描述字段，字段编号与 `.sproto` 保持一致：

```go
package main

import (
	"fmt"

	sproto "github.com/cnooljaw/gosproto"
)

type Person struct {
	Name string   `sproto:"string,0"`
	ID   int64    `sproto:"integer,1"`
	Tags []string `sproto:"string,2,array"`
}

func main() {
	input := &Person{Name: "Alice", ID: 10000, Tags: []string{"player", "中文"}}
	encoded, err := sproto.Encode(input)
	if err != nil {
		panic(err)
	}
	var output Person
	used, err := sproto.Decode(encoded, &output)
	if err != nil {
		panic(err)
	}
	fmt.Printf("decoded %d bytes: %+v\n", used, output)
}
```

对应的 Sproto 定义：

```text
.Person {
    name 0 : string
    id 1 : integer
    tags 2 : *string
}
```

## 协议定义与类型

字段 tag 格式为 `sproto:"类型,编号[,array][,name=字段名]"`。编号从 0 开始，同一结构体内不得重复；编码按编号排序。没有 `sproto` tag 的字段不参与编解码。

| Sproto 类型 | Go 类型 | Tag 示例 |
| --- | --- | --- |
| string | `string`、`*string` | `sproto:"string,0"` |
| binary | `[]byte` | `sproto:"string,0"` |
| integer | `int`、`int8`、`int16`、`int32`、`int64` 及其指针 | `sproto:"integer,0"` |
| integer | 对应 unsigned 类型及其指针，数值限于 `0..MaxInt64` | `sproto:"integer,0"` |
| boolean | `bool`、`*bool` | `sproto:"boolean,0"` |
| double | `float64`、`*float64` | `sproto:"double,0"` |
| 用户定义结构 | `*Struct` | `sproto:"struct,0"` |
| string 数组 | `[]string` | `sproto:"string,0,array"` |
| integer 数组 | 上述整数类型的切片 | `sproto:"integer,0,array"` |
| boolean 数组 | `[]bool` | `sproto:"boolean,0,array"` |
| double 数组 | `[]float64` | `sproto:"double,0,array"` |
| 结构数组 | `[]*Struct` | `sproto:"struct,0,array"` |

Sproto integer 是有符号 64 位整数；解码到较窄的 Go 类型发生溢出时返回错误。double 使用 IEEE-754 位模式，不能用整数字段的缩放代替。

值字段的零值会参与编码；nil 指针和 nil 切片表示字段缺失，非 nil 空切片表示存在的空数组。需要区分“缺失”和“零值”时使用指针。结构字段使用指针，集合使用切片，不使用指向切片的指针。字段必须导出。

可以手写上述类型，也可以通过 [sprotodump](https://github.com/lvzixun/sprotodump) 从 `.sproto` 生成类型和 Protocol 表；生成后检查导入路径为 `github.com/cnooljaw/gosproto`，并移除旧模板输出的 `MethodName` 字段；它是 Go 方法绑定元数据，不属于 Sproto。本库不解析 schema 文本或官方二进制 schema，不提供 Go map 映射、binary 数组、`float32` 或定点数的自动缩放。Lua 的弱类型转换不适用于 Go 的强类型字段。

## 编解码 API

- `Encode(message)`：编码结构体指针，返回未打包的字节。
- `Decode(data, target)`：解码到非 nil 结构体指针，返回消费的字节数。
- `EncodeWithLimits(message, limits)` / `DecodeWithLimits(data, target, limits)`：显式设置资源预算。
- `EncodePacked(message)` / `DecodePacked(data, target)`：组合结构体编解码与 Pack/Unpack，使用默认预算。
- `GetSprotoType(reflect.Type)`：检查 struct 的字段定义并缓存编码信息。
- `MustEncode` / `MustDecode`：遇到错误时 panic，只适合输入受程序控制的场景。

Decode 成功后替换目标结构体，未出现的字段回到 Go 零值；解码失败保留目标原值。未知字段安全跳过，便于增删字段时保持线格式兼容。修改已有字段的编号或物理类型不属于安全的协议演进。

编码先做有界尺寸预检，再按确切长度分配独占输出；嵌套和数组直接写入最终缓冲。仅在一次调用内复用头项数和数组宽度，不按对象地址跨调用缓存消息值；大型消息可使用额外的有界布局记录。安全类型化访问可能使临时输入对象逃逸，测量时应包含真实调用方的对象构造。解码不通过共享缓冲复用输出对象，失败时仍保留目标原值。

Decode 不要求输入恰好只有一条结构体；原生 RPC 的 package 和 body 可顺序解码，调用方根据返回的消费长度处理后续内容或检查尾部。

## 0 packing

`Pack(data)` 压缩零字节；`Unpack(data)` 恢复字节。解包结果按 8 字节对齐，最后可能有最多 7 个零字节填充；Pack 不是加密、认证或网络分帧。

```go
packed := sproto.Pack(encoded)
unpacked, err := sproto.UnpackWithLimit(packed, 1<<20)
if err != nil {
    return err
}
```

`UnpackWithLimit` 先检查完整 packed 边界与有限输出长度，再一次分配输出。当前实现的 maxBytes 只约束解包后含填充的长度，等于预算时允许接纳；packed 输入上限由连接或调用方另行设置。Pack 的输入由调用方限制；不应直接对无界输入调用 Pack。

## RPC

RPC 消息使用 Sproto 原生 package：

```text
.package {
    type 0 : integer
    session 1 : integer
}

echo 1 {
    request { text 0 : string }
    response { text 0 : string }
}
```

用 `Protocol` 描述协议编号、名称和 Request/Response 类型，传给 `NewRpc`。定义必须非 nil，Type 和 Name 各自唯一；Request/Response 为结构体指针的 `reflect.Type`。

下面的函数演示两个独立端点处理一次请求；实际项目把 request/reply 字节交给自己的连接发送：

```go
func echoRoundTrip() (string, error) {
	type Echo struct {
		Text string `sproto:"string,0"`
	}
	definitions := []*sproto.Protocol{{
		Type: 1, Name: "echo",
		Request:  reflect.TypeOf((*Echo)(nil)),
		Response: reflect.TypeOf((*Echo)(nil)),
	}}
	client, err := sproto.NewRpc(definitions)
	if err != nil {
		return "", err
	}
	server, err := sproto.NewRpc(definitions)
	if err != nil {
		return "", err
	}
	request, err := client.RequestEncode("echo", 1, &Echo{Text: "hello"})
	if err != nil {
		return "", err
	}
	defer client.CancelSession(1)
	mode, name, session, message, err := server.Dispatch(request)
	if err != nil {
		return "", err
	}
	if mode != sproto.RpcRequestMode {
		return "", fmt.Errorf("unexpected RPC mode: %v", mode)
	}
	reply, err := server.ResponseEncode(name, session, message.(*Echo))
	if err != nil {
		return "", err
	}
	_, _, _, response, err := client.Dispatch(reply)
	if err != nil {
		return "", err
	}
	return response.(*Echo).Text, nil
}
```

此函数需要导入 `fmt`、`reflect` 和 `sproto`。同一协议的 Request/Response 可以使用不同的具体类型。

- `RequestEncode(name, session, request)`：产生已打包的请求；协议定义带 Response 时登记等待。
- `ResponseEncode(name, session, response)`：生成关联原 session 的已打包回复。
- `Dispatch(packed)`：解包并返回 mode、协议名称、session 和具体消息。
- `GetProtocolByName`：按 Sproto 协议名称查找定义。
- `CancelSession(session)`：在超时、取消或发送失败后撤销等待。
- `ResetSessions()`：连接关闭时清理全部等待。

带 Response 的请求使用正 int32 session，每个 Rpc 最多保留 1024 个等待。session 由调用方分配，避免重复或回绕。没有 Response 的 Notify/Push 不登记等待，使用零 session。Rpc 不发送网络数据，也不提供超时或重试；这些由连接 owner 负责。

Rpc 使用固定的 type/session 包头，不提供自定义 package 或 ud 字段；Protocol 尚不表示官方 `response nil` 的无 body 确认响应。

每连接持有独立 Rpc，协议定义构造后保持只读。连接读写、分帧、Handler 调用、超时和重试由外层负责。GSR 使用自己的 Dispatcher，直接借用本库的编解码能力。

不联网的完整 Echo 示例见 [examples/echo](examples/echo)：

```sh
go run ./examples/echo
```

## 资源预算与错误

`Limits` 零值使用有限默认值：

| 项目 | 默认值 |
| --- | --- |
| MaxBytes | 8 MiB |
| MaxDepth | 64 |
| MaxElements | 100000 |

深度最多配置到 128。元素预算包含字段与集合元素，检查在递归或集合分配前进行；编码预算在生成报文前检查。`limits.Validate()` 可提前检查显式配置。

使用 `errors.Is` 判断 `ErrLimit`、`ErrInvalidLimits`、`ErrEncode`、`ErrDecode`、`ErrUnpack` 等错误。无效 schema 返回带字段信息的错误；不要用错误字符串实现业务分支。RPC 的未知协议、未知会话或重复协议定义分别使用 `ErrUnknownProtocol`、`ErrUnknownSession` 和 `ErrRepeatedRpc`。

## 测试

默认测试包括官方线格式与集合向量、基础类型与数组、嵌套结构、nil/零值、负整数与空数组、double 特殊位模式、字段增删兼容、坏包、精确预算边界，以及 RPC 非法输入、重复会话、连接隔离与等待清理：

```sh
go test ./... -timeout=2m
go vet ./...
go test -race ./... -timeout=2m
go test -coverprofile=coverage.out
go tool cover -func=coverage.out
go test -run '^$' -fuzz '^FuzzDecode$' -fuzztime=30s -parallel=2
go test -run '^$' -fuzz '^FuzzPackRoundTrip$' -fuzztime=30s -parallel=2
go test -run '^$' -fuzz '^FuzzCodecRoundTrip$' -fuzztime=30s -parallel=2
```

实际运行官方 C/Lua 对照时，将 `SPROTO_ORACLE_DIR` 指向已编译的官方仓库（含 `sproto/core.so`）：

```sh
SPROTO_ORACLE_DIR=/path/to/sproto go test -run '^(TestOfficialOracle|TestOfficialCollections)$' -timeout=2m
```

[CI](.github/workflows/test.yml) 固定官方提交 `e93fd5a0653b2bcab38d3b830958bf2e3a901b5f`，构建 C/Lua 并执行官方对照、race、vet 和有限 fuzz。默认测试不要求安装 Lua，对照测试未配置目录时会明确跳过。

三个 fuzz 入口分别检查有界坏包（含递归结构）、任意字节的 Pack/Unpack 往返，以及包含集合、嵌套和任意 double 位模式的规范编解码往返。测试覆盖已修复的主要行为和代表性的官方跨语言组合；固定用例和有限 fuzz 不证明所有输入被穷尽，覆盖率不作为完整兼容性的证明。

### 官方对照的已知限制

固定官方 C 版本的 `todword` 在 signed int 中计算后返回 size_t；所核对的 64 位构建中，部分低 32 位最高位为 1 的 64 位数值解码会被错误符号扩展，例如 MaxInt64 数组元素解为 -1。

完整数值边界向量由官方从独立 Lua 输入编码，再与 Go 编码逐字节比较并验证 Go 解码；另用官方解码核对四字节负整数数组。官方源码保持原样，Go 不复制该错误解码。固定向量、有限 fuzz 和一次跨语言对照不等于所有输入已被验证。

## Benchmark

```sh
go test -run '^$' -bench . -benchmem
```

性能随 schema、载荷和机器变化，请在实际目标环境中测量。

## 许可证

[MIT](LICENSE)，保留 xjdrew/gosproto 的原始版权。本项目从其 v0.1.0 独立维护，版本变化见 [CHANGES.md](CHANGES.md)。
