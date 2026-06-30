"use client";

import { useCallback, useState } from "react";
import { FEED_BOTTOM_SENTINEL_ROOT_MARGIN } from "./explore-config";
import { useFeedSentinel } from "./explore-hooks";

type LoginUnlockGateInput = {
  feedQueryKey: readonly unknown[];
  hasClientAccessToken: boolean;
  hasNextPage?: boolean;
  isFeedFetching: boolean;
  isPlaceholderData: boolean;
  postsLength: number;
};

export function useLoginUnlockGate({
  feedQueryKey,
  hasClientAccessToken,
  hasNextPage,
  isFeedFetching,
  isPlaceholderData,
  postsLength,
}: LoginUnlockGateInput) {
  const [loginUnlockOpen, setLoginUnlockOpen] = useState(false);
  const openLoginUnlock = useCallback(() => setLoginUnlockOpen(true), []);
  const requireLogin = useCallback(() => {
    if (hasClientAccessToken) {
      return false;
    }
    openLoginUnlock();
    return true;
  }, [hasClientAccessToken, openLoginUnlock]);

  const noMoreUnlockSentinelRef = useFeedSentinel({
    enabled:
      !hasClientAccessToken &&
      !isPlaceholderData &&
      !isFeedFetching &&
      postsLength > 0 &&
      !hasNextPage,
    onEnter: openLoginUnlock,
    resetKey: `${feedQueryKey.join(":")}:no-more:${postsLength}`,
    rootMargin: FEED_BOTTOM_SENTINEL_ROOT_MARGIN,
  });

  return {
    loginUnlockOpen,
    noMoreUnlockSentinelRef,
    openLoginUnlock,
    requireLogin,
    setLoginUnlockOpen,
  };
}
