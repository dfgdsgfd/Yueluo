package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
)

func (h NativeHandlers) adminMonitorActivities(c *gin.Context) {
	activities := []gin.H{}
	var users []domain.User
	_ = h.DB.WithContext(c.Request.Context()).Order("created_at DESC").Limit(10).Find(&users).Error
	for _, user := range users {
		activities = append(activities, gin.H{
			"id":         "user_" + strconv.FormatInt(user.ID, 10),
			"type":       "user_register",
			"user_id":    user.UserID,
			"nickname":   user.Nickname,
			"avatar":     h.signFileURLPtr(user.Avatar),
			"title":      "新用户注册",
			"content":    "用户 " + user.Nickname + " (" + user.UserID + ") 注册了账号",
			"target_id":  user.ID,
			"created_at": user.CreatedAt,
		})
	}
	var posts []domain.Post
	_ = h.DB.WithContext(c.Request.Context()).Where("is_draft = ?", false).Order("created_at DESC").Limit(10).Find(&posts).Error
	for _, post := range posts {
		activities = append(activities, gin.H{
			"id":         "post_" + strconv.FormatInt(post.ID, 10),
			"type":       "post_create",
			"title":      "新笔记发布",
			"content":    post.Title,
			"target_id":  post.ID,
			"created_at": post.CreatedAt,
		})
	}
	var comments []domain.Comment
	_ = h.DB.WithContext(c.Request.Context()).Order("created_at DESC").Limit(10).Find(&comments).Error
	for _, comment := range comments {
		activities = append(activities, gin.H{
			"id":         "comment_" + strconv.FormatInt(comment.ID, 10),
			"type":       "comment_create",
			"title":      "新评论",
			"content":    comment.Content,
			"target_id":  comment.PostID,
			"comment_id": comment.ID,
			"created_at": comment.CreatedAt,
		})
	}
	sort.SliceStable(activities, func(i, j int) bool {
		return anyTime(activities[i]["created_at"]).After(anyTime(activities[j]["created_at"]))
	})
	if len(activities) > 20 {
		activities = activities[:20]
	}
	writeSuccess(c, matrixMsgOK, gin.H{"activities": activities, "pagination": matrixPagination(1, 20, int64(len(activities)))})
}

func (h NativeHandlers) adminAPKFiles(c *gin.Context) {
	files := []gin.H{}
	for _, root := range h.uploadRoots() {
		dir := filepath.Join(root, "apk")
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !apkExts[strings.ToLower(filepath.Ext(entry.Name()))] {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			url := "/api/file/apk/" + entry.Name()
			files = append(files, gin.H{"name": entry.Name(), "size": info.Size(), "url": url, "signedUrl": h.signFileURL(url), "createdAt": info.ModTime()})
		}
	}
	sortFilesByCreated(files)
	writeSuccess(c, matrixMsgOK, gin.H{"files": files, "total": len(files)})
}

func (h NativeHandlers) adminBatchUpload(c *gin.Context) {
	path := c.Request.URL.Path
	switch {
	case strings.HasSuffix(path, "/files") && matrixMethod(c) == http.MethodGet:
		h.adminBatchUploadFiles(c)
	case strings.HasSuffix(path, "/files") && matrixMethod(c) == http.MethodDelete:
		h.adminBatchDeleteFiles(c)
	case strings.Contains(path, "/status/"):
		h.adminBatchStatus(c)
	case strings.HasSuffix(path, "/create") || strings.HasSuffix(path, "/async-create"):
		h.adminBatchCreate(c, strings.HasSuffix(path, "/async-create"))
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "批量上传路由不存在", nil)
	}
}

func (h NativeHandlers) adminBatchUploadFiles(c *gin.Context) {
	dir := h.batchUploadDir()
	_ = os.MkdirAll(dir, 0755)
	entries, _ := os.ReadDir(dir)
	images := []gin.H{}
	videos := []gin.H{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		item := gin.H{"name": entry.Name(), "size": info.Size(), "path": "/api/file/plsc/" + entry.Name(), "createdAt": info.ModTime()}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if batchImageExts[ext] {
			images = append(images, item)
		} else if batchVideoExts[ext] {
			videos = append(videos, item)
		}
	}
	sortFilesByCreated(images)
	sortFilesByCreated(videos)
	writeSuccess(c, "获取成功", gin.H{"images": images, "videos": videos})
}

func (h NativeHandlers) adminBatchDeleteFiles(c *gin.Context) {
	files := batchFilesFromAny(readBodyMap(c)["files"])
	if len(files) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "没有选择文件", nil)
		return
	}
	dir := h.batchUploadDir()
	deleted := 0
	for _, file := range files {
		name := filepath.Base(firstNonEmpty(file.Path, file.Name))
		if name == "." || name == string(filepath.Separator) || invalidSingleFilename(name) {
			continue
		}
		if err := os.Remove(filepath.Join(dir, name)); err == nil {
			deleted++
		}
	}
	writeSuccess(c, "删除成功", gin.H{"deletedCount": deleted})
}

func (h NativeHandlers) adminBatchStatus(c *gin.Context) {
	parts := strings.Split(c.Request.URL.Path, "/")
	batchID := parts[len(parts)-1]
	if h.Cache != nil {
		if value, ok := h.Cache.Get("batch-upload:" + batchID); ok {
			writeSuccess(c, matrixMsgOK, value)
			return
		}
	}
	if h.Queue != nil {
		enabled, status, err := h.Queue.BatchStatus(batchID)
		if err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
			return
		}
		if enabled {
			if total, ok := intFromAny(status["total"]); ok && total > 0 {
				writeSuccess(c, matrixMsgOK, status)
				return
			}
		}
	}
	enabled := h.queueAvailable(c.Request.Context())
	writeSuccess(c, matrixMsgOK, gin.H{"enabled": enabled, "batchId": batchID, "message": "批次状态不存在或已过期"})
}

func (h NativeHandlers) adminBatchCreate(c *gin.Context, async bool) {
	body := readBodyMap(c)
	userID, ok := int64FromAny(body["user_id"])
	if !ok || userID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少用户ID", nil)
		return
	}
	var user domain.User
	if err := h.DB.WithContext(c.Request.Context()).Where("id = ?", userID).First(&user).Error; writeDBError(c, err, "用户不存在") {
		return
	}
	postType, _ := intFromAny(body["type"])
	if postType == 0 {
		postType = 1
	}
	isDraft, _ := boolFromAny(body["is_draft"])
	tags := tagsFromAny(body["tags"])
	var created []gin.H
	var failed []gin.H
	if async {
		notes := notesFromAny(body["notes"])
		if len(notes) == 0 {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "没有笔记数据", nil)
			return
		}
		for _, note := range notes {
			if h.rejectPostContentOverLimit(c, sanitizePostContent(toString(note["content"]))) {
				return
			}
		}
		if h.queueAvailable(c.Request.Context()) {
			batchID, jobs, err := h.Queue.EnqueueBatchNoteCreate(c.Request.Context(), queueBatchNotesFromMaps(notes), userID, postType, tags, isDraft)
			if err != nil {
				response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
				return
			}
			data := gin.H{"batchId": batchID, "jobs": jobs, "count": len(jobs), "queueEnabled": true, "async": true}
			if h.Cache != nil {
				h.Cache.Set("batch-upload:"+batchID, data, 24*time.Hour)
			}
			writeSuccess(c, "已加入队列", data)
			return
		}
		for idx, note := range notes {
			post, err := h.createBatchPost(c.Request.Context(), userID, postType, toString(note["title"]), toString(note["content"]), isDraft, batchFilesFromAny(note["files"]), tags, toString(note["coverUrl"]))
			if err != nil {
				failed = append(failed, gin.H{"noteIndex": idx, "error": err.Error()})
				continue
			}
			created = append(created, gin.H{"id": post.ID, "noteIndex": idx})
		}
	} else {
		files := batchFilesFromAny(body["files"])
		if len(files) == 0 {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "没有选择文件", nil)
			return
		}
		imagesPerNote, _ := intFromAny(body["images_per_note"])
		if imagesPerNote <= 0 {
			imagesPerNote = 4
		}
		imagesPerNote = min(imagesPerNote, h.maxPostImages())
		title := toString(body["title"])
		content := sanitizePostContent(toString(body["content"]))
		if h.rejectPostContentOverLimit(c, content) {
			return
		}
		if postType == 1 {
			for i := 0; i < len(files); i += imagesPerNote {
				end := min(i+imagesPerNote, len(files))
				post, err := h.createBatchPost(c.Request.Context(), userID, postType, title, content, isDraft, files[i:end], tags, "")
				if err != nil {
					failed = append(failed, gin.H{"index": i, "error": err.Error()})
					continue
				}
				created = append(created, gin.H{"id": post.ID, "imageCount": end - i})
			}
		} else {
			for idx, file := range files {
				post, err := h.createBatchPost(c.Request.Context(), userID, postType, title, content, isDraft, []batchFile{file}, tags, "")
				if err != nil {
					failed = append(failed, gin.H{"index": idx, "error": err.Error()})
					continue
				}
				created = append(created, gin.H{"id": post.ID})
			}
		}
	}
	batchID := strconv.FormatInt(time.Now().UnixNano(), 36)
	data := gin.H{"posts": created, "count": len(created), "failed": failed, "async": false, "batchId": batchID}
	if h.Cache != nil {
		h.Cache.Set("batch-upload:"+batchID, data, 24*time.Hour)
	}
	if len(created) > 0 {
		h.bumpAdminResourceCacheVersions("posts")
	}
	writeSuccess(c, fmt.Sprintf("成功创建 %d 条笔记", len(created)), data)
}

func (h NativeHandlers) createBatchPost(ctx context.Context, userID int64, postType int, title string, content string, isDraft bool, files []batchFile, tags []string, coverURL string) (domain.Post, error) {
	var post domain.Post
	if postType == 1 && len(files) > h.maxPostImages() {
		return post, fmt.Errorf("error.post_images_limit")
	}
	content = sanitizePostContent(content)
	if postContentLength(content) > h.maxPostContentLength() {
		return post, fmt.Errorf("error.post_content_limit")
	}
	err := h.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		post = domain.Post{UserID: userID, Title: title, Content: content, Type: postType, IsDraft: isDraft, Visibility: "public", QualityLevel: "none"}
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		if postType == 1 {
			for _, file := range files {
				if err := tx.Create(&domain.PostImage{PostID: post.ID, ImageURL: h.normalizeFileURLForStorage(file.Path), IsFreePreview: true}).Error; err != nil {
					return err
				}
			}
		} else if len(files) > 0 {
			videoURL := h.normalizeFileURLForStorage(files[0].Path)
			cover := h.normalizeFileURLForStorage(coverURL)
			if cover == "" {
				cover = videoURL
			}
			if err := tx.Create(&domain.PostVideo{PostID: post.ID, VideoURL: videoURL, CoverURL: &cover}).Error; err != nil {
				return err
			}
		}
		for _, name := range tags {
			tagID, err := h.getOrCreateTag(ctx, tx, name)
			if err != nil {
				return err
			}
			if err := tx.Create(&domain.PostTag{PostID: post.ID, TagID: tagID}).Error; err != nil {
				return err
			}
			_ = tx.Model(&domain.Tag{}).Where("id = ?", tagID).Update("use_count", gorm.Expr("use_count + 1")).Error
		}
		return nil
	})
	if err == nil && post.Type != 1 {
		_ = h.enqueueVideoTranscodingForPost(ctx, post.ID)
	}
	if err == nil && !post.IsDraft {
		h.schedulePostImageArchive(post.ID)
		_ = h.enqueueAIPostAutoComment(ctx, post.ID)
	}
	return post, err
}

func (h NativeHandlers) getOrCreateTag(ctx context.Context, tx *gorm.DB, name string) (int, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, nil
	}
	var tag domain.Tag
	err := tx.WithContext(ctx).Where("name = ?", name).First(&tag).Error
	if err == nil {
		return tag.ID, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}
	tag = domain.Tag{Name: name}
	if err := tx.WithContext(ctx).Create(&tag).Error; err != nil {
		return 0, err
	}
	return tag.ID, nil
}
