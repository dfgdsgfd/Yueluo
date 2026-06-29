import type {
  BalanceConfigPayload,
  BalanceLocalPointsPayload,
  BalanceOrdersPayload,
  BalanceRechargeConfigPayload,
  BalanceUserBalancePayload,
  PurchaseContentResult,
  PointsLogsPayload,
  PointsOverviewPayload,
  PointsRedemptionsPayload,
  PointsGiftCardRedemption
} from "../types";
import {
  ApiRequestContext,
  apiGet,
  apiPost
} from "./core";

export function getBalanceConfig(context: ApiRequestContext = {}) {
  return apiGet<BalanceConfigPayload>("/api/balance/config", undefined, { auth: false, context });
}


export function getBalanceRechargeConfig(context: ApiRequestContext = {}) {
  return apiGet<BalanceRechargeConfigPayload>("/api/balance/recharge-config", undefined, {
    auth: false,
    context,
  });
}


export function getBalanceLocalPoints(context: ApiRequestContext = {}) {
  return apiGet<BalanceLocalPointsPayload>("/api/balance/local-points", undefined, { context });
}


export function getBalanceUserBalance(context: ApiRequestContext = {}) {
  return apiGet<BalanceUserBalancePayload>("/api/balance/user-balance", undefined, { context });
}


export function getBalanceOrders(
  query: { page?: number; limit?: number } = {},
  context: ApiRequestContext = {},
) {
  return apiGet<BalanceOrdersPayload>("/api/balance/orders", {
    page: query.page ?? 1,
    limit: query.limit ?? 5,
  }, { context });
}

export function purchaseContent(
  postId: string | number,
  paymentMethod: "balance" | "points",
  userCouponId?: string | number | null,
) {
  return apiPost<PurchaseContentResult>("/api/balance/purchase-content", {
    postId,
    paymentMethod,
    user_coupon_id: userCouponId ?? undefined,
  });
}


export function getPointsOverview(context: ApiRequestContext = {}) {
  return apiGet<PointsOverviewPayload>("/api/points/overview", undefined, { context });
}


export function getPointsLogs(
  query: { page?: number; limit?: number } = {},
  context: ApiRequestContext = {},
) {
  return apiGet<PointsLogsPayload>("/api/points/logs", {
    page: query.page ?? 1,
    limit: query.limit ?? 8,
  }, { context });
}


export function getPointsRedemptions(
  query: { page?: number; limit?: number } = {},
  context: ApiRequestContext = {},
) {
  return apiGet<PointsRedemptionsPayload>("/api/points/redemptions", {
    page: query.page ?? 1,
    limit: query.limit ?? 8,
  }, { context });
}


export function redeemPointsGiftCard(productId: string | number) {
  return apiPost<PointsGiftCardRedemption>(`/api/points/gift-cards/${encodeURIComponent(String(productId))}/redeem`);
}
