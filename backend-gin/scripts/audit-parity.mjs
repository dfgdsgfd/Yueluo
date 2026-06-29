import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const ginRoot = path.resolve(__dirname, '..');
const repoRoot = path.resolve(ginRoot, '..');

const matrixPath = path.join(ginRoot, 'internal', 'http', 'routes', 'route-matrix.json');
const handlersDir = path.join(ginRoot, 'internal', 'http', 'handlers');
const matrixAdminPath = path.join(handlersDir, 'matrix_admin.go');
const nativeRegisterPath = path.join(ginRoot, 'internal', 'http', 'routes', 'register.go');
const defaultExpressRef = path.resolve(repoRoot, '..', 'yuem-web', 'backend');
const expressRef = process.env.EXPRESS_REFERENCE_DIR
  ? path.resolve(process.env.EXPRESS_REFERENCE_DIR)
  : defaultExpressRef;

function readText(filePath) {
  return fs.readFileSync(filePath, 'utf8');
}

function walkFiles(dir, suffixes) {
  const out = [];
  if (!fs.existsSync(dir)) return out;
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      out.push(...walkFiles(fullPath, suffixes));
    } else if (suffixes.some((suffix) => entry.name.endsWith(suffix))) {
      out.push(fullPath);
    }
  }
  return out;
}

function loadMatrix() {
  return JSON.parse(readText(matrixPath));
}

function countBy(values, keyFn) {
  const counts = new Map();
  for (const value of values) {
    const key = keyFn(value);
    counts.set(key, (counts.get(key) || 0) + 1);
  }
  return [...counts.entries()]
    .map(([key, count]) => ({ key, count }))
    .sort((a, b) => b.count - a.count || a.key.localeCompare(b.key));
}

function functionAtOffset(text, offset) {
  const prefix = text.slice(0, offset);
  const matches = [...prefix.matchAll(/func\s+(?:\([^)]+\)\s*)?([A-Za-z0-9_]+)\s*\(/g)];
  return matches.length ? matches[matches.length - 1][1] : null;
}

function addFinding(findings, finding) {
  findings.push(finding);
}

function lineAtOffset(text, offset) {
  return text.slice(0, offset).split(/\r?\n/).length;
}

function acceptedReviewFinding(finding) {
  if (
    finding.file.endsWith('_test.go') &&
    finding.id === 'unfinished-marker' &&
    finding.excerpt.includes('todo')
  ) {
    return 'Test fixture rich text intentionally contains literal task item text.';
  }
  if (
    finding.file.endsWith('_test.go') &&
    finding.id === 'disabled-integration-shape'
  ) {
    return 'Test assertion intentionally covers disabled integration behavior.';
  }
  if (
    finding.id === 'empty-result-branch' &&
    finding.file === 'backend-gin/internal/http/handlers/database_admin.go' &&
    finding.function === 'AdminDatabaseTables'
  ) {
    return 'Admin database table listing preserves legacy empty-list shape when no database is configured.';
  }
  if (
    finding.id === 'empty-result-branch' &&
    finding.file === 'backend-gin/internal/http/handlers/matrix_admin_runtime.go' &&
    ['adminSystemLogs', 'adminObservabilityAccessLog'].includes(finding.function)
  ) {
    return 'Runtime observability endpoints preserve enabled=false plus empty items when backing storage is unavailable.';
  }
  if (
    finding.id === 'disabled-integration-shape' &&
    finding.excerpt.includes('error.payment_method_disabled')
  ) {
    return 'Paid-content integrations intentionally return i18n disabled-state keys when a payment method is unavailable.';
  }
  if (
    finding.id === 'disabled-integration-shape' &&
    finding.excerpt.includes('error.image_protection_disabled')
  ) {
    return 'Image protection integrations intentionally return an i18n disabled-state key when the feature is unavailable.';
  }
  if (
    finding.id === 'disabled-integration-shape' &&
    finding.excerpt.includes('account_disabled')
  ) {
    return 'Authentication audit logs intentionally record disabled-account failures with the legacy reason code.';
  }
  if (
    finding.id === 'disabled-integration-shape' &&
    finding.excerpt.includes('error.oauth2_disabled')
  ) {
    return 'OAuth app/mobile endpoints intentionally return an i18n disabled-state key when OAuth is unavailable.';
  }
  if (
    finding.id === 'disabled-integration-shape' &&
    finding.file === 'backend-gin/internal/http/handlers/post_image_archives.go' &&
    finding.excerpt.includes('"disabled"')
  ) {
    return 'Post image archive status intentionally exposes disabled state while preserving the legacy response shape.';
  }
  if (
    finding.id === 'disabled-integration-shape'
    && finding.file === 'backend-gin/internal/http/handlers/license.go'
    && finding.function === 'VerifyLicense'
    && finding.excerpt.includes('"reason": "disabled"')
  ) {
    return 'Legacy license.js returns data.valid=false with reason=disabled for inactive licenses.';
  }
  if (
    finding.id === 'disabled-integration-shape'
    && finding.file === 'backend-gin/internal/http/handlers/matrix_auth_oauth.go'
    && finding.function === 'oauthCallback'
    && finding.excerpt.includes('oauth2_disabled')
  ) {
    return 'Legacy auth.js redirects disabled OAuth callbacks to /?error=oauth2_disabled.';
  }
  if (
    finding.id === 'defensive-unowned-matrix-source'
    && finding.file === 'backend-gin/internal/http/handlers/matrix.go'
    && finding.function === 'MatrixRoute'
  ) {
    return 'Defensive fallback only; router tests require every matrix source to have native ownership.';
  }
  return '';
}

function scanGoHandlers() {
  const findings = [];
  const accepted = [];
  for (const file of walkFiles(handlersDir, ['.go'])) {
    const text = readText(file);
    const lines = text.split(/\r?\n/);
    const rel = path.relative(repoRoot, file).replaceAll(path.sep, '/');

    const scans = [
      {
        id: 'swallowed-error-empty-success',
        severity: 'high',
        re: /if\s+err\s*:=\s*[^{}]+;\s*err\s*!=\s*nil\s*\{\s*writeSuccess\(c,[^{}]*(?:\[\]gin\.H\{\}|gin\.H\{\})/gs,
      },
      {
        id: 'fixed-zero-or-empty-stats',
        severity: 'high',
        re: /"(?:today_active_users|processed|succeeded|failed|skipped)"\s*:\s*0|"version_updates"\s*:\s*\[\]gin\.H\{\}|"platform_stats"\s*:\s*\[\]gin\.H\{\}/g,
        allowFunctions: new Set(['adminGenerateMissingCovers']),
      },
      {
        id: 'unfinished-marker',
        severity: 'high',
        re: /not implemented|TODO|FIXME|placeholder route|stub route/gi,
      },
      {
        id: 'empty-result-branch',
        severity: 'medium',
        re: /writeSuccess\(c,[^\n]*(?:\[\]gin\.H\{\}|gin\.H\{"messages":\s*\[\]gin\.H\{\})/g,
        allowFunctions: new Set([
          'adminQueueDispatch',
          'adminGenerateMissingCovers',
          'imConversations',
          'imSync',
          'imUsers',
        ]),
      },
      {
        id: 'defensive-unowned-matrix-source',
        severity: 'medium',
        re: /route source not registered/g,
      },
      {
        id: 'disabled-integration-shape',
        severity: 'medium',
        re: /not enabled|disabled/g,
      },
    ];

    for (const scan of scans) {
      for (const match of text.matchAll(scan.re)) {
        const line = lineAtOffset(text, match.index);
        const functionName = functionAtOffset(text, match.index);
        if (scan.allowFunctions?.has(functionName)) continue;
        const finding = {
          id: scan.id,
          severity: scan.severity,
          file: rel,
          line,
          function: functionName,
          excerpt: lines[line - 1].trim(),
        };
        const acceptedReason = acceptedReviewFinding(finding);
        if (acceptedReason) {
          accepted.push({ ...finding, acceptedReason });
          continue;
        }
        addFinding(findings, finding);
      }
    }
  }
  return { findings, accepted };
}

const requiredAdminResources = {
  admins: { hidden: ['password'] },
  announcements: {},
  audit: {},
  'banned-word-categories': {},
  'banned-words': {},
  categories: {},
  collections: {},
  comments: {},
  'content-review': {},
  feedback: {},
  follows: {},
  licenses: {},
  likes: {},
  'media-library': {},
  'notification-templates': {},
  'open-apis': { hidden: ['api_key'] },
  posts: {},
  reports: {},
  'system-notifications': {},
  tags: {},
  'user-toolbar': {},
  users: { hidden: ['password'] },
};

function extractGoMapEntry(text, key) {
  const needle = `"${key}":`;
  const start = text.indexOf(needle);
  if (start < 0) return null;
  const open = text.indexOf('{', start);
  if (open < 0) return null;
  let depth = 0;
  for (let i = open; i < text.length; i += 1) {
    const ch = text[i];
    if (ch === '{') depth += 1;
    if (ch === '}') {
      depth -= 1;
      if (depth === 0) {
        return {
          start,
          line: lineAtOffset(text, start),
          text: text.slice(start, i + 1),
        };
      }
    }
  }
  return null;
}

function scanAdminCompatibilityConfig(matrix) {
  const findings = [];
  const matrixAdminText = readText(matrixAdminPath);
  const nativeRegisterText = readText(nativeRegisterPath);
  const rel = path.relative(repoRoot, matrixAdminPath).replaceAll(path.sep, '/');
  const resources = new Map();

  for (const key of Object.keys(requiredAdminResources)) {
    const entry = extractGoMapEntry(matrixAdminText, key);
    if (!entry) {
      addFinding(findings, {
        id: 'admin-resource-config-missing',
        severity: 'high',
        file: rel,
        line: 1,
        function: 'adminResources',
        excerpt: key,
      });
      continue;
    }
    resources.set(key, entry);
    for (const field of ['Filters:', 'SortFields:', 'ListShape:']) {
      if (!entry.text.includes(field)) {
        addFinding(findings, {
          id: 'admin-resource-config-incomplete',
          severity: 'high',
          file: rel,
          line: entry.line,
          function: 'adminResources',
          excerpt: `${key} missing ${field}`,
        });
      }
    }
    for (const hiddenField of requiredAdminResources[key].hidden || []) {
      if (!entry.text.includes('HiddenFields:') || !entry.text.includes(`"${hiddenField}"`)) {
        addFinding(findings, {
          id: 'admin-sensitive-field-exposed',
          severity: 'high',
          file: rel,
          line: entry.line,
          function: 'adminResources',
          excerpt: `${key} must hide ${hiddenField}`,
        });
      }
    }
  }

  const adminSegments = new Map();
  for (const route of matrix.routes.filter((route) => route.path.startsWith('/api/admin/'))) {
    const segment = route.path.replace('/api/admin/', '').split('/')[0];
    if (!adminSegments.has(segment)) adminSegments.set(segment, []);
    adminSegments.get(segment).push(`${route.method} ${route.path}`);
  }

  for (const [segment, routes] of adminSegments) {
    const hasResource = extractGoMapEntry(matrixAdminText, segment) !== null;
    const hasSpecialDispatch =
      matrixAdminText.includes(`/api/admin/${segment}`) ||
      nativeRegisterText.includes(`/api/admin/${segment}`);
    if (hasResource || hasSpecialDispatch) continue;
    addFinding(findings, {
      id: 'admin-matrix-segment-unhandled',
      severity: 'high',
      file: rel,
      line: 1,
      function: 'adminDispatch',
      excerpt: `${segment}: ${routes.slice(0, 3).join(', ')}`,
    });
  }

  return {
    findings,
    explicitResourceCount: resources.size,
    requiredResourceCount: Object.keys(requiredAdminResources).length,
    adminRouteSegments: adminSegments.size,
  };
}

function scanExpressReference(matrix) {
  const result = {
    path: expressRef,
    exists: fs.existsSync(expressRef),
    missingSources: [],
  };
  if (!result.exists) return result;

  const sourceFiles = [...new Set(matrix.routes.map((route) => route.sourceFile))];
  result.missingSources = sourceFiles.filter((sourceFile) => {
    const relative = sourceFile.replace(/^backend[\\/]/, '');
    return !fs.existsSync(path.join(expressRef, relative));
  });
  return result;
}

function main() {
  const matrix = loadMatrix();
  const routeCount = matrix.routes.length;
  const expectedRoutes = matrix.summary?.totalApiRoutes ?? routeCount;
  const wsCount = matrix.webSockets?.length || 0;
  const matrixCounts = countBy(matrix.routes, (route) => route.sourceFile);
  const express = scanExpressReference(matrix);
  const handlerScan = scanGoHandlers();
  const adminCompatibility = scanAdminCompatibilityConfig(matrix);
  const findings = [...handlerScan.findings, ...adminCompatibility.findings];
  const accepted = handlerScan.accepted;
  const highRisk = findings.filter((finding) => finding.severity === 'high');
  const reviewFindings = findings.length - highRisk.length;

  const report = {
    status: highRisk.length === 0 && reviewFindings === 0 ? 'pass' : 'attention-required',
    summary: {
      routeCount,
      expectedRoutes,
      websocketEntries: wsCount,
      expressReference: express.exists ? express.path : null,
      highRiskFindings: highRisk.length,
      reviewFindings,
      acceptedReviewFindings: accepted.length,
      totalFindings: findings.length,
    },
    matrix: {
      topSources: matrixCounts.slice(0, 10),
      missingExpressReferenceSources: express.missingSources,
    },
    adminCompatibility: {
      explicitResourceCount: adminCompatibility.explicitResourceCount,
      requiredResourceCount: adminCompatibility.requiredResourceCount,
      adminRouteSegments: adminCompatibility.adminRouteSegments,
    },
    findings,
    acceptedFindings: accepted,
  };

  console.log(JSON.stringify(report, null, 2));
  if (routeCount !== expectedRoutes || wsCount !== 1 || highRisk.length > 0 || reviewFindings > 0 || express.missingSources.length > 0) {
    process.exitCode = 1;
  }
}

main();
