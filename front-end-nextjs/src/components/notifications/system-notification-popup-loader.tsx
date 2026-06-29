"use client";

import dynamic from "next/dynamic";

const SystemNotificationPopup = dynamic(
  () =>
    import("@/components/notifications/system-notification-popup").then(
      (mod) => mod.SystemNotificationPopup,
    ),
  { ssr: false },
);

export function SystemNotificationPopupLoader() {
  return <SystemNotificationPopup />;
}
