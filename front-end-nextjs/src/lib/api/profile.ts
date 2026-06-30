import type {
  AuthUser,
  CreateVerificationInput,
  CreateVerificationPayload,
  FeedPost,
  OnboardingConfigPayload,
  OnboardingSubmitInput,
  ProfileTabs,
  UpdateUserProfileInput,
  UserApiKey,
  UserSearchResult,
  UserPrivacySettingsPayload,
  UserProfile,
  UserProfilePayload,
  VerificationApplication
} from "../types";
import {
  ApiRequestContext,
  apiDelete,
  apiGet,
  apiPost,
  apiPut,
  isRecord,
  numberFromUnknown
} from "./core";
import {
  getCurrentUser
} from "./auth";
import {
  normalizeFeedPayload
} from "./feed";
import {
  getFollowStatus
} from "./social";

export function mapUserProfile(user: AuthUser, isViewer = false): UserProfile {
  const interests = Array.isArray(user.interests)
    ? user.interests.filter((item): item is string => typeof item === "string")
    : undefined;
  const customFields = isRecord(user.custom_fields)
    ? Object.fromEntries(
        Object.entries(user.custom_fields)
          .filter(([, value]) => typeof value === "string")
          .map(([key, value]) => [key, value as string]),
      )
    : undefined;

  return {
    id: user.id,
    userId: user.user_id ?? String(user.id),
    displayName: user.nickname ?? user.user_id ?? "Yuem User",
    avatar: user.avatar ?? null,
    background: user.background ?? null,
    bio: user.bio ?? null,
    location: user.location ?? null,
    createdAt: user.created_at ?? null,
    verified: user.verified ?? null,
    verifiedName: user.verified_name ?? null,
    followCount: user.follow_count ?? 0,
    fansCount: user.fans_count ?? 0,
    likeCount: user.like_count ?? 0,
    collectCount: user.collect_count ?? 0,
    postCount: user.post_count ?? 0,
    isViewer,
    isFollowing: user.isFollowing,
    birthday: typeof user.birthday === "string" ? user.birthday : null,
    customFields,
    education: user.education ?? null,
    gender: user.gender ?? null,
    interests,
    major: user.major ?? null,
    mbti: user.mbti ?? null,
    zodiacSign: user.zodiac_sign ?? null,
  };
}


export type ProfilePostListKey = "posts" | "collections" | "history";


export function readProfilePostListCandidate(value: unknown): unknown[] | null {
  if (Array.isArray(value)) {
    return value;
  }

  if (!isRecord(value)) {
    return null;
  }

  const nestedKeys = ["data", "posts", "collections", "history", "items", "list"] as const;
  for (const key of nestedKeys) {
    if (Array.isArray(value[key])) {
      return value[key];
    }
  }

  return null;
}


export function unwrapProfilePostItem(item: unknown) {
  if (!isRecord(item)) {
    return item;
  }

  const nestedPost = item.post ?? item.note ?? item.target;
  if (isRecord(nestedPost)) {
    return {
      ...nestedPost,
      liked: nestedPost.liked ?? item.liked,
      collected: nestedPost.collected ?? item.collected,
    };
  }

  return item;
}


export function normalizeProfilePostList(
  payload: unknown,
  fallback: { page: number; limit: number },
  preferredKey: ProfilePostListKey = "posts",
) {
  const record = isRecord(payload) ? payload : {};
  const candidates = [
    record[preferredKey],
    record.posts,
    record.collections,
    record.history,
    record.items,
    record.list,
    record.data,
  ];
  const items =
    candidates.map(readProfilePostListCandidate).find((list) => list !== null) ??
    normalizeFeedPayload(payload, fallback).posts;
  const posts = items.map(unwrapProfilePostItem);

  return posts as FeedPost[];
}


export async function getUserProfilePostList(
  userId: string | number,
  query: { visibility?: string } = {},
  context: ApiRequestContext = {},
) {
  const page = 1;
  const limit = 20;
  const payload = await apiGet<unknown>(
    `/api/users/${encodeURIComponent(String(userId))}/posts`,
    {
      page,
      limit,
      visibility: query.visibility,
    },
    { context },
  );

  return normalizeProfilePostList(payload, { page, limit });
}


export async function getUserProfileCollections(
  userId: string | number,
  context: ApiRequestContext = {},
) {
  const page = 1;
  const limit = 20;
  const payload = await apiGet<unknown>(
    `/api/users/${encodeURIComponent(String(userId))}/collections`,
    { page, limit },
    { context },
  );

  return normalizeProfilePostList(payload, { page, limit }, "collections");
}


export async function getUserProfileLikes(
  userId: string | number,
  context: ApiRequestContext = {},
) {
  const page = 1;
  const limit = 20;
  const payload = await apiGet<unknown>(
    `/api/users/${encodeURIComponent(String(userId))}/likes`,
    { page, limit },
    { context },
  );

  return normalizeProfilePostList(payload, { page, limit });
}


export async function getUserHistoryPosts() {
  const page = 1;
  const limit = 20;
  const payload = await apiGet<unknown>("/api/users/history", { page, limit });

  return normalizeProfilePostList(payload, { page, limit }, "history");
}


export async function getUserPostTabs(
  userId: string | number,
  context: ApiRequestContext = {},
): Promise<ProfileTabs> {
  const [notes, privatePosts, collections, likes] = await Promise.all([
    getUserProfilePostList(userId, {}, context),
    getUserProfilePostList(userId, { visibility: "private" }, context),
    getUserProfileCollections(userId, context),
    getUserProfileLikes(userId, context),
  ]);

  return {
    notes,
    private: privatePosts,
    collections,
    likes,
  };
}


export async function getViewerProfileData(context: ApiRequestContext = {}): Promise<UserProfilePayload> {
  const user = await getCurrentUser(context);
  const tabs = await getUserPostTabs(user.user_id ?? user.id, context);

  return {
    profile: mapUserProfile({ ...user, post_count: tabs.notes.length }, true),
    tabs,
  };
}


export async function getUserProfileData(
  userId: string,
  context: ApiRequestContext = {},
): Promise<UserProfilePayload> {
  const [user, tabs, followStatus] = await Promise.all([
    apiGet<AuthUser>(`/api/users/${encodeURIComponent(userId)}`, undefined, { context }),
    getUserPostTabs(userId, context),
    getFollowStatus(userId, context),
  ]);

  return {
    profile: mapUserProfile({
      ...user,
      buttonType: followStatus.buttonType,
      isFollowing: followStatus.isFollowing ?? followStatus.followed,
      post_count: tabs.notes.length,
    }),
    tabs,
  };
}


export async function searchUsers(
  query: { keyword: string; page?: number; limit?: number },
): Promise<UserSearchResult[]> {
  const payload = await apiGet<unknown>("/api/users/search", {
    keyword: query.keyword.trim(),
    page: query.page ?? 1,
    limit: query.limit ?? 20,
  });

  return normalizeUserSearchResults(payload);
}


export async function listMentionUsers(
  query: { page?: number; limit?: number } = {},
): Promise<UserSearchResult[]> {
  const payload = await apiGet<unknown>("/api/users", {
    page: query.page ?? 1,
    limit: query.limit ?? 20,
  });

  return normalizeUserSearchResults(payload);
}


export function normalizeUserSearchResults(payload: unknown): UserSearchResult[] {
  const record = isRecord(payload) ? payload : {};
  const data = record.data;
  const dataRecord = isRecord(data) ? data : {};
  const candidates = [
    payload,
    data,
    record.users,
    record.items,
    record.list,
    record.results,
    dataRecord.users,
    dataRecord.items,
    dataRecord.list,
    dataRecord.results,
  ];
  const items = candidates.find((item): item is unknown[] => Array.isArray(item)) ?? [];

  return items.flatMap((item) => {
    if (!isRecord(item)) {
      return [];
    }

    const id = item.id ?? item.user_id ?? item.xise_id;
    const nickname = item.nickname ?? item.user_id ?? item.xise_id;
    if (id === undefined || nickname === undefined) {
      return [];
    }

    return [{
      id: id as number | string,
      user_id: typeof item.user_id === "string" ? item.user_id : undefined,
      xise_id: typeof item.xise_id === "string" ? item.xise_id : null,
      nickname: typeof item.nickname === "string" ? item.nickname : String(nickname),
      avatar: typeof item.avatar === "string" ? item.avatar : null,
      bio: typeof item.bio === "string" ? item.bio : null,
      verified: typeof item.verified === "boolean" || typeof item.verified === "number"
        ? item.verified
        : null,
      fans_count: item.fans_count === undefined ? undefined : numberFromUnknown(item.fans_count),
    }];
  });
}


export async function updateUserProfile(userId: string | number, input: UpdateUserProfileInput) {
  const user = await apiPut<AuthUser>(`/api/users/${encodeURIComponent(String(userId))}`, {
    nickname: input.nickname.trim(),
    bio: input.bio?.trim() ?? "",
    location: input.location?.trim() ?? "",
    avatar: input.avatar?.trim() ?? "",
    background: input.background?.trim() ?? "",
    birthday: input.birthday?.trim() ?? "",
    custom_fields: input.custom_fields ?? {},
    education: input.education?.trim() ?? "",
    gender: input.gender?.trim() ?? "",
    interests: input.interests ?? [],
    major: input.major?.trim() ?? "",
    mbti: input.mbti?.trim() ?? "",
    zodiac_sign: input.zodiac_sign?.trim() ?? "",
  });

  return mapUserProfile(user, true);
}


export function getOnboardingConfig() {
  return apiGet<OnboardingConfigPayload>("/api/users/onboarding-config", undefined, { auth: false });
}


export function submitOnboarding(input: OnboardingSubmitInput) {
  return apiPost<AuthUser>("/api/users/onboarding", input);
}


export async function getUserPrivacySettings() {
  const payload = await apiGet<Partial<UserPrivacySettingsPayload>>("/api/users/privacy-settings");
  return normalizePrivacySettings(payload);
}


export async function updateUserPrivacySettings(input: UserPrivacySettingsPayload) {
  const payload = await apiPut<Partial<UserPrivacySettingsPayload>>("/api/users/privacy-settings", input);
  return normalizePrivacySettings(payload);
}


export function getVerificationApplications() {
  return apiGet<VerificationApplication[]>("/api/users/verification/status");
}


export function createVerificationApplication(input: CreateVerificationInput) {
  const imageUrls = input.imageUrls?.map((url) => url.trim()).filter(Boolean).slice(0, 9) ?? [];
  return apiPost<CreateVerificationPayload>("/api/users/verification", {
    type: input.type,
    content: input.content.trim(),
    verifiedName: input.verifiedName?.trim() ?? "",
    imageUrls,
  });
}


export function getUserApiKeys() {
  return apiGet<UserApiKey[]>("/api/users/api-keys");
}


export function createUserApiKey(name: string) {
  return apiPost<UserApiKey>("/api/users/api-keys", { name: name.trim() });
}


export function deleteUserApiKey(id: string | number) {
  return apiDelete<Record<string, never>>(`/api/users/api-keys/${encodeURIComponent(String(id))}`);
}


export function normalizePrivacySettings(payload: Partial<UserPrivacySettingsPayload>): UserPrivacySettingsPayload {
  return {
    ai_auto_comment_enabled: payload.ai_auto_comment_enabled !== false,
    privacy_age: Boolean(payload.privacy_age),
    privacy_birthday: Boolean(payload.privacy_birthday),
    privacy_custom_fields: normalizePrivacyCustomFields(payload.privacy_custom_fields),
    privacy_mbti: Boolean(payload.privacy_mbti),
    privacy_zodiac: Boolean(payload.privacy_zodiac),
  };
}


export function normalizePrivacyCustomFields(value: unknown) {
  if (!isRecord(value)) {
    return {};
  }

  return Object.fromEntries(
    Object.entries(value).map(([key, item]) => [key, Boolean(item)]),
  );
}
