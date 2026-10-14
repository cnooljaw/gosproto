# gosproto：官方 wire 兼容修复分支

模块：`github.com/cnooljaw/gosproto`。基于 xjdrew/gosproto v0.1.0 的独立维护分支，修复记录见 [CHANGES.md](CHANGES.md)。兼容依据是官方 cloudwu/sproto，不采用改变官方类型解释的扩展。保留原始版权与 MIT [许可证](LICENSE)。

```go
import sproto "github.com/cnooljaw/gosproto"

data, err := sproto.Encode(message)
used, err := sproto.Decode(data, target)
```

可显式使用 `EncodeWithLimits`、`DecodeWithLimits` 和 `UnpackWithLimit`。零值 Limits 默认 MaxBytes=8 MiB、MaxDepth=64、MaxElements=100000；深度最多配置到 128。预算在递归、集合分配和解包扩容前检查。Decode 失败不修改调用方原值，返回已消费字节数供原生 package + body 分帧使用。

支持 string/binary、boolean、signed 64-bit integer、double、相应数组与嵌套结构。字段使用现有 `sproto` struct tag：double 为 `sproto:"double,0"`，double 数组为 `sproto:"double,0,array"`。unsigned Go 值仅在有符号 64 位范围内可编码；decode 到窄类型溢出时返回错误。map、定点数的自动缩放和 schema 文本解析不属于本修复版本的新增支持，不宣称覆盖所有官方特性。

RPC session 为正 int32，每个 Rpc 最多保留 1024 个等待响应；owner 在超时/发送失败后调用 CancelSession，断线调用 ResetSessions。不要跨连接共享可变会话表。原有流式 Service helper 保留，GSR 不使用它启动额外调度或 goroutine。

## 验证

默认 `go test ./...` 使用固定官方向量与回归测试。设置 `SPROTO_ORACLE_DIR` 为已编译官方 C/Lua 模块目录，可启用官方负整数数组解码与完整数值编码对照：

```sh
SPROTO_ORACLE_DIR=/path/to/official/sproto go test -race ./... -timeout=2m
go vet ./...
go test -run '^$' -fuzz '^FuzzDecode$' -fuzztime=30s -parallel=2
```

官方固定版本的 `todword` 用 signed int 中间表达式返回 size_t；在本机 64 位 C/Lua 构建中，低 32 位最高位为 1 的部分 64 位数值会被错误符号扩展（例如 integer 数组中的 MaxInt64 解为 -1）。Go 保留规范 64 位位模式，不复制该实现缺陷；完整边界向量由官方从独立 Lua 数值编码，再比较 Go 编码并验证 Go 解码。官方源码未被修改。

CI 固定官方提交 e93fd5a0653b2bcab38d3b830958bf2e3a901b5f 并构建 C/Lua oracle；生产编解码为纯 Go。有限 fuzz 和向量测试不等于所有输入都已穷尽。

## 上游文档（历史能力说明）

# gosproto

[sproto](https://github.com/cloudwu/sproto)'s encoder and decoder in golang.

## type map

sproto type      | golang type
---------------- | -------------------------------------------------
string           | \*string, []byte, string
integer          | \*int8, \*uint8, \*int16, \*uint16, \*int32, \*uint32, \*int64, \*uint64, \*int, \*uint, int8, uint8, int16, uint16, int32, uint32, int64, uint64, int, uint
boolean          | \*bool, bool
object           | \*struct
array of string  | []string
array of integer | []int8, []uint8, []int16, []uint16, []int32, []uint32, []int64, []uint64, []int, []uint
array of boolean | []bool
array of object  | []\*struct

## schema

You can define go struct corresponding to sproto schema directly as examples in all test cases.
Or use [sprotodump](https://github.com/lvzixun/sprotodump) to change sproto schema to go file.

## test

```
go test github.com/xjdrew/gosproto
```

## benchmark

```txt
BenchmarkEncode-8                 500000              2058 ns/op
BenchmarkDecode-8                 500000              3222 ns/op
BenchmarkEncodePacked-8           500000              2523 ns/op
BenchmarkDecodePacked-8           500000              3660 ns/op
PASS
ok      github.com/szyhf/go-sproto      5.864s
```

## 改进说明（improvement）

原来的sproto只支持指针类型的变量，要通过sproto.Int(ptr *int)类似的方法赋值访问，非常麻烦。

> 注：原来这是proto2的标准实现方法，但proto3已经不再使用指针的方案——20170228

这个库增强了Encode和Decode的能力，现在可以直接处理int string bool的值类型。

同时，使用相同位置及配置的值类型及对应指针，在tag一致的情况下，且指针均不为nil的情况下，编码结果是相等的，例如：

``` golang
type WithPtr struct{
	ID *int `sproto:"integer,0,name=ID"`
}

type WithVal struct{
	ID int `sproto:"integer,0,name=ID"`
}

```

如果WithPtr.ID=nil，则使用WithVal对编码进行Decode之后，WithVal.ID会等于0。

> 个人倾向尽可能不要使用nil作为值，作为一个用于跨平台的编码，nil容易在不同平台上产生不同的解析结果，极易产生歧义。

> 同样，个人不建议使用值类型的Struct，所以不支持了。

更多的实现效果请参考encode_test.go中的例子。
