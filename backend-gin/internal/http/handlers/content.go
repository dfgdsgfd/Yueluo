package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

const (
	msgContentInternal     = "\u670d\u52a1\u5668\u5185\u90e8\u9519\u8bef"
	msgPostMissing         = "\u7b14\u8bb0\u4e0d\u5b58\u5728"
	msgCommentMissing      = "\u8bc4\u8bba\u4e0d\u5b58\u5728"
	msgParentCommentMiss   = "\u7236\u8bc4\u8bba\u4e0d\u5b58\u5728"
	msgPostIDMissing       = "\u7f3a\u5c11\u7b14\u8bb0ID"
	msgCommentContentEmpty = "\u8bc4\u8bba\u5185\u5bb9\u4e0d\u80fd\u4e3a\u7a7a"
	msgCommentRequired     = "\u7b14\u8bb0ID\u548c\u8bc4\u8bba\u5185\u5bb9\u4e0d\u80fd\u4e3a\u7a7a"
	msgCommentForbidden    = "\u53ea\u80fd\u5220\u9664\u81ea\u5df1\u53d1\u5e03\u7684\u8bc4\u8bba"
	msgPostEditForbidden   = "\u53ea\u80fd\u7f16\u8f91\u81ea\u5df1\u7684\u7b14\u8bb0"
	msgPostDeleteForbidden = "\u53ea\u80fd\u5220\u9664\u81ea\u5df1\u7684\u7b14\u8bb0"
	msgLoginForDrafts      = "\u67e5\u770b\u8349\u7a3f\u9700\u8981\u767b\u5f55"
	msgLoginForContent     = "\u8bf7\u767b\u5f55\u540e\u67e5\u770b\u5185\u5bb9"
	defaultMaxPostImages   = 100
	defaultMaxPostContent  = 100000
)

type createCommentRequest struct {
	PostID   any    `json:"post_id"`
	Content  string `json:"content"`
	ParentID any    `json:"parent_id"`
}

func (h NativeHandlers) Comments(c *gin.Context) {
	postID, ok := int64FromAny(c.Query("post_id"))
	if !ok || postID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgPostIDMissing, nil)
		return
	}
	h.writeComments(c, postID, nil, false, true)
}

func (h NativeHandlers) CreateComment(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	var body createCommentRequest
	_ = c.ShouldBindJSON(&body)
	postID, okPost := int64FromAny(body.PostID)
	if !okPost || postID <= 0 || strings.TrimSpace(body.Content) == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgCommentRequired, nil)
		return
	}
	content := sanitizeCommentContent(body.Content)
	if strings.TrimSpace(content) == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgCommentContentEmpty, nil)
		return
	}
	var parentID *int64
	if parsed, ok := int64FromAny(body.ParentID); ok && parsed > 0 {
		parentID = &parsed
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	moderationEnabled := h.commentAIModerationEnabled()
	result, err := repositories.NewContentRepository(h.DB).CreateComment(c.Request.Context(), user.ID, postID, parentID, content)
	if h.writeContentError(c, err, false) {
		return
	}
	data := h.commentResponse(result.Comment, true)
	if moderationEnabled {
		if job := h.enqueueAIModeration(c.Request.Context(), services.AIModerationTargetComment, result.Comment.Comment.ID, user.ID, ""); job != nil {
			data["aiModerationJob"] = job
		}
	}
	if job := h.enqueueAICommentReply(c.Request.Context(), result.Comment.Comment.ID); job != nil {
		data["aiCommentReplyJob"] = job
	}
	if award := h.awardPointsBestEffort(c, user.ID, repositories.PointsTaskComment, result.Comment.Comment.ID, "评论奖励"); award != nil {
		data["points_award"] = award
	}
	h.bumpCacheVersions(cacheScopePosts, cacheScopeComments, cacheScopeSearch, cacheScopeNotifications)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "\u8bc4\u8bba\u6210\u529f", "data": data})
}

func (h NativeHandlers) CommentReplies(c *gin.Context) {
	parentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	h.writeComments(c, 0, &parentID, true, true)
}

func (h NativeHandlers) DeleteComment(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	commentID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	deleted, _, err := repositories.NewContentRepository(h.DB).DeleteComment(c.Request.Context(), user.ID, commentID)
	if h.writeContentError(c, err, false) {
		return
	}
	h.bumpCacheVersions(cacheScopePosts, cacheScopeComments, cacheScopeSearch, cacheScopeInteractions)
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"message": "\u5220\u9664\u6210\u529f",
		"data":    gin.H{"id": int(commentID), "deletedCount": deleted},
	})
}

func (h NativeHandlers) PostComments(c *gin.Context) {
	postID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgContentInternal, nil)
		return
	}
	repo := repositories.NewContentRepository(h.DB)
	post, err := repo.PostExists(c.Request.Context(), postID)
	if h.writeContentError(c, err, false) {
		return
	}
	if post.Type == repositories.PostTypeVideo && currentUserID(c) == 0 && h.Settings != nil && h.Settings.IsVideoGuestRestricted() && !post.PublicAccessExempt {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgLoginForContent, nil)
		return
	}
	h.writeComments(c, postID, nil, false, true)
}
