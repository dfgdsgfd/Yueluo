import fs from "node:fs";
import path from "node:path";
import vm from "node:vm";
import { fileURLToPath } from "node:url";

const projectRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const nextRoot = path.join(projectRoot, ".next");
const sharedBaseBudget = {
  maxKb: 1200,
};
const routeBudgets = {
  root: {
    manifest: "server/app/page_client-reference-manifest.js",
    key: "/page",
    maxKb: 620,
  },
  login: {
    manifest: "server/app/login/page_client-reference-manifest.js",
    key: "/login/page",
    maxKb: 320,
  },
  messages: {
    manifest: "server/app/messages/page_client-reference-manifest.js",
    key: "/messages/page",
    maxKb: 370,
  },
  notifications: {
    manifest: "server/app/notifications/page_client-reference-manifest.js",
    key: "/notifications/page",
    maxKb: 365,
  },
  profile: {
    manifest: "server/app/profile/page_client-reference-manifest.js",
    key: "/profile/page",
    maxKb: 450,
  },
  post: {
    manifest: "server/app/post/page_client-reference-manifest.js",
    key: "/post/page",
    maxKb: 500,
  },
  publish: {
    manifest: "server/app/publish/page_client-reference-manifest.js",
    key: "/publish/page",
    maxKb: 440,
  },
  wallet: {
    manifest: "server/app/wallet/page_client-reference-manifest.js",
    key: "/wallet/page",
    maxKb: 350,
  },
  adminLogin: {
    manifest: "server/app/admin/page_client-reference-manifest.js",
    key: "/admin/page",
    maxKb: 350,
  },
  adminConsole: {
    manifest: "server/app/admin/console/page_client-reference-manifest.js",
    key: "/admin/console/page",
    maxKb: 400,
  },
};

if (!fs.existsSync(nextRoot)) {
  throw new Error("Missing .next build output. Run npm run build first.");
}

function routeEntryFiles(budget) {
  const manifestPath = path.join(nextRoot, budget.manifest);
  const sandbox = { globalThis: {} };
  vm.runInNewContext(fs.readFileSync(manifestPath, "utf8"), sandbox);
  const manifest = sandbox.globalThis.__RSC_MANIFEST?.[budget.key];
  if (!manifest) {
    throw new Error(`Missing client reference manifest entry for ${budget.key}.`);
  }
  return new Set(Object.values(manifest.entryJSFiles).flat());
}

function fileBytes(file) {
  return fs.statSync(path.join(nextRoot, file)).size;
}

function toKb(bytes) {
  return Math.ceil(bytes / 1024);
}

const routeFiles = new Map(
  Object.entries(routeBudgets).map(([name, budget]) => [
    name,
    routeEntryFiles(budget),
  ]),
);
const routeCount = routeFiles.size;
const fileRouteCounts = new Map();
for (const files of routeFiles.values()) {
  for (const file of files) {
    fileRouteCounts.set(file, (fileRouteCounts.get(file) ?? 0) + 1);
  }
}
const sharedFiles = new Set(
  Array.from(fileRouteCounts.entries())
    .filter(([, count]) => count === routeCount)
    .map(([file]) => file),
);
const sharedBytes = Array.from(sharedFiles).reduce(
  (total, file) => total + fileBytes(file),
  0,
);
const sharedKb = toKb(sharedBytes);
const sharedStatus = sharedKb <= sharedBaseBudget.maxKb ? "PASS" : "FAIL";
console.log(`${sharedStatus} sharedBase: ${sharedKb}KB / ${sharedBaseBudget.maxKb}KB`);

let failed = false;
failed ||= sharedKb > sharedBaseBudget.maxKb;

for (const [name, budget] of Object.entries(routeBudgets)) {
  const files = routeFiles.get(name);
  const routeBytes = Array.from(files).reduce(
    (total, file) => total + (sharedFiles.has(file) ? 0 : fileBytes(file)),
    0,
  );
  const totalBytes = Array.from(files).reduce(
    (total, file) => total + fileBytes(file),
    0,
  );
  const kb = toKb(routeBytes);
  const totalKb = toKb(totalBytes);
  const status = kb <= budget.maxKb ? "PASS" : "FAIL";
  console.log(`${status} ${name}: ${kb}KB / ${budget.maxKb}KB (+${sharedKb}KB shared, ${totalKb}KB total)`);
  failed ||= kb > budget.maxKb;
}

if (failed) {
  process.exitCode = 1;
}
