"use client";

import { OperationsPanelView } from "./operations-panel-view";
import { useOperationsController } from "./use-operations-controller";

export function OperationsPanel({ token }: { token: string }) {
  const controller = useOperationsController({ token });
  return <OperationsPanelView controller={controller} />;
}
