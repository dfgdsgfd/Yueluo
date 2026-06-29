package handlers

import (
	"errors"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

func (h NativeHandlers) OpenPosts(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "获取帖子列表失败", nil)
		return
	}
	page := max(intQuery(c, "page", 1), 1)
	limit := min(max(intQuery(c, "limit", 20), 1), 100)
	var categoryID *int
	if raw := c.Query("category_id"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err == nil {
			categoryID = &parsed
		}
	}

	total, posts, err := repositories.NewOpenAPIRepository(h.DB).ListPublicPosts(c.Request.Context(), repositories.OpenPostListParams{
		Page:       page,
		Limit:      limit,
		CategoryID: categoryID,
	})
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "获取帖子列表失败", nil)
		return
	}
	list := make([]gin.H, 0, len(posts))
	for _, post := range posts {
		list = append(list, h.openPostResponse(post, false))
	}
	c.JSON(http.StatusOK, gin.H{
		"code": response.CodeSuccess,
		"data": gin.H{
			"list": list,
			"pagination": gin.H{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": int(math.Ceil(float64(total) / float64(limit))),
			},
		},
		"message": "success",
	})
}

func (h NativeHandlers) OpenPost(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "获取帖子详情失败", nil)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "帖子不存在", nil)
		return
	}
	post, err := repositories.NewOpenAPIRepository(h.DB).FindPublicPost(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JSON(c, http.StatusNotFound, response.CodeNotFound, "帖子不存在", nil)
			return
		}
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "获取帖子详情失败", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": h.openPostResponse(*post, true), "message": "success"})
}

func (h NativeHandlers) OpenPostImages(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "获取帖子图片失败", nil)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "帖子不存在", nil)
		return
	}
	images, err := repositories.NewOpenAPIRepository(h.DB).PublicPostImages(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.JSON(c, http.StatusNotFound, response.CodeNotFound, "帖子不存在", nil)
			return
		}
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "获取帖子图片失败", nil)
		return
	}
	data := make([]gin.H, 0, len(images))
	for _, image := range images {
		data = append(data, gin.H{"id": image.ID, "image_url": h.signFileURL(image.ImageURL), "is_free_preview": image.IsFreePreview, "sort_order": image.SortOrder})
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": data, "message": "success"})
}

func (h NativeHandlers) openPostResponse(bundle repositories.OpenPostBundle, includeVideos bool) gin.H {
	post := bundle.Post
	body := gin.H{
		"id":            post.ID,
		"user_id":       post.UserID,
		"title":         post.Title,
		"content":       post.Content,
		"category_id":   post.CategoryID,
		"type":          post.Type,
		"view_count":    post.ViewCount,
		"like_count":    post.LikeCount,
		"collect_count": post.CollectCount,
		"comment_count": post.CommentCount,
		"created_at":    post.CreatedAt,
		"is_draft":      post.IsDraft,
		"visibility":    post.Visibility,
		"user":          h.openUserResponse(bundle.User),
		"category":      openCategoryResponse(bundle.Category),
		"images":        h.openImageResponses(bundle.Images),
		"tags":          openTagResponses(bundle.Tags),
	}
	if includeVideos {
		body["videos"] = h.openVideoResponses(bundle.Videos)
	}
	return body
}

func (h NativeHandlers) openUserResponse(user *domain.User) any {
	if user == nil {
		return nil
	}
	return gin.H{
		"id":       user.ID,
		"nickname": user.Nickname,
		"avatar":   h.signFileURLPtr(user.Avatar),
		"user_id":  user.UserID,
	}
}

func openCategoryResponse(category *domain.Category) any {
	if category == nil {
		return nil
	}
	return gin.H{"id": category.ID, "name": category.Name}
}

func (h NativeHandlers) openImageResponses(images []domain.PostImage) []gin.H {
	out := make([]gin.H, 0, len(images))
	for _, image := range images {
		out = append(out, gin.H{"id": image.ID, "image_url": h.signFileURL(image.ImageURL), "is_free_preview": image.IsFreePreview, "sort_order": image.SortOrder})
	}
	return out
}

func (h NativeHandlers) openVideoResponses(videos []domain.PostVideo) []gin.H {
	out := make([]gin.H, 0, len(videos))
	for _, video := range videos {
		out = append(out, gin.H{"id": video.ID, "cover_url": h.signFileURLPtr(video.CoverURL), "video_url": h.signFileURL(video.VideoURL), "dash_url": h.signFileURLPtr(video.DashURL)})
	}
	return out
}

func openTagResponses(tags []domain.Tag) []gin.H {
	out := make([]gin.H, 0, len(tags))
	for _, tag := range tags {
		out = append(out, gin.H{"id": tag.ID, "name": tag.Name})
	}
	return out
}
