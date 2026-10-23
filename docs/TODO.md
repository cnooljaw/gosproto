# 待办

## TS 浏览器、Laya 与 Cocos 兼容性（尚未验证）

当前实现直接依赖 `node:buffer`。本轮仅迁移并验收 Node 环境；文件复制、TS编译成功和Node Benchmark均不证明浏览器或游戏引擎兼容。

后续工作：

- 确认目标浏览器及 Laya/Cocos 的实际版本、构建器和支持的运行平台。
- 评估显式Buffer适配或Uint8Array/DataView实现；不预设简单polyfill就满足API、所有权或性能契约。
- 在真实目标运行时测试模块导入、UTF-8、布尔、安全整数边界、数值/结构数组、session及坏包。
- 复用官方raw/packed向量，验证缓冲独占、失败恢复和buffer cap；如引入适配，单独测Encode/Decode/Pack/Unpack及组合路径。
- 验收后再更新支持矩阵；Node结果单独保留，不混为跨平台性能结论。

## 分发与发布

- Go优化版正式发布及使用方依赖升级：另行授权，不随目录迁移自动发布。
- npm包发布：目前private工程只用于构建、测试和单文件分发，真实调用者需要时再发布。
- 大型历史原始profile外置归档：本轮保全历史资料，后续单独设计附件和校验索引，不静默删证据。
