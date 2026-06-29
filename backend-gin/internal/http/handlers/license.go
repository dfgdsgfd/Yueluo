package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

type verifyLicenseRequest struct {
	LicenseKey   string `json:"license_key"`
	MachineID    string `json:"machine_id"`
	MachineModel string `json:"machine_model"`
}

func (h NativeHandlers) VerifyLicense(c *gin.Context) {
	var body verifyLicenseRequest
	_ = c.ShouldBindJSON(&body)

	if body.LicenseKey == "" {
		response.ValidationError(c, "授权码不能为空", gin.H{"valid": false})
		return
	}
	if body.MachineID == "" {
		response.ValidationError(c, "机器码不能为空（machine_id），Windows请使用: wmic csproduct get uuid", gin.H{"valid": false})
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "授权验证处理失败", gin.H{"valid": false, "reason": "server_error"})
		return
	}

	repo := repositories.NewLicenseRepository(h.DB)
	license, err := repo.FindByKey(c.Request.Context(), body.LicenseKey)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusOK, gin.H{
				"code":    response.CodeSuccess,
				"message": "授权码不存在",
				"data":    gin.H{"valid": false, "reason": "not_found"},
			})
			return
		}
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "授权验证处理失败", gin.H{"valid": false, "reason": "server_error"})
		return
	}

	if !license.IsActive {
		c.JSON(http.StatusOK, gin.H{
			"code":    response.CodeSuccess,
			"message": "授权码已被禁用",
			"data":    gin.H{"valid": false, "reason": "disabled"},
		})
		return
	}
	if license.ExpiresAt != nil && license.ExpiresAt.Before(time.Now()) {
		c.JSON(http.StatusOK, gin.H{
			"code":    response.CodeSuccess,
			"message": "授权码已过期",
			"data": gin.H{
				"valid":      false,
				"reason":     "expired",
				"expires_at": license.ExpiresAt,
			},
		})
		return
	}
	if license.MachineID != nil && *license.MachineID != body.MachineID {
		c.JSON(http.StatusOK, gin.H{
			"code":    response.CodeSuccess,
			"message": "授权码已绑定其他设备，一机一用不可转移",
			"data":    gin.H{"valid": false, "reason": "machine_mismatch"},
		})
		return
	}

	if err := repo.MarkVerified(c.Request.Context(), license.ID, body.MachineID, body.MachineModel, license.MachineID == nil); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "授权验证处理失败", gin.H{"valid": false, "reason": "server_error"})
		return
	}

	machineModel := body.MachineModel
	if machineModel == "" && license.MachineModel != nil {
		machineModel = *license.MachineModel
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"message": "授权验证通过",
		"data": gin.H{
			"valid":         true,
			"machine_model": machineModel,
			"expires_at":    license.ExpiresAt,
			"license_key":   license.LicenseKey,
		},
	})
}
