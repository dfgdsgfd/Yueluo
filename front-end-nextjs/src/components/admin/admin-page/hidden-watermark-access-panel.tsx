"use client";

import { HiddenWatermarkAccessView } from "./hidden-watermark-access-view";
import { useHiddenWatermarkAccessController } from "./use-hidden-watermark-access-controller";

export function HiddenWatermarkAccessPanel({ token }: { token: string }) {
  const controller = useHiddenWatermarkAccessController({ token });
  return <HiddenWatermarkAccessView controller={controller} />;
}
