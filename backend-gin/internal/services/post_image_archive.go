package services

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"yuem-go/backend-gin/internal/domain"
)

func (s *QueueService) runPostImageArchive(ctx context.Context, jobID string, requestedImageIDs []int64) error {
	if s == nil || s.db == nil {
		return errors.New("queue database is not configured")
	}
	var job domain.ImageProtectionJob
	if err := s.db.WithContext(ctx).Where("job_id = ? AND package_kind = ?", jobID, "post_archive").First(&job).Error; err != nil {
		return err
	}
	if job.Status == imageProtectionStatusExpired {
		return nil
	}
	if job.Status == imageProtectionStatusCompleted && job.PackagePath != "" {
		if stat, err := os.Stat(job.PackagePath); err == nil && !stat.IsDir() {
			return nil
		}
	}

	var post domain.Post
	if err := s.db.WithContext(ctx).Select("id", "user_id", "is_draft").Where("id = ?", job.PostID).First(&post).Error; err != nil {
		return err
	}
	if post.IsDraft {
		return errors.New("draft posts cannot generate image archives")
	}
	var images []domain.PostImage
	if err := s.db.WithContext(ctx).Where("post_id = ?", job.PostID).Order("sort_order ASC, id ASC").Find(&images).Error; err != nil {
		return err
	}
	images, err := selectPostArchiveImages(images, requestedImageIDs)
	if err != nil {
		return err
	}
	if len(images) == 0 || len(images) != job.ProtectedImageCount {
		return errors.New("post image archive selection changed")
	}
	if signature := postArchiveImageSignature(images); signature != job.SourceSignature {
		return errors.New("post image archive source changed")
	}

	started := time.Now()
	if err := s.updateImageProtectionJob(ctx, jobID, map[string]any{
		"status":                      imageProtectionStatusProcessing,
		"progress":                    5,
		"queue_position":              0,
		"estimated_wait_seconds":      0,
		"estimated_remaining_seconds": max(1, len(images)),
		"processed_image_count":       0,
		"current_step":                "preparing",
		"heartbeat_at":                started,
		"started_at":                  started,
		"finished_at":                 nil,
		"error_message":               nil,
		"error_code":                  nil,
		"updated_at":                  started,
	}); err != nil {
		return err
	}

	packageDir := filepath.Join(serviceAbsPath(s.cfg.Upload.RootDir), "archives", job.JobID)
	if err := os.RemoveAll(packageDir); err != nil {
		return err
	}
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		return err
	}
	keepWorkDir := false
	defer func() {
		if !keepWorkDir {
			_ = os.RemoveAll(packageDir)
		}
	}()

	packagePath := filepath.Join(packageDir, "images.zip")
	tmpPath := packagePath + ".tmp"
	file, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	zipWriter := zip.NewWriter(file)
	closeWithError := func(runErr error) error {
		_ = zipWriter.Close()
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return runErr
	}
	for idx, image := range images {
		now := time.Now()
		progress := 5 + idx*90/max(1, len(images))
		if err := s.updateImageProtectionJob(ctx, jobID, map[string]any{
			"progress":                    progress,
			"processed_image_count":       idx,
			"current_step":                "reading_source",
			"estimated_remaining_seconds": max(0, len(images)-idx),
			"heartbeat_at":                now,
			"updated_at":                  now,
		}); err != nil {
			return closeWithError(err)
		}
		data, _, filename, err := s.readProtectedImageSource(ctx, image.ImageURL)
		if err != nil {
			return closeWithError(fmt.Errorf("read image %d: %w", image.ID, err))
		}
		entryName := filepath.ToSlash(fmt.Sprintf("%03d_%s", idx+1, imageProtectionFilename(filename)))
		writer, err := zipWriter.Create(entryName)
		if err != nil {
			return closeWithError(err)
		}
		if _, err := writer.Write(data); err != nil {
			return closeWithError(err)
		}
		now = time.Now()
		if err := s.updateImageProtectionJob(ctx, jobID, map[string]any{
			"progress":                    5 + (idx+1)*90/max(1, len(images)),
			"processed_image_count":       idx + 1,
			"current_step":                "writing_archive",
			"estimated_remaining_seconds": max(0, len(images)-idx-1),
			"heartbeat_at":                now,
			"updated_at":                  now,
		}); err != nil {
			return closeWithError(err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	current, err := s.postImageArchiveJobStillCurrent(ctx, jobID, job.SourceSignature)
	if err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if !current {
		_ = os.Remove(tmpPath)
		return nil
	}
	if err := os.Rename(tmpPath, packagePath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	finished := time.Now()
	result := s.db.WithContext(ctx).Model(&domain.ImageProtectionJob{}).
		Where("job_id = ? AND status <> ? AND source_signature = ?", jobID, imageProtectionStatusExpired, job.SourceSignature).
		Updates(map[string]any{
			"status":                      imageProtectionStatusCompleted,
			"progress":                    100,
			"processed_image_count":       len(images),
			"current_step":                "completed",
			"estimated_remaining_seconds": 0,
			"heartbeat_at":                finished,
			"package_path":                packagePath,
			"finished_at":                 finished,
			"expires_at":                  nil,
			"updated_at":                  finished,
			"retryable":                   false,
			"error_message":               nil,
			"error_code":                  nil,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		_ = os.RemoveAll(packageDir)
		return nil
	}
	keepWorkDir = true
	return nil
}

func (s *QueueService) postImageArchiveJobStillCurrent(ctx context.Context, jobID, sourceSignature string) (bool, error) {
	var job domain.ImageProtectionJob
	if err := s.db.WithContext(ctx).Select("post_id", "status", "source_signature").Where("job_id = ?", jobID).First(&job).Error; err != nil {
		return false, err
	}
	if job.Status == imageProtectionStatusExpired || job.SourceSignature != sourceSignature {
		return false, nil
	}
	var images []domain.PostImage
	if err := s.db.WithContext(ctx).Where("post_id = ?", job.PostID).Order("sort_order ASC, id ASC").Find(&images).Error; err != nil {
		return false, err
	}
	return postArchiveImageSignature(images) == sourceSignature, nil
}

func postArchiveImageSignature(images []domain.PostImage) string {
	ordered := append([]domain.PostImage(nil), images...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].SortOrder != ordered[j].SortOrder {
			return ordered[i].SortOrder < ordered[j].SortOrder
		}
		return ordered[i].ID < ordered[j].ID
	})
	hash := sha256.New()
	for _, image := range ordered {
		_, _ = fmt.Fprintf(hash, "%d\x00%d\x00%s\x00%t\x00%t\n", image.ID, image.SortOrder, image.ImageURL, image.IsFreePreview, image.IsProtected)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func selectPostArchiveImages(images []domain.PostImage, requestedImageIDs []int64) ([]domain.PostImage, error) {
	requested := make(map[int64]struct{}, len(requestedImageIDs))
	for _, imageID := range requestedImageIDs {
		requested[imageID] = struct{}{}
	}
	selected := make([]domain.PostImage, 0, len(images))
	for _, image := range images {
		if len(requested) > 0 {
			if _, ok := requested[image.ID]; !ok {
				continue
			}
		}
		selected = append(selected, image)
	}
	if len(requested) > 0 && len(selected) != len(requested) {
		return nil, errors.New("post image archive selection changed")
	}
	return selected, nil
}
