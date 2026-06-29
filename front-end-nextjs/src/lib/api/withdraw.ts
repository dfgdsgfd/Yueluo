import type {
  AdminWithdrawOrdersPayload,
  WithdrawApplyPayload,
  WithdrawOrdersPayload,
  WithdrawPaymentCodePayload,
  WithdrawType,
  WithdrawWalletPayload
} from "../types";
import {
  ApiRequestContext,
  apiAdminRequest,
  apiGet,
  apiPost
} from "./core";

export function getWithdrawWallet(context: ApiRequestContext = {}) {
  return apiGet<WithdrawWalletPayload>("/api/withdraw/wallet", undefined, { context });
}


export function getWithdrawPaymentCode() {
  return apiGet<WithdrawPaymentCodePayload>("/api/withdraw/payment-code");
}


export function saveWithdrawPaymentCode(input: WithdrawPaymentCodePayload) {
  return apiPost<WithdrawPaymentCodePayload>("/api/withdraw/payment-code", input);
}


export function applyWithdraw(input: { amount: number; type: WithdrawType }) {
  return apiPost<WithdrawApplyPayload>("/api/withdraw/apply", input);
}


export function getWithdrawOrders(
  query: { page?: number; limit?: number; status?: string } = {},
  context: ApiRequestContext = {},
) {
  return apiGet<WithdrawOrdersPayload>("/api/withdraw/orders", {
    page: query.page ?? 1,
    limit: query.limit ?? 8,
    status: query.status,
  }, { context });
}


export function getAdminWithdrawOrders(
  query: { page?: number; limit?: number; status?: string; keyword?: string } = {},
  token?: string | null,
) {
  return apiAdminRequest<AdminWithdrawOrdersPayload>("/api/withdraw/admin/orders", {
    method: "GET",
    token,
    query: {
      page: query.page ?? 1,
      limit: query.limit ?? 8,
      status: query.status,
      keyword: query.keyword,
    },
  });
}


export function updateAdminWithdrawOrder(
  orderId: string | number,
  action: "approve" | "reject" | "payout",
  input: { remark?: string } = {},
  token?: string | null,
) {
  return apiAdminRequest<{ id: number | string; status: string }>(
    `/api/withdraw/admin/orders/${orderId}/${action}`,
    {
      method: "PUT",
      token,
      body: JSON.stringify(input),
    },
  );
}
