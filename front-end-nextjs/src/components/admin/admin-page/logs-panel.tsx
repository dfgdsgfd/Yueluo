"use client";

import { LogsPanelView } from "./logs-panel-view";
import { useLogsPanelController } from "./use-logs-panel-controller";

export function LogsPanel({ token }: { token: string }) {
  const controller = useLogsPanelController({ token });
  return <LogsPanelView controller={controller} />;
}
