"use client";

import {
  CircleDollarSign,
  ClipboardList,
  Edit3,
  Loader2,
  Package,
  Plus,
  ReceiptText,
  RotateCcw,
  Save,
  SlidersHorizontal,
  Star,
  Trash2,
  UploadCloud,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import type { PointsAchievementRule, PointsGiftCardProduct, PointsTaskConfig } from "@/lib/types";
import { achievementTriggerLabel } from "./settings-panel";
import { HeaderCard, MetricTile, Panel } from "./layout-widgets";
import { EmptyBlock, LoadingBlock } from "./resource-editor";
import { StatusPill } from "./resource-cells";
import { InfoTile } from "./operations-widgets";
import { formatDateTime, formatMoney, truthy } from "./helpers";
import { PointsImportDrawer, PointsProductDrawer, PointsRuleDrawer, PointsTaskDrawer } from "./points-panel-drawers";
import { pointsProductDraft, pointsRuleDraft, pointsTaskDraft, taskPeriodLabel } from "./points-panel-model";
import type { PointsPanelController } from "./use-points-panel-controller";

export function PointsPanelView({ controller }: { controller: PointsPanelController }) {
  const {
    stats,
    settings,
    tasks,
    rules,
    products,
    redemptions,
    dailyCapDraft,
    setDailyCapDraft,
    taskDraft,
    setTaskDraft,
    editingTask,
    setEditingTask,
    ruleDraft,
    setRuleDraft,
    editingRule,
    setEditingRule,
    productDraft,
    setProductDraft,
    editingProduct,
    setEditingProduct,
    importProduct,
    setImportProduct,
    importText,
    setImportText,
    loading,
    saving,
    saveSettings,
    clearAllPoints,
    resetTaskProgress,
    saveTask,
    deleteTask,
    saveRule,
    deleteRule,
    saveProduct,
    deleteProduct,
    importCodes,
  } = controller;

    return (
    <div className="grid gap-4">
      <HeaderCard icon={CircleDollarSign} title="积分运营" description="每日任务、提现加成、礼品卡库存与兑换记录" tone="green" />
      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <MetricTile label="积分用户" value={stats?.total_users ?? 0} tone="blue" />
        <MetricTile label="存量积分" value={formatMoney(stats?.total_points ?? 0)} tone="green" />
        <MetricTile label="今日发放" value={formatMoney(stats?.today_awarded ?? 0)} tone="amber" />
        <MetricTile label="可用卡密" value={stats?.available_cards ?? 0} tone="purple" />
      </section>

      <Panel
        title="每日上限"
        icon={SlidersHorizontal}
        action={
          <Button type="button" disabled={saving === "settings"} onClick={() => void saveSettings()} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]">
            {saving === "settings" ? <Loader2 className="size-4 animate-spin" /> : <Save className="size-4" />}
            保存
          </Button>
        }
      >
        <div className="grid gap-3 md:grid-cols-[220px_minmax(0,1fr)]">
          <label className="block">
            <span className="mb-1.5 block text-xs font-semibold text-[#5f636d]">每日最多获得积分</span>
            <input value={dailyCapDraft} onChange={(event) => setDailyCapDraft(event.target.value)} type="number" min="0" className="h-10 w-full rounded-lg border border-black/[0.08] bg-[#fafbfe] px-3 text-sm outline-none focus:border-[#1d4ed8]" />
          </label>
          <div className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3 text-sm text-[#666c78]">
            当前每日上限 {formatMoney(settings?.daily_cap ?? Number(dailyCapDraft || 0))} 分；单个任务仍按下面的每日次数限制分别控制。
          </div>
        </div>
      </Panel>

      <Panel title="积分维护" icon={RotateCcw}>
        <div className="grid gap-3 md:grid-cols-2">
          <div className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
            <p className="text-sm font-semibold text-[#343944]">清空全部积分</p>
            <p className="mt-1 text-xs leading-5 text-[#6f7582]">将所有用户当前积分归零，并为非零余额用户写入一条管理员清空日志。</p>
            <Button type="button" disabled={saving === "clear-points"} onClick={() => void clearAllPoints()} variant="outline" className="mt-3 rounded-lg border-[#dc2626]/20 bg-white text-[#b91c1c] hover:bg-[#fef2f2]">
              {saving === "clear-points" ? <Loader2 className="size-4 animate-spin" /> : <Trash2 className="size-4" />}
              清空全部积分
            </Button>
          </div>
          <div className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-3">
            <p className="text-sm font-semibold text-[#343944]">重置所有任务</p>
            <p className="mt-1 text-xs leading-5 text-[#6f7582]">清空任务完成记录、每日统计、成就奖励和提现加成，任务配置本身不会删除。</p>
            <Button type="button" disabled={saving === "reset-tasks"} onClick={() => void resetTaskProgress()} variant="outline" className="mt-3 rounded-lg border-black/[0.08] bg-white hover:bg-[#f6f7fb]">
              {saving === "reset-tasks" ? <Loader2 className="size-4 animate-spin" /> : <RotateCcw className="size-4" />}
              重置所有任务
            </Button>
          </div>
        </div>
      </Panel>

      <Panel title="任务配置" icon={ClipboardList} action={<Button type="button" onClick={() => { setEditingTask({} as PointsTaskConfig); setTaskDraft(pointsTaskDraft(null)); }} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]"><Plus className="size-4" />新增</Button>}>
        {loading ? (
          <LoadingBlock label="正在加载任务配置" />
        ) : tasks.length ? (
          <div className="grid gap-3 lg:grid-cols-2 xl:grid-cols-3">
            {tasks.map((task) => (
              <article key={String(task.id ?? task.task_type)} className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-4">
                <div className="mb-3 flex min-w-0 items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h3 className="truncate text-sm font-semibold text-[#17171d]">{task.name || task.task_type}</h3>
                    <p className="mt-1 truncate text-xs text-[#8b919e]">{task.task_type}</p>
                  </div>
                  <StatusPill value={truthy(task.is_active) ? "启用" : "停用"} tone={truthy(task.is_active) ? "green" : "slate"} />
                </div>
                <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-3">
                  <InfoTile label="积分" value={`+${formatMoney(task.points)}`} />
                  <InfoTile label="周期" value={taskPeriodLabel(task)} />
                  <InfoTile label={truthy(task.is_daily_task) ? "每日次数" : "完成上限"} value={String(task.daily_limit ?? 1)} />
                </div>
                <p className="mb-3 line-clamp-2 min-h-10 text-xs leading-5 text-[#6f7582]">{task.description || "无描述"}</p>
                <div className="flex flex-wrap gap-2">
                  <Button type="button" size="sm" variant="outline" onClick={() => { setEditingTask(task); setTaskDraft(pointsTaskDraft(task)); }} className="rounded-lg border-black/[0.08] bg-white"><Edit3 className="size-4" />编辑</Button>
                  <Button type="button" size="sm" variant="outline" disabled={saving === `task-${task.id}`} onClick={() => void deleteTask(task)} className="rounded-lg border-[#dc2626]/20 bg-white text-[#b91c1c] hover:bg-[#fef2f2]"><Trash2 className="size-4" />删除</Button>
                </div>
              </article>
            ))}
          </div>
        ) : (
          <EmptyBlock icon={ClipboardList} label="暂无任务配置" />
        )}
      </Panel>

      <Panel title="成就与提现加成" icon={Star} action={<Button type="button" onClick={() => { setEditingRule({} as PointsAchievementRule); setRuleDraft(pointsRuleDraft(null)); }} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]"><Plus className="size-4" />新增</Button>}>
        {loading ? (
          <LoadingBlock label="正在加载成就规则" />
        ) : rules.length ? (
          <div className="grid gap-3 lg:grid-cols-2 xl:grid-cols-3">
            {rules.map((rule) => (
              <article key={String(rule.id)} className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-4">
                <div className="mb-3 flex min-w-0 items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h3 className="truncate text-sm font-semibold text-[#17171d]">{rule.name}</h3>
                    <p className="mt-1 truncate text-xs text-[#8b919e]">{achievementTriggerLabel(rule.trigger_type)} ≥ {rule.threshold_value}</p>
                  </div>
                  <StatusPill value={truthy(rule.is_active) ? "启用" : "停用"} tone={truthy(rule.is_active) ? "green" : "slate"} />
                </div>
                <div className="mb-3 grid grid-cols-2 gap-2">
                  <InfoTile label="积分奖励" value={formatMoney(rule.points_reward)} />
                  <InfoTile label="提现加成" value={`${formatMoney(rule.creator_bonus_percent)}%`} />
                </div>
                <p className="mb-3 line-clamp-2 min-h-10 text-xs leading-5 text-[#6f7582]">{rule.description || "提现加成在创作者提现时额外到账，不改变销售结算记录"}</p>
                <div className="flex flex-wrap gap-2">
                  <Button type="button" size="sm" variant="outline" onClick={() => { setEditingRule(rule); setRuleDraft(pointsRuleDraft(rule)); }} className="rounded-lg border-black/[0.08] bg-white"><Edit3 className="size-4" />编辑</Button>
                  <Button type="button" size="sm" variant="outline" disabled={saving === `rule-${rule.id}`} onClick={() => void deleteRule(rule)} className="rounded-lg border-[#dc2626]/20 bg-white text-[#b91c1c] hover:bg-[#fef2f2]"><Trash2 className="size-4" />删除</Button>
                </div>
              </article>
            ))}
          </div>
        ) : (
          <EmptyBlock icon={Star} label="暂无成就规则" />
        )}
      </Panel>

      <Panel title="礼品卡商品" icon={Package} action={<Button type="button" onClick={() => { setEditingProduct({} as PointsGiftCardProduct); setProductDraft(pointsProductDraft(null)); }} className="h-9 rounded-lg bg-[#1d4ed8] px-3 hover:bg-[#1e40af]"><Plus className="size-4" />新增</Button>}>
        {loading ? (
          <LoadingBlock label="正在加载礼品卡" />
        ) : products.length ? (
          <div className="grid gap-3 lg:grid-cols-2 xl:grid-cols-3">
            {products.map((product) => (
              <article key={String(product.id)} className="rounded-lg border border-black/[0.06] bg-[#fafbfe] p-4">
                <div className="mb-3 flex min-w-0 items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h3 className="truncate text-sm font-semibold text-[#17171d]">{product.name}</h3>
                    <p className="mt-1 truncate text-xs text-[#8b919e]">{product.face_value || "礼品卡"}</p>
                  </div>
                  <StatusPill value={truthy(product.is_active) ? "启用" : "停用"} tone={truthy(product.is_active) ? "green" : "slate"} />
                </div>
                <div className="mb-3 grid grid-cols-2 gap-2">
                  <InfoTile label="兑换积分" value={formatMoney(product.points_required)} />
                  <InfoTile label="库存" value={String(product.available_stock ?? 0)} />
                </div>
                <p className="mb-3 line-clamp-2 min-h-10 text-xs leading-5 text-[#6f7582]">{product.description || "无描述"}</p>
                <div className="flex flex-wrap gap-2">
                  <Button type="button" size="sm" variant="outline" onClick={() => { setEditingProduct(product); setProductDraft(pointsProductDraft(product)); }} className="rounded-lg border-black/[0.08] bg-white"><Edit3 className="size-4" />编辑</Button>
                  <Button type="button" size="sm" variant="outline" onClick={() => setImportProduct(product)} className="rounded-lg border-black/[0.08] bg-white"><UploadCloud className="size-4" />导入卡密</Button>
                  <Button type="button" size="sm" variant="outline" disabled={saving === `product-${product.id}`} onClick={() => void deleteProduct(product)} className="rounded-lg border-[#dc2626]/20 bg-white text-[#b91c1c] hover:bg-[#fef2f2]"><Trash2 className="size-4" />删除</Button>
                </div>
              </article>
            ))}
          </div>
        ) : (
          <EmptyBlock icon={Package} label="暂无礼品卡商品" />
        )}
      </Panel>

      <Panel title="兑换记录" icon={ReceiptText}>
        {loading ? (
          <LoadingBlock label="正在加载兑换记录" />
        ) : redemptions?.list?.length ? (
          <div className="overflow-x-auto">
            <table className="min-w-[720px] w-full text-left text-sm">
              <thead className="text-xs text-[#8a8f9d]">
                <tr>
                  <th className="px-3 py-2 font-semibold">用户</th>
                  <th className="px-3 py-2 font-semibold">礼品卡</th>
                  <th className="px-3 py-2 font-semibold">卡密</th>
                  <th className="px-3 py-2 font-semibold">积分</th>
                  <th className="px-3 py-2 font-semibold">时间</th>
                </tr>
              </thead>
              <tbody>
                {redemptions.list.map((item) => (
                  <tr key={String(item.id)} className="border-t border-black/[0.06]">
                    <td className="px-3 py-3 text-[#30333b]">{String(item.user_id ?? "-")}</td>
                    <td className="px-3 py-3 text-[#30333b]">{item.product?.name || String(item.product_id)}</td>
                    <td className="max-w-[260px] truncate px-3 py-3 font-mono text-xs text-[#30333b]">{item.code || "-"}</td>
                    <td className="px-3 py-3 text-[#30333b]">{formatMoney(item.points_spent)}</td>
                    <td className="px-3 py-3 text-[#666c78]">{formatDateTime(item.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <EmptyBlock icon={ReceiptText} label="暂无兑换记录" />
        )}
      </Panel>

      <PointsTaskDrawer row={editingTask} draft={taskDraft} saving={saving === "task"} onDraftChange={(key, value) => setTaskDraft((current) => ({ ...current, [key]: value }))} onClose={() => setEditingTask(null)} onSubmit={(event) => void saveTask(event)} />
      <PointsRuleDrawer row={editingRule} draft={ruleDraft} saving={saving === "rule"} onDraftChange={(key, value) => setRuleDraft((current) => ({ ...current, [key]: value }))} onClose={() => setEditingRule(null)} onSubmit={(event) => void saveRule(event)} />
      <PointsProductDrawer row={editingProduct} draft={productDraft} saving={saving === "product"} onDraftChange={(key, value) => setProductDraft((current) => ({ ...current, [key]: value }))} onClose={() => setEditingProduct(null)} onSubmit={(event) => void saveProduct(event)} />
      <PointsImportDrawer product={importProduct} text={importText} saving={saving === "import"} onTextChange={setImportText} onClose={() => setImportProduct(null)} onSubmit={(event) => void importCodes(event)} />
    </div>
  );
}
