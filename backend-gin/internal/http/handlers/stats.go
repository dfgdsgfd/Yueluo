package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

func (h NativeHandlers) Stats(c *gin.Context) {
	if h.DB == nil {
		response.Error(c, "获取统计信息失败")
		return
	}
	stats, err := repositories.NewStatsRepository(h.DB).Counts(c.Request.Context())
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "获取统计信息失败", nil)
		return
	}
	response.Success(c, stats, "获取统计信息成功")
}
