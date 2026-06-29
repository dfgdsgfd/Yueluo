package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) AdminRedisMaintenance(c *gin.Context) {
	if h.RedisCare == nil {
		writeSuccess(c, matrixMsgOK, gin.H{
			"configured": false,
			"available":  false,
			"config":     services.ReadRedisMaintenanceConfig(h.Settings),
		})
		return
	}
	writeSuccess(c, matrixMsgOK, h.RedisCare.Status(c.Request.Context()))
}

func (h NativeHandlers) AdminRedisMaintenanceUpdate(c *gin.Context) {
	if h.Settings == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	var cfg services.RedisMaintenanceConfig
	if err := c.ShouldBindJSON(&cfg); err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.invalid_redis_maintenance_config", nil)
		return
	}
	if !services.SaveRedisMaintenanceConfig(c.Request.Context(), h.Settings, cfg) {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	h.AdminRedisMaintenance(c)
}

func (h NativeHandlers) AdminRedisMaintenanceRun(c *gin.Context) {
	if h.RedisCare == nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.redis_unavailable", nil)
		return
	}
	body := readBodyMap(c)
	categories := stringSliceFromAny(body["categories"])
	result, err := h.RedisCare.Run(c.Request.Context(), categories)
	if errors.Is(err, services.ErrRedisMaintenanceBusy) {
		response.JSON(c, http.StatusConflict, response.CodeError, "error.redis_maintenance_busy", nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "error.redis_maintenance_failed", gin.H{"detail": err.Error()})
		return
	}
	writeSuccess(c, "success", gin.H{"result": result, "status": h.RedisCare.Status(c.Request.Context())})
}
