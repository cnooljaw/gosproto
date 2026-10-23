# 仓库来源与维护入口

Go模块保持 `github.com/cnooljaw/gosproto`，根目录维护Go库，`typescript/`为TS权威源码，`benchmarks/`为独立Go/npm工程。当前分支沿用 `codex/official-compat`，从AGENTS.md进入维护，不以旧临时工作树作为入口。

## 来源

本仓保留gosproto完整历史，以 `3c505c3` 为目录迁移时的Go基点；从GSR `d2ef3c5d80e0824fe51da575927f3e1b5a4d2c88` 迁入TS、通用Benchmark和性能资料。TS保留sealindx/sproto-ts来源及BSD-2-Clause许可证；Go保留上游MIT版权。冻结protobuf夹具保留旧全名及字段编号，不带GSR Registry接线生成物。

原迁移清单、每文件SHA256、历史接入和验证日志现保存在[完整档案](benchmarks/README.md)。档案绑定v1.0.0提交，包含迁移来源清单全部8,754项历史文件；其中6项是ignored Python缓存。迁移验收原始结果仍见[MIGRATION-VALIDATION.md](MIGRATION-VALIDATION.md)，它记录迁移时点，不代表当前发布状态。

## v1.0.1 归档整理

历史原始数据、源码快照、实验脚本、汇编输出及GSR专属旧资料移出当前源码快照。仓库保留最终结果、对照表、设计原则、来源索引及活跃Benchmark。原始文件通过包内清单逐一校验，GitHub服务端计算的附件SHA256与本地一致；完整回下载因本机网络超时未完成，不将其写成通过。v1.0.0标签与Git历史保持原样。

新实验输出进入ignored `tmp/`。查旧资料时先校验附件并解压到临时目录；旧脚本保留历史路径，不承诺可以直接重跑，也不把旧控制失败、回退或未提交字样改写成当前结果。

## 使用方边界

GSR保留协议适配、连接、会话安全和真实WebSocket集成；本库不引入这些职责。使用方依赖升级独立执行，不用永久绝对replace替代正式版本。更新TS后由使用方同步工具记录源码、许可证及来源指纹，不在多个vendor副本分别优化。

Go根模块、TS和Benchmark分别验证；根目录Go测试不等于嵌套工程已验证。TS正式支持仅验收Node；当前入口依赖node:buffer，浏览器/Laya/Cocos的Web、小游戏及原生见[兼容性报告](TYPESCRIPT-RUNTIME-COMPATIBILITY.md)。npm包仍为private。
