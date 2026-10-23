# AGENTS.md

本文件是 gosproto 的工作入口。Go 库、TS 实现、独立 Benchmark 与协议文档在同一仓库维护；GSR 是使用方。

## 项目边界

- Go 模块保持 `github.com/cnooljaw/gosproto`，根目录为 Go 库；`typescript/` 为 TS 权威源码，`benchmarks/` 为独立 Go/npm 工程。
- 只处理 Sproto 编解码、Pack/Unpack、原生 type/session 关联；不引入 GSR、Gate、Service、连接管理或业务包装层。
- wire 以 cloudwu/sproto 固定官方实现为准。修改兼容性时先核对官方代码与独立向量，不以自身往返作为唯一证据。
- 保持 API、资源预算、失败原子性、结果独占与已有 buffer cap；改变这些契约先提出方案，由用户裁决。
- 不机械统一 Go/TS 内部实现，不引入 Protobuf 风格 Envelope。
- `docs/benchmarks/` 仅保留摘要、最终对照和归档索引；完整历史证据保存在Release附件。旧报告的下一步、冻结或未提交状态是当时记录，以当前状态文档为准；不将原始采样重新加入源码树。

## 文档入口

- [使用说明](README.md)、[Go 编码规范](docs/GO-CODE-STYLE.md)、[开发工作流](docs/DEVELOPMENT-WORKFLOW.md)。
- [TS 使用说明](typescript/README.md)、[TS 编码规则](docs/TYPESCRIPT-CODE-STYLE.md)、[待办](docs/TODO.md)。
- [Benchmark](benchmarks/README.md)、[优化状态](docs/GOSPROTO-OPTIMIZATION-STATUS.md)、[迁移说明](docs/REPOSITORY-MIGRATION.md)。
- [TS → Go 优化经验](docs/SPROTO-OPTIMIZATION-LESSONS.md)、[双主线目标](docs/GOSPROTO-DUAL-TRACK.md)。

## 实现与验证

- 修改先搜索已有回归，优先扩展相关层级；每个 Go Test 函数有说明可观察行为的中文注释。
- 用最小相关测试迭代，形成完整切片后跑受影响工程；不为纯文档变更重复全库测试。
- 性能改动必须保留官方 wire、边界检查、预算、错误顺序、GC 写屏障和所有权。unsafe/汇编限定在私有、类型证明和范围证明明确的入口，保留 purego/其他架构回退。
- profile 与正式计时分开；基准不并发争抢CPU。记录实际源码、工具链、输入和二进制指纹，保留波动、控制失败与局部回退。
- 小收益也可保留，不设统一的最低百分比淘汰线；区分已确认收益和噪声，不从候选值中扣控制值。
- 默认测试预算2分钟，package验收5分钟；超时先定位，不盲目重试或只延长timeout。保留测试缓存。
- 新增第三方依赖前说明用途和标准库不能满足之处。生产库不得依赖 Benchmark 或 GSR。
- Go 测试、TS 测试、Benchmark 是不同工程；根目录 `go test ./...` 不等于嵌套基准模块已验证。
- 当前 TS 依赖 `node:buffer`，浏览器/Laya/Cocos 兼容性列为待办，未经实测不能声明支持。

## Git

- 默认当前分支，不主动创建、切换、合并分支；不修改或丢弃用户无关改动。
- 默认修改、验证、汇报并保留工作树供审查。只有明确“提交/commit”才暂存和提交；推送、发布另行授权。
- 提交前检查 status、diff --check、cached diff及其空白检查；提交后汇报hash与工作树状态。
- 提交信息尽量使用中文。历史证据不改写原始采样与校验值；重型可再生输出进入ignored tmp。
