# Benchmark 摘要与历史档案

[当前优化状态](../GOSPROTO-OPTIMIZATION-STATUS.md) · [最终结果摘要](SUMMARY.md) · [完整72项对照](FINAL-COMPARISON.md) · [活跃Benchmark工程](../../benchmarks/README.md)

v1.0.1 起，本目录仅保留摘要、最终对照和归档索引。原始采样、汇编输出、源码快照、历史实验脚本、迁移来源清单和GSR历史资料已移至Release附件，原始字节与校验值不变。

## 下载与校验

- [完整档案（约37.1 MiB）](https://github.com/cnooljaw/gosproto/releases/download/v1.0.1/gosproto-v1.0.0-documentation-archive.tar.gz)
- [SHA256SUMS](https://github.com/cnooljaw/gosproto/releases/download/v1.0.1/SHA256SUMS)
- [机器可读索引](ARCHIVE.json)
- [v1.0.0 原始文档树](https://github.com/cnooljaw/gosproto/tree/v1.0.0/docs)

归档 SHA256：

```text
e90360f88bb620aa9f1ef9e89692d03c4f368e9faf8803b6df4a9867bcaa4d96
```

```sh
gh release download v1.0.1 --repo cnooljaw/gosproto --pattern 'gosproto-v1.0.0-documentation-archive.tar.gz' --pattern SHA256SUMS --dir tmp/history
cd tmp/history
shasum -a 256 -c SHA256SUMS
tar -xzf gosproto-v1.0.0-documentation-archive.tar.gz
```

也可从仓库根目录执行 `python3 benchmarks/tools/verify-history.py /path/to/gosproto-v1.0.0-documentation-archive.tar.gz`，校验归档整体SHA256、文件集合及全部逐文件内容；校验器不解压或执行旧脚本，需要Python >= 3.9。

使用全新的临时目录解压，不覆盖当前源码树。包内 `ARCHIVE-MANIFEST.json` 列出每个文件的路径、大小、SHA256及是否曾受Git跟踪；档案包含v1.0.0的全部8,768个文档文件，以及原迁移清单所列的6个ignored Python缓存，共8,774个文件（不含生成的归档清单自身）。原迁移清单的8,754项历史文件均已逐字节核对。缓存只为保全原迁移快照而归档，不进入当前源码分发。

档案绑定gosproto v1.0.0提交 `adcdd9451570c831a3b578f0dc5983d15763ffe3`。GSR来源与原始文件SHA256在包内 `docs/MIGRATION-SOURCES.json`；历史报告包含不同仓库提交，不能把所有提交都当成gosproto提交。

## 使用边界

日期目录、旧文档中的“下一步”“冻结”“未提交”等均是当时状态，当前入口为优化状态和TODO。旧脚本可能引用历史GSR目录、临时副本及二进制，不承诺解压后可以直接重跑；新实验使用活跃Benchmark工程，输出进入ignored `tmp/`。

当前设计资料中指向历史原文的链接固定到v1.0.0，便于在线查看；完整原始资料也可离线从附件解压。旧控制失败、反向、回退、未采用候选与原始hash完整保留，不因归档改写。

发布前已逐文件校验本地归档，并核对GitHub服务端计算的附件SHA256和字节数。完整回下载因本机网络超时未完成，机器可读索引中单独记录为false。

v1.0.0标签与Git历史保持原样；本次清理缩小后续版本快照和Go模块分发，不会消除完整Git历史中已存在的大文件。
