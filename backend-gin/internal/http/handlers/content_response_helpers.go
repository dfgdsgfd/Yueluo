package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

func (h NativeHandlers) writePostList(c *gin.Context, opts repositories.PostListOptions) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := positiveIntQuery(c, "limit", 20)
	currentUserID := currentUserID(c)
	if rawType := c.Query("type"); rawType != "" {
		if parsed, err := strconv.Atoi(rawType); err == nil {
			opts.Type = &parsed
		}
	}
	opts.PublicAccessOnly = opts.PublicAccessOnly || publicAccessExemptOnly(c)
	if opts.PublicAccessOnly {
		opts.IsDraft = false
	}
	if opts.Type != nil && *opts.Type == repositories.PostTypeVideo && currentUserID == 0 && h.Settings != nil && h.Settings.IsVideoGuestRestricted() && !opts.PublicAccessOnly {
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": postListData([]gin.H{}, page, limit, 0, opts)})
		return
	}
	if opts.IsDraft && currentUserID == 0 {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgLoginForDrafts, nil)
		return
	}
	opts.Page = page
	opts.Limit = limit
	opts.CurrentUserID = currentUserID
	opts.ExcludeVideoGuests = h.Settings != nil && h.Settings.IsVideoGuestRestricted()
	if opts.Mode == "recommended" && h.Settings != nil {
		opts.RecommendConfig = h.Settings.Get("recommend_config")
	}
	cacheKey := ""
	cacheable := h.Redis != nil && !opts.IsDraft
	if cacheable {
		cacheKey = h.postListCacheKey(c, opts, page, limit, currentUserID)
		var cached gin.H
		if h.Redis.CacheGetJSON(c.Request.Context(), cacheKey, &cached) {
			c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": cached})
			return
		}
	}
	if cacheable && cacheKey != "" && h.PostListGroup != nil {
		value, err, _ := h.PostListGroup.Do(cacheKey, func() (any, error) {
			var cached gin.H
			if h.Redis.CacheGetJSON(c.Request.Context(), cacheKey, &cached) {
				return cached, nil
			}
			data, err := h.loadPostListData(c, opts, page, limit, currentUserID)
			if err == nil {
				_ = h.Redis.CacheSet(c.Request.Context(), cacheKey, data, postListCacheTTL(opts))
			}
			return data, err
		})
		if h.writeContentError(c, err, true) {
			return
		}
		if data, ok := value.(gin.H); ok {
			c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": data})
			return
		}
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	data, err := h.loadPostListData(c, opts, page, limit, currentUserID)
	if h.writeContentError(c, err, true) {
		return
	}
	if cacheable && cacheKey != "" {
		_ = h.Redis.CacheSet(c.Request.Context(), cacheKey, data, postListCacheTTL(opts))
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": data})
}

func (h NativeHandlers) loadPostListData(c *gin.Context, opts repositories.PostListOptions, page int, limit int, currentUserID int64) (gin.H, error) {
	result, err := repositories.NewContentRepository(h.DB).ListPosts(c.Request.Context(), opts)
	if err != nil {
		return nil, err
	}
	postIDs := postBundleIDs(result.Posts)
	purchased, liked, collected, err := h.postInteractionSets(c, currentUserID, postIDs)
	if err != nil {
		return nil, err
	}
	posts := make([]gin.H, 0, len(result.Posts))
	for _, bundle := range result.Posts {
		item := h.postListResponse(bundle, currentUserID, purchased[bundle.Post.ID], liked[bundle.Post.ID], collected[bundle.Post.ID])
		if bundle.RecommendationScore != nil {
			item["_recommendationScore"] = *bundle.RecommendationScore
			item["_scoreBreakdown"] = bundle.ScoreBreakdown
		}
		posts = append(posts, item)
	}
	if currentUserID == 0 && opts.ExcludeVideoGuests {
		filtered := posts[:0]
		for _, post := range posts {
			if post["type"] != repositories.PostTypeVideo || post["public_access_exempt"] == true {
				filtered = append(filtered, post)
			}
		}
		posts = filtered
	}
	data := postListData(posts, page, limit, result.Total, opts)
	if result.HasFriends != nil {
		data["hasFriends"] = *result.HasFriends
		data["recommendedUsers"] = h.recommendedUsersResponse(result.RecommendedUsers)
	}
	if result.HasFollowing != nil {
		data["hasFollowing"] = *result.HasFollowing
		data["recommendedUsers"] = h.recommendedUsersResponse(result.RecommendedUsers)
	}
	if c.Query("debug") == "true" && opts.Mode == "recommended" {
		if result.Debug != nil {
			data["_recommendationDebug"] = result.Debug
		} else {
			data["_recommendationDebug"] = gin.H{"enabled": true, "statistics": gin.H{"returnedPosts": len(posts)}}
		}
	}
	return data, nil
}

func postListData(posts []gin.H, page, limit int, total int64, opts repositories.PostListOptions) gin.H {
	return gin.H{"posts": posts, "pagination": pagination(page, limit, total)}
}

func (h NativeHandlers) postListCacheKey(c *gin.Context, opts repositories.PostListOptions, page int, limit int, currentUserID int64) string {
	categoryID := "all"
	if opts.CategoryID != nil {
		categoryID = strconv.Itoa(*opts.CategoryID)
	}
	postType := "all"
	if opts.Type != nil {
		postType = strconv.Itoa(*opts.Type)
	}
	userID := "all"
	if opts.UserID != nil {
		userID = strconv.FormatInt(*opts.UserID, 10)
	}
	return h.cacheKeyWithVersions(
		cacheScopePosts,
		[]string{cacheScopeInteractions, cacheScopeSettings, cacheScopeUsers},
		opts.Mode,
		opts.Sort,
		page,
		limit,
		currentUserID,
		categoryID,
		postType,
		userID,
		opts.TimeRangeDays,
		opts.ExcludeVideoGuests,
		opts.PublicAccessOnly,
		c.Query("debug"),
	)
}

func postListCacheTTL(opts repositories.PostListOptions) time.Duration {
	switch opts.Mode {
	case "hot":
		return cacheTTL(45)
	case "recommended":
		return cacheTTL(30)
	default:
		return cacheTTL(25)
	}
}

func (h NativeHandlers) writeContentError(c *gin.Context, err error, edit bool) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, repositories.ErrContentPostMissing):
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgPostMissing, nil)
	case errors.Is(err, repositories.ErrContentParentMissing):
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgParentCommentMiss, nil)
	case errors.Is(err, repositories.ErrContentCommentMissing):
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgCommentMissing, nil)
	case errors.Is(err, repositories.ErrContentForbidden):
		message := msgCommentForbidden
		if edit {
			message = msgPostEditForbidden
		}
		response.JSON(c, http.StatusForbidden, response.CodeForbidden, message, nil)
	case errors.Is(err, gorm.ErrRecordNotFound):
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgPostMissing, nil)
	default:
		_ = c.Error(err)
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
	}
	return true
}

func (h NativeHandlers) resolvePostUserID(c *gin.Context, raw string) (*int64, error) {
	if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return &id, nil
	}
	var user domain.User
	if h.DB == nil {
		return nil, nil
	}
	err := h.DB.WithContext(c.Request.Context()).Where("user_id = ?", raw).Select("id").First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &user.ID, nil
}

func (h NativeHandlers) postInteractionSets(c *gin.Context, currentUserID int64, postIDs []int64) (map[int64]bool, map[int64]bool, map[int64]bool, error) {
	return repositories.NewContentRepository(h.DB).InteractionSets(c.Request.Context(), currentUserID, postIDs)
}

func (h NativeHandlers) commentResponse(bundle repositories.CommentBundle, includeReplyCount bool) gin.H {
	comment := bundle.Comment
	body := gin.H{
		"id":              int(comment.ID),
		"post_id":         int(comment.PostID),
		"user_id":         int(comment.UserID),
		"parent_id":       comment.ParentID,
		"content":         comment.Content,
		"like_count":      comment.LikeCount,
		"audit_status":    comment.AuditStatus,
		"is_public":       comment.IsPublic,
		"audit_result":    jsonRawOrNil(comment.AuditResult),
		"created_at":      comment.CreatedAt,
		"nickname":        nil,
		"user_avatar":     nil,
		"user_auto_id":    nil,
		"user_display_id": nil,
		"user_location":   nil,
		"verified":        nil,
		"liked":           bundle.Liked,
	}
	if bundle.User != nil {
		body["nickname"] = bundle.User.Nickname
		body["user_avatar"] = h.signFileURLPtr(bundle.User.Avatar)
		body["user_auto_id"] = int(bundle.User.ID)
		body["user_display_id"] = bundle.User.UserID
		body["user_location"] = bundle.User.Location
		body["verified"] = bundle.User.Verified
	}
	if includeReplyCount {
		body["reply_count"] = bundle.ReplyCount
	}
	return body
}

func (h NativeHandlers) postListResponse(bundle repositories.PostBundle, currentUserID int64, hasPurchased, liked, collected bool) gin.H {
	post := bundle.Post
	body := h.basePostResponse(bundle)
	isAuthor := currentUserID != 0 && post.UserID == currentUserID
	h.protectPostListItem(body, bundle.PaymentSetting, isAuthor, hasPurchased, bundle.Videos, bundle.Images)
	body["tags"] = tagsResponse(bundle.Tags)
	body["liked"] = liked
	body["collected"] = collected
	return body
}

func (h NativeHandlers) postDetailResponse(bundle repositories.PostBundle, currentUserID int64, hasPurchased, liked, collected bool) gin.H {
	body := h.basePostResponse(bundle)
	post := bundle.Post
	isAuthor := currentUserID != 0 && post.UserID == currentUserID
	canViewPaid := bundle.PaymentSetting == nil || !bundle.PaymentSetting.Enabled || isAuthor || hasPurchased
	access := imageAccessForViewer(bundle.Images, bundle.PaymentSetting, canViewPaid, isAuthor)
	body["images"] = h.detailImagesResponse(access.DirectImages)
	body["videos"] = h.videosResponse(bundle.Videos)
	body["attachment"] = h.attachmentResponse(bundle.Attachments)
	body["tags"] = tagsResponse(bundle.Tags)
	if post.Type == repositories.PostTypeVideo && len(bundle.Videos) > 0 {
		first := bundle.Videos[0]
		body["video_url"] = h.signFileURL(first.VideoURL)
		body["cover_url"] = h.signFileURLPtr(first.CoverURL)
		body["preview_video_url"] = h.signFileURLPtr(first.PreviewVideoURL)
	}
	body["isPaidContent"] = bundle.PaymentSetting != nil && bundle.PaymentSetting.Enabled
	body["hasPurchased"] = hasPurchased || isAuthor
	body["isAuthor"] = isAuthor
	body["resourceSectionPosition"] = h.postResourceSectionPosition()
	applyImageAccessMetadata(body, access)
	h.applyImageArchiveMetadata(body, access, bundle.PaymentSetting, canViewPaid)
	if bundle.PaymentSetting != nil {
		body["paymentSettings"] = paymentSettingsResponse(bundle.PaymentSetting)
	}
	if bundle.PaymentSetting != nil && bundle.PaymentSetting.Enabled && !isAuthor && !hasPurchased {
		protectPostDetail(body, bundle.PaymentSetting)
	}
	body["liked"] = liked
	body["collected"] = collected
	return body
}
