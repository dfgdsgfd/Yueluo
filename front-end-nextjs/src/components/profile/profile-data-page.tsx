"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ArrowLeft, Bell, MoreHorizontal } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { ApiError, getUserProfileData, getViewerProfileData } from "@/lib/api";
import type { UserProfilePayload } from "@/lib/types";
import { ProfileNotFound, ProfilePage } from "./profile-page";

type ProfileDataVariant = "viewer" | "user";

type ProfileCacheEntry = {
  payload: UserProfilePayload;
  updatedAt: number;
};

const PROFILE_CACHE_TTL_MS = 5 * 60 * 1000;
const profilePayloadCache = new Map<string, ProfileCacheEntry>();
const profileRequestCache = new Map<string, Promise<UserProfilePayload>>();

export function ViewerProfileDataPage({
  initialPayload,
  onBack,
}: {
  initialPayload?: UserProfilePayload | null;
  onBack?: () => void;
} = {}) {
  return (
    <ProfileDataPage
      key="viewer"
      variant="viewer"
      initialPayload={initialPayload}
      onBack={onBack}
    />
  );
}

export function UserProfileDataPage({
  initialPayload,
  onBack,
  userId,
}: {
  initialPayload?: UserProfilePayload | null;
  onBack?: () => void;
  userId: string;
}) {
  return (
    <ProfileDataPage
      key={getProfileCacheKey("user", userId)}
      initialPayload={initialPayload}
      userId={userId}
      variant="user"
      onBack={onBack}
    />
  );
}

export function prefetchViewerProfileData() {
  return loadCachedProfileData("viewer");
}

function ProfileDataPage({
  initialPayload,
  onBack,
  userId,
  variant,
}: {
  initialPayload?: UserProfilePayload | null;
  onBack?: () => void;
  userId?: string;
  variant: ProfileDataVariant;
}) {
  const cacheKey = getProfileCacheKey(variant, userId);
  const [payload, setPayload] = useState<UserProfilePayload | null>(() => {
    if (initialPayload) {
      setCachedProfilePayload(cacheKey, initialPayload);
      return initialPayload;
    }
    return getCachedProfilePayload(cacheKey);
  });
  const [payloadRevision, setPayloadRevision] = useState(0);
  const [error, setError] = useState<string | null>(null);
  const [notFound, setNotFound] = useState(false);

  useEffect(() => {
    if (initialPayload) {
      return;
    }
    let mounted = true;
    const nextCacheKey = getProfileCacheKey(variant, userId);

    async function loadProfile() {
      try {
        setError(null);
        setNotFound(false);
        const nextPayload = await loadCachedProfileData(variant, userId);

        if (mounted) {
          setPayload(nextPayload);
          setPayloadRevision((revision) => revision + 1);
        }
      } catch (loadError) {
        if (!mounted) {
          return;
        }

        if (
          variant === "user" &&
          loadError instanceof ApiError &&
          loadError.status === 404
        ) {
          profilePayloadCache.delete(nextCacheKey);
          setPayload(null);
          setNotFound(true);
          return;
        }

        const stalePayload = getCachedProfilePayload(nextCacheKey);
        if (stalePayload) {
          setPayload(stalePayload);
          setPayloadRevision((revision) => revision + 1);
          return;
        }

        const message =
          loadError instanceof Error ? loadError.message : "用户资料加载失败";
        setError(message);
        toast.error(message);
      }
    }

    void loadProfile();

    return () => {
      mounted = false;
    };
  }, [initialPayload, userId, variant]);

  if (notFound) {
    return <ProfileNotFound userId={userId ?? ""} onBack={onBack} />;
  }

  if (error) {
    return (
      <div className="theme-adaptive flex min-h-screen items-center justify-center bg-[#121212] px-6 text-white">
        <div className="w-full max-w-[420px] text-center">
          <h1 className="text-2xl font-bold">资料加载失败</h1>
          <p className="mt-3 text-sm leading-6 text-white/52">{error}</p>
          {onBack ? (
            <Button type="button" onClick={onBack} className="mt-6 h-10 px-5">
              返回信息流
            </Button>
          ) : (
            <Button asChild className="mt-6 h-10 px-5">
              <Link href="/">返回信息流</Link>
            </Button>
          )}
        </div>
      </div>
    );
  }

  if (!payload) {
    return <ProfilePageSkeleton onBack={onBack} />;
  }

  return (
    <ProfilePage
      key={`${cacheKey}:${payloadRevision}`}
      profile={payload.profile}
      tabs={payload.tabs}
      videoCenter={payload.videoCenter}
      variant={variant}
      onBack={onBack}
    />
  );
}

function getProfileCacheKey(variant: ProfileDataVariant, userId?: string) {
  return variant === "viewer" ? "viewer" : `user:${userId ?? ""}`;
}

function getCachedProfilePayload(cacheKey: string) {
  const cached = profilePayloadCache.get(cacheKey);
  if (!cached) {
    return null;
  }

  if (Date.now() - cached.updatedAt > PROFILE_CACHE_TTL_MS) {
    profilePayloadCache.delete(cacheKey);
    return null;
  }

  return cached.payload;
}

function setCachedProfilePayload(cacheKey: string, payload: UserProfilePayload) {
  profilePayloadCache.set(cacheKey, {
    payload,
    updatedAt: Date.now(),
  });
}

function loadCachedProfileData(variant: ProfileDataVariant, userId?: string) {
  const cacheKey = getProfileCacheKey(variant, userId);
  const inFlightRequest = profileRequestCache.get(cacheKey);
  if (inFlightRequest) {
    return inFlightRequest;
  }

  const request = (
    variant === "viewer" ? getViewerProfileData() : getUserProfileData(userId ?? "")
  )
    .then((nextPayload) => {
      setCachedProfilePayload(cacheKey, nextPayload);
      return nextPayload;
    })
    .finally(() => {
      profileRequestCache.delete(cacheKey);
    });

  profileRequestCache.set(cacheKey, request);
  return request;
}

function ProfilePageSkeleton({ onBack }: { onBack?: () => void }) {
  return (
    <div className="theme-adaptive min-h-screen bg-[#121212] text-[#e0e0e0]" aria-busy="true">
      <header className="fixed inset-x-0 top-0 z-40 h-14 border-b border-white/[0.07] bg-[#121212]/96 backdrop-blur lg:hidden">
        <div className="flex h-full items-center gap-2 px-3">
          {onBack ? (
            <Button
              type="button"
              onClick={onBack}
              variant="ghost"
              size="icon"
              aria-label="返回信息流"
              className="size-10 text-white hover:bg-white/[0.06]"
            >
              <ArrowLeft className="size-5" />
            </Button>
          ) : (
            <Button
              asChild
              variant="ghost"
              size="icon"
              aria-label="返回信息流"
              className="size-10 text-white hover:bg-white/[0.06]"
            >
              <Link href="/">
                <ArrowLeft className="size-5" />
              </Link>
            </Button>
          )}
          <div className="h-4 w-20 rounded bg-white/[0.12]" />
          <div className="ml-auto flex items-center gap-1">
            <span className="flex size-10 items-center justify-center text-white/55">
              <Bell className="size-5" />
            </span>
            <span className="flex size-10 items-center justify-center text-white/55">
              <MoreHorizontal className="size-5" />
            </span>
          </div>
        </div>
      </header>

      <main className="mx-auto min-h-screen w-full max-w-[1120px] px-4 pb-24 pt-[72px] sm:px-6 lg:px-8 lg:pb-14 lg:pt-10">
        <section className="overflow-hidden rounded-2xl border border-white/[0.08] bg-[#181818] shadow-[0_18px_50px_rgba(0,0,0,0.2)]">
          <div className="h-[168px] animate-pulse bg-gradient-to-b from-[#242428] to-[#1b1b1f] sm:h-[220px]" />
          <div className="px-5 pb-6 sm:px-8 sm:pb-8">
            <div className="-mt-12 flex items-end gap-4">
              <div className="size-24 shrink-0 animate-pulse rounded-full border-[3px] border-[#181818] bg-white/[0.1] sm:size-28" />
              <div className="min-w-0 flex-1 pb-2">
                <div className="h-6 w-28 animate-pulse rounded bg-white/[0.12]" />
                <div className="mt-3 h-4 w-36 animate-pulse rounded bg-white/[0.08]" />
              </div>
            </div>

            <div className="mt-5 flex gap-2">
              <div className="h-10 w-28 animate-pulse rounded-full bg-white/[0.12]" />
              <div className="h-10 w-20 animate-pulse rounded-full bg-white/[0.08]" />
              <div className="size-10 animate-pulse rounded-full bg-white/[0.08]" />
            </div>

            <div className="mt-5 space-y-3">
              <div className="h-4 w-4/5 animate-pulse rounded bg-white/[0.08]" />
              <div className="h-4 w-2/5 animate-pulse rounded bg-white/[0.06]" />
            </div>

            <div className="mt-5 flex gap-2">
              <div className="h-7 w-20 animate-pulse rounded-full bg-white/[0.07]" />
              <div className="h-7 w-28 animate-pulse rounded-full bg-white/[0.07]" />
            </div>

            <div className="mt-5 grid max-w-[520px] grid-cols-3 rounded-2xl border border-white/[0.07] bg-white/[0.035]">
              {[0, 1, 2].map((item) => (
                <div key={item} className="px-2 py-4 sm:px-3">
                  <div className="mx-auto h-5 w-10 animate-pulse rounded bg-white/[0.1]" />
                  <div className="mx-auto mt-2 h-3 w-12 animate-pulse rounded bg-white/[0.06]" />
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="mt-4 lg:mt-6">
          <div className="-mx-4 border-b border-white/[0.07] px-4 sm:-mx-6 sm:px-6 lg:-mx-8 lg:px-8">
            <div className="mx-auto flex h-[52px] max-w-[680px] items-center justify-center gap-3">
              {[0, 1, 2].map((item) => (
                <div key={item} className="h-4 flex-1 animate-pulse rounded bg-white/[0.07]" />
              ))}
            </div>
          </div>
          <div className="grid grid-cols-2 gap-3 pt-5 sm:grid-cols-3 lg:grid-cols-4">
            {[0, 1, 2, 3].map((item) => (
              <div
                key={item}
                className="aspect-[3/4] animate-pulse rounded-lg bg-white/[0.06]"
              />
            ))}
          </div>
        </section>
      </main>
    </div>
  );
}
