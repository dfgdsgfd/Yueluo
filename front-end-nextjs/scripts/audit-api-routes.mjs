import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";
import ts from "typescript";

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const frontendRoot = path.resolve(__dirname, "..");
const repoRoot = path.resolve(frontendRoot, "..");
const srcRoot = path.join(frontendRoot, "src");
const typesPath = path.join(srcRoot, "lib", "types.ts");
const apiContractsPath = path.join(srcRoot, "lib", "api", "core", "contracts.ts");
const matrixPath = path.join(
  repoRoot,
  "backend-gin",
  "internal",
  "http",
  "routes",
  "route-matrix.json",
);

const directMethodHelpers = new Map([
  ["apiGet", "GET"],
  ["apiPost", "POST"],
  ["apiPut", "PUT"],
  ["apiDelete", "DELETE"],
  ["apiUpload", "POST"],
  ["apiDownload", "GET"],
]);

const methodOptionHelpers = new Set([
  "adminRequest",
  "apiAdminEnvelope",
  "apiAdminRequest",
  "apiRequestEnvelope",
]);

const metadataApiPathSources = [
  {
    file: "src/app/robots.ts",
    reason: "robots metadata exposes crawl allow/disallow path prefixes, not executable API calls.",
  },
  {
    file: "src/lib/private-entry-paths.ts",
    reason: "private entry validation rejects reserved /api paths, not executable API calls.",
  },
  {
    file: "src/lib/seo.ts",
    reason: "SEO metadata may publish public file URL prefixes, not executable API calls.",
  },
  {
    file: "src/app/api/[...path]/route.ts",
    reason: "catch-all route handler builds an upstream proxy target from request params, not a fixed frontend API call.",
  },
];

function readJSON(filePath) {
  return JSON.parse(fs.readFileSync(filePath, "utf8"));
}

function walkFiles(dir) {
  const files = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      files.push(...walkFiles(fullPath));
      continue;
    }
    if (/\.(tsx?|jsx?)$/.test(entry.name) && !entry.name.endsWith(".d.ts")) {
      files.push(fullPath);
    }
  }
  return files;
}

function expressionPath(expression) {
  if (ts.isStringLiteral(expression) || ts.isNoSubstitutionTemplateLiteral(expression)) {
    return expression.text;
  }

  if (ts.isTemplateExpression(expression)) {
    let value = expression.head.text;
    expression.templateSpans.forEach((span, index) => {
      value += `:dynamic${index + 1}${span.literal.text}`;
    });
    return value;
  }

  return null;
}

function nodeContains(parent, child) {
  return child.getStart() >= parent.getStart() && child.getEnd() <= parent.getEnd();
}

function callName(expression) {
  if (ts.isIdentifier(expression)) {
    return expression.text;
  }
  if (ts.isPropertyAccessExpression(expression)) {
    return expression.name.text;
  }
  return null;
}

function stringProperty(objectLiteral, name) {
  for (const property of objectLiteral.properties) {
    if (!ts.isPropertyAssignment(property)) {
      continue;
    }
    const propertyName = ts.isIdentifier(property.name) || ts.isStringLiteral(property.name)
      ? property.name.text
      : null;
    if (propertyName !== name) {
      continue;
    }
    const initializer = property.initializer;
    if (ts.isStringLiteral(initializer) || ts.isNoSubstitutionTemplateLiteral(initializer)) {
      return initializer.text.toUpperCase();
    }
  }
  return null;
}

function methodFromCall(callExpression, pathNode) {
  const name = callName(callExpression.expression);

  if (directMethodHelpers.has(name)) {
    return directMethodHelpers.get(name);
  }

  if (methodOptionHelpers.has(name)) {
    const options = callExpression.arguments[1];
    if (options && ts.isObjectLiteralExpression(options)) {
      return stringProperty(options, "method") ?? "GET";
    }
    return "GET";
  }

  if (name === "buildApiUrl") {
    const fetchCall = callExpression.parent;
    if (
      fetchCall &&
      ts.isCallExpression(fetchCall) &&
      callName(fetchCall.expression) === "fetch" &&
      fetchCall.arguments[0] === callExpression
    ) {
      const options = fetchCall.arguments[1];
      if (options && ts.isObjectLiteralExpression(options)) {
        return stringProperty(options, "method") ?? "GET";
      }
      return "GET";
    }
  }

  if (
    name === "URL" &&
    ts.isNewExpression(callExpression) &&
    callExpression.arguments?.[0] &&
    nodeContains(callExpression.arguments[0], pathNode)
  ) {
    return "GET";
  }

  return "ANY";
}

function enclosingCallForPathNode(node) {
  let current = node.parent;
  while (current) {
    if (
      ts.isCallExpression(current) &&
      current.arguments[0] &&
      nodeContains(current.arguments[0], node)
    ) {
      return current;
    }
    if (
      ts.isNewExpression(current) &&
      current.arguments?.[0] &&
      nodeContains(current.arguments[0], node)
    ) {
      return current;
    }
    current = current.parent;
  }
  return null;
}

function collectFrontendCalls() {
  const calls = new Map();
  const ignored = [];

  for (const file of walkFiles(srcRoot)) {
    const text = fs.readFileSync(file, "utf8");
    const sourceFile = ts.createSourceFile(file, text, ts.ScriptTarget.Latest, true);
    const rel = path.relative(frontendRoot, file).replaceAll(path.sep, "/");
    const metadataRule = metadataApiPathSources.find((rule) => rule.file === rel);

    function visit(node) {
      const apiPath = expressionPath(node);
      if (apiPath?.startsWith("/api/")) {
        const line = sourceFile.getLineAndCharacterOfPosition(node.getStart()).line + 1;
        if (metadataRule) {
          ignored.push({
            path: apiPath,
            source: `${rel}:${line}`,
            reason: metadataRule.reason,
          });
          ts.forEachChild(node, visit);
          return;
        }
        const callExpression = enclosingCallForPathNode(node);
        const method = callExpression ? methodFromCall(callExpression, node) : "ANY";
        const key = `${method} ${apiPath}`;
        const current = calls.get(key) ?? { method, path: apiPath, sources: [] };
        current.sources.push(`${rel}:${line}`);
        calls.set(key, current);
      }
      ts.forEachChild(node, visit);
    }

    visit(sourceFile);
  }

  return {
    calls: [...calls.values()].sort((a, b) =>
      a.path.localeCompare(b.path) || a.method.localeCompare(b.method),
    ),
    ignored,
  };
}

function collectStringUnionMembers(typeName) {
  const text = fs.readFileSync(typesPath, "utf8");
  const sourceFile = ts.createSourceFile(typesPath, text, ts.ScriptTarget.Latest, true);
  const values = [];

  function visit(node) {
    if (
      ts.isTypeAliasDeclaration(node) &&
      node.name.text === typeName &&
      ts.isUnionTypeNode(node.type)
    ) {
      for (const member of node.type.types) {
        if (
          ts.isLiteralTypeNode(member) &&
          ts.isStringLiteral(member.literal)
        ) {
          values.push(member.literal.text);
        }
      }
    }
    ts.forEachChild(node, visit);
  }

  visit(sourceFile);
  return values;
}

function expandKnownFrontendDynamics(frontendCalls) {
  const adminResources = collectStringUnionMembers("AdminListResource");

  return frontendCalls.flatMap((call) => {
    if (call.path === "/api/:dynamic1") {
      return expandPathFromReturnValues(call, "getSwaggerDocsUrl", "");
    }

    if (call.path === "/api/:dynamic1.json") {
      return expandPathFromReturnValues(call, "getSwaggerJsonUrl", ".json");
    }

    if (call.path !== "/api/admin/:dynamic1") {
      return [call];
    }

    return adminResources.map((resource) => ({
      ...call,
      path: `/api/admin/${resource}`,
      expandedFrom: call.path,
    }));
  }).sort((a, b) =>
    a.path.localeCompare(b.path) || a.method.localeCompare(b.method),
  );
}

function expandPathFromReturnValues(call, functionName, suffix) {
  const paths = collectApiPathReturnValues(functionName, suffix);
  if (paths.length === 0) {
    return [call];
  }
  return paths.map((apiPath) => ({
    ...call,
    path: apiPath,
    expandedFrom: call.path,
  }));
}

function collectApiPathReturnValues(functionName, suffix) {
  const text = fs.readFileSync(path.join(srcRoot, "lib", "api", "auth.ts"), "utf8");
  const sourceFile = ts.createSourceFile("auth.ts", text, ts.ScriptTarget.Latest, true);
  const paths = new Set();
  const defaultSwaggerDocsPath = collectStringConst(apiContractsPath, "DEFAULT_SWAGGER_DOCS_PATH");

  function literalText(node) {
    if (ts.isStringLiteral(node) || ts.isNoSubstitutionTemplateLiteral(node)) {
      return node.text;
    }
    return null;
  }

  function visit(node) {
    if (
      ts.isFunctionDeclaration(node) &&
      node.name?.text === functionName &&
      node.body
    ) {
      for (const statement of node.body.statements) {
        if (!ts.isReturnStatement(statement) || !statement.expression) {
          continue;
        }
        const expression = statement.expression;
        if (ts.isTemplateExpression(expression) && expression.head.text === "/api/") {
          if (defaultSwaggerDocsPath) {
            paths.add(`${defaultSwaggerDocsPath}${suffix}`);
          }
        } else {
          const value = literalText(expression);
          if (value?.startsWith("/api/")) {
            paths.add(value);
          }
        }
      }
    }
    ts.forEachChild(node, visit);
  }

  visit(sourceFile);
  return [...paths].map((apiPath) => apiPath.startsWith("/api/") ? apiPath : `/api/${apiPath}`);
}

function collectStringConst(filePath, name) {
  const text = fs.readFileSync(filePath, "utf8");
  const sourceFile = ts.createSourceFile(filePath, text, ts.ScriptTarget.Latest, true);
  let value = null;

  function visit(node) {
    if (ts.isVariableDeclaration(node) && ts.isIdentifier(node.name) && node.name.text === name) {
      const initializer = node.initializer;
      if (initializer && (ts.isStringLiteral(initializer) || ts.isNoSubstitutionTemplateLiteral(initializer))) {
        value = initializer.text;
      }
    }
    ts.forEachChild(node, visit);
  }

  visit(sourceFile);
  return value;
}

function pathSegments(value) {
  return value.replace(/[?#].*$/, "").replace(/^\/+|\/+$/g, "").split("/");
}

function dynamicSegment(segment) {
  return (
    segment.startsWith(":") ||
    segment.startsWith("*") ||
    segment.startsWith("{") ||
    (segment.startsWith("${") && segment.endsWith("}")) ||
    segment === "[id]"
  );
}

function patternsIntersect(frontendPath, backendPath) {
  const frontend = pathSegments(frontendPath);
  const backend = pathSegments(backendPath);
  if (frontend.length !== backend.length) {
    return false;
  }

  return frontend.every((frontSegment, index) => {
    const backendSegment = backend[index];
    return (
      frontSegment === backendSegment ||
      dynamicSegment(frontSegment) ||
      dynamicSegment(backendSegment)
    );
  });
}

function methodsMatch(frontendMethod, backendMethod) {
  return (
    frontendMethod === "ANY" ||
    backendMethod === "ALL" ||
    frontendMethod.toUpperCase() === backendMethod.toUpperCase()
  );
}

function loadBackendRoutes() {
  const matrix = readJSON(matrixPath);
  const routes = matrix.routes.map((route) => ({
    method: route.method,
    path: route.path,
    source: route.sourceFile,
  }));

  for (const ws of matrix.webSockets ?? []) {
    routes.push({
      method: "GET",
      path: ws.path,
      source: ws.sourceFile,
    });
  }

  for (const route of matrix.routes) {
    if (route.method === "GET" && route.path === "/api/${SWAGGER_DOCS_PATH}.json") {
      routes.push({
        method: "GET",
        path: "/api/${SWAGGER_DOCS_PATH}",
        source: `${route.sourceFile}#registered-debug-page`,
      });
    }
  }

  return {
    expectedRoutes: matrix.summary.totalApiRoutes,
    expectedWebSockets: matrix.webSockets?.length ?? 0,
    routes,
  };
}

function main() {
  const backend = loadBackendRoutes();
  const frontend = collectFrontendCalls();
  const frontendCalls = expandKnownFrontendDynamics(frontend.calls);

  const results = frontendCalls.map((call) => {
    const matches = backend.routes.filter(
      (route) => methodsMatch(call.method, route.method) && patternsIntersect(call.path, route.path),
    );
    return { ...call, matches };
  });

  const unmatched = results.filter((result) => result.matches.length === 0);
  const ambiguous = results.filter((result) => result.matches.length > 12);

  console.log(
    JSON.stringify(
      {
        status: unmatched.length === 0 ? "pass" : "attention-required",
        summary: {
          frontendApiCalls: frontendCalls.length,
          unmatchedApiCalls: unmatched.length,
          broadDynamicMatches: ambiguous.length,
          ignoredMetadataReferences: frontend.ignored.length,
          backendApiRoutes: backend.expectedRoutes,
          backendWebSockets: backend.expectedWebSockets,
        },
        ignoredMetadataReferences: frontend.ignored,
        unmatched: unmatched.map(({ method, path: apiPath, sources, expandedFrom }) => ({
          method,
          path: apiPath,
          sources,
          expandedFrom,
        })),
        broadDynamicMatches: ambiguous.map(({ method, path: apiPath, sources, expandedFrom, matches }) => ({
          method,
          path: apiPath,
          sources,
          expandedFrom,
          matchCount: matches.length,
        })),
      },
      null,
      2,
    ),
  );

  if (unmatched.length > 0) {
    process.exitCode = 1;
  }
}

main();
