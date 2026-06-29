package services

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

func TestSplitAITextChunksPrefersParagraphs(t *testing.T) {
	input := strings.Repeat("A", 260) + "\n\n" + strings.Repeat("B", 260) + "\n\n" + strings.Repeat("C", 260)
	chunks := SplitAITextChunks(input, 500)
	if len(chunks) != 3 {
		t.Fatalf("chunks = %#v, want 3 paragraph chunks", chunks)
	}
	if chunks[0] != strings.Repeat("A", 260) || !strings.HasPrefix(chunks[1], "BBBB") {
		t.Fatalf("unexpected chunks: %#v", chunks)
	}
}

func TestSplitAITextChunksZeroDisablesChunking(t *testing.T) {
	input := strings.Repeat("长文", 4000) + "\n\n" + strings.Repeat("内容", 4000)
	chunks := SplitAITextChunks(input, 0)
	if len(chunks) != 1 {
		t.Fatalf("chunks = %d, want 1 when maxChars is 0", len(chunks))
	}
	if chunks[0] != input {
		t.Fatal("chunk content changed when chunking is disabled")
	}
}

func TestParsePublishGenerationFieldRejectsPromptEcho(t *testing.T) {
	input := "已有标题：\n你啊\n\n已有详情正文：\n是的\n\n可用于分析的图片数量：3"
	if got := parsePublishGenerationField(input, "title"); got != "" {
		t.Fatalf("parsePublishGenerationField() = %q, want empty prompt echo rejection", got)
	}
	if got := parsePublishGenerationField(`{"title":"`+input+`"}`, "title"); got != "" {
		t.Fatalf("parsePublishGenerationField(JSON echo) = %q, want empty prompt echo rejection", got)
	}
}

func TestParsePublishGenerationFieldParsesLabeledOutput(t *testing.T) {
	input := "标题：窗边这束花很亮\n详情：白色花瓶和木桌都很清楚，整体很干净。"
	if got := parsePublishGenerationField(input, "title"); got != "窗边这束花很亮" {
		t.Fatalf("parsePublishGenerationField(title) = %q, want labeled title", got)
	}
	if got := parsePublishGenerationField(input, "detail"); got != "白色花瓶和木桌都很清楚，整体很干净。" {
		t.Fatalf("parsePublishGenerationField(detail) = %q, want labeled detail", got)
	}
	if got := parsePublishGenerationField("标题：窗边这束花很亮", "detail"); got != "" {
		t.Fatalf("parsePublishGenerationField(partial detail) = %q, want empty instead of labeled fallback", got)
	}
}

func TestParsePublishGenerationTitleDetailParsesJSONAndPlainText(t *testing.T) {
	title, detail := parsePublishGenerationTitleDetail("窗边这束花很亮\n白色花瓶和木桌都很清楚，整体很干净。")
	if title != "窗边这束花很亮" || detail != "白色花瓶和木桌都很清楚，整体很干净。" {
		t.Fatalf("plain parse = %q/%q, want first line title and remaining detail", title, detail)
	}
	title, detail = parsePublishGenerationTitleDetail(`{"data":{"title":"JSON 标题","detail":"JSON 详情"}}`)
	if title != "JSON 标题" || detail != "JSON 详情" {
		t.Fatalf("json parse = %q/%q, want nested title/detail", title, detail)
	}
	title, detail = parsePublishGenerationTitleDetail("```json\n{\"title\":\"代码块标题\",\"detail\":\"代码块详情\"}\n```")
	if title != "代码块标题" || detail != "代码块详情" {
		t.Fatalf("json fence parse = %q/%q, want fenced title/detail", title, detail)
	}
	title, detail = parsePublishGenerationTitleDetail(`{"result":"包裹标题\n包裹详情第一行\n包裹详情第二行"}`)
	if title != "包裹标题" || detail != "包裹详情第一行\n包裹详情第二行" {
		t.Fatalf("json text payload parse = %q/%q, want first line title and remaining detail", title, detail)
	}
	title, detail = parsePublishGenerationTitleDetail(`{"output":{"text":"嵌套标题\n嵌套详情"}}`)
	if title != "嵌套标题" || detail != "嵌套详情" {
		t.Fatalf("nested json text payload parse = %q/%q, want nested text payload parsed", title, detail)
	}
}

func TestNormalizePublishGenerationTemplateSwitchesJSONModeDefaults(t *testing.T) {
	current := DefaultAIPromptTemplates()
	jsonDefaults := defaultAIPublishGenerationJSONTemplates()
	jsonDefault := jsonDefaults["publish_title_detail_generate"]
	raw := map[string]any{
		"publish_title_detail_generate": map[string]any{
			"enabled":        true,
			"taskType":       AITaskPublishTitleDetailGenerate,
			"systemPrompt":   jsonDefault.SystemPrompt,
			"userPrompt":     jsonDefault.UserPrompt,
			"prompt":         jsonDefault.UserPrompt,
			"structuredJson": true,
			"supportsVision": true,
		},
	}
	templates, valid := normalizeTemplatesSetting(raw, current)
	if !valid {
		t.Fatal("normalizeTemplatesSetting() rejected publish JSON mode template")
	}
	tmpl := templates["publish_title_detail_generate"]
	if !tmpl.StructuredJSON || !strings.Contains(tmpl.UserPrompt, `{"title":"...","detail":"..."}`) {
		t.Fatalf("template = %#v, want JSON default prompt when structuredJson is enabled", tmpl)
	}

	raw["publish_title_detail_generate"].(map[string]any)["structuredJson"] = false
	templates, valid = normalizeTemplatesSetting(raw, current)
	if !valid {
		t.Fatal("normalizeTemplatesSetting() rejected publish text mode template")
	}
	tmpl = templates["publish_title_detail_generate"]
	if tmpl.StructuredJSON || !strings.Contains(tmpl.UserPrompt, "第一行只写标题") || strings.Contains(tmpl.UserPrompt, `{"title":"...","detail":"..."}`) {
		t.Fatalf("template = %#v, want text default prompt when structuredJson is disabled", tmpl)
	}
}

func TestNormalizePublishGenerationTemplatePreservesCustomPromptText(t *testing.T) {
	current := DefaultAIPromptTemplates()
	customSystem := current["publish_detail_generate"].SystemPrompt + "\n请保留后台自定义补充。"
	customUser := current["publish_detail_generate"].UserPrompt + "\n结尾加一句更生活化的收束。"
	raw := map[string]any{
		"publish_detail_generate": map[string]any{
			"enabled":         true,
			"taskType":        AITaskPublishDetailGenerate,
			"systemPrompt":    customSystem,
			"userPrompt":      customUser,
			"prompt":          customUser,
			"structuredJson":  false,
			"supportsVision":  true,
			"temperature":     0.5,
			"maxOutputTokens": 900,
		},
	}
	templates, valid := normalizeTemplatesSetting(raw, current)
	if !valid {
		t.Fatal("normalizeTemplatesSetting() rejected custom publish generation prompt")
	}
	tmpl := templates["publish_detail_generate"]
	if tmpl.SystemPrompt != customSystem || tmpl.UserPrompt != customUser || tmpl.Prompt != customUser {
		t.Fatalf("custom prompt was overwritten: %#v", tmpl)
	}
}

func TestNormalizePublishGenerationTemplateMigratesLegacyDetailPrompt(t *testing.T) {
	current := DefaultAIPromptTemplates()
	legacy := legacyAIPublishGenerationTextTemplates()["publish_detail_generate"]
	raw := map[string]any{
		"publish_detail_generate": map[string]any{
			"enabled":         true,
			"taskType":        AITaskPublishDetailGenerate,
			"systemPrompt":    legacy.SystemPrompt,
			"userPrompt":      legacy.UserPrompt,
			"prompt":          legacy.Prompt,
			"structuredJson":  false,
			"supportsVision":  true,
			"temperature":     0.5,
			"maxOutputTokens": 900,
		},
	}
	templates, valid := normalizeTemplatesSetting(raw, current)
	if !valid {
		t.Fatal("normalizeTemplatesSetting() rejected legacy publish detail prompt")
	}
	tmpl := templates["publish_detail_generate"]
	if tmpl.UserPrompt != current["publish_detail_generate"].UserPrompt || !strings.Contains(tmpl.UserPrompt, "长文允许 Markdown") {
		t.Fatalf("legacy prompt was not migrated: %q", tmpl.UserPrompt)
	}
}

func TestModerationPromptContentShortensLongInputAndKeepsTail(t *testing.T) {
	input := strings.Repeat("A", aiModerationMaxPromptRunes) + "TAIL"
	shortened, truncated, originalRunes := moderationPromptContent(input)
	if !truncated {
		t.Fatal("truncated = false, want true")
	}
	if originalRunes != aiModerationMaxPromptRunes+4 {
		t.Fatalf("originalRunes = %d, want %d", originalRunes, aiModerationMaxPromptRunes+4)
	}
	if !strings.HasPrefix(shortened, strings.Repeat("A", 100)) || !strings.HasSuffix(shortened, "TAIL") {
		t.Fatalf("shortened content does not keep expected head/tail")
	}
	if !strings.Contains(shortened, "omitted for timeout protection") {
		t.Fatalf("shortened content missing omission marker")
	}
}

func TestAIServiceUpdateSettingsReasoningAndModelParameters(t *testing.T) {
	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	service := NewAIService(db, settings)

	err := service.UpdateSettings(context.Background(), map[string]any{
		"reasoningEffort": "medium",
		"maxRunSeconds":   7200,
		"modelParameters": `{"top_p":0.9,"seed":42}`,
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	cfg := service.Config()
	if cfg.ReasoningEffort != "medium" {
		t.Fatalf("ReasoningEffort = %q, want medium", cfg.ReasoningEffort)
	}
	if cfg.MaxRunSeconds != 7200 {
		t.Fatalf("MaxRunSeconds = %d, want 7200", cfg.MaxRunSeconds)
	}
	if fmt.Sprint(cfg.ModelParameters["top_p"]) != "0.9" || fmt.Sprint(cfg.ModelParameters["seed"]) != "42" {
		t.Fatalf("ModelParameters = %#v, want top_p and seed", cfg.ModelParameters)
	}

	if err := service.UpdateSettings(context.Background(), map[string]any{"modelParameters": `{"top_p":0.9} trailing`}); err == nil {
		t.Fatal("UpdateSettings() accepted invalid modelParameters JSON")
	}
	if err := service.UpdateSettings(context.Background(), map[string]any{"reasoningEffort": "extreme"}); err == nil {
		t.Fatal("UpdateSettings() accepted invalid reasoningEffort")
	}
	if err := service.UpdateSettings(context.Background(), map[string]any{"maxRunSeconds": 90000}); err == nil {
		t.Fatal("UpdateSettings() accepted invalid maxRunSeconds")
	}
}

func TestAIServiceUpdateSettingsAllowsUnlimitedAIControls(t *testing.T) {
	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	service := NewAIService(db, settings)

	err := service.UpdateSettings(context.Background(), map[string]any{
		"chunkMaxChars":   0,
		"maxOutputTokens": 0,
		"templates": map[string]any{
			"markdown_format": map[string]any{
				"enabled":         true,
				"taskType":        AITaskFormatMarkdown,
				"userPrompt":      "format {{input}}",
				"maxOutputTokens": 0,
			},
		},
		"moderation": map[string]any{
			"comment": map[string]any{
				"rules": map[string]any{
					"spam": map[string]any{"enabled": true, "action": AIModerationActionObserve, "sensitivity": 0.2},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	cfg := service.Config()
	if cfg.ChunkMaxChars != 0 || cfg.MaxOutputTokens != 0 {
		t.Fatalf("unlimited controls not preserved: chunk=%d max=%d", cfg.ChunkMaxChars, cfg.MaxOutputTokens)
	}
	if cfg.Templates["markdown_format"].MaxOutputTokens != 0 {
		t.Fatalf("template max output = %d, want 0", cfg.Templates["markdown_format"].MaxOutputTokens)
	}
	if got := cfg.Moderation.Comment.Rules["spam"].Sensitivity; got != 0.2 {
		t.Fatalf("moderation sensitivity = %v, want 0.2", got)
	}

	err = service.UpdateSettings(context.Background(), map[string]any{
		"maxOutputTokens": 64000,
		"templates": map[string]any{
			"markdown_format": map[string]any{
				"enabled":         true,
				"taskType":        AITaskFormatMarkdown,
				"userPrompt":      "format {{input}}",
				"maxOutputTokens": 64000,
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() large max output error = %v", err)
	}
	cfg = service.Config()
	if cfg.MaxOutputTokens != 64000 {
		t.Fatalf("max output tokens = %d, want 64000", cfg.MaxOutputTokens)
	}
	if cfg.Templates["markdown_format"].MaxOutputTokens != 64000 {
		t.Fatalf("template max output = %d, want 64000", cfg.Templates["markdown_format"].MaxOutputTokens)
	}
}

func TestAIServiceUpdateSettingsPublishGenerationTwoStepConfig(t *testing.T) {
	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	service := NewAIService(db, settings)

	err := service.UpdateSettings(context.Background(), map[string]any{
		"publishGeneration": map[string]any{
			"enabled":            true,
			"maxImages":          5,
			"imageSelectionMode": "random",
			"titleMaxChars":      18,
			"detail": map[string]any{
				"enabled":     true,
				"templateKey": "publish_detail_generate",
			},
			"title": map[string]any{
				"enabled":     true,
				"templateKey": "publish_title_generate",
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	cfg := service.Config().PublishGeneration
	if cfg.MaxImages != 5 || cfg.ImageSelectionMode != AIImageSelectionRandom || cfg.TitleMaxChars != 18 {
		t.Fatalf("publish config = %#v, want max/images/title chars", cfg)
	}
	if cfg.Detail.TemplateKey != "publish_detail_generate" || cfg.Title.TemplateKey != "publish_title_generate" {
		t.Fatalf("publish templates = %#v", cfg)
	}
}

func TestAIServiceUpdateSettingsContentFormatConfig(t *testing.T) {
	db := newAITestDB(t)
	if err := db.AutoMigrate(&domain.AIJob{}); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	service := NewAIService(db, settings)

	err := service.UpdateSettings(context.Background(), map[string]any{
		"contentFormat": map[string]any{
			"enabled": true,
			"format": map[string]any{
				"enabled":     true,
				"templateKey": "markdown_format_alt",
			},
			"custom": map[string]any{
				"enabled": false,
			},
		},
		"templates": map[string]any{
			"markdown_format_alt": map[string]any{
				"enabled":         true,
				"taskType":        AITaskFormatMarkdown,
				"systemPrompt":    "system",
				"userPrompt":      "alt {{input}}",
				"maxOutputTokens": 256,
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	cfg := service.Config().ContentFormat
	if !cfg.Enabled || cfg.Format.TemplateKey != "markdown_format_alt" || cfg.Custom.Enabled {
		t.Fatalf("content format config = %#v, want override and disabled custom", cfg)
	}
	if !cfg.Custom.Continuation.Enabled || cfg.Custom.Continuation.TriggerChars != 6000 ||
		cfg.Custom.Continuation.MaxRounds != 2 || cfg.Custom.Continuation.ContextChars != 2400 {
		t.Fatalf("custom continuation default = %#v, want normalized defaults", cfg.Custom.Continuation)
	}
	job, err := service.CreateJob(context.Background(), AIJobCreateInput{Request: AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "en",
		Input:       "plain text",
		TemplateKey: "markdown_format",
	}}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if job.TemplateKey != "markdown_format_alt" {
		t.Fatalf("job template = %q, want markdown_format_alt", job.TemplateKey)
	}
	err = service.RunStream(context.Background(), AIRequest{
		Type: AITaskPostCustomGenerate,
		Variables: map[string]any{
			"customPrompt": "write a post",
		},
		TemplateKey: "post_custom_generate",
	}, AIActor{Type: "user"}, func(AIStreamEvent) error { return nil })
	var aiErr AIError
	if !errors.As(err, &aiErr) || aiErrorCode(err) != "error.ai_template_disabled" {
		t.Fatalf("RunStream disabled custom error = %v, want error.ai_template_disabled", err)
	}
}

func TestAIServiceUpdateSettingsContentFormatContinuationConfig(t *testing.T) {
	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	service := NewAIService(db, settings)

	err := service.UpdateSettings(context.Background(), map[string]any{
		"contentFormat": map[string]any{
			"custom": map[string]any{
				"continuation": map[string]any{
					"enabled":      true,
					"triggerChars": 1800,
					"maxRounds":    3,
					"contextChars": 900,
				},
			},
			"format": map[string]any{
				"continuation": map[string]any{
					"enabled":      true,
					"triggerChars": 1800,
					"maxRounds":    3,
					"contextChars": 900,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	cfg := service.Config().ContentFormat
	if cfg.Custom.Continuation.TriggerChars != 1800 || cfg.Custom.Continuation.MaxRounds != 3 || cfg.Custom.Continuation.ContextChars != 900 {
		t.Fatalf("custom continuation = %#v, want custom values", cfg.Custom.Continuation)
	}
	if cfg.Format.Continuation != (AIContentContinuationConfig{}) {
		t.Fatalf("format continuation = %#v, want ignored outside custom generation", cfg.Format.Continuation)
	}
}

func TestAIServiceUpdateSettingsCommentReplyConfig(t *testing.T) {
	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	service := NewAIService(db, settings)

	err := service.UpdateSettings(context.Background(), map[string]any{
		"commentReply": map[string]any{
			"enabled":                  true,
			"templateKey":              "comment_reply",
			"delaySeconds":             7,
			"maxImages":                2,
			"imageSelectionMode":       "random",
			"style":                    "humorous",
			"maxRepliesPerAIComment":   5,
			"mentionEnabled":           true,
			"mentionName":              "@yueai",
			"mentionTemplateKey":       "comment_mention_reply",
			"mentionBotUserIdMin":      50,
			"mentionBotUserIdMax":      40,
			"maxMentionRepliesPerPost": 4,
		},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	cfg := service.Config().CommentReply
	if !cfg.Enabled || cfg.TemplateKey != "comment_reply" || cfg.DelaySeconds != 7 || cfg.MaxImages != 2 ||
		cfg.ImageSelectionMode != AIImageSelectionRandom || cfg.Style != "humorous" || cfg.MaxRepliesPerAIComment != 5 ||
		!cfg.MentionEnabled || cfg.MentionName != "yueai" || cfg.MentionTemplateKey != "comment_mention_reply" ||
		cfg.MentionBotUserIDMin != 40 || cfg.MentionBotUserIDMax != 50 || cfg.MaxMentionRepliesPerPost != 4 {
		t.Fatalf("comment reply config = %#v, want saved values", cfg)
	}
	if err := service.UpdateSettings(context.Background(), map[string]any{
		"commentReply": map[string]any{"maxRepliesPerAIComment": 99, "maxImages": 99, "maxMentionRepliesPerPost": 99},
	}); err != nil {
		t.Fatalf("UpdateSettings() clamped comment reply config error = %v", err)
	}
	cfg = service.Config().CommentReply
	if cfg.MaxRepliesPerAIComment != 5 || cfg.MaxImages != 2 || cfg.MaxMentionRepliesPerPost != 4 {
		t.Fatalf("invalid comment reply config changed values: %#v", cfg)
	}
}

func TestLimitPublishGenerationTitleUsesConfiguredLimit(t *testing.T) {
	got := limitPublishGenerationTitle("这是一个很长很长的标题", 8)
	if len([]rune(got)) != 8 {
		t.Fatalf("limited title = %q, want 8 runes", got)
	}
}

func TestAIServiceTemplateSystemAndUserPromptCompatibility(t *testing.T) {
	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	service := NewAIService(db, settings)

	err := service.UpdateSettings(context.Background(), map[string]any{
		"templates": map[string]any{
			"markdown_format": map[string]any{
				"enabled":         true,
				"taskType":        AITaskFormatMarkdown,
				"systemPrompt":    "",
				"userPrompt":      "new user {{input}}",
				"prompt":          "new user {{input}}",
				"style":           "humorous",
				"temperature":     0.2,
				"maxOutputTokens": 2048,
			},
			"legacy_prompt_only": map[string]any{
				"enabled":         true,
				"taskType":        "admin_copy",
				"prompt":          "legacy {{input}}",
				"temperature":     0.3,
				"maxOutputTokens": 512,
			},
		},
	})
	if err != nil {
		t.Fatalf("UpdateSettings() error = %v", err)
	}
	cfg := service.Config()
	current := cfg.Templates["markdown_format"]
	if current.SystemPrompt != "" || current.UserPrompt != "new user {{input}}" || current.Style != "humorous" {
		t.Fatalf("template = %#v, want cleared system prompt and user prompt/style", current)
	}
	legacy := cfg.Templates["legacy_prompt_only"]
	if legacy.UserPrompt != "legacy {{input}}" || legacy.Prompt != "legacy {{input}}" || legacy.SystemPrompt != "" {
		t.Fatalf("legacy template = %#v, want prompt as user prompt without default system prompt", legacy)
	}
	if messages := openAIMessages(current, AIRequest{Input: "body"}); len(messages) != 1 || messages[0]["role"] != "user" {
		t.Fatalf("messages = %#v, want only user message when system prompt is cleared", messages)
	}
}

func TestAIServiceRunStreamParsesOpenAICompatibleSSE(t *testing.T) {
	requestBody := make(chan map[string]any, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s, want /v1/chat/completions", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBody <- body
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}],\"usage\":{\"prompt_tokens\":2,\"completion_tokens\":3,\"total_tokens\":5}}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.AIGenerationLog{}, &domain.SystemSetting{}); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "test-model")
	settings.Set(context.Background(), AISettingChunkMaxChars, 1000)

	service := NewAIService(db, settings)
	events := []AIStreamEvent{}
	err = service.RunStream(context.Background(), AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "en",
		Input:       "plain text",
		TemplateKey: "markdown_format",
	}, AIActor{Type: "user"}, func(event AIStreamEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	var final AIStreamEvent
	for _, event := range events {
		if event.Type == "final" {
			final = event
		}
	}
	if final.Text != "Hello world" {
		t.Fatalf("final text = %q, want Hello world; events=%#v", final.Text, events)
	}
	body := <-requestBody
	if !requestIncludesMessageRole(body, "system") || !requestIncludesMessageRole(body, "user") {
		t.Fatalf("request body does not include system and user messages: %#v", body)
	}
	if final.Usage == nil || final.Usage.TotalTokens != 5 {
		t.Fatalf("usage = %#v, want total tokens 5", final.Usage)
	}
	var log domain.AIGenerationLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("missing generation log: %v", err)
	}
	if log.Status != "completed" || log.OutputSummary != "Hello world" || log.Model != "test-model" {
		t.Fatalf("unexpected log: %#v", log)
	}
}

func TestAIServiceRunTextParsesOpenAICompatibleJSONResponseAndLogsDetails(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s, want /v1/chat/completions", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": `{"title":"JSON title"}`}},
			},
			"usage": map[string]any{"prompt_tokens": 4, "completion_tokens": 5, "total_tokens": 9},
		})
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "test-model")

	service := NewAIService(db, settings)
	text, usage, err := service.RunText(context.Background(), AIRequest{
		Type:        AITaskPublishTitleGenerate,
		Locale:      "zh-CN",
		Input:       "已有标题：\n（空）\n\n已有详情正文：\n（空）\n\n可用于分析的图片数量：1",
		TemplateKey: "publish_title_generate",
		Options:     AIOptions{StructuredJSON: boolPtr(true)},
	}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("RunText() error = %v", err)
	}
	if text != `{"title":"JSON title"}` {
		t.Fatalf("text = %q, want JSON title payload", text)
	}
	if usage == nil || usage.TotalTokens != 9 {
		t.Fatalf("usage = %#v, want total tokens 9", usage)
	}
	var log domain.AIGenerationLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("missing generation log: %v", err)
	}
	if log.TotalTokens != 9 || log.OutputSummary != `{"title":"JSON title"}` {
		t.Fatalf("unexpected log tokens/output: tokens=%d output=%q", log.TotalTokens, log.OutputSummary)
	}
	var meta map[string]any
	if err := json.Unmarshal(log.Metadata, &meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if meta["upstreamStatus"] != float64(http.StatusOK) {
		t.Fatalf("upstreamStatus = %#v, want 200", meta["upstreamStatus"])
	}
	attempts, ok := meta["upstreamAttempts"].([]any)
	if !ok || len(attempts) != 1 {
		t.Fatalf("upstreamAttempts = %#v, want one attempt", meta["upstreamAttempts"])
	}
	attempt, ok := attempts[0].(map[string]any)
	if !ok {
		t.Fatalf("attempt = %#v, want map", attempts[0])
	}
	if attempt["nonStreamJson"] != true || attempt["status"] != float64(http.StatusOK) {
		t.Fatalf("attempt metadata = %#v, want non-stream 200", attempt)
	}
	if got := fmt.Sprint(attempt["outputSummary"]); !strings.Contains(got, "JSON title") {
		t.Fatalf("outputSummary = %q, want generated content detail", got)
	}
	usageMeta, ok := attempt["usage"].(map[string]any)
	if !ok || usageMeta["totalTokens"] != float64(9) {
		t.Fatalf("attempt usage = %#v, want total tokens 9", attempt["usage"])
	}
}

func TestAIServiceRunTextRecordsSanitizedHTTPRequestDetailsWhenEnabled(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "test-model")
	settings.Set(context.Background(), AISettingLogHTTPDetails, true)
	settings.Set(context.Background(), AISettingExtraHeaders, map[string]string{
		"X-Trace":   "trace-1",
		"X-API-Key": "custom-secret",
	})

	service := NewAIService(db, settings)
	_, _, err := service.RunText(context.Background(), AIRequest{
		Type:        AITaskPublishTitleGenerate,
		Locale:      "zh-CN",
		Input:       "要发送的正文",
		TemplateKey: "publish_title_generate",
		Images: []AIImageInput{{
			DataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(tinyPNGBytes(t)),
			Mime:    "image/png",
			Alt:     "inline.png",
		}},
	}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("RunText() error = %v", err)
	}

	attempt := firstAIUpstreamAttempt(t, db)
	request := testMap(t, attempt["request"])
	if request["detailsRecorded"] != true {
		t.Fatalf("request detailsRecorded = %#v, want true", request["detailsRecorded"])
	}
	headers := testMap(t, request["headers"])
	if headers["Authorization"] != "[redacted]" || headers["X-Api-Key"] != "[redacted]" {
		t.Fatalf("headers were not redacted: %#v", headers)
	}
	if headers["X-Trace"] != "trace-1" {
		t.Fatalf("X-Trace = %#v, want trace-1", headers["X-Trace"])
	}
	body := testMap(t, request["body"])
	if body["model"] != "test-model" {
		t.Fatalf("body model = %#v, want test-model", body["model"])
	}
	rawBody, _ := json.Marshal(body)
	bodyText := string(rawBody)
	if !strings.Contains(bodyText, "要发送的正文") {
		t.Fatalf("request body missing prompt input: %s", bodyText)
	}
	if strings.Contains(bodyText, "iVBOR") {
		t.Fatalf("request body leaked base64 image data: %s", bodyText)
	}
	if !strings.Contains(bodyText, "[redacted") {
		t.Fatalf("request body missing redacted image marker: %s", bodyText)
	}
}

func TestAIServiceRunTextDoesNotRecordHTTPRequestBodyByDefault(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")

	service := NewAIService(db, settings)
	_, _, err := service.RunText(context.Background(), AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "zh-CN",
		Input:       "默认不应完整记录",
		TemplateKey: "markdown_format",
	}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("RunText() error = %v", err)
	}

	attempt := firstAIUpstreamAttempt(t, db)
	request := testMap(t, attempt["request"])
	if request["detailsRecorded"] != false {
		t.Fatalf("request detailsRecorded = %#v, want false", request["detailsRecorded"])
	}
	if _, ok := request["body"]; ok {
		t.Fatalf("request body should not be recorded by default: %#v", request)
	}
	if _, ok := request["headers"]; ok {
		t.Fatalf("request headers should not be recorded by default: %#v", request)
	}
}

func TestAIServiceRunStreamFallsBackToTextForVisionContentSchemaError(t *testing.T) {
	requestBodies := make(chan map[string]any, 2)
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBodies <- body
		if calls.Add(1) == 1 {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = fmt.Fprint(w, `{"error":{"code":400,"message":"data did not match any variant of untagged enum ChatCompletionRequestUserMessageContent","type":"Bad Request"}}`)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Fallback comment\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "test-model")

	service := NewAIService(db, settings)
	text, _, err := service.RunText(context.Background(), AIRequest{
		Type:        AITaskPostAutoComment,
		Locale:      "zh-CN",
		Input:       "post body",
		TemplateKey: "post_auto_comment",
		Images: []AIImageInput{{
			URL:  "https://cdn.example.test/post/image.jpg",
			Mime: "image/jpeg",
			Alt:  "image.jpg",
		}},
	}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("RunText() error = %v", err)
	}
	if text != "Fallback comment" {
		t.Fatalf("text = %q, want fallback comment", text)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want 2", got)
	}
	first := <-requestBodies
	if !requestIncludesImageURL(first, "https://cdn.example.test/post/image.jpg") {
		t.Fatalf("first request should include image_url content: %#v", first)
	}
	second := <-requestBodies
	if requestIncludesImageURL(second, "https://cdn.example.test/post/image.jpg") {
		t.Fatalf("fallback request should not include image_url content: %#v", second)
	}
	if !requestIncludesText(second, "纯文本兼容模式") || !requestIncludesText(second, "image.jpg") {
		t.Fatalf("fallback request missing text-only image summary: %#v", second)
	}
}

func TestAIServiceRunStreamDoesNotFallBackToTextForPublishTitleContentSchemaError(t *testing.T) {
	requestBodies := make(chan map[string]any, 2)
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBodies <- body
		calls.Add(1)
		w.WriteHeader(http.StatusBadRequest)
		_, _ = fmt.Fprint(w, `{"error":{"code":400,"message":"data did not match any variant of untagged enum ChatCompletionRequestUserMessageContent","type":"Bad Request"}}`)
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "test-model")

	service := NewAIService(db, settings)
	text, _, err := service.RunText(context.Background(), AIRequest{
		Type:        AITaskPublishTitleGenerate,
		Locale:      "zh-CN",
		Input:       "已有标题：\n（空）\n\n已有详情正文：\n（空）\n\n可用于分析的图片数量：1",
		TemplateKey: "publish_title_generate",
		Images: []AIImageInput{{
			URL:  "https://cdn.example.test/post/image.jpg",
			Mime: "image/jpeg",
			Alt:  "image.jpg",
		}},
	}, AIActor{Type: "user"})
	if err == nil {
		t.Fatalf("RunText() error is nil, text=%q; want publish title image analysis failure", text)
	}
	var aiErr AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("RunText() error = %T %[1]v, want AIError", err)
	}
	if aiErr.Code != "error.ai_upstream_error" || aiErr.UpstreamStatus != http.StatusBadRequest {
		t.Fatalf("AIError = %#v, want upstream 400 without text fallback", aiErr)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want 2 image attempts", got)
	}
	for index := range 2 {
		body := <-requestBodies
		if !requestIncludesImageURL(body, "https://cdn.example.test/post/image.jpg") {
			t.Fatalf("request %d should include image_url content: %#v", index+1, body)
		}
		if requestIncludesText(body, "纯文本兼容模式") {
			t.Fatalf("request %d should not fall back to text-only image summary: %#v", index+1, body)
		}
	}
}

func TestAIServiceRunTextFailsOnOpenAICompatibleStreamErrorPayload(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"error\":{\"message\":\"Internal server error\",\"type\":\"internal_server_error\",\"code\":500}}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "test-model")

	service := NewAIService(db, settings)
	text, _, err := service.RunText(context.Background(), AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "zh-CN",
		Input:       "plain text",
		TemplateKey: "markdown_format",
	}, AIActor{Type: "user"})
	if err == nil {
		t.Fatalf("RunText() error is nil, text=%q; want upstream stream error", text)
	}
	var aiErr AIError
	if !errors.As(err, &aiErr) {
		t.Fatalf("RunText() error = %T %[1]v, want AIError", err)
	}
	if aiErr.Code != "error.ai_upstream_error" || aiErr.UpstreamStatus != http.StatusInternalServerError {
		t.Fatalf("AIError = %#v, want upstream error with 500", aiErr)
	}
	if !strings.Contains(aiErr.UpstreamDetail, "Internal server error") {
		t.Fatalf("UpstreamDetail = %q, want provider message", aiErr.UpstreamDetail)
	}
	var log domain.AIGenerationLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("missing generation log: %v", err)
	}
	if log.Status != "failed" || log.ErrorCode != "error.ai_upstream_error" {
		t.Fatalf("unexpected log status/error: %#v", log)
	}
	var meta map[string]any
	if err := json.Unmarshal(log.Metadata, &meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if meta["upstreamStatus"] != float64(http.StatusOK) {
		t.Fatalf("metadata upstreamStatus = %#v, want transport status 200", meta["upstreamStatus"])
	}
	if got := fmt.Sprint(meta["upstreamResponseSummary"]); !strings.Contains(got, "Internal server error") {
		t.Fatalf("upstreamResponseSummary = %q, want stream error payload", got)
	}
}

func TestAIServiceRunTextRetriesPublishTitleVisionStreamInternalErrorWithImages(t *testing.T) {
	requestBodies := make(chan map[string]any, 2)
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBodies <- body
		w.Header().Set("Content-Type", "text/event-stream")
		if calls.Add(1) == 1 {
			_, _ = fmt.Fprint(w, "data: {\"error\":{\"message\":\"Internal server error\",\"type\":\"internal_server_error\",\"code\":500}}\n\n")
			_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"{\\\"title\\\":\\\"Image stream title\\\"}\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "test-model")

	service := NewAIService(db, settings)
	text, _, err := service.RunText(context.Background(), AIRequest{
		Type:        AITaskPublishTitleGenerate,
		Locale:      "zh-CN",
		Input:       "已有标题：\n（空）\n\n已有详情正文：\n（空）\n\n可用于分析的图片数量：1",
		TemplateKey: "publish_title_generate",
		Images: []AIImageInput{{
			URL:  "https://cdn.example.test/post/image.jpg",
			Mime: "image/jpeg",
			Alt:  "image.jpg",
		}},
	}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("RunText() error = %v", err)
	}
	if text != `{"title":"Image stream title"}` {
		t.Fatalf("text = %q, want image retry JSON", text)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want 2", got)
	}
	first := <-requestBodies
	if !requestIncludesImageURL(first, "https://cdn.example.test/post/image.jpg") {
		t.Fatalf("first request should include image_url content: %#v", first)
	}
	second := <-requestBodies
	if !requestIncludesImageURL(second, "https://cdn.example.test/post/image.jpg") {
		t.Fatalf("second request should still include image_url content: %#v", second)
	}
	if _, ok := first["response_format"]; ok {
		t.Fatalf("publish generation should not force response_format by default: %#v", first)
	}
	if requestIncludesText(second, "纯文本兼容模式") {
		t.Fatalf("publish title retry should not use text-only image summary: %#v", second)
	}
}

func TestAIServiceRunTextFallsBackWhenStructuredJSONStreamFails(t *testing.T) {
	requestBodies := make(chan map[string]any, 2)
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBodies <- body
		w.Header().Set("Content-Type", "text/event-stream")
		if calls.Add(1) == 1 {
			_, _ = fmt.Fprint(w, "data: {\"error\":{\"message\":\"response_format json_object is not supported\",\"type\":\"invalid_request_error\",\"code\":400}}\n\n")
			_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
			return
		}
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"{\\\"title\\\":\\\"Plain JSON title\\\"}\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "test-model")

	service := NewAIService(db, settings)
	text, _, err := service.RunText(context.Background(), AIRequest{
		Type:        AITaskPublishTitleGenerate,
		Locale:      "zh-CN",
		Input:       "已有标题：\n（空）\n\n已有详情正文：\n（空）\n\n可用于分析的图片数量：0",
		TemplateKey: "publish_title_generate",
		Options:     AIOptions{StructuredJSON: boolPtr(true)},
	}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("RunText() error = %v", err)
	}
	if text != `{"title":"Plain JSON title"}` {
		t.Fatalf("text = %q, want fallback JSON", text)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want 2", got)
	}
	first := <-requestBodies
	if _, ok := first["response_format"]; !ok {
		t.Fatalf("first request missing response_format: %#v", first)
	}
	second := <-requestBodies
	if _, ok := second["response_format"]; ok {
		t.Fatalf("fallback request should omit response_format: %#v", second)
	}
}

func TestAIServiceGeneratePublishContentUsesPlainTextByDefault(t *testing.T) {
	requestBody := make(chan map[string]any, 2)
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBody <- body
		w.Header().Set("Content-Type", "text/event-stream")
		if calls.Add(1) == 1 {
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"自然一点的详情\"}}]}\n\n")
		} else {
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"自然一点的标题\"}}]}\n\n")
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "test-model")

	service := NewAIService(db, settings)
	result, err := service.GeneratePublishContent(context.Background(), AIPublishGenerationInput{
		Locale:     "zh-CN",
		NeedTitle:  true,
		NeedDetail: true,
		Title:      "旧标题",
		Images: []AIImageInput{{
			URL: "https://cdn.example.test/post/image.jpg",
		}},
	}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("GeneratePublishContent() error = %v", err)
	}
	if !result.GeneratedTitle || result.Title != "自然一点的标题" {
		t.Fatalf("result = %#v, want plain text title", result)
	}
	if !result.GeneratedDetail || result.Detail != "自然一点的详情" {
		t.Fatalf("result = %#v, want plain text detail", result)
	}
	detailBody := <-requestBody
	titleBody := <-requestBody
	if _, ok := detailBody["response_format"]; ok {
		t.Fatalf("publish detail generation should not force response_format by default: %#v", detailBody)
	}
	if !requestIncludesImageURL(detailBody, "https://cdn.example.test/post/image.jpg") {
		t.Fatalf("publish detail generation request should include image_url content: %#v", detailBody)
	}
	if requestIncludesImageURL(titleBody, "https://cdn.example.test/post/image.jpg") {
		t.Fatalf("publish title generation should not include image_url content: %#v", titleBody)
	}
	if !requestIncludesText(titleBody, "最终详情正文") {
		t.Fatalf("publish title generation should summarize from detail: %#v", titleBody)
	}
}

func TestAIServiceGeneratePublishContentUsesTwoStepRequestsWithImages(t *testing.T) {
	requestBody := make(chan map[string]any, 2)
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBody <- body
		w.Header().Set("Content-Type", "text/event-stream")
		if calls.Add(1) == 1 {
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"白色花瓶和木桌都很清楚，整体很干净。\"}}]}\n\n")
		} else {
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"窗边这束花很亮\"}}]}\n\n")
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "test-model")

	service := NewAIService(db, settings)
	result, err := service.GeneratePublishContent(context.Background(), AIPublishGenerationInput{
		Locale:     "zh-CN",
		NeedTitle:  true,
		NeedDetail: true,
		Images: []AIImageInput{{
			URL: "https://cdn.example.test/post/image.jpg",
		}},
	}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("GeneratePublishContent() error = %v", err)
	}
	if !result.GeneratedTitle || result.Title != "窗边这束花很亮" {
		t.Fatalf("result title = %#v, want summarized title", result)
	}
	if !result.GeneratedDetail || result.Detail != "白色花瓶和木桌都很清楚，整体很干净。" {
		t.Fatalf("result detail = %#v, want generated detail", result)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want detail and title requests", got)
	}
	detailBody := <-requestBody
	titleBody := <-requestBody
	if _, ok := detailBody["response_format"]; ok {
		t.Fatalf("publish detail generation should not force response_format by default: %#v", detailBody)
	}
	if !requestIncludesImageURL(detailBody, "https://cdn.example.test/post/image.jpg") {
		t.Fatalf("publish detail generation request should include image_url content: %#v", detailBody)
	}
	if requestIncludesImageURL(titleBody, "https://cdn.example.test/post/image.jpg") {
		t.Fatalf("publish title generation request should not include image_url content: %#v", titleBody)
	}
	if !requestIncludesText(titleBody, "最终详情正文") {
		t.Fatalf("publish title generation request should summarize from detail: %#v", titleBody)
	}
}

func TestAIServiceRunStreamDoesNotEchoPublishGenerationPromptWhenOutputBlank(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "test-model")

	input := "已有标题：\n你啊\n\n已有详情正文：\n是的\n\n可用于分析的图片数量：3"
	service := NewAIService(db, settings)
	text, _, err := service.RunText(context.Background(), AIRequest{
		Type:        AITaskPublishTitleGenerate,
		Locale:      "zh-CN",
		Input:       input,
		TemplateKey: "publish_title_generate",
		Options:     AIOptions{StructuredJSON: boolPtr(true)},
	}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("RunText() error = %v", err)
	}
	if text != "" {
		t.Fatalf("text = %q, want empty output instead of prompt echo", text)
	}
}

func TestAIServiceRunStreamKeepsTransformInputWhenOutputBlank(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "test-model")

	service := NewAIService(db, settings)
	text, _, err := service.RunText(context.Background(), AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "zh-CN",
		Input:       "保底原文",
		TemplateKey: "markdown_format",
	}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("RunText() error = %v", err)
	}
	if text != "保底原文" {
		t.Fatalf("text = %q, want transform fallback input", text)
	}
}

func TestAIServiceRunStreamContinuesLongCustomGeneration(t *testing.T) {
	var requestCount atomic.Int32
	requestBodies := make(chan map[string]any, 4)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		requestBodies <- body
		count := requestCount.Add(1)
		w.Header().Set("Content-Type", "text/event-stream")
		var text string
		switch count {
		case 1:
			text = "第一段开头"
		case 2:
			text = "第二段承接"
		default:
			text = "第三段收束"
		}
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", text)
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingContentFormat, map[string]any{
		"custom": map[string]any{
			"continuation": map[string]any{
				"enabled":      true,
				"triggerChars": 1000,
				"maxRounds":    2,
				"contextChars": 800,
			},
		},
	})
	service := NewAIService(db, settings)

	events := []AIStreamEvent{}
	err := service.RunStream(context.Background(), AIRequest{
		Type:   AITaskPostCustomGenerate,
		Locale: "zh-CN",
		Input:  "提纲",
		Variables: map[string]any{
			"customPrompt": "写 1 万字连载故事",
		},
	}, AIActor{Type: "user"}, func(event AIStreamEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if requestCount.Load() != 3 {
		t.Fatalf("request count = %d, want first generation plus two continuations", requestCount.Load())
	}
	var final AIStreamEvent
	stages := map[string]bool{}
	for _, event := range events {
		if event.Type == "progress" {
			stages[event.Stage] = true
		}
		if event.Type == "final" {
			final = event
		}
	}
	if final.Text != "第一段开头\n\n第二段承接\n\n第三段收束" {
		t.Fatalf("final text = %q, want continued output", final.Text)
	}
	if final.Summary["continuationRounds"] != 2 {
		t.Fatalf("summary continuation rounds = %#v, want 2; summary=%#v", final.Summary["continuationRounds"], final.Summary)
	}
	if !stages["continuation_start"] || !stages["continuation_connecting"] || !stages["continuation_done"] {
		t.Fatalf("stages = %#v, want continuation progress stages", stages)
	}
	<-requestBodies
	secondBody := <-requestBodies
	<-requestBodies
	if !requestIncludesText(secondBody, "Continuation control") || !requestIncludesText(secondBody, "Already generated ending") ||
		!requestIncludesText(secondBody, "第一段开头") {
		t.Fatalf("second request body missing compressed continuation context: %#v", secondBody)
	}
}

func TestAIServiceRunStreamSanitizesLeakedReasoning(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, delta := range []string{"<thi", "nk>hidden</think>\n", "Rea", "soning: internal\n\n", "# Final"} {
			payload, _ := json.Marshal(map[string]any{
				"choices": []map[string]any{
					{"delta": map[string]any{"content": delta}},
				},
			})
			_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	service := NewAIService(db, settings)

	var finalText string
	var streamedText string
	err := service.RunStream(context.Background(), AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "en",
		Input:       "plain text",
		TemplateKey: "markdown_format",
	}, AIActor{Type: "user"}, func(event AIStreamEvent) error {
		if event.Type == "chunk_delta" {
			streamedText += event.Delta
		}
		if event.Type == "final" {
			finalText = event.Text
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if finalText != "# Final" {
		t.Fatalf("final text = %q, want sanitized markdown", finalText)
	}
	if streamedText != "# Final" {
		t.Fatalf("streamed text = %q, want sanitized markdown", streamedText)
	}
	var log domain.AIGenerationLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("missing generation log: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(log.Metadata, &meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if meta["imageSendSuccessCount"] != float64(0) {
		t.Fatalf("metadata imageSendSuccessCount = %#v, want 0", meta["imageSendSuccessCount"])
	}
}

func TestAIServiceRunStreamStripsNullBytesBeforePersisting(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		payload, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{
				{"delta": map[string]any{"content": "A\x00B"}},
			},
		})
		_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	service := NewAIService(db, settings)

	var finalText string
	err := service.RunStream(context.Background(), AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "en",
		Input:       "plain text\x00",
		TemplateKey: "markdown_format",
	}, AIActor{Type: "user"}, func(event AIStreamEvent) error {
		if event.Type == "final" {
			finalText = event.Text
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if strings.ContainsRune(finalText, 0) || finalText != "AB" {
		t.Fatalf("final text = %q, want NUL stripped", finalText)
	}
	var log domain.AIGenerationLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("missing generation log: %v", err)
	}
	if strings.ContainsRune(log.InputSummary, 0) || strings.ContainsRune(log.OutputSummary, 0) {
		t.Fatalf("log contains NUL: input=%q output=%q", log.InputSummary, log.OutputSummary)
	}
}

func TestAIServiceRunStreamEmitsReasoningAndThinkingParameter(t *testing.T) {
	requestBody := make(chan map[string]any, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBody <- body
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"checking\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"done\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingShowReasoning, true)
	settings.Set(context.Background(), AISettingThinkingParameterEnabled, true)
	settings.Set(context.Background(), AISettingThinkingEnabled, false)
	settings.Set(context.Background(), AISettingReasoningEffort, "high")
	settings.Set(context.Background(), AISettingModelParameters, map[string]any{
		"top_p":          json.Number("0.9"),
		"seed":           json.Number("42"),
		"stream":         false,
		"stream_options": map[string]any{"include_usage": false},
	})

	service := NewAIService(db, settings)
	events := []AIStreamEvent{}
	err := service.RunStream(context.Background(), AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "en",
		Input:       "plain text",
		TemplateKey: "markdown_format",
	}, AIActor{Type: "user"}, func(event AIStreamEvent) error {
		events = append(events, event)
		return nil
	})
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	body := <-requestBody
	if value, ok := body["enable_thinking"].(bool); !ok || value {
		t.Fatalf("enable_thinking = %#v, want false", body["enable_thinking"])
	}
	if body["reasoning_effort"] != "high" {
		t.Fatalf("reasoning_effort = %#v, want high", body["reasoning_effort"])
	}
	if body["top_p"] != 0.9 || body["seed"] != float64(42) {
		t.Fatalf("custom model parameters missing: %#v", body)
	}
	if body["stream"] != true {
		t.Fatalf("stream = %#v, want true", body["stream"])
	}
	if streamOptions, ok := body["stream_options"].(map[string]any); !ok || streamOptions["include_usage"] != true {
		t.Fatalf("stream_options = %#v, want include_usage true", body["stream_options"])
	}
	var reasoningDelta, finalText string
	for _, event := range events {
		if event.Type == "reasoning_delta" {
			reasoningDelta += event.ReasoningDelta
		}
		if event.Type == "final" {
			finalText = event.Text
		}
	}
	if reasoningDelta != "checking" || finalText != "done" {
		t.Fatalf("reasoning/final = %q/%q; events=%#v", reasoningDelta, finalText, events)
	}
}

func TestAIServiceRetriesPublishGenerationWhenReasoningOnly(t *testing.T) {
	requestBody := make(chan map[string]any, 2)
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBody <- body
		w.Header().Set("Content-Type", "text/event-stream")
		if calls.Add(1) == 1 {
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"正在分析图片，但还没给最终正文\"}}]}\n\n")
		} else {
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"{\\\"detail\\\":\\\"窗边的花和木桌都很清楚。\\\"}\"}}]}\n\n")
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "nvidia/nemotron-3-nano-omni-30b-a3b-reasoning")
	settings.Set(context.Background(), AISettingShowReasoning, true)
	settings.Set(context.Background(), AISettingThinkingParameterEnabled, false)
	settings.Set(context.Background(), AISettingThinkingEnabled, false)

	service := NewAIService(db, settings)
	text, _, err := service.RunText(context.Background(), AIRequest{
		Type:        AITaskPublishDetailGenerate,
		Locale:      "zh-CN",
		Input:       "已有标题：\n（空）\n\n已有详情正文：\n（空）\n\n可用于分析的图片数量：1",
		TemplateKey: "publish_detail_generate",
		Images: []AIImageInput{{
			URL: "https://cdn.example.test/post/image.jpg",
		}},
	}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("RunText() error = %v", err)
	}
	if text != `{"detail":"窗边的花和木桌都很清楚。"}` {
		t.Fatalf("text = %q, want detail JSON from retry", text)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls = %d, want reasoning-only retry", got)
	}
	firstBody := <-requestBody
	secondBody := <-requestBody
	if _, ok := firstBody["enable_thinking"]; ok {
		t.Fatalf("first enable_thinking = %#v, want omitted before retry", firstBody["enable_thinking"])
	}
	if _, ok := secondBody["enable_thinking"]; ok {
		t.Fatalf("second enable_thinking = %#v, want NVIDIA nested parameter", secondBody["enable_thinking"])
	}
	chatTemplate, ok := secondBody["chat_template_kwargs"].(map[string]any)
	if !ok || chatTemplate["enable_thinking"] != false {
		t.Fatalf("second chat_template_kwargs = %#v, want enable_thinking false", secondBody["chat_template_kwargs"])
	}
}

func TestAIServiceTemplateRuntimeOverridesAffectRequestAndMetadata(t *testing.T) {
	requestBody := make(chan map[string]any, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBody <- body
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"hidden\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"done\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingShowReasoning, true)
	settings.Set(context.Background(), AISettingThinkingParameterEnabled, false)
	settings.Set(context.Background(), AISettingReasoningEffort, "low")

	showReasoning := false
	thinkingParameterEnabled := true
	thinkingEnabled := true
	reasoningEffort := "medium"
	saved := settings.Set(context.Background(), AISettingPromptTemplates, map[string]AITemplateConfig{
		"markdown_format": {
			Enabled:         true,
			TaskType:        AITaskFormatMarkdown,
			SystemPrompt:    "format",
			UserPrompt:      "{{input}}",
			Prompt:          "{{input}}",
			Temperature:     0.4,
			MaxOutputTokens: 2048,
			RuntimeOverrides: AIRuntimeOverrides{
				Enabled:                  true,
				ShowReasoning:            &showReasoning,
				ThinkingParameterEnabled: &thinkingParameterEnabled,
				ThinkingEnabled:          &thinkingEnabled,
				ReasoningEffort:          &reasoningEffort,
				ModelParameters:          map[string]any{"top_p": 0.7},
			},
		},
	})
	if !saved {
		t.Fatal("failed to save prompt template")
	}

	service := NewAIService(db, settings)
	events := []AIStreamEvent{}
	if err := service.RunStream(context.Background(), AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "en",
		Input:       "plain text",
		TemplateKey: "markdown_format",
	}, AIActor{Type: "user"}, func(event AIStreamEvent) error {
		events = append(events, event)
		return nil
	}); err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}

	body := <-requestBody
	if body["enable_thinking"] != true || body["reasoning_effort"] != "medium" || body["top_p"] != 0.7 {
		t.Fatalf("request body missing runtime overrides: %#v", body)
	}
	for _, event := range events {
		if event.Type == "reasoning_delta" {
			t.Fatalf("reasoning_delta emitted despite template override disabling display: %#v", events)
		}
	}

	var log domain.AIGenerationLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("missing generation log: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(log.Metadata, &meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if meta["showReasoning"] != false || meta["thinkingParameterEnabled"] != true ||
		meta["thinkingEnabled"] != true || meta["reasoningEffort"] != "medium" {
		t.Fatalf("metadata missing effective runtime overrides: %#v", meta)
	}
	params := testMap(t, meta["modelParameters"])
	if params["top_p"] != float64(0.7) {
		t.Fatalf("metadata modelParameters = %#v, want top_p 0.7", params)
	}
}

func TestAIServiceRunStreamFormatsEveryLargeTextChunk(t *testing.T) {
	var requestCount int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"formatted-part-%d\"}}]}\n\n", requestCount)
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingChunkMaxChars, 500)

	input := strings.Repeat("A", 520) + "\n\n" + strings.Repeat("B", 520) + "\n\n" + strings.Repeat("C", 120)
	service := NewAIService(db, settings)
	var finalText string
	progressEvents := []AIStreamEvent{}
	err := service.RunStream(context.Background(), AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "en",
		Input:       input,
		TemplateKey: "markdown_format",
	}, AIActor{Type: "user"}, func(event AIStreamEvent) error {
		if event.Type == "progress" {
			progressEvents = append(progressEvents, event)
		}
		if event.Type == "final" {
			finalText = event.Text
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if requestCount != 5 {
		t.Fatalf("request count = %d, want 5", requestCount)
	}
	for _, want := range []string{"formatted-part-1", "formatted-part-2", "formatted-part-3", "formatted-part-4", "formatted-part-5"} {
		if !strings.Contains(finalText, want) {
			t.Fatalf("final text missing %q: %q", want, finalText)
		}
	}
	lastProgress := progressEvents[len(progressEvents)-1]
	if lastProgress.TotalChunks != 5 || lastProgress.ProcessedChars != lastProgress.TotalChars || lastProgress.Percent != 100 {
		t.Fatalf("last progress = %#v, want all chunks/chars complete", lastProgress)
	}
}

func TestAIServiceRunStreamClassifiesStreamDeadlineAsTimeout(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	service := NewAIService(db, settings)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, _, err := service.RunText(ctx, AIRequest{
		Type:        AITaskPublishTitleGenerate,
		Locale:      "en",
		Input:       "title input",
		TemplateKey: "publish_title_generate",
	}, AIActor{Type: "user"})
	if aiErrorCode(err) != "error.ai_timeout" {
		t.Fatalf("RunText() error = %v, code=%s, want error.ai_timeout", err, aiErrorCode(err))
	}
}

func TestAIServiceRunStreamKeepsAliveWhileDeltasArrive(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		for _, part := range []string{"one", " two", " three"} {
			_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", part)
			flusher.Flush()
			time.Sleep(120 * time.Millisecond)
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingTimeoutSeconds, 1)
	settings.Set(context.Background(), AISettingMaxRunSeconds, 5)
	service := NewAIService(db, settings)

	text, _, err := service.RunText(context.Background(), AIRequest{
		Type:        AITaskPublishTitleGenerate,
		Locale:      "en",
		Input:       "title input",
		TemplateKey: "publish_title_generate",
	}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("RunText() error = %v", err)
	}
	if text != "one two three" {
		t.Fatalf("text = %q, want streamed output", text)
	}
}

func TestAIServiceRunStreamRetriesPartialChunkWithoutDuplicateOutput(t *testing.T) {
	var requestCount int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "text/event-stream")
		if requestCount == 1 {
			_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial-\"}}]}\n\n")
			return
		}
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"complete\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	service := NewAIService(db, settings)

	var finalText string
	err := service.RunStream(context.Background(), AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "en",
		Input:       "plain text",
		TemplateKey: "markdown_format",
	}, AIActor{Type: "user"}, func(event AIStreamEvent) error {
		if event.Type == "final" {
			finalText = event.Text
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if requestCount != 2 {
		t.Fatalf("request count = %d, want retry once", requestCount)
	}
	if finalText != "complete" {
		t.Fatalf("final text = %q, want retry output without partial duplicate", finalText)
	}
}

func TestAIServiceRunStreamSplitsLargeChunkAfterRetryableFailure(t *testing.T) {
	var requestCount int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "text/event-stream")
		if requestCount <= aiChunkPrimaryAttempts {
			_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"failed-%d\"}}]}\n\n", requestCount)
			return
		}
		_, _ = fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"subpart-%d\"}}]}\n\n", requestCount-aiChunkPrimaryAttempts)
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingChunkMaxChars, 2000)
	service := NewAIService(db, settings)

	input := strings.Repeat("A", 1200) + "\n\n" + strings.Repeat("B", 1200)
	var final AIStreamEvent
	err := service.RunStream(context.Background(), AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "en",
		Input:       input,
		TemplateKey: "markdown_format",
	}, AIActor{Type: "user"}, func(event AIStreamEvent) error {
		if event.Type == "final" {
			final = event
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if !strings.Contains(final.Text, "subpart-1") || !strings.Contains(final.Text, "subpart-2") {
		t.Fatalf("final text = %q, want split retry outputs", final.Text)
	}
	fallbacks, ok := final.Summary["fallbackChunks"].([]map[string]any)
	if !ok || len(fallbacks) != 1 || fallbacks[0]["mode"] != "split_retry" {
		t.Fatalf("summary fallback = %#v, want split_retry entry", final.Summary["fallbackChunks"])
	}
}

func TestAIServiceRunStreamFallsBackToOriginalTextWhenChunkStillFails(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"}}]}\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingChunkMaxChars, 500)
	service := NewAIService(db, settings)

	input := strings.Repeat("原文", 120)
	var final AIStreamEvent
	err := service.RunStream(context.Background(), AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "zh-CN",
		Input:       input,
		TemplateKey: "markdown_format",
	}, AIActor{Type: "user"}, func(event AIStreamEvent) error {
		if event.Type == "final" {
			final = event
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunStream() error = %v", err)
	}
	if final.Text != input {
		t.Fatalf("final text = %q, want original input fallback", final.Text)
	}
	if got := final.Summary["fallbackChunkCount"]; got != 1 {
		t.Fatalf("fallback count = %#v, want 1; summary=%#v", got, final.Summary)
	}
	var log domain.AIGenerationLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("missing generation log: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(log.Metadata, &meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if meta["fallbackChunkCount"] != float64(1) {
		t.Fatalf("log fallback count = %#v, want 1; meta=%#v", meta["fallbackChunkCount"], meta)
	}
}

func TestAIServiceRunJobPersistsReasoningWithoutDeltaOutput(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"reasoning_content\":\"thinking\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"final output\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	if err := db.AutoMigrate(&domain.AIJob{}); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingShowReasoning, true)
	service := NewAIService(db, settings)

	job, err := service.CreateJob(context.Background(), AIJobCreateInput{Request: AIRequest{
		Type:        AITaskPublishDetailGenerate,
		Locale:      "zh-CN",
		Input:       "生成详情",
		TemplateKey: "publish_detail_generate",
	}}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	if err := service.RunJobByID(context.Background(), job.JobID); err != nil {
		t.Fatalf("RunJobByID() error = %v", err)
	}
	updated, err := service.Job(context.Background(), job.JobID, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("Job() error = %v", err)
	}
	if updated.Status != AIJobStatusCompleted || updated.Output != "final output" {
		t.Fatalf("job status/output = %s/%q, want completed/final output", updated.Status, updated.Output)
	}
	if updated.Reasoning != "thinking" {
		t.Fatalf("job reasoning = %q, want thinking", updated.Reasoning)
	}
}

func TestAIServiceRunJobPersistsLiveDeltaPreview(t *testing.T) {
	deltaWritten := make(chan struct{})
	var deltaOnce sync.Once
	releaseDone := make(chan struct{})
	var releaseOnce sync.Once
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"live preview\"}}]}\n\n")
		w.(http.Flusher).Flush()
		deltaOnce.Do(func() { close(deltaWritten) })
		<-releaseDone
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()
	defer releaseOnce.Do(func() { close(releaseDone) })

	db := newAITestDB(t)
	if err := db.AutoMigrate(&domain.AIJob{}); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	service := NewAIService(db, settings)

	job, err := service.CreateJob(context.Background(), AIJobCreateInput{Request: AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "en",
		Input:       "plain text",
		TemplateKey: "markdown_format",
	}}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- service.RunJobByID(context.Background(), job.JobID)
	}()
	<-deltaWritten
	deadline := time.Now().Add(2 * time.Second)
	for {
		preview, err := service.Job(context.Background(), job.JobID, AIActor{Type: "user"})
		if err != nil {
			t.Fatalf("Job() error = %v", err)
		}
		if preview.Output == "live preview" && preview.Status == AIJobStatusRunning && preview.CompletionTokens > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("preview output/status/tokens = %q/%s/%d, want live preview/running/positive", preview.Output, preview.Status, preview.CompletionTokens)
		}
		time.Sleep(20 * time.Millisecond)
	}
	releaseOnce.Do(func() { close(releaseDone) })
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("RunJobByID() error = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunJobByID() did not finish")
	}
}

func TestAIJobTokenRateWindowUsesTenSecondAverage(t *testing.T) {
	start := time.Unix(100, 0)
	window := aiJobTokenRateWindow{}
	window.seed(start)
	if got := window.record(start.Add(5*time.Second), 50); got != 10 {
		t.Fatalf("5s rate = %v, want 10", got)
	}
	if got := window.record(start.Add(15*time.Second), 150); got != 10 {
		t.Fatalf("10s rolling rate = %v, want 10", got)
	}
}

func TestAIServiceQueuedProgressIncludesActiveJobTokens(t *testing.T) {
	db := newAITestDB(t)
	if err := db.AutoMigrate(&domain.AIJob{}); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(db, nil)
	service := NewAIService(db, settings)
	actorID := int64(7)
	active := domain.AIJob{
		JobID:            "active-job",
		TaskType:         AITaskFormatMarkdown,
		TemplateKey:      "markdown_format",
		ActorType:        "user",
		ActorID:          &actorID,
		Status:           AIJobStatusRunning,
		Stage:            "chunk_start",
		CompletionTokens: 42,
		TokensPerSecond:  4.2,
		CreatedAt:        time.Now().Add(-time.Minute),
	}
	if err := db.Create(&active).Error; err != nil {
		t.Fatal(err)
	}
	waiting := domain.AIJob{
		JobID:       "waiting-job",
		TaskType:    AITaskFormatMarkdown,
		TemplateKey: "markdown_format",
		ActorType:   "user",
		Status:      AIJobStatusQueued,
		Stage:       "queued",
		CreatedAt:   time.Now(),
	}
	if err := db.Create(&waiting).Error; err != nil {
		t.Fatal(err)
	}
	event := AIStreamEvent{Type: "progress", Stage: "queued", QueuePosition: 1, QueueTotal: 1}
	service.enrichQueuedAIStreamEvent(context.Background(), waiting, &event)
	if event.ActiveJobID != active.JobID || event.ActiveActorID != actorID || event.ActiveTokens != 42 || event.ActiveRate != 4.2 {
		t.Fatalf("active event = %#v, want active job token snapshot", event)
	}
	metadata := aiStreamQueueJobMetadata(event)
	activeMeta, _ := metadata["activeJob"].(map[string]any)
	if activeMeta == nil || activeMeta["generatedTokens"] != 42 {
		t.Fatalf("active metadata = %#v, want generatedTokens 42", metadata["activeJob"])
	}
}

func TestAIServiceRunJobBroadcastsLiveEvents(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"live\"}}]}\n\n")
		w.(http.Flusher).Flush()
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\" stream\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	if err := db.AutoMigrate(&domain.AIJob{}); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	service := NewAIService(db, settings)

	job, err := service.CreateJob(context.Background(), AIJobCreateInput{Request: AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "en",
		Input:       "plain text",
		TemplateKey: "markdown_format",
	}}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	ctx := t.Context()
	events, unsubscribe := service.SubscribeJobEvents(ctx, job.JobID)
	defer unsubscribe()
	errCh := make(chan error, 1)
	go func() {
		errCh <- service.RunJobByID(context.Background(), job.JobID)
	}()
	var deltas strings.Builder
	var final string
	deadline := time.After(2 * time.Second)
	for final == "" {
		select {
		case event, ok := <-events:
			if !ok {
				t.Fatal("event subscription closed before final event")
			}
			if event.JobID != job.JobID {
				t.Fatalf("event job id = %q, want %q", event.JobID, job.JobID)
			}
			switch event.Type {
			case "chunk_delta":
				deltas.WriteString(event.Delta)
			case "final":
				final = event.Text
			}
		case <-deadline:
			t.Fatalf("timed out waiting for live events; deltas=%q final=%q", deltas.String(), final)
		}
	}
	if deltas.String() != "live stream" || final != "live stream" {
		t.Fatalf("broadcast deltas/final = %q/%q, want live stream/live stream", deltas.String(), final)
	}
	if err := <-errCh; err != nil {
		t.Fatalf("RunJobByID() error = %v", err)
	}
}

func TestAIServiceRunStreamEmitsQueueProgress(t *testing.T) {
	requestStarted := make(chan struct{}, 1)
	releaseUpstream := make(chan struct{})
	var once sync.Once
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		once.Do(func() {
			requestStarted <- struct{}{}
			<-releaseUpstream
		})
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingModel, "test-model")
	settings.Set(context.Background(), AISettingConcurrency, 1)
	service := NewAIService(db, settings)

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- service.RunStream(context.Background(), AIRequest{
			Type:        AITaskFormatMarkdown,
			Locale:      "en",
			Input:       "first",
			TemplateKey: "markdown_format",
		}, AIActor{Type: "user"}, func(event AIStreamEvent) error { return nil })
	}()

	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("first request did not reach upstream")
	}

	events := []AIStreamEvent{}
	secondDone := make(chan error, 1)
	go func() {
		secondDone <- service.RunStream(context.Background(), AIRequest{
			Type:        AITaskFormatMarkdown,
			Locale:      "en",
			Input:       "second",
			TemplateKey: "markdown_format",
		}, AIActor{Type: "user"}, func(event AIStreamEvent) error {
			events = append(events, event)
			return nil
		})
	}()

	deadline := time.After(2 * time.Second)
	for {
		queued := false
		for _, event := range events {
			if event.Type == "progress" && event.Stage == "queued" && event.QueuePosition == 1 && event.QueueTotal == 1 {
				queued = true
			}
		}
		if queued {
			break
		}
		select {
		case <-deadline:
			close(releaseUpstream)
			t.Fatalf("second request did not emit queued progress: %#v", events)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	close(releaseUpstream)
	if err := <-firstDone; err != nil {
		t.Fatalf("first RunStream() error = %v", err)
	}
	if err := <-secondDone; err != nil {
		t.Fatalf("second RunStream() error = %v", err)
	}
}

func TestAIServiceRunJobKeepsGateWaiterQueued(t *testing.T) {
	requestStarted := make(chan struct{}, 1)
	releaseUpstream := make(chan struct{})
	var releaseOnce sync.Once
	var upstreamRequests int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if atomic.AddInt32(&upstreamRequests, 1) == 1 {
			requestStarted <- struct{}{}
			<-releaseUpstream
		}
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()
	defer releaseOnce.Do(func() { close(releaseUpstream) })

	db := newAITestDB(t)
	if err := db.AutoMigrate(&domain.AIJob{}); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingConcurrency, 1)
	service := NewAIService(db, settings)

	firstJob := createAIJobForRunTest(t, service, "first")
	secondJob := createAIJobForRunTest(t, service, "second")

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- service.RunJobByID(context.Background(), firstJob.JobID)
	}()
	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("first job did not reach upstream")
	}

	secondDone := make(chan error, 1)
	go func() {
		secondDone <- service.RunJobByID(context.Background(), secondJob.JobID)
	}()

	waitForAIJobState(t, service, secondJob.JobID, func(job domain.AIJob) bool {
		return job.Status == AIJobStatusQueued && job.Stage == "queued"
	})
	firstSnapshot, err := service.Job(context.Background(), firstJob.JobID, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("first Job() error = %v", err)
	}
	if firstSnapshot.Status != AIJobStatusRunning {
		t.Fatalf("first job status = %s, want running", firstSnapshot.Status)
	}
	if firstSnapshot.EstimatedTokens <= 0 {
		t.Fatalf("first job estimated tokens = %d, want positive while running", firstSnapshot.EstimatedTokens)
	}
	secondSnapshot, err := service.Job(context.Background(), secondJob.JobID, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("second Job() error = %v", err)
	}
	if secondSnapshot.Status == AIJobStatusRunning {
		t.Fatal("second job was marked running while still waiting for AI concurrency")
	}
	if secondSnapshot.EstimatedTokens <= 0 {
		t.Fatalf("second job estimated tokens = %d, want queued job estimate preserved", secondSnapshot.EstimatedTokens)
	}

	releaseOnce.Do(func() { close(releaseUpstream) })
	if err := <-firstDone; err != nil {
		t.Fatalf("first RunJobByID() error = %v", err)
	}
	if err := <-secondDone; err != nil {
		t.Fatalf("second RunJobByID() error = %v", err)
	}
}

func TestAIServiceRunJobHonorsTemplateConcurrency(t *testing.T) {
	requestStarted := make(chan struct{}, 1)
	releaseUpstream := make(chan struct{})
	var releaseOnce sync.Once
	var upstreamRequests int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if atomic.AddInt32(&upstreamRequests, 1) == 1 {
			requestStarted <- struct{}{}
			<-releaseUpstream
		}
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()
	defer releaseOnce.Do(func() { close(releaseUpstream) })

	db := newAITestDB(t)
	if err := db.AutoMigrate(&domain.AIJob{}); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	templates := DefaultAIPromptTemplates()
	markdownTemplate := templates["markdown_format"]
	markdownTemplate.Concurrency = 1
	templates["markdown_format"] = markdownTemplate
	settings.Set(context.Background(), AISettingPromptTemplates, templates)
	service := NewAIService(db, settings)

	firstJob := createAIJobForRunTest(t, service, "first")
	secondJob := createAIJobForRunTest(t, service, "second")

	firstDone := make(chan error, 1)
	go func() {
		firstDone <- service.RunJobByID(context.Background(), firstJob.JobID)
	}()
	select {
	case <-requestStarted:
	case <-time.After(2 * time.Second):
		t.Fatal("first job did not reach upstream")
	}

	secondDone := make(chan error, 1)
	go func() {
		secondDone <- service.RunJobByID(context.Background(), secondJob.JobID)
	}()

	waitForAIJobState(t, service, secondJob.JobID, func(job domain.AIJob) bool {
		return job.Status == AIJobStatusQueued && job.Stage == "queued"
	})
	if got := atomic.LoadInt32(&upstreamRequests); got != 1 {
		t.Fatalf("upstream requests while template gate is full = %d, want 1", got)
	}

	releaseOnce.Do(func() { close(releaseUpstream) })
	if err := <-firstDone; err != nil {
		t.Fatalf("first RunJobByID() error = %v", err)
	}
	if err := <-secondDone; err != nil {
		t.Fatalf("second RunJobByID() error = %v", err)
	}
}

func TestAIServiceRunJobIdleTimeoutClearsQueueMetadata(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	if err := db.AutoMigrate(&domain.AIJob{}); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingTimeoutSeconds, 1)
	service := NewAIService(db, settings)

	job := createAIJobForRunTest(t, service, "timeout")
	if err := db.Model(&domain.AIJob{}).Where("id = ?", job.ID).Update("metadata", jsonData(map[string]any{
		"queueJob": map[string]any{"jobId": job.JobID, "state": "queued", "queuePosition": 1, "queueCount": 1},
	})).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.RunJobByID(context.Background(), job.JobID); err != nil {
		t.Fatalf("RunJobByID() error = %v", err)
	}
	updated, err := service.Job(context.Background(), job.JobID, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("Job() error = %v", err)
	}
	if updated.Status != AIJobStatusFailed || updated.ErrorCode != "error.ai_timeout" {
		t.Fatalf("job status/error = %s/%s, want failed/error.ai_timeout", updated.Status, updated.ErrorCode)
	}
	metadata := jsonMapFromLog(updated.Metadata)
	if _, ok := metadata["queueJob"]; ok {
		t.Fatalf("queue metadata was not cleared: %#v", metadata)
	}
}

func TestAIServiceRunJobBroadcastsUpstreamEvents(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"live\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db := newAITestDB(t)
	if err := db.AutoMigrate(&domain.AIJob{}); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	service := NewAIService(db, settings)
	job := createAIJobForRunTest(t, service, "upstream")

	events, unsubscribe := service.SubscribeJobEvents(t.Context(), job.JobID)
	defer unsubscribe()
	errCh := make(chan error, 1)
	go func() {
		errCh <- service.RunJobByID(context.Background(), job.JobID)
	}()

	upstreamSeen := false
	finalSeen := false
	for !finalSeen {
		select {
		case event := <-events:
			if event.Type == "upstream_event" {
				upstreamSeen = upstreamSeen || event.Upstream["phase"] == "stream_data" || event.Upstream["phase"] == "decoded_delta"
			}
			if event.Type == "final" {
				finalSeen = true
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timed out waiting for upstream event")
		}
	}
	if !upstreamSeen {
		t.Fatal("did not receive upstream_event from job stream")
	}
	if err := <-errCh; err != nil {
		t.Fatalf("RunJobByID() error = %v", err)
	}
}

func TestAIServiceMasksSecret(t *testing.T) {
	if got := MaskSecret("sk-1234567890"); got != "sk-1••••7890" {
		t.Fatalf("MaskSecret() = %q", got)
	}
}

func TestQueueServiceAIImageURLSignsLocalFilesForModelAccess(t *testing.T) {
	queue := &QueueService{cfg: config.Config{
		API: config.APIConfig{BaseURL: "https://api.example.test"},
		Upload: config.UploadConfig{
			LocalBase:   "http://localhost:3001",
			FileSigning: config.UploadFileSigningConfig{Secret: "file-secret", TTL: time.Hour},
		},
		Frontend: config.FrontendConfig{BaseURL: "https://front.example.test"},
	}}

	for _, raw := range []string{
		"/api/file/images/cover.webp",
		"api/file/images/cover.webp",
		"http://localhost:3001/api/file/images/cover.webp",
	} {
		got := queue.aiImageURL(raw)
		if !strings.HasPrefix(got, "https://api.example.test/api/file/images/cover.webp?") ||
			!strings.Contains(got, "pvimg_exp=") || !strings.Contains(got, "sign=") {
			t.Fatalf("aiImageURL(%q) = %q, want signed absolute API URL", raw, got)
		}
	}

	if got := queue.aiImageURL("https://cdn.example.test/public.webp?x=1"); got != "https://cdn.example.test/public.webp?x=1" {
		t.Fatalf("external image URL changed: %q", got)
	}

	got := queue.aiImageURL("https://xse.example.test/api/file/images/cover.webp?stale=1#frag")
	if !strings.HasPrefix(got, "https://xse.example.test/api/file/images/cover.webp?") ||
		!strings.Contains(got, "pvimg_exp=") || !strings.Contains(got, "sign=") ||
		strings.Contains(got, "stale=1") || strings.Contains(got, "#frag") {
		t.Fatalf("absolute local file URL was not re-signed cleanly: %q", got)
	}

	inputs := queue.aiImageInputs(context.Background(), []domain.PostImage{{ImageURL: "/api/file/images/missing.webp"}})
	if len(inputs) != 1 {
		t.Fatalf("aiImageInputs() = %d inputs, want 1", len(inputs))
	}
	if inputs[0].DataURL != "" {
		t.Fatalf("aiImageInputs() unexpectedly inlined missing image: %#v", inputs[0])
	}
	if !strings.HasPrefix(inputs[0].URL, "https://api.example.test/api/file/images/missing.webp?") ||
		!strings.Contains(inputs[0].URL, "pvimg_exp=") || !strings.Contains(inputs[0].URL, "sign=") {
		t.Fatalf("aiImageInputs() did not fall back to signed absolute URL: %#v", inputs[0])
	}
}

func TestQueueServiceCreateAIPostAutoCommentCreatesBotCommentWithImageContext(t *testing.T) {
	requestBody := make(chan map[string]any, 1)
	var upstreamRequests int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&upstreamRequests, 1)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBody <- body
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Great detail on the cover image.\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Post{},
		&domain.Comment{},
		&domain.Like{},
		&domain.Notification{},
		&domain.PostImage{},
		&domain.Tag{},
		&domain.PostTag{},
		&domain.AIGenerationLog{},
		&domain.SystemSetting{},
	); err != nil {
		t.Fatal(err)
	}
	author := domain.User{ID: 10, UserID: "author", Nickname: "Author", IsActive: true, AIAutoCommentEnabled: true}
	bot := domain.User{ID: 20, UserID: "ai-bot", Nickname: "AI Bot", IsActive: true}
	otherBot := domain.User{ID: 21, UserID: "ai-bot-2", Nickname: "AI Bot 2", IsActive: true}
	post := domain.Post{ID: 30, UserID: author.ID, Title: "A note", Content: "Look at this image", Type: 1, Visibility: "public", QualityLevel: "none"}
	if err := db.Create(&author).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&bot).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&otherBot).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	imageDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(imageDir, "cover.webp"), tinyPNGBytes(t), 0600); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.PostImage{PostID: post.ID, ImageURL: "/api/file/images/cover.webp", IsFreePreview: true, SortOrder: 1}).Error; err != nil {
		t.Fatal(err)
	}

	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingAutoCommentEnabled, true)
	settings.Set(context.Background(), AISettingAutoCommentBotUserID, bot.ID)
	settings.Set(context.Background(), AISettingAutoCommentMaxImages, 1)
	settings.Set(context.Background(), AISettingAutoCommentImageMode, AIImageSelectionRandom)
	ai := NewAIService(db, settings)
	queue := &QueueService{
		db:       db,
		settings: settings,
		ai:       ai,
		cfg: config.Config{
			API:    config.APIConfig{BaseURL: "https://api.example.test"},
			Upload: config.UploadConfig{Image: config.UploadImageConfig{LocalUploadDir: imageDir}},
		},
	}

	commentID, err := queue.createAIPostAutoComment(context.Background(), aiPostAutoCommentTaskPayload{PostID: post.ID, BotUserID: bot.ID})
	if err != nil {
		t.Fatalf("createAIPostAutoComment() error = %v", err)
	}
	if commentID == 0 {
		t.Fatal("comment id is zero")
	}
	var comment domain.Comment
	if err := db.Where("id = ?", commentID).First(&comment).Error; err != nil {
		t.Fatal(err)
	}
	if comment.UserID != bot.ID || comment.PostID != post.ID || comment.Content != "Great detail on the cover image." {
		t.Fatalf("unexpected comment: %#v", comment)
	}
	if !commentHasAIAutoCommentMarker(comment.AuditResult) {
		t.Fatalf("comment audit_result missing AI marker: %s", string(comment.AuditResult))
	}
	var updated domain.Post
	if err := db.Where("id = ?", post.ID).First(&updated).Error; err != nil {
		t.Fatal(err)
	}
	if updated.CommentCount != 1 {
		t.Fatalf("comment_count = %d, want 1", updated.CommentCount)
	}
	body := <-requestBody
	if !requestIncludesDataImage(body) {
		t.Fatalf("request body does not include inline image data: %#v", body)
	}
	var aiLog domain.AIGenerationLog
	if err := db.Where("task_type = ?", AITaskPostAutoComment).First(&aiLog).Error; err != nil {
		t.Fatalf("missing ai generation log: %v", err)
	}
	var aiLogMeta map[string]any
	if err := json.Unmarshal(aiLog.Metadata, &aiLogMeta); err != nil {
		t.Fatalf("decode ai log metadata: %v", err)
	}
	if aiLogMeta["imageSendSuccessCount"] != float64(1) {
		t.Fatalf("ai log imageSendSuccessCount = %#v, want 1", aiLogMeta["imageSendSuccessCount"])
	}
	if aiLogMeta["imageSelectionMode"] != AIImageSelectionRandom {
		t.Fatalf("ai log imageSelectionMode = %#v, want random", aiLogMeta["imageSelectionMode"])
	}
	if !requestIncludesText(body, "Images attached: 1") {
		t.Fatalf("request body missing selected image count in prompt: %#v", body)
	}
	settings.Set(context.Background(), AISettingAutoCommentBotUserID, otherBot.ID)
	_, err = queue.createAIPostAutoComment(context.Background(), aiPostAutoCommentTaskPayload{PostID: post.ID, BotUserID: otherBot.ID})
	if err == nil {
		t.Fatal("second createAIPostAutoComment() error is nil, want skip")
	}
	var commentCount int64
	if err := db.Model(&domain.Comment{}).Where("post_id = ?", post.ID).Count(&commentCount).Error; err != nil {
		t.Fatal(err)
	}
	if commentCount != 1 {
		t.Fatalf("comment count = %d, want 1", commentCount)
	}
	if got := atomic.LoadInt32(&upstreamRequests); got != 1 {
		t.Fatalf("upstream requests = %d, want 1", got)
	}
}

func TestQueueServiceAIPostAutoCommentMaxImagesZeroSendsNoImages(t *testing.T) {
	requestBody := make(chan map[string]any, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBody <- body
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Nice note.\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&domain.User{}, &domain.Post{}, &domain.Comment{}, &domain.Like{}, &domain.Notification{},
		&domain.PostImage{}, &domain.Tag{}, &domain.PostTag{}, &domain.AIGenerationLog{}, &domain.SystemSetting{},
	); err != nil {
		t.Fatal(err)
	}
	author := domain.User{ID: 110, UserID: "author-zero", Nickname: "Author", IsActive: true, AIAutoCommentEnabled: true}
	bot := domain.User{ID: 120, UserID: "ai-bot-zero", Nickname: "AI Bot", IsActive: true}
	post := domain.Post{ID: 130, UserID: author.ID, Title: "A note", Content: "Look at this image", Type: 1, Visibility: "public", QualityLevel: "none"}
	for _, row := range []any{&author, &bot, &post, &domain.PostImage{PostID: post.ID, ImageURL: "/api/file/images/cover.webp", IsFreePreview: true, SortOrder: 1}} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingAutoCommentEnabled, true)
	settings.Set(context.Background(), AISettingAutoCommentBotUserID, bot.ID)
	settings.Set(context.Background(), AISettingAutoCommentMaxImages, 0)
	queue := &QueueService{
		db:       db,
		settings: settings,
		ai:       NewAIService(db, settings),
		cfg:      config.Config{API: config.APIConfig{BaseURL: "https://api.example.test"}},
	}

	if _, err := queue.createAIPostAutoComment(context.Background(), aiPostAutoCommentTaskPayload{PostID: post.ID, BotUserID: bot.ID}); err != nil {
		t.Fatalf("createAIPostAutoComment() error = %v", err)
	}
	body := <-requestBody
	if requestIncludesImageURL(body, "https://api.example.test/api/file/images/cover.webp") || requestIncludesDataImage(body) {
		t.Fatalf("request body unexpectedly includes image payload: %#v", body)
	}
	var aiLog domain.AIGenerationLog
	if err := db.Where("task_type = ?", AITaskPostAutoComment).First(&aiLog).Error; err != nil {
		t.Fatalf("missing ai generation log: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(aiLog.Metadata, &meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if meta["imageSendSuccessCount"] != float64(0) {
		t.Fatalf("imageSendSuccessCount = %#v, want 0", meta["imageSendSuccessCount"])
	}
}

func TestQueueServiceAIPostAutoCommentUsesRandomBotUIDRange(t *testing.T) {
	requestBody := make(chan map[string]any, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBody <- body
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"范围账号评论\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&domain.User{}, &domain.Post{}, &domain.Comment{}, &domain.Like{}, &domain.Notification{},
		&domain.PostImage{}, &domain.Tag{}, &domain.PostTag{}, &domain.AIGenerationLog{}, &domain.SystemSetting{},
	); err != nil {
		t.Fatal(err)
	}
	author := domain.User{ID: 210, UserID: "author-range", Nickname: "Author", IsActive: true, AIAutoCommentEnabled: true}
	inactiveBot := domain.User{ID: 220, UserID: "inactive-bot", Nickname: "Inactive", IsActive: false}
	activeBot := domain.User{ID: 221, UserID: "active-bot", Nickname: "Active", IsActive: true}
	post := domain.Post{ID: 230, UserID: author.ID, Title: "A note", Content: "Range test", Type: 1, Visibility: "public", QualityLevel: "none"}
	for _, row := range []any{&author, &inactiveBot, &activeBot, &post} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}

	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingAutoCommentEnabled, true)
	settings.Set(context.Background(), AISettingAutoCommentBotUserID, 0)
	settings.Set(context.Background(), AISettingAutoCommentBotUserIDMin, inactiveBot.ID)
	settings.Set(context.Background(), AISettingAutoCommentBotUserIDMax, activeBot.ID)
	settings.Set(context.Background(), AISettingAutoCommentStyle, "humorous")
	queue := &QueueService{
		db:       db,
		settings: settings,
		ai:       NewAIService(db, settings),
		cfg:      config.Config{API: config.APIConfig{BaseURL: "https://api.example.test"}},
	}

	commentID, err := queue.createAIPostAutoComment(context.Background(), aiPostAutoCommentTaskPayload{PostID: post.ID})
	if err != nil {
		t.Fatalf("createAIPostAutoComment() error = %v", err)
	}
	var comment domain.Comment
	if err := db.Where("id = ?", commentID).First(&comment).Error; err != nil {
		t.Fatal(err)
	}
	if comment.UserID != activeBot.ID {
		t.Fatalf("comment user_id = %d, want active range bot %d", comment.UserID, activeBot.ID)
	}
	body := <-requestBody
	if !requestIncludesText(body, "witty") {
		t.Fatalf("request body missing humorous style instruction: %#v", body)
	}
}

func TestQueueServiceAIPostAutoCommentDisabledByAuthorSkipsUpstream(t *testing.T) {
	var upstreamRequests int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&upstreamRequests, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Should not happen\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&domain.User{}, &domain.Post{}, &domain.Comment{}, &domain.Like{}, &domain.Notification{},
		&domain.PostImage{}, &domain.Tag{}, &domain.PostTag{}, &domain.AIGenerationLog{}, &domain.SystemSetting{},
	); err != nil {
		t.Fatal(err)
	}
	author := domain.User{ID: 310, UserID: "author-off", Nickname: "Author", IsActive: true}
	bot := domain.User{ID: 320, UserID: "ai-bot-off", Nickname: "AI Bot", IsActive: true}
	post := domain.Post{ID: 330, UserID: author.ID, Title: "A note", Content: "Opted out", Type: 1, Visibility: "public", QualityLevel: "none"}
	for _, row := range []any{&author, &bot, &post} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&domain.User{}).Where("id = ?", author.ID).Update("ai_auto_comment_enabled", false).Error; err != nil {
		t.Fatal(err)
	}

	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingAutoCommentEnabled, true)
	settings.Set(context.Background(), AISettingAutoCommentBotUserID, bot.ID)
	queue := &QueueService{
		db:       db,
		settings: settings,
		ai:       NewAIService(db, settings),
		cfg:      config.Config{API: config.APIConfig{BaseURL: "https://api.example.test"}},
	}

	if _, err := queue.createAIPostAutoComment(context.Background(), aiPostAutoCommentTaskPayload{PostID: post.ID, BotUserID: bot.ID, AuthorID: author.ID}); err == nil {
		t.Fatal("createAIPostAutoComment() error is nil, want disabled-by-author skip")
	}
	if got := atomic.LoadInt32(&upstreamRequests); got != 0 {
		t.Fatalf("upstream requests = %d, want 0", got)
	}
	var commentCount int64
	if err := db.Model(&domain.Comment{}).Where("post_id = ?", post.ID).Count(&commentCount).Error; err != nil {
		t.Fatal(err)
	}
	if commentCount != 0 {
		t.Fatalf("comment count = %d, want 0", commentCount)
	}
}

func TestQueueServiceCreateAICommentReplyCreatesBotReplyWithThreadContext(t *testing.T) {
	requestBody := make(chan map[string]any, 1)
	var upstreamRequests int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&upstreamRequests, 1)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBody <- body
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Thanks for noticing that detail.\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&domain.User{}, &domain.Post{}, &domain.Comment{}, &domain.Like{}, &domain.Notification{},
		&domain.PostImage{}, &domain.Tag{}, &domain.PostTag{}, &domain.AIGenerationLog{}, &domain.SystemSetting{},
	); err != nil {
		t.Fatal(err)
	}
	author := domain.User{ID: 310, UserID: "author-reply", Nickname: "Author", IsActive: true}
	bot := domain.User{ID: 320, UserID: "ai-bot-reply", Nickname: "AI Bot", IsActive: true}
	user := domain.User{ID: 330, UserID: "reader-reply", Nickname: "Reader", IsActive: true}
	post := domain.Post{ID: 340, UserID: author.ID, Title: "Thread title", Content: "Thread body with image detail", Type: 1, Visibility: "public", QualityLevel: "none"}
	root := domain.Comment{ID: 350, PostID: post.ID, UserID: bot.ID, Content: "The cover light is really soft.", AuditStatus: 1, IsPublic: true, AuditResult: jsonData(map[string]any{
		"source":   aiAutoCommentAuditSource,
		"taskType": AITaskPostAutoComment,
	}), CreatedAt: time.Now().Add(-3 * time.Minute)}
	triggerParent := root.ID
	trigger := domain.Comment{ID: 351, PostID: post.ID, UserID: user.ID, ParentID: &triggerParent, Content: "Yes, the cover light is my favorite part.", AuditStatus: 1, IsPublic: true, CreatedAt: time.Now().Add(-2 * time.Minute)}
	for _, row := range []any{&author, &bot, &user, &post, &root, &trigger} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	imageDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(imageDir, "reply-cover.webp"), tinyPNGBytes(t), 0600); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.PostImage{PostID: post.ID, ImageURL: "/api/file/images/reply-cover.webp", IsFreePreview: true, SortOrder: 1}).Error; err != nil {
		t.Fatal(err)
	}

	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingCommentReply, AICommentReplyConfig{
		Enabled:                true,
		TemplateKey:            "comment_reply",
		DelaySeconds:           0,
		MaxImages:              1,
		ImageSelectionMode:     AIImageSelectionOrdered,
		Style:                  "bold",
		MaxRepliesPerAIComment: 2,
	})
	queue := &QueueService{
		db:       db,
		settings: settings,
		ai:       NewAIService(db, settings),
		cfg: config.Config{
			API:    config.APIConfig{BaseURL: "https://api.example.test"},
			Upload: config.UploadConfig{Image: config.UploadImageConfig{LocalUploadDir: imageDir}},
		},
	}

	replyID, err := queue.createAICommentReply(context.Background(), aiCommentReplyTaskPayload{TriggerCommentID: trigger.ID})
	if err != nil {
		t.Fatalf("createAICommentReply() error = %v", err)
	}
	if replyID == 0 {
		t.Fatal("reply id is zero")
	}
	var reply domain.Comment
	if err := db.Where("id = ?", replyID).First(&reply).Error; err != nil {
		t.Fatal(err)
	}
	if reply.UserID != bot.ID || reply.PostID != post.ID || reply.ParentID == nil || *reply.ParentID != trigger.ID || reply.Content != "Thanks for noticing that detail." {
		t.Fatalf("unexpected reply: %#v", reply)
	}
	if !commentHasAICommentReplyMarker(reply.AuditResult) {
		t.Fatalf("reply audit_result missing AI reply marker: %s", string(reply.AuditResult))
	}
	meta := aiCommentAuditMap(reply.AuditResult)
	if aiCommentMetaInt64(meta, "rootAICommentID") != root.ID || aiCommentMetaInt64(meta, "triggerCommentID") != trigger.ID {
		t.Fatalf("reply audit meta = %#v, want root %d trigger %d", meta, root.ID, trigger.ID)
	}
	if aiCommentMetaInt64(meta, "replySequence") != 1 {
		t.Fatalf("replySequence = %#v, want 1", meta["replySequence"])
	}
	body := <-requestBody
	for _, text := range []string{"Thread title", "Thread body with image detail", "The cover light is really soft.", "Yes, the cover light is my favorite part.", "Images attached: 1", "sharper, more energetic"} {
		if !requestIncludesText(body, text) {
			t.Fatalf("request body missing %q: %#v", text, body)
		}
	}
	if !requestIncludesDataImage(body) {
		t.Fatalf("request body does not include inline image data: %#v", body)
	}
	var aiLog domain.AIGenerationLog
	if err := db.Where("task_type = ?", AITaskCommentReply).First(&aiLog).Error; err != nil {
		t.Fatalf("missing ai generation log: %v", err)
	}
	var aiLogMeta map[string]any
	if err := json.Unmarshal(aiLog.Metadata, &aiLogMeta); err != nil {
		t.Fatalf("decode ai log metadata: %v", err)
	}
	if aiLogMeta["imageSendSuccessCount"] != float64(1) {
		t.Fatalf("ai log imageSendSuccessCount = %#v, want 1", aiLogMeta["imageSendSuccessCount"])
	}
	_, err = queue.createAICommentReply(context.Background(), aiCommentReplyTaskPayload{TriggerCommentID: trigger.ID})
	if err == nil {
		t.Fatal("duplicate createAICommentReply() error is nil, want skip")
	}
	if got := atomic.LoadInt32(&upstreamRequests); got != 1 {
		t.Fatalf("upstream requests = %d, want 1", got)
	}
}

func TestQueueServiceAICommentReplySkipsAtMaxReplies(t *testing.T) {
	var upstreamRequests int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&upstreamRequests, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Should not happen.\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.Post{}, &domain.Comment{}, &domain.Like{}, &domain.Notification{}, &domain.AIGenerationLog{}, &domain.SystemSetting{}); err != nil {
		t.Fatal(err)
	}
	author := domain.User{ID: 410, UserID: "author-max", Nickname: "Author", IsActive: true}
	bot := domain.User{ID: 420, UserID: "ai-bot-max", Nickname: "AI Bot", IsActive: true}
	user := domain.User{ID: 430, UserID: "reader-max", Nickname: "Reader", IsActive: true}
	post := domain.Post{ID: 440, UserID: author.ID, Title: "Max title", Content: "Max body", Type: 1, Visibility: "public", QualityLevel: "none"}
	root := domain.Comment{ID: 450, PostID: post.ID, UserID: bot.ID, Content: "Root AI comment.", AuditStatus: 1, IsPublic: true, AuditResult: jsonData(map[string]any{
		"source":   aiAutoCommentAuditSource,
		"taskType": AITaskPostAutoComment,
	}), CreatedAt: time.Now().Add(-4 * time.Minute)}
	triggerParent := root.ID
	firstUserReply := domain.Comment{ID: 451, PostID: post.ID, UserID: user.ID, ParentID: &triggerParent, Content: "First user reply.", AuditStatus: 1, IsPublic: true, CreatedAt: time.Now().Add(-3 * time.Minute)}
	firstAIParent := firstUserReply.ID
	firstAIReply := domain.Comment{ID: 452, PostID: post.ID, UserID: bot.ID, ParentID: &firstAIParent, Content: "Existing AI reply.", AuditStatus: 1, IsPublic: true, AuditResult: jsonData(map[string]any{
		"source":           aiCommentReplyAuditSource,
		"taskType":         AITaskCommentReply,
		"rootAICommentID":  root.ID,
		"triggerCommentID": firstUserReply.ID,
	}), CreatedAt: time.Now().Add(-2 * time.Minute)}
	secondParent := firstAIReply.ID
	secondUserReply := domain.Comment{ID: 453, PostID: post.ID, UserID: user.ID, ParentID: &secondParent, Content: "Second user reply.", AuditStatus: 1, IsPublic: true, CreatedAt: time.Now().Add(-time.Minute)}
	for _, row := range []any{&author, &bot, &user, &post, &root, &firstUserReply, &firstAIReply, &secondUserReply} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingCommentReply, AICommentReplyConfig{
		Enabled:                true,
		TemplateKey:            "comment_reply",
		DelaySeconds:           0,
		MaxImages:              0,
		ImageSelectionMode:     AIImageSelectionOrdered,
		Style:                  "normal",
		MaxRepliesPerAIComment: 1,
	})
	queue := &QueueService{
		db:       db,
		settings: settings,
		ai:       NewAIService(db, settings),
		cfg:      config.Config{API: config.APIConfig{BaseURL: "https://api.example.test"}},
	}
	if _, err := queue.createAICommentReply(context.Background(), aiCommentReplyTaskPayload{TriggerCommentID: secondUserReply.ID}); err == nil {
		t.Fatal("createAICommentReply() error is nil, want max-reply skip")
	}
	if got := atomic.LoadInt32(&upstreamRequests); got != 0 {
		t.Fatalf("upstream requests = %d, want 0", got)
	}
}

func TestQueueServiceAICommentMentionReplyUsesRandomBotAndPrompt(t *testing.T) {
	requestBody := make(chan map[string]any, 1)
	var upstreamRequests int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&upstreamRequests, 1)
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		requestBody <- body
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"这是图片里的封面细节。\"}}]}\n\n")
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer upstream.Close()

	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&domain.User{}, &domain.Post{}, &domain.Comment{}, &domain.Like{}, &domain.Notification{},
		&domain.PostImage{}, &domain.Tag{}, &domain.PostTag{}, &domain.AIGenerationLog{}, &domain.SystemSetting{},
	); err != nil {
		t.Fatal(err)
	}
	author := domain.User{ID: 510, UserID: "author-mention", Nickname: "Author", IsActive: true}
	inactiveBot := domain.User{ID: 520, UserID: "inactive-mention-bot", Nickname: "Inactive", IsActive: false}
	activeBot := domain.User{ID: 521, UserID: "active-mention-bot", Nickname: "Active Bot", IsActive: true}
	user := domain.User{ID: 530, UserID: "reader-mention", Nickname: "Reader", IsActive: true}
	post := domain.Post{ID: 540, UserID: author.ID, Title: "Mention title", Content: "Mention body with a cover object", Type: 1, Visibility: "public", QualityLevel: "none"}
	trigger := domain.Comment{ID: 550, PostID: post.ID, UserID: user.ID, Content: "@yueai这是什么", AuditStatus: 1, IsPublic: true, CreatedAt: time.Now().Add(-time.Minute)}
	previous := domain.Comment{ID: 551, PostID: post.ID, UserID: author.ID, Content: "Previous public context.", AuditStatus: 1, IsPublic: true, CreatedAt: time.Now().Add(-2 * time.Minute)}
	for _, row := range []any{&author, &inactiveBot, &activeBot, &user, &post, &previous, &trigger} {
		if err := db.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	imageDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(imageDir, "mention-cover.webp"), tinyPNGBytes(t), 0600); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.PostImage{PostID: post.ID, ImageURL: "/api/file/images/mention-cover.webp", IsFreePreview: true, SortOrder: 1}).Error; err != nil {
		t.Fatal(err)
	}

	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingBaseURL, upstream.URL+"/v1")
	settings.Set(context.Background(), AISettingCommentReply, AICommentReplyConfig{
		Enabled:                  false,
		TemplateKey:              "comment_reply",
		DelaySeconds:             0,
		MaxImages:                1,
		ImageSelectionMode:       AIImageSelectionOrdered,
		Style:                    "normal",
		MaxRepliesPerAIComment:   2,
		MentionEnabled:           true,
		MentionName:              "yueai",
		MentionTemplateKey:       "comment_mention_reply",
		MentionBotUserIDMin:      inactiveBot.ID,
		MentionBotUserIDMax:      activeBot.ID,
		MaxMentionRepliesPerPost: 2,
	})
	queue := &QueueService{
		db:       db,
		settings: settings,
		ai:       NewAIService(db, settings),
		cfg: config.Config{
			API:    config.APIConfig{BaseURL: "https://api.example.test"},
			Upload: config.UploadConfig{Image: config.UploadImageConfig{LocalUploadDir: imageDir}},
		},
	}

	replyID, err := queue.createAICommentReply(context.Background(), aiCommentReplyTaskPayload{TriggerCommentID: trigger.ID})
	if err != nil {
		t.Fatalf("createAICommentReply() mention error = %v", err)
	}
	var reply domain.Comment
	if err := db.Where("id = ?", replyID).First(&reply).Error; err != nil {
		t.Fatal(err)
	}
	if reply.UserID != activeBot.ID || reply.ParentID == nil || *reply.ParentID != trigger.ID || reply.Content != "这是图片里的封面细节。" {
		t.Fatalf("unexpected mention reply: %#v", reply)
	}
	if !commentHasAICommentMentionReplyMarker(reply.AuditResult) {
		t.Fatalf("reply audit_result missing mention marker: %s", string(reply.AuditResult))
	}
	meta := aiCommentAuditMap(reply.AuditResult)
	if meta["mentionName"] != "yueai" || meta["mentionQuery"] != "这是什么" || aiCommentMetaInt64(meta, "triggerCommentID") != trigger.ID {
		t.Fatalf("mention audit meta = %#v", meta)
	}
	body := <-requestBody
	for _, text := range []string{"Mention title", "Mention body with a cover object", "Previous public context.", "@yueai这是什么", "这是什么", "Images attached: 1"} {
		if !requestIncludesText(body, text) {
			t.Fatalf("request body missing %q: %#v", text, body)
		}
	}
	if !requestIncludesDataImage(body) {
		t.Fatalf("request body does not include inline image data: %#v", body)
	}
	var aiLog domain.AIGenerationLog
	if err := db.Where("task_type = ?", AITaskCommentMentionReply).First(&aiLog).Error; err != nil {
		t.Fatalf("missing mention ai generation log: %v", err)
	}
	if _, err := queue.createAICommentReply(context.Background(), aiCommentReplyTaskPayload{TriggerCommentID: trigger.ID}); err == nil {
		t.Fatal("duplicate mention createAICommentReply() error is nil, want skip")
	}
	secondTrigger := domain.Comment{ID: 560, PostID: post.ID, UserID: user.ID, Content: "@yueai 再看一次", AuditStatus: 1, IsPublic: true}
	if err := db.Create(&secondTrigger).Error; err != nil {
		t.Fatal(err)
	}
	settings.Set(context.Background(), AISettingCommentReply, AICommentReplyConfig{
		Enabled:                  false,
		TemplateKey:              "comment_reply",
		DelaySeconds:             0,
		MaxImages:                1,
		ImageSelectionMode:       AIImageSelectionOrdered,
		Style:                    "normal",
		MaxRepliesPerAIComment:   2,
		MentionEnabled:           true,
		MentionName:              "yueai",
		MentionTemplateKey:       "comment_mention_reply",
		MentionBotUserIDMin:      inactiveBot.ID,
		MentionBotUserIDMax:      activeBot.ID,
		MaxMentionRepliesPerPost: 1,
	})
	if _, err := queue.createAICommentReply(context.Background(), aiCommentReplyTaskPayload{TriggerCommentID: secondTrigger.ID}); err == nil {
		t.Fatal("second mention createAICommentReply() error is nil, want max-reply skip")
	}
	if got := atomic.LoadInt32(&upstreamRequests); got != 1 {
		t.Fatalf("upstream requests = %d, want 1", got)
	}
}

func requestIncludesImageURL(value any, want string) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if key == "url" && item == want {
				return true
			}
			if requestIncludesImageURL(item, want) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if requestIncludesImageURL(item, want) {
				return true
			}
		}
	}
	return false
}

func requestIncludesMessageRole(value any, want string) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if key == "role" && item == want {
				return true
			}
			if requestIncludesMessageRole(item, want) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if requestIncludesMessageRole(item, want) {
				return true
			}
		}
	}
	return false
}

func requestIncludesText(value any, want string) bool {
	switch typed := value.(type) {
	case string:
		return strings.Contains(typed, want)
	case map[string]any:
		for _, item := range typed {
			if requestIncludesText(item, want) {
				return true
			}
		}
	case []any:
		for _, item := range typed {
			if requestIncludesText(item, want) {
				return true
			}
		}
	}
	return false
}

func requestIncludesDataImage(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if key == "url" {
				if text, ok := item.(string); ok && strings.HasPrefix(text, "data:image/") {
					return true
				}
			}
			if requestIncludesDataImage(item) {
				return true
			}
		}
	case []any:
		if slices.ContainsFunc(typed, requestIncludesDataImage) {
			return true
		}
	}
	return false
}

func firstAIUpstreamAttempt(t *testing.T, db *gorm.DB) map[string]any {
	t.Helper()
	var log domain.AIGenerationLog
	if err := db.First(&log).Error; err != nil {
		t.Fatalf("missing generation log: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(log.Metadata, &meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	attempts, ok := meta["upstreamAttempts"].([]any)
	if !ok || len(attempts) == 0 {
		t.Fatalf("upstreamAttempts = %#v, want at least one attempt", meta["upstreamAttempts"])
	}
	return testMap(t, attempts[0])
}

func testMap(t *testing.T, value any) map[string]any {
	t.Helper()
	typed, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value = %#v, want map[string]any", value)
	}
	return typed
}

func tinyPNGBytes(t *testing.T) []byte {
	t.Helper()
	data, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestAISettingsDefaultsAndGroup(t *testing.T) {
	service := NewAIService(nil, nil)
	if got := service.Config().Concurrency; got != 5 {
		t.Fatalf("default concurrency = %d, want 5", got)
	}
	if got := DefaultSettings[AISettingConcurrency]; got != 5 {
		t.Fatalf("DefaultSettings concurrency = %#v, want 5", got)
	}
	if got := DefaultSettings[AISettingLogHTTPDetails]; got != false {
		t.Fatalf("DefaultSettings log HTTP details = %#v, want false", got)
	}
	commentReply, ok := DefaultSettings[AISettingCommentReply].(AICommentReplyConfig)
	if !ok {
		t.Fatalf("DefaultSettings comment reply type = %T, want AICommentReplyConfig", DefaultSettings[AISettingCommentReply])
	}
	if commentReply.Enabled || commentReply.TemplateKey != "comment_reply" || commentReply.MaxRepliesPerAIComment != 3 ||
		commentReply.MentionEnabled || commentReply.MentionName != "yueai" || commentReply.MentionTemplateKey != "comment_mention_reply" ||
		commentReply.MaxMentionRepliesPerPost != 3 {
		t.Fatalf("DefaultSettings comment reply = %#v, want disabled comment_reply and mention defaults", commentReply)
	}
	if got := settingGroup(AISettingConcurrency); got != "ai" {
		t.Fatalf("settingGroup(%q) = %q, want ai", AISettingConcurrency, got)
	}
}

func TestAICommentReplyReadyAllowsMentionOnlyConfig(t *testing.T) {
	redisServer := miniredis.RunT(t)
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.SystemSetting{}); err != nil {
		t.Fatal(err)
	}
	settings := NewSettingsService(db, nil)
	settings.Set(context.Background(), AISettingEnabled, true)
	settings.Set(context.Background(), AISettingAPIKey, "test-key")
	settings.Set(context.Background(), AISettingCommentReply, AICommentReplyConfig{
		Enabled:             false,
		MentionEnabled:      true,
		MentionName:         "yueai",
		MentionBotUserIDMin: 10,
		MentionBotUserIDMax: 20,
	})
	queue := NewQueueService(db, config.Config{
		Redis: config.RedisConfig{Addr: redisServer.Addr()},
		Queue: config.QueueConfig{Enabled: true},
	}, settings, NewAIService(db, settings))
	defer queue.Close()

	if !queue.AICommentReplyReady() {
		t.Fatal("AICommentReplyReady() = false, want true when mention replies are enabled")
	}
	settings.Set(context.Background(), AISettingCommentReply, AICommentReplyConfig{Enabled: false, MentionEnabled: false})
	if queue.AICommentReplyReady() {
		t.Fatal("AICommentReplyReady() = true, want false when both comment reply modes are disabled")
	}
}

func TestAIModerationLogStoresContentSnapshot(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.AIModerationLog{}, &domain.Audit{}); err != nil {
		t.Fatal(err)
	}
	queue := &QueueService{db: db}
	content := "评论内容\x00详情"
	payload := aiModerateContentTaskPayload{TargetType: AIModerationTargetComment, TargetID: 12, UserID: 34}
	cfg := AIModerationTargetConfig{
		Rules: map[string]AIModerationRuleConfig{
			"porn": {Enabled: true, Action: AIModerationActionDelete, Sensitivity: 0.65},
		},
	}
	decision := aiModerationDecision{
		Violation: true,
		Raw:       map[string]any{"rules": map[string]any{"porn": map[string]any{"confidence": 0.9}}},
		Rules: map[string]aiModerationRuleResult{
			"porn": {Violation: true, Matched: true, Confidence: 0.9, Threshold: moderationConfidenceThreshold(0.65), Sensitivity: 0.65, Action: AIModerationActionDelete, Sources: []string{aiModerationRuleSourceScore}},
		},
	}
	modelResult := jsonData(moderationModelResultMap(`{"rules":{"porn":{"confidence":0.9}}}`, decision, cfg, AIModerationActionDelete, "flagged"))
	if err := queue.writeModerationLog(context.Background(), payload, content, cfg, decision, "flagged", AIModerationActionDelete, modelResult, moderationCategoriesJSON(decision), "", ""); err != nil {
		t.Fatalf("writeModerationLog() error = %v", err)
	}
	var row domain.AIModerationLog
	if err := db.First(&row).Error; err != nil {
		t.Fatalf("missing moderation log: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(row.Metadata, &meta); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	if got := fmt.Sprint(meta["contentSnapshot"]); got != "评论内容详情" {
		t.Fatalf("contentSnapshot = %q, want sanitized content", got)
	}
	if got := fmt.Sprint(meta["triggerReason"]); !strings.Contains(got, "porn matched") || !strings.Contains(got, "confidence 0.90") {
		t.Fatalf("triggerReason = %q, want concrete matched rule and confidence", got)
	}
	var model map[string]any
	if err := json.Unmarshal(row.ModelResult, &model); err != nil {
		t.Fatalf("decode model result: %v", err)
	}
	if _, ok := model["rawJson"].(map[string]any); !ok {
		t.Fatalf("model rawJson = %#v, want parsed raw JSON object", model["rawJson"])
	}
	if _, ok := model["enabledRules"].(map[string]any); !ok {
		t.Fatalf("model enabledRules = %#v, want rule config snapshot", model["enabledRules"])
	}
	var audit domain.Audit
	if err := db.First(&audit).Error; err != nil {
		t.Fatalf("missing audit row: %v", err)
	}
	if strings.ContainsRune(audit.Content, 0) || audit.Content != "评论内容详情" {
		t.Fatalf("audit content = %q, want sanitized content", audit.Content)
	}
}

func TestApplyModerationDecisionDeletesCommentForDeleteAction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.Post{}, &domain.Comment{}, &domain.Like{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	post := domain.Post{ID: 70, UserID: 7, Title: "post", Type: 1, Visibility: "public", CommentCount: 1, CreatedAt: now}
	comment := domain.Comment{ID: 71, PostID: post.ID, UserID: 8, Content: "bad", IsPublic: true, AuditStatus: 1, CreatedAt: now}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&comment).Error; err != nil {
		t.Fatal(err)
	}
	redisServer := miniredis.RunT(t)
	cache := NewRedisStore(config.RedisConfig{Addr: redisServer.Addr(), CacheEnabled: true})
	queue := &QueueService{db: db, cache: cache}
	before := cache.CacheVersion(context.Background(), "comments")
	payload := aiModerateContentTaskPayload{TargetType: AIModerationTargetComment, TargetID: comment.ID, UserID: comment.UserID}
	decision := aiModerationDecision{Violation: true}
	if err := queue.applyModerationDecision(context.Background(), payload, decision, AIModerationActionDelete); err != nil {
		t.Fatalf("applyModerationDecision() error = %v", err)
	}
	var remaining int64
	if err := db.Model(&domain.Comment{}).Count(&remaining).Error; err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("remaining comments = %d, want 0", remaining)
	}
	var updated domain.Post
	if err := db.First(&updated, post.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.CommentCount != 0 {
		t.Fatalf("comment_count = %d, want 0", updated.CommentCount)
	}
	if after := cache.CacheVersion(context.Background(), "comments"); after <= before {
		t.Fatalf("comments cache version = %d, want > %d after AI delete", after, before)
	}
	if after := cache.CacheVersion(context.Background(), "posts"); after <= before {
		t.Fatalf("posts cache version = %d, want > %d after AI delete", after, before)
	}
}

func TestParseAIModerationDecisionAcceptsCommonProviderShapes(t *testing.T) {
	cfg := AIModerationTargetConfig{
		Rules: map[string]AIModerationRuleConfig{
			"spam":                {Enabled: true, Action: AIModerationActionObserve, Sensitivity: 0.65},
			"porn":                {Enabled: true, Action: AIModerationActionDelete, Sensitivity: 0.65},
			"political_sensitive": {Enabled: true, Action: AIModerationActionDelete, Sensitivity: 0.65},
		},
	}
	tests := []struct {
		name       string
		raw        string
		wantAction string
	}{
		{
			name:       "listed flagged category",
			raw:        `{"safe":false,"flaggedCategories":["porn"],"reason":"adult content"}`,
			wantAction: AIModerationActionDelete,
		},
		{
			name:       "nested category result",
			raw:        "```json\n{\"categories\":{\"political_sensitive\":{\"flagged\":true,\"score\":0.91,\"severity\":\"high\"}},\"recommended_action\":\"remove\"}\n```",
			wantAction: AIModerationActionDelete,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := parseAIModerationDecision(tt.raw, cfg, AIModerationTargetComment)
			if !decision.Violation {
				t.Fatalf("Violation = false, want true; decision=%+v", decision)
			}
			if action := moderationActionForDecision(decision, cfg, AIModerationTargetComment); action != tt.wantAction {
				t.Fatalf("action = %q, want %q; decision=%+v", action, tt.wantAction, decision)
			}
		})
	}
}

func TestParseAIModerationDecisionRequiresRuleThresholdForAction(t *testing.T) {
	cfg := AIModerationTargetConfig{
		Rules: map[string]AIModerationRuleConfig{
			"porn": {Enabled: true, Action: AIModerationActionDelete, Sensitivity: 0.2},
			"spam": {Enabled: true, Action: AIModerationActionObserve, Sensitivity: 0.65},
		},
	}
	raw := `{"violation":true,"action":"delete","reason":"borderline","rules":{"porn":{"violation":true,"confidence":0.4,"severity":"medium","reason":"suggestive but weak"}}}`
	decision := parseAIModerationDecision(raw, cfg, AIModerationTargetComment)
	if decision.Violation {
		t.Fatalf("Violation = true, want false when confidence is below sensitivity threshold; decision=%+v", decision)
	}
	if action := moderationActionForDecision(decision, cfg, AIModerationTargetComment); action != AIModerationActionObserve {
		t.Fatalf("action = %q, want observe for below-threshold rule", action)
	}
	if decision.Rules["porn"].Matched {
		t.Fatalf("porn rule matched below threshold: %+v", decision.Rules["porn"])
	}
	if len(decision.Raw) == 0 {
		t.Fatalf("raw decision JSON was not retained")
	}
}

func TestModerationActionUsesMatchedRuleConfigNotTopLevelAction(t *testing.T) {
	cfg := AIModerationTargetConfig{
		Rules: map[string]AIModerationRuleConfig{
			"spam": {Enabled: true, Action: AIModerationActionObserve, Sensitivity: 0.65},
		},
	}
	raw := `{"violation":true,"action":"delete","rules":{"spam":{"violation":true,"confidence":0.95,"severity":"high","reason":"repeated ad"}}}`
	decision := parseAIModerationDecision(raw, cfg, AIModerationTargetComment)
	if !decision.Violation || !decision.Rules["spam"].Matched {
		t.Fatalf("spam should match: %+v", decision)
	}
	if action := moderationActionForDecision(decision, cfg, AIModerationTargetComment); action != AIModerationActionObserve {
		t.Fatalf("action = %q, want configured spam observe action", action)
	}
}

func newAITestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.AIGenerationLog{}, &domain.SystemSetting{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func createAIJobForRunTest(t *testing.T, service *AIService, input string) domain.AIJob {
	t.Helper()
	job, err := service.CreateJob(context.Background(), AIJobCreateInput{Request: AIRequest{
		Type:        AITaskFormatMarkdown,
		Locale:      "en",
		Input:       input,
		TemplateKey: "markdown_format",
	}}, AIActor{Type: "user"})
	if err != nil {
		t.Fatalf("CreateJob() error = %v", err)
	}
	return job
}

func waitForAIJobState(t *testing.T, service *AIService, jobID string, match func(domain.AIJob) bool) domain.AIJob {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for {
		job, err := service.Job(context.Background(), jobID, AIActor{Type: "user"})
		if err != nil {
			t.Fatalf("Job() error = %v", err)
		}
		if match(job) {
			return job
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for job state; last status/stage = %s/%s", job.Status, job.Stage)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
