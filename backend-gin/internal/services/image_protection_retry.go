package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/repositories"
)

func (s *QueueService) imageProtectionSkipFailedImage(
	ctx context.Context,
	jobID string,
	stageErr imageProtectionStageError,
	failedImages *[]imageProtectionStageError,
	allowedFailures, allowedPercent, processed, total int,
	started time.Time,
	secondsPerImage float64,
) bool {
	*failedImages = append(*failedImages, stageErr)
	if len(*failedImages) > allowedFailures {
		return false
	}
	message := imageProtectionPartialFailureMessage(*failedImages, total, allowedPercent)
	warningCode := "partial_images_skipped"
	progress, processed := s.monotonicImageProtectionProgress(ctx, jobID, 10+(processed*80/max(1, total)), processed)
	_ = s.updateImageProtectionJob(ctx, jobID, map[string]any{
		"processed_image_count":       processed,
		"progress":                    progress,
		"current_step":                "image_skipped",
		"estimated_remaining_seconds": s.imageProtectionRemainingSeconds(started, processed, total, 100, secondsPerImage),
		"error_message":               &message,
		"error_code":                  &warningCode,
		"heartbeat_at":                time.Now(),
		"updated_at":                  time.Now(),
	})
	return true
}

func imageProtectionAllowedFailures(total, percent int) int {
	if total <= 0 || percent <= 0 {
		return 0
	}
	if percent >= 100 {
		return max(0, total-1)
	}
	return max(0, total*percent/100)
}

func imageProtectionPartialFailureMessage(failedImages []imageProtectionStageError, total, allowedPercent int) string {
	if len(failedImages) == 0 {
		return ""
	}
	latest := failedImages[len(failedImages)-1]
	return fmt.Sprintf("%d/%d protected images skipped within %d%% allowed failure threshold; latest: %s", len(failedImages), max(1, total), allowedPercent, latest.Error())
}

func imageProtectionFailureSummary(failedImages []imageProtectionStageError, total, allowedFailures, allowedPercent int) error {
	if len(failedImages) == 0 {
		return errors.New("no protected images were processed")
	}
	latest := failedImages[len(failedImages)-1]
	return fmt.Errorf("protected image failures exceeded allowed threshold (%d/%d failed, allowed %d at %d%%): %w", len(failedImages), max(1, total), allowedFailures, allowedPercent, latest)
}

func processProtectedImageWithRetry(ctx context.Context, processor *ImageProcessor, input ProcessImageInput, traceToken string) (ProcessedImage, HiddenWatermarkData, error) {
	var lastErr error
	attemptCount := processor.hiddenWatermarkAttemptCountForInput(input)
	for attempt := range attemptCount {
		input.WatermarkAttempt = attempt
		processed, err := processor.Process(ctx, input)
		if err != nil {
			lastErr = err
			if isHiddenWatermarkRemoteServiceFailure(err) {
				return ProcessedImage{}, HiddenWatermarkData{}, err
			}
			continue
		}
		if !processed.WatermarkApplied {
			lastErr = fmt.Errorf("image watermark was not applied: %s", processed.WatermarkWarning)
			continue
		}
		extracted := HiddenWatermarkData{
			Found: true, Valid: true, Version: domain.ImageWatermarkPayloadVersion,
			TraceToken: traceToken, TraceResolved: true, PayloadBytes: domain.ImageWatermarkShortCodeBytes,
			PayloadBits: domain.ImageWatermarkShortCodeBytes * 8, PayloadFormat: "short_code_v1",
			WatermarkEngine: processed.WatermarkEngine,
		}
		if processed.WatermarkEngine == "remote" {
			reportHiddenWatermarkProgress(input.WatermarkProgress, HiddenWatermarkProgress{
				Stage: "transport_verification", Percent: 98,
			})
			if transportErr := processor.VerifyMessagingTransportWithReference(ctx, processed.Data, processed.WatermarkReference, traceToken, attempt, input.PreferRobust); transportErr != nil {
				lastErr = transportErr
				if isHiddenWatermarkRemoteServiceFailure(transportErr) {
					return ProcessedImage{}, HiddenWatermarkData{}, transportErr
				}
				continue
			}
		}
		return processed, extracted, nil
	}
	return ProcessedImage{}, HiddenWatermarkData{}, fmt.Errorf("image watermark verification failed after %d attempts: %w", attemptCount, lastErr)
}

func isHiddenWatermarkRemoteServiceFailure(err error) bool {
	return errors.Is(err, ErrHiddenWatermarkRemoteUnavailable) ||
		errors.Is(err, ErrHiddenWatermarkRemoteTimeout) ||
		errors.Is(err, ErrHiddenWatermarkRemoteRejected)
}

func (s *QueueService) reportImageProtectionProgress(
	ctx context.Context,
	jobID, stage, profile string,
	processed, total, stagePercent int,
	started time.Time,
	secondsPerImage float64,
) error {
	if total <= 0 {
		total = 1
	}
	processed = min(max(processed, 0), total)
	stagePercent = min(max(stagePercent, 0), 100)
	progress := 5 + ((processed*90)+(stagePercent*90/100))/total
	progress = min(max(progress, 5), 95)
	now := time.Now()
	progress, processed = s.monotonicImageProtectionProgress(ctx, jobID, progress, processed)
	updates := map[string]any{
		"progress":                    progress,
		"processed_image_count":       processed,
		"current_step":                stage,
		"estimated_remaining_seconds": s.imageProtectionRemainingSeconds(started, processed, total, stagePercent, secondsPerImage),
		"heartbeat_at":                now,
		"updated_at":                  now,
	}
	if profile != "" {
		updates["active_profile"] = profile
	}
	return s.updateImageProtectionJob(ctx, jobID, updates)
}

func (s *QueueService) monotonicImageProtectionProgress(ctx context.Context, jobID string, progress, processed int) (int, int) {
	if s == nil || s.db == nil {
		return progress, processed
	}
	var job domain.ImageProtectionJob
	if err := s.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Select("progress", "processed_image_count").
		First(&job).Error; err != nil {
		return progress, processed
	}
	return max(progress, job.Progress), max(processed, job.ProcessedImageCount)
}

func (s *QueueService) imageProtectionRemainingSeconds(started time.Time, processed, total, stagePercent int, secondsPerImage float64) int {
	if total <= 0 || processed >= total {
		return 0
	}
	if secondsPerImage <= 0 {
		secondsPerImage = 20
	}
	if processed > 0 {
		elapsed := time.Since(started).Seconds()
		if elapsed > 0 {
			secondsPerImage = math.Max(1, elapsed/float64(processed))
		}
	}
	fractionDone := float64(processed) + float64(min(max(stagePercent, 0), 100))/100
	remainingImages := math.Max(0, float64(total)-fractionDone)
	return int(math.Ceil(remainingImages * secondsPerImage))
}

func (s *QueueService) imageProtectionWatermarkTrace(ctx context.Context, job domain.ImageProtectionJob, post domain.Post, viewer domain.User, image domain.PostImage, source []byte) (domain.ImageWatermarkTrace, error) {
	var existing domain.ImageWatermarkTrace
	err := s.db.WithContext(ctx).
		Where("trace_type = ? AND job_id = ? AND image_id = ?", domain.ImageWatermarkTraceProtected, job.JobID, image.ID).
		First(&existing).Error
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return existing, err
	}
	sum := sha256.Sum256(source)
	trace, reserveErr := repositories.ReserveImageWatermarkTrace(ctx, s.db, repositories.ImageWatermarkTraceInput{
		TraceType:      domain.ImageWatermarkTraceProtected,
		FieldFlags:     domain.ImageWatermarkFieldUID | domain.ImageWatermarkFieldUserID | domain.ImageWatermarkFieldUsername | domain.ImageWatermarkFieldTime | domain.ImageWatermarkFieldSourceHash,
		ShortCodeBytes: domain.ImageWatermarkShortCodeBytes,
		UserID:         viewer.ID,
		UserDisplayID:  viewer.UserID,
		Username:       firstNonEmptyImageString(viewer.Nickname, viewer.UserID),
		PostID:         &post.ID,
		ImageID:        &image.ID,
		JobID:          job.JobID,
		SourceURL:      image.ImageURL,
		SourceHash:     hex.EncodeToString(sum[:]),
	})
	if reserveErr == nil {
		return trace, nil
	}
	if lookupErr := s.db.WithContext(ctx).
		Where("trace_type = ? AND job_id = ? AND image_id = ?", domain.ImageWatermarkTraceProtected, job.JobID, image.ID).
		First(&existing).Error; lookupErr == nil {
		return existing, nil
	}
	return domain.ImageWatermarkTrace{}, reserveErr
}

func traceShortCode(trace domain.ImageWatermarkTrace) string {
	if trace.ShortCode == nil {
		return ""
	}
	return strings.TrimSpace(*trace.ShortCode)
}
