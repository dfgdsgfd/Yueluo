package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) AdminSystemUpdateStatus(c *gin.Context) {
	service := services.NewSystemUpdateServiceWithConfig(h.DB, h.Config)
	payload, err := service.Status(c.Request.Context())
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	writeSuccess(c, matrixMsgOK, payload)
}

func (h NativeHandlers) AdminSystemUpdateCheck(c *gin.Context) {
	service := services.NewSystemUpdateServiceWithConfig(h.DB, h.Config)
	payload, err := service.Check(c.Request.Context())
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	writeSuccess(c, matrixMsgOK, payload)
}

func (h NativeHandlers) AdminSystemUpdateReleases(c *gin.Context) {
	service := services.NewSystemUpdateServiceWithConfig(h.DB, h.Config)
	payload, err := service.ReleaseOptions(c.Request.Context())
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	writeSuccess(c, matrixMsgOK, payload)
}

func (h NativeHandlers) AdminSystemUpdateSaveConfig(c *gin.Context) {
	var input services.SystemUpdateConfigInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "配置格式不正确", nil)
		return
	}
	service := services.NewSystemUpdateServiceWithConfig(h.DB, h.Config)
	payload, err := service.SaveConfig(c.Request.Context(), input)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	writeSuccess(c, "更新配置已保存", payload)
}

func (h NativeHandlers) AdminSystemUpdateRun(c *gin.Context) {
	var input services.SystemUpdateRunInput
	if err := c.ShouldBindJSON(&input); err != nil {
		input = services.SystemUpdateRunInput{Frontend: true, Backend: true}
	}
	service := services.NewSystemUpdateServiceWithConfig(h.DB, h.Config)
	payload, err := service.Run(c.Request.Context(), input)
	if payload != nil {
		writeSuccess(c, "更新任务已执行", payload)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, err.Error(), nil)
		return
	}
	writeSuccess(c, "更新任务已执行", payload)
}
