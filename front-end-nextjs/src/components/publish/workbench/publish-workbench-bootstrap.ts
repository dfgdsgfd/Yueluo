"use client";

import { getDraftPosts, getHotCategories } from "@/lib/api";
import type { Category, FeedPost } from "@/lib/types";

type PublishWorkbenchBootstrapData = {
  categories: Category[];
  drafts: FeedPost[];
};

type PublishWorkbenchBootstrapCacheEntry = PublishWorkbenchBootstrapData & {
  updatedAt: number;
};

const PUBLISH_WORKBENCH_BOOTSTRAP_CACHE_TTL_MS = 60 * 1000;

let publishWorkbenchBootstrapCache: PublishWorkbenchBootstrapCacheEntry | null = null;

let publishWorkbenchBootstrapRequest: Promise<PublishWorkbenchBootstrapData> | null = null;

export function getCachedPublishWorkbenchData(): PublishWorkbenchBootstrapData | null {
  if (!publishWorkbenchBootstrapCache) {
    return null;
  }

  if (
    Date.now() - publishWorkbenchBootstrapCache.updatedAt >
    PUBLISH_WORKBENCH_BOOTSTRAP_CACHE_TTL_MS
  ) {
    publishWorkbenchBootstrapCache = null;
    return null;
  }

  return {
    categories: publishWorkbenchBootstrapCache.categories,
    drafts: publishWorkbenchBootstrapCache.drafts,
  };
}

export async function loadPublishWorkbenchData(options: { refresh?: boolean } = {}) {
  if (!options.refresh) {
    const cached = getCachedPublishWorkbenchData();
    if (cached) {
      return cached;
    }
  }

  if (publishWorkbenchBootstrapRequest) {
    return publishWorkbenchBootstrapRequest;
  }

  publishWorkbenchBootstrapRequest = Promise.all([
    getHotCategories(20).catch((): Category[] => []),
    getDraftPosts({ limit: 20 }).then((payload) => payload.posts).catch((): FeedPost[] => []),
  ])
    .then(([categories, drafts]) => {
      const data = { categories, drafts };
      publishWorkbenchBootstrapCache = {
        ...data,
        updatedAt: Date.now(),
      };
      return data;
    })
    .finally(() => {
      publishWorkbenchBootstrapRequest = null;
    });

  return publishWorkbenchBootstrapRequest;
}

export function prefetchPublishWorkbenchData() {
  return loadPublishWorkbenchData();
}
