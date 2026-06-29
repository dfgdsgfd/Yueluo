import { randomBytes } from "node:crypto";
import { chmodSync, copyFileSync, existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { spawnSync } from "node:child_process";

const root = path.resolve(import.meta.dirname, "..");
const secretsDir = path.join(root, ".release-secrets");
const storePath = path.join(secretsDir, "yuem-release.p12");
const propertiesPath = path.join(secretsDir, "keystore.properties");
const certificatePath = path.join(secretsDir, "yuem-release-certificate.pem");

if (existsSync(storePath) || existsSync(propertiesPath)) {
  throw new Error("Release signing material already exists; refusing to overwrite it.");
}

mkdirSync(secretsDir, { recursive: true, mode: 0o700 });
const workingDir = mkdtempSync(path.join(tmpdir(), "yuem-signing-"));
const privateKey = path.join(workingDir, "private-key.pem");
const certificate = path.join(workingDir, "certificate.pem");
const password = randomBytes(36).toString("base64url");

try {
  run("openssl", [
    "req",
    "-x509",
    "-newkey",
    "rsa:4096",
    "-sha256",
    "-nodes",
    "-days",
    "10000",
    "-subj",
    "/CN=Yuem/OU=Android/O=Yuelk/C=US",
    "-keyout",
    privateKey,
    "-out",
    certificate,
  ]);
  run("openssl", [
    "pkcs12",
    "-export",
    "-out",
    storePath,
    "-inkey",
    privateKey,
    "-in",
    certificate,
    "-name",
    "yuem-release",
    "-passout",
    `pass:${password}`,
  ]);
  copyFileSync(certificate, certificatePath);
  writeFileSync(
    propertiesPath,
    [
      "storeFile=../.release-secrets/yuem-release.p12",
      `storePassword=${password}`,
      "keyAlias=yuem-release",
      `keyPassword=${password}`,
      "storeType=PKCS12",
      "",
    ].join("\n"),
    { mode: 0o600 },
  );
  chmodSync(storePath, 0o600);
  chmodSync(certificatePath, 0o644);

  const fingerprint = spawnSync(
    "openssl",
    ["x509", "-in", certificatePath, "-noout", "-fingerprint", "-sha256"],
    { encoding: "utf8" },
  );
  if (fingerprint.status !== 0) {
    throw new Error(fingerprint.stderr || "Unable to read certificate fingerprint.");
  }
  writeFileSync(
    path.join(secretsDir, "certificate-fingerprint.txt"),
    `${fingerprint.stdout.trim()}\npackage=com.yuelk.xsewebfast\nalias=yuem-release\n`,
    { mode: 0o600 },
  );
} finally {
  rmSync(workingDir, { recursive: true, force: true });
}

console.log(`Generated release signing material in ${secretsDir}`);

function run(command, args) {
  const result = spawnSync(command, args, { stdio: "inherit" });
  if (result.status !== 0) {
    throw new Error(`${command} failed with exit code ${result.status}`);
  }
}
