package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

const (
	FileRecycleStorageLocal = "local"

	FileRecycleStatusRecycled    = "recycled"
	FileRecycleStatusMissing     = "missing"
	FileRecycleStatusSkipped     = "skipped"
	FileRecycleStatusFailed      = "failed"
	FileRecycleStatusPurged      = "purged"
	FileRecycleStatusPurgeFailed = "purge_failed"
)

type FileRecycleService struct {
	db       *gorm.DB
	cfg      config.Config
	logger   *zap.Logger
	settings *SettingsService
}

type FileRecycleInput struct {
	GroupID      string
	ResourceType string
	ResourceID   int64
	PostID       *int64
	UserID       *int64
	Kind         string
	Storage      string
	OriginalURL  string
	OriginalPath string
	IsDir        bool
	Metadata     map[string]any
}

type FileRecycleResult struct {
	ID           int64  `json:"id,omitempty"`
	GroupID      string `json:"group_id,omitempty"`
	ResourceType string `json:"resource_type,omitempty"`
	ResourceID   int64  `json:"resource_id,omitempty"`
	PostID       *int64 `json:"post_id,omitempty"`
	UserID       *int64 `json:"user_id,omitempty"`
	Kind         string `json:"kind,omitempty"`
	OriginalURL  string `json:"original_url,omitempty"`
	OriginalPath string `json:"original_path,omitempty"`
	RecycledPath string `json:"recycled_path,omitempty"`
	IsDir        bool   `json:"is_dir,omitempty"`
	FileCount    int64  `json:"file_count,omitempty"`
	SizeBytes    int64  `json:"size_bytes,omitempty"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
}

type FileRecycleSummary struct {
	Results  []FileRecycleResult `json:"results"`
	Recycled int                 `json:"recycled"`
	Missing  int                 `json:"missing"`
	Skipped  int                 `json:"skipped"`
	Failed   int                 `json:"failed"`
	Purged   int                 `json:"purged"`
}

func NewFileRecycleService(db *gorm.DB, cfg config.Config, logger *zap.Logger) *FileRecycleService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FileRecycleService{db: db, cfg: cfg, logger: logger}
}

func NewFileRecycleServiceWithSettings(db *gorm.DB, cfg config.Config, logger *zap.Logger, settings *SettingsService) *FileRecycleService {
	service := NewFileRecycleService(db, cfg, logger)
	service.settings = settings
	return service
}

func (s *FileRecycleService) Enabled() bool {
	return s != nil && s.cfg.Upload.Recycle.Enabled
}

func (s *FileRecycleService) WithSettings(settings *SettingsService) *FileRecycleService {
	if s != nil {
		s.settings = settings
	}
	return s
}

func (s *FileRecycleService) Retention() time.Duration {
	fallback := DefaultFileRecycleRetentionDays
	if s != nil && s.cfg.Upload.Recycle.Retention > 0 {
		fallback = int((s.cfg.Upload.Recycle.Retention + 24*time.Hour - 1) / (24 * time.Hour))
	}
	days := fallback
	if s != nil && s.settings != nil {
		days = s.settings.Int(FileRecycleRetentionDaysKey, fallback)
	}
	return time.Duration(clampFileRecycleRetentionDays(days)) * 24 * time.Hour
}

func (s *FileRecycleService) CleanupInterval() time.Duration {
	fallback := DefaultFileRecycleCleanupIntervalHours
	if s != nil && s.cfg.Upload.Recycle.CleanupInterval > 0 {
		fallback = int((s.cfg.Upload.Recycle.CleanupInterval + time.Hour - 1) / time.Hour)
	}
	hours := fallback
	if s != nil && s.settings != nil {
		hours = s.settings.Int(FileRecycleCleanupIntervalHoursKey, fallback)
	}
	return time.Duration(clampFileRecycleCleanupIntervalHours(hours)) * time.Hour
}

func (s *FileRecycleService) RecycleRoot() string {
	if s == nil {
		return ""
	}
	root := strings.TrimSpace(s.cfg.Upload.Recycle.RootDir)
	if root == "" {
		root = filepath.Join(firstNonEmptyString(s.cfg.Upload.RootDir, "uploads"), ".trash")
	}
	return serviceAbsPath(root)
}

func (s *FileRecycleService) RecycleLocal(ctx context.Context, inputs []FileRecycleInput) (FileRecycleSummary, error) {
	summary := FileRecycleSummary{Results: []FileRecycleResult{}}
	if s == nil || len(inputs) == 0 {
		return summary, nil
	}
	if !s.cfg.Upload.Recycle.Enabled {
		return s.skipAll(ctx, inputs, "recycle_disabled"), nil
	}
	now := time.Now()
	retention := s.Retention()
	defaultGroupID := s.newGroupID(now)
	manifestGroups := map[string][]FileRecycleResult{}
	for _, input := range inputs {
		input = s.normalizeInput(input, defaultGroupID)
		result := s.recycleOne(ctx, input, now, now.Add(retention))
		summary.Results = append(summary.Results, result)
		manifestGroups[input.GroupID] = append(manifestGroups[input.GroupID], result)
		switch result.Status {
		case FileRecycleStatusRecycled:
			summary.Recycled++
		case FileRecycleStatusMissing:
			summary.Missing++
		case FileRecycleStatusSkipped:
			summary.Skipped++
		default:
			summary.Failed++
		}
	}
	for groupID, results := range manifestGroups {
		if err := s.writeManifest(groupID, results, now); err != nil {
			s.logger.Warn("write file recycle manifest failed", zap.String("group_id", groupID), zap.Error(err))
		}
	}
	return summary, nil
}

func (s *FileRecycleService) PurgeIDs(ctx context.Context, ids []int64) (FileRecycleSummary, error) {
	summary := FileRecycleSummary{Results: []FileRecycleResult{}}
	if s == nil || s.db == nil || len(ids) == 0 {
		return summary, nil
	}
	var items []domain.FileRecycleItem
	if err := s.db.WithContext(ctx).Where("id IN ?", ids).Order("purge_after ASC, deleted_at ASC, id ASC").Find(&items).Error; err != nil {
		return summary, err
	}
	return s.purgeItems(ctx, items), nil
}

func (s *FileRecycleService) PurgeExpired(ctx context.Context, limit int) (FileRecycleSummary, error) {
	summary := FileRecycleSummary{Results: []FileRecycleResult{}}
	if s == nil || s.db == nil {
		return summary, nil
	}
	if limit <= 0 {
		limit = 200
	}
	var items []domain.FileRecycleItem
	err := s.db.WithContext(ctx).
		Where("purge_after <= ? AND status <> ?", time.Now(), FileRecycleStatusPurged).
		Order("purge_after ASC, deleted_at ASC, id ASC").
		Limit(limit).
		Find(&items).Error
	if err != nil {
		return summary, err
	}
	return s.purgeItems(ctx, items), nil
}

func (s *FileRecycleService) RecycleOrphanedDASH(ctx context.Context, grace time.Duration, limit int) (FileRecycleSummary, error) {
	summary := FileRecycleSummary{Results: []FileRecycleResult{}}
	if s == nil || s.db == nil || !s.cfg.Upload.Recycle.Enabled {
		return summary, nil
	}
	if grace <= 0 {
		grace = 2 * time.Hour
	}
	if limit <= 0 {
		limit = 100
	}
	dashRoot := filepath.Join(serviceAbsPath(s.cfg.Upload.Video.LocalUploadDir), "dash")
	dashRoot = filepath.Clean(dashRoot)
	if _, err := os.Stat(dashRoot); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return summary, nil
		}
		return summary, err
	}
	cutoff := time.Now().Add(-grace)
	inputs := make([]FileRecycleInput, 0)
	err := filepath.WalkDir(dashRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == dashRoot || len(inputs) >= limit {
			return nil
		}
		manifest := filepath.Join(path, "manifest.mpd")
		info, statErr := os.Stat(manifest)
		if statErr != nil || info.IsDir() || info.ModTime().After(cutoff) {
			return nil
		}
		rel, relErr := filepath.Rel(dashRoot, path)
		if relErr != nil || rel == "." || strings.HasPrefix(rel, "..") {
			return nil
		}
		dashURL := "/api/file/videos/dash/" + filepath.ToSlash(rel) + "/manifest.mpd"
		var count int64
		if err := s.db.WithContext(ctx).Model(&domain.PostVideo{}).Where("dash_url = ?", dashURL).Count(&count).Error; err != nil || count > 0 {
			return nil
		}
		inputs = append(inputs, FileRecycleInput{
			ResourceType: "video_dash_orphan",
			Kind:         "video_dash_orphan",
			Storage:      FileRecycleStorageLocal,
			OriginalURL:  dashURL,
			OriginalPath: path,
			IsDir:        true,
			Metadata:     map[string]any{"source": "dash_orphan_sweep"},
		})
		return nil
	})
	if err != nil {
		return summary, err
	}
	return s.RecycleLocal(ctx, inputs)
}

func (s *FileRecycleService) recycleOne(ctx context.Context, input FileRecycleInput, deletedAt time.Time, purgeAfter time.Time) FileRecycleResult {
	result := FileRecycleResult{
		GroupID:      input.GroupID,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		PostID:       input.PostID,
		UserID:       input.UserID,
		Kind:         input.Kind,
		OriginalURL:  input.OriginalURL,
		OriginalPath: input.OriginalPath,
		IsDir:        input.IsDir,
	}
	sourcePath := serviceAbsPath(input.OriginalPath)
	if sourcePath == "" {
		result.Status = FileRecycleStatusSkipped
		result.Error = "empty_path"
		return s.recordRecycleItem(ctx, input, result, deletedAt, purgeAfter)
	}
	result.OriginalPath = sourcePath
	safe, rel := s.safeRecycleSourcePath(sourcePath, input.IsDir)
	if !safe {
		result.Status = FileRecycleStatusSkipped
		result.Error = "unsafe_upload_path"
		return s.recordRecycleItem(ctx, input, result, deletedAt, purgeAfter)
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result.Status = FileRecycleStatusMissing
			result.Error = "missing"
			return s.recordRecycleItem(ctx, input, result, deletedAt, purgeAfter)
		}
		result.Status = FileRecycleStatusFailed
		result.Error = err.Error()
		return s.recordRecycleItem(ctx, input, result, deletedAt, purgeAfter)
	}
	result.IsDir = info.IsDir()
	count, size, err := pathStats(sourcePath)
	if err != nil {
		result.Status = FileRecycleStatusFailed
		result.Error = err.Error()
		return s.recordRecycleItem(ctx, input, result, deletedAt, purgeAfter)
	}
	result.FileCount = count
	result.SizeBytes = size
	destination := s.uniqueDestinationPath(s.destinationPath(input, rel, deletedAt))
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		result.Status = FileRecycleStatusFailed
		result.Error = err.Error()
		return s.recordRecycleItem(ctx, input, result, deletedAt, purgeAfter)
	}
	if err := movePath(sourcePath, destination); err != nil {
		result.Status = FileRecycleStatusFailed
		result.Error = err.Error()
		return s.recordRecycleItem(ctx, input, result, deletedAt, purgeAfter)
	}
	result.RecycledPath = destination
	result.Status = FileRecycleStatusRecycled
	return s.recordRecycleItem(ctx, input, result, deletedAt, purgeAfter)
}

func (s *FileRecycleService) skipAll(ctx context.Context, inputs []FileRecycleInput, reason string) FileRecycleSummary {
	now := time.Now()
	summary := FileRecycleSummary{Results: make([]FileRecycleResult, 0, len(inputs))}
	defaultGroupID := s.newGroupID(now)
	for _, input := range inputs {
		input = s.normalizeInput(input, defaultGroupID)
		result := FileRecycleResult{
			GroupID:      input.GroupID,
			ResourceType: input.ResourceType,
			ResourceID:   input.ResourceID,
			PostID:       input.PostID,
			UserID:       input.UserID,
			Kind:         input.Kind,
			OriginalURL:  input.OriginalURL,
			OriginalPath: input.OriginalPath,
			Status:       FileRecycleStatusSkipped,
			Error:        reason,
		}
		summary.Results = append(summary.Results, s.recordRecycleItem(ctx, input, result, now, now))
		summary.Skipped++
	}
	return summary
}

func (s *FileRecycleService) purgeItems(ctx context.Context, items []domain.FileRecycleItem) FileRecycleSummary {
	summary := FileRecycleSummary{Results: []FileRecycleResult{}}
	now := time.Now()
	for _, item := range items {
		result := fileRecycleResultFromItem(item)
		if item.Status != FileRecycleStatusPurged && strings.TrimSpace(item.RecycledPath) != "" {
			if err := s.removeRecycledPath(item.RecycledPath, item.IsDir); err != nil && !errors.Is(err, os.ErrNotExist) {
				result.Status = FileRecycleStatusPurgeFailed
				result.Error = err.Error()
				summary.Failed++
				_ = s.db.WithContext(ctx).Model(&domain.FileRecycleItem{}).Where("id = ?", item.ID).Updates(map[string]any{
					"status":     FileRecycleStatusPurgeFailed,
					"error":      err.Error(),
					"updated_at": now,
				}).Error
				summary.Results = append(summary.Results, result)
				continue
			}
		}
		result.Status = FileRecycleStatusPurged
		result.Error = ""
		result.RecycledPath = item.RecycledPath
		summary.Purged++
		_ = s.db.WithContext(ctx).Model(&domain.FileRecycleItem{}).Where("id = ?", item.ID).Updates(map[string]any{
			"status":     FileRecycleStatusPurged,
			"purged_at":  now,
			"error":      "",
			"updated_at": now,
		}).Error
		summary.Results = append(summary.Results, result)
	}
	return summary
}

func (s *FileRecycleService) recordRecycleItem(ctx context.Context, input FileRecycleInput, result FileRecycleResult, deletedAt time.Time, purgeAfter time.Time) FileRecycleResult {
	if s.db == nil {
		return result
	}
	meta := input.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	if result.Error != "" {
		meta["error"] = result.Error
	}
	metaBytes, _ := json.Marshal(meta)
	item := domain.FileRecycleItem{
		GroupID:      input.GroupID,
		ResourceType: input.ResourceType,
		ResourceID:   input.ResourceID,
		PostID:       input.PostID,
		UserID:       input.UserID,
		Kind:         input.Kind,
		Storage:      firstNonEmptyString(input.Storage, FileRecycleStorageLocal),
		OriginalURL:  input.OriginalURL,
		OriginalPath: result.OriginalPath,
		RecycledPath: result.RecycledPath,
		IsDir:        result.IsDir,
		FileCount:    result.FileCount,
		SizeBytes:    result.SizeBytes,
		Status:       result.Status,
		DeletedAt:    deletedAt,
		PurgeAfter:   purgeAfter,
		Error:        result.Error,
		Metadata:     datatypes.JSON(metaBytes),
		CreatedAt:    deletedAt,
	}
	if err := s.db.WithContext(ctx).Create(&item).Error; err != nil {
		result.Status = FileRecycleStatusFailed
		result.Error = err.Error()
		return result
	}
	result.ID = item.ID
	return result
}

func (s *FileRecycleService) normalizeInput(input FileRecycleInput, defaultGroupID string) FileRecycleInput {
	if strings.TrimSpace(input.GroupID) == "" {
		input.GroupID = defaultGroupID
	}
	if strings.TrimSpace(input.ResourceType) == "" {
		input.ResourceType = "unknown"
	}
	if strings.TrimSpace(input.Kind) == "" {
		input.Kind = "file"
	}
	if strings.TrimSpace(input.Storage) == "" {
		input.Storage = FileRecycleStorageLocal
	}
	return input
}

func (s *FileRecycleService) newGroupID(now time.Time) string {
	return strconv.FormatInt(now.UnixNano(), 36) + "-" + shortHash(now.String())[:8]
}

func (s *FileRecycleService) destinationPath(input FileRecycleInput, sourceRel string, deletedAt time.Time) string {
	groupRoot := s.groupRoot(input, deletedAt)
	kindDir := recycleKindDir(input.Kind)
	if sourceRel == "" {
		sourceRel = filepath.Base(input.OriginalPath)
	}
	sourceRel = safeRelativePath(filepath.ToSlash(sourceRel))
	if sourceRel == "" {
		sourceRel = shortHash(input.OriginalPath)
	}
	return filepath.Join(groupRoot, "files", filepath.FromSlash(kindDir), filepath.FromSlash(sourceRel))
}

func (s *FileRecycleService) groupRoot(input FileRecycleInput, deletedAt time.Time) string {
	id := input.ResourceID
	if id == 0 && input.PostID != nil {
		id = *input.PostID
	}
	resourceID := "resource-" + strconv.FormatInt(id, 10)
	if input.ResourceType == "post" && id > 0 {
		resourceID = "post-" + strconv.FormatInt(id, 10)
	}
	return filepath.Join(
		s.RecycleRoot(),
		safeRelativePath(input.ResourceType),
		deletedAt.Format("2006"),
		deletedAt.Format("01"),
		deletedAt.Format("02"),
		resourceID,
		safeRelativePath(input.GroupID),
	)
}

func (s *FileRecycleService) writeManifest(groupID string, results []FileRecycleResult, deletedAt time.Time) error {
	if len(results) == 0 {
		return nil
	}
	input := FileRecycleInput{GroupID: groupID, ResourceType: results[0].ResourceType, ResourceID: results[0].ResourceID, PostID: results[0].PostID}
	groupRoot := s.groupRoot(input, deletedAt)
	if err := os.MkdirAll(groupRoot, 0755); err != nil {
		return err
	}
	payload := map[string]any{
		"group_id":   groupID,
		"created_at": deletedAt.UTC().Format(time.RFC3339Nano),
		"items":      results,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(groupRoot, "manifest.json"), data, 0644)
}

func (s *FileRecycleService) safeRecycleSourcePath(path string, isDir bool) (bool, string) {
	path = filepath.Clean(serviceAbsPath(path))
	recycleRoot := s.RecycleRoot()
	if withinRoot(path, recycleRoot) {
		return false, ""
	}
	root := s.matchingUploadRoot(path)
	if root == "" {
		return false, ""
	}
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." || rel == "" || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return false, ""
	}
	if isDir && rel == "." {
		return false, ""
	}
	return true, rel
}

func (s *FileRecycleService) matchingUploadRoot(path string) string {
	path = filepath.Clean(serviceAbsPath(path))
	best := ""
	for _, root := range s.uploadRoots() {
		root = filepath.Clean(serviceAbsPath(root))
		if root == "" || !withinRoot(path, root) {
			continue
		}
		if len(root) > len(best) {
			best = root
		}
	}
	return best
}

func (s *FileRecycleService) uploadRoots() []string {
	cfg := s.cfg.Upload
	roots := []string{
		cfg.RootDir,
		cfg.Image.LocalUploadDir,
		cfg.Video.LocalUploadDir,
		cfg.Video.LegacyUploadDir,
		cfg.Video.CoverDir,
		cfg.Attachment.LocalUploadDir,
		cfg.AvatarDir,
		cfg.BannerDir,
		cfg.Temp.RootDir,
		filepath.Join(cfg.RootDir, "images"),
		filepath.Join(cfg.RootDir, "attachments"),
		filepath.Join(cfg.RootDir, "videos"),
		filepath.Join(cfg.RootDir, "covers"),
		filepath.Join(cfg.RootDir, "thumbnails"),
		filepath.Join(cfg.RootDir, "media"),
		filepath.Join(cfg.RootDir, "apk"),
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		root = filepath.Clean(serviceAbsPath(root))
		if root == s.RecycleRoot() || seen[root] {
			continue
		}
		seen[root] = true
		out = append(out, root)
	}
	return out
}

func (s *FileRecycleService) uniqueDestinationPath(path string) string {
	path = filepath.Clean(path)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; i < 1000; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
	return fmt.Sprintf("%s-%d%s", base, time.Now().UnixNano(), ext)
}

func recycleKindDir(kind string) string {
	switch strings.TrimSpace(kind) {
	case "image", "post_image", "images":
		return "images"
	case "attachment", "post_attachment", "attachments":
		return "attachments"
	case "video_cover", "cover", "covers":
		return "covers"
	case "thumbnail", "thumbnails":
		return "thumbnails"
	case "video_dash", "video_dash_orphan":
		return "videos/dash"
	case "video", "video_source", "video_preview", "preview_video", "videos":
		return "videos"
	default:
		return safeRelativePath(kind)
	}
}

func fileRecycleResultFromItem(item domain.FileRecycleItem) FileRecycleResult {
	return FileRecycleResult{
		ID:           item.ID,
		GroupID:      item.GroupID,
		ResourceType: item.ResourceType,
		ResourceID:   item.ResourceID,
		PostID:       item.PostID,
		UserID:       item.UserID,
		Kind:         item.Kind,
		OriginalURL:  item.OriginalURL,
		OriginalPath: item.OriginalPath,
		RecycledPath: item.RecycledPath,
		IsDir:        item.IsDir,
		FileCount:    item.FileCount,
		SizeBytes:    item.SizeBytes,
		Status:       item.Status,
		Error:        item.Error,
	}
}

func pathStats(path string) (int64, int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, err
	}
	if !info.IsDir() {
		return 1, info.Size(), nil
	}
	var count int64
	var size int64
	err = filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		count++
		size += info.Size()
		return nil
	})
	return count, size, err
}

func movePath(source, destination string) error {
	source = filepath.Clean(source)
	destination = filepath.Clean(destination)
	if err := os.Rename(source, destination); err == nil {
		return nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return err
	}
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := copyRecycledDir(source, destination); err != nil {
			return err
		}
		return os.RemoveAll(source)
	}
	if err := copyRecycledFile(source, destination, info.Mode()); err != nil {
		return err
	}
	return os.Remove(source)
}

func copyRecycledDir(source, destination string) error {
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			return os.MkdirAll(target, info.Mode().Perm())
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyRecycledFile(path, target, info.Mode())
	})
}

func copyRecycledFile(source, destination string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(destination, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode.Perm())
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}

func withinRoot(path, root string) bool {
	path = filepath.Clean(serviceAbsPath(path))
	root = filepath.Clean(serviceAbsPath(root))
	if path == "" || root == "" {
		return false
	}
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
