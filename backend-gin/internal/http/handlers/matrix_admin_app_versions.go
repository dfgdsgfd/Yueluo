package handlers

import (
	"database/sql"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
)

func (h NativeHandlers) adminAppVersions(c *gin.Context) {
	path := c.Request.URL.Path
	method := matrixMethod(c)
	switch {
	case path == "/api/admin/app-versions/stats" && method == http.MethodGet:
		h.adminAppVersionStats(c)
	case path == "/api/admin/app-versions/last-form-data" && method == http.MethodGet:
		h.adminLastAppForm(c)
	case path == "/api/admin/app-versions/last-form-data" && method == http.MethodPost:
		h.adminSaveLastAppForm(c)
	case path == "/api/admin/app-versions" && method == http.MethodGet:
		h.adminAppVersionList(c)
	case path == "/api/admin/app-versions" && method == http.MethodPost:
		h.adminAppVersionCreate(c)
	case path == "/api/admin/app-versions" && method == http.MethodDelete:
		h.adminAppVersionBulkDelete(c)
	case strings.HasPrefix(path, "/api/admin/app-versions/") && method == http.MethodGet:
		h.adminAppVersionDetail(c)
	case strings.HasPrefix(path, "/api/admin/app-versions/") && method == http.MethodPut:
		h.adminAppVersionUpdate(c)
	case strings.HasPrefix(path, "/api/admin/app-versions/") && method == http.MethodDelete:
		h.adminAppVersionDelete(c)
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "admin route not found", nil)
	}
}

func (h NativeHandlers) adminAppVersionList(c *gin.Context) {
	page, limit, offset := pageLimit(c, 20)
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.AppVersion{})
	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("app_name LIKE ? OR version_name LIKE ? OR platform LIKE ?", like, like, like)
	}
	if platform := strings.TrimSpace(c.Query("platform")); platform != "" {
		query = query.Where("platform = ?", platform)
	}
	if rawActive := strings.TrimSpace(c.Query("is_active")); rawActive != "" {
		if active, ok := boolFromAny(rawActive); ok {
			query = query.Where("is_active = ?", active)
		}
	}
	var total int64
	if err := query.Count(&total).Error; writeDBError(c, err, "") {
		return
	}
	var rows []domain.AppVersion
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&rows).Error; writeDBError(c, err, "") {
		return
	}
	items := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		items = append(items, appVersionMap(row))
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"items": items,
		"total": total,
		"page":  page,
		"limit": limit,
		"pages": int(math.Ceil(float64(total) / float64(limit))),
	})
}

func (h NativeHandlers) adminAppVersionDetail(c *gin.Context) {
	version, ok := h.findAppVersion(c)
	if !ok {
		return
	}
	writeSuccess(c, matrixMsgOK, appVersionMap(version))
}

func (h NativeHandlers) adminAppVersionCreate(c *gin.Context) {
	body := readBodyMap(c)
	appName := strings.TrimSpace(toString(body["app_name"]))
	versionName := strings.TrimSpace(toString(body["version_name"]))
	platform := strings.TrimSpace(toString(body["platform"]))
	downloadURL := strings.TrimSpace(toString(body["download_url"]))
	versionCode, hasVersionCode := intFromAny(body["version_code"])
	sizeBytes := max(appVersionSizeBytes(body), 0)
	if appName == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "应用名称不能为空", nil)
		return
	}
	if !hasVersionCode {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "版本号不能为空", nil)
		return
	}
	if versionName == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "版本名称不能为空", nil)
		return
	}
	if platform == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "平台不能为空", nil)
		return
	}
	if downloadURL == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "下载地址不能为空", nil)
		return
	}
	forceUpdate, _ := boolFromAny(body["force_update"])
	isActive := true
	if rawActive, exists := body["is_active"]; exists {
		isActive, _ = boolFromAny(rawActive)
	}
	row := domain.AppVersion{
		AppName:     appName,
		VersionCode: versionCode,
		VersionName: versionName,
		Platform:    platform,
		DownloadURL: downloadURL,
		SizeBytes:   sizeBytes,
		UpdateLog:   trimOptionalString(body["update_log"]),
		ForceUpdate: forceUpdate,
		IsActive:    isActive,
	}
	if err := h.DB.WithContext(c.Request.Context()).Create(&row).Error; writeDBError(c, err, "") {
		return
	}
	h.cacheLastAppVersionForm(c, appVersionFormSnapshot(body))
	writeSuccess(c, "创建成功", gin.H{"id": row.ID})
}

func (h NativeHandlers) adminAppVersionUpdate(c *gin.Context) {
	version, ok := h.findAppVersion(c)
	if !ok {
		return
	}
	body := readBodyMap(c)
	updates := map[string]any{}
	if raw, exists := body["app_name"]; exists {
		value := strings.TrimSpace(toString(raw))
		if value == "" {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "应用名称不能为空", nil)
			return
		}
		updates["app_name"] = value
	}
	if raw, exists := body["version_code"]; exists {
		if parsed, ok := intFromAny(raw); ok {
			updates["version_code"] = parsed
		}
	}
	if raw, exists := body["version_name"]; exists {
		value := strings.TrimSpace(toString(raw))
		if value == "" {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "版本名称不能为空", nil)
			return
		}
		updates["version_name"] = value
	}
	if raw, exists := body["platform"]; exists {
		value := strings.TrimSpace(toString(raw))
		if value == "" {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "平台不能为空", nil)
			return
		}
		updates["platform"] = value
	}
	if raw, exists := body["download_url"]; exists {
		value := strings.TrimSpace(toString(raw))
		if value == "" {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "下载地址不能为空", nil)
			return
		}
		updates["download_url"] = value
	}
	if raw, exists := body["size_bytes"]; exists {
		if parsed, ok := int64FromAny(raw); ok && parsed >= 0 {
			updates["size_bytes"] = parsed
		}
	}
	if raw, exists := body["size_mb"]; exists {
		if parsed, ok := appVersionSizeMBBytes(raw); ok && parsed >= 0 {
			updates["size_bytes"] = parsed
		}
	}
	if raw, exists := body["update_log"]; exists {
		updates["update_log"] = trimOptionalString(raw)
	}
	if raw, exists := body["force_update"]; exists {
		updates["force_update"], _ = boolFromAny(raw)
	}
	if raw, exists := body["is_active"]; exists {
		updates["is_active"], _ = boolFromAny(raw)
	}
	if len(updates) > 0 {
		if err := h.DB.WithContext(c.Request.Context()).Model(&domain.AppVersion{}).Where("id = ?", version.ID).Updates(updates).Error; writeDBError(c, err, "") {
			return
		}
	}
	writeSimpleSuccess(c, "更新成功")
}

func (h NativeHandlers) adminAppVersionDelete(c *gin.Context) {
	version, ok := h.findAppVersion(c)
	if !ok {
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Delete(&domain.AppVersion{}, version.ID).Error; writeDBError(c, err, "") {
		return
	}
	writeSimpleSuccess(c, "删除成功")
}

func (h NativeHandlers) adminAppVersionBulkDelete(c *gin.Context) {
	ids := int64SliceFromAny(readBodyMap(c)["ids"])
	if len(ids) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请提供要删除的ID列表", nil)
		return
	}
	if err := h.DB.WithContext(c.Request.Context()).Where("id IN ?", ids).Delete(&domain.AppVersion{}).Error; writeDBError(c, err, "") {
		return
	}
	writeSimpleSuccess(c, "成功删除 "+strconv.Itoa(len(ids))+" 条记录")
}

func (h NativeHandlers) findAppVersion(c *gin.Context) (domain.AppVersion, bool) {
	id, ok := intFromAny(matrixParam(c, "id"))
	if !ok {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "应用版本不存在", nil)
		return domain.AppVersion{}, false
	}
	var version domain.AppVersion
	err := h.DB.WithContext(c.Request.Context()).Where("id = ?", id).Take(&version).Error
	if writeDBError(c, err, "应用版本不存在") {
		return domain.AppVersion{}, false
	}
	return version, true
}

func (h NativeHandlers) adminAppVersionStats(c *gin.Context) {
	var totalUsers int64
	if err := h.DB.WithContext(c.Request.Context()).
		Model(&domain.AppUsageLog{}).
		Where("event_type = ?", "app_open").
		Distinct("device_id").
		Count(&totalUsers).Error; writeDBError(c, err, "") {
		return
	}

	today := time.Now()
	todayStart := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	var todayActiveUsers int64
	if err := h.DB.WithContext(c.Request.Context()).
		Model(&domain.AppUsageLog{}).
		Where("event_type = ? AND created_at >= ?", "app_open", todayStart).
		Distinct("device_id").
		Count(&todayActiveUsers).Error; writeDBError(c, err, "") {
		return
	}

	type versionNameRow struct {
		VersionCode int    `gorm:"column:version_code"`
		VersionName string `gorm:"column:version_name"`
	}
	var versionNames []versionNameRow
	if err := h.DB.WithContext(c.Request.Context()).
		Model(&domain.AppVersion{}).
		Select("version_code, version_name").
		Find(&versionNames).Error; writeDBError(c, err, "") {
		return
	}
	versionNameByCode := map[int]string{}
	for _, row := range versionNames {
		versionNameByCode[row.VersionCode] = row.VersionName
	}

	type versionUpdateRow struct {
		VersionCode int   `gorm:"column:version_code"`
		UpdateCount int64 `gorm:"column:update_count"`
	}
	var versionRows []versionUpdateRow
	if err := h.DB.WithContext(c.Request.Context()).
		Model(&domain.AppUsageLog{}).
		Select("version_code, COUNT(device_id) AS update_count").
		Where("event_type = ? AND version_code IS NOT NULL", "update_complete").
		Group("version_code").
		Scan(&versionRows).Error; writeDBError(c, err, "") {
		return
	}
	sort.SliceStable(versionRows, func(i, j int) bool {
		return versionRows[i].VersionCode > versionRows[j].VersionCode
	})
	versionUpdates := make([]gin.H, 0, len(versionRows))
	for _, row := range versionRows {
		versionName := versionNameByCode[row.VersionCode]
		if versionName == "" {
			versionName = "v" + strconv.Itoa(row.VersionCode)
		}
		versionUpdates = append(versionUpdates, gin.H{
			"version_code": row.VersionCode,
			"version_name": versionName,
			"update_count": row.UpdateCount,
		})
	}

	var durationStats struct {
		TotalSeconds sql.NullInt64   `gorm:"column:total_seconds"`
		AvgSeconds   sql.NullFloat64 `gorm:"column:avg_seconds"`
		ReportCount  int64           `gorm:"column:report_count"`
	}
	if err := h.DB.WithContext(c.Request.Context()).
		Model(&domain.AppUsageLog{}).
		Select("COALESCE(SUM(duration), 0) AS total_seconds, AVG(duration) AS avg_seconds, COUNT(id) AS report_count").
		Where("event_type = ? AND duration IS NOT NULL", "usage_duration").
		Scan(&durationStats).Error; writeDBError(c, err, "") {
		return
	}
	avgSeconds := 0
	if durationStats.AvgSeconds.Valid {
		avgSeconds = int(math.Round(durationStats.AvgSeconds.Float64))
	}

	platformStats := []gin.H{}
	for _, platform := range []string{"android", "android_fast", "ios"} {
		var userCount int64
		if err := h.DB.WithContext(c.Request.Context()).
			Model(&domain.AppUsageLog{}).
			Where("event_type = ? AND platform = ?", "app_open", platform).
			Distinct("device_id").
			Count(&userCount).Error; writeDBError(c, err, "") {
			return
		}
		if userCount > 0 {
			platformStats = append(platformStats, gin.H{"platform": platform, "user_count": userCount})
		}
	}

	writeSuccess(c, matrixMsgOK, gin.H{
		"total_users":        totalUsers,
		"today_active_users": todayActiveUsers,
		"version_updates":    versionUpdates,
		"usage_duration": gin.H{
			"total_seconds": nullInt64OrZero(durationStats.TotalSeconds),
			"avg_seconds":   avgSeconds,
			"report_count":  durationStats.ReportCount,
		},
		"platform_stats": platformStats,
	})
}

func (h NativeHandlers) adminLastAppForm(c *gin.Context) {
	if h.Cache != nil {
		if value, ok := h.Cache.Get("app_version:last_form_data"); ok {
			writeSuccess(c, matrixMsgOK, value)
			return
		}
	}
	var cached gin.H
	if h.Redis != nil && h.Redis.GetJSON(c.Request.Context(), "app_version:last_form_data", &cached) {
		if h.Cache != nil {
			h.Cache.Set("app_version:last_form_data", cached, 30*24*time.Hour)
		}
		writeSuccess(c, matrixMsgOK, cached)
		return
	}
	writeSuccess(c, matrixMsgOK, emptyAppVersionFormSnapshot())
}

func (h NativeHandlers) adminSaveLastAppForm(c *gin.Context) {
	body := readBodyMap(c)
	h.cacheLastAppVersionForm(c, appVersionFormSnapshot(body))
	writeSimpleSuccess(c, "保存成功")
}

func appVersionSizeBytes(body map[string]any) int64 {
	if parsed, ok := appVersionSizeMBBytes(body["size_mb"]); ok {
		return parsed
	}
	if parsed, ok := int64FromAny(body["size_bytes"]); ok {
		return parsed
	}
	return 0
}

func appVersionSizeMBBytes(value any) (int64, bool) {
	mb, ok := float64FromAny(value)
	if !ok {
		return 0, false
	}
	return int64(math.Round(mb * 1024 * 1024)), true
}
