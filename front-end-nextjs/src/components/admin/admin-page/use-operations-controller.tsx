import type {
  FormEvent
} from "react";
import {
  useCallback,
  useEffect,
  useState
} from "react";
import {
  toast
} from "sonner";
import {
  adminRequest
} from "@/lib/api";
import type {
  AdminListRow
} from "@/lib/types";
import type {
  ActivityItem,
  ApkFileItem,
  AppVersionStatsPayload,
  BatchUploadFilesPayload,
  ContentReviewSettingsPayload,
  MissingCoverStats,
  OperationResult,
  PickerSelection
} from "./types";
import {
  stripFileExtension
} from "./resource-editor";
import {
  firstPickerID,
  pickerIDs
} from "./object-picker";
import {
  errorMessage,
  fieldText,
  formatCompact
} from "./helpers";

export function useOperationsController({ token }: { token: string }) {
  const [activities, setActivities] = useState<ActivityItem[]>([]);
  const [apkFiles, setApkFiles] = useState<ApkFileItem[]>([]);
  const [missingCovers, setMissingCovers] = useState<MissingCoverStats | null>(null);
  const [batchFiles, setBatchFiles] = useState<BatchUploadFilesPayload | null>(null);
  const [appStats, setAppStats] = useState<AppVersionStatsPayload | null>(null);
  const [reviewSettings, setReviewSettings] = useState<ContentReviewSettingsPayload | null>(null);
  const [recommendConfig, setRecommendConfig] = useState<Record<string, unknown>>({});
  const [templateDefaults, setTemplateDefaults] = useState<AdminListRow[]>([]);
  const [testUsers, setTestUsers] = useState<AdminListRow[]>([]);
  const [lastAppForm, setLastAppForm] = useState<Record<string, unknown>>({});
  const [bannedWordsDraft, setBannedWordsDraft] = useState("");
  const [pushPosts, setPushPosts] = useState<PickerSelection[]>([]);
  const [pushScore, setPushScore] = useState("10");
  const [coverLimit, setCoverLimit] = useState("50");
  const [postBatchPosts, setPostBatchPosts] = useState<PickerSelection[]>([]);
  const [postBatchCategory, setPostBatchCategory] = useState<PickerSelection[]>([]);
  const [postTransferUser, setPostTransferUser] = useState<PickerSelection[]>([]);
  const [qualityBatchPosts, setQualityBatchPosts] = useState<PickerSelection[]>([]);
  const [qualityBatchLevel, setQualityBatchLevel] = useState("medium");
  const [recommendBatchPosts, setRecommendBatchPosts] = useState<PickerSelection[]>([]);
  const [recommendBatchBoost, setRecommendBatchBoost] = useState("10");
  const [recommendBatchPinned, setRecommendBatchPinned] = useState(false);
  const [recommendBatchSuppressed, setRecommendBatchSuppressed] = useState(false);
  const [recommendBatchReason, setRecommendBatchReason] = useState("后台批量推荐");
  const [earningsUsers, setEarningsUsers] = useState<PickerSelection[]>([]);
  const [earningsAmount, setEarningsAmount] = useState("");
  const [templateSubject, setTemplateSubject] = useState("测试通知");
  const [templateContent, setTemplateContent] = useState("你好，{{nickname}}，这是一条 {{siteName}} 测试通知。");
  const [batchUsers, setBatchUsers] = useState<PickerSelection[]>([]);
  const [batchTitle, setBatchTitle] = useState("");
  const [batchContent, setBatchContent] = useState("");
  const [batchStatusId, setBatchStatusId] = useState("");
  const [operationResult, setOperationResult] = useState<OperationResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState<string | null>(null);
  const [operationTab, setOperationTab] = useState<"system" | "content" | "growth" | "assets" | "finance">("system");

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [activityData, apkData, coverData, batchData, appData, reviewData, recommendData, defaultsData, usersData, appFormData] = await Promise.allSettled([
        adminRequest<{ activities?: unknown[] }>("/api/admin/monitor/activities", { method: "GET", token }),
        adminRequest<{ files?: unknown[] }>("/api/admin/apk-files", { method: "GET", token }),
        adminRequest<MissingCoverStats>("/api/admin/videos/missing-covers/stats", { method: "GET", token }),
        adminRequest<BatchUploadFilesPayload>("/api/admin/batch-upload/files", { method: "GET", token }),
        adminRequest<AppVersionStatsPayload>("/api/admin/app-versions/stats", { method: "GET", token }),
        adminRequest<ContentReviewSettingsPayload>("/api/admin/content-review/settings", { method: "GET", token }),
        adminRequest<Record<string, unknown>>("/api/admin/recommendation/config", { method: "GET", token }),
        adminRequest<AdminListRow[]>("/api/admin/notification-templates/defaults", { method: "GET", token }),
        adminRequest<AdminListRow[]>("/api/admin/test-users", { method: "GET", token }),
        adminRequest<Record<string, unknown>>("/api/admin/app-versions/last-form-data", { method: "GET", token }),
      ]);
      setActivities(activityData.status === "fulfilled" ? (activityData.value.activities ?? []) as ActivityItem[] : []);
      setApkFiles(apkData.status === "fulfilled" ? (apkData.value.files ?? []) as ApkFileItem[] : []);
      setMissingCovers(coverData.status === "fulfilled" ? coverData.value : null);
      setBatchFiles(batchData.status === "fulfilled" ? batchData.value : null);
      setAppStats(appData.status === "fulfilled" ? appData.value : null);
      setReviewSettings(reviewData.status === "fulfilled" ? reviewData.value : null);
      setRecommendConfig(recommendData.status === "fulfilled" && recommendData.value && typeof recommendData.value === "object" ? recommendData.value : {});
      setTemplateDefaults(defaultsData.status === "fulfilled" && Array.isArray(defaultsData.value) ? defaultsData.value : []);
      setTestUsers(usersData.status === "fulfilled" && Array.isArray(usersData.value) ? usersData.value : []);
      setLastAppForm(appFormData.status === "fulfilled" && appFormData.value && typeof appFormData.value === "object" ? appFormData.value : {});
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => {
    queueMicrotask(() => {
      void load();
    });
  }, [load]);

  async function generateCovers() {
    if (!window.confirm("确认生成缺失视频封面？")) return;
    setActing("covers");
    try {
      await adminRequest("/api/admin/videos/generate-missing-covers", { method: "POST", token, body: JSON.stringify({ limit: Number(coverLimit) || 50 }) });
      toast.success("已提交封面生成任务");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function updateReviewSettings(next: ContentReviewSettingsPayload) {
    setActing("review");
    try {
      const updated = await adminRequest<ContentReviewSettingsPayload>("/api/admin/content-review/settings", { method: "PUT", token, body: JSON.stringify(next) });
      setReviewSettings(updated);
      toast.success("审核设置已更新");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function saveRecommendConfig(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setActing("recommend-config");
    try {
      await adminRequest("/api/admin/recommendation/config", { method: "PUT", token, body: JSON.stringify(recommendConfig) });
      toast.success("推荐配置已保存");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function pushRecommendation(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const postID = firstPickerID(pushPosts);
    if (!postID) {
      toast.error("请选择要推荐的帖子");
      return;
    }
    setActing("push");
    try {
      await adminRequest("/api/admin/recommendation/push", { method: "POST", token, body: JSON.stringify({ post_id: postID, boost_score: Number(pushScore) || 10, reason: "运营控制台主动推荐" }) });
      setPushPosts([]);
      toast.success("推荐已生效");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function importBannedWords() {
    const words = bannedWordsDraft.split(/\r?\n|,|，/).map((word) => word.trim()).filter(Boolean);
    if (!words.length) {
      toast.error("请输入要导入的敏感词");
      return;
    }
    setActing("banned-import");
    try {
      const result = await adminRequest<{ count?: number }>("/api/admin/banned-words/import", { method: "POST", token, body: JSON.stringify({ words }) });
      setBannedWordsDraft("");
      toast.success(`已导入 ${formatCompact(result.count ?? words.length)} 个敏感词`);
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function exportBannedWords() {
    setActing("banned-export");
    try {
      const rows = await adminRequest<AdminListRow[]>("/api/admin/banned-words/export", { method: "GET", token });
      toast.success(`已读取 ${formatCompact(Array.isArray(rows) ? rows.length : 0)} 个敏感词`);
      setBannedWordsDraft(Array.isArray(rows) ? rows.map((row) => fieldText(row, "word")).filter((word) => word !== "-").join("\n") : "");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function runPostBatch(action: "private" | "public" | "category" | "transfer") {
    const ids = pickerIDs(postBatchPosts);
    if (!ids.length) {
      toast.error("请选择要操作的帖子");
      return;
    }
    setActing(`post-${action}`);
    try {
      const payload: Record<string, unknown> = { ids };
      let path = "/api/admin/posts/set-private";
      let method = "PUT";
      if (action === "public") path = "/api/admin/posts/set-public";
      if (action === "category") {
        const categoryID = firstPickerID(postBatchCategory);
        if (!categoryID) {
          toast.error("请选择分类");
          return;
        }
        path = "/api/admin/posts/set-category";
        payload.category_id = categoryID;
      }
      if (action === "transfer") {
        const targetUser = postTransferUser[0];
        if (!targetUser?.displayId) {
          toast.error("请选择目标作者");
          return;
        }
        path = "/api/admin/posts/transfer";
        method = "POST";
        payload.target_user_display_id = targetUser.displayId;
      }
      const result = await adminRequest<Record<string, unknown>>(path, { method, token, body: JSON.stringify(payload) });
      setOperationResult({ title: "内容批量操作", message: "操作已完成", values: result });
      toast.success("内容批量操作已完成");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function runQualityBatch() {
    const ids = pickerIDs(qualityBatchPosts);
    if (!ids.length) {
      toast.error("请选择要标记的帖子");
      return;
    }
    setActing("quality-batch");
    try {
      const result = await adminRequest<Record<string, unknown>>("/api/admin/posts-quality/batch", {
        method: "PUT",
        token,
        body: JSON.stringify({ post_ids: ids, quality_level: qualityBatchLevel }),
      });
      setOperationResult({ title: "内容质量批量标记", message: "质量标记已完成", values: result });
      toast.success("内容质量已批量更新");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function runRecommendationBatch() {
    const ids = pickerIDs(recommendBatchPosts);
    if (!ids.length) {
      toast.error("请选择要配置推荐的帖子");
      return;
    }
    setActing("recommend-batch");
    try {
      const result = await adminRequest<Record<string, unknown>>("/api/admin/recommendation/post-configs/batch", {
        method: "POST",
        token,
        body: JSON.stringify({
          post_ids: ids,
          boost_score: Number(recommendBatchBoost) || 0,
          is_pinned: recommendBatchPinned,
          is_suppressed: recommendBatchSuppressed,
          is_active: true,
          reason: recommendBatchReason,
        }),
      });
      setOperationResult({ title: "推荐规则批量写入", message: "推荐规则已保存", values: result });
      toast.success("推荐规则已批量保存");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function runEarnings(action: "info" | "add" | "deduct" | "transfer") {
    const userId = firstPickerID(earningsUsers);
    if (!userId) {
      toast.error("请选择用户");
      return;
    }
    setActing(`earnings-${action}`);
    try {
      const endpoint = action === "info"
        ? `/api/admin/users/${encodeURIComponent(userId)}/earnings-info`
        : `/api/admin/users/${encodeURIComponent(userId)}/${action === "add" ? "add-earnings" : action === "deduct" ? "deduct-earnings" : "transfer-to-earnings"}`;
      const result = await adminRequest<Record<string, unknown>>(endpoint, {
        method: action === "info" ? "GET" : "POST",
        token,
        body: action === "info" ? undefined : JSON.stringify({ amount: Number(earningsAmount) || 0 }),
      });
      setOperationResult({ title: "收益调账", message: "操作已完成", values: result });
      toast.success("收益操作已完成");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function previewTemplate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setActing("template-preview");
    try {
      const result = await adminRequest<Record<string, unknown>>("/api/admin/notification-templates/preview", {
        method: "POST",
        token,
        body: JSON.stringify({ subject: templateSubject, content: templateContent }),
      });
      setOperationResult({ title: "通知模板预览", message: "模板已渲染", values: result });
      toast.success("模板预览已生成");
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function createBatchPosts() {
    const files = [...(batchFiles?.images ?? []), ...(batchFiles?.videos ?? [])];
    const userID = firstPickerID(batchUsers);
    if (!userID || !files.length) {
      toast.error("请选择发布用户，并确认已有批量素材");
      return;
    }
    setActing("batch-create");
    try {
      const result = await adminRequest<Record<string, unknown>>("/api/admin/batch-upload/create", {
        method: "POST",
        token,
        body: JSON.stringify({
          user_id: userID,
          type: (batchFiles?.videos ?? []).length ? 2 : 1,
          title: batchTitle,
          content: batchContent,
          files,
          is_draft: true,
        }),
      });
      setOperationResult({ title: "批量素材发布", message: "创建请求已完成", values: result });
      toast.success("批量素材发布已执行");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function createBatchPostsAsync() {
    const files = [...(batchFiles?.images ?? []), ...(batchFiles?.videos ?? [])];
    const userID = firstPickerID(batchUsers);
    if (!userID || !files.length) {
      toast.error("请选择发布用户，并确认已有批量素材");
      return;
    }
    setActing("batch-async-create");
    try {
      const notes = files.map((file) => ({
        title: batchTitle || stripFileExtension(file.name ?? "批量素材"),
        content: batchContent,
        files: [file],
      }));
      const result = await adminRequest<Record<string, unknown>>("/api/admin/batch-upload/async-create", {
        method: "POST",
        token,
        body: JSON.stringify({
          user_id: userID,
          type: (batchFiles?.videos ?? []).length ? 2 : 1,
          notes,
          is_draft: true,
        }),
      });
      setOperationResult({ title: "批量素材异步发布", message: "已提交异步创建", values: result });
      toast.success("批量素材已加入异步创建");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function deleteBatchFiles(kind: "all" | "images" | "videos") {
    const files = kind === "images" ? (batchFiles?.images ?? []) : kind === "videos" ? (batchFiles?.videos ?? []) : [...(batchFiles?.images ?? []), ...(batchFiles?.videos ?? [])];
    if (!files.length) {
      toast.error("暂无可删除的批量素材");
      return;
    }
    if (!window.confirm(`确认删除 ${files.length} 个批量素材文件？`)) return;
    setActing(`batch-delete-${kind}`);
    try {
      const result = await adminRequest<Record<string, unknown>>("/api/admin/batch-upload/files", {
        method: "DELETE",
        token,
        body: JSON.stringify({ files }),
      });
      setOperationResult({ title: "批量素材清理", message: "文件删除已完成", values: result });
      toast.success("批量素材已删除");
      await load();
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  async function checkBatchStatus() {
    if (!batchStatusId.trim()) {
      toast.error("请输入批次编号");
      return;
    }
    setActing("batch-status");
    try {
      const result = await adminRequest<Record<string, unknown>>(`/api/admin/batch-upload/status/${encodeURIComponent(batchStatusId.trim())}`, { method: "GET", token });
      setOperationResult({ title: "批次状态", message: "状态已读取", values: result });
    } catch (error) {
      toast.error(errorMessage(error));
    } finally {
      setActing(null);
    }
  }

  return {
    activities,
    acting,
    apkFiles,
    appStats,
    bannedWordsDraft,
    batchContent,
    batchFiles,
    batchStatusId,
    batchTitle,
    batchUsers,
    checkBatchStatus,
    coverLimit,
    createBatchPosts,
    createBatchPostsAsync,
    deleteBatchFiles,
    earningsAmount,
    earningsUsers,
    exportBannedWords,
    generateCovers,
    importBannedWords,
    lastAppForm,
    loading,
    missingCovers,
    operationResult,
    operationTab,
    postBatchCategory,
    postBatchPosts,
    postTransferUser,
    previewTemplate,
    pushPosts,
    pushRecommendation,
    pushScore,
    qualityBatchLevel,
    qualityBatchPosts,
    recommendBatchBoost,
    recommendBatchPinned,
    recommendBatchPosts,
    recommendBatchReason,
    recommendBatchSuppressed,
    recommendConfig,
    reviewSettings,
    runEarnings,
    runPostBatch,
    runQualityBatch,
    runRecommendationBatch,
    saveRecommendConfig,
    setBannedWordsDraft,
    setBatchContent,
    setBatchStatusId,
    setBatchTitle,
    setBatchUsers,
    setCoverLimit,
    setEarningsAmount,
    setEarningsUsers,
    setOperationTab,
    setPostBatchCategory,
    setPostBatchPosts,
    setPostTransferUser,
    setPushPosts,
    setPushScore,
    setQualityBatchLevel,
    setQualityBatchPosts,
    setRecommendBatchBoost,
    setRecommendBatchPinned,
    setRecommendBatchPosts,
    setRecommendBatchReason,
    setRecommendBatchSuppressed,
    setRecommendConfig,
    setTemplateContent,
    setTemplateSubject,
    templateContent,
    templateDefaults,
    templateSubject,
    testUsers,
    token,
    updateReviewSettings,
  };
}
