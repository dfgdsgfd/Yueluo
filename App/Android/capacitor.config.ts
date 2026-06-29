import type { CapacitorConfig } from "@capacitor/cli";
import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";

const envFromFiles = loadEnvFiles([".env", ".env.local"]);
const defaultMobileServerUrl = "https://xse.yuelk.com";
const serverUrl =
  firstEnvValue([
    "CAP_SERVER_URL",
    "NEXT_PUBLIC_YUEM_MOBILE_SERVER_URL",
    "NEXT_PUBLIC_API_BASE_URL",
  ]) ?? defaultMobileServerUrl;
const appVersion =
  firstEnvValue([
    "NEXT_PUBLIC_YUEM_MOBILE_APP_VERSION",
    "YUEM_MOBILE_APP_VERSION",
  ]) ?? "1.0.0";
const isCleartextServer = serverUrl.startsWith("http://");
const navigationHosts = uniqueValues([
  "localhost",
  "127.0.0.1",
  "10.0.2.2",
  safeHostname(serverUrl),
  ...envHostList("NEXT_PUBLIC_YUEM_IN_APP_HOSTS"),
]);

const config: CapacitorConfig = {
  appId: "com.yuelk.xsewebfast",
  appName: "月梦-快速版",
  webDir: "www",
  appendUserAgent: ` YuemAndroid/${appVersion} /XseWebApp${appVersion}`,
  backgroundColor: "#121212",
  loggingBehavior: "none",
  zoomEnabled: false,
  android: {
    allowMixedContent: isCleartextServer,
    backgroundColor: "#121212",
    captureInput: true,
    webContentsDebuggingEnabled: false,
  },
  server: {
    url: serverUrl,
    cleartext: isCleartextServer,
    allowNavigation: navigationHosts,
    errorPath: "offline.html",
  },
  plugins: {
    SplashScreen: {
      launchAutoHide: true,
      launchFadeOutDuration: 240,
      launchShowDuration: 900,
      backgroundColor: "#121212",
      androidScaleType: "CENTER_CROP",
      showSpinner: false,
      splashFullScreen: true,
      splashImmersive: false,
    },
    StatusBar: {
      overlaysWebView: false,
      style: "DARK",
      backgroundColor: "#121212",
    },
    Keyboard: {
      resize: "native",
      resizeOnFullScreen: true,
    },
  },
};

export default config;

function loadEnvFiles(files: string[]) {
  const values: Record<string, string> = {};
  for (const file of files) {
    const path = resolve(process.cwd(), file);
    if (!existsSync(path)) continue;
    for (const line of readFileSync(path, "utf8").split(/\r?\n/u)) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith("#")) continue;
      const separator = trimmed.indexOf("=");
      if (separator < 0) continue;
      const key = trimmed.slice(0, separator).trim();
      const value = trimmed
        .slice(separator + 1)
        .trim()
        .replace(/^['"]|['"]$/gu, "");
      values[key] = value;
    }
  }
  return values;
}

function firstEnvValue(names: string[]) {
  for (const name of names) {
    const value = (process.env[name] ?? envFromFiles[name] ?? "").trim();
    if (value) return value.replace(/\/$/u, "");
  }
  return undefined;
}

function safeHostname(rawUrl: string) {
  try {
    return new URL(rawUrl).hostname;
  } catch {
    return undefined;
  }
}

function envHostList(name: string) {
  return (process.env[name] ?? envFromFiles[name] ?? "")
    .split(",")
    .map((value) => value.trim())
    .filter(Boolean)
    .map(
      (value) =>
        safeHostname(value.includes("://") ? value : `https://${value}`) ??
        value.replace(/:\d+$/u, ""),
    );
}

function uniqueValues(values: Array<string | undefined>) {
  return [...new Set(values.filter((value): value is string => Boolean(value)))];
}
