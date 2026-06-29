import type { PointsAchievementRule, PointsGiftCardProduct, PointsTaskConfig } from "@/lib/types";
import { truthy } from "./helpers";

export function pointsTaskDraft(row: PointsTaskConfig | null): Record<string, unknown> {
  return {
    task_type: row?.task_type ?? "comment",
    name: row?.name ?? "评论",
    description: row?.description ?? "",
    points: row?.points ?? 1,
    daily_limit: row?.daily_limit ?? 10,
    is_daily_task: row?.is_daily_task ?? true,
    is_active: row?.is_active ?? true,
    sort_order: row?.sort_order ?? 10,
  };
}


export function pointsTaskPayload(draft: Record<string, unknown>) {
  return {
    task_type: String(draft.task_type ?? "").trim(),
    name: String(draft.name ?? "").trim(),
    description: emptyToNull(draft.description),
    points: Number(draft.points ?? 0),
    daily_limit: Number(draft.daily_limit ?? 0),
    is_daily_task: Boolean(draft.is_daily_task),
    is_active: Boolean(draft.is_active),
    sort_order: Number(draft.sort_order ?? 0),
  };
}


export function taskPeriodLabel(task: PointsTaskConfig) {
  return truthy(task.is_daily_task) ? "每日重复" : "固定一次";
}


export function pointsRuleDraft(row: PointsAchievementRule | null): Record<string, unknown> {
  return {
    name: row?.name ?? "",
    trigger_type: row?.trigger_type ?? "total_posts",
    threshold_value: row?.threshold_value ?? 10,
    points_reward: row?.points_reward ?? 0,
    creator_bonus_percent: row?.creator_bonus_percent ?? 0,
    bonus_days: row?.bonus_days ?? 0,
    description: row?.description ?? "",
    is_active: row?.is_active ?? true,
  };
}


export function pointsRulePayload(draft: Record<string, unknown>) {
  return {
    name: String(draft.name ?? "").trim(),
    trigger_type: String(draft.trigger_type ?? "total_posts"),
    threshold_value: Number(draft.threshold_value ?? 0),
    points_reward: Number(draft.points_reward ?? 0),
    creator_bonus_percent: Number(draft.creator_bonus_percent ?? 0),
    bonus_days: Number(draft.bonus_days ?? 0),
    description: emptyToNull(draft.description),
    is_active: Boolean(draft.is_active),
  };
}


export function pointsProductDraft(row: PointsGiftCardProduct | null): Record<string, unknown> {
  return {
    name: row?.name ?? "",
    description: row?.description ?? "",
    face_value: row?.face_value ?? "",
    points_required: row?.points_required ?? 100,
    is_active: row?.is_active ?? true,
    sort_order: row?.sort_order ?? 10,
  };
}


export function pointsProductPayload(draft: Record<string, unknown>) {
  return {
    name: String(draft.name ?? "").trim(),
    description: emptyToNull(draft.description),
    face_value: emptyToNull(draft.face_value),
    points_required: Number(draft.points_required ?? 0),
    is_active: Boolean(draft.is_active),
    sort_order: Number(draft.sort_order ?? 0),
  };
}


export function emptyToNull(value: unknown) {
  const text = String(value ?? "").trim();
  return text ? text : null;
}
