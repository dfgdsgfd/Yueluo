package handlers

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

var batchImageExts = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".bmp": true}

var batchVideoExts = map[string]bool{".mp4": true, ".avi": true, ".mov": true, ".wmv": true, ".flv": true, ".webm": true, ".mkv": true}

var apkExts = map[string]bool{".apk": true, ".apks": true}

const adminPerformanceCacheTTL = 5 * time.Second

func (h NativeHandlers) queueAvailable(ctx context.Context) bool {
	return h.Queue != nil && h.Queue.Available()
}

func (h NativeHandlers) adminQueueNames(c *gin.Context) {
	enabled := h.queueAvailable(c.Request.Context())
	names := services.QueueNames
	status := gin.H{"enabled": enabled}
	if h.Queue != nil {
		names = h.Queue.Names()
		status = gin.H(h.Queue.RuntimeStatus())
	}
	writeSuccess(c, matrixMsgOK, gin.H{"enabled": enabled, "names": names, "status": status})
}

func (h NativeHandlers) adminQueueStats(c *gin.Context) {
	if h.Queue == nil {
		stats := services.EmptyQueueStats()
		writeSuccess(c, matrixMsgOK, gin.H{
			"enabled":      false,
			"available":    false,
			"status":       gin.H{"enabled": false, "available": false, "message": "queue service is not initialized"},
			"queues":       services.QueueStatsByNameForResponse(stats),
			"queueList":    stats,
			"queuesByName": services.QueueStatsByNameForResponse(stats),
			"names":        services.QueueNames,
			"events":       []gin.H{},
			"auditLog":     h.auditLogRuntimeStatus(),
		})
		return
	}
	enabled, queueList, queuesByName := h.Queue.Stats()
	status := h.Queue.RuntimeStatus()
	writeSuccess(c, matrixMsgOK, gin.H{
		"enabled":       enabled,
		"available":     enabled,
		"status":        status,
		"queues":        queuesByName,
		"queueList":     queueList,
		"queuesByName":  queuesByName,
		"names":         h.Queue.Names(),
		"runtimeStatus": status,
		"events":        h.Queue.RecentEvents(c.Request.Context(), "", 100),
		"auditLog":      h.auditLogRuntimeStatus(),
	})
}

func (h NativeHandlers) adminSystemLogs(c *gin.Context) {
	if h.Observe == nil {
		writeSuccess(c, matrixMsgOK, gin.H{"enabled": false, "items": []gin.H{}})
		return
	}
	limit := int64(positiveIntQuery(c, "limit", 30))
	items, nextCursor, hasMore, err := h.Observe.SystemLogs(c.Request.Context(), limit, c.Query("cursor"))
	if writeDBError(c, err, "") {
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{"enabled": true, "items": items, "nextCursor": nextCursor, "hasMore": hasMore, "retention_hours": int(h.Observe.SystemLogRetention().Hours())})
}

func (h NativeHandlers) AdminSystemLogs(c *gin.Context) {
	h.adminSystemLogs(c)
}

func (h NativeHandlers) adminPerformance(c *gin.Context) {
	if h.Observe == nil {
		writeSuccess(c, matrixMsgOK, gin.H{"enabled": false})
		return
	}
	options := services.PerformanceOptions{
		Window:    adminDurationQuery(c.Query("range"), h.Observe.MetricsRetention()),
		Bucket:    adminDurationQuery(c.Query("bucket"), h.Config.Observe.MetricsBucket),
		SlowLimit: int64(positiveIntQuery(c, "slowLimit", 50)),
	}
	payload := h.cachedAdminPerformancePayload(c.Request.Context(), options)
	writeSuccess(c, matrixMsgOK, payload)
}

func (h NativeHandlers) cachedAdminPerformancePayload(ctx context.Context, options services.PerformanceOptions) gin.H {
	if h.Cache != nil {
		key := services.CacheKey(
			"admin_performance",
			0,
			int64(options.Window.Seconds()),
			int64(options.Bucket.Seconds()),
			options.SlowLimit,
			int64(h.Observe.MetricsRetention().Seconds()),
		)
		if value, ok := h.Cache.Get(key); ok {
			if payload, ok := value.(gin.H); ok {
				return payload
			}
		}
		payload := h.adminPerformancePayload(ctx, options)
		h.Cache.Set(key, payload, adminPerformanceCacheTTL)
		return payload
	}
	return h.adminPerformancePayload(ctx, options)
}

func (h NativeHandlers) adminPerformancePayload(ctx context.Context, options services.PerformanceOptions) gin.H {
	raw := h.Observe.Performance(ctx, h.DB, options)
	payload := gin.H{}
	maps.Copy(payload, raw)
	payload["enabled"] = true
	payload["retention_hours"] = int(h.Observe.MetricsRetention().Hours())
	if h.Redis != nil {
		redisStatus := h.Redis.Status(ctx, h.Config.Redis)
		payload["redis"] = redisStatus
		if info, ok := redisStatus["info"].(map[string]any); ok {
			if redisVersion := toString(info["redis_version"]); redisVersion != "" {
				versions := gin.H{}
				if raw, err := json.Marshal(payload["versions"]); err == nil {
					_ = json.Unmarshal(raw, &versions)
				}
				versions["redis"] = redisVersion
				payload["versions"] = versions
			}
		}
	}
	return payload
}

func (h NativeHandlers) AdminPerformance(c *gin.Context) {
	h.adminPerformance(c)
}

func (h NativeHandlers) adminObservabilityEvents(c *gin.Context) {
	if h.Observe == nil {
		writeSuccess(c, matrixMsgOK, gin.H{
			"enabled": false,
			"type":    c.DefaultQuery("type", "errors"),
			"items":   []gin.H{},
			"pagination": gin.H{
				"page":  1,
				"limit": positiveIntQuery(c, "limit", 30),
				"total": 0,
			},
		})
		return
	}
	status := 0
	if raw := strings.TrimSpace(c.Query("status")); raw != "" {
		status, _ = strconv.Atoi(raw)
	}
	payload := h.Observe.Events(c.Request.Context(), services.ObservabilityEventOptions{
		Type:    c.DefaultQuery("type", "errors"),
		Window:  adminDurationQuery(c.Query("range"), h.Observe.MetricsRetention()),
		Page:    positiveIntQuery(c, "page", 1),
		Limit:   positiveIntQuery(c, "limit", 30),
		Keyword: c.Query("keyword"),
		Method:  c.Query("method"),
		Status:  status,
	})
	writeSuccess(c, matrixMsgOK, payload)
}

func (h NativeHandlers) AdminObservabilityEvents(c *gin.Context) {
	h.adminObservabilityEvents(c)
}

func (h NativeHandlers) adminObservabilityAccessLog(c *gin.Context) {
	if h.Observe == nil {
		writeSuccess(c, matrixMsgOK, gin.H{"enabled": false, "items": []gin.H{}})
		return
	}
	limit := int64(positiveIntQuery(c, "limit", 100))
	items, err := h.Observe.AccessLogs(c.Request.Context(), limit)
	if writeDBError(c, err, "") {
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{"enabled": true, "items": items, "limit": limit})
}

func (h NativeHandlers) AdminObservabilityAccessLog(c *gin.Context) {
	h.adminObservabilityAccessLog(c)
}

func (h NativeHandlers) adminQueueDispatch(c *gin.Context) {
	parts := strings.Split(strings.TrimPrefix(c.Request.URL.Path, "/api/admin/queues/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "队列名称不能为空", nil)
		return
	}
	name := parts[0]
	if !knownQueueName(name) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "队列不存在", nil)
		return
	}
	enabled := h.queueAvailable(c.Request.Context())
	if !enabled {
		if matrixMethod(c) == http.MethodGet {
			writeSuccess(c, matrixMsgOK, gin.H{"enabled": enabled, "jobs": []gin.H{}})
			return
		}
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "队列服务未启用", nil)
		return
	}
	switch {
	case matrixMethod(c) == http.MethodDelete && len(parts) == 1:
		h.adminQueueClean(c, name)
	case matrixMethod(c) == http.MethodGet && len(parts) == 2 && parts[1] == "jobs":
		h.adminQueueJobs(c, name)
	case matrixMethod(c) == http.MethodGet && len(parts) == 3 && parts[1] == "jobs":
		h.adminQueueJobDetail(c, name, parts[2])
	case matrixMethod(c) == http.MethodPost && len(parts) == 4 && parts[1] == "jobs" && parts[3] == "retry":
		h.adminQueueRetry(c, name, parts[2])
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "队列路由不存在", nil)
	}
}

func (h NativeHandlers) adminQueueJobs(c *gin.Context, name string) {
	status := c.DefaultQuery("status", "waiting")
	start := positiveIntQuery(c, "start", 0)
	end := positiveIntQuery(c, "end", 20)
	enabled, jobs, err := h.Queue.Jobs(name, status, start, end)
	if err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeError, err.Error(), nil)
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{"enabled": enabled, "status": status, "start": start, "end": end, "count": len(jobs), "jobs": jobs})
}

func (h NativeHandlers) adminQueueJobDetail(c *gin.Context, name string, jobID string) {
	enabled, job, err := h.Queue.Job(name, jobID)
	if err != nil || job == nil {
		message := "任务不存在"
		if err != nil {
			message = err.Error()
		}
		response.JSON(c, http.StatusBadRequest, response.CodeError, message, nil)
		return
	}
	writeSuccess(c, matrixMsgOK, gin.H{"enabled": enabled, "job": job})
}

func (h NativeHandlers) adminQueueRetry(c *gin.Context, name string, jobID string) {
	if err := h.Queue.Retry(name, jobID); err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeError, err.Error(), nil)
		return
	}
	writeSimpleSuccess(c, "任务已重新加入队列")
}

func (h NativeHandlers) adminQueueClean(c *gin.Context, name string) {
	if err := h.Queue.Clean(name); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	writeSimpleSuccess(c, "队列已清空")
}
