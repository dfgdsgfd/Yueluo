import {
  fixtureInitialFeedData
} from "../fixtures";
import type {
  BackendComment,
  Category,
  CommentsPayload,
  FeedMode,
  FeedPayload,
  FeedPost,
  FeedQuery,
  InitialFeedData,
  AppDownloadConfig,
  PostImageArchiveJob,
  ProtectedPackageJob,
  ProtectedPackageEvent,
  PublishPostInput,
  SearchFeedQuery,
  SearchPayload,
  PostTag,
  PostPurchaseUsersPayload,
  UserToolbarItem
} from "../types";
import {
  ApiRequestContext,
  ApiUnauthorizedError,
  apiDelete,
  apiDownload,
  apiGet,
  apiPost,
  apiPut,
  applyAuthorizationHeader,
  buildApiUrl,
  getStoredAccessToken,
  getRequestAccessToken,
  isRecord,
  normalizeAdminPagination,
  numberFromUnknown,
  truthyEnvValues
} from "./core";
import { isAccessBlockApiError } from "./core/access-block";
import {
  getAuthConfig,
  getCurrentUser
} from "./auth";
import {
  normalizeVideoCenterConfig,
  videoCenterConfigFromAuthConfig
} from "../video-center";
import { normalizeSiteProfile } from "../seo";

export async function getFeedPage(
  mode: FeedMode,
  context: ApiRequestContext = {},
  query: FeedQuery = {},
): Promise<FeedPayload> {
  const endpointByMode = {
    recommended: "/api/posts/recommended",
    hot: "/api/posts/hot",
    videoCenter: "/api/posts/video-center",
    following: "/api/posts/following",
  } satisfies Record<FeedMode, string>;
  const page = query.page ?? 1;
  const limit = query.limit ?? (mode === "following" ? 20 : 24);
  const categoryId = query.category_id ?? null;
  const shouldUseCategoryList = mode === "recommended" && categoryId !== null;

  const payload = await apiGet<unknown>(
    shouldUseCategoryList ? "/api/posts" : endpointByMode[mode],
    {
      page,
      limit,
      category: shouldUseCategoryList ? categoryId : undefined,
      category_id: shouldUseCategoryList ? undefined : categoryId,
      sort: query.sort ?? (mode === "following" ? "time" : undefined),
    },
    {
      context,
      auth: mode === "following" || shouldAuthenticateFeedRequest(context),
      signal: context.signal,
    },
  );

  return normalizeFeedPayload(payload, { page, limit });
}


export function normalizeFeedPayload(
  payload: unknown,
  fallback: { page: number; limit: number },
): FeedPayload {
  const record = isRecord(payload) ? payload : {};
  const nestedPosts = isRecord(record.posts) ? record.posts : null;
  const posts = Array.isArray(record.posts)
    ? record.posts
    : Array.isArray(nestedPosts?.data)
      ? nestedPosts.data
      : Array.isArray(record.data)
        ? record.data
        : [];
  const paginationSource = record.pagination ?? nestedPosts?.pagination ?? record;

  return {
    posts: posts as FeedPost[],
    pagination: normalizeAdminPagination(paginationSource, {
      ...fallback,
      itemCount: posts.length,
    }),
  };
}


export function normalizeSearchFeedPayload(
  payload: SearchPayload,
  fallback: { page: number; limit: number },
): FeedPayload {
  const group = payload.posts;
  const posts = group?.data ?? payload.data ?? [];
  const paginationSource = group?.pagination ?? payload.pagination;

  return {
    posts,
    pagination: normalizeAdminPagination(paginationSource, {
      ...fallback,
      itemCount: posts.length,
    }),
  };
}


export async function searchFeed(
  query: SearchFeedQuery,
  context: ApiRequestContext = {},
): Promise<FeedPayload> {
  const keyword = query.keyword?.trim();
  const tag = query.tag?.trim();
  const page = query.page ?? 1;
  const limit = query.limit ?? 24;

  const payload = await apiGet<SearchPayload>(
    "/api/search",
    {
      keyword,
      tag,
      type: query.type ?? "all",
      page,
      limit,
    },
    {
      context,
      auth: shouldAuthenticateFeedRequest(context),
      signal: context.signal,
    },
  );

  return normalizeSearchFeedPayload(payload, { page, limit });
}


export function hasNextFeedPage(pagination: FeedPayload["pagination"]) {
  if (typeof pagination.hasNextPage === "boolean") {
    return pagination.hasNextPage;
  }

  if (pagination.pages !== undefined) {
    return pagination.page < pagination.pages;
  }

  if (pagination.total !== undefined) {
    const limit = pagination.limit ?? pagination.pageSize ?? 24;
    return pagination.page * limit < pagination.total;
  }

  return false;
}


export function getPostDetail(postId: string | number, context: ApiRequestContext = {}) {
  return apiGet<FeedPost>(`/api/posts/${postId}`, { skipViewCount: true }, {
    context,
    auth: shouldAuthenticateFeedRequest(context),
    signal: context.signal,
  });
}

export function getPostProtectionConfig() {
  return apiGet<{
    archive?: { enabled: boolean; threshold: number };
    enabled: boolean;
    maxImages: number;
    maxContentLength: number;
    noticeEnabled: boolean;
    selectAllEnabled: boolean;
    paymentMethods: { balance: boolean; points: boolean };
    paymentMaxPrices?: { balance?: number; points?: number };
  }>("/api/posts/protection-config", undefined, {
    auth: false,
  });
}

export function getPostPurchaseUsers(postId: string | number, page = 1, limit = 20) {
  return apiGet<PostPurchaseUsersPayload>(`/api/posts/${postId}/purchases`, { page, limit }, {
    auth: true,
  });
}

export function getAppDownloadConfig(context: ApiRequestContext = {}) {
  return apiGet<AppDownloadConfig>("/api/app/download-config", undefined, {
    auth: false,
    context,
  });
}

export function getUserToolbarItems(context: ApiRequestContext = {}) {
  return apiGet<UserToolbarItem[]>("/api/users/toolbar/items", undefined, {
    auth: false,
    context,
  });
}

export function createProtectedPackage(postId: string | number) {
  return apiPost<ProtectedPackageJob>(`/api/posts/${postId}/protected-package`, {});
}

export function getProtectedPackageStatus(jobId: string) {
  return apiGet<ProtectedPackageJob>(`/api/protected-packages/${jobId}`);
}

export async function streamProtectedPackageEvents(
  jobId: string,
  onEvent: (event: ProtectedPackageEvent) => void,
  options: { signal?: AbortSignal } = {},
) {
  const headers = new Headers({ Accept: "application/x-ndjson" });
  applyAuthorizationHeader(headers, getStoredAccessToken());
  const response = await fetch(buildApiUrl(`/api/protected-packages/${jobId}/events`), {
    method: "GET",
    headers,
    cache: "no-store",
    signal: options.signal,
  });
  if (!response.ok || !response.body) {
    throw new Error(`protected package event stream failed: ${response.status}`);
  }
  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";
  let terminal = false;
  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value, { stream: !done });
    const lines = buffer.split("\n");
    buffer = lines.pop() ?? "";
    for (const line of lines) {
      if (!line.trim()) continue;
      const event = JSON.parse(line) as ProtectedPackageEvent;
      onEvent(event);
      if (["result", "error"].includes(event.type)) {
        terminal = true;
      }
    }
    if (done) break;
  }
  if (!terminal) {
    throw new Error("protected package event stream ended before completion");
  }
}

export function downloadProtectedPackage(jobId: string) {
  return apiDownload(`/api/protected-packages/${jobId}/download`);
}

export function getPostImageArchive(postId: string | number) {
  return apiGet<PostImageArchiveJob>(`/api/posts/${postId}/image-archive`);
}

export function downloadPostImageArchive(jobId: string) {
  return apiDownload(`/api/image-archives/${jobId}/download`);
}


export async function getDraftPosts(
  query: { page?: number; limit?: number } = {},
): Promise<FeedPayload> {
  const page = query.page ?? 1;
  const limit = query.limit ?? 20;
  const payload = await apiGet<unknown>("/api/posts", {
    page,
    limit,
    is_draft: 1,
  });

  return normalizeFeedPayload(payload, { page, limit });
}


export function updatePost(postId: string | number, input: PublishPostInput) {
  return apiPut<{ id?: number | string }>(`/api/posts/${postId}`, input);
}


export function deletePost(postId: string | number) {
  return apiDelete<{ deletedCount?: number; id?: number | string }>(
    `/api/posts/${postId}`,
  );
}


export function recordUserHistory(postId: string | number) {
  return apiPost<{ success?: boolean }>("/api/users/history", {
    post_id: postId,
  });
}


export function getPostComments(
  postId: string | number,
  page = 1,
  limit = 20,
  context: ApiRequestContext = {},
) {
  return apiGet<CommentsPayload>(
    `/api/posts/${postId}/comments`,
    { page, limit },
    { context, auth: shouldAuthenticateFeedRequest(context) },
  );
}


export function getCommentReplies(
  commentId: string | number,
  page = 1,
  limit = 20,
  context: ApiRequestContext = {},
) {
  return apiGet<CommentsPayload>(
    `/api/comments/${commentId}/replies`,
    { page, limit },
    { context, auth: shouldAuthenticateFeedRequest(context) },
  );
}


type InteractionRequestOptions = {
  redirectOnUnauthorized?: boolean;
};

export function createComment(
  postId: string | number,
  content: string,
  parentId?: string | number | null,
  options: InteractionRequestOptions = {},
) {
  return apiPost<BackendComment>("/api/comments", {
    post_id: postId,
    content,
    ...(parentId ? { parent_id: parentId } : {}),
  }, {
    redirectOnUnauthorized: options.redirectOnUnauthorized,
  });
}


export function deleteComment(commentId: string | number) {
  return apiDelete<{ deletedCount?: number; id?: number | string }>(
    `/api/comments/${commentId}`,
  );
}


export function toggleLike(postId: string | number, options: InteractionRequestOptions = {}) {
  return apiPost<{ liked: boolean }>("/api/likes", {
    target_type: 1,
    target_id: postId,
  }, {
    redirectOnUnauthorized: options.redirectOnUnauthorized,
  });
}


export function toggleCommentLike(
  commentId: string | number,
  options: InteractionRequestOptions = {},
) {
  return apiPost<{ liked: boolean }>("/api/likes", {
    target_type: 2,
    target_id: commentId,
  }, {
    redirectOnUnauthorized: options.redirectOnUnauthorized,
  });
}


export function toggleCollect(postId: string | number, options: InteractionRequestOptions = {}) {
  return apiPost<{ collected: boolean }>(`/api/posts/${postId}/collect`, undefined, {
    redirectOnUnauthorized: options.redirectOnUnauthorized,
  });
}


export async function getNoteCategories(context: ApiRequestContext = {}) {
  const payload = await apiGet<unknown>("/api/categories", undefined, {
    context,
    auth: shouldAuthenticateFeedRequest(context),
  });

  return normalizeCategoriesPayload(payload);
}


export function normalizeCategoriesPayload(payload: unknown): Category[] {
  const record = isRecord(payload) ? payload : null;
  const items = Array.isArray(payload)
    ? payload
    : Array.isArray(record?.categories)
      ? record.categories
      : Array.isArray(record?.data)
        ? record.data
        : Array.isArray(record?.items)
          ? record.items
          : Array.isArray(record?.list)
            ? record.list
            : [];

  return items
    .map((item) => normalizeCategory(item))
    .filter((category): category is Category => Boolean(category));
}


export function normalizeCategory(item: unknown): Category | null {
  if (!isRecord(item)) {
    return null;
  }

  const id = numberFromUnknown(item.id, Number.NaN);
  const rawName = item.name ?? item.category_name ?? item.title ?? item.category_title;
  const name = String(rawName ?? "").trim();
  if (!Number.isFinite(id) || !name) {
    return null;
  }

  return {
    id,
    name,
    display_name:
      typeof item.display_name === "string" ? item.display_name : undefined,
    translations:
      isRecord(item.translations)
        ? Object.fromEntries(Object.entries(item.translations).filter(([, value]) => typeof value === "string"))
        : undefined,
    category_title:
      typeof item.category_title === "string" ? item.category_title : undefined,
    use_count:
      item.use_count === undefined ? undefined : numberFromUnknown(item.use_count),
  };
}


export async function getHotCategories(limit = 8): Promise<Category[]> {
  try {
    const payload = await apiGet<unknown>("/api/categories/hot", { limit }, { auth: false });
    return normalizeCategoriesPayload(payload);
  } catch (error) {
    if (isAccessBlockApiError(error)) {
      throw error;
    }
    return [];
  }
}


export async function getPostTags(context: ApiRequestContext = {}): Promise<PostTag[]> {
  try {
    const payload = await apiGet<unknown>("/api/tags", undefined, {
      context,
      auth: shouldAuthenticateFeedRequest(context),
    });
    return normalizeTagsPayload(payload);
  } catch (error) {
    if (isAccessBlockApiError(error)) {
      throw error;
    }
    return [];
  }
}


export function normalizeTagsPayload(payload: unknown): PostTag[] {
  const record = isRecord(payload) ? payload : null;
  const items = Array.isArray(payload)
    ? payload
    : Array.isArray(record?.data)
      ? record.data
      : Array.isArray(record?.tags)
        ? record.tags
        : Array.isArray(record?.items)
          ? record.items
          : Array.isArray(record?.list)
            ? record.list
            : [];

  return items
    .map((item: unknown) => normalizeTag(item))
    .filter((tag): tag is PostTag => Boolean(tag));
}


export function normalizeTag(item: unknown): PostTag | null {
  if (!isRecord(item)) {
    return null;
  }

  const id = numberFromUnknown(item.id, Number.NaN);
  const rawName = item.name ?? item.tag_name ?? item.title;
  const name = String(rawName ?? "").trim();
  if (!Number.isFinite(id) || !name) {
    return null;
  }

  return {
    id,
    name,
    use_count:
      item.use_count === undefined ? undefined : numberFromUnknown(item.use_count),
  };
}


export async function getInitialFeedData(
  context: ApiRequestContext = {},
): Promise<InitialFeedData> {
  try {
    const [feed, categories, authConfig, toolbarItems, viewer] = await Promise.all([
      getFeedPage("recommended", context),
      getNoteCategories(context).catch(() => []),
      getAuthConfig(context).catch(() => null),
      getUserToolbarItems(context).catch(() => []),
      getRequestAccessToken(context)
        ? getCurrentUser(context).catch(() => null)
        : Promise.resolve(null),
    ]);
    const videoCenter = authConfig
      ? videoCenterConfigFromAuthConfig(authConfig)
      : normalizeVideoCenterConfig();

    return {
      posts: feed.posts,
      categories,
      pagination: feed.pagination,
      source: "backend",
      toolbarItems,
      viewer: viewer ? { createdAt: viewer.created_at ?? null } : undefined,
      siteProfile: normalizeSiteProfile(authConfig?.siteProfile),
      videoCenter,
    };
  } catch (error) {
    if (isAccessBlockApiError(error)) {
      throw error;
    }

    if (error instanceof ApiUnauthorizedError) {
      if (shouldAllowGuestPagePreview()) {
        return emptyBackendInitialFeedData();
      }

      throw error;
    }

    if (!shouldUseFeedFixtureFallback()) {
      throw error;
    }

    return fixtureInitialFeedData;
  }
}


export function shouldUseFeedFixtureFallback() {
  const rawValue =
    process.env.FEED_FIXTURE_FALLBACK ??
    process.env.NEXT_PUBLIC_FEED_FIXTURE_FALLBACK;

  if (rawValue !== undefined) {
    return truthyEnvValues.has(rawValue.trim().toLowerCase());
  }

  return false;
}


export function shouldAuthenticateFeedRequest(context?: ApiRequestContext) {
  return !shouldAllowGuestPagePreview() || Boolean(getRequestAccessToken(context));
}


export function shouldAllowGuestPagePreview() {
  const rawValue =
    process.env.ALLOW_GUEST_PAGE_PREVIEW ??
    process.env.NEXT_PUBLIC_ALLOW_GUEST_PAGE_PREVIEW;

  return rawValue !== undefined && truthyEnvValues.has(rawValue.trim().toLowerCase());
}


export function emptyBackendInitialFeedData(): InitialFeedData {
  return {
    posts: [],
    categories: [],
    pagination: { page: 1, limit: 24, total: 0, pages: 0, hasNextPage: false },
    source: "backend",
    backendNotice: "guest_preview_unauthorized",
    toolbarItems: [],
    siteProfile: normalizeSiteProfile(),
  };
}


export function createPost(input: PublishPostInput) {
  return apiPost<{ id: number }>("/api/posts", input);
}
