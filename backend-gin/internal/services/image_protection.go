package services

import (
	"archive/zip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/repositories"
)

const (
	imageProtectionStatusQueued     = "queued"
	imageProtectionStatusProcessing = "processing"
	imageProtectionStatusCompleted  = "completed"
	imageProtectionStatusFailed     = "failed"
	imageProtectionStatusExpired    = "expired"
)

func (s *QueueService) processImageProtectionPackage(ctx context.Context, task *asynq.Task) error {
	var payload imageProtectionTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return fmt.Errorf("%w: %v", asynq.SkipRetry, err)
	}
	if strings.TrimSpace(payload.JobID) == "" {
		return fmt.Errorf("%w: image protection job id is empty", asynq.SkipRetry)
	}
	if err := s.cleanupExpiredImageProtectionPackages(ctx); err != nil {
		s.recordQueueEvent(ctx, queueEvent{Queue: QueueImageProtection, Type: TaskImageProtection, Event: "cleanup_failed", At: time.Now().UnixMilli(), Error: err.Error(), Detail: map[string]any{"jobId": payload.JobID}})
	}
	var runErr error
	if payload.Kind == "post_archive" || task.Type() == TaskPostImageArchive {
		runErr = s.runPostImageArchive(ctx, payload.JobID, payload.ImageIDs)
	} else {
		runErr = s.runImageProtectionPackage(ctx, payload.JobID, payload.ImageIDs)
	}
	if runErr != nil {
		s.failImageProtectionJob(ctx, payload.JobID, runErr)
		return fmt.Errorf("%w: %v", asynq.SkipRetry, runErr)
	}
	result, _ := json.Marshal(map[string]any{"jobId": payload.JobID})
	_, _ = task.ResultWriter().Write(result)
	return nil
}

func (s *QueueService) runImageProtectionPackage(ctx context.Context, jobID string, requestedImageIDs []int64) error {
	if s == nil || s.db == nil {
		return errors.New("queue database is not configured")
	}
	now := time.Now()
	secondsPerImage := s.imageProtectionSecondsPerImage(ctx)
	var job domain.ImageProtectionJob
	if err := s.db.WithContext(ctx).Where("job_id = ?", jobID).First(&job).Error; err != nil {
		return err
	}
	if job.Status == imageProtectionStatusExpired {
		return nil
	}
	if job.Status == imageProtectionStatusCompleted && job.PackagePath != "" {
		if stat, statErr := os.Stat(job.PackagePath); statErr == nil && !stat.IsDir() {
			return nil
		}
	}
	if err := s.db.WithContext(ctx).Model(&domain.ImageProtectionJob{}).Where("job_id = ?", jobID).Updates(map[string]any{
		"status":                      imageProtectionStatusProcessing,
		"progress":                    5,
		"queue_position":              0,
		"estimated_wait_seconds":      0,
		"estimated_remaining_seconds": int(math.Ceil(float64(maxInt(1, job.ProtectedImageCount)) * secondsPerImage)),
		"processed_image_count":       0,
		"current_step":                "preparing",
		"heartbeat_at":                now,
		"started_at":                  now,
		"finished_at":                 nil,
		"error_message":               nil,
		"error_code":                  nil,
		"updated_at":                  now,
	}).Error; err != nil {
		return err
	}

	post, viewer, images, err := s.imageProtectionJobContext(ctx, job, requestedImageIDs)
	if err != nil {
		return err
	}
	if len(images) == 0 {
		return errors.New("no protected images are available for this viewer")
	}
	if len(images) != job.ProtectedImageCount {
		return errors.New("protected image selection changed")
	}

	secret := s.cfg.WebP.HiddenWatermark.Secret
	if secret == "" {
		secret = s.cfg.Upload.FileSigning.Secret
	}
	processor := NewImageProcessorWithRemote(
		s.settings,
		secret,
		s.cfg.Upload.Image.MaxSizeBytes,
		nil,
		HiddenWatermarkRemoteClientConfigFromConfig(s.cfg),
	)
	processor.RequireRemoteHiddenWatermark()
	processor.SetTraceResolver(func(resolveCtx context.Context, token string) bool {
		_, resolveErr := repositories.ResolveImageWatermarkTrace(resolveCtx, s.db, token)
		return resolveErr == nil
	})
	processor.SetPayloadResolver(func(resolveCtx context.Context, payload []byte) HiddenWatermarkData {
		trace, resolveErr := repositories.ResolveImageWatermarkTraceByShortCode(resolveCtx, s.db, fmt.Sprintf("%x", payload), len(payload))
		if resolveErr != nil {
			return HiddenWatermarkData{Found: false}
		}
		return HiddenWatermarkData{
			Found:         true,
			Valid:         true,
			Version:       trace.PayloadVersion,
			TraceToken:    trace.Token,
			TraceResolved: true,
			TraceType:     trace.TraceType,
			PayloadBytes:  len(payload),
			PayloadBits:   len(payload) * 8,
			PayloadFormat: "short_code_v1",
		}
	})
	processor.SetRecoverDimensionResolver(func(resolveCtx context.Context) [][2]int {
		dimensions, resolveErr := repositories.ListImageWatermarkRecoverDimensions(resolveCtx, s.db, 32)
		if resolveErr != nil {
			return nil
		}
		return dimensions
	})
	viewerUser := &RequestUser{ID: viewer.ID, UserID: viewer.UserID, Nickname: viewer.Nickname, Username: viewer.UserID}
	maxDimension := 2048
	allowedFailurePercent := 20
	if s.settings != nil {
		maxDimension = s.settings.Int("image_protection_max_dimension", maxDimension)
		allowedFailurePercent = min(max(s.settings.Int("image_protection_allowed_failure_percent", allowedFailurePercent), 0), 100)
	}
	allowedFailures := imageProtectionAllowedFailures(len(images), allowedFailurePercent)
	failedImages := make([]imageProtectionStageError, 0)
	successfulImages := 0
	packageDir := filepath.Join(serviceAbsPath(s.cfg.Upload.Temp.RootDir), "protected-packages", job.JobID)
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return err
	}
	keepWorkDir := false
	defer func() {
		if !keepWorkDir {
			_ = os.RemoveAll(packageDir)
		}
	}()
	packagePath := filepath.Join(packageDir, "package.zip")
	tmpPath := packagePath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	zipWriter := zip.NewWriter(file)
	for idx, image := range images {
		_ = s.reportImageProtectionProgress(ctx, jobID, "reading_source", "", idx, len(images), 5, now, secondsPerImage)
		data, contentType, filename, err := s.readProtectedImageSource(ctx, image.ImageURL)
		if err != nil {
			stageErr := imageProtectionStageError{
				stage: "reading_source", imageIndex: idx + 1, imageID: image.ID, err: err,
			}
			if s.imageProtectionSkipFailedImage(ctx, jobID, stageErr, &failedImages, allowedFailures, allowedFailurePercent, idx+1, len(images), now, secondsPerImage) {
				continue
			}
			_ = zipWriter.Close()
			_ = file.Close()
			_ = os.Remove(tmpPath)
			return imageProtectionFailureSummary(failedImages, len(images), allowedFailures, allowedFailurePercent)
		}
		archiveData := data
		entryName := filepath.ToSlash(fmt.Sprintf("%03d_%s", idx+1, imageProtectionFilename(filename)))
		if image.IsProtected {
			trace, err := s.imageProtectionWatermarkTrace(ctx, job, post, viewer, image, data)
			if err != nil {
				stageErr := imageProtectionStageError{
					stage: "preparing_watermark_trace", imageIndex: idx + 1, imageID: image.ID, err: err,
				}
				if s.imageProtectionSkipFailedImage(ctx, jobID, stageErr, &failedImages, allowedFailures, allowedFailurePercent, idx+1, len(images), now, secondsPerImage) {
					continue
				}
				_ = zipWriter.Close()
				_ = file.Close()
				_ = os.Remove(tmpPath)
				return imageProtectionFailureSummary(failedImages, len(images), allowedFailures, allowedFailurePercent)
			}
			watermarkInput := ProcessImageInput{
				Data:                  data,
				Filename:              fmt.Sprintf("%03d_%s", idx+1, filename),
				ContentType:           contentType,
				Purpose:               ImagePurposeContent,
				User:                  viewerUser,
				ForceWatermark:        true,
				WatermarkTraceToken:   trace.Token,
				WatermarkPayloadToken: traceShortCode(trace),
				MinDimension:          protectedWatermarkMinDimension,
				MaxDimension:          maxDimension,
				PreferRobust:          true,
				ForceRemoteAdaptive:   true,
				VerifyWithReference:   true,
				WatermarkProgress: func(value HiddenWatermarkProgress) {
					stage := value.Stage
					if stage == "decoding" || stage == "encoding" {
						stage = "embedding"
					}
					_ = s.reportImageProtectionProgress(ctx, jobID, stage, value.Profile, idx, len(images), value.Percent, now, secondsPerImage)
				},
			}
			processed, _, err := processProtectedImageWithRetry(ctx, processor, watermarkInput, trace.Token)
			if err != nil {
				stageErr := imageProtectionStageError{
					stage: "embedding", imageIndex: idx + 1, imageID: image.ID, err: err,
				}
				if s.imageProtectionSkipFailedImage(ctx, jobID, stageErr, &failedImages, allowedFailures, allowedFailurePercent, idx+1, len(images), now, secondsPerImage) {
					continue
				}
				_ = zipWriter.Close()
				_ = file.Close()
				_ = os.Remove(tmpPath)
				return imageProtectionFailureSummary(failedImages, len(images), allowedFailures, allowedFailurePercent)
			}
			now := time.Now()
			if err := s.db.WithContext(ctx).Model(&domain.ImageWatermarkTrace{}).Where("id = ?", trace.ID).Updates(map[string]any{
				"engine":           processed.WatermarkEngine,
				"watermark_width":  processed.Width,
				"watermark_height": processed.Height,
				"updated_at":       now,
			}).Error; err != nil {
				stageErr := imageProtectionStageError{
					stage: "updating_watermark_trace", imageIndex: idx + 1, imageID: image.ID, err: err,
				}
				if s.imageProtectionSkipFailedImage(ctx, jobID, stageErr, &failedImages, allowedFailures, allowedFailurePercent, idx+1, len(images), now, secondsPerImage) {
					continue
				}
				_ = zipWriter.Close()
				_ = file.Close()
				_ = os.Remove(tmpPath)
				return imageProtectionFailureSummary(failedImages, len(images), allowedFailures, allowedFailurePercent)
			}
			archiveData = processed.Data
			entryName = filepath.ToSlash(processed.Filename)
		}
		_ = s.reportImageProtectionProgress(ctx, jobID, "writing_archive", "", idx, len(images), 96, now, secondsPerImage)
		writer, err := zipWriter.Create(entryName)
		if err != nil {
			stageErr := imageProtectionStageError{
				stage: "writing_archive", imageIndex: idx + 1, imageID: image.ID, err: err,
			}
			if s.imageProtectionSkipFailedImage(ctx, jobID, stageErr, &failedImages, allowedFailures, allowedFailurePercent, idx+1, len(images), now, secondsPerImage) {
				continue
			}
			_ = zipWriter.Close()
			_ = file.Close()
			_ = os.Remove(tmpPath)
			return imageProtectionFailureSummary(failedImages, len(images), allowedFailures, allowedFailurePercent)
		}
		if _, err := writer.Write(archiveData); err != nil {
			stageErr := imageProtectionStageError{
				stage: "writing_archive", imageIndex: idx + 1, imageID: image.ID, err: err,
			}
			if s.imageProtectionSkipFailedImage(ctx, jobID, stageErr, &failedImages, allowedFailures, allowedFailurePercent, idx+1, len(images), now, secondsPerImage) {
				continue
			}
			_ = zipWriter.Close()
			_ = file.Close()
			_ = os.Remove(tmpPath)
			return imageProtectionFailureSummary(failedImages, len(images), allowedFailures, allowedFailurePercent)
		}
		successfulImages++
		completedProgress, completedProcessed := s.monotonicImageProtectionProgress(ctx, jobID, 10+((idx+1)*80/len(images)), idx+1)
		_ = s.updateImageProtectionJob(ctx, jobID, map[string]any{
			"processed_image_count":       completedProcessed,
			"progress":                    completedProgress,
			"current_step":                "image_completed",
			"estimated_remaining_seconds": s.imageProtectionRemainingSeconds(now, idx+1, len(images), 100, secondsPerImage),
			"heartbeat_at":                time.Now(),
			"updated_at":                  time.Now(),
		})
	}
	if successfulImages == 0 {
		_ = zipWriter.Close()
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return imageProtectionFailureSummary(failedImages, len(images), allowedFailures, allowedFailurePercent)
	}
	_ = s.reportImageProtectionProgress(ctx, jobID, "finalizing", "", len(images), len(images), 100, now, secondsPerImage)
	closeErr := zipWriter.Close()
	fileErr := file.Close()
	if closeErr != nil {
		_ = os.Remove(tmpPath)
		return closeErr
	}
	if fileErr != nil {
		_ = os.Remove(tmpPath)
		return fileErr
	}
	_ = os.Remove(packagePath)
	if err := os.Rename(tmpPath, packagePath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	finished := time.Now()
	retention := s.cfg.Upload.Temp.ProtectedPackageRetention
	if retention <= 0 {
		retention = 2 * time.Hour
	}
	expires := finished.Add(retention)
	completionUpdates := map[string]any{
		"status":                      imageProtectionStatusCompleted,
		"progress":                    100,
		"current_step":                "completed",
		"estimated_remaining_seconds": 0,
		"heartbeat_at":                finished,
		"package_path":                packagePath,
		"finished_at":                 finished,
		"expires_at":                  expires,
		"updated_at":                  finished,
		"retryable":                   false,
		"error_code":                  nil,
	}
	if len(failedImages) > 0 {
		message := imageProtectionPartialFailureMessage(failedImages, len(images), allowedFailurePercent)
		warningCode := "partial_images_skipped"
		completionUpdates["error_message"] = &message
		completionUpdates["error_code"] = &warningCode
	} else {
		completionUpdates["error_message"] = nil
	}
	if err := s.updateImageProtectionJob(ctx, jobID, completionUpdates); err != nil {
		return err
	}
	keepWorkDir = true
	targetID := post.ID
	title := "notification.imageProtectionReady.title"
	_ = s.db.WithContext(ctx).Create(&domain.Notification{UserID: job.UserID, SenderID: post.UserID, Type: 12, Title: title, TargetID: &targetID}).Error
	return nil
}
