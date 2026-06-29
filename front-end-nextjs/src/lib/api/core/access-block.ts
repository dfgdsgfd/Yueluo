import { ApiError } from "./contracts";

const ACCESS_BLOCK_HEADER = "X-Access-Block";
const ACCESS_BLOCK_RULE_ID_HEADER = "X-Access-Block-Rule-ID";
const ACCESS_BLOCK_ACTION_HEADER = "X-Access-Block-Action";
const ACCESS_BLOCK_STATUS_CODE_HEADER = "X-Access-Block-Status-Code";

export type AccessBlockErrorDetails = {
  accessBlocked: true;
  action: "status" | "redirect" | string;
  redirectUrl: string | null;
  requestId: string | null;
  ruleId: string | null;
  statusCode: number;
};

export function accessBlockDetailsFromResponse(response: Response): AccessBlockErrorDetails | null {
  const marked = response.headers.get(ACCESS_BLOCK_HEADER) === "1";
  const implicitStatusBlock = response.status === 444;
  const opaqueRedirect = response.type === "opaqueredirect";
  if (!marked && !implicitStatusBlock && !opaqueRedirect) {
    return null;
  }

  const action = response.headers.get(ACCESS_BLOCK_ACTION_HEADER) ??
    (opaqueRedirect || isRedirectStatus(response.status) ? "redirect" : "status");
  const statusCode = numberFromHeader(response.headers.get(ACCESS_BLOCK_STATUS_CODE_HEADER), response.status);

  return {
    accessBlocked: true,
    action,
    redirectUrl: response.headers.get("Location"),
    requestId: response.headers.get("X-Request-ID"),
    ruleId: response.headers.get(ACCESS_BLOCK_RULE_ID_HEADER),
    statusCode,
  };
}

export function throwAccessBlockResponse(response: Response, requestUrl?: string) {
  let details = accessBlockDetailsFromResponse(response);
  if (!details) {
    return;
  }
  const fallbackRedirectUrl =
    details.action === "redirect" ? requestUrl || response.url || null : null;
  if (!details.redirectUrl && fallbackRedirectUrl) {
    details = { ...details, redirectUrl: fallbackRedirectUrl };
  }
  if (details.action === "redirect" && details.redirectUrl && typeof window !== "undefined") {
    window.location.assign(details.redirectUrl);
  }
  throw new ApiError("error.access_blocked", {
    status: response.status || details.statusCode,
    details,
  });
}

export function isAccessBlockApiError(error: unknown) {
  if (!(error instanceof ApiError)) {
    return false;
  }
  const details = error.details;
  return Boolean(
    details &&
      typeof details === "object" &&
      !Array.isArray(details) &&
      (details as { accessBlocked?: unknown }).accessBlocked === true,
  );
}

function numberFromHeader(value: string | null, fallback: number) {
  const numberValue = Number(value);
  return Number.isFinite(numberValue) && numberValue > 0 ? numberValue : fallback;
}

function isRedirectStatus(status: number) {
  return status >= 300 && status < 400;
}
