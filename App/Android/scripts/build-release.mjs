import { createHash } from "node:crypto";
import { copyFileSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";

const root = path.resolve(import.meta.dirname, "..");
const androidDir = path.join(root, "android");
const distDir = path.join(root, "dist");
const environment = {
  ...process.env,
  ANDROID_HOME: process.env.ANDROID_HOME ?? path.join(process.env.HOME ?? "", "Library/Android/sdk"),
  ANDROID_SDK_ROOT: process.env.ANDROID_SDK_ROOT ?? process.env.ANDROID_HOME ?? path.join(process.env.HOME ?? "", "Library/Android/sdk"),
};

if (process.env.YUEM_SKIP_SYNC !== "1") {
  run(process.execPath, [path.join(root, "node_modules/@capacitor/cli/bin/capacitor"), "sync", "android"], root);
}
if (process.env.YUEM_SKIP_GRADLE !== "1") {
  run("./gradlew", ["clean", ":app:lintRelease", ":app:assembleRelease", ":app:bundleRelease", "--no-daemon"], androidDir);
}

rmSync(distDir, { recursive: true, force: true });
mkdirSync(distDir, { recursive: true });
const artifacts = [
  {
    source: path.join(androidDir, "app/build/outputs/apk/release/app-release.apk"),
    output: path.join(distDir, "月梦-快速版-1.0.0-release.apk"),
  },
  {
    source: path.join(androidDir, "app/build/outputs/bundle/release/app-release.aab"),
    output: path.join(distDir, "月梦-快速版-1.0.0-release.aab"),
  },
];

const checksums = [];
for (const artifact of artifacts) {
  copyFileSync(artifact.source, artifact.output);
  const digest = createHash("sha256").update(readFileSync(artifact.output)).digest("hex");
  checksums.push(`${digest}  ${path.basename(artifact.output)}`);
}
writeFileSync(path.join(distDir, "SHA256SUMS"), `${checksums.join("\n")}\n`);
console.log(`Release artifacts written to ${distDir}`);

function run(command, args, cwd) {
  const result = spawnSync(command, args, { cwd, env: environment, stdio: "inherit" });
  if (result.status !== 0) {
    throw new Error(`${command} failed with exit code ${result.status}`);
  }
}
