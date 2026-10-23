import { spawnSync } from "node:child_process";
import { rmSync } from "node:fs";
import { localGoEnvironment, benchmarkRoot } from "./local-go.mjs";
const { temporary, ...env } = localGoEnvironment();
try {
  const result = spawnSync("go", process.argv.slice(2), { cwd: benchmarkRoot, env, stdio: "inherit" });
  if (result.error) throw result.error;
  process.exitCode = result.status ?? 1;
} finally { rmSync(temporary, { recursive: true, force: true }); }
