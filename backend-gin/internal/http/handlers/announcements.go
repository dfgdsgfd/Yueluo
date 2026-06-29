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

type announcementDTO struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	Type        string `json:"type"`
	PublishedAt any    `json:"published_at"`
	ExpiresAt   any    `json:"expires_at"`
	CreatedAt   any    `json:"created_at"`
}

func (h NativeHandlers) Announcements(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}

	page := max(intQuery(c, "page", 1), 1)
	pageSize := min(max(intQuery(c, "pageSize", 20), 1), 50)

	repo := repositories.NewAnnouncementRepository(h.DB)
	total, list, err := repo.ListPublished(c.Request.Context(), repositories.AnnouncementListParams{
		Page:     page,
		PageSize: pageSize,
		Type:     c.Query("type"),
	})
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}

	serialized := make([]announcementDTO, 0, len(list))
	for _, item := range list {
		serialized = append(serialized, announcementResponse(item))
	}
	c.JSON(http.StatusOK, gin.H{
		"code": response.CodeSuccess,
		"data": gin.H{
			"list": serialized,
			"pagination": gin.H{
				"total":    total,
				"page":     page,
				"pageSize": pageSize,
				"pages":    int(math.Ceil(float64(total) / float64(pageSize))),
			},
		},
	})
}

func (h NativeHandlers) Announcement(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	announcement, err := repositories.NewAnnouncementRepository(h.DB).FindPublished(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.NotFound(c, "公告不存在")
			return
		}
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "服务器内部错误", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": response.CodeSuccess,
		"data": announcementResponse(*announcement),
	})
}

func announcementResponse(item domain.Announcement) announcementDTO {
	return announcementDTO{
		ID:          strconv.FormatInt(item.ID, 10),
		Title:       item.Title,
		Content:     item.Content,
		Type:        item.Type,
		PublishedAt: item.PublishedAt,
		ExpiresAt:   item.ExpiresAt,
		CreatedAt:   item.CreatedAt,
	}
}

func intQuery(c *gin.Context, key string, fallback int) int {
	value, err := strconv.Atoi(c.Query(key))
	if err != nil {
		return fallback
	}
	return value
}
