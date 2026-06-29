"use client";

import type { AuthUser } from "@/lib/types/auth";
import { MobilePublishView } from "./mobile-publish-view";
import { useMobilePublishController } from "./use-mobile-publish-controller";

export { prefetchMobilePublishData } from "./mobile-publish-bootstrap";

export function MobilePublishPage({ initialUser }: { initialUser?: AuthUser | null }) {
  const controller = useMobilePublishController({ initialUser });
  return <MobilePublishView controller={controller} />;
}
