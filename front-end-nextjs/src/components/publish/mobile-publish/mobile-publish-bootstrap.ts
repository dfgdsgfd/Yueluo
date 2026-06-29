"use client";

import { getNoteCategories, getPostTags } from "@/lib/api";
import type { Category, PostTag } from "@/lib/types";

type MobilePublishBootstrapData = {
  categories: Category[];
  tags: PostTag[];
};

type MobilePublishBootstrapCacheEntry = MobilePublishBootstrapData & {
  updatedAt: number;
};

const MOBILE_PUBLISH_BOOTSTRAP_CACHE_TTL_MS = 5 * 60 * 1000;

let mobilePublishBootstrapCache: MobilePublishBootstrapCacheEntry | null = null;

let mobilePublishBootstrapRequest: Promise<MobilePublishBootstrapData> | null = null;

export function getCachedMobilePublishData(): MobilePublishBootstrapData | null {
  if (!mobilePublishBootstrapCache) {
    return null;
  }

  if (Date.now() - mobilePublishBootstrapCache.updatedAt > MOBILE_PUBLISH_BOOTSTRAP_CACHE_TTL_MS) {
    mobilePublishBootstrapCache = null;
    return null;
  }

  return {
    categories: mobilePublishBootstrapCache.categories,
    tags: mobilePublishBootstrapCache.tags,
  };
}

export async function loadMobilePublishData(options: { refresh?: boolean } = {}) {
  if (!options.refresh) {
    const cached = getCachedMobilePublishData();
    if (cached) {
      return cached;
    }
  }

  if (mobilePublishBootstrapRequest) {
    return mobilePublishBootstrapRequest;
  }

  mobilePublishBootstrapRequest = Promise.all([
    getNoteCategories().catch((): Category[] => []),
    getPostTags().catch((): PostTag[] => []),
  ])
    .then(([categories, tags]) => {
      const data = { categories, tags };
      mobilePublishBootstrapCache = {
        ...data,
        updatedAt: Date.now(),
      };
      return data;
    })
    .finally(() => {
      mobilePublishBootstrapRequest = null;
    });

  return mobilePublishBootstrapRequest;
}

export function prefetchMobilePublishData() {
  return loadMobilePublishData();
}
