import type {
  InviteClickPayload,
  InviteCodePayload,
  InviteInfoPayload,
  InviteStatsPayload
} from "../types";
import {
  apiGet,
  apiPost
} from "./core";

export function getInviteMyCode() {
  return apiGet<InviteCodePayload>("/api/invite/my-code");
}


export function getInviteStats(query: { page?: number; limit?: number } = {}) {
  return apiGet<InviteStatsPayload>("/api/invite/stats", {
    page: query.page ?? 1,
    limit: query.limit ?? 10,
  });
}


export function getInviteInfo(code: string) {
  return apiGet<InviteInfoPayload>(`/api/invite/info/${encodeURIComponent(code)}`, undefined, { auth: false });
}


export function recordInviteClick(code: string) {
  return apiPost<InviteClickPayload>(`/api/invite/click/${encodeURIComponent(code)}`, undefined, { auth: false });
}
