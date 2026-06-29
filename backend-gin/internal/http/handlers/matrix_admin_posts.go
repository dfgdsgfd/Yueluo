package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) adminPostsVisibility(c *gin.Context, visibility string) {
	body := readBodyMap(c)
	ids, selectedAll, ok := h.adminPostIDsFromBody(c, body, "请提供要设置的笔记ID列表")
	if !ok {
		return
	}
	if len(ids) == 0 && selectedAll {
		writeSimpleSuccess(c, "没有匹配的笔记")
		return
	}
	res := h.DB.WithContext(c.Request.Context()).Model(&domain.Post{}).Where("id IN ?", ids).Update("visibility", visibility)
	if writeDBError(c, res.Error, "") {
		return
	}
	h.bumpAdminResourceCacheVersions("posts")
	writeSimpleSuccess(c, "成功更新 "+strconv.FormatInt(res.RowsAffected, 10)+" 篇帖子")
}

func (h NativeHandlers) adminPostsSetCategory(c *gin.Context) {
	body := readBodyMap(c)
	var categoryValue any
	if raw, exists := body["category_id"]; exists && raw != nil && toString(raw) != "" {
		categoryID, ok := intFromAny(raw)
		if !ok || categoryID <= 0 {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请提供有效的分类ID", nil)
			return
		}
		var count int64
		if err := h.DB.WithContext(c.Request.Context()).Model(&domain.Category{}).Where("id = ?", categoryID).Count(&count).Error; writeDBError(c, err, "") {
			return
		}
		if count == 0 {
			response.JSON(c, http.StatusNotFound, response.CodeNotFound, "分类不存在", nil)
			return
		}
		categoryValue = categoryID
	} else {
		categoryValue = nil
	}
	ids, selectedAll, ok := h.adminPostIDsFromBody(c, body, "请提供要设置的笔记ID列表")
	if !ok {
		return
	}
	if len(ids) == 0 && selectedAll {
		writeSimpleSuccess(c, "没有匹配的笔记")
		return
	}
	res := h.DB.WithContext(c.Request.Context()).Model(&domain.Post{}).Where("id IN ?", ids).Update("category_id", categoryValue)
	if writeDBError(c, res.Error, "") {
		return
	}
	h.bumpAdminResourceCacheVersions("posts")
	writeSimpleSuccess(c, "成功更新 "+strconv.FormatInt(res.RowsAffected, 10)+" 篇帖子")
}

func (h NativeHandlers) adminPostsTransfer(c *gin.Context) {
	body := readBodyMap(c)
	targetUserID, targetUserLabel, ok := h.adminTargetUserID(c, body)
	if !ok {
		return
	}
	ids, selectedAll, ok := h.adminPostIDsFromBody(c, body, "请提供要转移的笔记ID列表")
	if !ok {
		return
	}
	if len(ids) == 0 && selectedAll {
		writeSimpleSuccess(c, "没有匹配的笔记")
		return
	}
	res := h.DB.WithContext(c.Request.Context()).Model(&domain.Post{}).Where("id IN ?", ids).Update("user_id", targetUserID)
	if writeDBError(c, res.Error, "") {
		return
	}
	h.bumpAdminResourceCacheVersions("posts")
	writeSimpleSuccess(c, "成功将 "+strconv.FormatInt(res.RowsAffected, 10)+" 篇笔记转移给用户 "+targetUserLabel)
}

func (h NativeHandlers) adminPostCreateCompat(c *gin.Context) {
	body := readBodyMap(c)
	userID, ok := int64FromAny(body["user_id"])
	if !ok || userID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少用户ID", nil)
		return
	}
	if !h.adminUserExists(c, userID) {
		return
	}
	if imageCount := len(h.adminPostImageURLs(body)); imageCount > h.maxPostImages() {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.post_images_limit", gin.H{"maxImages": h.maxPostImages()})
		return
	}

	postType := 1
	if value, ok := intFromAny(body["type"]); ok {
		postType = value
	}
	isDraft := true
	if raw, exists := body["is_draft"]; exists {
		isDraft = jsBoolean(raw)
	}
	post := domain.Post{
		UserID:       userID,
		Title:        toString(body["title"]),
		Content:      sanitizePostContent(toString(body["content"])),
		CategoryID:   adminNullableCategoryID(body["category_id"]),
		Type:         postType,
		IsDraft:      isDraft,
		Visibility:   "public",
		QualityLevel: "none",
	}
	if h.rejectPostContentOverLimit(c, post.Content) {
		return
	}
	if visibility := toString(body["visibility"]); adminVisibilityValid(visibility) {
		post.Visibility = visibility
	}
	if raw, exists := body["public_access_exempt"]; exists {
		post.PublicAccessExempt = jsBoolean(raw)
	}
	if err := h.DB.WithContext(c.Request.Context()).Create(&post).Error; writeDBError(c, err, "") {
		return
	}
	if _, hasImages := body["images"]; hasImages {
		if err := h.adminReplacePostImages(c, post.ID, body); writeDBError(c, err, "") {
			return
		}
	} else if _, hasImageURLs := body["image_urls"]; hasImageURLs {
		if err := h.adminReplacePostImages(c, post.ID, body); writeDBError(c, err, "") {
			return
		}
	}
	if rawTags, hasTags := body["tags"]; hasTags {
		if err := h.adminReplacePostTags(c, post.ID, rawTags, false); writeDBError(c, err, "") {
			return
		}
	}
	if rawURL := strings.TrimSpace(toString(body["video_url"])); rawURL != "" {
		if err := h.adminReplacePostVideo(c, post.ID, body); writeDBError(c, err, "") {
			return
		}
	}
	data := gin.H{"id": post.ID}
	if !post.IsDraft {
		h.schedulePostImageArchive(post.ID)
		if job := h.enqueueAIPostAutoComment(c.Request.Context(), post.ID, post.UserID); job != nil {
			data["aiAutoCommentJob"] = job
		}
	}
	h.bumpAdminResourceCacheVersions("posts")
	writeSuccess(c, "笔记创建成功", data)
}

func (h NativeHandlers) adminPostUpdateCompat(c *gin.Context) {
	postID, ok := int64FromAny(matrixParam(c, "id"))
	if !ok || postID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效的帖子ID", nil)
		return
	}
	var existing domain.Post
	if err := h.DB.WithContext(c.Request.Context()).Where("id = ?", postID).Take(&existing).Error; writeDBError(c, err, "帖子不存在") {
		return
	}
	body := readBodyMap(c)
	if imageCount := len(h.adminPostImageURLs(body)); imageCount > h.maxPostImages() {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.post_images_limit", gin.H{"maxImages": h.maxPostImages()})
		return
	}
	updates := map[string]any{}
	if raw, exists := body["title"]; exists {
		updates["title"] = toString(raw)
	}
	if raw, exists := body["content"]; exists {
		content := sanitizePostContent(toString(raw))
		if h.rejectPostContentOverLimit(c, content) {
			return
		}
		updates["content"] = content
	}
	if raw, exists := body["category_id"]; exists {
		updates["category_id"] = adminNullableCategoryID(raw)
	}
	if raw, exists := body["view_count"]; exists {
		viewCount, _ := int64FromAny(raw)
		if viewCount < 0 {
			viewCount = 0
		}
		updates["view_count"] = viewCount
	}
	if raw, exists := body["is_draft"]; exists {
		updates["is_draft"] = jsBoolean(raw)
	}
	if raw, exists := body["visibility"]; exists {
		visibility := toString(raw)
		if adminVisibilityValid(visibility) {
			updates["visibility"] = visibility
		}
	}
	if raw, exists := body["public_access_exempt"]; exists {
		updates["public_access_exempt"] = jsBoolean(raw)
	}
	if len(updates) > 0 {
		if err := h.DB.WithContext(c.Request.Context()).Model(&domain.Post{}).Where("id = ?", postID).Updates(updates).Error; writeDBError(c, err, "") {
			return
		}
	}
	if _, hasImages := body["images"]; hasImages {
		if err := h.adminReplacePostImages(c, postID, body); writeDBError(c, err, "") {
			return
		}
	} else if _, hasImageURLs := body["image_urls"]; hasImageURLs {
		if err := h.adminReplacePostImages(c, postID, body); writeDBError(c, err, "") {
			return
		}
	}
	if _, hasVideoURL := body["video_url"]; hasVideoURL {
		if err := h.adminReplacePostVideo(c, postID, body); writeDBError(c, err, "") {
			return
		}
	} else if _, hasCoverURL := body["cover_url"]; hasCoverURL {
		if err := h.adminReplacePostVideo(c, postID, body); writeDBError(c, err, "") {
			return
		}
	} else if _, hasVideo := body["video"]; hasVideo {
		if err := h.adminReplacePostVideo(c, postID, body); writeDBError(c, err, "") {
			return
		}
	}
	if rawTags, hasTags := body["tags"]; hasTags {
		if err := h.adminReplacePostTags(c, postID, rawTags, true); writeDBError(c, err, "") {
			return
		}
	}
	h.schedulePostImageArchive(postID)
	if value, ok := updates["is_draft"].(bool); ok && !value && existing.IsDraft {
		_ = h.enqueueAIPostAutoComment(c.Request.Context(), postID)
	}
	h.bumpAdminResourceCacheVersions("posts")
	writeSimpleSuccess(c, "笔记更新成功")
}

func (h NativeHandlers) adminPostDeleteCompat(c *gin.Context) {
	postID, ok := int64FromAny(matrixParam(c, "id"))
	if !ok || postID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效的帖子ID", nil)
		return
	}
	var post domain.Post
	if err := h.DB.WithContext(c.Request.Context()).Where("id = ?", postID).Take(&post).Error; writeDBError(c, err, "帖子不存在") {
		return
	}
	files := h.postLocalFileRecycleInputs(c.Request.Context(), []int64{postID})
	if err := h.adminDecrementPostTags(c, []int64{postID}); writeDBError(c, err, "") {
		return
	}
	err := h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("post_id = ?", postID).Delete(&domain.PostImage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&domain.PostVideo{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&domain.PostAttachment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&domain.PostPaymentSetting{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&domain.PostTag{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&domain.ImageWatermarkTrace{}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", postID).Delete(&domain.Post{}).Error
	})
	if writeDBError(c, err, "") {
		return
	}
	h.expirePostImageArchiveJobs(c.Request.Context(), postID, "post_deleted")
	fileDeletion := h.recycleLocalPostFiles(c.Request.Context(), files)
	h.bumpAdminResourceCacheVersions("posts")
	h.recordSystemFileDeletionAudit(c, "post_delete", []int64{postID}, fileDeletion, map[string]any{"source": "admin_post_delete"})
	writeSimpleSuccess(c, "删除成功")
}

func (h NativeHandlers) adminPostsBulkDeleteCompat(c *gin.Context) {
	body := readBodyMap(c)
	ids, selectedAll, ok := h.adminPostIDsFromBody(c, body, "请提供要删除的ID列表")
	if !ok {
		return
	}
	if len(ids) == 0 && selectedAll {
		writeSimpleSuccess(c, "没有匹配的笔记")
		return
	}
	if err := h.adminDecrementPostTags(c, ids); writeDBError(c, err, "") {
		return
	}
	files := h.postLocalFileRecycleInputs(c.Request.Context(), ids)
	err := h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("post_id IN ?", ids).Delete(&domain.PostImage{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id IN ?", ids).Delete(&domain.PostVideo{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id IN ?", ids).Delete(&domain.PostAttachment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id IN ?", ids).Delete(&domain.PostPaymentSetting{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id IN ?", ids).Delete(&domain.PostTag{}).Error; err != nil {
			return err
		}
		if err := tx.Where("post_id IN ?", ids).Delete(&domain.ImageWatermarkTrace{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Delete(&domain.Post{}).Error
	})
	if writeDBError(c, err, "") {
		return
	}
	for _, postID := range ids {
		h.expirePostImageArchiveJobs(c.Request.Context(), postID, "post_deleted")
	}
	fileDeletion := h.recycleLocalPostFiles(c.Request.Context(), files)
	h.bumpAdminResourceCacheVersions("posts")
	h.recordSystemFileDeletionAudit(c, "post_delete", ids, fileDeletion, map[string]any{"source": "admin_posts_bulk_delete"})
	writeSimpleSuccess(c, "成功删除 "+strconv.Itoa(len(ids))+" 条记录")
}

func (h NativeHandlers) adminPostIDsFromBody(c *gin.Context, body map[string]any, emptyIDsMessage string) ([]int64, bool, bool) {
	if jsBoolean(body["selectAll"]) {
		query := h.DB.WithContext(c.Request.Context()).Table("posts p").Select("p.id")
		params := mapFromAny(body["searchParams"])
		if title := strings.TrimSpace(toString(params["title"])); title != "" {
			query = query.Where("p.title LIKE ?", "%"+title+"%")
		}
		if postType, ok := intFromAny(params["type"]); ok {
			query = query.Where("p.type = ?", postType)
		}
		if raw, exists := params["is_draft"]; exists && toString(raw) != "" {
			query = query.Where("p.is_draft = ?", toString(raw) == "1" || strings.EqualFold(toString(raw), "true"))
		}
		if raw, exists := params["public_access_exempt"]; exists && toString(raw) != "" {
			query = query.Where("p.public_access_exempt = ?", toString(raw) == "1" || strings.EqualFold(toString(raw), "true"))
		}
		if displayID := strings.TrimSpace(toString(params["user_display_id"])); displayID != "" {
			query = query.Joins("LEFT JOIN users u ON u.id = p.user_id").Where("u.user_id LIKE ?", "%"+displayID+"%")
		}
		var ids []int64
		if err := query.Pluck("p.id", &ids).Error; writeDBError(c, err, "") {
			return nil, true, false
		}
		return ids, true, true
	}
	ids := int64SliceFromAny(firstPresent(body, "ids", "post_ids"))
	if len(ids) == 0 {
		if emptyIDsMessage == "" {
			emptyIDsMessage = "请提供要操作的帖子ID列表"
		}
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, emptyIDsMessage, nil)
		return nil, false, false
	}
	return ids, false, true
}

func (h NativeHandlers) adminTargetUserID(c *gin.Context, body map[string]any) (int64, string, bool) {
	displayID := strings.TrimSpace(toString(body["target_user_display_id"]))
	if displayID == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请提供目标用户汐社号", nil)
		return 0, "", false
	}

	var user domain.User
	err := h.DB.WithContext(c.Request.Context()).Where("user_id = ? AND is_active = ?", displayID, true).Take(&user).Error
	if writeDBError(c, err, "未找到汐社号为 "+displayID+" 的用户，或该用户已被停用") {
		return 0, "", false
	}
	label := strings.TrimSpace(user.Nickname)
	if label == "" {
		label = user.UserID
	}
	return user.ID, label, true
}

func (h NativeHandlers) adminReplacePostImages(c *gin.Context, postID int64, body map[string]any) error {
	urls := h.adminPostImageURLs(body)
	inputs := make([]repositories.PostImageInput, 0, len(urls))
	for _, url := range urls {
		inputs = append(inputs, repositories.PostImageInput{URL: url})
	}
	return h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		if err := repositories.PreparePostImageWatermarkTraceRebind(c.Request.Context(), tx, postID, inputs); err != nil {
			return err
		}
		if err := tx.Where("post_id = ?", postID).Delete(&domain.PostImage{}).Error; err != nil {
			return err
		}
		rows := make([]domain.PostImage, 0, len(urls))
		for _, url := range urls {
			row := domain.PostImage{PostID: postID, ImageURL: url}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			rows = append(rows, row)
		}
		return repositories.BindPostImageWatermarkTraces(c.Request.Context(), tx, postID, 0, rows)
	})
}

func (h NativeHandlers) adminReplacePostVideo(c *gin.Context, postID int64, body map[string]any) error {
	if err := h.DB.WithContext(c.Request.Context()).Where("post_id = ?", postID).Delete(&domain.PostVideo{}).Error; err != nil {
		return err
	}
	videoURL := ""
	coverURL := ""
	if video := mapFromAny(body["video"]); len(video) > 0 {
		videoURL = h.normalizeFileURLForStorage(toString(video["url"]))
		coverURL = h.normalizeFileURLForStorage(toString(firstPresent(video, "coverUrl", "cover_url")))
	} else {
		videoURL = h.normalizeFileURLForStorage(toString(body["video_url"]))
		coverURL = h.normalizeFileURLForStorage(toString(body["cover_url"]))
	}
	if videoURL == "" {
		return nil
	}
	video := domain.PostVideo{PostID: postID, VideoURL: videoURL, CoverURL: &coverURL}
	if err := h.DB.WithContext(c.Request.Context()).Create(&video).Error; err != nil {
		return err
	}
	_, _ = h.enqueueVideoTranscoding(c.Request.Context(), services.VideoTranscodingInput{
		VideoID:  video.ID,
		PostID:   postID,
		VideoURL: videoURL,
		CoverURL: coverURL,
	})
	return nil
}

func (h NativeHandlers) adminReplacePostTags(c *gin.Context, postID int64, raw any, decrementOld bool) error {
	if decrementOld {
		if err := h.adminDecrementPostTags(c, []int64{postID}); err != nil {
			return err
		}
	}
	if err := h.DB.WithContext(c.Request.Context()).Where("post_id = ?", postID).Delete(&domain.PostTag{}).Error; err != nil {
		return err
	}
	for _, rawTag := range anySlice(raw) {
		tagID, ok, err := h.adminTagIDFromAny(c, rawTag)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		if err := h.DB.WithContext(c.Request.Context()).Create(&domain.PostTag{PostID: postID, TagID: tagID}).Error; err != nil {
			return err
		}
		if err := h.DB.WithContext(c.Request.Context()).Model(&domain.Tag{}).Where("id = ?", tagID).Update("use_count", gorm.Expr("use_count + ?", 1)).Error; err != nil {
			return err
		}
	}
	return nil
}

func (h NativeHandlers) adminTagIDFromAny(c *gin.Context, raw any) (int, bool, error) {
	tagMap := mapFromAny(raw)
	if len(tagMap) > 0 {
		idText := toString(tagMap["id"])
		if !jsBoolean(tagMap["is_new"]) && !strings.HasPrefix(idText, "temp_") {
			if tagID, ok := intFromAny(tagMap["id"]); ok && tagID > 0 {
				return tagID, true, nil
			}
		}
		name := strings.TrimSpace(toString(tagMap["name"]))
		if name == "" {
			return 0, false, nil
		}
		tag := domain.Tag{Name: name}
		err := h.DB.WithContext(c.Request.Context()).Where("name = ?", name).FirstOrCreate(&tag, domain.Tag{Name: name}).Error
		return tag.ID, err == nil, err
	}
	if name := strings.TrimSpace(toString(raw)); name != "" {
		tag := domain.Tag{Name: name}
		err := h.DB.WithContext(c.Request.Context()).Where("name = ?", name).FirstOrCreate(&tag, domain.Tag{Name: name}).Error
		return tag.ID, err == nil, err
	}
	return 0, false, nil
}

func (h NativeHandlers) adminDecrementPostTags(c *gin.Context, postIDs []int64) error {
	if len(postIDs) == 0 {
		return nil
	}
	var rows []domain.PostTag
	if err := h.DB.WithContext(c.Request.Context()).Where("post_id IN ?", postIDs).Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if err := h.DB.WithContext(c.Request.Context()).Model(&domain.Tag{}).Where("id = ?", row.TagID).Update("use_count", gorm.Expr("use_count - ?", 1)).Error; err != nil {
			return err
		}
	}
	return nil
}

func (h NativeHandlers) adminPostImageURLs(body map[string]any) []string {
	urls := []string{}
	seen := map[string]bool{}
	add := func(raw any) {
		url := h.normalizeFileURLForStorage(cleanAdminMediaURL(toString(raw)))
		if url != "" && !seen[url] {
			seen[url] = true
			urls = append(urls, url)
		}
	}
	for _, raw := range anySlice(body["image_urls"]) {
		add(raw)
	}
	for _, raw := range anySlice(body["images"]) {
		if image := mapFromAny(raw); len(image) > 0 {
			for _, key := range []string{"url", "preview", "src", "path", "link"} {
				if value := toString(image[key]); value != "" {
					add(value)
					break
				}
			}
			continue
		}
		add(raw)
	}
	return urls
}

func cleanAdminMediaURL(value string) string {
	value = strings.ReplaceAll(strings.TrimSpace(value), "`", "")
	return strings.Join(strings.Fields(value), "")
}

func adminNullableCategoryID(raw any) *int {
	if raw == nil || toString(raw) == "" {
		return nil
	}
	if id, ok := intFromAny(raw); ok {
		return &id
	}
	return nil
}

func adminVisibilityValid(visibility string) bool {
	switch visibility {
	case "public", "private", "friends_only":
		return true
	default:
		return false
	}
}

func mapFromAny(value any) map[string]any {
	switch typed := value.(type) {
	case map[string]any:
		return typed
	case gin.H:
		return map[string]any(typed)
	default:
		return map[string]any{}
	}
}

func anySlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	case []string:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, item)
		}
		return out
	default:
		return nil
	}
}
