# 独立协议仓库迁移（2026-10-23）

本目录以现有gosproto `3c505c3` 的完整提交历史检出，模块路径与当前分支不变。原 `/private/tmp/gsr-gosproto-fork` 及其中未提交删除保留原样；这些删除未导入新工作目录，不覆盖、不恢复旧工作树。

从GSR `d2ef3c5` 迁入TS权威源码、通用Benchmark及优化文档；逐文件原始来源与SHA256见 [清单](MIGRATION-SOURCES.json)。复制后对活跃工具和说明进行路径/模块适配，冻结夹具不带入GSR Registry接线生成物，历史 `docs/benchmarks` 证据原样保留，仅新增索引。Go库、TS codec算法未因迁移改写。

GSR保留 `protocol/sproto`、Agent、客户端会话/安全检查和真实WebSocket集成测试。其TS vendor副本来自本仓，带来源与哈希校验。Go正式依赖仍v0.4.0；新优化版发布及GSR升级另行执行，不用永久绝对replace替代发布。

Go根模块与Benchmark嵌套模块分别验证；TS工程独立构建。TS当前只声明Node支持，浏览器/Laya/Cocos见 [TODO](TODO.md)。

本轮保留现有Go历史；GSR提取文件的历史通过原仓库提交和来源清单追溯，不重写GSR历史，不建立嵌套Git仓库或submodule。

[旧GSR接入研究](SPROTO-INTEGRATION-HISTORY.md) 与 [旧GSR验证日志](GSR-VALIDATION-HISTORY.md) 保留当时原文，包含GSR专属相对路径；它们仅供历史检索，当前入口为README和本迁移说明。

迁移验收结果见 [MIGRATION-VALIDATION.md](MIGRATION-VALIDATION.md)。

## 在其他 Codex 项目中继续

直接将本仓根目录作为项目目录，从AGENTS.md进入；当前分支为 `codex/official-compat`，沿用当前分支工作，不以旧临时目录作为维护入口。

- 当前代码与最新已发布版本不同：GSR正式Go依赖仍为v0.4.0，迁移和后续优化尚未推送或发布；发布及使用方升级需要另行安排。
- 当前状态以GOSPROTO-OPTIMIZATION-STATUS.md和MIGRATION-VALIDATION.md为准；历史报告的“下一步”“冻结”“未提交”等字样是当时记录。
- Go根模块、typescript与benchmarks分别验证；不要把根目录Go测试通过等同于TS和跨语言基准通过。
- 浏览器/Laya/Cocos兼容性在TODO.md，仍未验证；npm发布暂不是前置要求。
- 原临时Go工作树及其删除状态保留供追溯；活跃维护统一在本仓。迁移资料已按用户明确指示整理为本地提交，推送另行授权。
- 后续更新TS时，GSR由自己的同步工具消费本仓版本，不直接在GSR的vendor副本优化。
