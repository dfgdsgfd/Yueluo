import type {
  AdminAiReviewStatusPayload,
  AdminAccessLogAnalyticsPayload,
  AdminAccessLogItem,
  AdminBalanceAuditLogItem,
  AdminBatchGenerateUsersPayload,
  AdminDashboardHotContentPayload,
  AdminDashboardOverviewPayload,
  AdminDashboardTrendsPayload,
  AdminGuestAccessStatusPayload,
  AdminLogListPayload,
  AdminListResource,
  AdminListRow,
  AdminSecurityAuditLogItem,
  AdminPointsAuditLogItem,
  AdminUserPointsUpdatePayload,
  AdminStatsOverviewPayload,
  FileRecycleInspectPayload
} from "../types";
import {
  QueryValue,
  apiAdminEnvelope,
  apiAdminDownload,
  apiAdminRequest,
  normalizeAdminListEnvelope
} from "./core";

export function getAdminStatsOverview(token?: string | null) {
  return apiAdminRequest<AdminStatsOverviewPayload>("/api/admin/stats/overview", {
    method: "GET",
    token,
  });
}


export function getAdminAiReviewStatus(token?: string | null) {
  return apiAdminRequest<AdminAiReviewStatusPayload>("/api/admin/ai-review-status", {
    method: "GET",
    token,
  });
}


export function toggleAdminAiReview(enabled: boolean, token?: string | null) {
  return apiAdminRequest<unknown>("/api/admin/ai-review-toggle", {
    method: "POST",
    body: JSON.stringify({ enabled }),
    token,
  });
}


export function getAdminGuestAccessStatus(token?: string | null) {
  return apiAdminRequest<AdminGuestAccessStatusPayload>("/api/admin/guest-access-status", {
    method: "GET",
    token,
  });
}


export function toggleAdminGuestAccess(restricted: boolean, token?: string | null) {
  return apiAdminRequest<unknown>("/api/admin/guest-access-toggle", {
    method: "POST",
    body: JSON.stringify({ restricted }),
    token,
  });
}


export async function getAdminList<T extends AdminListRow = AdminListRow>(
  resource: AdminListResource,
  query: {
    page?: number;
    limit?: number;
    keyword?: string;
    status?: string;
    sortBy?: string;
    sortField?: string;
    sortOrder?: "ASC" | "DESC";
    filters?: Record<string, QueryValue>;
    basePath?: string;
  } = {},
  token?: string | null,
) {
  const page = query.page ?? 1;
  const limit = query.limit ?? 10;
  const path = query.basePath ?? `/api/admin/${resource}`;
  const envelope = await apiAdminEnvelope<unknown>(path, {
    method: "GET",
    token,
    query: {
      page,
      limit,
      pageSize: limit,
      keyword: query.keyword,
      search: query.keyword,
      status: query.status,
      sortBy: query.sortBy ?? query.sortField,
      sortField: query.sortField ?? query.sortBy,
      sortOrder: query.sortOrder,
      ...query.filters,
    },
  });

  return normalizeAdminListEnvelope<T>(envelope, { page, limit });
}


export function getAdminResourceDetail<T extends AdminListRow = AdminListRow>(
  resource: AdminListResource,
  id: string | number,
  token?: string | null,
  basePath?: string,
) {
  return apiAdminRequest<T>(`${basePath ?? `/api/admin/${resource}`}/${encodeURIComponent(String(id))}`, {
    method: "GET",
    token,
  });
}


export function createAdminResource<T extends AdminListRow = AdminListRow>(
  resource: AdminListResource,
  input: Record<string, unknown>,
  token?: string | null,
  basePath?: string,
) {
  return apiAdminRequest<T>(basePath ?? `/api/admin/${resource}`, {
    method: "POST",
    body: JSON.stringify(input),
    token,
  });
}


export function updateAdminResource<T = unknown>(
  resource: AdminListResource,
  id: string | number,
  input: Record<string, unknown>,
  token?: string | null,
  basePath?: string,
) {
  return apiAdminRequest<T>(`${basePath ?? `/api/admin/${resource}`}/${encodeURIComponent(String(id))}`, {
    method: "PUT",
    body: JSON.stringify(input),
    token,
  });
}


export function deleteAdminResource<T = unknown>(
  resource: AdminListResource,
  id: string | number,
  token?: string | null,
  basePath?: string,
) {
  return apiAdminRequest<T>(`${basePath ?? `/api/admin/${resource}`}/${encodeURIComponent(String(id))}`, {
    method: "DELETE",
    token,
  });
}


export function bulkDeleteAdminResource<T = unknown>(
  resource: AdminListResource,
  ids: Array<string | number>,
  token?: string | null,
  basePath?: string,
) {
  return apiAdminRequest<T>(basePath ?? `/api/admin/${resource}`, {
    method: "DELETE",
    body: JSON.stringify({ ids }),
    token,
  });
}

export function batchGenerateAdminUsers(count: number, token?: string | null) {
  return apiAdminRequest<AdminBatchGenerateUsersPayload>("/api/admin/users/batch-generate", {
    method: "POST",
    token,
    body: JSON.stringify({ count }),
  });
}


export function adminRequest<T>(
  path: string,
  options: RequestInit & {
    query?: Record<string, QueryValue>;
    token?: string | null;
    timeoutMs?: number;
  } = {},
) {
  return apiAdminRequest<T>(path, options);
}

export function inspectFileRecycleItem(id: string | number, token?: string | null) {
  return apiAdminRequest<FileRecycleInspectPayload>(`/api/admin/file-recycle-bin/${encodeURIComponent(String(id))}/inspect`, {
    method: "GET",
    token,
  });
}

export function previewFileRecycleItem(id: string | number, token?: string | null) {
  return apiAdminDownload(`/api/admin/file-recycle-bin/${encodeURIComponent(String(id))}/preview`, {
    method: "GET",
    token,
  });
}

export function downloadFileRecycleItem(id: string | number, token?: string | null) {
  return apiAdminDownload(`/api/admin/file-recycle-bin/${encodeURIComponent(String(id))}/download`, {
    method: "GET",
    token,
  });
}


export function getAdminDashboardOverview(token?: string | null) {
  return apiAdminRequest<AdminDashboardOverviewPayload>("/api/admin/dashboard/overview", {
    method: "GET",
    token,
  });
}


export function getAdminDashboardTrends(days = 7, token?: string | null) {
  return apiAdminRequest<AdminDashboardTrendsPayload>("/api/admin/dashboard/trends", {
    method: "GET",
    token,
    query: { days },
  });
}


export function getAdminDashboardHotContent(token?: string | null) {
  return apiAdminRequest<AdminDashboardHotContentPayload>("/api/admin/dashboard/hot-content", {
    method: "GET",
    token,
  });
}

export function getAdminAccessLogAnalytics(
  query: Record<string, QueryValue> = {},
  token?: string | null,
) {
  return apiAdminRequest<AdminAccessLogAnalyticsPayload>("/api/admin/logs/access/analytics", {
    method: "GET",
    token,
    query,
  });
}

export function getAdminAccessLogs(
  query: Record<string, QueryValue> = {},
  token?: string | null,
) {
  return apiAdminRequest<AdminLogListPayload<AdminAccessLogItem>>("/api/admin/logs/access", {
    method: "GET",
    token,
    query,
  });
}

export function getAdminSecurityAuditLogs(
  query: Record<string, QueryValue> = {},
  token?: string | null,
) {
  return apiAdminRequest<AdminLogListPayload<AdminSecurityAuditLogItem>>("/api/admin/logs/security", {
    method: "GET",
    token,
    query,
  });
}

export function getAdminPointsAuditLogs(
  query: Record<string, QueryValue> = {},
  token?: string | null,
) {
  return apiAdminRequest<AdminLogListPayload<AdminPointsAuditLogItem>>("/api/admin/logs/points", {
    method: "GET",
    token,
    query,
  });
}

export function getAdminBalanceAuditLogs(
  query: Record<string, QueryValue> = {},
  token?: string | null,
) {
  return apiAdminRequest<AdminLogListPayload<AdminBalanceAuditLogItem>>("/api/admin/logs/balance", {
    method: "GET",
    token,
    query,
  });
}

export function updateAdminUserPoints(
  userId: string | number,
  input: { operation: "add" | "deduct" | "set"; amount: number; reason?: string },
  token?: string | null,
) {
  return apiAdminRequest<AdminUserPointsUpdatePayload>(`/api/admin/users/${encodeURIComponent(String(userId))}/points`, {
    method: "POST",
    token,
    body: JSON.stringify(input),
  });
}
