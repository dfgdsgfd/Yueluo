package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func (s *QueueService) imageProtectionJobContext(ctx context.Context, job domain.ImageProtectionJob, requestedImageIDs []int64) (domain.Post, domain.User, []domain.PostImage, error) {
	var post domain.Post
	if err := s.db.WithContext(ctx).Where("id = ?", job.PostID).Select("id", "user_id", "title").First(&post).Error; err != nil {
		return post, domain.User{}, nil, err
	}
	var viewer domain.User
	if err := s.db.WithContext(ctx).Where("id = ?", job.UserID).Select("id", "user_id", "nickname").First(&viewer).Error; err != nil {
		return post, viewer, nil, err
	}
	var images []domain.PostImage
	if err := s.db.WithContext(ctx).Where("post_id = ?", job.PostID).Order("sort_order ASC, id ASC").Find(&images).Error; err != nil {
		return post, viewer, nil, err
	}
	var payment domain.PostPaymentSetting
	paid := false
	if err := s.db.WithContext(ctx).Where("post_id = ?", job.PostID).First(&payment).Error; err == nil && payment.Enabled {
		paid = true
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return post, viewer, nil, err
	}
	canViewPaid := !paid || post.UserID == job.UserID
	if paid && !canViewPaid {
		var count int64
		if err := s.db.WithContext(ctx).Model(&domain.UserPurchasedContent{}).Where("user_id = ? AND post_id = ?", job.UserID, job.PostID).Count(&count).Error; err != nil {
			return post, viewer, nil, err
		}
		canViewPaid = count > 0
	}
	eligible, err := selectProtectedPackageImages(images, canViewPaid, requestedImageIDs)
	if err != nil {
		return post, viewer, nil, err
	}
	return post, viewer, eligible, nil
}

func selectProtectedPackageImages(images []domain.PostImage, canViewPaid bool, requestedImageIDs []int64) ([]domain.PostImage, error) {
	images = append([]domain.PostImage(nil), images...)
	if len(images) > 0 {
		images[0].IsFreePreview = true
		images[0].IsProtected = false
	}
	requested := make(map[int64]struct{}, len(requestedImageIDs))
	for _, imageID := range requestedImageIDs {
		requested[imageID] = struct{}{}
	}
	eligible := make([]domain.PostImage, 0, len(images))
	for _, image := range images {
		if len(requested) > 0 {
			if _, ok := requested[image.ID]; !ok {
				continue
			}
		} else if !image.IsProtected {
			continue
		}
		if canViewPaid || image.IsFreePreview {
			eligible = append(eligible, image)
		}
	}
	if len(requested) > 0 && len(eligible) != len(requested) {
		return nil, errors.New("protected image selection changed")
	}
	return eligible, nil
}

func (s *QueueService) readProtectedImageSource(ctx context.Context, raw string) ([]byte, string, string, error) {
	source := strings.TrimSpace(raw)
	if source == "" {
		return nil, "", "", errors.New("empty image url")
	}
	if localBase := strings.TrimSpace(s.cfg.Upload.LocalBase); localBase != "" && strings.HasPrefix(source, localBase) {
		if parsed, err := url.Parse(source); err == nil {
			source = parsed.Path
		}
	}
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return nil, "", "", err
		}
		resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
		if err != nil {
			return nil, "", "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, "", "", fmt.Errorf("fetch image failed: %d", resp.StatusCode)
		}
		data, err := io.ReadAll(io.LimitReader(resp.Body, maxDecodedImagePixels))
		if err != nil {
			return nil, "", "", err
		}
		filename := filepath.Base(resp.Request.URL.Path)
		return data, imageProtectionContentType(data, resp.Header.Get("Content-Type")), imageProtectionFilename(filename), nil
	}
	pathOnly := source
	if parsed, err := url.Parse(source); err == nil && parsed.Path != "" {
		pathOnly = parsed.Path
	}
	pathOnly = strings.TrimPrefix(pathOnly, "/")
	parts := strings.SplitN(pathOnly, "/", 4)
	if len(parts) >= 4 && parts[0] == "api" && parts[1] == "file" {
		fileType := parts[2]
		name, _ := url.PathUnescape(parts[3])
		for _, dir := range queueConfiguredUploadTypeDirs(s.cfg, fileType) {
			candidate := filepath.Join(dir, filepath.FromSlash(name))
			data, err := os.ReadFile(candidate)
			if err == nil {
				return data, imageProtectionContentType(data, ""), imageProtectionFilename(filepath.Base(candidate)), nil
			}
		}
	}
	candidate := source
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(serviceAbsPath("."), filepath.FromSlash(candidate))
	}
	data, err := os.ReadFile(candidate)
	if err != nil {
		return nil, "", "", err
	}
	return data, imageProtectionContentType(data, ""), imageProtectionFilename(filepath.Base(candidate)), nil
}

func (s *QueueService) updateImageProtectionJob(ctx context.Context, jobID string, updates map[string]any) error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.WithContext(ctx).Model(&domain.ImageProtectionJob{}).Where("job_id = ?", jobID).Updates(updates).Error
}

func (s *QueueService) failImageProtectionJob(ctx context.Context, jobID string, err error) {
	if err == nil {
		return
	}
	message := err.Error()
	errorCode := imageProtectionErrorCode(err)
	now := time.Now()
	_ = s.updateImageProtectionJob(ctx, jobID, map[string]any{
		"status":        imageProtectionStatusFailed,
		"current_step":  "failed",
		"error_message": &message,
		"error_code":    &errorCode,
		"retryable":     true,
		"finished_at":   now,
		"updated_at":    now,
	})
}

func imageProtectionErrorCode(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrHiddenWatermarkRemoteUnavailable) {
		return "watermark_server_unavailable"
	}
	if errors.Is(err, ErrHiddenWatermarkRemoteTimeout) {
		return "watermark_server_timeout"
	}
	if errors.Is(err, ErrHiddenWatermarkRemoteRejected) {
		return "watermark_server_rejected"
	}
	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "remote service is not configured"):
		return "watermark_server_unavailable"
	case strings.Contains(message, "fetch image"), strings.Contains(message, "empty image url"):
		return "image_source_unavailable"
	case strings.Contains(message, "no protected images"):
		return "no_eligible_images"
	case strings.Contains(message, "selection changed"):
		return "selection_changed"
	case strings.Contains(message, "watermark"):
		return "watermark_verification_failed"
	default:
		return "package_generation_failed"
	}
}
