"use client";

import dynamic from "next/dynamic";
import { useCallback, useEffect, useState } from "react";
import { AccessBlockGuard } from "@/components/access-block/access-block-guard";
import { AUTH_USER_EVENT, getStoredAccessToken } from "@/lib/api";
import { POINTS_AWARD_EVENT } from "@/lib/points-award-events";

const MaintenanceGate = dynamic(
  () =>
    import("@/components/maintenance/maintenance-gate").then(
      (module) => module.MaintenanceGate,
    ),
  { ssr: false },
);

const PointsAwardFloatLayer = dynamic(
  () =>
    import("@/components/points/points-award-float-layer").then(
      (module) => module.PointsAwardFloatLayer,
    ),
  { ssr: false },
);

const SystemNotificationPopupLoader = dynamic(
  () =>
    import("@/components/notifications/system-notification-popup-loader").then(
      (module) => module.SystemNotificationPopupLoader,
    ),
  { ssr: false },
);

const OnboardingGate = dynamic(
  () =>
    import("@/components/profile/onboarding/onboarding-gate").then(
      (module) => module.OnboardingGate,
    ),
  { ssr: false },
);

export function DeferredGlobalFeatures() {
  const [idleFeaturesReady, setIdleFeaturesReady] = useState(false);
  const [pointsReady, setPointsReady] = useState(false);
  const [onboardingVisible, setOnboardingVisible] = useState(false);
  const closeOnboarding = useCallback(() => setOnboardingVisible(false), []);
  const probeOnboardingState = useCallback(() => {
    if (getStoredAccessToken()) {
      setOnboardingVisible(true);
      return true;
    }
    return false;
  }, []);

  useEffect(() => {
    const onPointsAward = () => setPointsReady(true);
    window.addEventListener(POINTS_AWARD_EVENT, onPointsAward, { once: true });
    const onUserUpdated = (event: Event) => {
      const detail = "detail" in event ? (event as CustomEvent<unknown>).detail : null;
      const user = detail && typeof detail === "object" ? detail as { profile_completed?: boolean } : null;
      if (user?.profile_completed !== true && getStoredAccessToken()) {
        setOnboardingVisible(true);
        return;
      }
    };
    window.addEventListener(AUTH_USER_EVENT, onUserUpdated);

    let cancelled = false;
    let resolved = false;
    const timers: number[] = [];
    const scheduleProbe = (delay: number) => {
      timers.push(
        window.setTimeout(() => {
          if (cancelled || resolved) {
            return;
          }
          resolved = probeOnboardingState();
        }, delay),
      );
    };
    for (const delay of [0, 80, 250, 600, 1200, 2000]) {
      scheduleProbe(delay);
    }

    const scheduleIdle: typeof window.requestIdleCallback =
      window.requestIdleCallback ??
      ((callback) =>
        window.setTimeout(
          () => callback({ didTimeout: false, timeRemaining: () => 0 }),
          1200,
        ));
    const cancelIdle: typeof window.cancelIdleCallback =
      window.cancelIdleCallback ?? ((id) => window.clearTimeout(id));
    const idleId = scheduleIdle(() => setIdleFeaturesReady(true), {
      timeout: 3500,
    });

    return () => {
      cancelled = true;
      window.removeEventListener(POINTS_AWARD_EVENT, onPointsAward);
      window.removeEventListener(AUTH_USER_EVENT, onUserUpdated);
      timers.forEach((timer) => window.clearTimeout(timer));
      cancelIdle(idleId);
    };
  }, [probeOnboardingState]);

  return (
    <>
      <AccessBlockGuard />
      {pointsReady ? <PointsAwardFloatLayer /> : null}
      {onboardingVisible ? (
        <OnboardingGate onClose={closeOnboarding} />
      ) : null}
      <SystemNotificationPopupLoader />
      {idleFeaturesReady ? (
        <>
          <MaintenanceGate />
        </>
      ) : null}
    </>
  );
}
