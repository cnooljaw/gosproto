import { execFileSync } from "node:child_process";
import { mkdtempSync, readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { tmpdir } from "node:os";
import { fileURLToPath } from "node:url";

export const benchmarkRoot = fileURLToPath(new URL("../", import.meta.url));
// 每次独立临时modfile；不修改任何正式go.mod，也不依赖GSR或全局go.work。
export function localGoEnvironment() {
  const temporary = mkdtempSync(resolve(tmpdir(), "sproto-benchmark-"));
  const modfile = resolve(temporary, "benchmark.mod");
  writeFileSync(modfile, readFileSync(resolve(benchmarkRoot, "go.mod")));
  writeFileSync(resolve(temporary, "benchmark.sum"), readFileSync(resolve(benchmarkRoot, "go.sum")));
  const library = resolve(process.env.SPROTO_GO_DIR ?? resolve(benchmarkRoot, ".."));
  const env = { ...process.env, GOWORK: "off", GOFLAGS: "" };
  execFileSync("go", ["mod", "edit", `-modfile=${modfile}`, `-replace=github.com/cnooljaw/gosproto=${library}`], { cwd: benchmarkRoot, env });
  return { ...env, GOFLAGS: `${process.env.GOFLAGS ?? ""} -modfile=${modfile}`.trim(), temporary };
}
