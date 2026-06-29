function parseEnvLine(line) {
  const text = line.trim().replace(/^\uFEFF/, "");
  if (!text || text.startsWith("#")) {
    return null;
  }
  const normalized = text.startsWith("export ") ? text.slice("export ".length).trim() : text;
  const index = normalized.indexOf("=");
  if (index <= 0) {
    return null;
  }
  const key = normalized.slice(0, index).trim();
  if (!/^[A-Z_][A-Z0-9_]*$/.test(key)) {
    return null;
  }
  let value = normalized.slice(index + 1).trim();
  if (value.length >= 2) {
    const quote = value[0];
    if ((quote === `"` || quote === `'`) && value.at(-1) === quote) {
      value = value.slice(1, -1);
      if (quote === `"`) {
        value = value
          .replaceAll(String.raw`\n`, "\n")
          .replaceAll(String.raw`\r`, "\r")
          .replaceAll(String.raw`\t`, "\t")
          .replaceAll(String.raw`\"`, `"`)
          .replaceAll(String.raw`\\`, "\\");
      }
    }
  }
  return [key, value];
}

function loadEnvFile(filePath) {
  const values = {};
  if (!fs.existsSync(filePath)) {
    return values;
  }
  for (const line of fs.readFileSync(filePath, "utf8").split(/\r?\n/)) {
    const parsed = parseEnvLine(line);
    if (parsed) {
      values[parsed[0]] = parsed[1];
    }
  }
  return values;
}

function mergeEnvInBackendOrder() {
  const env = { ...process.env };
  const explicitFiles = [];
  for (const key of ["GIN_ENV_FILE", "ENV_FILE"]) {
    if (!process.env[key]) {
      continue;
    }
    explicitFiles.push(
      ...process.env[key]
        .split(path.delimiter)
        .map((part) => part.trim())
        .filter(Boolean),
    );
  }

  for (const filePath of [...explicitFiles, backendEnvPath, rootEnvPath]) {
    const values = loadEnvFile(filePath);
    for (const [key, value] of Object.entries(values)) {
      if (env[key] === undefined) {
        env[key] = value;
      }
    }
  }
  return env;
}

function loadFrontendEnv() {
  return {
    ...loadEnvFile(path.join(repoRoot, "front-end-nextjs", ".env")),
    ...loadEnvFile(frontendEnvLocalPath),
    ...process.env,
  };
}

function commandExists(command) {
  const probe = process.platform === "win32" ? "where.exe" : "command";
  const args = process.platform === "win32" ? [command] : ["-v", command];
  const result = spawnSync(probe, args, { encoding: "utf8", shell: process.platform !== "win32" });
  return result.status === 0;
}

function addCheck(checks, id, status, message, details = {}) {
  checks.push({ id, status, message, details });
}

function fileIncludes(filePath, pattern) {
  if (!fs.existsSync(filePath)) {
    return false;
  }
  return fs.readFileSync(filePath, "utf8").includes(pattern);
}

function fileText(filePath) {
  if (!fs.existsSync(filePath)) {
    return "";
  }
  return fs.readFileSync(filePath, "utf8");
}

function goFunctionBody(source, functionName) {
  const marker = `func ${functionName}`;
  const markerIndex = source.indexOf(marker);
  if (markerIndex < 0) {
    return "";
  }

  const openIndex = source.indexOf("{", markerIndex);
  if (openIndex < 0) {
    return "";
  }

  let depth = 0;
  for (let index = openIndex; index < source.length; index += 1) {
    const char = source[index];
    if (char === "{") {
      depth += 1;
    } else if (char === "}") {
      depth -= 1;
      if (depth === 0) {
        return source.slice(openIndex + 1, index);
      }
    }
  }

  return "";
}

function escapeRegExp(text) {
  return text.replace(/[.*+?^${}()|[\]\\]/g, String.raw`\$&`);
}

function exampleEnvValue(text, key) {
  const pattern = new RegExp(`^\\s*#?\\s*${escapeRegExp(key)}\\s*=(.*)$`);
  for (const line of text.split(/\r?\n/)) {
    const match = line.match(pattern);
    if (match) {
      return match[1].trim();
    }
  }
  return null;
}

function exampleEnvIncludesKey(text, key) {
  return exampleEnvValue(text, key) !== null;
}

function isTruthyEnvValue(value) {
  return ["1", "true", "yes", "on"].includes(value.trim().toLowerCase());
}

function positiveIntegerFromUnknown(value, fallback) {
  const numeric = Number(value);
  return Number.isInteger(numeric) && numeric > 0 ? numeric : fallback;
}

function nonNegativeIntegerFromUnknown(value, fallback) {
  const numeric = Number(value);
  return Number.isInteger(numeric) && numeric >= 0 ? numeric : fallback;
}

function delay(ms) {
  return new Promise((resolve) => {
    setTimeout(resolve, ms);
  });
}

function sanitizeTarget(target) {
  if (!target) {
    return null;
  }
  return {
    driver: target.driver,
    host: target.host,
    port: target.port,
    database: target.database,
  };
}

function isRecord(value) {
  return Boolean(value) && typeof value === "object" && !Array.isArray(value);
}

function databaseTarget(env) {
  const databaseURL = env.DATABASE_URL?.trim();
  if (databaseURL) {
    try {
      const parsed = new URL(databaseURL);
      const driver = parsed.protocol.replace(/:$/, "");
      const defaultPort = driver.startsWith("postgres") ? 5432 : 3306;
      return {
        driver,
        host: parsed.hostname,
        port: Number(parsed.port || defaultPort),
        database: parsed.pathname.replace(/^\//, "") || undefined,
      };
    } catch {
      return { error: "DATABASE_URL parse failed" };
    }
  }

  const host = env.DB_HOST?.trim();
  const user = env.DB_USER?.trim();
  const database = env.DB_NAME?.trim();
  if (!host || !user || !database) {
    return null;
  }
  const driver = (env.DB_DRIVER || env.DATABASE_DRIVER || "mysql").trim();
  return {
    driver,
    host,
    port: Number(env.DB_PORT || (driver.startsWith("postgres") ? 5432 : 3306)),
    database,
  };
}

function redisTarget(env) {
  const addr = env.REDIS_ADDR?.trim();
  if (addr) {
    const index = addr.lastIndexOf(":");
    if (index > 0) {
      return {
        host: addr.slice(0, index),
        port: Number(addr.slice(index + 1)),
      };
    }
  }
  const host = env.REDIS_HOST?.trim();
  if (!host) {
    return null;
  }
  return {
    host,
    port: Number(env.REDIS_PORT || 6379),
  };
}

async function resolveHost(host) {
  try {
    const result = await dns.lookup(host);
    return { ok: true, address: result.address };
  } catch (error) {
    return { ok: false, error: error.code || error.message };
  }
}

function connectTCP(host, port, timeoutMs = 1500) {
  return new Promise((resolve) => {
    const socket = new net.Socket();
    let settled = false;
    const finish = (result) => {
      if (settled) {
        return;
      }
      settled = true;
      socket.destroy();
      resolve(result);
    };

    socket.setTimeout(timeoutMs);
    socket.once("connect", () => finish({ ok: true }));
    socket.once("timeout", () => finish({ ok: false, error: "timeout" }));
    socket.once("error", (error) => finish({ ok: false, error: error.code || error.message }));
    socket.connect(port, host);
  });
}

async function checkNetworkTarget(checks, id, label, target) {
  if (!target || target.error) {
    addCheck(checks, id, "fail", `${label} is not configured`, { error: target?.error });
    return;
  }
  if (!target.host || !target.port || Number.isNaN(target.port)) {
    addCheck(checks, id, "fail", `${label} host or port is missing`, sanitizeTarget(target));
    return;
  }

  const dnsResult = await resolveHost(target.host);
  if (!dnsResult.ok) {
    addCheck(checks, id, "fail", `${label} host does not resolve`, {
      ...sanitizeTarget(target),
      error: dnsResult.error,
    });
    return;
  }

  const tcpResult = await connectTCP(target.host, target.port);
  addCheck(
    checks,
    id,
    tcpResult.ok ? "pass" : "fail",
    tcpResult.ok ? `${label} TCP endpoint is reachable` : `${label} TCP endpoint is not reachable`,
    {
      ...sanitizeTarget(target),
      resolvedAddress: dnsResult.address,
      error: tcpResult.error,
    },
  );
}

async function checkHttpHealth(checks, id, label, url) {
  try {
    const response = await fetchWithRetry(url, {}, { retries: httpRetryCount });
    addCheck(
      checks,
      id,
      response.ok ? "pass" : "fail",
      response.ok ? `${label} health endpoint is reachable` : `${label} health endpoint returned non-2xx`,
      { url, status: response.status, retries: httpRetryCount },
    );
  } catch (error) {
    addCheck(checks, id, "fail", `${label} health endpoint is not reachable`, {
      url,
      error: fetchErrorMessage(error),
      timeoutMs: httpTimeoutMs,
      retries: httpRetryCount,
    });
  }
}

function fetchErrorMessage(error) {
  if (error.name === "AbortError") {
    return "timeout";
  }

  return error.cause?.code || error.code || error.cause?.message || error.message;
}

async function fetchWithTimeout(url, options = {}, timeoutMs = httpTimeoutMs) {
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), timeoutMs);

  try {
    return await fetch(url, {
      ...options,
      signal: controller.signal,
    });
  } finally {
    clearTimeout(timer);
  }
}

async function fetchWithRetry(url, options = {}, { retries = 0, timeoutMs = httpTimeoutMs } = {}) {
  let lastError;

  for (let attempt = 0; attempt <= retries; attempt += 1) {
    try {
      return await fetchWithTimeout(url, options, timeoutMs);
    } catch (error) {
      lastError = error;
      if (attempt >= retries) {
        break;
      }
      await delay(Math.min(250 * (attempt + 1), 1000));
    }
  }

  throw lastError;
}

async function fetchJsonWithTimeout(url, timeoutMs = httpTimeoutMs) {
  const response = await fetchWithRetry(url, {}, { retries: httpRetryCount, timeoutMs });
  const text = await response.text();
  let payload = null;

  try {
    payload = text ? JSON.parse(text) : null;
  } catch (error) {
    return {
      ok: false,
      status: response.status,
      parseError: error.message,
      textSnippet: text.slice(0, 160),
    };
  }

  return {
    ok: response.ok,
    status: response.status,
    payload,
  };
}

async function requestJsonWithTimeout(url, options = {}, timeoutMs = httpTimeoutMs) {
  const requestHeaders = {
    ...(options.body && !options.headers?.["content-type"]
      ? { "content-type": "application/json" }
      : {}),
    ...(options.headers ?? {}),
  };
  const method = String(options.method ?? "GET").toUpperCase();
  const retries = method === "GET" || method === "HEAD" ? httpRetryCount : 0;
  const response = await fetchWithRetry(
    url,
    {
      ...options,
      headers: requestHeaders,
    },
    { retries, timeoutMs },
  );
  const text = await response.text();
  let payload = null;

  try {
    payload = text ? JSON.parse(text) : null;
  } catch (error) {
    return {
      status: response.status,
      ok: false,
      parseError: error.message,
      textSnippet: text.slice(0, 160),
    };
  }

  return {
    ok: response.ok,
    status: response.status,
    payload,
  };
}

async function fetchManualRedirectWithTimeout(url, timeoutMs = httpTimeoutMs) {
  return fetchWithRetry(
    url,
    {
      redirect: "manual",
    },
    { retries: httpRetryCount, timeoutMs },
  );
}

async function checkBackendHealth(checks, env) {
  const port = env.GIN_PORT || env.PORT || "3001";
  await checkHttpHealth(checks, "backend-health", "backend", `http://localhost:${port}/api/health`);
}

function healthUrlFromOrigin(origin) {
  try {
    return new URL("/api/health", origin).toString();
  } catch {
    return null;
  }
}

function authConfigUrlFromOrigin(origin) {
  try {
    return new URL("/api/auth/auth-config", origin).toString();
  } catch {
    return null;
  }
}

function apiUrlFromOrigin(origin, pathAndQuery) {
  try {
    return new URL(pathAndQuery, origin).toString();
  } catch {
    return null;
  }
}

function unwrapAuthConfigPayload(payload) {
  if (!isRecord(payload)) {
    return null;
  }

  if ("code" in payload && payload.code !== 200) {
    return null;
  }

  if ("success" in payload && payload.success === false) {
    return null;
  }

  if ("data" in payload) {
    return isRecord(payload.data) ? payload.data : null;
  }

  return payload;
}

function unwrapApiPayload(payload) {
  if (!isRecord(payload)) {
    return payload;
  }

  if ("data" in payload) {
    return payload.data;
  }

  return payload;
}

function extractAccessToken(payload) {
  const data = unwrapApiPayload(payload);
  if (!isRecord(data)) {
    return "";
  }

  const tokenSource = isRecord(data.tokens) ? data.tokens : data;
  const token = tokenSource.access_token ?? tokenSource.token ?? tokenSource.accessToken;
  return typeof token === "string" ? token : "";
}

function isUnauthorizedApiResult(result) {
  if (result.status === 401) {
    return true;
  }

  return isRecord(result.payload) && result.payload.code === 401;
}

function isGuestAuthRejection(result) {
  if (result.status === 401 || result.status === 403) {
    return true;
  }

  return isRecord(result.payload) && (result.payload.code === 401 || result.payload.code === 403);
}

function validateFeedPayload(payload) {
  const data = unwrapApiPayload(payload);
  const problems = [];

  if (!isRecord(data)) {
    problems.push("feed data is not an object");
    return problems;
  }
  if (!Array.isArray(data.posts)) {
    problems.push("feed data.posts is not an array");
  }
  if (!isRecord(data.pagination)) {
    problems.push("feed data.pagination is not an object");
  } else {
    for (const field of ["page"]) {
      if (typeof data.pagination[field] !== "number") {
        problems.push(`feed pagination.${field} is not a number`);
      }
    }
  }

  return problems;
}

