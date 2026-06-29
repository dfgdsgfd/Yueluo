import { mkdirSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";

const root = process.cwd();
const payload = {
  version_tag: process.env.GITHUB_REF_NAME || process.env.VERSION_TAG || "unknown",
  commit_hash: process.env.GITHUB_SHA || process.env.COMMIT_HASH || "unknown",
  build_time: process.env.BUILD_TIME || new Date().toISOString(),
  github_run_id: process.env.GITHUB_RUN_ID || "unknown",
};

for (const relativePath of ["public/version.json", ".next/version.json"]) {
  const target = join(root, relativePath);
  mkdirSync(dirname(target), { recursive: true });
  writeFileSync(target, `${JSON.stringify(payload, null, 2)}\n`, "utf8");
}
