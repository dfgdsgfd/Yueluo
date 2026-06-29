package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
)

func (h NativeHandlers) adminQualityRewardSettings(c *gin.Context) {
	var rows []domain.PostQualityRewardSetting
	err := h.DB.WithContext(c.Request.Context()).Order("id ASC").Find(&rows).Error
	if err != nil {
		writeSuccess(c, matrixMsgOK, gin.H{"data": defaultQualityRewardSettings()})
		return
	}
	if len(rows) == 0 {
		defaults := []domain.PostQualityRewardSetting{
			{QualityLevel: "low", RewardAmount: 1, Description: stringPtr("低质量奖励"), IsActive: true},
			{QualityLevel: "medium", RewardAmount: 3, Description: stringPtr("中质量奖励"), IsActive: true},
			{QualityLevel: "high", RewardAmount: 5, Description: stringPtr("高质量奖励"), IsActive: true},
		}
		if err := h.DB.WithContext(c.Request.Context()).Create(&defaults).Error; err == nil {
			rows = defaults
		} else {
			writeSuccess(c, matrixMsgOK, gin.H{"data": defaultQualityRewardSettings()})
			return
		}
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, qualityRewardSettingMap(row))
	}
	writeSuccess(c, matrixMsgOK, gin.H{"data": out})
}

func (h NativeHandlers) adminUpdateQualityRewardSetting(c *gin.Context) {
	settingID, ok := intFromAny(matrixParam(c, "id"))
	if !ok || settingID <= 0 {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "设置不存在", nil)
		return
	}
	var existing domain.PostQualityRewardSetting
	err := h.DB.WithContext(c.Request.Context()).Where("id = ?", settingID).Take(&existing).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			response.JSON(c, http.StatusNotFound, response.CodeNotFound, "设置不存在", nil)
			return
		}
		response.JSON(c, http.StatusServiceUnavailable, response.CodeError, "质量奖励功能暂不可用，请先运行数据库迁移", nil)
		return
	}

	body := readBodyMap(c)
	updates := map[string]any{}
	if raw, exists := body["reward_amount"]; exists {
		amount, _ := strconv.ParseFloat(toString(raw), 64)
		updates["reward_amount"] = amount
	}
	if raw, exists := body["description"]; exists {
		updates["description"] = toString(raw)
	}
	if raw, exists := body["is_active"]; exists {
		updates["is_active"] = raw == true || toString(raw) == "true"
	}
	if len(updates) > 0 {
		if err := h.DB.WithContext(c.Request.Context()).Model(&domain.PostQualityRewardSetting{}).Where("id = ?", settingID).Updates(updates).Error; writeDBError(c, err, "") {
			return
		}
	}
	writeSimpleSuccess(c, "更新成功")
}

func qualityRewardSettingMap(row domain.PostQualityRewardSetting) gin.H {
	return gin.H{
		"id":            row.ID,
		"quality_level": row.QualityLevel,
		"reward_amount": row.RewardAmount,
		"description":   row.Description,
		"is_active":     row.IsActive,
		"created_at":    row.CreatedAt,
		"updated_at":    row.UpdatedAt,
	}
}

func defaultQualityRewardSettings() []gin.H {
	return []gin.H{
		{"id": 1, "quality_level": "low", "reward_amount": 1.00, "description": "低质量奖励", "is_active": true},
		{"id": 2, "quality_level": "medium", "reward_amount": 3.00, "description": "中质量奖励", "is_active": true},
		{"id": 3, "quality_level": "high", "reward_amount": 5.00, "description": "高质量奖励", "is_active": true},
	}
}
