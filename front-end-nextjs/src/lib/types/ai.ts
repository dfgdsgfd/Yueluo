export type AITemplateConfig = {
  enabled: boolean;
  taskType: string;
  prompt: string;
  systemPrompt?: string;
  userPrompt?: string;
  style?: "normal" | "humorous" | "bold" | string;
  model?: string;
  temperature: number;
  maxOutputTokens: number;
  concurrency?: number;
  structuredJson?: boolean;
  supportsVision?: boolean;
  runtimeOverrides?: AIRuntimeOverrides;
};

export type AIRuntimeOverrides = {
  enabled: boolean;
  showReasoning?: boolean;
  thinkingParameterEnabled?: boolean;
  thinkingEnabled?: boolean;
  reasoningEffort?: string;
  modelParameters?: Record<string, unknown>;
};

export type AIAutoCommentConfig = {
  enabled: boolean;
  botUserId: number;
  botUserIdMin: number;
  botUserIdMax: number;
  templateKey: string;
  delaySeconds: number;
  maxImages: number;
  imageSelectionMode?: "ordered" | "random" | string;
  style: "normal" | "humorous" | "bold" | string;
};

export type AICommentReplyConfig = {
  enabled: boolean;
  templateKey: string;
  delaySeconds: number;
  maxImages: number;
  imageSelectionMode?: "ordered" | "random" | string;
  style: "normal" | "humorous" | "bold" | string;
  maxRepliesPerAIComment: number;
  mentionEnabled: boolean;
  mentionName: string;
  mentionTemplateKey: string;
  mentionBotUserIdMin: number;
  mentionBotUserIdMax: number;
  maxMentionRepliesPerPost: number;
};

export type AIModerationRuleConfig = {
  enabled: boolean;
  action: "observe" | "delete" | "private" | string;
  sensitivity: number;
};

export type AIModerationTargetConfig = {
  enabled: boolean;
  templateKey: string;
  prompt?: string;
  rules: Record<string, AIModerationRuleConfig>;
};

export type AIModerationConfig = {
  comment: AIModerationTargetConfig;
  post: AIModerationTargetConfig;
};

export type AIPublishGenerationTargetConfig = {
  enabled: boolean;
  templateKey: string;
};

export type AIPublishGenerationConfig = {
  enabled: boolean;
  detail: AIPublishGenerationTargetConfig;
  title: AIPublishGenerationTargetConfig;
  combined: AIPublishGenerationTargetConfig;
  maxImages: number;
  imageSelectionMode?: "ordered" | "random" | string;
  titleMaxChars: number;
};

export type AIContentFormatTargetConfig = {
  enabled: boolean;
  templateKey: string;
  continuation?: AIContentContinuationConfig;
};

export type AIContentContinuationConfig = {
  enabled: boolean;
  triggerChars: number;
  maxRounds: number;
  contextChars: number;
};

export type AIContentFormatConfig = {
  enabled: boolean;
  format: AIContentFormatTargetConfig;
  polish: AIContentFormatTargetConfig;
  custom: AIContentFormatTargetConfig;
};

export type AIPublicSettings = {
  enabled: boolean;
  baseUrl: string;
  apiKeySet: boolean;
  apiKeyMasked: string;
  model: string;
  extraHeaders: Record<string, string>;
  timeoutSeconds: number;
  maxRunSeconds: number;
  chunkMaxChars: number;
  concurrency: number;
  temperature: number;
  maxOutputTokens: number;
  templates: Record<string, AITemplateConfig>;
  showReasoning: boolean;
  thinkingParameterEnabled: boolean;
  thinkingEnabled: boolean;
  reasoningEffort: string;
  modelParameters: Record<string, unknown>;
  logHttpDetails: boolean;
  contentFormat: AIContentFormatConfig;
  autoComment: AIAutoCommentConfig;
  commentReply: AICommentReplyConfig;
  moderation: AIModerationConfig;
  publishGeneration: AIPublishGenerationConfig;
  defaultTemplates?: Record<string, AITemplateConfig>;
};

export type AISettingsUpdate = Partial<
  Omit<AIPublicSettings, "apiKeySet" | "apiKeyMasked">
> & {
  apiKey?: string;
  clearApiKey?: boolean;
};

export type AIRequestInput = {
  type: string;
  locale?: string;
  input: string;
  templateKey?: string;
  variables?: Record<string, unknown>;
  options?: {
    temperature?: number;
    maxOutputTokens?: number;
    structuredJson?: boolean;
  };
  images?: Array<{
    url?: string;
    dataUrl?: string;
    mime?: string;
    alt?: string;
  }>;
};

export type AIPublishGenerationInput = {
  locale?: string;
  title?: string;
  detail?: string;
  body?: string;
  tags?: string[];
  needTitle: boolean;
  needDetail: boolean;
  images?: Array<{
    url?: string;
    dataUrl?: string;
    mime?: string;
    alt?: string;
  }>;
};

export type AIPublishGenerationResult = {
  enabled: boolean;
  title?: string;
  detail?: string;
  generatedTitle: boolean;
  generatedDetail: boolean;
  skipped?: Record<string, string>;
  maxImages: number;
  imageSelectionMode?: string;
  titleMaxChars?: number;
  imageSendSuccessCount: number;
  usage?: Record<string, {
    promptTokens?: number;
    completionTokens?: number;
    totalTokens?: number;
  }>;
};

export type AIModerationDebugInput = {
  targetType: "comment" | "post" | string;
  content: string;
  templateKey?: string;
  systemPrompt?: string;
  userPrompt?: string;
  prompt?: string;
  config?: AIModerationTargetConfig;
};

export type AIModerationDebugResult = {
  targetType: string;
  templateKey: string;
  promptInput: string;
  rawOutput: string;
  status: string;
  action: string;
  decision: Record<string, unknown>;
  usage?: {
    promptTokens?: number;
    completionTokens?: number;
    totalTokens?: number;
  };
};

export type AIProgressEvent = {
  jobId?: string;
  percent: number;
  currentChunk: number;
  totalChunks: number;
  currentChunkChars?: number;
  processedChars?: number;
  totalChars?: number;
  stage: string;
  queuePosition?: number;
  queueTotal?: number;
  etaSeconds?: number;
  estimatedTokens?: number;
  tokensPerSecond?: number;
  activeJobId?: string;
  activeActorId?: number;
  activeActorDisplayId?: string;
  activeGeneratedTokens?: number;
  activeTokensPerSecond?: number;
};

export type AIChunkDeltaEvent = {
  jobId?: string;
  chunkIndex: number;
  delta: string;
};

export type AIChunkDoneEvent = {
  jobId?: string;
  chunkIndex: number;
  text: string;
};

export type AIReasoningDeltaEvent = {
  jobId?: string;
  chunkIndex: number;
  reasoningDelta: string;
};

export type AIReasoningDoneEvent = {
  jobId?: string;
  chunkIndex: number;
  reasoning: string;
};

export type AIFinalEvent = {
  jobId?: string;
  text: string;
  tokensPerSecond?: number;
  usage?: {
    promptTokens?: number;
    completionTokens?: number;
    totalTokens?: number;
  };
  summary?: Record<string, unknown>;
};

export type AIUpstreamEvent = {
  jobId?: string;
  chunkIndex?: number;
  stage?: string;
  upstream?: Record<string, unknown>;
};

export type AIErrorEvent = {
  code: string;
  message: string;
  detail?: string;
};

export type AIStreamEvent =
  | ({ type: "progress" } & AIProgressEvent)
  | ({ type: "chunk_delta" } & AIChunkDeltaEvent)
  | ({ type: "chunk_done" } & AIChunkDoneEvent)
  | ({ type: "reasoning_delta" } & AIReasoningDeltaEvent)
  | ({ type: "reasoning_done" } & AIReasoningDoneEvent)
  | ({ type: "final" } & AIFinalEvent)
  | ({ type: "upstream_event" } & AIUpstreamEvent)
  | ({ type: "error" } & AIErrorEvent);

export type AIJobStreamEvent =
  | { type: "connected"; jobId?: string }
  | { type: "heartbeat"; jobId?: string; at?: number }
  | { type: "job"; job: AIJobPayload }
  | ({ type: "progress" } & AIProgressEvent)
  | ({ type: "chunk_delta" } & AIChunkDeltaEvent)
  | ({ type: "chunk_done" } & AIChunkDoneEvent)
  | ({ type: "reasoning_delta" } & AIReasoningDeltaEvent)
  | ({ type: "reasoning_done" } & AIReasoningDoneEvent)
  | ({ type: "final" } & AIFinalEvent)
  | ({ type: "upstream_event" } & AIUpstreamEvent)
  | ({ type: "error" } & AIErrorEvent);

export type AIStreamHandlers = {
  onProgress?: (event: AIProgressEvent) => void;
  onChunkDelta?: (event: AIChunkDeltaEvent) => void;
  onChunkDone?: (event: AIChunkDoneEvent) => void;
  onReasoningDelta?: (event: AIReasoningDeltaEvent) => void;
  onReasoningDone?: (event: AIReasoningDoneEvent) => void;
  onFinal?: (event: AIFinalEvent) => void;
  onUpstream?: (event: AIUpstreamEvent) => void;
  onError?: (event: AIErrorEvent) => void;
  onEvent?: (event: AIStreamEvent) => void;
};

export type AIJobStreamHandlers = {
  onConnected?: (event: Extract<AIJobStreamEvent, { type: "connected" }>) => void;
  onJob?: (job: AIJobPayload) => void;
  onProgress?: (event: AIProgressEvent) => void;
  onChunkDelta?: (event: AIChunkDeltaEvent) => void;
  onChunkDone?: (event: AIChunkDoneEvent) => void;
  onReasoningDelta?: (event: AIReasoningDeltaEvent) => void;
  onReasoningDone?: (event: AIReasoningDoneEvent) => void;
  onFinal?: (event: AIFinalEvent) => void;
  onUpstream?: (event: AIUpstreamEvent) => void;
  onError?: (event: AIErrorEvent) => void;
  onEvent?: (event: AIJobStreamEvent) => void;
};

export type AIGenerationLogItem = {
  id: number;
  jobId: string;
  type: string;
  templateKey: string;
  actorType: string;
  actorId?: number | null;
  actorDisplayId?: string | null;
  inputSummary: string;
  outputSummary: string;
  status: string;
  model: string;
  baseUrl: string;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  estimatedCost: number;
  errorCode: string;
  errorMessage?: string;
  durationMs: number;
  tokensPerSecond?: number;
  metadata?: Record<string, unknown> | null;
  createdAt: string;
  updatedAt?: string | null;
};

export type AIGenerationLogsPayload = {
  items: AIGenerationLogItem[];
  pagination?: {
    page: number;
    limit: number;
    total: number;
    pages: number;
  };
};

export type AIJobInput = AIRequestInput & {
  requestHash?: string;
};

export type AIJobStatus =
  | "queued"
  | "running"
  | "completed"
  | "failed"
  | "canceled"
  | string;

export type AIJobPayload = {
  id: number;
  jobId: string;
  requestHash: string;
  type: string;
  templateKey: string;
  actorType: string;
  actorId?: number | null;
  actorDisplayId?: string | null;
  status: AIJobStatus;
  stage: string;
  percent: number;
  currentChunk: number;
  totalChunks: number;
  processedChars: number;
  totalChars: number;
  inputSummary: string;
  output: string;
  reasoning: string;
  errorCode: string;
  errorMessage?: string;
  upstreamStatus?: number;
  upstreamDetail?: string;
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  estimatedTokens: number;
  tokensPerSecond?: number;
  metadata?: Record<string, unknown> | null;
  queueJob?: Record<string, unknown>;
  startedAt?: string | null;
  finishedAt?: string | null;
  createdAt: string;
  updatedAt?: string | null;
};

export type AIJobListPayload = {
  items: AIJobPayload[];
  pagination?: {
    page: number;
    limit: number;
    total: number;
    pages: number;
  };
  stats?: {
    queued?: number;
    running?: number;
    active?: number;
    [key: string]: unknown;
  };
};
