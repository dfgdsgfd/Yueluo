package handlers

import (
	"context"
	"errors"
	"maps"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"

	"go.uber.org/zap"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/services"
)

type postFileDeletionResult struct {
	Path         string `json:"path"`
	RecycledPath string `json:"recycled_path,omitempty"`
	Kind         string `json:"kind"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
}

type postFileDeletionSummary struct {
	Results  []postFileDeletionResult
	Recycled int
	Deleted  int
	Missing  int
	Skipped  int
	Failed   int
}

type localFileRecycleRef struct {
	Path  string
	IsDir bool
}

func (h NativeHandlers) postLocalFileRecycleInputs(ctx context.Context, postIDs []int64) []services.FileRecycleInput {
	if h.DB == nil || len(postIDs) == 0 || !h.Config.Upload.DeletePostUploadsOnDelete {
		return nil
	}
	userByPostID := map[int64]int64{}
	var posts []domain.Post
	if err := h.DB.WithContext(ctx).Model(&domain.Post{}).Select("id", "user_id").Where("id IN ?", postIDs).Find(&posts).Error; err == nil {
		for _, post := range posts {
			userByPostID[post.ID] = post.UserID
		}
	}
	seen := map[string]bool{}
	inputs := []services.FileRecycleInput{}
	add := func(postID int64, kind string, rawURL string, extra map[string]any) {
		rawURL = strings.TrimSpace(rawURL)
		if rawURL == "" {
			return
		}
		for _, ref := range h.localRecycleRefsFromStoredFileURL(rawURL) {
			if ref.Path == "" || seen[ref.Path] {
				continue
			}
			seen[ref.Path] = true
			postIDValue := postID
			userIDValue := userByPostID[postID]
			metadata := map[string]any{"source": "post_delete"}
			maps.Copy(metadata, extra)
			inputs = append(inputs, services.FileRecycleInput{
				ResourceType: "post",
				ResourceID:   postID,
				PostID:       &postIDValue,
				UserID:       &userIDValue,
				Kind:         kind,
				Storage:      services.FileRecycleStorageLocal,
				OriginalURL:  rawURL,
				OriginalPath: ref.Path,
				IsDir:        ref.IsDir,
				Metadata:     metadata,
			})
		}
	}

	var images []domain.PostImage
	if err := h.DB.WithContext(ctx).Where("post_id IN ?", postIDs).Find(&images).Error; err == nil {
		for _, image := range images {
			add(image.PostID, "image", image.ImageURL, map[string]any{"model": "post_images"})
		}
	}

	var attachments []domain.PostAttachment
	if err := h.DB.WithContext(ctx).Where("post_id IN ?", postIDs).Find(&attachments).Error; err == nil {
		for _, attachment := range attachments {
			add(attachment.PostID, "attachment", attachment.AttachmentURL, map[string]any{"model": "post_attachments"})
		}
	}

	var videos []domain.PostVideo
	if err := h.DB.WithContext(ctx).Where("post_id IN ?", postIDs).Find(&videos).Error; err == nil {
		for _, video := range videos {
			add(video.PostID, "video_source", video.VideoURL, map[string]any{"model": "post_videos", "field": "video_url"})
			add(video.PostID, "video_cover", stringPtrValue(video.CoverURL), map[string]any{"model": "post_videos", "field": "cover_url"})
			add(video.PostID, "video_dash", stringPtrValue(video.DashURL), map[string]any{"model": "post_videos", "field": "dash_url"})
			add(video.PostID, "video_preview", stringPtrValue(video.PreviewVideoURL), map[string]any{"model": "post_videos", "field": "preview_video_url"})
		}
	}

	return inputs
}

func (h NativeHandlers) postLocalFilePaths(ctx context.Context, postIDs []int64) []string {
	inputs := h.postLocalFileRecycleInputs(ctx, postIDs)
	paths := make([]string, 0, len(inputs))
	for _, input := range inputs {
		if input.OriginalPath != "" {
			paths = append(paths, input.OriginalPath)
		}
	}
	return paths
}

func (h NativeHandlers) recycleLocalPostFiles(ctx context.Context, inputs []services.FileRecycleInput) postFileDeletionSummary {
	if len(inputs) == 0 {
		return postFileDeletionSummary{Results: []postFileDeletionResult{}}
	}
	if h.FileRecycle != nil && h.FileRecycle.Enabled() {
		summary, err := h.FileRecycle.RecycleLocal(ctx, inputs)
		if err == nil {
			return postFileDeletionSummaryFromRecycle(summary)
		}
		if h.Config.Server.Env != "test" {
			zap.L().Warn("file recycle failed, falling back to hard delete", zap.Error(err))
		}
	}
	paths := make([]string, 0, len(inputs))
	seen := map[string]bool{}
	for _, input := range inputs {
		path := strings.TrimSpace(input.OriginalPath)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	return h.deleteLocalPostFiles(paths)
}

func postFileDeletionSummaryFromRecycle(summary services.FileRecycleSummary) postFileDeletionSummary {
	out := postFileDeletionSummary{Results: make([]postFileDeletionResult, 0, len(summary.Results))}
	for _, result := range summary.Results {
		item := postFileDeletionResult{
			Path:         result.OriginalPath,
			RecycledPath: result.RecycledPath,
			Kind:         result.Kind,
			Status:       result.Status,
			Error:        result.Error,
		}
		out.Results = append(out.Results, item)
		switch result.Status {
		case services.FileRecycleStatusRecycled:
			out.Recycled++
		case services.FileRecycleStatusMissing:
			out.Missing++
		case services.FileRecycleStatusSkipped:
			out.Skipped++
		default:
			out.Failed++
		}
	}
	return out
}

func (h NativeHandlers) localRecycleRefsFromStoredFileURL(raw string) []localFileRecycleRef {
	canonicalPath, _, ok := h.localFilePathFromURL(raw)
	if !ok {
		return nil
	}
	rest := strings.TrimPrefix(canonicalPath, "/api/file/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return nil
	}
	isDashManifest := parts[0] == "videos" && strings.Contains(parts[1], "/dash/") && pathpkg.Base(parts[1]) == "manifest.mpd"
	var fallback string
	for _, dir := range h.uploadCandidateDirs(parts[0]) {
		candidate, ok := safeJoinRelative(dir, parts[1])
		if !ok {
			continue
		}
		if isDashManifest {
			candidate = filepath.Dir(candidate)
		}
		if fallback == "" {
			fallback = candidate
		}
		if stat, err := os.Stat(candidate); err == nil {
			if isDashManifest && stat.IsDir() {
				return []localFileRecycleRef{{Path: candidate, IsDir: true}}
			}
			if !isDashManifest && stat.Mode().IsRegular() {
				return []localFileRecycleRef{{Path: candidate, IsDir: false}}
			}
		}
	}
	if fallback != "" {
		return []localFileRecycleRef{{Path: fallback, IsDir: isDashManifest}}
	}
	return nil
}

func (h NativeHandlers) postFileURLs(ctx context.Context, postIDs []int64) []string {
	urls := []string{}

	var images []domain.PostImage
	if err := h.DB.WithContext(ctx).Where("post_id IN ?", postIDs).Find(&images).Error; err == nil {
		for _, image := range images {
			urls = append(urls, image.ImageURL)
		}
	}

	var attachments []domain.PostAttachment
	if err := h.DB.WithContext(ctx).Where("post_id IN ?", postIDs).Find(&attachments).Error; err == nil {
		for _, attachment := range attachments {
			urls = append(urls, attachment.AttachmentURL)
		}
	}

	var videos []domain.PostVideo
	if err := h.DB.WithContext(ctx).Where("post_id IN ?", postIDs).Find(&videos).Error; err == nil {
		for _, video := range videos {
			urls = append(urls, video.VideoURL)
			urls = append(urls, stringPtrValue(video.CoverURL))
			urls = append(urls, stringPtrValue(video.DashURL))
			urls = append(urls, stringPtrValue(video.PreviewVideoURL))
		}
	}

	return urls
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func (h NativeHandlers) localPathsFromStoredFileURL(raw string) []string {
	refs := h.localRecycleRefsFromStoredFileURL(raw)
	paths := make([]string, 0, len(refs))
	for _, ref := range refs {
		if ref.Path != "" {
			paths = append(paths, ref.Path)
		}
	}
	return paths
}

func (h NativeHandlers) deleteLocalPostFiles(paths []string) postFileDeletionSummary {
	summary := postFileDeletionSummary{Results: []postFileDeletionResult{}}
	for _, rawPath := range paths {
		path := strings.TrimSpace(rawPath)
		if path == "" {
			continue
		}
		result := postFileDeletionResult{Path: path, Kind: "file"}
		if stat, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				result.Status = "missing"
				summary.Missing++
			} else {
				result.Status = "failed"
				result.Error = err.Error()
				summary.Failed++
			}
			summary.Results = append(summary.Results, result)
			continue
		} else if stat.IsDir() {
			result.Kind = "dir"
			if !h.safeRemovableUploadDir(path) {
				result.Status = "skipped"
				result.Error = "unsafe_upload_dir"
				summary.Skipped++
				summary.Results = append(summary.Results, result)
				continue
			}
			if err := os.RemoveAll(path); err != nil {
				result.Status = "failed"
				result.Error = err.Error()
				summary.Failed++
			} else {
				result.Status = "deleted"
				summary.Deleted++
			}
			summary.Results = append(summary.Results, result)
			continue
		}
		if err := os.Remove(path); err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			summary.Failed++
		} else {
			result.Status = "deleted"
			summary.Deleted++
		}
		summary.Results = append(summary.Results, result)
	}
	return summary
}

func (h NativeHandlers) safeRemovableUploadDir(dir string) bool {
	dir = filepath.Clean(dir)
	for _, root := range append(h.uploadRoots(), h.uploadTypeDirs("videos")...) {
		root = filepath.Clean(root)
		rel, err := filepath.Rel(root, dir)
		if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	if h.Config.Server.Env != "test" {
		zap.L().Warn("refused to remove path outside upload roots", zap.String("path", dir))
	}
	return false
}

func (s postFileDeletionSummary) attempted() int {
	return len(s.Results)
}

func (s postFileDeletionSummary) outcome() string {
	success := s.Deleted + s.Recycled
	switch {
	case s.Failed > 0 && success > 0:
		return "partial"
	case s.Failed > 0:
		return "failure"
	case s.Skipped > 0 && success > 0:
		return "partial"
	case s.Skipped > 0:
		return "skipped"
	case success > 0:
		return "success"
	default:
		return "skipped"
	}
}

func (s postFileDeletionSummary) auditMetadata() map[string]any {
	results := make([]map[string]any, 0, len(s.Results))
	for _, result := range s.Results {
		item := map[string]any{
			"path":   result.Path,
			"kind":   result.Kind,
			"status": result.Status,
		}
		if result.RecycledPath != "" {
			item["recycled_path"] = result.RecycledPath
		}
		if result.Error != "" {
			item["error"] = result.Error
		}
		results = append(results, item)
	}
	return map[string]any{
		"attempted_count": len(s.Results),
		"deleted_count":   s.Deleted,
		"recycled_count":  s.Recycled,
		"missing_count":   s.Missing,
		"skipped_count":   s.Skipped,
		"failed_count":    s.Failed,
		"results":         results,
	}
}
