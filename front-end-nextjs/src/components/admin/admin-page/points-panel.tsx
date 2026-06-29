"use client";

import { PointsPanelView } from "./points-panel-view";
import { usePointsPanelController } from "./use-points-panel-controller";

export function PointsPanel({ token }: { token: string }) {
  const controller = usePointsPanelController({ token });
  return <PointsPanelView controller={controller} />;
}

export { PointsDrawer, PointsImportDrawer, PointsProductDrawer, PointsRuleDrawer, PointsTaskDrawer } from "./points-panel-drawers";
export {
  emptyToNull,
  pointsProductDraft,
  pointsProductPayload,
  pointsRuleDraft,
  pointsRulePayload,
  pointsTaskDraft,
  pointsTaskPayload,
  taskPeriodLabel,
} from "./points-panel-model";
