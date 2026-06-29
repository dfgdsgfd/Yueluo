package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) AdminAIJobs(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	page, limit, offset := pageLimit(c, 20)
	taskType := strings.TrimSpace(c.Query("type"))
	statuses := adminAIJobStatuses(c.Query("status"))
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.AIJob{})
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	if taskType != "" {
		query = query.Where("task_type = ?", taskType)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	var rows []domain.AIJob
	if err := query.Order("updated_at DESC, created_at DESC").Limit(limit).Offset(offset).Find(&rows).Error; err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, h.aiJobResponse(c.Request.Context(), row))
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"items":      items,
		"pagination": matrixPagination(page, limit, total),
		"stats":      h.adminAIJobStats(c.Request.Context()),
	})
}

func (h NativeHandlers) AdminCancelAIJob(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.invalid_token", nil)
		return
	}
	job, err := h.AI.CancelJob(c.Request.Context(), c.Param("jobId"), services.AIActor{Type: "admin", ID: &user.ID, DisplayID: requestUserDisplayID(user)})
	if err != nil {
		writeAIHTTPError(c, err)
		return
	}
	data := h.aiJobResponse(c.Request.Context(), job)
	if h.Queue != nil {
		data["queueAbandon"] = h.Queue.AbandonAIJobRun(c.Request.Context(), job.JobID)
	}
	writeSuccess(c, matrixMsgOK, data)
}

func adminAIJobStatuses(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "active" {
		return []string{services.AIJobStatusQueued, services.AIJobStatusRunning}
	}
	if raw == "all" {
		return nil
	}
	parts := strings.Split(raw, ",")
	statuses := make([]string, 0, len(parts))
	for _, part := range parts {
		status := strings.TrimSpace(part)
		if status != "" {
			statuses = append(statuses, status)
		}
	}
	return statuses
}

func (h NativeHandlers) adminAIJobStats(ctx context.Context) gin.H {
	stats := gin.H{"queued": 0, "running": 0, "active": 0}
	if h.DB == nil {
		return stats
	}
	var rows []struct {
		Status string
		Count  int64
	}
	if err := h.DB.WithContext(ctx).Model(&domain.AIJob{}).
		Select("status, COUNT(*) AS count").
		Where("status IN ?", []string{services.AIJobStatusQueued, services.AIJobStatusRunning}).
		Group("status").
		Scan(&rows).Error; err != nil {
		stats["error"] = matrixMsgInternal
		return stats
	}
	active := int64(0)
	for _, row := range rows {
		stats[row.Status] = row.Count
		active += row.Count
	}
	stats["active"] = active
	return stats
}
