# Go同函数/同源码配对诊断

在独立归档副本中运行A/A或A/B，保留ABBA/BAAB、GC观测和原始轮次；不修改正式依赖或库工作树。控制阈值用于测量校准，不是优化收益淘汰线。

```sh
python3 tools/go-pairs/test_run.py
python3 tools/go-pairs/run.py --library .. --baseline HEAD --output /absolute/new-output --mode self --case game_large/sproto_encode
```

从benchmarks目录运行；output必须不存在。`clone`使用同源码独立符号；`compare`需要显式`--candidate-patch`。参数和统计保持既有实现，环境变量前缀为SPROTO_PAIR。候选小收益完整保留，compare不自动裁决采用。

旧实验中GSR_PAIR及GSR路径属于历史证据，不重写归档脚本。
