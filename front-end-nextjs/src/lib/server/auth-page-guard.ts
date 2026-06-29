import "server-only";

import { headers } from "next/headers";
import { redirect } from "next/navigation";
import {
  apiRequestContextFromHeaders,
  getRequestAccessToken,
} from "@/lib/api";

export async function requirePageAccessToken(returnPath: string) {
  const headerStore = await headers();
  const context = apiRequestContextFromHeaders(headerStore);
  if (getRequestAccessToken(context)) {
    return context;
  }

  redirect(`/login?next=${encodeURIComponent(normalizeReturnPath(returnPath))}`);
}

function normalizeReturnPath(returnPath: string) {
  const normalized = returnPath.trim();
  if (!normalized.startsWith("/") || normalized.startsWith("//")) {
    return "/";
  }

  return normalized;
}
