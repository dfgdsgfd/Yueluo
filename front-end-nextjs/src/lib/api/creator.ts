import type {
  CreatorConfigPayload,
  CreatorOverviewPayload,
  CreatorWithdrawPayload,
  CreatorEarningsLogPayload,
  CreatorPaidContentPayload,
  CreatorQualityRewardsPayload,
  CreatorStatsPayload,
  CreatorTrendsPayload
} from "../types";
import {
  ApiRequestContext,
  apiGet,
  apiPost
} from "./core";

export function getCreatorConfig() {
  return apiGet<CreatorConfigPayload>("/api/creator-center/config", undefined, { auth: false });
}


export function getCreatorOverview(context: ApiRequestContext = {}) {
  return apiGet<CreatorOverviewPayload>("/api/creator-center/overview", undefined, { context });
}


export function getCreatorStats(days = 30, context: ApiRequestContext = {}) {
  return apiGet<CreatorStatsPayload>("/api/creator-center/stats", { days }, { context });
}


export function getCreatorTrends(days = 14, context: ApiRequestContext = {}) {
  return apiGet<CreatorTrendsPayload>("/api/creator-center/trends", { days }, { context });
}


export function getCreatorEarningsLog(
  query: { page?: number; limit?: number; type?: string } = {},
  context: ApiRequestContext = {},
) {
  return apiGet<CreatorEarningsLogPayload>("/api/creator-center/earnings-log", {
    page: query.page ?? 1,
    limit: query.limit ?? 6,
    type: query.type,
  }, { context });
}


export function getCreatorPaidContent(
  query: { page?: number; limit?: number } = {},
  context: ApiRequestContext = {},
) {
  return apiGet<CreatorPaidContentPayload>("/api/creator-center/paid-content", {
    page: query.page ?? 1,
    limit: query.limit ?? 4,
  }, { context });
}


export function getCreatorQualityRewards(
  query: { page?: number; limit?: number } = {},
  context: ApiRequestContext = {},
) {
  return apiGet<CreatorQualityRewardsPayload>("/api/creator-center/quality-rewards", {
    page: query.page ?? 1,
    limit: query.limit ?? 4,
  }, { context });
}


export function withdrawCreatorEarnings(amount: number) {
  return apiPost<CreatorWithdrawPayload>("/api/creator-center/withdraw", { amount });
}
