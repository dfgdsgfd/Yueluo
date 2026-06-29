import type {
  CreateReportPayload,
  DislikeStatusPayload,
  FollowStatusPayload,
  ReportReason,
  ReportStatusPayload
} from "../types";
import {
  ApiRequestContext,
  ApiUnauthorizedError,
  apiDelete,
  apiGet,
  apiPost,
  getStoredAccessToken
} from "./core";

export function followUser(userId: string | number) {
  return apiPost<{ followed: boolean; isFollowing?: boolean; buttonType?: string }>(
    `/api/users/${userId}/follow`,
  );
}


export function unfollowUser(userId: string | number) {
  return apiDelete<{ followed: boolean; isFollowing?: boolean; buttonType?: string }>(
    `/api/users/${userId}/follow`,
  );
}


export function getFollowStatus(
  userId: string | number,
  context: ApiRequestContext = {},
) {
  return apiGet<FollowStatusPayload>(
    `/api/users/${userId}/follow-status`,
    undefined,
    { context },
  );
}


export async function getOptionalFollowStatus(userId: string | number) {
  const token = getStoredAccessToken();
  if (!token) {
    return null;
  }

  try {
    return await apiGet<FollowStatusPayload>(
      `/api/users/${userId}/follow-status`,
      undefined,
      { auth: false, context: { token } },
    );
  } catch (error) {
    if (error instanceof ApiUnauthorizedError) {
      return null;
    }
    throw error;
  }
}


export function toggleDislike(postId: string | number) {
  return apiPost<DislikeStatusPayload>("/api/dislikes", { post_id: postId });
}


export function getDislikeStatus(postId: string | number) {
  return apiGet<DislikeStatusPayload>("/api/dislikes", { post_id: postId });
}


export async function getOptionalDislikeStatus(postId: string | number) {
  const token = getStoredAccessToken();
  if (!token) {
    return null;
  }

  try {
    return await apiGet<DislikeStatusPayload>(
      "/api/dislikes",
      { post_id: postId },
      { auth: false, context: { token } },
    );
  } catch (error) {
    if (error instanceof ApiUnauthorizedError) {
      return null;
    }
    throw error;
  }
}


export function createReport(input: {
  targetType: "post" | "user" | "comment";
  targetId: string | number;
  reason: ReportReason;
  description?: string;
}) {
  return apiPost<CreateReportPayload>("/api/reports", {
    target_type: input.targetType,
    target_id: input.targetId,
    reason: input.reason,
    description: input.description?.trim() || undefined,
  });
}


export function getReportStatus(input: {
  targetType: "post" | "user" | "comment";
  targetId: string | number;
}) {
  return apiGet<ReportStatusPayload>("/api/reports/check", {
    target_type: input.targetType,
    target_id: input.targetId,
  });
}


export async function getOptionalReportStatus(input: {
  targetType: "post" | "user" | "comment";
  targetId: string | number;
}) {
  const token = getStoredAccessToken();
  if (!token) {
    return null;
  }

  try {
    return await apiGet<ReportStatusPayload>(
      "/api/reports/check",
      {
        target_type: input.targetType,
        target_id: input.targetId,
      },
      { auth: false, context: { token } },
    );
  } catch (error) {
    if (error instanceof ApiUnauthorizedError) {
      return null;
    }
    throw error;
  }
}
