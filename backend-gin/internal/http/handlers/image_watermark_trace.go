package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) imageProcessor() *services.ImageProcessor {
	processor := h.Images
	if processor == nil {
		secret := h.Config.WebP.HiddenWatermark.Secret
		if secret == "" {
			secret = h.Config.Upload.FileSigning.Secret
		}
		processor = services.NewImageProcessorWithRemote(
			h.Settings,
			secret,
			h.Config.Upload.Image.MaxSizeBytes,
			nil,
			services.HiddenWatermarkRemoteClientConfigFromConfig(h.Config),
		)
	}
	if h.DB != nil {
		processor.SetTraceResolver(func(ctx context.Context, token string) bool {
			_, err := repositories.ResolveImageWatermarkTrace(ctx, h.DB, token)
			return err == nil
		})
		processor.SetPayloadResolver(func(ctx context.Context, payload []byte) services.HiddenWatermarkData {
			trace, err := repositories.ResolveImageWatermarkTraceByShortCode(ctx, h.DB, fmt.Sprintf("%x", payload), len(payload))
			if err != nil {
				return services.HiddenWatermarkData{Found: false}
			}
			return imageWatermarkDataFromTrace(trace, "short_code_v1")
		})
		processor.SetRecoverDimensionResolver(func(ctx context.Context) [][2]int {
			dimensions, err := repositories.ListImageWatermarkRecoverDimensions(ctx, h.DB, 32)
			if err != nil {
				return nil
			}
			return dimensions
		})
	}
	return processor
}

func (h NativeHandlers) enrichImageWatermarkResult(ctx context.Context, result services.HiddenWatermarkData) services.HiddenWatermarkData {
	if result.TraceToken == "" {
		return result
	}
	trace, err := repositories.ResolveImageWatermarkTrace(ctx, h.DB, result.TraceToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			result.Found = false
			result.Valid = false
			result.TraceResolved = false
		}
		return result
	}
	payloadFormat := firstNonEmptyHandler(result.PayloadFormat, "short_code_v1")
	enriched := imageWatermarkDataFromTrace(trace, payloadFormat)
	enriched.WatermarkEngine = firstNonEmptyHandler(trace.Engine, result.WatermarkEngine)
	enriched.PayloadBytes = result.PayloadBytes
	enriched.PayloadBits = result.PayloadBits
	result = enriched
	result.WatermarkEngine = firstNonEmptyHandler(trace.Engine, result.WatermarkEngine)
	return appendImageWatermarkIncludedFields(result)
}

func imageWatermarkTraceShortCode(trace domain.ImageWatermarkTrace) string {
	if trace.ShortCode == nil {
		return ""
	}
	return strings.TrimSpace(*trace.ShortCode)
}

func imageWatermarkDataFromTrace(trace domain.ImageWatermarkTrace, payloadFormat string) services.HiddenWatermarkData {
	result := services.HiddenWatermarkData{
		Found:           true,
		Valid:           true,
		TraceResolved:   true,
		TraceToken:      trace.Token,
		TraceType:       trace.TraceType,
		Version:         trace.PayloadVersion,
		PayloadBytes:    trace.PayloadBytes,
		PayloadBits:     trace.PayloadBytes * 8,
		PayloadFormat:   payloadFormat,
		WatermarkEngine: trace.Engine,
	}
	if trace.FieldFlags&domain.ImageWatermarkFieldUID != 0 {
		result.UID = trace.UserID
	}
	if trace.FieldFlags&domain.ImageWatermarkFieldUserID != 0 {
		result.UserID = trace.UserDisplayID
	}
	if trace.FieldFlags&domain.ImageWatermarkFieldUsername != 0 {
		result.Username = trace.Username
	}
	if trace.FieldFlags&domain.ImageWatermarkFieldTime != 0 {
		result.UploadedAt = trace.CreatedAt
	}
	if trace.FieldFlags&domain.ImageWatermarkFieldSourceHash != 0 {
		result.SourceHash = trace.SourceHash
	}
	if trace.FieldFlags&domain.ImageWatermarkFieldCustom != 0 {
		result.CustomText = trace.CustomText
	}
	if trace.PostID != nil {
		result.PostID = *trace.PostID
	}
	if trace.ImageID != nil {
		result.ImageID = *trace.ImageID
	}
	result.JobID = trace.JobID
	return appendImageWatermarkIncludedFields(result)
}

func appendImageWatermarkIncludedFields(result services.HiddenWatermarkData) services.HiddenWatermarkData {
	result.IncludedFields = []string{"traceToken", "payloadBytes", "traceType"}
	if result.UID > 0 {
		result.IncludedFields = append(result.IncludedFields, "uid")
	}
	if result.UserID != "" {
		result.IncludedFields = append(result.IncludedFields, "userId")
	}
	if result.Username != "" {
		result.IncludedFields = append(result.IncludedFields, "username")
	}
	if !result.UploadedAt.IsZero() {
		result.IncludedFields = append(result.IncludedFields, "uploadedAt")
	}
	if result.SourceHash != "" {
		result.IncludedFields = append(result.IncludedFields, "sourceHash")
	}
	if result.CustomText != "" {
		result.IncludedFields = append(result.IncludedFields, "customText")
	}
	if result.PostID > 0 {
		result.IncludedFields = append(result.IncludedFields, "postId")
	}
	if result.ImageID > 0 {
		result.IncludedFields = append(result.IncludedFields, "imageId")
	}
	if result.JobID != "" {
		result.IncludedFields = append(result.IncludedFields, "jobId")
	}
	return result
}
