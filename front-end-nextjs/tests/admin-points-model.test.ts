import { describe, expect, it } from "vitest";
import {
  emptyToNull,
  pointsProductDraft,
  pointsProductPayload,
  pointsRuleDraft,
  pointsRulePayload,
  pointsTaskDraft,
  pointsTaskPayload,
  taskPeriodLabel,
} from "../src/components/admin/admin-page/points-panel-model";

describe("admin points panel model helpers", () => {
  it("keeps task draft defaults and payload coercion stable", () => {
    expect(pointsTaskDraft(null)).toEqual({
      task_type: "comment",
      name: "评论",
      description: "",
      points: 1,
      daily_limit: 10,
      is_daily_task: true,
      is_active: true,
      sort_order: 10,
    });

    expect(pointsTaskPayload({
      task_type: " post ",
      name: " 发帖 ",
      description: " ",
      points: "2",
      daily_limit: "3",
      is_daily_task: false,
      is_active: true,
      sort_order: "7",
    })).toEqual({
      task_type: "post",
      name: "发帖",
      description: null,
      points: 2,
      daily_limit: 3,
      is_daily_task: false,
      is_active: true,
      sort_order: 7,
    });
  });

  it("keeps rule and product draft serialization compatible", () => {
    expect(pointsRuleDraft(null)).toMatchObject({
      trigger_type: "total_posts",
      threshold_value: 10,
      is_active: true,
    });
    expect(pointsRulePayload({ name: " 达人 ", trigger_type: "total_likes", threshold_value: "20", points_reward: "5", creator_bonus_percent: "1.5", bonus_days: "", description: "" })).toMatchObject({
      name: "达人",
      trigger_type: "total_likes",
      threshold_value: 20,
      points_reward: 5,
      creator_bonus_percent: 1.5,
      bonus_days: 0,
      description: null,
    });

    expect(pointsProductDraft(null)).toMatchObject({
      points_required: 100,
      is_active: true,
      sort_order: 10,
    });
    expect(pointsProductPayload({ name: " 礼品卡 ", description: "", face_value: " 50 ", points_required: "1000", is_active: false, sort_order: "9" })).toEqual({
      name: "礼品卡",
      description: null,
      face_value: "50",
      points_required: 1000,
      is_active: false,
      sort_order: 9,
    });
  });

  it("keeps small compatibility helpers unchanged", () => {
    expect(emptyToNull("  value  ")).toBe("value");
    expect(emptyToNull("   ")).toBeNull();
    expect(taskPeriodLabel({ is_daily_task: true } as Parameters<typeof taskPeriodLabel>[0])).toBe("每日重复");
    expect(taskPeriodLabel({ is_daily_task: false } as Parameters<typeof taskPeriodLabel>[0])).toBe("固定一次");
  });
});
