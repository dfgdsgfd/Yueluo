package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

type imageProtectionStageError struct {
	stage      string
	imageIndex int
	imageID    int64
	err        error
}

func (e imageProtectionStageError) Error() string {
	detail := e.stage
	if e.imageIndex > 0 {
		detail = fmt.Sprintf("%s image %d", detail, e.imageIndex)
	}
	if e.imageID > 0 {
		detail = fmt.Sprintf("%s (id %d)", detail, e.imageID)
	}
	if e.err == nil {
		return detail
	}
	return detail + ": " + e.err.Error()
}

func (e imageProtectionStageError) Unwrap() error {
	return e.err
}

func (s *QueueService) cleanupExpiredImageProtectionPackages(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	var jobs []domain.ImageProtectionJob
	now := time.Now()
	if err := s.db.WithContext(ctx).Where("expires_at IS NOT NULL AND expires_at < ? AND package_path <> ''", now).Find(&jobs).Error; err != nil {
		return err
	}
	for _, job := range jobs {
		if job.PackagePath != "" {
			_ = os.RemoveAll(filepath.Dir(job.PackagePath))
		}
		_ = s.db.WithContext(ctx).Model(&domain.ImageProtectionJob{}).Where("id = ?", job.ID).Updates(map[string]any{
			"status":       imageProtectionStatusExpired,
			"package_path": "",
			"updated_at":   now,
		}).Error
	}
	return nil
}

func queueConfiguredUploadTypeDirs(cfg config.Config, fileType string) []string {
	dirs := []string{}
	switch fileType {
	case "images":
		dirs = append(dirs, cfg.Upload.Image.LocalUploadDir)
	case "covers":
		dirs = append(dirs, cfg.Upload.Video.CoverDir)
	case "media":
		dirs = append(dirs, "uploads/media")
	case "thumbnails":
		dirs = append(dirs, "uploads/thumbnails")
	}
	if root := strings.TrimSpace(cfg.Upload.RootDir); root != "" {
		dirs = append(dirs, filepath.Join(root, fileType))
	}
	out := make([]string, 0, len(dirs))
	seen := map[string]struct{}{}
	for _, dir := range dirs {
		dir = serviceAbsPath(dir)
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		out = append(out, dir)
	}
	return out
}

func imageProtectionContentType(data []byte, declared string) string {
	if parsed, _, err := mime.ParseMediaType(strings.TrimSpace(declared)); err == nil && strings.HasPrefix(parsed, "image/") {
		return parsed
	}
	if len(data) > 0 {
		detected := http.DetectContentType(data[:minIntImageProtection(len(data), 512)])
		if strings.HasPrefix(detected, "image/") {
			return detected
		}
	}
	return "application/octet-stream"
}

func imageProtectionFilename(value string) string {
	value = filepath.Base(strings.TrimSpace(value))
	if value == "." || value == "" {
		return "image"
	}
	return value
}

func imageProtectionShortMarker(secret string, jobID string, postID, viewerID, imageID int64) string {
	sum := sha256.Sum256([]byte(imageProtectionHMAC(secret, jobID, postID, viewerID, imageID)))
	return hex.EncodeToString(sum[:])[:4]
}

func imageProtectionHMAC(secret string, jobID string, postID, viewerID, imageID int64) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(fmt.Appendf(nil, "%s|%d|%d|%d", jobID, postID, viewerID, imageID))
	return hex.EncodeToString(mac.Sum(nil))
}

func minIntImageProtection(a, b int) int {
	if a < b {
		return a
	}
	return b
}
