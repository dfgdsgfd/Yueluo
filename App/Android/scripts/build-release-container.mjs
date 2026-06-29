import path from "node:path";
import { spawnSync } from "node:child_process";

const root = path.resolve(import.meta.dirname, "..");
const image = "yuem-android-builder:api36";

run(process.execPath, [path.join(root, "node_modules/@capacitor/cli/bin/capacitor"), "sync", "android"], root);
run("docker", ["buildx", "build", "--load", "--platform", "linux/amd64", "-f", "Dockerfile.build", "-t", image, "."], root);
run("docker", [
  "run", "--rm", "--platform", "linux/amd64",
  "-v", `${root}:/workspace`,
  "-v", "yuem-android-gradle-cache:/root/.gradle",
  "-w", "/workspace/android",
  image,
  "./gradlew", "clean", ":app:lintRelease", ":app:assembleRelease", ":app:bundleRelease", "--no-daemon",
], root);
run(process.execPath, [path.join(root, "scripts/build-release.mjs")], root, {
  YUEM_SKIP_SYNC: "1",
  YUEM_SKIP_GRADLE: "1",
});
run(process.execPath, [path.join(root, "scripts/verify-release.mjs")], root);

function run(command, args, cwd, extraEnvironment = {}) {
  const result = spawnSync(command, args, {
    cwd,
    env: { ...process.env, ...extraEnvironment },
    stdio: "inherit",
  });
  if (result.status !== 0) {
    throw new Error(`${command} failed with exit code ${result.status}`);
  }
}
