import { spawnSync } from "node:child_process";
import dns from "node:dns/promises";
import fs from "node:fs";
import net from "node:net";
import path from "node:path";
import vm from "node:vm";
import { fileURLToPath } from "node:url";

const scriptsDir = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(scriptsDir, "..");
const partsDir = path.join(scriptsDir, "integration-readiness");
const partNames = [
  "00-config.mjs",
  "01-env-network.mjs",
  "02-validators.mjs",
  "10-auth-contracts.mjs",
  "20-public-read.mjs",
  "30-authenticated-read.mjs",
  "40-write-smoke.mjs",
  "41-withdraw-draft-interactions.mjs",
  "42-follow-write.mjs",
  "43-cross-account-admin-read.mjs",
  "50-im-notification-static.mjs",
  "51-creator-admin-static.mjs",
  "52-route-matrix-static.mjs",
  "53-auth-session-static.mjs",
  "54-publish-static.mjs",
  "55-content-static.mjs",
  "56-profile-static.mjs",
  "60-environment-and-main.mjs"
];
const source = partNames
  .map((name) => fs.readFileSync(path.join(partsDir, name), "utf8"))
  .join("\n");
const runtimeKey = Symbol.for("yuem.readiness.runtime");

globalThis[runtimeKey] = { spawnSync, dns, fs, net, path, repoRoot };
try {
  vm.runInThisContext(source, {
    filename: path.join(partsDir, "readiness-runtime.mjs"),
  });
} finally {
  delete globalThis[runtimeKey];
}
