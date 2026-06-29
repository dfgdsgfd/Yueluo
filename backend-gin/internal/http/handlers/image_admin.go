package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"

	"yuem-go/backend-gin/internal/domain"
	appmiddleware "yuem-go/backend-gin/internal/http/middleware"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

var imageBooleanSettings = map[string]struct{}{
	"image_webp_enabled":                 {},
	"image_webp_lossless":                {},
	"image_libvips_enabled":              {},
	"image_archive_enabled":              {},
	"image_protection_enabled":           {},
	"image_protection_notice_enabled":    {},
	"image_select_all_enabled":           {},
	"paid_content_balance_enabled":       {},
	"paid_content_points_enabled":        {},
	"hidden_watermark_enabled":           {},
	"hidden_watermark_protected_only":    {},
	"hidden_watermark_include_uid":       {},
	"hidden_watermark_include_user_id":   {},
	"hidden_watermark_include_username":  {},
	"hidden_watermark_include_time":      {},
	"hidden_watermark_include_file_hash": {},
	"hidden_watermark_include_custom":    {},
	"hidden_watermark_extract_all_users": {},
}

var imageNumberSettingRanges = map[string][2]int{
	"image_webp_quality":                                {1, 100},
	"image_avatar_webp_quality":                         {1, 100},
	"image_webp_method":                                 {0, 6},
	"image_webp_alpha_quality":                          {0, 100},
	"image_max_width":                                   {0, 16_384},
	"image_max_height":                                  {0, 16_384},
	"image_processing_concurrency":                      {1, 8},
	"image_post_max_count":                              {1, 500},
	"image_archive_threshold":                           {1, 500},
	"paid_content_points_max_price":                     {1, 1_000_000},
	"paid_content_balance_max_price":                    {1, 1_000_000},
	"image_protection_max_dimension":                    {0, 16_384},
	"image_protection_allowed_failure_percent":          {0, 100},
	"image_protection_webp_quality":                     {1, 100},
	"hidden_watermark_block_width":                      {4, 64},
	"hidden_watermark_block_height":                     {4, 64},
	"hidden_watermark_d1":                               {1, 64},
	"hidden_watermark_d2":                               {0, 64},
	"hidden_watermark_golay_seed":                       {0, 2_147_483_647},
	"hidden_watermark_remote_password_wm":               {0, 2_147_483_647},
	"hidden_watermark_remote_password_img":              {0, 2_147_483_647},
	"hidden_watermark_remote_custom_d1":                 {1, 64},
	"hidden_watermark_remote_custom_d2":                 {0, 64},
	"hidden_watermark_remote_timeout_seconds":           {10, 300},
	"hidden_watermark_remote_operation_timeout_seconds": {10, 300},
}

func normalizeImageSetting(key string, value any) (any, bool) {
	if _, ok := imageBooleanSettings[key]; ok {
		parsed, valid := boolFromAny(value)
		return parsed, valid
	}
	if bounds, ok := imageNumberSettingRanges[key]; ok {
		parsed, valid := intFromAny(value)
		if !valid || parsed < bounds[0] || parsed > bounds[1] {
			return nil, false
		}
		if key == "hidden_watermark_block_width" || key == "hidden_watermark_block_height" {
			if parsed%2 != 0 {
				parsed++
			}
			if parsed > bounds[1] {
				return nil, false
			}
		}
		return parsed, true
	}
	if key == "hidden_watermark_custom_text" {
		text := strings.TrimSpace(toString(value))
		return text, utf8.ValidString(text) && utf8.RuneCountInString(text) <= 255
	}
	if key == "hidden_watermark_profile" {
		profile := strings.TrimSpace(toString(value))
		switch profile {
		case "current", "author_recommended", "fidelity", "robust", "custom":
			return profile, true
		default:
			return nil, false
		}
	}
	if key == "hidden_watermark_remote_profile" {
		profile := strings.TrimSpace(toString(value))
		switch profile {
		case "adaptive", "fidelity", "balanced", "strong", "official", "custom":
			return profile, true
		default:
			return nil, false
		}
	}
	if key == "hidden_watermark_remote_engine" {
		engine := strings.TrimSpace(toString(value))
		switch engine {
		case "auto", "blind_watermark", "dwt_dct_svd":
			return engine, true
		default:
			return nil, false
		}
	}
	if key == "image_protection_output_mode" {
		mode := strings.TrimSpace(toString(value))
		switch mode {
		case "lossless_webp", "quality_webp":
			return mode, true
		default:
			return nil, false
		}
	}
	if key == "hidden_watermark_engine" {
		engine := strings.TrimSpace(toString(value))
		switch engine {
		case "auto", "local", "remote":
			return engine, true
		default:
			return nil, false
		}
	}
	if key == "hidden_watermark_coefficient_mode" {
		mode := strings.TrimSpace(toString(value))
		switch mode {
		case "d1", "d1d2":
			return mode, true
		default:
			return nil, false
		}
	}
	if key == "hidden_watermark_ecc_mode" {
		mode := strings.TrimSpace(toString(value))
		switch mode {
		case "golay", "none":
			return mode, true
		default:
			return nil, false
		}
	}
	if key == "hidden_watermark_extract_user_ids" || key == "hidden_watermark_extract_usernames" {
		return normalizeWatermarkAccessList(value), true
	}
	if strings.HasPrefix(key, "image_") || strings.HasPrefix(key, "hidden_watermark_") {
		return nil, false
	}
	return value, true
}

func (h NativeHandlers) adminExtractImageWatermark(c *gin.Context) {
	if acceptsWatermarkProgressStream(c) {
		h.extractImageWatermarkStream(c)
		return
	}
	h.extractImageWatermark(c)
}

func (h NativeHandlers) UserExtractImageWatermark(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, matrixMsgInvalidToken, nil)
		return
	}
	if !h.userCanExtractImageWatermark(user) {
		h.recordWatermarkInspection(c, watermarkUploadMeta{}, nil, "forbidden", http.StatusForbidden)
		response.JSON(c, http.StatusForbidden, response.CodeForbidden, "error.hidden_watermark_extract_forbidden", nil)
		return
	}
	if acceptsWatermarkProgressStream(c) {
		h.extractImageWatermarkStream(c)
		return
	}
	h.extractImageWatermark(c)
}

func (h NativeHandlers) userCanExtractImageWatermark(user *services.RequestUser) bool {
	if user == nil || user.Type == "admin" {
		return user != nil
	}
	if h.Settings == nil {
		return false
	}
	if h.Settings.Bool("hidden_watermark_extract_all_users") {
		return true
	}
	userIDs := h.Settings.StringArray("hidden_watermark_extract_user_ids")
	if containsFold(userIDs, strconv.FormatInt(user.ID, 10)) || containsFold(userIDs, user.UserID) {
		return true
	}
	if user.XiseID != nil && containsFold(userIDs, *user.XiseID) {
		return true
	}
	usernames := h.Settings.StringArray("hidden_watermark_extract_usernames")
	return containsFold(usernames, user.UserID) || containsFold(usernames, user.Nickname) || containsFold(usernames, user.Username)
}

func (h NativeHandlers) extractImageWatermark(c *gin.Context) {
	data, referenceData, upload, cleanup, err := h.readWatermarkUpload(c)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		reason := "image_invalid"
		message := "error.image_invalid"
		if errors.Is(err, errWatermarkFileRequired) {
			reason = "file_required"
			message = "error.image_file_required"
		} else if errors.Is(err, errWatermarkFileTooLarge) {
			reason = "file_too_large"
			message = "error.image_file_too_large"
		}
		h.recordWatermarkInspection(c, upload, nil, reason, http.StatusBadRequest)
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, message, nil)
		return
	}
	processor := h.imageProcessor()
	result, err := processor.ExtractWithReference(c.Request.Context(), data, referenceData)
	if err != nil {
		h.recordWatermarkInspection(c, upload, nil, "extract_failed", http.StatusBadRequest)
		status := http.StatusBadRequest
		message := err.Error()
		if errors.Is(err, services.ErrHiddenWatermarkRemoteTimeout) {
			status = http.StatusGatewayTimeout
			message = services.ErrHiddenWatermarkRemoteTimeout.Error()
		}
		response.JSON(c, status, response.CodeValidationError, message, nil)
		return
	}
	result = h.enrichImageWatermarkResult(c.Request.Context(), result)
	h.recordWatermarkInspection(c, upload, &result, "", http.StatusOK)
	writeSuccess(c, matrixMsgOK, result)
}

type watermarkExtractionStreamEvent struct {
	Type      string                        `json:"type"`
	Stage     string                        `json:"stage,omitempty"`
	Percent   int                           `json:"percent,omitempty"`
	Completed int                           `json:"completed,omitempty"`
	Total     int                           `json:"total,omitempty"`
	ElapsedMS int64                         `json:"elapsedMs"`
	Heartbeat bool                          `json:"heartbeat,omitempty"`
	Source    string                        `json:"source,omitempty"`
	Result    *services.HiddenWatermarkData `json:"result,omitempty"`
	Error     string                        `json:"error,omitempty"`
	Retryable bool                          `json:"retryable,omitempty"`
	Status    int                           `json:"status,omitempty"`
}

type watermarkExtractionResult struct {
	result services.HiddenWatermarkData
	err    error
}

func (h NativeHandlers) extractImageWatermarkStream(c *gin.Context) {
	data, referenceData, upload, cleanup, err := h.readWatermarkUpload(c)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		h.writeWatermarkUploadError(c, upload, err)
		return
	}

	timeout := time.Duration(h.Settings.Int("hidden_watermark_remote_timeout_seconds", 50)+5) * time.Second
	if timeout < 15*time.Second {
		timeout = 55 * time.Second
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()
	startedAt := time.Now()
	progressCh := make(chan services.HiddenWatermarkProgress, 64)
	resultCh := make(chan watermarkExtractionResult, 1)
	processor := h.imageProcessor()
	go func() {
		result, extractErr := processor.ExtractWithReferenceProgress(ctx, data, referenceData, func(progress services.HiddenWatermarkProgress) {
			select {
			case progressCh <- progress:
			default:
			}
		})
		resultCh <- watermarkExtractionResult{result: result, err: extractErr}
	}()

	c.Header("Content-Type", "application/x-ndjson; charset=utf-8")
	c.Header("Cache-Control", "no-cache, no-store")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	latest := services.HiddenWatermarkProgress{Stage: "queued", Percent: 1}
	lastProgressAt := startedAt
	h.writeWatermarkStreamEvent(c, watermarkExtractionStreamEvent{
		Type: "progress", Stage: latest.Stage, Percent: latest.Percent, ElapsedMS: 0,
	})
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case progress := <-progressCh:
			if progress.Percent < latest.Percent {
				progress.Percent = latest.Percent
			}
			latest = progress
			lastProgressAt = time.Now()
			h.writeWatermarkStreamEvent(c, watermarkExtractionStreamEvent{
				Type:      "progress",
				Stage:     progress.Stage,
				Percent:   progress.Percent,
				Completed: progress.Completed,
				Total:     progress.Total,
				ElapsedMS: time.Since(startedAt).Milliseconds(),
				Heartbeat: progress.Heartbeat,
				Source:    progress.Source,
			})
		case outcome := <-resultCh:
			if outcome.err != nil {
				status, message, retryable := watermarkExtractionError(outcome.err)
				h.recordWatermarkInspection(c, upload, nil, "extract_failed", status)
				h.writeWatermarkStreamEvent(c, watermarkExtractionStreamEvent{
					Type:      "error",
					Stage:     latest.Stage,
					Percent:   latest.Percent,
					ElapsedMS: time.Since(startedAt).Milliseconds(),
					Error:     message,
					Retryable: retryable,
					Status:    status,
				})
				return
			}
			result := h.enrichImageWatermarkResult(ctx, outcome.result)
			h.recordWatermarkInspection(c, upload, &result, "", http.StatusOK)
			h.writeWatermarkStreamEvent(c, watermarkExtractionStreamEvent{
				Type: "progress", Stage: "complete", Percent: 100, ElapsedMS: time.Since(startedAt).Milliseconds(),
			})
			h.writeWatermarkStreamEvent(c, watermarkExtractionStreamEvent{
				Type: "result", Stage: "complete", Percent: 100, ElapsedMS: time.Since(startedAt).Milliseconds(), Result: &result,
			})
			return
		case <-ticker.C:
			if time.Since(lastProgressAt) < 3*time.Second {
				continue
			}
			h.writeWatermarkStreamEvent(c, watermarkExtractionStreamEvent{
				Type:      "heartbeat",
				Stage:     latest.Stage,
				Percent:   latest.Percent,
				Completed: latest.Completed,
				Total:     latest.Total,
				ElapsedMS: time.Since(startedAt).Milliseconds(),
				Heartbeat: true,
				Source:    "gateway",
			})
		case <-ctx.Done():
			if errors.Is(ctx.Err(), context.Canceled) && c.Request.Context().Err() != nil {
				return
			}
			h.writeWatermarkStreamEvent(c, watermarkExtractionStreamEvent{
				Type:      "error",
				Stage:     latest.Stage,
				Percent:   latest.Percent,
				ElapsedMS: time.Since(startedAt).Milliseconds(),
				Error:     services.ErrHiddenWatermarkRemoteTimeout.Error(),
				Retryable: true,
				Status:    http.StatusGatewayTimeout,
			})
			return
		}
	}
}

func (h NativeHandlers) writeWatermarkUploadError(c *gin.Context, upload watermarkUploadMeta, err error) {
	reason := "image_invalid"
	message := "error.image_invalid"
	if errors.Is(err, errWatermarkFileRequired) {
		reason = "file_required"
		message = "error.image_file_required"
	} else if errors.Is(err, errWatermarkFileTooLarge) {
		reason = "file_too_large"
		message = "error.image_file_too_large"
	}
	h.recordWatermarkInspection(c, upload, nil, reason, http.StatusBadRequest)
	response.JSON(c, http.StatusBadRequest, response.CodeValidationError, message, nil)
}

func watermarkExtractionError(err error) (status int, message string, retryable bool) {
	if errors.Is(err, services.ErrHiddenWatermarkRemoteTimeout) || errors.Is(err, context.DeadlineExceeded) {
		return http.StatusGatewayTimeout, services.ErrHiddenWatermarkRemoteTimeout.Error(), true
	}
	return http.StatusBadRequest, err.Error(), false
}

func acceptsWatermarkProgressStream(c *gin.Context) bool {
	return c != nil && strings.Contains(strings.ToLower(c.GetHeader("Accept")), "application/x-ndjson")
}

func (h NativeHandlers) writeWatermarkStreamEvent(c *gin.Context, event watermarkExtractionStreamEvent) {
	if c == nil || c.Writer == nil {
		return
	}
	_ = json.NewEncoder(c.Writer).Encode(event)
	if flusher, ok := c.Writer.(http.Flusher); ok {
		flusher.Flush()
	}
}

var (
	errWatermarkFileRequired = errors.New("watermark image file is required")
	errWatermarkFileTooLarge = errors.New("watermark image file is too large")
)

type watermarkUploadMeta struct {
	Filename    string
	ContentType string
	Size        int64
}

func (h NativeHandlers) readWatermarkUpload(c *gin.Context) ([]byte, []byte, watermarkUploadMeta, func(), error) {
	meta := watermarkUploadMeta{}
	if c == nil || c.Request == nil {
		return nil, nil, meta, nil, errWatermarkFileRequired
	}
	reader, err := c.Request.MultipartReader()
	if err != nil {
		return nil, nil, meta, nil, errWatermarkFileRequired
	}
	workDir := ""
	if h.TempStorage != nil {
		workDir, err = h.TempStorage.NewWorkDir("watermark-inspector")
	} else {
		workDir, err = os.MkdirTemp("", "yuem-watermark-inspector-")
	}
	if err != nil {
		return nil, nil, meta, nil, err
	}
	cleanup := func() {
		_ = os.RemoveAll(workDir)
	}
	maxBytes := h.Config.Upload.Image.MaxSizeBytes
	if maxBytes <= 0 {
		maxBytes = 20 << 20
	}
	var data []byte
	var referenceData []byte
	for {
		part, nextErr := reader.NextPart()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nil, nil, meta, cleanup, nextErr
		}
		formName := part.FormName()
		if formName != "file" && formName != "reference_file" {
			_ = part.Close()
			continue
		}
		if formName == "file" {
			meta.Filename = filepath.Base(strings.TrimSpace(part.FileName()))
			meta.ContentType = strings.TrimSpace(part.Header.Get("Content-Type"))
		}
		path := filepath.Join(workDir, formName)
		out, createErr := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if createErr != nil {
			_ = part.Close()
			return nil, nil, meta, cleanup, createErr
		}
		written, copyErr := io.Copy(out, io.LimitReader(part, maxBytes+1))
		closeErr := errors.Join(out.Close(), part.Close())
		if formName == "file" {
			meta.Size = written
		}
		if copyErr != nil || closeErr != nil {
			return nil, nil, meta, cleanup, errors.Join(copyErr, closeErr)
		}
		if written > maxBytes {
			return nil, nil, meta, cleanup, errWatermarkFileTooLarge
		}
		uploadedData, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, nil, meta, cleanup, readErr
		}
		if formName == "reference_file" {
			referenceData = uploadedData
		} else {
			data = uploadedData
		}
	}
	if len(data) == 0 {
		return nil, nil, meta, cleanup, errWatermarkFileRequired
	}
	return data, referenceData, meta, cleanup, nil
}

func (h NativeHandlers) recordWatermarkInspection(c *gin.Context, upload watermarkUploadMeta, result *services.HiddenWatermarkData, reasonCode string, status int) {
	if h.DB == nil || c == nil || c.Request == nil {
		return
	}
	actorID, actorDisplayID, actorType := requestUserActor(c)
	metadata := map[string]any{
		"filename":     upload.Filename,
		"content_type": upload.ContentType,
		"file_size":    upload.Size,
	}
	outcome := "success"
	if reasonCode != "" {
		outcome = "failure"
	}
	if result != nil {
		metadata["found"] = result.Found
		metadata["valid"] = result.Valid
		metadata["uid"] = result.UID
		metadata["user_id"] = result.UserID
		metadata["username"] = result.Username
		metadata["source_hash"] = result.SourceHash
		metadata["post_id"] = result.PostID
		metadata["job_id"] = result.JobID
		metadata["trace_token"] = result.TraceToken
		metadata["trace_type"] = result.TraceType
		metadata["trace_resolved"] = result.TraceResolved
		metadata["payload_bytes"] = result.PayloadBytes
		metadata["payload_format"] = result.PayloadFormat
		metadata["watermark_engine"] = result.WatermarkEngine
	}
	raw, _ := json.Marshal(metadata)
	var actorDisplay *string
	if actorDisplayID != "" {
		actorDisplay = &actorDisplayID
	}
	row := domain.SecurityAuditLog{
		Category:        "hidden_watermark",
		Action:          "extract",
		Outcome:         outcome,
		ActorID:         actorID,
		ActorType:       firstNonEmptyHandler(actorType, "unknown"),
		ActorDisplayID:  actorDisplay,
		IP:              h.clientIP(c),
		UserAgent:       c.Request.UserAgent(),
		BrowserLanguage: firstAcceptLanguage(c.GetHeader("Accept-Language")),
		Method:          c.Request.Method,
		Path:            c.Request.URL.Path,
		Status:          status,
		ReasonCode:      reasonCode,
		RequestID:       c.Writer.Header().Get(appmiddleware.RequestIDHeader),
		Metadata:        datatypes.JSON(raw),
		CreatedAt:       time.Now(),
	}
	_ = h.DB.WithContext(c.Request.Context()).Create(&row).Error
}

func normalizeWatermarkAccessList(value any) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, item := range parseStringSlice(value) {
		for _, part := range strings.FieldsFunc(item, func(r rune) bool {
			return r == ',' || r == '，' || r == '\n' || r == '\r' || r == '\t'
		}) {
			text := strings.TrimSpace(part)
			if text == "" {
				continue
			}
			key := strings.ToLower(text)
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, text)
		}
	}
	return out
}

func containsFold(values []string, needle string) bool {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return false
	}
	for _, value := range values {
		if strings.EqualFold(strings.TrimSpace(value), needle) {
			return true
		}
	}
	return false
}
