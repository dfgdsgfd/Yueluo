import type { NextRequest } from "next/server";
import { NextResponse } from "next/server";
import {
  ACCESS_TOKEN_COOKIE,
  ACCESS_TOKEN_COOKIE_FALLBACKS,
  HTTP_ACCESS_TOKEN_COOKIE,
} from "@/lib/api/core/contracts";
import { getPrivateEntryPaths } from "@/lib/private-entry-paths";

const INTERNAL_ADMIN_PATH = "/admin";
const INTERNAL_BACKEND_API_PATH = "/backend-api";
const protectedEntryPaths = ["/publish", "/user"] as const;
const accessTokenCookieNames = [
  HTTP_ACCESS_TOKEN_COOKIE,
  ACCESS_TOKEN_COOKIE,
  ...ACCESS_TOKEN_COOKIE_FALLBACKS,
] as const;

export function proxy(request: NextRequest) {
  const entries = getPrivateEntryPaths();
  const pathname = normalizeRequestPath(request.nextUrl.pathname);

  if (isProtectedPath(pathname) && !hasAccessTokenCookie(request)) {
    return redirectToLogin(request, pathname);
  }

  if (pathname === entries.admin) {
    return rewritePrivateEntry(request, INTERNAL_ADMIN_PATH);
  }
  if (pathname === entries.backendApi) {
    return rewritePrivateEntry(request, INTERNAL_BACKEND_API_PATH);
  }

  if (
    (entries.admin !== INTERNAL_ADMIN_PATH && pathname === INTERNAL_ADMIN_PATH) ||
    (entries.backendApi !== INTERNAL_BACKEND_API_PATH && pathname === INTERNAL_BACKEND_API_PATH)
  ) {
    return new NextResponse(null, {
      status: 404,
      headers: {
        "Cache-Control": "private, no-store",
      },
    });
  }

  return NextResponse.next();
}

function isProtectedPath(pathname: string) {
  return protectedEntryPaths.some(
    (entryPath) => pathname === entryPath || pathname.startsWith(`${entryPath}/`),
  );
}

function hasAccessTokenCookie(request: NextRequest) {
  return accessTokenCookieNames.some((name) =>
    Boolean(request.cookies.get(name)?.value.trim()),
  );
}

function redirectToLogin(request: NextRequest, pathname: string) {
  const destination = request.nextUrl.clone();
  destination.pathname = "/login";
  destination.search = "";
  destination.searchParams.set("next", `${pathname}${request.nextUrl.search}`);
  return NextResponse.redirect(destination);
}

function rewritePrivateEntry(request: NextRequest, destinationPath: string) {
  if (normalizeRequestPath(request.nextUrl.pathname) === destinationPath) {
    return NextResponse.next();
  }
  const destination = request.nextUrl.clone();
  destination.pathname = destinationPath;
  return NextResponse.rewrite(destination);
}

function normalizeRequestPath(pathname: string) {
  return pathname.length > 1 ? pathname.replace(/\/+$/, "") : pathname;
}

export const config = {
  matcher: ["/((?!api(?:/|$)|_next(?:/|$)|.*\\.[^/]+$).*)"],
};
