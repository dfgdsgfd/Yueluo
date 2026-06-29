package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) adminMissingCoverStats(c *gin.Context) {
	var videos []domain.PostVideo
	_ = h.DB.WithContext(c.Request.Context()).Where("cover_url IS NULL OR cover_url = ''").Find(&videos).Error
	accessible := 0
	remote := 0
	for _, video := range videos {
		if path, ok := h.videoURLToLocalPath(video.VideoURL); ok {
			if _, err := os.Stat(path); err == nil {
				accessible++
			}
		} else if strings.HasPrefix(strings.ToLower(video.VideoURL), "http://") || strings.HasPrefix(strings.ToLower(video.VideoURL), "https://") {
			remote++
		}
	}
	writeSuccess(c, matrixMsgOK, gin.H{"total": len(videos), "accessible": accessible, "remote": remote})
}

func (h NativeHandlers) adminGenerateMissingCovers(c *gin.Context) {
	body := readBodyMap(c)
	limit, _ := intFromAny(body["limit"])
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var videos []domain.PostVideo
	if err := h.DB.WithContext(c.Request.Context()).Where("cover_url IS NULL OR cover_url = ''").Limit(limit).Find(&videos).Error; writeDBError(c, err, "") {
		return
	}
	if len(videos) == 0 {
		processed := 0
		writeSuccess(c, "没有需要处理的视频", gin.H{"processed": processed, "succeeded": 0, "failed": 0, "skipped": 0, "details": []gin.H{}})
		return
	}
	succeeded, failed, skipped := 0, 0, 0
	details := []gin.H{}
	for _, video := range videos {
		source := video.VideoURL
		if path, ok := h.videoURLToLocalPath(video.VideoURL); ok {
			if _, err := os.Stat(path); err == nil {
				source = path
			}
		}
		if source == "" {
			skipped++
			details = append(details, gin.H{"post_id": video.PostID, "video_id": video.ID, "status": "skipped", "reason": "视频文件无法访问"})
			continue
		}
		coverURL, err := h.generateVideoThumbnail(c.Request.Context(), source)
		if err != nil {
			failed++
			details = append(details, gin.H{"post_id": video.PostID, "video_id": video.ID, "status": "failed", "reason": err.Error()})
			continue
		}
		if err := h.DB.WithContext(c.Request.Context()).Model(&domain.PostVideo{}).Where("id = ?", video.ID).Update("cover_url", coverURL).Error; err != nil {
			failed++
			details = append(details, gin.H{"post_id": video.PostID, "video_id": video.ID, "status": "failed", "reason": err.Error()})
			continue
		}
		succeeded++
		details = append(details, gin.H{"post_id": video.PostID, "video_id": video.ID, "status": "success", "cover_url": h.signFileURL(coverURL)})
	}
	if succeeded > 0 {
		h.bumpAdminResourceCacheVersions("post_videos")
	}
	writeSuccess(c, fmt.Sprintf("处理完成：成功 %d 个，失败 %d 个，跳过 %d 个", succeeded, failed, skipped), gin.H{"processed": len(videos), "succeeded": succeeded, "failed": failed, "skipped": skipped, "details": details})
}

func (h NativeHandlers) generateVideoThumbnail(ctx context.Context, source string) (string, error) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return "", errors.New("ffmpeg 未安装，无法生成视频封面")
	}
	dir := absPath("uploads/thumbnails")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	name := strconv.FormatInt(time.Now().UnixMilli(), 10) + "_" + randomHex(6) + ".jpg"
	outPath := filepath.Join(dir, name)
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-ss", "1", "-i", source, "-frames:v", "1", "-q:v", "2", outPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("封面生成失败: %s", strings.TrimSpace(string(output)))
	}
	return "/api/file/thumbnails/" + name, nil
}

func (h NativeHandlers) videoURLToLocalPath(raw string) (string, bool) {
	if raw == "" {
		return "", false
	}
	path := raw
	if parsed, err := url.Parse(raw); err == nil && parsed.Path != "" {
		path = parsed.Path
	}
	path = strings.TrimPrefix(path, h.Config.Upload.LocalBase)
	path = strings.TrimPrefix(path, "/api/file/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 {
		return "", false
	}
	return h.findUploadFile(parts[0], parts[1])
}

func (h NativeHandlers) batchUploadDir() string {
	roots := h.uploadRoots()
	if len(roots) == 0 {
		return absPath("uploads/plsc")
	}
	return filepath.Join(roots[0], "plsc")
}

type batchFile struct {
	Name string
	Path string
}

func batchFilesFromAny(value any) []batchFile {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]batchFile, 0, len(values))
	for _, item := range values {
		switch typed := item.(type) {
		case string:
			out = append(out, batchFile{Name: filepath.Base(typed), Path: normalizeBatchPath(typed)})
		case map[string]any:
			path := normalizeBatchPath(firstNonEmpty(toString(typed["path"]), toString(typed["url"]), toString(typed["name"])))
			out = append(out, batchFile{Name: firstNonEmpty(toString(typed["name"]), filepath.Base(path)), Path: path})
		}
	}
	return out
}

func normalizeBatchPath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		if parsed, err := url.Parse(value); err == nil {
			value = parsed.Path
		}
	}
	if strings.HasPrefix(value, "/api/file/") {
		return value
	}
	return "/api/file/plsc/" + filepath.Base(value)
}

func tagsFromAny(value any) []string {
	switch typed := value.(type) {
	case []any:
		out := []string{}
		for _, item := range typed {
			if m, ok := item.(map[string]any); ok {
				if name := toString(m["name"]); name != "" {
					out = append(out, name)
				}
			} else if name := toString(item); name != "" {
				out = append(out, name)
			}
		}
		return out
	default:
		return parseStringSlice(value)
	}
}

func notesFromAny(value any) []map[string]any {
	values, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(values))
	for _, item := range values {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func queueBatchNotesFromMaps(notes []map[string]any) []services.QueueBatchNote {
	out := make([]services.QueueBatchNote, 0, len(notes))
	for _, note := range notes {
		files := batchFilesFromAny(note["files"])
		queueFiles := make([]services.QueueBatchFile, 0, len(files))
		for _, file := range files {
			queueFiles = append(queueFiles, services.QueueBatchFile{Name: file.Name, Path: file.Path})
		}
		out = append(out, services.QueueBatchNote{
			Title:    toString(note["title"]),
			Content:  toString(note["content"]),
			Files:    queueFiles,
			CoverURL: firstNonEmpty(toString(note["coverUrl"]), toString(note["cover_url"])),
		})
	}
	return out
}

func knownQueueName(name string) bool {
	return slices.Contains(services.QueueNames, name)
}

func adminDurationQuery(raw string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		if fallback > 0 {
			return fallback
		}
		return 0
	}
	if parsed, err := time.ParseDuration(value); err == nil {
		return parsed
	}
	if before, ok := strings.CutSuffix(value, "d"); ok {
		days, err := strconv.Atoi(before)
		if err == nil && days > 0 {
			return time.Duration(days) * 24 * time.Hour
		}
	}
	if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
		return time.Duration(parsed) * time.Second
	}
	if fallback > 0 {
		return fallback
	}
	return 0
}

func anyTime(value any) time.Time {
	if typed, ok := value.(time.Time); ok {
		return typed
	}
	return time.Time{}
}

func sortFilesByCreated(files []gin.H) {
	sort.SliceStable(files, func(i, j int) bool {
		return anyTime(files[i]["createdAt"]).After(anyTime(files[j]["createdAt"]))
	})
}

func renderTemplate(template string, vars map[string]string) string {
	out := template
	for key, value := range vars {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
		out = strings.ReplaceAll(out, "{{ "+key+" }}", value)
	}
	return out
}

func notificationSampleVars(siteName string, siteURL string) map[string]string {
	return map[string]string{
		"siteName":  siteName,
		"siteUrl":   siteURL,
		"username":  "测试用户",
		"nickname":  "测试用户",
		"title":     "测试标题",
		"content":   "这是一条测试通知内容",
		"postTitle": "测试笔记",
	}
}
