# 迁移验收（2026-10-23）

[结构化结果](MIGRATION-VALIDATION.json) · [来源清单](MIGRATION-SOURCES.json) · [迁移说明](REPOSITORY-MIGRATION.md)

## 源码与证据

- 8,748个原已跟踪的历史Benchmark证据文件逐字节保全；原采样、原始hash、脚本和历史绝对路径不改写。
- 另有6个本机Python缓存保留在ignored目录，不进入提交；原已跟踪的6份日志作为证据一并迁移。
- 43个Go核心文件（含测试、汇编与go.mod）等于gosproto `3c505c3`；TS源码及固定输入与GSR `d2ef3c5` 一致。
- Go模块路径不变，生产库和独立Benchmark均不导入GSR。冻结protobuf schema保留旧全名/字段编号，GSR Registry接线生成物排除。
- GSR vendor源码与许可证等于权威源码。离线校验通过，篡改源码和缺失许可证记录均被拒绝。

## 验证结果

- 新目录Go完整测试、purego、race/checkptr、vet通过。
- 固定官方C/Lua重新构建通过，5组代表消息及20组正确性签名一致。机器上Lua为x86_64，oracle按Lua架构构建；不将该C/Lua运行视为ARM64性能测试。
- 独立TS四项回归通过；独立Benchmark检查5类消息/40操作、4类Echo/32操作、134个原生API和Pack形态通过。
- 固定sprotodump/protoc完整生成可重复；包装脚本规范上游同源行Request/Response的声明顺序，不改字段与Protocol表。
- 配对工具8项单测通过，脱离GSR运行单项self控制通过。控制值只验证测量流程，不作为优化收益或排名。
- game_large的Go/TS六步骤CPU与分配profile驱动通过，独立采集并生成两端摘要。
- 两套完整Benchmark驱动分别完成Go/TS 32项和40项、每项五轮及报告生成。这里只验收迁移后的完整工作流，不据此更新优化胜负。
- GSR Plain/MAC真实Go↔TS链路四项全部通过、零跳过；受版本管理及新增文件的干净工作树快照全仓Go测试通过。

## 环境限制与待办

GSR本机直接执行 `go test ./...` 会扫描ignored tmp下既有残缺实验，出现重复定义、缺少旧模块等失败；没有修改或删除这些资料。使用排除ignored文件的当前工作树快照验证独立检出的真实结果，不把原命令失败写成通过。

CI配置已加入独立TS和基准工程检查，尚未推送或触发GitHub CI。TS浏览器/Laya/Cocos兼容性仍是 [待办](TODO.md)；没有新增此类支持声明。Go正式发布和GSR依赖升级未随迁移执行，正式版本仍v0.4.0。

本轮新采样和profile保留在benchmarks/tmp（ignored），历史归档不增加新的性能结论。迁移验收后用户已明确要求本地提交；本次据此整理迁移变更，未推送或发布。
