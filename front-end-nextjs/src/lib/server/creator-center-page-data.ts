import "server-only";

import {
  getCreatorEarningsLog,
  getCreatorOverview,
  getCurrentUser,
  getNotificationUnreadCount,
  getRequestAccessToken,
  type ApiRequestContext,
} from "@/lib/api";
import type {
  AuthUser,
  CreatorEarningsLogItem,
  CreatorOverviewPayload,
} from "@/lib/types";

export type CreatorCenterInitialData = {
  authenticated: boolean;
  currentUser: AuthUser | null;
  earnings: CreatorEarningsLogItem[];
  overview: CreatorOverviewPayload | null;
  overviewFailed: boolean;
  unreadCount: number;
};

export async function getCreatorCenterInitialData(
  context: ApiRequestContext,
): Promise<CreatorCenterInitialData> {
  const authenticated = Boolean(getRequestAccessToken(context));
  if (!authenticated) {
    return {
      authenticated,
      currentUser: null,
      earnings: [],
      overview: null,
      overviewFailed: false,
      unreadCount: 0,
    };
  }

  const [user, overview, earnings, unreadCount] = await Promise.allSettled([
    getCurrentUser(context),
    getCreatorOverview(context),
    getCreatorEarningsLog({ limit: 5 }, context),
    getNotificationUnreadCount(context),
  ]);

  return {
    authenticated,
    currentUser: user.status === "fulfilled" ? user.value : null,
    earnings:
      earnings.status === "fulfilled" ? earnings.value.list.slice(0, 5) : [],
    overview: overview.status === "fulfilled" ? overview.value : null,
    overviewFailed: overview.status === "rejected",
    unreadCount:
      unreadCount.status === "fulfilled" ? unreadCount.value.total ?? 0 : 0,
  };
}
