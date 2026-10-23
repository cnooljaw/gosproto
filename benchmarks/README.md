# Protobuf / Sproto Go 与 TS Benchmark

独立于GSR运行，比较同一数据下的正文、packing及冻结应用帧。Go库和TS源码取本仓；Protobuf Go v1.36.1、TS @bufbuild/protobuf 2.15.0。所有输入、计时边界和wire签名继承迁移前基线。

## 安装与校验

```sh
npm --prefix ../typescript ci
npm ci
npm run check
```

`check` 构建TS、执行Go夹具测试、检查TS正文/帧/原生RPC组合与Pack形态，不进行正式性能排名。Go比较代码在独立模块中；根目录Go库测试不会自动运行这里。

## 正式采样与profile

```sh
npm run bench:protocol
npm run bench:representative
npm run bench:api
npm run profile:protocol
npm run analyze:pack
npm run go -- test ./go -run '^$' -bench 'Benchmark(Pack|Unpack)Shapes' -benchmem
```

默认输出在本目录ignored `tmp/`。参数输出路径使用绝对路径；每次采样记录源码/工具链/夹具和依赖覆盖。CPU profile与分配profile分别采集，不与正式计时并发。可用 `SPROTO_PROFILE_SCENARIO=game_large` 聚焦profile。

Go入口通过独立临时modfile引用本仓Go源码，结束清理覆盖文件；不修改go.mod、不依赖GSR或全局go.work。可设置 `SPROTO_GO_DIR` 显式测另一份库。直接 `go test` 则使用go.mod中已发布v0.4.0；不要混淆两者。正式GSR依赖保持发布版本。

## 口径与夹具

- representative：AddressBook、两档游戏状态、896整数数组、4KiB ASCII，共40项；Encode/Decode、Pack/Unpack、分阶段组合分别报告。
- protocol：Echo短字符串、UTF-8、4KiB ASCII与零字节，共32项，区分payload和frame。
- api：原生pencode/pdecode、attach/dispatch与手工组合单独标记；不替换历史分阶段排名。
- shape：普通word密度、FF、零段、短尾和长度边界。
- Go/TS的编码输出分配、全新解码对象计入时间；schema解析、固定wire/语义检查不计时。

`schema/legacy`、`gen/legacy`、TS `gen/gsr` 是迁移时冻结的Benchmark夹具；保留旧protobuf全名和字段编号，不导入GSR，也不自动跟随GSR协议变化。原始数据和wire SHA256保持；历史客户端帧比对在GSR集成中保留，本工程使用冻结RPC schema的原生attach作独立比对。

## 官方与生成工具

```sh
SPROTO_ORACLE_DIR=/path/to/pinned-sproto npm run check:oracle
SPROTODUMP_DIR=/path/to/pinned-sprotodump npm run generate
```

官方源码版本与哈希见 `oracle-source.json`，生成需要protoc、protoc-gen-go、固定sprotodump及Lua。生成物提交以便普通使用方不安装生成工具。包装脚本固定上游同源行Request/Response的声明顺序并规范末尾换行，字段与Protocol表保持不变。

[Go配对工具](tools/go-pairs/README.md) · [当前优化状态](../docs/GOSPROTO-OPTIMIZATION-STATUS.md) · [历史报告索引](../docs/benchmarks/README.md)

历史实验的原始数据和脚本保存在Release附件；本工程、固定夹具及生成源码继续维护。下载档案后可执行 `python3 tools/verify-history.py /path/to/gosproto-v1.0.0-documentation-archive.tar.gz` 做整体和逐文件校验（Python >=3.9），无需解压。
