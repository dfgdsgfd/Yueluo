const DEFAULT_ADMIN_ENTRY_PATH = "/admin";
const DEFAULT_BACKEND_API_ENTRY_PATH = "/backend-api";

export type PrivateEntryPaths = {
  admin: string;
  backendApi: string;
};

export function getPrivateEntryPaths(): PrivateEntryPaths {
  const admin = normalizePrivateEntryPath(
    process.env.ADMIN_ENTRY_PATH,
    DEFAULT_ADMIN_ENTRY_PATH,
    "ADMIN_ENTRY_PATH",
  );
  const backendApi = normalizePrivateEntryPath(
    process.env.BACKEND_API_ENTRY_PATH,
    DEFAULT_BACKEND_API_ENTRY_PATH,
    "BACKEND_API_ENTRY_PATH",
  );

  if (admin === backendApi) {
    throw new Error("ADMIN_ENTRY_PATH and BACKEND_API_ENTRY_PATH must be different.");
  }
  if (admin === DEFAULT_BACKEND_API_ENTRY_PATH) {
    throw new Error("ADMIN_ENTRY_PATH cannot use the reserved /backend-api path.");
  }
  if (backendApi === DEFAULT_ADMIN_ENTRY_PATH) {
    throw new Error("BACKEND_API_ENTRY_PATH cannot use the reserved /admin path.");
  }

  return { admin, backendApi };
}

function normalizePrivateEntryPath(
  value: string | undefined,
  fallback: string,
  envName: string,
) {
  const trimmed = value?.trim();
  if (!trimmed) {
    return fallback;
  }

  const normalized = trimmed.length > 1 ? trimmed.replace(/\/+$/, "") : trimmed;
  if (
    normalized === "/" ||
    !/^\/[A-Za-z0-9/_-]+$/.test(normalized) ||
    normalized.includes("//") ||
    normalized.split("/").some((segment) => segment === "." || segment === "..") ||
    normalized === "/api" ||
    normalized.startsWith("/api/") ||
    normalized === "/_next" ||
    normalized.startsWith("/_next/")
  ) {
    throw new Error(
      `${envName} must be a private application path such as /ops-7f3a and cannot use /api or /_next.`,
    );
  }
  return normalized;
}
