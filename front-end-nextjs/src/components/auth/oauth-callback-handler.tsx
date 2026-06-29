"use client";

import { useEffect, useRef } from "react";
import { useRouter } from "next/navigation";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import {
  getCurrentUser,
  getStoredAccessToken,
  getStoredRefreshToken,
  storeAuthenticatedUser,
  storeOAuthCallbackTokens,
} from "@/lib/api";

const oauthCallbackParams = [
  "oauth2_login",
  "access_token",
  "refresh_token",
  "is_new_user",
  "error",
  "message",
];

const bootstrapGlobal = "__YUEM_OAUTH_CALLBACK__";

export type OAuthCallbackTokens = {
  accessToken: string | null;
  refreshToken: string | null;
};

declare global {
  interface Window {
    [bootstrapGlobal]?: OAuthCallbackTokens;
  }
}

export function OAuthCallbackHandler() {
  const t = useTranslations("login");
  const router = useRouter();
  const handledRef = useRef(false);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const bootstrappedTokens = window[bootstrapGlobal] ?? null;
    const isOAuthCallback =
      params.get("oauth2_login") === "success" || Boolean(bootstrappedTokens);

    if (!isOAuthCallback || handledRef.current) {
      return;
    }
    handledRef.current = true;

    const accessToken =
      bootstrappedTokens?.accessToken ??
      params.get("access_token") ??
      getStoredAccessToken();
    const refreshToken =
      bootstrappedTokens?.refreshToken ??
      params.get("refresh_token") ??
      getStoredRefreshToken();
    removeOAuthParams(params);
    window[bootstrapGlobal] = undefined;

    if (accessToken) {
      storeOAuthCallbackTokens({ accessToken, refreshToken });
    }

    window.history.replaceState(null, "", cleanUrl(params));

    getCurrentUser(accessToken ? { token: accessToken } : {}, { auth: true })
      .then((user) => {
        if (accessToken) {
          storeOAuthCallbackTokens({ accessToken, refreshToken, user });
        } else {
          storeAuthenticatedUser(user);
        }
        toast.success(t("oauthSuccess"));
        router.replace("/");
      })
      .catch(() => {
        if (accessToken) {
          toast.success(t("oauthSuccess"));
          router.replace("/");
          return;
        }
        toast.error(t("oauthStatusFailed"));
        router.replace("/login");
      });
  }, [router, t]);

  return null;
}

function removeOAuthParams(params: URLSearchParams) {
  for (const key of oauthCallbackParams) {
    params.delete(key);
  }
}

function cleanUrl(params: URLSearchParams) {
  const query = params.toString();
  return `${window.location.pathname}${query ? `?${query}` : ""}${window.location.hash}`;
}
