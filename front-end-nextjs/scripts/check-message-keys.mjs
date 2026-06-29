import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const messagesRoot = path.join(root, "src", "messages");
const locales = ["en", "zh-CN", "zh-TW", "vi", "ja", "ko"];

function flatten(value, prefix = "") {
  return Object.entries(value).flatMap(([key, child]) => {
    const next = prefix ? `${prefix}.${key}` : key;
    return child && typeof child === "object" && !Array.isArray(child) ? flatten(child, next) : [next];
  });
}

function loadLocale(locale) {
  const directory = path.join(messagesRoot, locale);
  const files = fs.readdirSync(directory).filter((file) => file.endsWith(".json")).sort();
  const messages = {};
  for (const file of files) {
    const bundle = JSON.parse(fs.readFileSync(path.join(directory, file), "utf8"));
    for (const [namespace, value] of Object.entries(bundle)) {
      assert(!(namespace in messages), `${locale}: duplicate namespace ${namespace}`);
      messages[namespace] = value;
    }
  }
  return { files, keys: flatten(messages).sort() };
}

const baseline = loadLocale("en");
for (const locale of locales) {
  const current = loadLocale(locale);
  assert.deepStrictEqual(current.files, baseline.files, `${locale}: message bundle set differs from en`);
  assert.deepStrictEqual(current.keys, baseline.keys, `${locale}: message key set differs from en`);
}

console.log(`${locales.length} locales share ${baseline.keys.length} message keys across ${baseline.files.length} bundles`);
