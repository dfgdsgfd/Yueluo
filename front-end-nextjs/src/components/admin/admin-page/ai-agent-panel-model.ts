import type { useTranslations } from "next-intl";
import type {
  AICommentReplyConfig,
  AIModerationConfig,
  AIPublicSettings,
  AISettingsUpdate,
  AIPublishGenerationConfig,
  AIContentFormatConfig,
} from "@/lib/types";

export type AIAgentPanelT = ReturnType<typeof useTranslations<"adminPortal.aiAgentPanel">>;

export type AIDraftSettings = AIPublicSettings & {
  apiKey?: string;
  clearApiKey?: boolean;
};

export function toDraftSettings(settings: AIPublicSettings): AIDraftSettings {
  const autoComment = settings.autoComment ?? {
    enabled: false,
    botUserId: 0,
    botUserIdMin: 0,
    botUserIdMax: 0,
    templateKey: "post_auto_comment",
    delaySeconds: 10,
    maxImages: 4,
    imageSelectionMode: "ordered",
    style: "normal",
  };
  return {
    ...structuredClone(settings),
    maxRunSeconds: Number(settings.maxRunSeconds ?? 3600),
    showReasoning: Boolean(settings.showReasoning),
    thinkingParameterEnabled: Boolean(settings.thinkingParameterEnabled),
    thinkingEnabled: Boolean(settings.thinkingEnabled),
    reasoningEffort: settings.reasoningEffort || "",
    modelParameters: settings.modelParameters ?? {},
    logHttpDetails: Boolean(settings.logHttpDetails),
    contentFormat: normalizeAIContentFormat(settings.contentFormat),
    moderation: normalizeAIModeration(settings.moderation),
    commentReply: normalizeAICommentReply(settings.commentReply),
    publishGeneration: normalizeAIPublishGeneration(settings.publishGeneration),
    autoComment: {
      enabled: Boolean(autoComment.enabled),
      botUserId: Number(autoComment.botUserId || 0),
      botUserIdMin: Number(autoComment.botUserIdMin || 0),
      botUserIdMax: Number(autoComment.botUserIdMax || 0),
      templateKey: autoComment.templateKey || "post_auto_comment",
      delaySeconds: Number(autoComment.delaySeconds ?? 10),
      maxImages: Number(autoComment.maxImages ?? 4),
      imageSelectionMode: normalizeImageSelectionMode(autoComment.imageSelectionMode),
      style: autoComment.style || "normal",
    },
    apiKey: "",
  };
}

export function normalizeAIContentFormat(config?: AIContentFormatConfig): AIContentFormatConfig {
  return {
    enabled: config?.enabled ?? true,
    format: {
      enabled: config?.format?.enabled ?? true,
      templateKey: config?.format?.templateKey || "markdown_format",
    },
    polish: {
      enabled: config?.polish?.enabled ?? true,
      templateKey: config?.polish?.templateKey || "post_polish",
    },
    custom: {
      enabled: config?.custom?.enabled ?? true,
      templateKey: config?.custom?.templateKey || "post_custom_generate",
      continuation: {
        enabled: config?.custom?.continuation?.enabled ?? true,
        triggerChars: clampNumber(config?.custom?.continuation?.triggerChars, 1000, 100000, 6000),
        maxRounds: clampNumber(config?.custom?.continuation?.maxRounds, 1, 8, 2),
        contextChars: clampNumber(config?.custom?.continuation?.contextChars, 600, 20000, 2400),
      },
    },
  };
}

export function normalizeAICommentReply(config?: AICommentReplyConfig): AICommentReplyConfig {
  return {
    enabled: Boolean(config?.enabled),
    templateKey: config?.templateKey || "comment_reply",
    delaySeconds: clampNumber(config?.delaySeconds, 0, 3600, 10),
    maxImages: clampNumber(config?.maxImages, 0, 12, 4),
    imageSelectionMode: normalizeImageSelectionMode(config?.imageSelectionMode),
    style: config?.style || "normal",
    maxRepliesPerAIComment: clampNumber(config?.maxRepliesPerAIComment, 1, 20, 3),
    mentionEnabled: Boolean(config?.mentionEnabled),
    mentionName: normalizeMentionName(config?.mentionName),
    mentionTemplateKey: config?.mentionTemplateKey || "comment_mention_reply",
    mentionBotUserIdMin: clampNumber(config?.mentionBotUserIdMin, 0, 999999999, 0),
    mentionBotUserIdMax: clampNumber(config?.mentionBotUserIdMax, 0, 999999999, 0),
    maxMentionRepliesPerPost: clampNumber(config?.maxMentionRepliesPerPost, 1, 20, 3),
  };
}

function normalizeMentionName(value: unknown) {
  const text = String(value ?? "yueai").trim().replace(/^@+/, "");
  return text || "yueai";
}

export function normalizeAIPublishGeneration(config?: AIPublishGenerationConfig): AIPublishGenerationConfig {
  return {
    enabled: config?.enabled ?? true,
    detail: {
      enabled: config?.detail?.enabled ?? true,
      templateKey: config?.detail?.templateKey || "publish_detail_generate",
    },
    title: {
      enabled: config?.title?.enabled ?? true,
      templateKey: config?.title?.templateKey || "publish_title_generate",
    },
    combined: {
      enabled: config?.combined?.enabled ?? false,
      templateKey: config?.combined?.templateKey || "publish_title_detail_generate",
    },
    maxImages: clampNumber(config?.maxImages, 0, 12, 3),
    imageSelectionMode: normalizeImageSelectionMode(config?.imageSelectionMode),
    titleMaxChars: clampNumber(config?.titleMaxChars, 8, 80, 40),
  };
}

function normalizeImageSelectionMode(value: unknown) {
  return value === "random" ? "random" : "ordered";
}

export function normalizeAIModeration(moderation?: AIModerationConfig): AIModerationConfig {
  const commentRules = moderation?.comment?.rules ?? {};
  const postRules = moderation?.post?.rules ?? {};
  return {
    comment: {
      enabled: Boolean(moderation?.comment?.enabled),
      templateKey: moderation?.comment?.templateKey || "comment_moderation",
      prompt: moderation?.comment?.prompt ?? "",
      rules: {
        spam: normalizeModerationRule(commentRules.spam, "observe"),
        porn: normalizeModerationRule(commentRules.porn, "delete"),
        political_sensitive: normalizeModerationRule(commentRules.political_sensitive, "delete"),
      },
    },
    post: {
      enabled: Boolean(moderation?.post?.enabled),
      templateKey: moderation?.post?.templateKey || "post_moderation",
      prompt: moderation?.post?.prompt ?? "",
      rules: {
        spam: normalizeModerationRule(postRules.spam, "observe"),
        porn: normalizeModerationRule(postRules.porn, "private"),
        political_sensitive: normalizeModerationRule(postRules.political_sensitive, "private"),
      },
    },
  };
}

function normalizeModerationRule(rule: AIModerationConfig["comment"]["rules"][string] | undefined, defaultAction: string) {
  return {
    enabled: rule?.enabled ?? true,
    action: rule?.action || defaultAction,
    sensitivity: clampModerationSensitivity(rule?.sensitivity),
  };
}

function clampModerationSensitivity(value: unknown) {
  const parsed = Number(value ?? 0.65);
  if (!Number.isFinite(parsed)) return 0.65;
  return Math.max(0, Math.min(1, parsed));
}

function clampNumber(value: unknown, min: number, max: number, fallback: number) {
  const parsed = Number(value ?? fallback);
  if (!Number.isFinite(parsed)) return fallback;
  return Math.max(min, Math.min(max, parsed));
}

export function toSettingsUpdate(draft: AIDraftSettings): AISettingsUpdate {
  const payload: Partial<AIDraftSettings> = { ...draft };
  delete payload.apiKeySet;
  delete payload.apiKeyMasked;
  const { apiKey, clearApiKey, ...rest } = payload;
  return {
    ...rest,
    apiKey: apiKey?.trim() ? apiKey.trim() : undefined,
    clearApiKey,
  };
}

export function formatExtraHeaders(headers: Record<string, string>) {
  if (!headers || Object.keys(headers).length === 0) {
    return "";
  }
  return JSON.stringify(headers, null, 2);
}

export function formatModelParameters(params: Record<string, unknown>) {
  if (!params || Object.keys(params).length === 0) {
    return "";
  }
  return JSON.stringify(params, null, 2);
}

export function parseExtraHeaders(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return {};
  }
  try {
    const parsed = JSON.parse(trimmed) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return null;
    }
    const headers: Record<string, string> = {};
    for (const [key, headerValue] of Object.entries(parsed)) {
      const normalizedKey = key.trim();
      const normalizedValue = String(headerValue ?? "").trim();
      if (normalizedKey && normalizedValue) {
        headers[normalizedKey] = normalizedValue;
      }
    }
    return headers;
  } catch {
    return null;
  }
}

export function parseModelParameters(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return {};
  }
  try {
    const parsed = JSON.parse(trimmed) as unknown;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return null;
    }
    return parsed as Record<string, unknown>;
  } catch {
    return null;
  }
}

export function aiLogTypeLabel(t: AIAgentPanelT, value: string) {
  const key = `logs.types.${value}`;
  return t.has(key) ? t(key) : value;
}

export function aiLogStatusLabel(t: AIAgentPanelT, value: string) {
  const key = `logs.statuses.${value}`;
  return t.has(key) ? t(key) : value;
}

export function aiLogActorLabel(t: AIAgentPanelT, value: string) {
  const key = `logs.actors.${value}`;
  return t.has(key) ? t(key) : value;
}

export function aiJobTypeLabel(t: AIAgentPanelT, value: string) {
  const key = `logs.types.${value}`;
  return t.has(key) ? t(key) : value;
}

export function aiJobStatusLabel(t: AIAgentPanelT, value: string) {
  const key = `jobs.statuses.${value}`;
  return t.has(key) ? t(key) : value;
}

export function formatAdminQueueEta(t: AIAgentPanelT, seconds: number) {
  if (!Number.isFinite(seconds) || seconds < 60) {
    return t("eta.lessThanMinute");
  }
  return t("eta.minutes", { count: Math.ceil(seconds / 60) });
}

export function formatAdminRelativeTime(t: AIAgentPanelT, value?: string | null) {
  if (!value) {
    return t("jobs.time.unknown");
  }
  const date = new Date(value);
  const timestamp = date.getTime();
  if (!Number.isFinite(timestamp)) {
    return t("jobs.time.unknown");
  }
  const seconds = Math.max(0, Math.floor((Date.now() - timestamp) / 1000));
  if (seconds < 60) {
    return t("jobs.time.secondsAgo", { count: seconds });
  }
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) {
    return t("jobs.time.minutesAgo", { count: minutes });
  }
  const hours = Math.floor(minutes / 60);
  if (hours < 24) {
    return t("jobs.time.hoursAgo", { count: hours });
  }
  return t("jobs.time.daysAgo", { count: Math.floor(hours / 24) });
}
