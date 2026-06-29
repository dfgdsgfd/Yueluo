import { readFileSync } from "node:fs";
import { extname } from "node:path";
import { spawnSync } from "node:child_process";

const MAX_BYTES = 40_000;
const MAX_LINES = 800;
const TARGET_BYTES = 30_000;
const TARGET_LINES = 600;

const SOURCE_EXTENSIONS = new Set([".go", ".ts", ".tsx", ".mjs", ".py", ".css"]);
const EXCLUDED_PATHS = new Set([
  "backend-gin/internal/http/swaggerui/swagger-ui.css",
  "backend-gin/internal/http/swaggerui/scalar-api-reference.css",
]);

function isTestOrFixture(path) {
  return (
    path.includes("/tests/") ||
    path.includes("/testdata/") ||
    path.endsWith("_test.go") ||
    /\.(?:test|spec)\.(?:ts|tsx)$/.test(path) ||
    /(?:^|\/)test_[^/]+\.py$/.test(path)
  );
}

function trackedFiles() {
  const result = spawnSync(
    "git",
    ["ls-files", "--cached", "--others", "--exclude-standard", "-z"],
    { encoding: "utf8" },
  );
  if (result.status !== 0) {
    throw new Error(result.stderr || "git ls-files failed");
  }
  return result.stdout.split("\0").filter(Boolean);
}

const rows = trackedFiles()
  .filter((path) => SOURCE_EXTENSIONS.has(extname(path)))
  .filter((path) => !EXCLUDED_PATHS.has(path))
  .filter((path) => !isTestOrFixture(path))
  .map((path) => {
    const source = readFileSync(path, "utf8");
    return {
      path,
      bytes: Buffer.byteLength(source),
      lines: source === "" ? 0 : source.split(/\r?\n/).length,
    };
  })
  .sort((left, right) => right.bytes - left.bytes);

const violations = rows.filter((row) => row.bytes > MAX_BYTES || row.lines > MAX_LINES);
const targetMisses = rows.filter((row) => row.bytes > TARGET_BYTES || row.lines > TARGET_LINES);

const report = {
  status: violations.length === 0 ? "pass" : "fail",
  limits: { maxBytes: MAX_BYTES, maxLines: MAX_LINES },
  targets: { bytes: TARGET_BYTES, lines: TARGET_LINES },
  checkedFiles: rows.length,
  hardLimitViolations: violations,
  targetMisses,
  largestFiles: rows.slice(0, 20),
};

console.log(JSON.stringify(report, null, 2));
if (violations.length > 0) {
  process.exitCode = 1;
}
