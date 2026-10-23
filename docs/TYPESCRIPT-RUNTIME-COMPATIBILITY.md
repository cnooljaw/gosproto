# TS 运行时兼容性检查（2026-10-23）

结论：当前 TS 入口依赖 Node Buffer，不能直接作为通用浏览器、LayaAir 或 Cocos 游戏运行时代码分发。显式 Buffer 适配的浏览器试验通过了本轮探针；引擎 Web、小游戏和原生仍需各自验收。本轮只检查和记录，没有修改生产源码、公开 API、依赖或发布配置。

## 检查对象与环境

- 仓库提交：`ae956fe`；权威源码：`typescript/src/sproto.ts`。
- 源码 SHA256：`1b1258d8258e15d3b6df18e98da9667139a0788c277c2d6f21c549cbb05617c8`。
- 构建 JS SHA256：`68ee5e27bbc34d7ca163bd39bcd89f47ac945b31501316d56bda3e1f649db817`。
- Node `v23.11.0`、TypeScript `5.9.3`；浏览器 Edge/Chromium `150.0.0.0`（macOS，UA 报告版本）。
- 本机 LayaAirIDE `3.3.10`，其已有工具包含 esbuild `0.18.11` 与 `buffer` `5.7.1`。本轮复用这些工具进行独立打包，没有运行 Laya 工程构建，也未安装新依赖。
- 未提供目标引擎工程和版本；Cocos 文档核对采用 Creator `3.8 LTS`，不是实际工程版本证明。
- 用户目标覆盖 Web、小游戏和原生；小游戏平台、基础库版本及原生系统/JS 引擎版本尚未指定。

## 兼容矩阵

| 环境 | 当前实现的结论 | 证据与边界 |
| --- | --- | --- |
| Node | 本轮回归通过 | 原有四项测试全通过、零跳过 |
| 原生浏览器 ESM，无适配 | 模块加载失败 | 同源 HTTP 服务加载未修改的构建 JS 失败；ES2022 控制模块成功 |
| 浏览器，显式 alias + Buffer 实现 | 本轮探针通过 | 临时将 `node:buffer` 解析到 `buffer@5.7.1`，不代表正式支持或性能验收 |
| LayaAir Web | 当前入口存在 Node 依赖阻塞；具体工程未实测 | 本机 esbuild 的 browser 打包默认不能解析 `node:buffer`；不是完整 IDE 构建结果 |
| LayaAir 小游戏 / 原生 | 不能声明支持 | 需要显式字节适配、目标打包配置和真机验证 |
| Cocos Creator 3.8 默认模块解析 | 当前入口不兼容 | 官方明确不支持 Node 内置模块及 `node:` 协议 |
| Cocos Web / 小游戏 / 原生，适配后 | 未实测 | 需要在实际引擎项目和对应目标运行时验收 |

安装 IDE、编辑器进程可用 Node、TS 编译成功，均不能作为游戏运行时支持 Node 内置模块的证据。

## 已确认的阻塞

1. **静态 Node 模块导入。** `sproto.ts:4` 直接导入 `node:buffer`，构建 JS 原样保留；模块顶层立即 `Buffer.alloc`。仅注入全局 `Buffer` 不能满足这个静态依赖。esbuild 在 `platform: browser` 且无 alias 时返回 `Could not resolve "node:buffer"`；真实浏览器的动态导入返回 `TypeError: Failed to fetch dynamically imported module: node:buffer`。
2. **依赖 Buffer 语义。** 使用 `alloc`、`allocUnsafe`、`byteLength`、`write(..., "utf8")`、`toString("utf8", start, end)`、`writeUInt16LE` 和 `copy`。`copy` 还用于重叠区间压紧头部、FF 段复制及结果独占副本。标准 `Uint8Array` 没有这些同名方法。
3. **输入不能直接换成 Uint8Array。** 对官方 UTF-8 向量传入 Node Buffer，解码为 `中文🙂`；传入同字节的 `new Uint8Array(wire)`，字符串变为整个输入的逗号分隔十进制字节。原因是调用了 TypedArray 的 `toString`，参数不再表示 UTF-8 和字节范围。这是静默错误。全非零八字节 FF 段的 Pack/Unpack 则抛出 `TypeError: source.copy is not a function`。部分纯整数输入成功不能证明输入类型通用。
4. **声明文件也依赖 Node。** `dist/sproto.d.ts` 导入 `node:buffer`，公开参数和返回值使用 Buffer。将声明文件复制到无 Node 类型的独立 DOM 消费工程，以 `types: []`、`skipLibCheck: false` 检查，得到 `TS2307: Cannot find module 'node:buffer' or its corresponding type declarations`。增加 Node 类型只能解决类型解析，不能提供客户端运行时。
5. **没有跨平台分发入口。** 当前包只导出同一个 ESM 入口，未声明运行时 polyfill 依赖或 browser 条件入口。编译目标为 ES2022 / NodeNext，包含类字段和 `??=`；目标引擎需要核对其转译链与最低运行时。源码有 `@ts-nocheck`，因此项目配置中的 `strict` 不构成该文件的类型安全证明。

可独立复现 Uint8Array 输入问题：

```sh
npm --prefix typescript run build
node --input-type=module <<'JS'
import { Sproto } from './typescript/dist/index.js';
const codec = new Sproto('.Text { text 0 : string }');
const wire = codec.encode('Text', { text: '中文🙂' });
console.log(codec.decode('Text', wire));
console.log(codec.decode('Text', new Uint8Array(wire)));
try { codec.pack(new Uint8Array(8).fill(255)); }
catch (error) { console.log(error.message); }
JS
```

## 临时浏览器适配试验

生产源码保持原样；通过 esbuild 私有解析插件将 `node:buffer` 精确解析为本机 IDE 所带 `buffer/index.js`，输出 ESM 试验包。该 npm Buffer 实现是第三方依赖；本轮只借用已有安装，未加入本仓依赖。标准库提供 Uint8Array/DataView，但没有现有代码所需的完整 Buffer 方法和 UTF-8 接口，因此不能靠改名替代。

在 Edge 中执行了以下探针：

- 读取仓库固定的 20 组官方正确性向量；每组检查 Encode、Decode、Pack、PEncode、PDecode、Unpack→Decode 共六条路径，包含布尔及正负安全整数数组。
- 独立官方 UTF-8 字节向量：中文及 emoji 的编码字节和解码值一致。
- 全部 256 个 mask × 四种有效字节值，共 1024 组精确 Pack 字节与 Unpack 检查。
- FF 段长度 2040 / 2048 / 2056 字节的恢复检查；这部分是自身往返，不标成额外官方向量。
- Encode、PEncode、Unpack 结果跨后续调用保持不变；大小消息交替到 64 KiB、坏 FF 后恢复、安全整数超界拒绝、解包超 8 MiB 拒绝。
- 原生 type/session RPC 的请求、响应与 session=7 关联一次往返。

这些探针全部通过。它们不是原有 Node 测试套件的全量浏览器移植，不包含完整坏包空间、全部 UTF-8 非法序列、所有 cap 边界、16 KiB 缓存保留上限检查或性能采样，也没有运行小游戏/原生包。

本轮临时页面、打包产物及探针放在 ignored `tmp/ts-runtime-audit/`；未纳入生产分发。适配试验说明可以继续评估显式 Buffer 方案，不能将当前包标成已支持各引擎。

## 后续方案边界

| 方案 | 工作与代价 |
| --- | --- |
| 显式 Buffer 适配入口 | 保留现有 Buffer 调用方式；需要声明实际依赖、确定每种构建器的解析规则、处理类型声明及网络 ArrayBuffer/Uint8Array 输入，并验证各目标的包体和性能 |
| Uint8Array/DataView 通用字节内核 | 减少对 Node API 的依赖；需要实现 UTF-8、重叠复制和范围行为，重新验收输出所有权、预算与 cap，并明确如何保持现有 Node Buffer API |

仅增加全局 Buffer、仅装 `@types/node`、将导入字符串简单改成 `buffer`，都不能完成跨平台适配。Cocos 3.8 官方说明其不处理 package.json 的传统 `browser` 字段；若采用条件入口，需要按其支持的 exports 条件设计。

若改变公开输入/输出类型、资源预算、错误语义或 buffer cap，先提交方案由用户裁决。实现后在实际 Laya/Cocos 版本中分别验收 Web、指定小游戏基础库及 iOS/Android 等原生目标；复用官方 raw/packed 向量，覆盖 UTF-8、布尔、安全整数、数组、session、坏包、输入偏移、独占结果和缓存保留限制。性能应分别测量六条编解码路径，不能沿用 Node 数字。

## 外部依据

- [Cocos Creator 3.8 Module Specification](https://docs.cocos.com/creator/3.8/manual/en/scripting/modules/spec.html)：Node 内置模块不支持、条件 exports 与传统 browser 字段的区别。
- [feross/buffer](https://github.com/feross/buffer)：面向浏览器的 Buffer 实现及显式引入方式；本轮实际试验使用本机已有的 5.7.1。
