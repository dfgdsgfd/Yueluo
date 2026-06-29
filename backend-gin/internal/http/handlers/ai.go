package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) FormatMarkdownStream(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.invalid_token", nil)
		return
	}
	req, ok := readAIRequest(c)
	if !ok {
		return
	}
	if strings.TrimSpace(req.Type) == "" {
		req.Type = services.AITaskFormatMarkdown
	}
	if req.TemplateKey == "" {
		req.TemplateKey = defaultUserAITemplateForTask(req.Type)
	}
	h.runAIStream(c, req, services.AIActor{
		Type:      "user",
		ID:        &user.ID,
		DisplayID: requestUserDisplayID(user),
	})
}

func (h NativeHandlers) CreateAIJob(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.invalid_token", nil)
		return
	}
	req, requestHash, ok := readAIJobRequest(c)
	if !ok {
		return
	}
	if strings.TrimSpace(req.Type) == "" {
		req.Type = services.AITaskFormatMarkdown
	}
	if req.TemplateKey == "" {
		req.TemplateKey = defaultUserAITemplateForTask(req.Type)
	}
	h.createAIJob(c, req, requestHash, services.AIActor{Type: "user", ID: &user.ID, DisplayID: requestUserDisplayID(user)})
}

func (h NativeHandlers) AdminCreateAIJob(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.invalid_token", nil)
		return
	}
	req, requestHash, ok := readAIJobRequest(c)
	if !ok {
		return
	}
	if strings.TrimSpace(req.Type) == "" {
		req.Type = "admin_copy"
	}
	h.createAIJob(c, req, requestHash, services.AIActor{Type: "admin", ID: &user.ID, DisplayID: requestUserDisplayID(user)})
}

func (h NativeHandlers) AIJobStatus(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.invalid_token", nil)
		return
	}
	h.writeAIJob(c, c.Param("jobId"), services.AIActor{Type: "user", ID: &user.ID, DisplayID: requestUserDisplayID(user)})
}

func (h NativeHandlers) AIJobStream(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.invalid_token", nil)
		return
	}
	h.streamAIJob(c, c.Param("jobId"), services.AIActor{Type: "user", ID: &user.ID, DisplayID: requestUserDisplayID(user)})
}

func (h NativeHandlers) AdminAIJobStatus(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.invalid_token", nil)
		return
	}
	h.writeAIJob(c, c.Param("jobId"), services.AIActor{Type: "admin", ID: &user.ID, DisplayID: requestUserDisplayID(user)})
}

func (h NativeHandlers) AdminAIJobStream(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.invalid_token", nil)
		return
	}
	h.streamAIJob(c, c.Param("jobId"), services.AIActor{Type: "admin", ID: &user.ID, DisplayID: requestUserDisplayID(user)})
}

func (h NativeHandlers) AIActiveJob(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.invalid_token", nil)
		return
	}
	requestHash := strings.TrimSpace(c.Query("requestHash"))
	if requestHash == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_request_body", nil)
		return
	}
	job, err := h.AI.ActiveJob(c.Request.Context(), requestHash, services.AIActor{Type: "user", ID: &user.ID, DisplayID: requestUserDisplayID(user)})
	if err != nil {
		writeAIHTTPError(c, err)
		return
	}
	writeSuccess(c, matrixMsgOK, h.aiJobResponse(c.Request.Context(), job))
}

func (h NativeHandlers) CancelAIJob(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.invalid_token", nil)
		return
	}
	job, err := h.AI.CancelJob(c.Request.Context(), c.Param("jobId"), services.AIActor{Type: "user", ID: &user.ID, DisplayID: requestUserDisplayID(user)})
	if err != nil {
		writeAIHTTPError(c, err)
		return
	}
	data := h.aiJobResponse(c.Request.Context(), job)
	if h.Queue != nil {
		data["queueAbandon"] = h.Queue.AbandonAIJobRun(c.Request.Context(), job.JobID)
	}
	writeSuccess(c, matrixMsgOK, data)
}

func defaultUserAITemplateForTask(taskType string) string {
	switch strings.TrimSpace(taskType) {
	case services.AITaskPostPolish:
		return "post_polish"
	case services.AITaskPostCustomGenerate:
		return "post_custom_generate"
	default:
		return "markdown_format"
	}
}

func (h NativeHandlers) AdminAIGenerateStream(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.invalid_token", nil)
		return
	}
	req, ok := readAIRequest(c)
	if !ok {
		return
	}
	if strings.TrimSpace(req.Type) == "" {
		req.Type = "admin_copy"
	}
	h.runAIStream(c, req, services.AIActor{
		Type:      "admin",
		ID:        &user.ID,
		DisplayID: requestUserDisplayID(user),
	})
}

func (h NativeHandlers) AdminAISettings(c *gin.Context) {
	if h.AI == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.ai_settings_unavailable", nil)
		return
	}
	switch c.Request.Method {
	case http.MethodGet:
		writeSuccess(c, matrixMsgOK, h.AI.PublicSettings())
	case http.MethodPut:
		body := readBodyMap(c)
		if err := h.AI.UpdateSettings(c.Request.Context(), body); err != nil {
			writeAIHTTPError(c, err)
			return
		}
		writeSuccess(c, matrixMsgOK, h.AI.PublicSettings())
	default:
		response.JSON(c, http.StatusMethodNotAllowed, response.CodeError, "error.method_not_allowed", nil)
	}
}

func (h NativeHandlers) AdminAILogs(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	page, limit, offset := pageLimit(c, 20)
	taskType := strings.TrimSpace(c.Query("type"))
	status := strings.TrimSpace(c.Query("status"))
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.AIGenerationLog{})
	if taskType != "" {
		query = query.Where("task_type = ?", taskType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	var rows []domain.AIGenerationLog
	if err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, aiLogResponse(row))
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"items":      items,
		"pagination": matrixPagination(page, limit, total),
	})
}

func (h NativeHandlers) AIPublishGenerationSettings(c *gin.Context) {
	if h.AI == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.ai_settings_unavailable", nil)
		return
	}
	writeSuccess(c, matrixMsgOK, h.AI.PublicPublishGenerationConfig())
}

func (h NativeHandlers) GeneratePublishContent(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.invalid_token", nil)
		return
	}
	if h.AI == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.ai_settings_unavailable", nil)
		return
	}
	var body struct {
		Locale     string                  `json:"locale"`
		Title      string                  `json:"title"`
		Detail     string                  `json:"detail"`
		Body       string                  `json:"body"`
		Tags       []string                `json:"tags"`
		NeedTitle  bool                    `json:"needTitle"`
		NeedDetail bool                    `json:"needDetail"`
		Images     []services.AIImageInput `json:"images"`
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_request_body", nil)
		return
	}
	detail := body.Detail
	if strings.TrimSpace(detail) == "" {
		detail = body.Body
	}
	if !body.NeedTitle && !body.NeedDetail {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.ai_input_required", nil)
		return
	}
	if strings.TrimSpace(body.Title) == "" && strings.TrimSpace(detail) == "" && len(body.Images) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.ai_input_required", nil)
		return
	}
	h.touchAIAnalysisUploadAssets(c, user.ID, body.Images)
	result, err := h.AI.GeneratePublishContent(c.Request.Context(), services.AIPublishGenerationInput{
		Locale:     body.Locale,
		Title:      body.Title,
		Detail:     detail,
		Tags:       body.Tags,
		NeedTitle:  body.NeedTitle,
		NeedDetail: body.NeedDetail,
		Images:     h.aiReachableImages(c, body.Images),
	}, services.AIActor{Type: "user", ID: &user.ID, DisplayID: requestUserDisplayID(user)})
	if err != nil {
		writeAIHTTPError(c, err)
		return
	}
	writeSuccess(c, matrixMsgOK, result)
}

func (h NativeHandlers) AdminAIModerationDebug(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.invalid_token", nil)
		return
	}
	if h.AI == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.ai_settings_unavailable", nil)
		return
	}
	var body struct {
		TargetType   string                             `json:"targetType"`
		Content      string                             `json:"content"`
		TemplateKey  string                             `json:"templateKey"`
		SystemPrompt string                             `json:"systemPrompt"`
		UserPrompt   string                             `json:"userPrompt"`
		Prompt       string                             `json:"prompt"`
		Config       *services.AIModerationTargetConfig `json:"config"`
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_request_body", nil)
		return
	}
	result, err := h.AI.DebugModeration(c.Request.Context(), services.AIModerationDebugInput{
		TargetType:   body.TargetType,
		Content:      body.Content,
		TemplateKey:  body.TemplateKey,
		SystemPrompt: body.SystemPrompt,
		UserPrompt:   body.UserPrompt,
		Prompt:       body.Prompt,
		Config:       body.Config,
	}, services.AIActor{Type: "admin", ID: &user.ID, DisplayID: requestUserDisplayID(user)})
	if err != nil {
		writeAIHTTPError(c, err)
		return
	}
	writeSuccess(c, matrixMsgOK, result)
}

func readAIRequest(c *gin.Context) (services.AIRequest, bool) {
	var req services.AIRequest
	decoder := json.NewDecoder(c.Request.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&req); err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_request_body", nil)
		return req, false
	}
	if strings.TrimSpace(req.Input) == "" && len(req.Images) == 0 && !hasCustomGeneratePrompt(req) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.ai_input_required", nil)
		return req, false
	}
	return req, true
}

func readAIJobRequest(c *gin.Context) (services.AIRequest, string, bool) {
	var body struct {
		services.AIRequest
		RequestHash string `json:"requestHash"`
	}
	decoder := json.NewDecoder(c.Request.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_request_body", nil)
		return services.AIRequest{}, "", false
	}
	req := body.AIRequest
	if strings.TrimSpace(req.Input) == "" && len(req.Images) == 0 && !hasCustomGeneratePrompt(req) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.ai_input_required", nil)
		return services.AIRequest{}, "", false
	}
	return req, strings.TrimSpace(body.RequestHash), true
}

func hasCustomGeneratePrompt(req services.AIRequest) bool {
	if strings.TrimSpace(req.Type) != services.AITaskPostCustomGenerate {
		return false
	}
	value, ok := req.Variables["customPrompt"]
	if !ok {
		return false
	}
	return strings.TrimSpace(fmt.Sprint(value)) != ""
}

func (h NativeHandlers) runAIStream(c *gin.Context, req services.AIRequest, actor services.AIActor) {
	if h.AI == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.ai_settings_unavailable", nil)
		return
	}
	req.Images = h.aiReachableImages(c, req.Images)
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-store")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	emit := func(event services.AIStreamEvent) error {
		if !shouldEmitAIStreamEventToActor(event, actor) {
			return nil
		}
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event.Type, data); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}
	if err := h.AI.RunStream(c.Request.Context(), req, actor, emit); err != nil {
		if c.Request.Context().Err() != nil {
			return
		}
		_ = emit(services.AIStreamEvent{
			Type:    "error",
			Code:    aiErrorCode(err),
			Message: aiErrorCode(err),
			Detail:  aiErrorDetail(err),
		})
	}
}

func (h NativeHandlers) createAIJob(c *gin.Context, req services.AIRequest, requestHash string, actor services.AIActor) {
	if h.AI == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.ai_settings_unavailable", nil)
		return
	}
	req.Images = h.aiReachableImages(c, req.Images)
	h.touchAIJobUploadAssets(c, actor, req)
	job, err := h.AI.CreateJob(c.Request.Context(), services.AIJobCreateInput{Request: req, RequestHash: requestHash}, actor)
	if err != nil {
		writeAIHTTPError(c, err)
		return
	}
	var queueJob map[string]any
	if job.Status == services.AIJobStatusQueued && h.Queue != nil {
		queueJob, err = h.Queue.EnqueueAIJobRun(c.Request.Context(), job.JobID)
		if err != nil {
			job, _ = h.AI.CancelJob(c.Request.Context(), job.JobID, actor)
			response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.queue_unavailable", gin.H{"detail": err.Error(), "job": h.aiJobResponse(c.Request.Context(), job)})
			return
		}
	}
	data := h.aiJobResponse(c.Request.Context(), job)
	if queueJob != nil {
		if existingQueueJob, ok := data["queueJob"].(map[string]any); ok {
			data["queueJob"] = mergeAIQueueJobPayload(existingQueueJob, queueJob)
		} else {
			data["queueJob"] = queueJob
		}
	}
	writeSuccess(c, matrixMsgOK, data)
}

func (h NativeHandlers) aiReachableImages(c *gin.Context, images []services.AIImageInput) []services.AIImageInput {
	if len(images) == 0 {
		return nil
	}
	out := make([]services.AIImageInput, 0, len(images))
	for _, image := range images {
		next := services.AIImageInput{
			URL:     h.aiReachableImageURL(c, image.URL),
			DataURL: strings.TrimSpace(image.DataURL),
			Mime:    strings.TrimSpace(image.Mime),
			Alt:     strings.TrimSpace(image.Alt),
		}
		if strings.TrimSpace(next.URL) == "" && strings.TrimSpace(next.DataURL) == "" {
			continue
		}
		out = append(out, next)
	}
	return out
}

func (h NativeHandlers) aiReachableImageURL(c *gin.Context, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	signed := strings.TrimSpace(h.signFileURL(raw))
	if signed == "" {
		return signed
	}
	if absoluteHTTPURL(signed) {
		if localHTTPBaseURL(signed) {
			if base := h.aiPublicBaseURL(c); base != "" && !localHTTPBaseURL(base) {
				if parsed, err := url.Parse(signed); err == nil {
					return strings.TrimRight(base, "/") + parsed.RequestURI()
				}
			}
		}
		return signed
	}
	if strings.HasPrefix(signed, "/") {
		if base := h.aiPublicBaseURL(c); base != "" {
			return strings.TrimRight(base, "/") + signed
		}
	}
	return signed
}

func (h NativeHandlers) aiPublicBaseURL(c *gin.Context) string {
	for _, candidate := range []string{
		h.Config.Upload.LocalBase,
		h.Config.API.BaseURL,
		h.Config.Frontend.BaseURL,
	} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" && absoluteHTTPURL(candidate) && !localHTTPBaseURL(candidate) {
			return strings.TrimRight(candidate, "/")
		}
	}
	if base := requestPublicBaseURL(c); base != "" {
		return base
	}
	for _, candidate := range []string{
		h.Config.Upload.LocalBase,
		h.Config.API.BaseURL,
		h.Config.Frontend.BaseURL,
	} {
		candidate = strings.TrimSpace(candidate)
		if candidate != "" && absoluteHTTPURL(candidate) {
			return strings.TrimRight(candidate, "/")
		}
	}
	return ""
}

func requestPublicBaseURL(c *gin.Context) string {
	if c == nil || c.Request == nil {
		return ""
	}
	host := strings.TrimSpace(firstNonEmpty(c.GetHeader("X-Forwarded-Host"), c.Request.Host))
	if host == "" {
		return ""
	}
	proto := strings.TrimSpace(firstNonEmpty(c.GetHeader("X-Forwarded-Proto"), ""))
	if proto == "" {
		if c.Request.TLS != nil {
			proto = "https"
		} else {
			proto = "http"
		}
	}
	proto = strings.ToLower(strings.Split(proto, ",")[0])
	host = strings.Split(host, ",")[0]
	if proto != "http" && proto != "https" {
		return ""
	}
	base := proto + "://" + strings.TrimSpace(host)
	if !absoluteHTTPURL(base) || localHTTPBaseURL(base) {
		return ""
	}
	return strings.TrimRight(base, "/")
}

func absoluteHTTPURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func localHTTPBaseURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func (h NativeHandlers) writeAIJob(c *gin.Context, jobID string, actor services.AIActor) {
	if h.AI == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.ai_settings_unavailable", nil)
		return
	}
	job, err := h.AI.Job(c.Request.Context(), jobID, actor)
	if err != nil {
		writeAIHTTPError(c, err)
		return
	}
	writeSuccess(c, matrixMsgOK, h.aiJobResponse(c.Request.Context(), job))
}

func (h NativeHandlers) streamAIJob(c *gin.Context, jobID string, actor services.AIActor) {
	if h.AI == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.ai_settings_unavailable", nil)
		return
	}
	if _, err := h.AI.Job(c.Request.Context(), jobID, actor); err != nil {
		writeAIHTTPError(c, err)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache, no-store")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	emit := func(event string, data any) error {
		raw, err := json.Marshal(data)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, raw); err != nil {
			return err
		}
		c.Writer.Flush()
		return nil
	}
	if err := emit("connected", gin.H{"jobId": jobID}); err != nil {
		return
	}
	events, unsubscribe := h.AI.SubscribeJobEvents(c.Request.Context(), jobID)
	defer unsubscribe()
	ticker := time.NewTicker(200 * time.Millisecond)
	heartbeat := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	defer heartbeat.Stop()
	lastSignature := ""
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			if !shouldEmitAIStreamEventToActor(event, actor) {
				continue
			}
			if err := emit(event.Type, event); err != nil {
				return
			}
		case <-heartbeat.C:
			if err := emit("heartbeat", gin.H{"jobId": jobID, "at": time.Now().UnixMilli()}); err != nil {
				return
			}
		case <-ticker.C:
			job, err := h.AI.Job(c.Request.Context(), jobID, actor)
			if err != nil {
				_ = emit("error", gin.H{"code": aiErrorCode(err), "message": aiErrorCode(err), "detail": aiErrorDetail(err)})
				return
			}
			responseJob := h.aiJobResponse(c.Request.Context(), job)
			signature := aiJobStreamSignature(job, responseJob)
			if signature != lastSignature {
				lastSignature = signature
				if err := emit("job", gin.H{"job": responseJob}); err != nil {
					return
				}
			}
			if job.Status == services.AIJobStatusCompleted || job.Status == services.AIJobStatusFailed || job.Status == services.AIJobStatusCanceled {
				return
			}
		}
	}
}

func writeAIHTTPError(c *gin.Context, err error) {
	code := aiErrorCode(err)
	status := http.StatusBadRequest
	switch code {
	case "error.ai_disabled", "error.ai_api_key_missing", "error.ai_settings_unavailable":
		status = http.StatusServiceUnavailable
	case "error.ai_timeout":
		status = http.StatusGatewayTimeout
	case "error.ai_upstream_unavailable", "error.ai_upstream_error":
		status = http.StatusBadGateway
	}
	response.JSON(c, status, response.CodeError, code, nil)
}

func shouldEmitAIStreamEventToActor(event services.AIStreamEvent, actor services.AIActor) bool {
	if event.Type == "" {
		return false
	}
	return event.Type != "upstream_event" || actor.Type == "admin"
}

func aiJobStreamSignature(job domain.AIJob, responseJob gin.H) string {
	updatedAt := ""
	if job.UpdatedAt != nil {
		updatedAt = job.UpdatedAt.Format(time.RFC3339Nano)
	}
	queueSignature := ""
	if queueJob, ok := responseJob["queueJob"]; ok {
		if data, err := json.Marshal(queueJob); err == nil {
			queueSignature = string(data)
		}
	}
	return fmt.Sprintf("%s|%s|%s|%d|%d|%d|%d|%d|%d|%s|%d|%d|%s|%s",
		job.JobID,
		job.Status,
		job.Stage,
		job.Percent,
		job.CurrentChunk,
		job.TotalChunks,
		job.ProcessedChars,
		len(job.Output),
		len(job.Reasoning),
		job.ErrorCode,
		job.UpstreamStatus,
		len(job.UpstreamDetail),
		updatedAt,
		queueSignature,
	)
}

func aiErrorCode(err error) string {
	var aiErr services.AIError
	if errors.As(err, &aiErr) && aiErr.Code != "" {
		return aiErr.Code
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "error.not_found"
	}
	return "error.ai_request_failed"
}

func aiErrorDetail(err error) string {
	var aiErr services.AIError
	if errors.As(err, &aiErr) {
		if aiErr.UpstreamDetail != "" {
			return aiErr.UpstreamDetail
		}
	}
	return ""
}

func requestUserDisplayID(user *services.RequestUser) *string {
	if user == nil {
		return nil
	}
	display := strings.TrimSpace(user.UserID)
	if user.XiseID != nil && strings.TrimSpace(*user.XiseID) != "" {
		display = strings.TrimSpace(*user.XiseID)
	}
	if display == "" {
		display = strconv.FormatInt(user.ID, 10)
	}
	return &display
}

func aiLogResponse(row domain.AIGenerationLog) gin.H {
	return gin.H{
		"id":               row.ID,
		"jobId":            row.JobID,
		"type":             row.TaskType,
		"templateKey":      row.TemplateKey,
		"actorType":        row.ActorType,
		"actorId":          row.ActorID,
		"actorDisplayId":   row.ActorDisplayID,
		"inputSummary":     row.InputSummary,
		"outputSummary":    row.OutputSummary,
		"status":           row.Status,
		"model":            row.Model,
		"baseUrl":          row.BaseURL,
		"promptTokens":     row.PromptTokens,
		"completionTokens": row.CompletionTokens,
		"totalTokens":      row.TotalTokens,
		"estimatedCost":    row.EstimatedCost,
		"errorCode":        row.ErrorCode,
		"errorMessage":     row.ErrorMessage,
		"durationMs":       row.DurationMS,
		"tokensPerSecond":  row.TokensPerSecond,
		"metadata":         jsonValue([]byte(row.Metadata)),
		"createdAt":        row.CreatedAt,
		"updatedAt":        row.UpdatedAt,
	}
}
