import type { CreatorListPagination } from "./creator";

export type BalanceConfigPayload = {
  enabled: boolean;
};

export type BalanceRechargeOption = {
  amount: number;
  bonus?: number;
};

export type BalanceGiftCardOption = {
  amount: number;
  price: number;
  discount?: number;
};

export type BalanceGiftCardPurchaseConfig = {
  enabled: boolean;
  options?: BalanceGiftCardOption[];
  disallow_self_use?: boolean;
  expires_in_days?: number;
  user_discounts?: unknown[];
};

export type BalanceRechargeConfigPayload = {
  recharge_url?: string;
  custom_amount_enable?: boolean;
  min_amount?: number;
  max_amount?: number;
  options?: BalanceRechargeOption[];
  gift_card_purchase?: BalanceGiftCardPurchaseConfig;
  [key: string]: unknown;
};

export type BalanceLocalPointsPayload = {
  points: number;
};

export type BalanceUserBalancePayload = {
  balance: number;
  cash_balance?: number;
  vip_level?: number;
  username?: string | null;
};

export type BalancePurchaseOrderItem = {
  id: number | string;
  price: number;
  paid_amount: number;
  discount_rate?: number;
  purchase_type?: string | null;
  purchased_at?: string;
  post_id?: number | string | null;
  post_title?: string | null;
  post_cover?: string | null;
};

export type BalanceOrdersPayload = {
  list: BalancePurchaseOrderItem[];
  pagination: CreatorListPagination;
};

export type WithdrawWalletPayload = {
  cash_balance: number;
  total_income: number;
  frozen_amount: number;
};

export type WithdrawPaymentCodePayload = {
  wechat_url?: string | null;
  alipay_url?: string | null;
};

export type WithdrawType = "cash" | "moon_coin";

export type WithdrawOrderItem = {
  id: number | string;
  amount: number;
  type: WithdrawType | string;
  status: string;
  remark?: string | null;
  created_at?: string;
  updated_at?: string;
};

export type WithdrawOrdersPayload = {
  list: WithdrawOrderItem[];
  pagination: CreatorListPagination;
};

export type WithdrawApplyPayload = {
  orderId: number | string;
  amount: number;
  type: WithdrawType | string;
  status: string;
};

export type CreatorWithdrawPayload = {
  amount: number;
  bonusAmount?: number;
  bonusPercent?: number;
  newEarningsBalance: number;
  newPointsBalance: number;
};
