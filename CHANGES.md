# v0.2.0

本版本是以 xjdrew/gosproto v0.1.0（ec7216542bfa5c334cc7021fb0b21d10b7463bca）为基线的独立修复版本。模块路径为 github.com/cnooljaw/gosproto，保留上游历史与 MIT 许可证。

- 按官方 C 实现扩展四字节有符号整数，修复数组负值失真和 unsigned 编码变量遮蔽。
- 接受官方规范空数组及官方允许的空数组变体；非空数组先检查元素宽度再运算。
- 增加原生 double 标量、指针和数组，使用 IEEE-754 位模式。
- Decode 失败时保留调用方原值，窄整数溢出与不匹配物理类型返回错误，未知字段静默跳过。
- 提供有限字节/深度/元素预算和有界 Unpack；RPC 等待上限为 1024，提供 CancelSession/ResetSessions。
- 增加官方向量、C/Lua 编码与解码对照、回归、race 与 fuzz 验证，清除上游一处不可达 return。

官方基准固定为 cloudwu/sproto e93fd5a0653b2bcab38d3b830958bf2e3a901b5f。修复不改变官方 wire，也不引入 GSR、Protobuf 或运行时 C/Lua 依赖。Go >= 1.24。

已记录固定官方 C 版本的 todword 符号扩展缺陷；64 位边界依据独立官方编码向量验证，不模拟其错误解码。
