"use client";
import type {
  FormEvent
} from "react";
import {
  ClipboardList,
  Loader2,
  Megaphone,
  Save,
  X
} from "lucide-react";
import {
  Button
} from "@/components/ui/button";
import type {
  AdminListPayload,
  AdminListRow
} from "@/lib/types";
import {
  CouponAdminItem,
  PickerSelection,
  couponIssueTargetOptions,
  couponTypeOptions
} from "./types";
import {
  EmptyBlock
} from "./resource-editor";
import {
  AdminObjectPicker,
  TextField
} from "./object-picker";
import {
  StatusPill
} from "./resource-cells";
import {
  InfoTile
} from "./operations-widgets";
import {
  couponUsageStatusLabel,
  couponUsageUser,
  couponValueLabel,
  fieldText,
  formatCompact,
  formatDateTime,
  readableValue
} from "./helpers";
import {
  BooleanSelect,
  SelectField
} from "./form-fields";

export function CouponEditorDrawer({
  row,
  draft,
  saving,
  onDraftChange,
  onClose,
  onSubmit,
}: {
  row: CouponAdminItem | null;
  draft: Record<string, unknown>;
  saving: boolean;
  onDraftChange: (key: string, value: unknown) => void;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  if (row === null) return null;
  const isEdit = row.id !== undefined;
  return (
    <div className="fixed inset-0 z-50">
      <button type="button" aria-label="关闭优惠券编辑遮罩" className="absolute inset-0 bg-[#17171d]/28" onClick={onClose} />
      <aside className="absolute inset-y-0 right-0 flex w-full max-w-[560px] flex-col bg-white shadow-2xl">
        <div className="flex h-16 shrink-0 items-center gap-3 border-b border-black/[0.06] px-4">
          <h2 className="min-w-0 flex-1 truncate text-base font-semibold text-[#17171d]">{isEdit ? "编辑优惠券" : "新建优惠券"}</h2>
          <Button type="button" size="icon" variant="ghost" onClick={onClose} className="size-10 rounded-lg">
            <X className="size-5" />
          </Button>
        </div>
        <form onSubmit={onSubmit} className="flex min-h-0 flex-1 flex-col">
          <div className="min-h-0 flex-1 overflow-y-auto p-4">
            <div className="grid gap-3">
              <TextField label="名称 *" value={draft.name} onChange={(value) => onDraftChange("name", value)} />
              <TextField label="券码" value={draft.code} onChange={(value) => onDraftChange("code", value)} placeholder="留空由后端自动生成" />
              <SelectField label="类型 *" value={draft.type ?? "amount"} options={couponTypeOptions} onChange={(value) => onDraftChange("type", value)} />
              <div className="grid gap-3 sm:grid-cols-2">
                <TextField label="优惠值 *" value={draft.value} onChange={(value) => onDraftChange("value", value)} type="number" />
                <TextField label="最低订单" value={draft.min_order} onChange={(value) => onDraftChange("min_order", value)} type="number" />
                <TextField label="最高优惠" value={draft.max_discount} onChange={(value) => onDraftChange("max_discount", value)} type="number" placeholder="可为空" />
                <TextField label="总库存" value={draft.total_count} onChange={(value) => onDraftChange("total_count", value)} type="number" placeholder="-1 为不限" />
              </div>
              <div className="grid gap-3 sm:grid-cols-2">
                <TextField label="开始时间 *" value={draft.start_time} onChange={(value) => onDraftChange("start_time", value)} type="datetime-local" />
                <TextField label="结束时间 *" value={draft.end_time} onChange={(value) => onDraftChange("end_time", value)} type="datetime-local" />
              </div>
              <BooleanSelect label="启用" value={draft.is_active} onChange={(value) => onDraftChange("is_active", value)} />
              <label className="grid gap-1.5">
                <span className="text-xs font-semibold text-[#666c78]">描述</span>
                <textarea value={String(draft.description ?? "")} onChange={(event) => onDraftChange("description", event.target.value)} className="min-h-[110px] rounded-lg border border-black/[0.08] bg-[#fafbfe] p-3 text-sm outline-none focus:border-[#1d4ed8]" />
              </label>
            </div>
          </div>
          <div className="flex shrink-0 justify-end gap-2 border-t border-black/[0.06] p-4">
            <Button type="button" variant="outline" onClick={onClose} className="rounded-lg border-black/[0.08] bg-white hover:bg-[#f6f7fb]">取消</Button>
            <Button type="submit" disabled={saving} className="rounded-lg bg-[#1d4ed8] hover:bg-[#1e40af]">
              {saving ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
              保存
            </Button>
          </div>
        </form>
      </aside>
    </div>
  );
}


export function CouponIssueDrawer({
  token,
  coupon,
  target,
  users,
  acting,
  onTargetChange,
  onUsersChange,
  onClose,
  onSubmit,
}: {
  token: string;
  coupon: CouponAdminItem | null;
  target: string;
  users: PickerSelection[];
  acting: boolean;
  onTargetChange: (value: string) => void;
  onUsersChange: (value: PickerSelection[]) => void;
  onClose: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
}) {
  if (!coupon) return null;
  return (
    <div className="fixed inset-0 z-50">
      <button type="button" aria-label="关闭优惠券发放遮罩" className="absolute inset-0 bg-[#17171d]/28" onClick={onClose} />
      <aside className="absolute inset-y-0 right-0 flex w-full max-w-[460px] flex-col bg-white shadow-2xl">
        <div className="flex h-16 shrink-0 items-center gap-3 border-b border-black/[0.06] px-4">
          <h2 className="min-w-0 flex-1 truncate text-base font-semibold text-[#17171d]">发放优惠券</h2>
          <Button type="button" size="icon" variant="ghost" onClick={onClose} className="size-10 rounded-lg">
            <X className="size-5" />
          </Button>
        </div>
        <form onSubmit={onSubmit} className="flex min-h-0 flex-1 flex-col">
          <div className="min-h-0 flex-1 overflow-y-auto p-4">
            <div className="mb-3 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
              <p className="text-sm font-semibold text-[#17171d]">{coupon.name || `优惠券 ${coupon.id}`}</p>
              <p className="mt-1 text-xs text-[#8b919e]">{couponValueLabel(coupon)} · {coupon.code || "自动券码"}</p>
            </div>
            <div className="grid gap-3">
              <SelectField label="发放对象" value={target} options={couponIssueTargetOptions} onChange={onTargetChange} />
              {target === "users" ? (
                <AdminObjectPicker
                  token={token}
                  resource="users"
                  label="指定用户"
                  multiple
                  value={users}
                  onChange={onUsersChange}
                  placeholder="搜索昵称、账号或邮箱"
                  emptyLabel="未找到用户"
                />
              ) : null}
            </div>
          </div>
          <div className="flex shrink-0 justify-end gap-2 border-t border-black/[0.06] p-4">
            <Button type="button" variant="outline" onClick={onClose} className="rounded-lg border-black/[0.08] bg-white hover:bg-[#f6f7fb]">取消</Button>
            <Button type="submit" disabled={acting} className="rounded-lg bg-[#1d4ed8] hover:bg-[#1e40af]">
              {acting ? <Loader2 className="size-4 animate-spin" /> : <Megaphone className="size-4" />}
              发放
            </Button>
          </div>
        </form>
      </aside>
    </div>
  );
}


export function CouponUsageDrawer({ coupon, payload, onClose }: { coupon: CouponAdminItem | null; payload: AdminListPayload<AdminListRow> | null; onClose: () => void }) {
  if (!coupon) return null;
  const rows = payload?.items ?? [];
  return (
    <div className="fixed inset-0 z-50">
      <button type="button" aria-label="关闭优惠券记录遮罩" className="absolute inset-0 bg-[#17171d]/28" onClick={onClose} />
      <aside className="absolute inset-y-0 right-0 flex w-full max-w-[560px] flex-col bg-white shadow-2xl">
        <div className="flex h-16 shrink-0 items-center gap-3 border-b border-black/[0.06] px-4">
          <h2 className="min-w-0 flex-1 truncate text-base font-semibold text-[#17171d]">领取使用记录</h2>
          <Button type="button" size="icon" variant="ghost" onClick={onClose} className="size-10 rounded-lg">
            <X className="size-5" />
          </Button>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto p-4">
          <div className="mb-3 rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
            <p className="text-sm font-semibold text-[#17171d]">{coupon.name || `优惠券 ${coupon.id}`}</p>
            <p className="mt-1 text-xs text-[#8b919e]">共 {formatCompact(payload?.pagination.total ?? rows.length)} 条记录</p>
          </div>
          {rows.length ? (
            <div className="grid gap-2">
              {rows.map((row) => (
                <article key={String(row.id)} className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
                  <div className="mb-2 flex min-w-0 items-center justify-between gap-3">
                    <div className="min-w-0">
                      <p className="truncate text-sm font-semibold text-[#252932]">{couponUsageUser(row)}</p>
                      <p className="truncate text-xs text-[#8b919e]">账号 {fieldText(row, "user_id")}</p>
                    </div>
                    <StatusPill value={couponUsageStatusLabel(row.status)} tone={String(row.status) === "used" ? "green" : "amber"} />
                  </div>
                  <div className="grid gap-2 sm:grid-cols-3">
                    <InfoTile label="领取" value={formatDateTime(row.created_at)} />
                    <InfoTile label="使用" value={formatDateTime(row.used_at)} />
                    <InfoTile label="发放人" value={readableValue(row.issued_by)} />
                  </div>
                </article>
              ))}
            </div>
          ) : (
            <EmptyBlock icon={ClipboardList} label="暂无领取记录" />
          )}
        </div>
      </aside>
    </div>
  );
}
