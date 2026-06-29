"use client";

import type { FormEvent } from "react";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { adminRequest } from "@/lib/api";
import type {
  PointsAchievementRule,
  PointsAdminSettingsPayload,
  PointsAdminStatsPayload,
  PointsGiftCardImportPayload,
  PointsGiftCardProduct,
  PointsMaintenancePayload,
  PointsRedemptionsPayload,
  PointsTaskConfig,
} from "@/lib/types";
import { errorMessage, formatCompact } from "./helpers";
import {
  pointsProductDraft,
  pointsProductPayload,
  pointsRuleDraft,
  pointsRulePayload,
  pointsTaskDraft,
  pointsTaskPayload,
} from "./points-panel-model";

export function usePointsPanelController({ token }: { token: string }) {

  const [stats, setStats] = useState<PointsAdminStatsPayload | null>(null);
  const [settings, setSettings] = useState<PointsAdminSettingsPayload | null>(null);
  const [tasks, setTasks] = useState<PointsTaskConfig[]>([]);
  const [rules, setRules] = useState<PointsAchievementRule[]>([]);
  const [products, setProducts] = useState<PointsGiftCardProduct[]>([]);
  const [redemptions, setRedemptions] = useState<PointsRedemptionsPayload | null>(null);
  const [dailyCapDraft, setDailyCapDraft] = useState("50");
  const [taskDraft, setTaskDraft] = useState<Record<string, unknown>>(() => pointsTaskDraft(null));
  const [editingTask, setEditingTask] = useState<PointsTaskConfig | null>(null);
  const [ruleDraft, setRuleDraft] = useState<Record<string, unknown>>(() => pointsRuleDraft(null));
  const [editingRule, setEditingRule] = useState<PointsAchievementRule | null>(null);
  const [productDraft, setProductDraft] = useState<Record<string, unknown>>(() => pointsProductDraft(null));
  const [editingProduct, setEditingProduct] = useState<PointsGiftCardProduct | null>(null);
  const [importProduct, setImportProduct] = useState<PointsGiftCardProduct | null>(null);
  const [importText, setImportText] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [statsData, settingsData, tasksData, rulesData, productsData, redemptionsData] = await Promise.all([
        adminRequest<PointsAdminStatsPayload>("/api/points/admin/stats", { method: "GET", token }),
        adminRequest<PointsAdminSettingsPayload>("/api/points/admin/settings", { method: "GET", token }),
        adminRequest<{ list?: PointsTaskConfig[] }>("/api/points/admin/tasks", { method: "GET", token }),
        adminRequest<{ list?: PointsAchievementRule[] }>("/api/points/admin/achievement-rules", { method: "GET", token }),
        adminRequest<{ list?: PointsGiftCardProduct[] }>("/api/points/admin/gift-card-products", { method: "GET", token }),
        adminRequest<PointsRedemptionsPayload>("/api/points/admin/redemptions", { method: "GET", token, query: { page: 1, limit: 8 } }),
      ]);
      setStats(statsData);
      setSettings(settingsData);
      setDailyCapDraft(String(settingsData.daily_cap ?? 50));
      setTasks(tasksData.list ?? []);
      setRules(rulesData.list ?? []);
      setProducts(productsData.list ?? []);
      setRedemptions(redemptionsData);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    queueMicrotask(() => {
      void load();
    });
  }, [load]);

  async function saveSettings() {
    setSaving("settings");
    try {
      await adminRequest("/api/points/admin/settings", { method: "PUT", token, body: JSON.stringify({ daily_cap: Number(dailyCapDraft) }) });
      toast.success("积分上限已保存");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(null);
    }
  }

  async function clearAllPoints() {
    if (!window.confirm("确认清空所有用户当前积分？该操作会写入清空日志，但无法自动恢复余额。")) return;
    setSaving("clear-points");
    try {
      const result = await adminRequest<PointsMaintenancePayload>("/api/points/admin/clear-balances", {
        method: "POST",
        token,
        body: JSON.stringify({ reason: "管理员清空全部积分" }),
      });
      toast.success(`已清空 ${formatCompact(result.affected_users ?? 0)} 个用户的积分`);
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(null);
    }
  }

  async function resetTaskProgress() {
    if (!window.confirm("确认重置所有积分任务进度？用户可重新完成每日任务、固定任务和成就任务。")) return;
    setSaving("reset-tasks");
    try {
      const result = await adminRequest<PointsMaintenancePayload>("/api/points/admin/reset-task-progress", { method: "POST", token });
      toast.success(`已重置任务记录 ${formatCompact((result.deleted_events ?? 0) + (result.deleted_stats ?? 0) + (result.deleted_achievements ?? 0))} 条`);
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(null);
    }
  }

  async function saveTask(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving("task");
    try {
      const body = pointsTaskPayload(taskDraft);
      if (editingTask?.id) {
        await adminRequest(`/api/points/admin/tasks/${encodeURIComponent(String(editingTask.id))}`, { method: "PUT", token, body: JSON.stringify(body) });
        toast.success("任务已更新");
      } else {
        await adminRequest("/api/points/admin/tasks", { method: "POST", token, body: JSON.stringify(body) });
        toast.success("任务已创建");
      }
      setEditingTask(null);
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(null);
    }
  }

  async function deleteTask(row: PointsTaskConfig) {
    if (!row.id || !window.confirm(`确认删除任务 ${row.name || row.task_type}？`)) return;
    setSaving(`task-${row.id}`);
    try {
      await adminRequest(`/api/points/admin/tasks/${encodeURIComponent(String(row.id))}`, { method: "DELETE", token });
      toast.success("任务已删除");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(null);
    }
  }

  async function saveRule(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving("rule");
    try {
      const body = pointsRulePayload(ruleDraft);
      if (editingRule?.id) {
        await adminRequest(`/api/points/admin/achievement-rules/${encodeURIComponent(String(editingRule.id))}`, { method: "PUT", token, body: JSON.stringify(body) });
        toast.success("成就规则已更新");
      } else {
        await adminRequest("/api/points/admin/achievement-rules", { method: "POST", token, body: JSON.stringify(body) });
        toast.success("成就规则已创建");
      }
      setEditingRule(null);
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(null);
    }
  }

  async function deleteRule(row: PointsAchievementRule) {
    if (!row.id || !window.confirm(`确认删除规则 ${row.name}？`)) return;
    setSaving(`rule-${row.id}`);
    try {
      await adminRequest(`/api/points/admin/achievement-rules/${encodeURIComponent(String(row.id))}`, { method: "DELETE", token });
      toast.success("成就规则已删除");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(null);
    }
  }

  async function saveProduct(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSaving("product");
    try {
      const body = pointsProductPayload(productDraft);
      if (editingProduct?.id) {
        await adminRequest(`/api/points/admin/gift-card-products/${encodeURIComponent(String(editingProduct.id))}`, { method: "PUT", token, body: JSON.stringify(body) });
        toast.success("礼品卡已更新");
      } else {
        await adminRequest("/api/points/admin/gift-card-products", { method: "POST", token, body: JSON.stringify(body) });
        toast.success("礼品卡已创建");
      }
      setEditingProduct(null);
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(null);
    }
  }

  async function deleteProduct(row: PointsGiftCardProduct) {
    if (!row.id || !window.confirm(`确认删除礼品卡 ${row.name}？`)) return;
    setSaving(`product-${row.id}`);
    try {
      await adminRequest(`/api/points/admin/gift-card-products/${encodeURIComponent(String(row.id))}`, { method: "DELETE", token });
      toast.success("礼品卡已删除");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(null);
    }
  }

  async function importCodes(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!importProduct?.id) return;
    setSaving("import");
    try {
      const result = await adminRequest<PointsGiftCardImportPayload>(`/api/points/admin/gift-card-products/${encodeURIComponent(String(importProduct.id))}/import-codes`, {
        method: "POST",
        token,
        body: JSON.stringify({ text: importText }),
      });
      toast.success(`已导入 ${formatCompact(result.imported)} 个，跳过 ${formatCompact(result.skipped)} 个`);
      setImportText("");
      setImportProduct(null);
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setSaving(null);
    }
  }

  return {
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
    load,
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
  };
}

export type PointsPanelController = ReturnType<typeof usePointsPanelController>;
