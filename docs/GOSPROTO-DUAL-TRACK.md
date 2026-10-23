# Go Sproto Body 与 Packing 双主线

历史双主线计划已完成既定切片；[当前优化状态](GOSPROTO-OPTIMIZATION-STATUS.md)和[最终结果](benchmarks/SUMMARY.md)是现状入口。原始目标、当时Go/TS比例表和逐轮进展完整保存在[档案](benchmarks/README.md)及[v1.0.0原文](https://github.com/cnooljaw/gosproto/blob/v1.0.0/docs/GOSPROTO-DUAL-TRACK.md)，不把旧“下一步”当作当前任务。

## 长期目标

- Body：分别看Encode、Decode中的类型访问、布局、对象构造与字段赋值。已有局部领先不等于没有可优化空间。
- Packing：分别看Pack、Unpack和完整packed消息的成本；正文优势不能掩盖解包或组合路径的缺口。
- 按当前业务热点和工程价值交替选择单一候选，不机械按轮数推进；某条主线未解决不构成冻结另一条主线的理由。

## 评价方式

分别记录纯正文、单独Pack/Unpack、分阶段组合、融合EncodePacked/DecodePacked及原生RPC入口。保留正常消息的绝对耗时、相对同语言同场PB比例、分配与局部代价，不能将手工封装或微基准冒充真实入口。

Go/TS运行时、整数与对象模型、UTF-8行为和资源校验不同。TS历史优势可以作为探索参照，不是Go必达比例，也不是跨语言绝对排名。当前剩余局部缺口见最终结果，未经复测不沿用旧比例做新结论。

控制失败、同源码clone偏差、累计/中位方向分歧和小包退化完整保留；不从候选值减控制值，不设固定收益比例淘汰线。较大改写要同时评估收益、可信度、维护和所有权成本。

## 不变的契约

官方wire、公开API、presence、错误顺序、失败原子性、预算、输出cap、独占结果及GC写屏障不因性能目标改变。生成器、对象模型或跨调用内存复用属于独立设计；TS浏览器/Laya/Cocos支持另见[待办](TODO.md)。新实验使用[活跃Benchmark](../benchmarks/README.md)，原始输出进入ignored `tmp/`。
