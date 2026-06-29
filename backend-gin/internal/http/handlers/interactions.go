package handlers

import (
	"errors"
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

type likeRequest struct {
	TargetType any `json:"target_type"`
	TargetID   any `json:"target_id"`
}

func (h NativeHandlers) ToggleLike(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "无效的访问令牌", nil)
		return
	}
	var body likeRequest
	_ = c.ShouldBindJSON(&body)
	targetType, okType := intFromAny(body.TargetType)
	targetID, okID := int64FromAny(body.TargetID)
	if !okType || !okID {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少必要参数", nil)
		return
	}
	if targetType != 1 && targetType != 2 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效的目标类型", nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	result, err := repositories.NewInteractionsRepository(h.DB, h.notificationSuppressionConfig()).ToggleLike(c.Request.Context(), user.ID, targetType, targetID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	markAccessOperation(c, "like", likeTargetTypeLabel(targetType), targetID)
	h.bumpCacheVersions(cacheScopePosts, cacheScopeComments, cacheScopeSearch, cacheScopeInteractions, cacheScopeNotifications)
	if result.Liked {
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "点赞成功", "data": gin.H{
			"liked":        true,
			"points_award": h.awardPointsBestEffort(c, user.ID, repositories.PointsTaskLike, strconv.FormatInt(int64(targetType), 10)+":"+strconv.FormatInt(targetID, 10), "点赞奖励"),
		}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "取消点赞成功", "data": gin.H{"liked": false}})
}

func (h NativeHandlers) RemoveLike(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "无效的访问令牌", nil)
		return
	}
	var body likeRequest
	_ = c.ShouldBindJSON(&body)
	targetType, okType := intFromAny(body.TargetType)
	targetID, okID := int64FromAny(body.TargetID)
	if !okType || !okID {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少必要参数", nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	err := repositories.NewInteractionsRepository(h.DB).RemoveLike(c.Request.Context(), user.ID, targetType, targetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JSON(c, http.StatusNotFound, response.CodeNotFound, "点赞记录不存在", nil)
			return
		}
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	markAccessOperation(c, "like", likeTargetTypeLabel(targetType), targetID)
	h.bumpCacheVersions(cacheScopePosts, cacheScopeComments, cacheScopeSearch, cacheScopeInteractions)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "取消点赞成功"})
}

func likeTargetTypeLabel(targetType int) string {
	if targetType == 2 {
		return "comment"
	}
	return "post"
}

type dislikeRequest struct {
	PostID any `json:"post_id"`
}

func (h NativeHandlers) ToggleDislike(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "无效的访问令牌", nil)
		return
	}
	var body dislikeRequest
	_ = c.ShouldBindJSON(&body)
	postID, okID := int64FromAny(body.PostID)
	if !okID {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少必要参数", nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	disliked, err := repositories.NewInteractionsRepository(h.DB).ToggleDislike(c.Request.Context(), user.ID, postID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	h.bumpCacheVersions(cacheScopePosts, cacheScopeSearch, cacheScopeInteractions)
	if disliked {
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "已标记不喜欢", "data": gin.H{"disliked": true}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "已取消不喜欢", "data": gin.H{"disliked": false}})
}

func (h NativeHandlers) DislikeStatus(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "无效的访问令牌", nil)
		return
	}
	postID, err := strconv.ParseInt(c.Query("post_id"), 10, 64)
	if c.Query("post_id") == "" || err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少必要参数", nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	disliked, err := repositories.NewInteractionsRepository(h.DB).HasDislike(c.Request.Context(), user.ID, postID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"disliked": disliked}})
}

type reportRequest struct {
	TargetType  string `json:"target_type"`
	TargetID    any    `json:"target_id"`
	Reason      string `json:"reason"`
	Description string `json:"description"`
}

var validReportTargetTypes = map[string]bool{"post": true, "user": true, "comment": true}
var validReportReasons = map[string]bool{
	"spam": true, "porn": true, "violence": true, "fake": true,
	"harassment": true, "copyright": true, "other": true,
}

func (h NativeHandlers) CreateReport(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "无效的访问令牌", nil)
		return
	}
	var body reportRequest
	_ = c.ShouldBindJSON(&body)
	targetID, okID := int64FromAny(body.TargetID)
	if body.TargetType == "" || !okID || body.Reason == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少必要参数", nil)
		return
	}
	if !validReportTargetTypes[body.TargetType] {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效的举报类型", nil)
		return
	}
	if !validReportReasons[body.Reason] {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效的举报原因", nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	repo := repositories.NewInteractionsRepository(h.DB)
	exists, err := repo.ReportExists(c.Request.Context(), user.ID, body.TargetType, targetID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	if exists {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "你已经举报过该内容", nil)
		return
	}
	var description *string
	if sanitized := sanitizeMarkdownSubmittedText(body.Description); sanitized != "" {
		description = &sanitized
	}
	report, err := repo.CreateReport(c.Request.Context(), domain.Report{
		ReporterID:  user.ID,
		TargetType:  body.TargetType,
		TargetID:    targetID,
		Reason:      body.Reason,
		Description: description,
	})
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"message": "举报已提交，感谢你的反馈",
		"data":    gin.H{"id": strconv.FormatInt(report.ID, 10)},
	})
}

func (h NativeHandlers) ReportStatus(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "无效的访问令牌", nil)
		return
	}
	targetType := c.Query("target_type")
	targetID, err := strconv.ParseInt(c.Query("target_id"), 10, 64)
	if targetType == "" || c.Query("target_id") == "" || err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少必要参数", nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	reported, err := repositories.NewInteractionsRepository(h.DB).ReportExists(c.Request.Context(), user.ID, targetType, targetID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"reported": reported}})
}

func currentUser(c *gin.Context) (*services.RequestUser, bool) {
	value, ok := c.Get("user")
	if !ok {
		return nil, false
	}
	user, ok := value.(*services.RequestUser)
	return user, ok
}

func int64FromAny(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		return int64(typed), true
	case int:
		return int64(typed), true
	case int64:
		return typed, true
	case string:
		if strings.TrimSpace(typed) == "" {
			return 0, false
		}
		parsed, err := strconv.ParseInt(typed, 10, 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
