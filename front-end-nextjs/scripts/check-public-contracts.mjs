import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import ts from "typescript";
import { fileURLToPath } from "node:url";

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const snapshotPath = path.join(root, "scripts", "public-contracts.snapshot.json");
const entryPaths = {
  api: path.join(root, "src", "lib", "api.ts"),
  types: path.join(root, "src", "lib", "types.ts"),
};
const configPath = path.join(root, "tsconfig.json");
const configFile = ts.readConfigFile(configPath, ts.sys.readFile);
const parsed = ts.parseJsonConfigFileContent(configFile.config, ts.sys, root);
const program = ts.createProgram({ rootNames: Object.values(entryPaths), options: parsed.options });
const checker = program.getTypeChecker();

const snapshot = Object.fromEntries(
  Object.entries(entryPaths).map(([name, entryPath]) => {
    const sourceFile = program.getSourceFile(entryPath);
    if (!sourceFile) throw new Error(`missing TypeScript entry ${entryPath}`);
    const moduleSymbol = checker.getSymbolAtLocation(sourceFile);
    if (!moduleSymbol) throw new Error(`missing module symbol for ${entryPath}`);
    const exports = checker
      .getExportsOfModule(moduleSymbol)
      .map((symbol) => symbol.getName())
      .filter((exportName) => exportName !== "default")
      .sort();
    return [name, exports];
  }),
);

if (process.argv.includes("--update")) {
  fs.writeFileSync(snapshotPath, `${JSON.stringify(snapshot, null, 2)}\n`);
  console.log(`updated ${path.relative(root, snapshotPath)}`);
} else {
  const expected = JSON.parse(fs.readFileSync(snapshotPath, "utf8"));
  assert.deepStrictEqual(snapshot, expected, "public API/type exports changed");
  console.log(`public contracts match ${path.relative(root, snapshotPath)}`);
}
