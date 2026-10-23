import { execFileSync } from "node:child_process";
import { mkdirSync, readFileSync, writeFileSync, readdirSync } from "node:fs";
import { resolve } from "node:path";
import { fileURLToPath } from "node:url";
const root = fileURLToPath(new URL("../", import.meta.url));
const tool = process.env.SPROTODUMP_DIR;
if (!tool) throw new Error("set SPROTODUMP_DIR to the pinned lvzixun/sprotodump checkout");
for (const path of ["gen/sproto", "typescript/gen/benchmark", "typescript/gen/gsr"]) mkdirSync(resolve(root, path), { recursive: true });
const plugin = `--plugin=protoc-gen-es=${resolve(root, "node_modules/.bin/protoc-gen-es")}`;
const goOptions = ["--go_out=.", "--go_opt=module=github.com/cnooljaw/gosproto/benchmarks"];
execFileSync("protoc", ["-Ischema", ...goOptions, plugin, "--es_out=typescript/gen/benchmark", "--es_opt=target=ts,import_extension=js", "schema/representative.proto"], { cwd: root, stdio: "inherit" });
execFileSync("protoc", ["-Ischema/legacy", ...goOptions, plugin, "--es_out=typescript/gen", "--es_opt=target=ts,import_extension=js", "gsr/common/v1/common.proto", "gsr/protocol/v1/options.proto", "gsr/example/echo/v1/echo.proto"], { cwd: root, stdio: "inherit" });
for (const [source, output, name] of [["schema/representative.sproto", "gen/sproto/representative.go", "benchsp"], ["schema/rpc/c2s.sproto", "gen/legacy/c2s/c2s.go", "c2s"]]) {
  mkdirSync(resolve(root, output, ".."), { recursive: true });
  execFileSync(process.env.LUA ?? "lua", [resolve(tool, "sprotodump.lua"), "-go", resolve(root, source), "-o", resolve(root, output), "-p", name], { cwd: tool, stdio: "inherit" });
  let generated = readFileSync(resolve(root, output), "utf8")
    .replaceAll("github.com/xjdrew/gosproto", "github.com/cnooljaw/gosproto")
    .replace(/^\/\/ source:.*$/m, `// source: ${source}`)
    .replace(/^[\t ]*MethodName:[^\n]*\n/gm, "");
  if (name === "c2s") {
    generated = generated.replace(/(Name:\s+)"[^".]+\./g, '$1"');
    // 上游Request/Response同源行排序不稳定；仅固定声明顺序，不改变字段及Protocol表。
    const pattern = /^type (\w+) struct \{\n[\s\S]*?^\}\n/gm;
    const declarations = [...generated.matchAll(pattern)].sort((a, b) => a[1] < b[1] ? -1 : a[1] > b[1] ? 1 : 0);
    let index = 0;
    generated = generated.replace(pattern, () => declarations[index++][0]);
  }
  writeFileSync(resolve(root, output), generated);
  execFileSync("gofmt", ["-w", output], { cwd: root, stdio: "inherit" });
}

// 生成器输出统一单个末尾换行，保持重复生成可审查。
function normalize(directory) {
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const path = resolve(directory, entry.name);
    if (entry.isDirectory()) normalize(path);
    else if (entry.name.endsWith(".ts")) writeFileSync(path, readFileSync(path, "utf8").trimEnd() + "\n");
  }
}
normalize(resolve(root, "typescript/gen"));
