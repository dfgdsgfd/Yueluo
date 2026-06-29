export type WalletField =
  | "balanceConfig"
  | "rechargeConfig"
  | "localPoints"
  | "userBalance"
  | "withdrawOrders"
  | "balanceOrders"
  | "pointsOverview"
  | "pointsLogs";

export type WalletError = {
  label: string;
  message: string;
};

export type WalletBalanceStatus = "idle" | "loading" | "ready" | "error";

export type RechargeDisplayOption = {
  amount: number;
  detail: string;
  key: string;
};

export const loadLabelKeys = {
  balanceConfig: "errors.sources.balanceConfig",
  rechargeConfig: "errors.sources.rechargeConfig",
  localPoints: "errors.sources.localPoints",
  userBalance: "errors.sources.userBalance",
  withdrawOrders: "errors.sources.withdrawOrders",
  balanceOrders: "errors.sources.balanceOrders",
  pointsOverview: "errors.sources.pointsOverview",
  pointsLogs: "errors.sources.pointsLogs",
} as const satisfies Record<WalletField, string>;

export const hiddenPointLogTypes = new Set(["withdraw_from_earnings", "transfer_to_earnings"]);
