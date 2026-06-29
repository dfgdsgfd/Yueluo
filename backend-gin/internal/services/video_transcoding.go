package services

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hibiken/asynq"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

type VideoTranscodingInput struct {
	VideoID    int64  `json:"video_id"`
	PostID     int64  `json:"post_id"`
	UserID     int64  `json:"user_id"`
	VideoURL   string `json:"video_url"`
	SourcePath string `json:"source_path"`
	CoverURL   string `json:"cover_url"`
}

type videoTranscodingTaskPayload struct {
	VideoID    int64  `json:"video_id"`
	PostID     int64  `json:"post_id"`
	UserID     int64  `json:"user_id"`
	VideoURL   string `json:"video_url"`
	SourcePath string `json:"source_path"`
	CoverURL   string `json:"cover_url"`
	EnqueuedAt int64  `json:"enqueued_at"`
}

type videoTranscodingResult struct {
	VideoURL string `json:"video_url"`
	DashURL  string `json:"dash_url"`
	CoverURL string `json:"cover_url,omitempty"`
	Skipped  bool   `json:"skipped,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

var errVideoTranscodingTargetGone = errors.New("video transcoding target no longer exists")

func (s *QueueService) VideoTranscodingReady() bool {
	return s != nil && s.cfg.Video.Enabled && s.Available()
}

func (s *QueueService) EnqueueVideoTranscoding(ctx context.Context, input VideoTranscodingInput) (map[string]any, error) {
	if s == nil || !s.cfg.Video.Enabled {
		return nil, errors.New("video transcoding disabled")
	}
	if !s.Available() {
		return nil, errors.New("queue service disabled")
	}
	input.VideoURL = normalizeStoredVideoURL(input.VideoURL, s.cfg)
	if input.SourcePath == "" {
		input.SourcePath, _ = localVideoPathFromURL(input.VideoURL, s.cfg)
	}
	if input.VideoURL == "" {
		input.VideoURL = canonicalVideoURLFromPath(input.SourcePath, s.cfg)
	}
	if input.VideoURL == "" || input.SourcePath == "" {
		return nil, errors.New("video source is not a local upload")
	}
	payload := videoTranscodingTaskPayload{
		VideoID:    input.VideoID,
		PostID:     input.PostID,
		UserID:     input.UserID,
		VideoURL:   input.VideoURL,
		SourcePath: serviceAbsPath(input.SourcePath),
		CoverURL:   normalizeStoredVideoURL(input.CoverURL, s.cfg),
		EnqueuedAt: time.Now().UnixMilli(),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	taskID := videoTranscodingTaskID(payload)
	options := []asynq.Option{
		asynq.Queue(QueueVideoTranscoding),
		asynq.TaskID(taskID),
		asynq.MaxRetry(maxInt(0, s.cfg.Queue.Retry.Attempts)),
		asynq.Timeout(6 * time.Hour),
	}
	if retention := s.completedRetention(QueueVideoTranscoding, 72*time.Hour); retention > 0 {
		options = append(options, asynq.Retention(retention))
	}
	info, err := s.client.EnqueueContext(ctx, newQueueTask(TaskVideoTranscoding, data, QueueVideoTranscoding, payload.EnqueuedAt), options...)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate") || strings.Contains(strings.ToLower(err.Error()), "conflict") {
			s.recordQueueEvent(ctx, queueEvent{
				TaskID: taskID,
				Queue:  QueueVideoTranscoding,
				Type:   TaskVideoTranscoding,
				Event:  "duplicate",
				State:  "duplicate",
				At:     payload.EnqueuedAt,
				Detail: map[string]any{"videoId": payload.VideoID, "postId": payload.PostID},
			})
			return map[string]any{"id": taskID, "queue": QueueVideoTranscoding, "duplicate": true, "enqueuedAt": payload.EnqueuedAt}, nil
		}
		return nil, err
	}
	s.recordQueueEvent(ctx, queueEvent{
		TaskID: info.ID,
		Queue:  info.Queue,
		Type:   TaskVideoTranscoding,
		Event:  "enqueued",
		State:  info.State.String(),
		At:     payload.EnqueuedAt,
		Detail: map[string]any{"videoId": payload.VideoID, "postId": payload.PostID},
	})
	return map[string]any{"id": info.ID, "queue": info.Queue, "state": info.State.String(), "enqueuedAt": payload.EnqueuedAt}, nil
}

func (s *QueueService) EnqueueVideoTranscodingForPost(ctx context.Context, postID int64) ([]map[string]any, error) {
	if s == nil || s.db == nil || postID == 0 {
		return nil, nil
	}
	if !s.cfg.Video.Enabled || !s.Available() {
		return nil, nil
	}
	var post domain.Post
	if err := s.db.WithContext(ctx).Where("id = ?", postID).Select("id", "user_id").First(&post).Error; err != nil {
		return nil, err
	}
	var videos []domain.PostVideo
	if err := s.db.WithContext(ctx).Where("post_id = ?", postID).Find(&videos).Error; err != nil {
		return nil, err
	}
	jobs := make([]map[string]any, 0, len(videos))
	for _, video := range videos {
		if video.DashURL != nil && strings.TrimSpace(*video.DashURL) != "" {
			continue
		}
		cover := ""
		if video.CoverURL != nil {
			cover = *video.CoverURL
		}
		job, err := s.EnqueueVideoTranscoding(ctx, VideoTranscodingInput{
			VideoID:  video.ID,
			PostID:   video.PostID,
			UserID:   post.UserID,
			VideoURL: video.VideoURL,
			CoverURL: cover,
		})
		if err != nil {
			return jobs, err
		}
		jobs = append(jobs, job)
	}
	return jobs, nil
}

func (s *QueueService) processVideoTranscoding(ctx context.Context, task *asynq.Task) error {
	var payload videoTranscodingTaskPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}
	result, err := s.transcodeVideo(ctx, payload)
	if err != nil {
		return err
	}
	encoded, _ := json.Marshal(result)
	_, _ = task.ResultWriter().Write(encoded)
	return nil
}

func (s *QueueService) transcodeVideo(ctx context.Context, payload videoTranscodingTaskPayload) (videoTranscodingResult, error) {
	result := videoTranscodingResult{VideoURL: payload.VideoURL}
	sourcePath := payload.SourcePath
	if sourcePath == "" {
		sourcePath, _ = localVideoPathFromURL(payload.VideoURL, s.cfg)
	}
	sourcePath = serviceAbsPath(sourcePath)
	if sourcePath == "" {
		result.Skipped = true
		result.Reason = "source is not a local upload"
		return result, nil
	}
	if exists, err := s.videoTranscodingTargetExists(ctx, payload); err != nil {
		return result, err
	} else if !exists {
		result.Skipped = true
		result.Reason = "post_deleted"
		return result, nil
	}
	if _, err := os.Stat(sourcePath); err != nil {
		if exists, existsErr := s.videoTranscodingTargetExists(ctx, payload); existsErr == nil && !exists {
			result.Skipped = true
			result.Reason = "post_deleted_source_unavailable"
			return result, nil
		}
		return result, fmt.Errorf("video source unavailable: %w", err)
	}
	outputDir, dashURL, err := videoDashOutput(sourcePath, payload, s.cfg)
	if err != nil {
		return result, err
	}
	manifestPath := filepath.Join(outputDir, "manifest.mpd")
	if _, err := os.Stat(manifestPath); err != nil {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return result, err
		}
		if err := runFFmpegDash(ctx, s.cfg.Video, sourcePath, manifestPath); err != nil {
			return result, err
		}
	}
	if exists, err := s.videoTranscodingTargetExists(ctx, payload); err != nil {
		return result, err
	} else if !exists {
		s.recycleTranscodingOutputs(ctx, payload, outputDir, dashURL, "", "post_deleted_after_dash")
		result.Skipped = true
		result.Reason = "post_deleted"
		return result, nil
	}
	coverURL := strings.TrimSpace(payload.CoverURL)
	if coverURL == "" || coverURL == payload.VideoURL {
		if generated, err := generateVideoCover(ctx, s.cfg.Video, sourcePath); err == nil {
			coverURL = generated
		}
	}
	if exists, err := s.videoTranscodingTargetExists(ctx, payload); err != nil {
		return result, err
	} else if !exists {
		s.recycleTranscodingOutputs(ctx, payload, outputDir, dashURL, coverURL, "post_deleted_after_cover")
		result.Skipped = true
		result.Reason = "post_deleted"
		return result, nil
	}
	if s.db != nil {
		if err := s.updateTranscodedVideoRows(ctx, payload, dashURL, coverURL); err != nil {
			if errors.Is(err, errVideoTranscodingTargetGone) {
				s.recycleTranscodingOutputs(ctx, payload, outputDir, dashURL, coverURL, "post_deleted_before_update")
				result.Skipped = true
				result.Reason = "post_deleted"
				return result, nil
			}
			return result, err
		}
	}
	result.DashURL = dashURL
	result.CoverURL = coverURL
	if s.cfg.Video.DeleteOriginal && strings.HasPrefix(normalizeStoredVideoURL(payload.VideoURL, s.cfg), "/api/file/videos/") {
		_ = os.Remove(sourcePath)
	}
	return result, nil
}

func (s *QueueService) updateTranscodedVideoRows(ctx context.Context, payload videoTranscodingTaskPayload, dashURL string, coverURL string) error {
	updates := map[string]any{"dash_url": dashURL}
	if strings.TrimSpace(coverURL) != "" {
		updates["cover_url"] = coverURL
	}
	query := s.db.WithContext(ctx).Model(&domain.PostVideo{})
	if payload.VideoID > 0 {
		res := query.Where("id = ?", payload.VideoID).Updates(updates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			if exists, err := s.videoTranscodingTargetExists(ctx, payload); err != nil {
				return err
			} else if !exists {
				return errVideoTranscodingTargetGone
			}
		}
		return nil
	}
	videoURL := normalizeStoredVideoURL(payload.VideoURL, s.cfg)
	if videoURL == "" {
		return nil
	}
	query = query.Where("video_url = ?", videoURL)
	if payload.PostID > 0 {
		query = query.Where("post_id = ?", payload.PostID)
	}
	res := query.Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		if exists, err := s.videoTranscodingTargetExists(ctx, payload); err != nil {
			return err
		} else if !exists {
			return errVideoTranscodingTargetGone
		}
	}
	return nil
}

func (s *QueueService) videoTranscodingTargetExists(ctx context.Context, payload videoTranscodingTaskPayload) (bool, error) {
	if s == nil || s.db == nil {
		return true, nil
	}
	query := s.db.WithContext(ctx).Table("post_videos pv").Joins("JOIN posts p ON p.id = pv.post_id")
	if payload.VideoID > 0 {
		query = query.Where("pv.id = ?", payload.VideoID)
	} else {
		videoURL := normalizeStoredVideoURL(payload.VideoURL, s.cfg)
		if videoURL == "" {
			return true, nil
		}
		query = query.Where("pv.video_url = ?", videoURL)
	}
	if payload.PostID > 0 {
		query = query.Where("pv.post_id = ?", payload.PostID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *QueueService) recycleTranscodingOutputs(ctx context.Context, payload videoTranscodingTaskPayload, dashDir string, dashURL string, coverURL string, reason string) {
	if s == nil || s.db == nil || !s.cfg.Upload.Recycle.Enabled {
		return
	}
	postID := payload.PostID
	userID := payload.UserID
	inputs := []FileRecycleInput{}
	if strings.TrimSpace(dashDir) != "" {
		inputs = append(inputs, FileRecycleInput{
			ResourceType: "post",
			ResourceID:   payload.PostID,
			PostID:       &postID,
			UserID:       &userID,
			Kind:         "video_dash",
			Storage:      FileRecycleStorageLocal,
			OriginalURL:  dashURL,
			OriginalPath: dashDir,
			IsDir:        true,
			Metadata:     map[string]any{"source": "video_transcoding", "reason": reason, "video_id": payload.VideoID},
		})
	}
	if coverPath, ok := localThumbnailPathFromURL(coverURL, s.cfg); ok {
		inputs = append(inputs, FileRecycleInput{
			ResourceType: "post",
			ResourceID:   payload.PostID,
			PostID:       &postID,
			UserID:       &userID,
			Kind:         "thumbnail",
			Storage:      FileRecycleStorageLocal,
			OriginalURL:  coverURL,
			OriginalPath: coverPath,
			Metadata:     map[string]any{"source": "video_transcoding", "reason": reason, "video_id": payload.VideoID},
		})
	}
	if len(inputs) == 0 {
		return
	}
	_, _ = NewFileRecycleService(s.db, s.cfg, nil).RecycleLocal(ctx, inputs)
}

func runFFmpegDash(ctx context.Context, cfg config.VideoTranscodingConfig, sourcePath string, manifestPath string) error {
	ffmpeg, err := executablePath(cfg.FFmpegPath)
	if err != nil {
		return fmt.Errorf("ffmpeg unavailable: %w", err)
	}
	args := []string{"-y"}
	if cfg.FFmpeg.HardwareAccel && strings.TrimSpace(cfg.FFmpeg.HardwareAccelType) != "" {
		args = append(args, "-hwaccel", strings.TrimSpace(cfg.FFmpeg.HardwareAccelType))
	}
	args = append(args, "-i", sourcePath)
	resolutions := usableDashResolutions(cfg.DASH.Resolutions)
	if len(resolutions) == 0 {
		args = append(args, "-map", "0:v:0")
	} else {
		for range resolutions {
			args = append(args, "-map", "0:v:0")
		}
	}
	args = append(args, "-map", "0:a:0?", "-c:v", "libx264", "-preset", firstNonEmptyString(cfg.FFmpeg.Preset, "medium"), "-profile:v", firstNonEmptyString(cfg.FFmpeg.Profile, "main"))
	if cfg.FFmpeg.CRF != nil {
		args = append(args, "-crf", strconv.Itoa(*cfg.FFmpeg.CRF))
	}
	if cfg.FFmpeg.GOPSize != nil {
		args = append(args, "-g", strconv.Itoa(*cfg.FFmpeg.GOPSize))
	}
	if cfg.FFmpeg.BFrames != nil {
		args = append(args, "-bf", strconv.Itoa(*cfg.FFmpeg.BFrames))
	}
	if cfg.FFmpeg.RefFrames != nil {
		args = append(args, "-refs", strconv.Itoa(*cfg.FFmpeg.RefFrames))
	}
	if cfg.MaxThreads > 0 {
		args = append(args, "-threads", strconv.Itoa(cfg.MaxThreads))
	}
	if cfg.FFmpeg.PixelFormat != "" {
		args = append(args, "-pix_fmt", cfg.FFmpeg.PixelFormat)
	}
	for idx, resolution := range resolutions {
		args = append(args,
			"-filter:v:"+strconv.Itoa(idx), dashScaleFilter(resolution),
			"-b:v:"+strconv.Itoa(idx), strconv.Itoa(clampBitrate(resolution.Bitrate, cfg.DASH.MinBitrate, cfg.DASH.MaxBitrate))+"k",
			"-maxrate:v:"+strconv.Itoa(idx), strconv.Itoa(clampBitrate(resolution.Bitrate, cfg.DASH.MinBitrate, cfg.DASH.MaxBitrate))+"k",
			"-bufsize:v:"+strconv.Itoa(idx), strconv.Itoa(clampBitrate(resolution.Bitrate, cfg.DASH.MinBitrate, cfg.DASH.MaxBitrate)*2)+"k",
		)
	}
	args = append(args,
		"-c:a", "aac",
		"-b:a", strconv.Itoa(maxInt(32, cfg.FFmpeg.AudioBitrate))+"k",
		"-ar", strconv.Itoa(maxInt(8000, cfg.FFmpeg.AudioSampleRate)),
		"-f", "dash",
		"-seg_duration", strconv.Itoa(maxInt(1, cfg.DASH.SegmentDuration)),
		"-use_template", "1",
		"-use_timeline", "1",
		manifestPath,
	)
	cmd := exec.CommandContext(ctx, ffmpeg, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg dash failed: %w: %s%s", err, ffmpegPermissionHint(ffmpeg, err), outputSummary(output))
	}
	return nil
}

func generateVideoCover(ctx context.Context, cfg config.VideoTranscodingConfig, sourcePath string) (string, error) {
	ffmpeg, err := executablePath(cfg.FFmpegPath)
	if err != nil {
		return "", err
	}
	dir := serviceAbsPath("uploads/thumbnails")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	name := strconv.FormatInt(time.Now().UnixMilli(), 10) + "_" + shortHash(sourcePath) + ".jpg"
	outPath := filepath.Join(dir, name)
	cmd := exec.CommandContext(ctx, ffmpeg, "-y", "-ss", "1", "-i", sourcePath, "-frames:v", "1", "-q:v", "2", outPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg cover failed: %w: %s%s", err, ffmpegPermissionHint(ffmpeg, err), outputSummary(output))
	}
	return "/api/file/thumbnails/" + name, nil
}

func videoDashOutput(sourcePath string, payload videoTranscodingTaskPayload, cfg config.Config) (string, string, error) {
	infoTime := time.Now()
	if info, err := os.Stat(sourcePath); err == nil && !info.ModTime().IsZero() {
		infoTime = info.ModTime()
	}
	timestamp := uploadTimestamp(sourcePath, infoTime)
	format := strings.TrimSpace(cfg.Video.OutputFormat)
	if format == "" {
		format = "{date}/{userId}/{timestamp}"
	}
	replacements := map[string]string{
		"{date}":       infoTime.Format("2006-01-02"),
		"{userId}":     strconv.FormatInt(payload.UserID, 10),
		"{postId}":     strconv.FormatInt(payload.PostID, 10),
		"{videoId}":    strconv.FormatInt(payload.VideoID, 10),
		"{timestamp}":  timestamp,
		"{sourceHash}": shortHash(sourcePath),
	}
	rel := format
	for key, value := range replacements {
		if value == "0" && (key == "{userId}" || key == "{postId}" || key == "{videoId}") {
			value = "unknown"
		}
		rel = strings.ReplaceAll(rel, key, value)
	}
	rel = safeRelativePath(rel)
	if rel == "" {
		rel = infoTime.Format("2006-01-02") + "/unknown/" + timestamp
	}
	outputDir := filepath.Join(serviceAbsPath(cfg.Upload.Video.LocalUploadDir), "dash", filepath.FromSlash(rel))
	dashURL := "/api/file/videos/dash/" + rel + "/manifest.mpd"
	return outputDir, dashURL, nil
}

func localVideoPathFromURL(raw string, cfg config.Config) (string, bool) {
	raw = normalizeStoredVideoURL(raw, cfg)
	prefix := "/api/file/videos/"
	if !strings.HasPrefix(raw, prefix) {
		return "", false
	}
	rel := strings.TrimPrefix(raw, prefix)
	if safeRelativePath(rel) != rel {
		return "", false
	}
	path := filepath.Join(serviceAbsPath(cfg.Upload.Video.LocalUploadDir), filepath.FromSlash(rel))
	return path, true
}

func localThumbnailPathFromURL(raw string, cfg config.Config) (string, bool) {
	raw = normalizeStoredFileURL(raw, cfg)
	prefix := "/api/file/thumbnails/"
	if !strings.HasPrefix(raw, prefix) {
		return "", false
	}
	rel := strings.TrimPrefix(raw, prefix)
	if safeRelativePath(rel) != rel {
		return "", false
	}
	return filepath.Join(serviceAbsPath("uploads/thumbnails"), filepath.FromSlash(rel)), true
}

func canonicalVideoURLFromPath(raw string, cfg config.Config) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	path := serviceAbsPath(raw)
	root := serviceAbsPath(cfg.Upload.Video.LocalUploadDir)
	rel, err := filepath.Rel(root, path)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return ""
	}
	return "/api/file/videos/" + filepath.ToSlash(rel)
}

func normalizeStoredVideoURL(raw string, cfg config.Config) string {
	return normalizeStoredFileURL(raw, cfg)
}

func normalizeStoredFileURL(raw string, cfg config.Config) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	localBase := strings.TrimRight(cfg.Upload.LocalBase, "/")
	if localBase != "" && strings.HasPrefix(raw, localBase+"/api/file/") {
		raw = strings.TrimPrefix(raw, localBase)
	}
	return raw
}

func serviceAbsPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}

func executablePath(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "ffmpeg"
	}
	if strings.ContainsAny(value, `/\`) {
		info, err := os.Stat(value)
		if err != nil {
			return "", err
		}
		if info.IsDir() {
			return "", fmt.Errorf("%s is a directory", value)
		}
		if runtime.GOOS != "windows" && info.Mode()&0o111 == 0 {
			return "", fmt.Errorf("%s has no executable bit (mode %s)", value, info.Mode().Perm())
		}
		return value, nil
	}
	return exec.LookPath(value)
}

func videoTranscodingTaskID(payload videoTranscodingTaskPayload) string {
	if payload.VideoID > 0 {
		return "video-transcoding:video:" + strconv.FormatInt(payload.VideoID, 10)
	}
	key := payload.VideoURL
	if key == "" {
		key = payload.SourcePath
	}
	return "video-transcoding:source:" + shortHash(key)
}

func usableDashResolutions(values []config.DASHResolution) []config.DASHResolution {
	out := make([]config.DASHResolution, 0, len(values))
	for _, value := range values {
		if value.Height <= 0 || value.Bitrate <= 0 {
			continue
		}
		out = append(out, value)
	}
	return out
}

func dashScaleFilter(value config.DASHResolution) string {
	width := value.Width
	if width <= 0 {
		width = -2
	}
	return fmt.Sprintf("scale=w=%d:h=%d:force_original_aspect_ratio=decrease:force_divisible_by=2", width, value.Height)
}

func clampBitrate(value, minValue, maxValue int) int {
	if minValue > 0 && value < minValue {
		value = minValue
	}
	if maxValue > 0 && value > maxValue {
		value = maxValue
	}
	if value <= 0 {
		return 1200
	}
	return value
}

func uploadTimestamp(path string, fallback time.Time) string {
	base := filepath.Base(path)
	if idx := strings.IndexByte(base, '_'); idx > 0 {
		if _, err := strconv.ParseInt(base[:idx], 10, 64); err == nil {
			return base[:idx]
		}
	}
	return strconv.FormatInt(fallback.UnixMilli(), 10)
}

func safeRelativePath(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "\\", "/")
	parts := strings.Split(value, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			continue
		}
		out = append(out, part)
	}
	return strings.Join(out, "/")
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:16]
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
