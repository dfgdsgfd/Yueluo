package handlers

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

var versionCleaner = regexp.MustCompile(`[^0-9.]`)

func (h NativeHandlers) AppDownloadConfig(c *gin.Context) {
	config := gin.H{
		"android":      h.appDownloadPlatformConfig("android"),
		"android_fast": h.appDownloadPlatformConfig("android_fast"),
		"ios":          h.appDownloadPlatformConfig("ios"),
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": config, "message": "success"})
}

func (h NativeHandlers) appDownloadPlatformConfig(platform string) gin.H {
	prefix := "app_download_" + platform + "_"
	payload := gin.H{
		"enabled":       h.settingValue(prefix+"enabled", true),
		"name":          h.settingValue(prefix+"name", ""),
		"version_name":  h.settingValue(prefix+"version_name", ""),
		"version_code":  h.settingValue(prefix+"version_code", 0),
		"download_url":  h.settingValue(prefix+"download_url", ""),
		"size_label":    h.settingValue(prefix+"size_label", ""),
		"size_bytes":    h.settingValue(prefix+"size_bytes", 0),
		"release_notes": h.settingValue(prefix+"release_notes", ""),
	}
	if platform == "ios" {
		payload["bundle_id"] = h.settingValue(prefix+"bundle_id", "")
	} else {
		payload["package_name"] = h.settingValue(prefix+"package_name", "")
	}
	return payload
}

func (h NativeHandlers) settingValue(key string, fallback any) any {
	if h.Settings == nil {
		if value, ok := services.DefaultSettings[key]; ok {
			return value
		}
		return fallback
	}
	value := h.Settings.Get(key)
	if value == nil {
		return fallback
	}
	return value
}

func (h NativeHandlers) CheckAppUpdate(c *gin.Context) {
	platform := c.Query("platform")
	versionName := c.Query("version_name")
	versionCode := c.Query("version_code")

	if platform == "" || (versionName == "" && versionCode == "") {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少必要参数: platform, version_name", nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "应用版本功能暂不可用", nil)
		return
	}

	versions, err := repositories.NewAppRepository(h.DB).ActiveVersions(c.Request.Context(), platform)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "检查更新失败", nil)
		return
	}
	if len(versions) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"has_update": false}, "message": "已是最新版本"})
		return
	}
	sort.SliceStable(versions, func(i, j int) bool {
		return compareVersionNames(versions[i].VersionName, versions[j].VersionName) > 0
	})
	latest := versions[0]

	hasUpdate := false
	if versionName != "" {
		hasUpdate = compareVersionNames(latest.VersionName, versionName) > 0
	} else {
		currentCode, err := strconv.Atoi(versionCode)
		if err != nil {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "version_code 必须为数字", nil)
			return
		}
		hasUpdate = latest.VersionCode > currentCode
	}

	if !hasUpdate {
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "data": gin.H{"has_update": false}, "message": "已是最新版本"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": response.CodeSuccess,
		"data": gin.H{
			"has_update":   true,
			"version_code": latest.VersionCode,
			"version_name": latest.VersionName,
			"download_url": latest.DownloadURL,
			"update_log":   latest.UpdateLog,
			"force_update": latest.ForceUpdate,
		},
		"message": "发现新版本",
	})
}

type reportAppEventRequest struct {
	DeviceID    string `json:"device_id"`
	EventType   string `json:"event_type"`
	VersionCode any    `json:"version_code"`
	Platform    string `json:"platform"`
	Duration    any    `json:"duration"`
}

func (h NativeHandlers) ReportAppEvent(c *gin.Context) {
	var body reportAppEventRequest
	_ = c.ShouldBindJSON(&body)
	if body.DeviceID == "" || body.EventType == "" || body.Platform == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少必要参数: device_id, event_type, platform", nil)
		return
	}
	validEvents := map[string]bool{"app_open": true, "update_check": true, "update_complete": true, "usage_duration": true}
	if !validEvents[body.EventType] {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效的事件类型，支持: app_open, update_check, update_complete, usage_duration", nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "使用记录功能暂不可用", nil)
		return
	}

	var versionCode *int
	if parsed, ok := intFromAny(body.VersionCode); ok {
		versionCode = &parsed
	}
	var versionID *int
	repo := repositories.NewAppRepository(h.DB)
	if versionCode != nil {
		version, err := repo.VersionByCode(c.Request.Context(), body.Platform, *versionCode)
		if err == nil {
			versionID = &version.ID
		} else if err != gorm.ErrRecordNotFound {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, "上报失败", nil)
			return
		}
	}
	var duration *int
	if body.EventType == "usage_duration" {
		if parsed, ok := intFromAny(body.Duration); ok {
			duration = &parsed
		}
	}
	log := domain.AppUsageLog{
		DeviceID:    truncateString(body.DeviceID, 100),
		EventType:   body.EventType,
		VersionCode: versionCode,
		VersionID:   versionID,
		Platform:    truncateString(body.Platform, 20),
		Duration:    duration,
	}
	if err := repo.CreateUsageLog(c.Request.Context(), log); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "上报失败", nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "上报成功"})
}

func compareVersionNames(a, b string) int {
	cleanA := versionCleaner.ReplaceAllString(a, "")
	cleanB := versionCleaner.ReplaceAllString(b, "")
	partsA := strings.Split(cleanA, ".")
	partsB := strings.Split(cleanB, ".")
	maxLen := max(len(partsB), len(partsA))
	for i := range maxLen {
		numA := 0
		if i < len(partsA) {
			numA, _ = strconv.Atoi(partsA[i])
		}
		numB := 0
		if i < len(partsB) {
			numB, _ = strconv.Atoi(partsB[i])
		}
		if numA > numB {
			return 1
		}
		if numA < numB {
			return -1
		}
	}
	return 0
}

func intFromAny(value any) (int, bool) {
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case string:
		if strings.TrimSpace(typed) == "" {
			return 0, false
		}
		parsed, err := strconv.Atoi(typed)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func truncateString(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
