package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) CreatePost(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	var raw map[string]any
	_ = c.ShouldBindJSON(&raw)
	maxImages := h.maxPostImages()
	if imageInputCount(raw["images"]) > maxImages {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.post_images_limit", gin.H{"maxImages": maxImages})
		return
	}
	input := h.createPostInputFromMap(raw, user.ID)
	if input.Type == repositories.PostTypeImage {
		input.Images = repositories.NormalizePostImageInputs(input.Images)
		input.PaymentSettings = normalizeImagePaymentSettings(input.Images, input.PaymentSettings)
	}
	input.Content = sanitizePostContent(input.Content)
	if h.rejectPostContentOverLimit(c, input.Content) {
		return
	}
	if h.rejectImageProtectionWhenDisabled(c, input.Images) {
		return
	}
	if h.rejectInvalidImagePayment(c, input.Images, input.PaymentSettings) {
		return
	}
	moderationEnabled := h.postAIModerationEnabled() && !input.IsDraft
	originalVisibility := input.Visibility
	if moderationEnabled {
		input.AuditResult = jsonBytes(gin.H{"source": "ai_moderation", "mode": "post_publish_review", "originalVisibility": originalVisibility})
	}
	postID, err := repositories.NewContentRepository(h.DB).CreatePost(c.Request.Context(), input)
	if h.writeContentError(c, err, false) {
		return
	}
	msg := "\u53d1\u5e03\u6210\u529f"
	if input.IsDraft {
		msg = "\u8349\u7a3f\u4fdd\u5b58\u6210\u529f"
	}
	data := gin.H{"id": int(postID)}
	if jobs := h.enqueueVideoTranscodingForPost(c.Request.Context(), postID); len(jobs) > 0 {
		data["transcodingJobs"] = jobs
	}
	if moderationEnabled {
		if job := h.enqueueAIModeration(c.Request.Context(), services.AIModerationTargetPost, postID, user.ID, originalVisibility); job != nil {
			data["aiModerationJob"] = job
		}
	}
	if !input.IsDraft {
		h.schedulePostImageArchive(postID)
		if job := h.enqueueAIPostAutoComment(c.Request.Context(), postID, user.ID); job != nil {
			data["aiAutoCommentJob"] = job
		}
		data["points_award"] = h.awardPointsBestEffort(c, user.ID, repositories.PointsTaskPost, postID, "发布奖励")
		if awards := h.evaluateAchievementsBestEffort(c, user.ID); len(awards) > 0 {
			data["points_awards"] = awards
		}
	}
	h.bumpCacheVersions(cacheScopePosts, cacheScopeFileAccess, cacheScopeSearch, cacheScopeUsers, cacheScopeNotifications)
	markAccessOperation(c, ternaryString(input.IsDraft, "draft_save", "post_create"), "post", postID)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msg, "data": data})
}

func (h NativeHandlers) postPurchaseUserResponse(item repositories.PostPurchaseUserBundle) gin.H {
	var buyer gin.H
	if item.Buyer != nil {
		buyer = gin.H{
			"id":       expressBigInt(item.Buyer.ID),
			"user_id":  item.Buyer.UserID,
			"nickname": item.Buyer.Nickname,
			"avatar":   h.signFileURLPtr(item.Buyer.Avatar),
			"verified": item.Buyer.Verified,
		}
	}
	return gin.H{
		"id":             expressBigInt(item.Purchase.ID),
		"buyer":          buyer,
		"price":          item.Purchase.Price,
		"paid_amount":    item.Purchase.PaidAmount,
		"discount_rate":  item.Purchase.DiscountRate,
		"purchase_type":  item.Purchase.PurchaseType,
		"payment_method": normalizePaymentMethodForResponse(item.Purchase.PaymentMethod),
		"purchased_at":   item.Purchase.PurchasedAt,
	}
}

func (h NativeHandlers) UpdatePost(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	var raw map[string]any
	_ = c.ShouldBindJSON(&raw)
	maxImages := h.maxPostImages()
	if images, exists := raw["images"]; exists && imageInputCount(images) > maxImages {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.post_images_limit", gin.H{"maxImages": maxImages})
		return
	}
	input := h.updatePostInputFromMap(raw, user.ID, postID)
	if input.ImagesSet {
		input.Images = repositories.NormalizePostImageInputs(input.Images)
	}
	if input.ImagesSet && input.PaymentSet {
		input.PaymentSettings = normalizeImagePaymentSettings(input.Images, input.PaymentSettings)
	}
	if input.Content != nil {
		sanitized := sanitizePostContent(*input.Content)
		input.Content = &sanitized
		if h.rejectPostContentOverLimit(c, sanitized) {
			return
		}
	}
	if input.ImagesSet && h.rejectImageProtectionUpdateWhenDisabled(c, postID, input.Images) {
		return
	}
	if h.rejectInvalidUpdatedImagePayment(c, postID, input) {
		return
	}
	err = repositories.NewContentRepository(h.DB).UpdatePost(c.Request.Context(), input)
	if h.writeContentError(c, err, true) {
		return
	}
	data := gin.H{}
	if input.VideoSet {
		if jobs := h.enqueueVideoTranscodingForPost(c.Request.Context(), postID); len(jobs) > 0 {
			data["transcodingJobs"] = jobs
		}
	}
	h.schedulePostImageArchive(postID)
	if input.IsDraft != nil && !*input.IsDraft {
		if job := h.enqueueAIPostAutoComment(c.Request.Context(), postID, user.ID); job != nil {
			data["aiAutoCommentJob"] = job
		}
	}
	h.bumpCacheVersions(cacheScopePosts, cacheScopeFileAccess, cacheScopeSearch, cacheScopeUsers)
	behavior := "post_update"
	if input.IsDraft != nil && *input.IsDraft {
		behavior = "draft_save"
	}
	markAccessOperation(c, behavior, "post", postID)
	if len(data) > 0 {
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "\u66f4\u65b0\u6210\u529f", "data": data})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "\u66f4\u65b0\u6210\u529f"})
}

func (h NativeHandlers) DeletePost(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	var postState struct {
		IsDraft bool
	}
	_ = h.DB.WithContext(c.Request.Context()).Model(&domain.Post{}).Select("is_draft").Where("id = ? AND user_id = ?", postID, user.ID).First(&postState).Error
	files := h.postLocalFileRecycleInputs(c.Request.Context(), []int64{postID})
	err = repositories.NewContentRepository(h.DB).DeletePost(c.Request.Context(), user.ID, postID)
	if h.writeContentError(c, err, true) {
		return
	}
	h.expirePostImageArchiveJobs(c.Request.Context(), postID, "post_deleted")
	fileDeletion := h.recycleLocalPostFiles(c.Request.Context(), files)
	h.bumpCacheVersions(cacheScopePosts, cacheScopeFileAccess, cacheScopeComments, cacheScopeSearch, cacheScopeUsers, cacheScopeNotifications)
	deleteReason := ternaryString(postState.IsDraft, "draft_delete", "post_delete")
	h.recordSystemFileDeletionAudit(c, deleteReason, []int64{postID}, fileDeletion, map[string]any{"source": "user_post_delete"})
	markAccessOperation(c, deleteReason, "post", postID)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "\u5220\u9664\u6210\u529f"})
}

func (h NativeHandlers) ToggleCollect(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	collected, err := repositories.NewContentRepository(h.DB, h.notificationSuppressionConfig()).ToggleCollection(c.Request.Context(), user.ID, postID)
	if h.writeContentError(c, err, false) {
		return
	}
	msg := "\u6536\u85cf\u6210\u529f"
	if !collected {
		msg = "\u53d6\u6d88\u6536\u85cf\u6210\u529f"
	}
	data := gin.H{"collected": collected}
	if collected {
		data["points_award"] = h.awardPointsBestEffort(c, user.ID, repositories.PointsTaskCollect, postID, "收藏奖励")
	}
	h.bumpCacheVersions(cacheScopePosts, cacheScopeSearch, cacheScopeInteractions, cacheScopeNotifications)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msg, "data": data})
}

func (h NativeHandlers) writeComments(c *gin.Context, postID int64, parentID *int64, asc bool, replyCount bool) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := positiveIntQuery(c, "limit", 20)
	cacheKey := ""
	if h.Redis != nil {
		parentKey := "root"
		if parentID != nil {
			parentKey = strconv.FormatInt(*parentID, 10)
		}
		cacheKey = h.cacheKeyWithVersions(cacheScopeComments, []string{cacheScopeInteractions}, postID, parentKey, currentUserID(c), page, limit, asc, replyCount)
		var cached gin.H
		if h.Redis.CacheGetJSON(c.Request.Context(), cacheKey, &cached) {
			c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": cached})
			return
		}
	}
	total, comments, err := repositories.NewContentRepository(h.DB).Comments(c.Request.Context(), postID, parentID, currentUserID(c), page, limit, asc, replyCount)
	if h.writeContentError(c, err, false) {
		return
	}
	items := make([]gin.H, 0, len(comments))
	for _, comment := range comments {
		items = append(items, h.commentResponse(comment, replyCount))
	}
	data := gin.H{"comments": items, "pagination": pagination(page, limit, total)}
	if h.Redis != nil && cacheKey != "" {
		_ = h.Redis.CacheSet(c.Request.Context(), cacheKey, data, cacheTTL(30))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": data})
}
