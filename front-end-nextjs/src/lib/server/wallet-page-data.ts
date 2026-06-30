import "server-only";

import {
  getBalanceConfig,
  getBalanceLocalPoints,
  getBalanceOrders,
  getBalanceRechargeConfig,
  getPointsLogs,
  getPointsOverview,
  getPointsRedemptions,
  getRequestAccessToken,
  getWithdrawOrders,
  getWithdrawWallet,
  type ApiRequestContext,
} from "@/lib/api";
import type {
  BalanceConfigPayload,
  BalanceLocalPointsPayload,
  BalanceOrdersPayload,
  BalanceRechargeConfigPayload,
  PointsLogsPayload,
  PointsOverviewPayload,
  PointsRedemptionsPayload,
  WithdrawWalletPayload,
  WithdrawOrdersPayload,
} from "@/lib/types";

export type WalletInitialData = {
  authenticated: boolean;
  balanceConfig: BalanceConfigPayload | null;
  rechargeConfig: BalanceRechargeConfigPayload | null;
  localPoints: BalanceLocalPointsPayload | null;
  userBalance: WithdrawWalletPayload | null;
  withdrawOrders: WithdrawOrdersPayload | null;
  balanceOrders: BalanceOrdersPayload | null;
  pointsOverview: PointsOverviewPayload | null;
  pointsLogs: PointsLogsPayload | null;
  pointsRedemptions: PointsRedemptionsPayload | null;
};

export async function getWalletInitialData(
  context: ApiRequestContext,
): Promise<WalletInitialData> {
  const authenticated = Boolean(getRequestAccessToken(context));
  const publicRequests = [
    getBalanceConfig(context),
    getBalanceRechargeConfig(context),
  ] as const;
  const privateRequests = authenticated
    ? ([
        getBalanceLocalPoints(context),
        getWithdrawWallet(context),
        getWithdrawOrders({ limit: 8 }, context),
        getBalanceOrders({ limit: 5 }, context),
        getPointsOverview(context),
        getPointsLogs({ limit: 8 }, context),
        getPointsRedemptions({ limit: 8 }, context),
      ] as const)
    : null;
  const [publicResults, privateResults] = await Promise.all([
    Promise.allSettled(publicRequests),
    privateRequests ? Promise.allSettled(privateRequests) : Promise.resolve(null),
  ]);

  return {
    authenticated,
    balanceConfig: fulfilledValue(publicResults[0]),
    rechargeConfig: fulfilledValue(publicResults[1]),
    localPoints: fulfilledValue(privateResults?.[0]),
    userBalance: fulfilledValue(privateResults?.[1]),
    withdrawOrders: fulfilledValue(privateResults?.[2]),
    balanceOrders: fulfilledValue(privateResults?.[3]),
    pointsOverview: fulfilledValue(privateResults?.[4]),
    pointsLogs: fulfilledValue(privateResults?.[5]),
    pointsRedemptions: fulfilledValue(privateResults?.[6]),
  };
}

function fulfilledValue<T>(
  result: PromiseSettledResult<T> | null | undefined,
): T | null {
  return result?.status === "fulfilled" ? result.value : null;
}
