# 待办

## TS 浏览器、Laya 与 Cocos 兼容性（当前入口不兼容，引擎待验收）

当前实现直接依赖 `node:buffer`。2026-10-23 检查已复现浏览器模块加载失败、Uint8Array 输入的字符串误解码及 FF 段缺少 copy 的异常；显式 alias + Buffer 的临时浏览器探针通过，详见 [兼容性检查](TYPESCRIPT-RUNTIME-COMPATIBILITY.md)。正式支持仍只验收 Node；Web、小游戏与原生均是目标，实际 Laya/Cocos 工程尚未验收。文件复制、TS编译成功和Node Benchmark均不证明游戏引擎兼容。

后续工作：

- 确认目标浏览器及 Laya/Cocos 的实际版本、构建器和支持的运行平台。
- 评估显式Buffer适配或Uint8Array/DataView实现；不预设简单polyfill就满足API、所有权或性能契约。
- 在真实目标运行时测试模块导入、UTF-8、布尔、安全整数边界、数值/结构数组、session及坏包。
- 复用官方raw/packed向量，验证缓冲独占、失败恢复和buffer cap；如引入适配，单独测Encode/Decode/Pack/Unpack及组合路径。
- 验收后再更新支持矩阵；Node结果单独保留，不混为跨平台性能结论。

## 分发与发布

- 使用方（包括GSR）升级至Go库最新正式版本：另行执行；本仓发布不自动修改使用方依赖。
- npm包发布：目前private工程只用于构建、测试和单文件分发，真实调用者需要时再发布。
- 历史资料已在v1.0.1中外置为[完整档案](benchmarks/README.md)，原始字节、来源及校验值保全。后续新采样继续写入ignored tmp，按需归档，不重新加入源码树。
