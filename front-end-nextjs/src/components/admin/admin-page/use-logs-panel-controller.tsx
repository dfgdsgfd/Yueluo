"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import { useTranslations } from "next-intl";
import { toast } from "sonner";
import { getAdminAccessLogAnalytics, getAdminAccessLogs, getAdminBalanceAuditLogs, getAdminPointsAuditLogs, getAdminSecurityAuditLogs } from "@/lib/api";
import type {
  AdminAccessLogAnalyticsPayload,
  AdminAccessLogItem,
  AdminBalanceAuditLogItem,
  AdminLogListPayload,
  AdminPointsAuditLogItem,
  AdminSecurityAuditLogItem,
} from "@/lib/types";
import { errorMessage } from "./helpers";
import { initialLogPages } from "./logs-panel-types";
import type { LogRange, LogTab, VisitorMode } from "./logs-panel-types";

export function useLogsPanelController({ token }: { token: string }) {

  const t = useTranslations("adminLogs");
  const searchParams = useSearchParams();
  const initialCategory = searchParams?.get("category")?.trim() ?? "";
  const [range, setRange] = useState<LogRange>("7d");
  const [visitor, setVisitor] = useState<VisitorMode>("exclude");
  const [tab, setTab] = useState<LogTab>(initialCategory ? "security" : "access");
  const [keywordDraft, setKeywordDraft] = useState("");
  const [keyword, setKeyword] = useState("");
  const [anomaliesOnly, setAnomaliesOnly] = useState(false);
  const [pageByTab, setPageByTab] = useState<Record<LogTab, number>>(initialLogPages);
  const [pageSize, setPageSize] = useState(30);
  const [analytics, setAnalytics] = useState<AdminAccessLogAnalyticsPayload | null>(null);
  const [accessLogs, setAccessLogs] = useState<AdminLogListPayload<AdminAccessLogItem> | null>(null);
  const [operationLogs, setOperationLogs] = useState<AdminLogListPayload<AdminAccessLogItem> | null>(null);
  const [securityLogs, setSecurityLogs] = useState<AdminLogListPayload<AdminSecurityAuditLogItem> | null>(null);
  const [systemLogs, setSystemLogs] = useState<AdminLogListPayload<AdminSecurityAuditLogItem> | null>(null);
  const [pointsLogs, setPointsLogs] = useState<AdminLogListPayload<AdminPointsAuditLogItem> | null>(null);
  const [balanceLogs, setBalanceLogs] = useState<AdminLogListPayload<AdminBalanceAuditLogItem> | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const visitorParam = visitor === "all" ? undefined : visitor;
      const baseQuery = { range, visitor: visitorParam, keyword: keyword.trim() || undefined };
      const financeQuery = {
        range,
        keyword: keyword.trim() || undefined,
        anomaly: anomaliesOnly ? "only" : undefined,
      };
      const analyticsQuery = { range, visitor: visitorParam, rankingLimit: 8 };
      const [analyticsData, accessData, operationData, securityData, systemData, pointsData, balanceData] = await Promise.all([
        getAdminAccessLogAnalytics(analyticsQuery, token),
        getAdminAccessLogs({ ...baseQuery, page: pageByTab.access, limit: pageSize }, token),
        getAdminAccessLogs({ ...baseQuery, page: pageByTab.operation, limit: pageSize, behavior_group: "operation" }, token),
        getAdminSecurityAuditLogs({ ...baseQuery, page: pageByTab.security, limit: pageSize, ...(initialCategory ? { category: initialCategory } : { exclude_category: "system" }) }, token),
        getAdminSecurityAuditLogs({ ...baseQuery, page: pageByTab.system, limit: pageSize, category: "system" }, token),
        getAdminPointsAuditLogs({ ...financeQuery, page: pageByTab.points, limit: pageSize }, token),
        getAdminBalanceAuditLogs({ ...financeQuery, page: pageByTab.balance, limit: pageSize }, token),
      ]);
      setAnalytics(analyticsData);
      setAccessLogs(accessData);
      setOperationLogs(operationData);
      setSecurityLogs(securityData);
      setSystemLogs(systemData);
      setPointsLogs(pointsData);
      setBalanceLogs(balanceData);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setLoading(false);
    }
  }, [anomaliesOnly, initialCategory, keyword, pageByTab, pageSize, range, visitor, token]);

  useEffect(() => {
    queueMicrotask(() => void load());
  }, [load]);

  const totals = analytics?.totals ?? {};
  const status = analytics?.status ?? accessLogs?.status ?? securityLogs?.status ?? systemLogs?.status ?? {};
  const chartSeries = useMemo(() => [
    { key: "pv", name: t("pv"), color: "#2563eb" },
    { key: "active_users", name: t("activeUsers"), color: "#16a34a" },
    { key: "unique_ips", name: t("uniqueIps"), color: "#9333ea" },
    { key: "post_views", name: t("postViews"), color: "#f59e0b" },
    { key: "security_events", name: t("securityEvents"), color: "#dc2626" },
  ], [t]);
  const activeLogs = tab === "access"
    ? accessLogs
    : tab === "operation"
      ? operationLogs
      : tab === "security"
        ? securityLogs
        : tab === "system"
          ? systemLogs
          : tab === "points"
            ? pointsLogs
            : balanceLogs;

  function resetPages() {
    setPageByTab(initialLogPages());
  }

  function applySearch() {
    setKeyword(keywordDraft.trim());
    resetPages();
    if (keywordDraft.trim() === keyword && Object.values(pageByTab).every((page) => page === 1)) {
      void load();
    }
  }

  return {
    t,
    range,
    setRange,
    visitor,
    setVisitor,
    tab,
    setTab,
    keywordDraft,
    setKeywordDraft,
    anomaliesOnly,
    setAnomaliesOnly,
    pageByTab,
    setPageByTab,
    pageSize,
    setPageSize,
    analytics,
    accessLogs,
    operationLogs,
    securityLogs,
    systemLogs,
    pointsLogs,
    balanceLogs,
    loading,
    load,
    totals,
    status,
    chartSeries,
    activeLogs,
    resetPages,
    applySearch,
  };
}

export type LogsPanelController = ReturnType<typeof useLogsPanelController>;
