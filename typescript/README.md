# Sproto TypeScript

与Go库同仓维护的Sproto实现，源自 sealindx/sproto-ts 提交 `bbee46fd76e2c9a7401ff9efd1dcadb538908130`，保留 [BSD-2-Clause许可证](LICENSE)。当前只验收 Node.js；直接依赖 node:buffer，浏览器、Laya、Cocos [待验证](../docs/TODO.md)。

[运行时兼容性检查](../docs/TYPESCRIPT-RUNTIME-COMPATIBILITY.md) 已复现当前入口在浏览器加载失败，以及直接传入 Uint8Array 的错误行为。显式 Buffer 适配的浏览器探针通过；小游戏、原生及实际引擎工程仍需分别验收。

## 构建与测试

需要 Node >=22.15.0：

```sh
npm ci
npm test
```

`src/sproto.ts` 为唯一维护源码，`dist/sproto.js` 和声明文件由构建生成，不手工维护JS副本。本工程目前private，不要求npm发布。

```js
import { Sproto } from "./dist/index.js";
const codec = new Sproto(".Message { text 0 : string }");
const wire = codec.pencode("Message", { text: "hello" });
console.log(codec.pdecode("Message", wire));
```

## 使用方复制

可以复制 `src/sproto.ts`（TS项目）或构建的 `dist/sproto.js`（JS项目），同时带上LICENSE并记录来源仓库、提交和SHA256。后续修改回本仓验证后再同步，不在多个vendor副本各自维护。

GSR提供 `sync-sproto.mjs` 同步及离线校验。源码尚未提交时来源记录会明确标记工作树快照，不能把HEAD误当成该文件已经存在的提交。

JS Number仅接受安全整数；超范围拒绝，不提供完整int64的BigInt表示。保持现有API、独占输出、编码/解包预算及16KiB解包缓存保留上限。当前测试不构成所有Sproto类型和所有输入的穷尽证明。
