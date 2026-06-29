import {
  cancelAIJob,
  createAIJob,
  getAIPublishGenerationSettings,
  streamAIJob,
} from "@/lib/api";
import type {
  AIJobPayload,
  AIPublishGenerationConfig,
  AIPublishGenerationResult,
  UploadAsset,
} from "@/lib/types";
import { useLocale } from "next-intl";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { queueFromAIJob, type QueueState } from "./ai-job-queue";
import { parsePublishGenerationPreview } from "./ai-publish-generation-parser";

type PublishGenerationPhase = "idle" | "queued" | "running" | "done" | "error" | "canceled";
type PublishGenerationTask = "detail" | "title";

export type PublishGenerationState = {
  enabled: boolean;
  loading: boolean;
  config: AIPublishGenerationConfig | null;
  phase: PublishGenerationPhase;
  stage: string;
  percent: number;
  activeField: PublishGenerationTask | null;
  generatedTitle: string;
  generatedDetail: string;
  reasoning: string;
  error: string;
  queue: QueueState | null;
  canCancel: boolean;
};

export type PublishGenerationRequest = {
  locale: string;
  title: string;
  body: string;
  tags: string[];
  images: UploadAsset[];
};

export type PublishGenerationApplyResult = {
  title: string;
  body: string;
  result: AIPublishGenerationResult | null;
};

export type PublishGenerationRunInput = Omit<PublishGenerationRequest, "locale">;

type PublishGenerationJobResult = {
  text: string;
  job: AIJobPayload;
};

const publishGenerationRunningStatuses = new Set(["queued", "running"]);
const detailProgressWeight = 0.75;

export const defaultPublishGenerationConfig: AIPublishGenerationConfig = {
  enabled: true,
  detail: { enabled: true, templateKey: "publish_detail_generate" },
  title: { enabled: true, templateKey: "publish_title_generate" },
  combined: { enabled: false, templateKey: "publish_title_detail_generate" },
  maxImages: 3,
  imageSelectionMode: "ordered",
  titleMaxChars: 40,
};

export async function loadPublishGenerationConfig() {
  try {
    return normalizePublishGenerationConfig(await getAIPublishGenerationSettings());
  } catch {
    return defaultPublishGenerationConfig;
  }
}

export function publishGenerationHasRemoteImages(images: UploadAsset[]) {
  return images.some((asset) => isModelReachableImage(asset));
}

export function usePublishGeneration() {
  const locale = useLocale();
  const generationRef = useRef(0);
  const abortRef = useRef<AbortController | null>(null);
  const activeJobIdRef = useRef("");
  const [publishGeneration, setPublishGeneration] = useState<PublishGenerationState>({
    enabled: true,
    loading: true,
    config: null,
    phase: "idle",
    stage: "",
    percent: 0,
    activeField: null,
    generatedTitle: "",
    generatedDetail: "",
    reasoning: "",
    error: "",
    queue: null,
    canCancel: false,
  });

  useEffect(() => {
    let cancelled = false;
    void loadPublishGenerationConfig().then((config) => {
      if (!cancelled) {
        setPublishGeneration((current) => ({
          ...current,
          loading: false,
          config,
        }));
      }
    });
    return () => {
      cancelled = true;
      abortRef.current?.abort();
    };
  }, []);

  const activeConfig = normalizePublishGenerationConfig(publishGeneration.config);
  const runPublishGenerationStream = useCallback(
    async (input: PublishGenerationRunInput) => {
      const config = normalizePublishGenerationConfig(publishGeneration.config);
      if (!publishGeneration.enabled || !config.enabled) {
        return { title: input.title, body: input.body, result: null };
      }
      const images = publishGenerationImages(input.images, config);
      if (images.length === 0) {
        return { title: input.title, body: input.body, result: null };
      }
      const tasks: PublishGenerationTask[] = [];
      if (publishGenerationNeedsDetail(input.body, config)) {
        tasks.push("detail");
      }
      if (config.title.enabled) {
        tasks.push("title");
      }
      if (tasks.length === 0) {
        return { title: input.title, body: input.body, result: null };
      }

      abortRef.current?.abort();
      const generation = generationRef.current + 1;
      generationRef.current = generation;
      setPublishGeneration((current) => ({
        ...current,
        phase: "queued",
        stage: "queued",
        percent: 2,
        activeField: tasks[0],
        generatedTitle: "",
        generatedDetail: "",
        reasoning: "",
        error: "",
        queue: null,
        canCancel: true,
      }));

      let nextTitle = input.title;
      let nextBody = input.body;
      const result: AIPublishGenerationResult = {
        enabled: true,
        generatedTitle: false,
        generatedDetail: false,
        maxImages: config.maxImages,
        imageSelectionMode: config.imageSelectionMode,
        titleMaxChars: config.titleMaxChars,
        imageSendSuccessCount: images.length,
        skipped: {},
      };

      try {
        for (const task of tasks) {
          if (generationRef.current !== generation) {
            break;
          }
          setPublishGeneration((current) => ({
            ...current,
            activeField: task,
            phase: "queued",
            stage: "queued",
            percent: Math.max(current.percent, task === "detail" ? 2 : 76),
            queue: null,
            reasoning: "",
          }));
          const taskInput = { ...input, title: nextTitle, body: nextBody };
          const jobResult = await runPublishGenerationJob({
            config,
            field: task,
            images: task === "detail" ? images : [],
            input: taskInput,
            locale,
            onController: (controller) => {
              abortRef.current = controller;
            },
            onJobId: (jobId) => {
              activeJobIdRef.current = jobId;
            },
            onPreview: (preview) => {
              if (generationRef.current !== generation) return;
              const text = publishGenerationTaskOutput(preview, task, config.titleMaxChars);
              if (task === "title") {
                setPublishGeneration((current) => ({ ...current, generatedTitle: text }));
                return;
              }
              setPublishGeneration((current) => ({ ...current, generatedDetail: text }));
            },
            onReasoning: (reasoning) => {
              if (generationRef.current !== generation) return;
              setPublishGeneration((current) => ({ ...current, reasoning }));
            },
            onState: (patch) => {
              if (generationRef.current !== generation) return;
              setPublishGeneration((current) => ({
                ...current,
                ...patch,
                percent: combinePublishGenerationPercent(task, patch.percent ?? current.percent),
              }));
            },
          });
          const text = publishGenerationTaskOutput(jobResult.text, task, config.titleMaxChars);
          if (!text) {
            throw new Error("error.ai_empty_output");
          }
          if (task === "detail" && text) {
            nextBody = text;
            result.detail = text;
            result.generatedDetail = true;
            setPublishGeneration((current) => ({ ...current, generatedDetail: text }));
          }
          if (task === "title" && text) {
            nextTitle = text;
            result.title = text;
            result.generatedTitle = true;
            setPublishGeneration((current) => ({ ...current, generatedTitle: text }));
          }
        }
        if (generationRef.current === generation) {
          setPublishGeneration((current) => ({
            ...current,
            phase: "done",
            stage: "completed",
            percent: 100,
            activeField: null,
            queue: null,
            canCancel: false,
          }));
        }
        return { title: nextTitle, body: nextBody, result };
      } catch (error) {
        if (isAbortError(error)) {
          setPublishGeneration((current) => ({
            ...current,
            phase: "canceled",
            stage: "canceled",
            canCancel: false,
            activeField: null,
            queue: null,
          }));
          return { title: nextTitle, body: nextBody, result: null };
        }
        const message = error instanceof Error ? error.message : "error.ai_request_failed";
        setPublishGeneration((current) => ({
          ...current,
          phase: "error",
          stage: "error",
          error: message,
          canCancel: false,
          activeField: null,
          queue: null,
        }));
        throw error;
      } finally {
        if (generationRef.current === generation) {
          abortRef.current = null;
          activeJobIdRef.current = "";
        }
      }
    },
    [locale, publishGeneration.config, publishGeneration.enabled],
  );

  const cancelPublishGeneration = useCallback(async () => {
    const jobId = activeJobIdRef.current;
    generationRef.current += 1;
    abortRef.current?.abort();
    abortRef.current = null;
    if (jobId) {
      try {
        await cancelAIJob(jobId);
      } catch {
        // Best effort; the stream abort already returns control to the user.
      }
    }
    setPublishGeneration((current) => ({
      ...current,
      phase: "canceled",
      stage: "canceled",
      activeField: null,
      queue: null,
      canCancel: false,
    }));
  }, []);

  const helpers = useMemo(
    () => ({
      activeConfig,
      canUseImages: (images: UploadAsset[]) => publishGenerationHasRemoteImages(images),
      needsTitle: () => Boolean(activeConfig.title.enabled),
      needsDetail: (body: string) => publishGenerationNeedsDetail(body, activeConfig),
      selectedImageCount: (images: UploadAsset[]) => publishGenerationImages(images, activeConfig).length,
    }),
    [activeConfig],
  );

  return {
    cancelPublishGeneration,
    publishGeneration,
    runPublishGenerationStream,
    setPublishGeneration,
    ...helpers,
  };
}

async function runPublishGenerationJob({
  config,
  field,
  images,
  input,
  locale,
  onController,
  onJobId,
  onPreview,
  onReasoning,
  onState,
}: {
  config: AIPublishGenerationConfig;
  field: PublishGenerationTask;
  images: ReturnType<typeof publishGenerationImages>;
  input: PublishGenerationRunInput;
  locale: string;
  onController: (controller: AbortController | null) => void;
  onJobId: (jobId: string) => void;
  onPreview: (preview: string) => void;
  onReasoning: (reasoning: string) => void;
  onState: (patch: Partial<PublishGenerationState>) => void;
}): Promise<PublishGenerationJobResult> {
  const job = await createAIJob({
    type: publishGenerationTaskType(field),
    locale,
    input: publishGenerationPromptInput(field, input, images.length, config),
    templateKey: publishGenerationTemplateKey(config, field),
    requestHash: publishGenerationRequestHash(field, input, images, config),
    variables: publishGenerationVariables(field, input, images.length, config),
    images,
  });
  onJobId(job.jobId);
  const applyJobPreview = (nextJob: AIJobPayload) => {
    const output = publishGenerationTaskOutput(nextJob.output ?? "", field, config.titleMaxChars);
    if (output) {
      onPreview(output);
    }
    if (nextJob.reasoning) {
      onReasoning(nextJob.reasoning);
    }
  };
  onState({
    phase: phaseFromPublishGenerationJob(job),
    stage: job.stage || "queued",
    percent: job.percent ?? 0,
    queue: queueFromAIJob(job),
  });
  applyJobPreview(job);
  if (!publishGenerationRunningStatuses.has(job.status)) {
    if (job.status === "completed") {
      return { text: job.output ?? "", job };
    }
    throw new Error(publishGenerationJobError(job));
  }

  const controller = new AbortController();
  onController(controller);
  let latestJob = job;
  try {
    const streamedJob = await streamAIJob(job.jobId, {
      onConnected: () => {
        onState({ stage: "connected", phase: "running" });
      },
      onJob: (nextJob) => {
        latestJob = nextJob;
        onState({
          phase: phaseFromPublishGenerationJob(nextJob),
          stage: nextJob.stage || "",
          percent: nextJob.percent ?? 0,
          queue: queueFromAIJob(nextJob),
        });
        applyJobPreview(nextJob);
      },
      onError: (event) => {
        throw new Error(event.message || event.code || "error.ai_request_failed");
      },
    }, { signal: controller.signal });
    latestJob = streamedJob ?? latestJob;
  } finally {
    onController(null);
  }
  if (latestJob.status !== "completed") {
    throw new Error(publishGenerationJobError(latestJob));
  }
  return { text: latestJob.output ?? "", job: latestJob };
}

function normalizePublishGenerationConfig(config: AIPublishGenerationConfig | null | undefined): AIPublishGenerationConfig {
  return {
    ...defaultPublishGenerationConfig,
    ...config,
    detail: {
      ...defaultPublishGenerationConfig.detail,
      ...config?.detail,
      enabled: config?.detail?.enabled ?? true,
    },
    title: {
      ...defaultPublishGenerationConfig.title,
      ...config?.title,
      enabled: config?.title?.enabled ?? true,
    },
    combined: {
      ...defaultPublishGenerationConfig.combined,
      ...config?.combined,
    },
    maxImages: clampNumber(config?.maxImages, 0, 12, defaultPublishGenerationConfig.maxImages),
    imageSelectionMode: config?.imageSelectionMode === "random" ? "random" : "ordered",
    titleMaxChars: clampNumber(config?.titleMaxChars, 8, 80, defaultPublishGenerationConfig.titleMaxChars),
  };
}

function publishGenerationTemplateKey(config: AIPublishGenerationConfig, field: PublishGenerationTask) {
  return field === "title" ? config.title.templateKey : config.detail.templateKey;
}

function publishGenerationTaskType(field: PublishGenerationTask) {
  return field === "title" ? "publish_title_generate" : "publish_detail_generate";
}

function publishGenerationJobError(job: AIJobPayload) {
  const code = job.errorCode || "error.ai_request_failed";
  const detail = (job.upstreamDetail || job.errorMessage || "").trim();
  return detail ? `${code}: ${detail}` : code;
}

function publishGenerationNeedsDetail(_body: string, config: AIPublishGenerationConfig) {
  return Boolean(config.detail.enabled);
}

function publishGenerationImages(images: UploadAsset[], config: AIPublishGenerationConfig) {
  const maxImages = Math.max(0, config.maxImages);
  if (maxImages <= 0) return [];
  const reachable = images.filter((asset) => isModelReachableImage(asset));
  const selected = config.imageSelectionMode === "random"
    ? shufflePublishGenerationImages(reachable).slice(0, maxImages)
    : reachable.slice(0, maxImages);
  return selected.map((asset, index) => ({
    url: asset.signedUrl || asset.url,
    mime: asset.contentType,
    alt: asset.originalname || `image ${index + 1}`,
  }));
}

function isModelReachableImage(asset: UploadAsset) {
  const url = asset.signedUrl || asset.url;
  return Boolean(url && !url.startsWith("blob:"));
}

function publishGenerationVariables(
  field: PublishGenerationTask,
  input: PublishGenerationRunInput,
  imageCount: number,
  config: AIPublishGenerationConfig,
) {
  if (field === "title") {
    return {
      existingTitle: input.title.trim(),
      detail: input.body.trim(),
      titleMaxChars: config.titleMaxChars,
    };
  }
  return {
    existingTitle: input.title.trim(),
    existingDetail: input.body.trim(),
    imageSendSuccessCount: imageCount,
    imageSelectionMode: config.imageSelectionMode,
  };
}

function publishGenerationPromptInput(
  field: PublishGenerationTask,
  input: PublishGenerationRunInput,
  imageCount: number,
  config: AIPublishGenerationConfig,
) {
  if (field === "title") {
    const lines = [
      `已有标题：\n${input.title.trim() || "（空）"}`,
      `最终详情正文：\n${input.body.trim() || "（空）"}`,
      `标题字数上限：${config.titleMaxChars}`,
    ];
    const tags = normalizedTags(input.tags);
    if (tags.length > 0) {
      lines.push(`标签/话题：${tags.join(", ")}`);
    }
    return lines.join("\n\n");
  }
  const lines = [
    `已有标题：\n${input.title.trim() || "（空）"}`,
    `已有详情正文：\n${input.body.trim() || "（空）"}`,
  ];
  const tags = normalizedTags(input.tags);
  if (tags.length > 0) {
    lines.push(`标签/话题：${tags.join(", ")}`);
  }
  lines.push(`可用于分析的图片数量：${imageCount}`);
  lines.push(`图片取样模式：${config.imageSelectionMode}`);
  return lines.join("\n\n");
}

function phaseFromPublishGenerationJob(job: AIJobPayload): PublishGenerationPhase {
  switch (job.status) {
    case "queued":
      return "queued";
    case "running":
      return "running";
    case "completed":
      return "done";
    case "canceled":
      return "canceled";
    case "failed":
      return "error";
    default:
      return "running";
  }
}

function combinePublishGenerationPercent(task: PublishGenerationTask, jobPercent: number) {
  const normalized = Math.max(0, Math.min(100, jobPercent)) / 100;
  if (task === "detail") {
    return Math.min(75, Math.round(normalized * detailProgressWeight * 100));
  }
  return Math.min(99, Math.round(75 + normalized * 25));
}

function publishGenerationRequestHash(
  field: PublishGenerationTask,
  input: PublishGenerationRunInput,
  images: ReturnType<typeof publishGenerationImages>,
  config: AIPublishGenerationConfig,
) {
  const seed = JSON.stringify({
    field,
    title: input.title.trim(),
    body: input.body.trim(),
    tags: input.tags,
    images: images.map((image) => image.url),
    imageSelectionMode: config.imageSelectionMode,
    titleMaxChars: config.titleMaxChars,
  });
  let hash = 0;
  for (let index = 0; index < seed.length; index += 1) {
    hash = (hash * 31 + seed.charCodeAt(index)) | 0;
  }
  return `publish-generation:${field}:${Math.abs(hash)}`;
}

function cleanPublishGenerationText(value: string) {
  return value
    .trim()
    .replace(/^```(?:markdown|text)?\s*/i, "")
    .replace(/```$/i, "")
    .trim();
}

function publishGenerationTaskOutput(value: string, field: PublishGenerationTask, titleMaxChars: number) {
  const parsed = parsePublishGenerationPreview(value, field);
  const cleaned = cleanPublishGenerationText(value);
  const text = parsed || (isStructuredPublishGenerationOutput(cleaned) ? "" : cleaned);
  return field === "title" ? limitPublishGenerationTitle(text, titleMaxChars) : text;
}

function limitPublishGenerationTitle(value: string, maxChars: number) {
  const text = cleanPublishGenerationText(value).replace(/^["'`]+|["'`]+$/g, "").trim();
  const limit = clampNumber(maxChars, 8, 80, 40);
  const runes = Array.from(text);
  return runes.length > limit ? runes.slice(0, limit).join("").trim() : text;
}

function isStructuredPublishGenerationOutput(value: string) {
  const trimmed = value.trim();
  return (
    trimmed.startsWith("{") ||
    trimmed.startsWith("[") ||
    /(^|\n)\s*(标题|Title|title|详情|正文|Detail|Details|Body|detail|details|body)\s*[:：]/.test(trimmed)
  );
}

function normalizedTags(tags: string[]) {
  return tags.map((tag) => tag.trim()).filter(Boolean);
}

function shufflePublishGenerationImages<T>(items: T[]) {
  const out = [...items];
  for (let index = out.length - 1; index > 0; index -= 1) {
    const target = Math.floor(Math.random() * (index + 1));
    [out[index], out[target]] = [out[target], out[index]];
  }
  return out;
}

function clampNumber(value: unknown, min: number, max: number, fallback: number) {
  const parsed = Number(value ?? fallback);
  if (!Number.isFinite(parsed)) return fallback;
  return Math.max(min, Math.min(max, parsed));
}

function isAbortError(error: unknown) {
  return error instanceof DOMException && error.name === "AbortError";
}
