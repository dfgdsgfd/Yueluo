import { spawnSync } from "node:child_process";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __filename = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(__filename), "..");
const readinessScriptPath = path.join(repoRoot, "scripts", "check-integration-readiness.mjs");

const sensitiveEnvKeys = [
  "INTEGRATION_USER_A_ACCESS_TOKEN",
  "INTEGRATION_USER_A_ID",
  "INTEGRATION_USER_A_PASSWORD",
  "INTEGRATION_USER_B_ACCESS_TOKEN",
  "INTEGRATION_USER_B_ID",
  "INTEGRATION_USER_B_PASSWORD",
  "INTEGRATION_ADMIN_ACCESS_TOKEN",
  "INTEGRATION_ADMIN_USERNAME",
  "INTEGRATION_ADMIN_PASSWORD",
  "INTEGRATION_WRITE_SMOKE_POST_ID",
];

const writeSmokeIds = [
  "user-withdraw-payment-code-write-smoke",
  "user-draft-post-write-smoke",
  "user-post-interaction-write-smoke",
  "user-cross-account-follow-write-smoke",
  "user-cross-account-im-write-smoke",
  "admin-runtime-toggle-write-smoke",
];

const assertions = [];

function envFor(overrides = {}) {
  const env = {
    ...process.env,
    INTEGRATION_HTTP_RETRY_COUNT: "0",
  };

  for (const key of sensitiveEnvKeys) {
    env[key] = "";
  }
  env.INTEGRATION_ENABLE_WRITE_SMOKE = "";
  env.FEED_FIXTURE_FALLBACK = "";
  env.NEXT_PUBLIC_FEED_FIXTURE_FALLBACK = "";
  env.API_BASE_URL = "";
  env.BACKEND_ORIGIN = "";
  env.NEXT_PUBLIC_API_BASE_URL = "";
  env.NEXT_PUBLIC_BACKEND_ORIGIN = "";

  return {
    ...env,
    ...overrides,
  };
}

function runReadiness(name, overrides = {}) {
  const result = spawnSync(process.execPath, [readinessScriptPath], {
    cwd: repoRoot,
    encoding: "utf8",
    env: envFor(overrides),
    maxBuffer: 30 * 1024 * 1024,
  });

  let report = null;
  try {
    report = JSON.parse(result.stdout);
  } catch (error) {
    throw new Error(`${name}: readiness output is not valid JSON: ${error.message}`);
  }

  return {
    name,
    status: result.status,
    stdout: result.stdout,
    stderr: result.stderr,
    report,
  };
}

function checkById(report, id) {
  return report.checks.find((check) => check.id === id) ?? null;
}

function assertScenario(condition, scenario, message, details = {}) {
  assertions.push({
    scenario,
    status: condition ? "pass" : "fail",
    message,
    details,
  });
}

function assertCheckStatus(run, id, expectedStatus) {
  const check = checkById(run.report, id);
  assertScenario(
    check?.status === expectedStatus,
    run.name,
    `${id} should be ${expectedStatus}`,
    {
      actualStatus: check?.status ?? null,
      actualMessage: check?.message ?? null,
    },
  );
}

function assertNoTokenLeak(run, token) {
  assertScenario(
    !run.stdout.includes(token) && !run.stderr.includes(token),
    run.name,
    "readiness report should not leak the supplied token",
  );
}

function writeSmokeChecks(run) {
  return run.report.checks.filter((check) => writeSmokeIds.includes(check.id));
}

function assertWriteSmokeOriginMissing(run, id) {
  const check = checkById(run.report, id);
  assertScenario(
    check?.status === "fail" && String(check.message).includes("BACKEND_ORIGIN"),
    run.name,
    `${id} should fail explicitly when backend origin is missing`,
    {
      actualStatus: check?.status ?? null,
      actualMessage: check?.message ?? null,
    },
  );
}

const runs = [];

const baseline = runReadiness("baseline-no-write-smoke");
runs.push(baseline);
assertCheckStatus(baseline, "frontend-feed-fixture-fallback", "pass");
assertCheckStatus(baseline, "frontend-env-doc-contract", "pass");
assertScenario(
  writeSmokeChecks(baseline).length === 0,
  baseline.name,
  "write smoke checks should not run by default",
  { writeSmokeChecks: writeSmokeChecks(baseline).map((check) => check.id) },
);

const fixtureFallback = runReadiness("fixture-fallback-gate", {
  FEED_FIXTURE_FALLBACK: "true",
});
runs.push(fixtureFallback);
assertCheckStatus(fixtureFallback, "frontend-feed-fixture-fallback", "fail");

const writeSmokeNoCredentials = runReadiness("write-smoke-no-credentials", {
  INTEGRATION_ENABLE_WRITE_SMOKE: "true",
});
runs.push(writeSmokeNoCredentials);
for (const id of writeSmokeIds) {
  assertCheckStatus(writeSmokeNoCredentials, id, "fail");
}

const invalidUserToken = "codex-invalid-token";
const invalidUserRun = runReadiness("write-smoke-invalid-user-token", {
  INTEGRATION_ENABLE_WRITE_SMOKE: "true",
  INTEGRATION_USER_A_ACCESS_TOKEN: invalidUserToken,
  INTEGRATION_WRITE_SMOKE_POST_ID: "codex-smoke-post",
});
runs.push(invalidUserRun);
for (const id of [
  "user-withdraw-payment-code-write-smoke",
  "user-draft-post-write-smoke",
  "user-post-interaction-write-smoke",
]) {
  assertWriteSmokeOriginMissing(invalidUserRun, id);
}
assertNoTokenLeak(invalidUserRun, invalidUserToken);

const invalidAdminToken = "codex-invalid-admin-token";
const invalidAdminRun = runReadiness("write-smoke-invalid-admin-token", {
  INTEGRATION_ENABLE_WRITE_SMOKE: "true",
  INTEGRATION_ADMIN_ACCESS_TOKEN: invalidAdminToken,
});
runs.push(invalidAdminRun);
assertWriteSmokeOriginMissing(invalidAdminRun, "admin-runtime-toggle-write-smoke");
assertNoTokenLeak(invalidAdminRun, invalidAdminToken);

const failures = assertions.filter((assertion) => assertion.status === "fail");
const summary = {
  status: failures.length === 0 ? "pass" : "fail",
  scenarios: runs.map((run) => ({
    name: run.name,
    readinessStatus: run.report.status,
    summary: run.report.summary,
    exitStatus: run.status,
  })),
  assertions,
};

console.log(JSON.stringify(summary, null, 2));
if (failures.length > 0) {
  process.exitCode = 1;
}
