import { existsSync } from "node:fs";
import path from "node:path";
import { spawnSync } from "node:child_process";

const root = path.resolve(import.meta.dirname, "..");
const apk = path.join(root, "dist/月梦-快速版-1.0.0-release.apk");
const aab = path.join(root, "dist/月梦-快速版-1.0.0-release.aab");
const image = "yuem-android-builder:api36";

for (const required of [apk, aab]) {
  if (!existsSync(required)) {
    throw new Error(`Missing ${required}`);
  }
}

const imageCheck = spawnSync("docker", ["image", "inspect", image], { cwd: root, stdio: "ignore" });
if (imageCheck.status !== 0) {
  run("docker", ["buildx", "build", "--load", "--platform", "linux/amd64", "-f", "Dockerfile.build", "-t", image, "."]);
}

run("docker", [
  "run", "--rm", "--platform", "linux/amd64",
  "-v", `${root}:/workspace`,
  "-w", "/workspace",
  image,
  "bash", "-lc",
  "/opt/android-sdk/build-tools/36.0.0/apksigner verify --verbose --print-certs dist/月梦-快速版-1.0.0-release.apk && java -jar /opt/bundletool.jar validate --bundle=dist/月梦-快速版-1.0.0-release.aab && jarsigner -verify dist/月梦-快速版-1.0.0-release.aab",
]);
console.log("APK signature and AAB structure/signature verified.");

function run(command, args) {
  const result = spawnSync(command, args, { cwd: root, stdio: "inherit" });
  if (result.status !== 0) {
    throw new Error(`${command} failed with exit code ${result.status}`);
  }
}
