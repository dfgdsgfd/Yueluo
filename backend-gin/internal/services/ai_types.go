package services

import (
	"net/http"
	"sync"

	"gorm.io/gorm"
)

const (
	AITaskFormatMarkdown      = "format_markdown"
	AITaskPostPolish          = "post_polish"
	AITaskPostCustomGenerate  = "post_custom_generate"
	AITaskPostAutoComment     = "post_auto_comment"
	AITaskCommentReply        = "comment_reply"
	AITaskCommentMentionReply = "comment_mention_reply"

	AISettingEnabled                  = "ai_agent_enabled"
	AISettingBaseURL                  = "ai_agent_base_url"
	AISettingAPIKey                   = "ai_agent_api_key"
	AISettingModel                    = "ai_agent_model"
	AISettingExtraHeaders             = "ai_agent_extra_headers"
	AISettingTimeoutSeconds           = "ai_agent_timeout_seconds"
	AISettingMaxRunSeconds            = "ai_agent_max_run_seconds"
	AISettingChunkMaxChars            = "ai_agent_chunk_max_chars"
	AISettingConcurrency              = "ai_agent_concurrency"
	AISettingTemperature              = "ai_agent_temperature"
	AISettingMaxOutputTokens          = "ai_agent_max_output_tokens"
	AISettingPromptTemplates          = "ai_agent_prompt_templates"
	AISettingShowReasoning            = "ai_agent_show_reasoning"
	AISettingThinkingParameterEnabled = "ai_agent_thinking_parameter_enabled"
	AISettingThinkingEnabled          = "ai_agent_thinking_enabled"
	AISettingReasoningEffort          = "ai_agent_reasoning_effort"
	AISettingModelParameters          = "ai_agent_model_parameters"
	AISettingLogHTTPDetails           = "ai_agent_log_http_details"
	AISettingContentFormat            = "ai_agent_content_format"
	AISettingAutoCommentEnabled       = "ai_agent_auto_comment_enabled"
	AISettingAutoCommentBotUserID     = "ai_agent_auto_comment_bot_user_id"
	AISettingAutoCommentBotUserIDMin  = "ai_agent_auto_comment_bot_user_id_min"
	AISettingAutoCommentBotUserIDMax  = "ai_agent_auto_comment_bot_user_id_max"
	AISettingAutoCommentTemplateKey   = "ai_agent_auto_comment_template_key"
	AISettingAutoCommentDelaySeconds  = "ai_agent_auto_comment_delay_seconds"
	AISettingAutoCommentMaxImages     = "ai_agent_auto_comment_max_images"
	AISettingAutoCommentStyle         = "ai_agent_auto_comment_style"
	AISettingAutoCommentImageMode     = "ai_agent_auto_comment_image_selection_mode"
	AISettingCommentReply             = "ai_agent_comment_reply"
	AISettingModeration               = "ai_agent_moderation"
	AISettingPublishGeneration        = "ai_agent_publish_generation"
)

type AIService struct {
	db             *gorm.DB
	settings       *SettingsService
	client         *http.Client
	gate           *aiConcurrencyGate
	projectGatesMu sync.Mutex
	projectGates   map[string]*aiConcurrencyGate
	jobEvents      *aiJobEventBroker
}

type AIConfig struct {
	Enabled                  bool                        `json:"enabled"`
	BaseURL                  string                      `json:"baseUrl"`
	APIKey                   string                      `json:"-"`
	Model                    string                      `json:"model"`
	ExtraHeaders             map[string]string           `json:"extraHeaders"`
	TimeoutSeconds           int                         `json:"timeoutSeconds"`
	MaxRunSeconds            int                         `json:"maxRunSeconds"`
	ChunkMaxChars            int                         `json:"chunkMaxChars"`
	Concurrency              int                         `json:"concurrency"`
	Temperature              float64                     `json:"temperature"`
	MaxOutputTokens          int                         `json:"maxOutputTokens"`
	Templates                map[string]AITemplateConfig `json:"templates"`
	ShowReasoning            bool                        `json:"showReasoning"`
	ThinkingParameterEnabled bool                        `json:"thinkingParameterEnabled"`
	ThinkingEnabled          bool                        `json:"thinkingEnabled"`
	ReasoningEffort          string                      `json:"reasoningEffort"`
	ModelParameters          map[string]any              `json:"modelParameters"`
	LogHTTPDetails           bool                        `json:"logHttpDetails"`
	ContentFormat            AIContentFormatConfig       `json:"contentFormat"`
	AutoComment              AIAutoCommentConfig         `json:"autoComment"`
	CommentReply             AICommentReplyConfig        `json:"commentReply"`
	Moderation               AIModerationConfig          `json:"moderation"`
	PublishGeneration        AIPublishGenerationConfig   `json:"publishGeneration"`
}

type AITemplateConfig struct {
	Enabled          bool               `json:"enabled"`
	TaskType         string             `json:"taskType"`
	Prompt           string             `json:"prompt"`
	SystemPrompt     string             `json:"systemPrompt"`
	UserPrompt       string             `json:"userPrompt"`
	Style            string             `json:"style"`
	Model            string             `json:"model"`
	Temperature      float64            `json:"temperature"`
	MaxOutputTokens  int                `json:"maxOutputTokens"`
	Concurrency      int                `json:"concurrency"`
	StructuredJSON   bool               `json:"structuredJson"`
	SupportsVision   bool               `json:"supportsVision"`
	RuntimeOverrides AIRuntimeOverrides `json:"runtimeOverrides"`

	systemPromptSet     bool
	userPromptSet       bool
	promptSet           bool
	styleSet            bool
	modelSet            bool
	maxOutputSet        bool
	concurrencySet      bool
	runtimeOverridesSet bool
}

type AIRuntimeOverrides struct {
	Enabled                  bool           `json:"enabled"`
	ShowReasoning            *bool          `json:"showReasoning,omitempty"`
	ThinkingParameterEnabled *bool          `json:"thinkingParameterEnabled,omitempty"`
	ThinkingEnabled          *bool          `json:"thinkingEnabled,omitempty"`
	ReasoningEffort          *string        `json:"reasoningEffort,omitempty"`
	ModelParameters          map[string]any `json:"modelParameters,omitempty"`
}

type AIContentFormatConfig struct {
	Enabled bool                        `json:"enabled"`
	Format  AIContentFormatTargetConfig `json:"format"`
	Polish  AIContentFormatTargetConfig `json:"polish"`
	Custom  AIContentFormatTargetConfig `json:"custom"`
}

type AIContentFormatTargetConfig struct {
	Enabled      bool                        `json:"enabled"`
	TemplateKey  string                      `json:"templateKey"`
	Continuation AIContentContinuationConfig `json:"continuation"`
}

type AIContentContinuationConfig struct {
	Enabled      bool `json:"enabled"`
	TriggerChars int  `json:"triggerChars"`
	MaxRounds    int  `json:"maxRounds"`
	ContextChars int  `json:"contextChars"`
}

type AIPublicSettings struct {
	Enabled                  bool                        `json:"enabled"`
	BaseURL                  string                      `json:"baseUrl"`
	APIKeySet                bool                        `json:"apiKeySet"`
	APIKeyMasked             string                      `json:"apiKeyMasked"`
	Model                    string                      `json:"model"`
	ExtraHeaders             map[string]string           `json:"extraHeaders"`
	TimeoutSeconds           int                         `json:"timeoutSeconds"`
	MaxRunSeconds            int                         `json:"maxRunSeconds"`
	ChunkMaxChars            int                         `json:"chunkMaxChars"`
	Concurrency              int                         `json:"concurrency"`
	Temperature              float64                     `json:"temperature"`
	MaxOutputTokens          int                         `json:"maxOutputTokens"`
	Templates                map[string]AITemplateConfig `json:"templates"`
	ShowReasoning            bool                        `json:"showReasoning"`
	ThinkingParameterEnabled bool                        `json:"thinkingParameterEnabled"`
	ThinkingEnabled          bool                        `json:"thinkingEnabled"`
	ReasoningEffort          string                      `json:"reasoningEffort"`
	ModelParameters          map[string]any              `json:"modelParameters"`
	LogHTTPDetails           bool                        `json:"logHttpDetails"`
	ContentFormat            AIContentFormatConfig       `json:"contentFormat"`
	AutoComment              AIAutoCommentConfig         `json:"autoComment"`
	CommentReply             AICommentReplyConfig        `json:"commentReply"`
	Moderation               AIModerationConfig          `json:"moderation"`
	PublishGeneration        AIPublishGenerationConfig   `json:"publishGeneration"`
	DefaultTemplates         map[string]AITemplateConfig `json:"defaultTemplates"`
}

type AIAutoCommentConfig struct {
	Enabled            bool   `json:"enabled"`
	BotUserID          int64  `json:"botUserId"`
	BotUserIDMin       int64  `json:"botUserIdMin"`
	BotUserIDMax       int64  `json:"botUserIdMax"`
	TemplateKey        string `json:"templateKey"`
	DelaySeconds       int    `json:"delaySeconds"`
	MaxImages          int    `json:"maxImages"`
	ImageSelectionMode string `json:"imageSelectionMode"`
	Style              string `json:"style"`
}

type AICommentReplyConfig struct {
	Enabled                  bool   `json:"enabled"`
	TemplateKey              string `json:"templateKey"`
	DelaySeconds             int    `json:"delaySeconds"`
	MaxImages                int    `json:"maxImages"`
	ImageSelectionMode       string `json:"imageSelectionMode"`
	Style                    string `json:"style"`
	MaxRepliesPerAIComment   int    `json:"maxRepliesPerAIComment"`
	MentionEnabled           bool   `json:"mentionEnabled"`
	MentionName              string `json:"mentionName"`
	MentionTemplateKey       string `json:"mentionTemplateKey"`
	MentionBotUserIDMin      int64  `json:"mentionBotUserIdMin"`
	MentionBotUserIDMax      int64  `json:"mentionBotUserIdMax"`
	MaxMentionRepliesPerPost int    `json:"maxMentionRepliesPerPost"`
}

type AIModerationConfig struct {
	Comment AIModerationTargetConfig `json:"comment"`
	Post    AIModerationTargetConfig `json:"post"`
}

type AIModerationTargetConfig struct {
	Enabled     bool                              `json:"enabled"`
	TemplateKey string                            `json:"templateKey"`
	Prompt      string                            `json:"prompt"`
	Rules       map[string]AIModerationRuleConfig `json:"rules"`
}

type AIModerationRuleConfig struct {
	Enabled     bool    `json:"enabled"`
	Action      string  `json:"action"`
	Sensitivity float64 `json:"sensitivity"`
}

type AIRequest struct {
	Type        string         `json:"type"`
	Locale      string         `json:"locale"`
	Input       string         `json:"input"`
	TemplateKey string         `json:"templateKey"`
	Variables   map[string]any `json:"variables"`
	Options     AIOptions      `json:"options"`
	Images      []AIImageInput `json:"images"`
}

type AIOptions struct {
	Temperature     *float64 `json:"temperature,omitempty"`
	MaxOutputTokens *int     `json:"maxOutputTokens,omitempty"`
	StructuredJSON  *bool    `json:"structuredJson,omitempty"`
	TimeoutSeconds  *int     `json:"-"`
}

type AIImageInput struct {
	URL     string `json:"url"`
	DataURL string `json:"dataUrl"`
	Mime    string `json:"mime"`
	Alt     string `json:"alt"`
}

type AIActor struct {
	Type      string
	ID        *int64
	DisplayID *string
}

type AIStreamEvent struct {
	Type              string         `json:"-"`
	JobID             string         `json:"jobId,omitempty"`
	Percent           int            `json:"percent,omitempty"`
	CurrentChunk      int            `json:"currentChunk,omitempty"`
	TotalChunks       int            `json:"totalChunks,omitempty"`
	CurrentChunkChars int            `json:"currentChunkChars,omitempty"`
	ProcessedChars    int            `json:"processedChars,omitempty"`
	TotalChars        int            `json:"totalChars,omitempty"`
	Stage             string         `json:"stage,omitempty"`
	QueuePosition     int            `json:"queuePosition,omitempty"`
	QueueTotal        int            `json:"queueTotal,omitempty"`
	ETASeconds        int            `json:"etaSeconds,omitempty"`
	EstimatedTokens   int            `json:"estimatedTokens,omitempty"`
	TokensPerSecond   float64        `json:"tokensPerSecond,omitempty"`
	ActiveJobID       string         `json:"activeJobId,omitempty"`
	ActiveActorID     int64          `json:"activeActorId,omitempty"`
	ActiveDisplayID   string         `json:"activeActorDisplayId,omitempty"`
	ActiveTokens      int            `json:"activeGeneratedTokens,omitempty"`
	ActiveRate        float64        `json:"activeTokensPerSecond,omitempty"`
	ChunkIndex        int            `json:"chunkIndex,omitempty"`
	Delta             string         `json:"delta,omitempty"`
	ReasoningDelta    string         `json:"reasoningDelta,omitempty"`
	Text              string         `json:"text,omitempty"`
	Reasoning         string         `json:"reasoning,omitempty"`
	Usage             *AIUsage       `json:"usage,omitempty"`
	Summary           map[string]any `json:"summary,omitempty"`
	Upstream          map[string]any `json:"upstream,omitempty"`
	Code              string         `json:"code,omitempty"`
	Message           string         `json:"message,omitempty"`
	Detail            string         `json:"detail,omitempty"`
}

type AIUsage struct {
	PromptTokens     int `json:"promptTokens,omitempty"`
	CompletionTokens int `json:"completionTokens,omitempty"`
	TotalTokens      int `json:"totalTokens,omitempty"`
}

type AIError struct {
	Code           string
	Err            error
	UpstreamStatus int
	UpstreamDetail string
}

func (e AIError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Code
}

func (e AIError) Unwrap() error { return e.Err }
