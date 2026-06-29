"use client";

import type { FormEvent, ReactNode } from "react";
import { Loader2, Save, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { PointsAchievementRule, PointsGiftCardProduct, PointsTaskConfig } from "@/lib/types";
import { achievementTriggerOptions, pointsTaskTypeOptions } from "./types";
import { TextField } from "./object-picker";
import { BooleanSelect, SelectField, TextareaField } from "./form-fields";
import { truthy } from "./helpers";

export function PointsTaskDrawer({
  row,
  draft,
  saving,
  onDraftChange,
  onClose,
  onSubmit,
}: {
  row: PointsTaskConfig | null;
  draft: Record<string, unknown>;
  saving: boolean;
  onDraftChange: (key: string, value: unknown) => void;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  if (row === null) return null;
  const isEdit = row.id !== undefined;
  const isDaily = truthy(draft.is_daily_task);
  return (
    <PointsDrawer title={isEdit ? "编辑任务" : "新建任务"} saving={saving} onClose={onClose} onSubmit={onSubmit}>
      <SelectField label="任务类型 *" value={draft.task_type} options={pointsTaskTypeOptions} onChange={(value) => onDraftChange("task_type", value)} />
      <TextField label="任务名称 *" value={draft.name} onChange={(value) => onDraftChange("name", value)} />
      <div className="grid gap-3 sm:grid-cols-2">
        <TextField label="每次积分 *" value={draft.points} onChange={(value) => onDraftChange("points", value)} type="number" />
        <TextField label={isDaily ? "每日次数 *" : "完成次数上限"} value={draft.daily_limit} onChange={(value) => onDraftChange("daily_limit", value)} type="number" />
        <TextField label="排序" value={draft.sort_order} onChange={(value) => onDraftChange("sort_order", value)} type="number" />
        <BooleanSelect label="启用" value={draft.is_active} onChange={(value) => onDraftChange("is_active", value)} />
      </div>
      <BooleanSelect
        label="任务周期"
        value={draft.is_daily_task}
        onChange={(value) => {
          onDraftChange("is_daily_task", value);
          if (!value) {
            onDraftChange("daily_limit", 1);
          }
        }}
        onLabel="每日重复"
        offLabel="固定一次"
      />
      <TextareaField label="描述" value={draft.description} onChange={(value) => onDraftChange("description", value)} />
    </PointsDrawer>
  );
}


export function PointsRuleDrawer({
  row,
  draft,
  saving,
  onDraftChange,
  onClose,
  onSubmit,
}: {
  row: PointsAchievementRule | null;
  draft: Record<string, unknown>;
  saving: boolean;
  onDraftChange: (key: string, value: unknown) => void;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  if (row === null) return null;
  const isEdit = row.id !== undefined;
  return (
    <PointsDrawer title={isEdit ? "编辑成就规则" : "新建成就规则"} saving={saving} onClose={onClose} onSubmit={onSubmit}>
      <TextField label="规则名称 *" value={draft.name} onChange={(value) => onDraftChange("name", value)} />
      <SelectField label="触发类型 *" value={draft.trigger_type ?? "total_posts"} options={achievementTriggerOptions} onChange={(value) => onDraftChange("trigger_type", value)} />
      <div className="grid gap-3 sm:grid-cols-2">
        <TextField label="达成阈值 *" value={draft.threshold_value} onChange={(value) => onDraftChange("threshold_value", value)} type="number" />
        <TextField label="积分奖励" value={draft.points_reward} onChange={(value) => onDraftChange("points_reward", value)} type="number" />
        <TextField label="提现加成百分比" value={draft.creator_bonus_percent} onChange={(value) => onDraftChange("creator_bonus_percent", value)} type="number" />
        <TextField label="加成天数" value={draft.bonus_days} onChange={(value) => onDraftChange("bonus_days", value)} type="number" placeholder="0 为长期" />
      </div>
      <BooleanSelect label="启用" value={draft.is_active} onChange={(value) => onDraftChange("is_active", value)} />
      <TextareaField label="描述" value={draft.description} onChange={(value) => onDraftChange("description", value)} />
    </PointsDrawer>
  );
}


export function PointsProductDrawer({
  row,
  draft,
  saving,
  onDraftChange,
  onClose,
  onSubmit,
}: {
  row: PointsGiftCardProduct | null;
  draft: Record<string, unknown>;
  saving: boolean;
  onDraftChange: (key: string, value: unknown) => void;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  if (row === null) return null;
  const isEdit = row.id !== undefined;
  return (
    <PointsDrawer title={isEdit ? "编辑礼品卡" : "新建礼品卡"} saving={saving} onClose={onClose} onSubmit={onSubmit}>
      <TextField label="礼品卡名称 *" value={draft.name} onChange={(value) => onDraftChange("name", value)} />
      <div className="grid gap-3 sm:grid-cols-2">
        <TextField label="面值" value={draft.face_value} onChange={(value) => onDraftChange("face_value", value)} placeholder="例如 50 元" />
        <TextField label="兑换积分 *" value={draft.points_required} onChange={(value) => onDraftChange("points_required", value)} type="number" />
        <TextField label="排序" value={draft.sort_order} onChange={(value) => onDraftChange("sort_order", value)} type="number" />
        <BooleanSelect label="启用" value={draft.is_active} onChange={(value) => onDraftChange("is_active", value)} />
      </div>
      <TextareaField label="描述" value={draft.description} onChange={(value) => onDraftChange("description", value)} />
    </PointsDrawer>
  );
}


export function PointsImportDrawer({
  product,
  text,
  saving,
  onTextChange,
  onClose,
  onSubmit,
}: {
  product: PointsGiftCardProduct | null;
  text: string;
  saving: boolean;
  onTextChange: (value: string) => void;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  if (!product) return null;
  return (
    <PointsDrawer title={`导入卡密 · ${product.name}`} saving={saving} onClose={onClose} onSubmit={onSubmit} saveLabel="导入">
      <label className="grid gap-1.5">
        <span className="text-xs font-semibold text-[#666c78]">卡密 TXT 内容</span>
        <textarea
          value={text}
          onChange={(event) => onTextChange(event.target.value)}
          className="min-h-[320px] rounded-lg border border-black/[0.08] bg-[#fafbfe] p-3 font-mono text-sm outline-none focus:border-[#1d4ed8]"
          placeholder={"CODE-001\nCODE-002\nCODE-003"}
        />
      </label>
      <div className="rounded-lg border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
        每行一个卡密，重复卡密会自动跳过。用户兑换成功后会立即看到卡密。
      </div>
    </PointsDrawer>
  );
}


export function PointsDrawer({
  title,
  saving,
  saveLabel = "保存",
  children,
  onClose,
  onSubmit,
}: {
  title: string;
  saving: boolean;
  saveLabel?: string;
  children: ReactNode;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <div className="fixed inset-0 z-50">
      <button type="button" aria-label="关闭积分抽屉遮罩" className="absolute inset-0 bg-[#17171d]/28" onClick={onClose} />
      <aside className="absolute inset-y-0 right-0 flex w-full max-w-[560px] flex-col bg-white shadow-2xl">
        <div className="flex h-16 shrink-0 items-center gap-3 border-b border-black/[0.06] px-4">
          <h2 className="min-w-0 flex-1 truncate text-base font-semibold text-[#17171d]">{title}</h2>
          <Button type="button" size="icon" variant="ghost" onClick={onClose} className="size-10 rounded-lg">
            <X className="size-5" />
          </Button>
        </div>
        <form onSubmit={onSubmit} className="flex min-h-0 flex-1 flex-col">
          <div className="min-h-0 flex-1 overflow-y-auto p-4">
            <div className="grid gap-3">{children}</div>
          </div>
          <div className="flex shrink-0 justify-end gap-2 border-t border-black/[0.06] p-4">
            <Button type="button" variant="outline" onClick={onClose} className="rounded-lg border-black/[0.08] bg-white hover:bg-[#f6f7fb]">取消</Button>
            <Button type="submit" disabled={saving} className="rounded-lg bg-[#1d4ed8] hover:bg-[#1e40af]">
              {saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
              {saveLabel}
            </Button>
          </div>
        </form>
      </aside>
    </div>
  );
}
