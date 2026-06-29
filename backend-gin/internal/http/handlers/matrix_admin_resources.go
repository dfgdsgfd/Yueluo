package handlers

import (
	"database/sql"
	"maps"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/security"
	"yuem-go/backend-gin/internal/services"
)

func adminSegments(c *gin.Context) []string {
	path := strings.TrimPrefix(c.Request.URL.Path, "/api/admin/")
	if path == c.Request.URL.Path {
		return nil
	}
	parts := strings.Split(strings.Trim(path, "/"), "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func adminResourceFromSegments(segments []string) (adminResource, string) {
	if len(segments) == 0 {
		return adminResource{}, ""
	}
	if segments[0] == "recommendation" && len(segments) >= 2 {
		resource := adminResources[segments[1]]
		id := ""
		if len(segments) >= 3 {
			id = segments[2]
		}
		return resource, id
	}
	resource := adminResources[segments[0]]
	id := ""
	if len(segments) >= 2 {
		id = segments[1]
	}
	return resource, id
}

func adminFilters(specs ...string) map[string]adminFilter {
	out := map[string]adminFilter{}
	for _, spec := range specs {
		parts := strings.SplitN(spec, ":", 3)
		if len(parts) != 3 {
			continue
		}
		out[parts[0]] = adminFilter{Mode: parts[1], Column: parts[2]}
	}
	return out
}

func adminSortFields(specs ...string) map[string]string {
	out := map[string]string{}
	for _, spec := range specs {
		parts := strings.SplitN(spec, ":", 2)
		if len(parts) != 2 {
			continue
		}
		out[parts[0]] = parts[1]
	}
	return out
}

func (h NativeHandlers) adminResourceQuery(c *gin.Context, resource adminResource) *gorm.DB {
	table := resource.Table
	if resource.Alias != "" {
		table += " " + resource.Alias
	}
	query := h.DB.WithContext(c.Request.Context()).Table(table)
	for _, join := range resource.Joins {
		query = query.Joins(join)
	}
	for _, where := range resource.BaseWhere {
		query = query.Where(where.Clause, where.Args...)
	}
	return query
}

func adminSelect(resource adminResource) string {
	if strings.TrimSpace(resource.Select) != "" {
		return resource.Select
	}
	if resource.Alias != "" {
		return resource.Alias + ".*"
	}
	return "*"
}

func adminIDColumn(resource adminResource) string {
	if resource.Alias != "" {
		return resource.Alias + ".id"
	}
	return "id"
}

func adminOrder(c *gin.Context, resource adminResource) string {
	order := resource.DefaultOrder
	if order == "" {
		order = adminIDColumn(resource) + " DESC"
	}
	sortField := c.Query("sortField")
	if sortField == "" {
		return order
	}
	column, ok := resource.SortFields[sortField]
	if !ok || column == "" {
		return order
	}
	sortOrder := strings.ToUpper(c.DefaultQuery("sortOrder", "DESC"))
	if sortOrder != "ASC" {
		sortOrder = "DESC"
	}
	return column + " " + sortOrder
}

func adminPageLimit(c *gin.Context, resource adminResource) (int, int, int) {
	page := positiveIntQuery(c, "page", 1)
	limitKey := firstNonEmpty(resource.PageSizeKey, "limit")
	limit := positiveIntQuery(c, limitKey, 20)
	if resource.PageSizeKey != "" && c.Query(resource.PageSizeKey) == "" {
		limit = positiveIntQuery(c, "limit", 20)
	}
	if limit > 200 {
		limit = 200
	}
	return page, limit, (page - 1) * limit
}

func applySearch(query *gorm.DB, fields []string, keyword string) *gorm.DB {
	clauses := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields))
	for _, field := range fields {
		clauses = append(clauses, field+" LIKE ?")
		args = append(args, "%"+keyword+"%")
	}
	return query.Where(strings.Join(clauses, " OR "), args...)
}

func applyAdminFilters(c *gin.Context, query *gorm.DB, resource adminResource) *gorm.DB {
	for key, values := range c.Request.URL.Query() {
		if len(values) == 0 || values[0] == "" {
			continue
		}
		switch key {
		case "page", "limit", "pageSize", "keyword", "search", "sortField", "sortOrder":
			continue
		}
		filter, ok := resource.Filters[key]
		if !ok || filter.Column == "" {
			continue
		}
		value := strings.TrimSpace(values[0])
		switch filter.Mode {
		case "like":
			query = query.Where(filter.Column+" LIKE ?", "%"+value+"%")
		case "int":
			if parsed, ok := intFromAny(value); ok {
				query = query.Where(filter.Column+" = ?", parsed)
			}
		case "int64":
			if parsed, ok := int64FromAny(value); ok {
				query = query.Where(filter.Column+" = ?", parsed)
			}
		case "bool":
			if parsed, ok := boolFromAny(value); ok {
				query = query.Where(filter.Column+" = ?", parsed)
			}
		case "nullint":
			if strings.EqualFold(value, "null") {
				query = query.Where(filter.Column + " IS NULL")
				continue
			}
			if parsed, ok := intFromAny(value); ok {
				query = query.Where(filter.Column+" = ?", parsed)
			}
		default:
			query = query.Where(filter.Column+" = ?", value)
		}
	}
	return query
}

func adminWritableColumns(resource adminResource) map[string]bool {
	switch resource.Table {
	case "posts":
		return map[string]bool{
			"user_id":              true,
			"title":                true,
			"content":              true,
			"category_id":          true,
			"type":                 true,
			"view_count":           true,
			"like_count":           true,
			"collect_count":        true,
			"comment_count":        true,
			"created_at":           true,
			"is_draft":             true,
			"visibility":           true,
			"public_access_exempt": true,
			"quality_level":        true,
			"quality_marked_at":    true,
			"quality_reward":       true,
		}
	default:
		return nil
	}
}

func adminPreprocessCreate(resource adminResource, body map[string]any) (gin.H, error) {
	extra := gin.H{}
	now := time.Now()
	if adminResourceHasColumn(resource, "created_at") {
		if _, ok := body["created_at"]; !ok {
			body["created_at"] = now
		}
	}
	if adminResourceHasColumn(resource, "updated_at") {
		if _, ok := body["updated_at"]; !ok {
			body["updated_at"] = now
		}
	}
	if resource.Table == "admin" {
		if password := toString(body["password"]); password != "" {
			passwordHash, err := security.HashPassword(password)
			if err != nil {
				return nil, err
			}
			body["password"] = passwordHash
		}
	}
	if resource.Table == "users" {
		password := toString(body["password"])
		if password == "" {
			password = "123456"
		}
		passwordHash, err := security.HashPassword(password)
		if err != nil {
			return nil, err
		}
		body["password"] = passwordHash
		if _, ok := body["is_active"]; !ok {
			body["is_active"] = true
		}
	}
	if resource.Table == "open_apis" {
		rawKey := "oapi_" + randomHex(32)
		body["api_key"] = sha256Hex(rawKey)
		body["api_key_prefix"] = rawKey[:12]
		extra["api_key"] = rawKey
	}
	if resource.Table == "categories" {
		if value, ok := body["translations"]; ok {
			body["translations"] = normalizedCategoryTranslations(value, toString(body["name"]), toString(body["category_title"]))
		}
	}
	adminPreprocessAnnouncementSchedule(resource, body, now)
	return extra, nil
}

func adminPreprocessUpdate(resource adminResource, body map[string]any) error {
	now := time.Now()
	if (resource.Table == "admin" || resource.Table == "users") && toString(body["password"]) != "" {
		passwordHash, err := security.HashPassword(toString(body["password"]))
		if err != nil {
			return err
		}
		body["password"] = passwordHash
	}
	if adminResourceHasColumn(resource, "updated_at") {
		if _, ok := body["updated_at"]; !ok {
			body["updated_at"] = now
		}
	}
	if resource.Table == "categories" {
		if value, ok := body["translations"]; ok {
			body["translations"] = normalizedCategoryTranslations(value, toString(body["name"]), toString(body["category_title"]))
		}
	}
	adminPreprocessAnnouncementSchedule(resource, body, now)
	return nil
}

func normalizedCategoryTranslations(value any, name, legacyTitle string) any {
	input := map[string]string{}
	switch typed := jsonValueAny(value).(type) {
	case map[string]any:
		for key, item := range typed {
			input[key] = toString(item)
		}
	case map[string]string:
		input = typed
	}
	return jsonBytes(completeCategoryTranslations(input, name, legacyTitle))
}

func adminPreprocessAnnouncementSchedule(resource adminResource, body map[string]any, now time.Time) {
	if resource.Table != "announcements" {
		return
	}
	if raw, exists := body["duration_days"]; exists {
		delete(body, "duration_days")
		if days, ok := intFromAny(raw); ok && days > 0 {
			body["expires_at"] = now.AddDate(0, 0, days)
		} else {
			body["expires_at"] = nil
		}
	}
	if published, exists := boolFromAny(body["is_published"]); exists {
		if published {
			if value, ok := body["published_at"]; !ok || value == nil || strings.TrimSpace(toString(value)) == "" {
				body["published_at"] = now
			}
			return
		}
		body["published_at"] = nil
		body["expires_at"] = nil
	}
}

func adminResourceHasColumn(resource adminResource, column string) bool {
	needle := "." + column
	for _, value := range resource.SortFields {
		if strings.EqualFold(value, column) || strings.HasSuffix(strings.ToLower(value), needle) {
			return true
		}
	}
	if strings.Contains(strings.ToLower(resource.DefaultOrder), column) {
		return true
	}
	if strings.Contains(strings.ToLower(resource.Select), column) {
		return true
	}
	return false
}

func (h NativeHandlers) normalizeRows(rows []map[string]any, resource adminResource) []gin.H {
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, h.normalizeRow(row, resource))
	}
	return out
}

func (h NativeHandlers) normalizeRow(row map[string]any, resource adminResource) gin.H {
	out := gin.H{}
	maps.Copy(out, row)
	for _, key := range resource.HiddenFields {
		delete(out, key)
	}
	if _, ok := out["category_id_value"]; ok {
		out["category"] = gin.H{"id": out["category_id_value"], "name": out["category_name"]}
		delete(out, "category_id_value")
		delete(out, "category_name")
	}
	switch resource.Name {
	case "posts":
		images := []any{}
		postType, _ := intFromAny(out["type"])
		if (postType == 2 || postType == 3) && toString(out["video_url"]) != "" {
			images = append(images, h.signFileURL(toString(out["video_url"])))
		} else if toString(out["first_image_url"]) != "" {
			images = append(images, h.signFileURL(toString(out["first_image_url"])))
		}
		out["images"] = images
		if _, exists := out["tags"]; !exists {
			out["tags"] = []gin.H{}
		}
		delete(out, "first_image_url")
	case "feedback":
		out["user"] = gin.H{"id": out["user_id_value"], "user_id": out["user_display_id"], "nickname": out["user_nickname"], "avatar": h.signAnyFileURL(out["user_avatar"])}
		delete(out, "user_id_value")
		delete(out, "user_display_id")
		delete(out, "user_nickname")
		delete(out, "user_avatar")
	case "reports":
		out["reporter"] = gin.H{"id": out["reporter_id_value"], "user_id": out["reporter_user_id"], "nickname": out["reporter_nickname"], "avatar": h.signAnyFileURL(out["reporter_avatar"])}
		delete(out, "reporter_id_value")
		delete(out, "reporter_user_id")
		delete(out, "reporter_nickname")
		delete(out, "reporter_avatar")
	case "audit":
		if value, exists := out["audit_result"]; exists {
			out["audit_result"] = h.signVerificationAuditResult(jsonValueAny(value))
		}
	case "ai-moderation-logs":
		for _, key := range []string{"categories", "model_result", "metadata"} {
			if value, exists := out[key]; exists {
				out[key] = jsonValueAny(value)
			}
		}
	}
	for key, value := range out {
		if adminMediaURLColumn(key) {
			out[key] = h.signAnyFileURL(value)
		}
	}
	return out
}

func writeAdminList(c *gin.Context, resource adminResource, rows []gin.H, page int, limit int, total int64) {
	message := firstNonEmpty(resource.ListMessage, matrixMsgOK)
	pagination := matrixPagination(page, limit, total)
	switch resource.ListShape {
	case adminListTopLevel:
		c.JSON(http.StatusOK, gin.H{
			"code":       response.CodeSuccess,
			"message":    message,
			"data":       rows,
			"pagination": gin.H{"page": page, "limit": limit, "total": total, "totalPages": pagination["pages"]},
		})
	case adminListItems:
		writeSuccess(c, message, gin.H{"items": rows, "total": total, "page": page, "pageSize": limit})
	case adminListAudit:
		writeSuccess(c, message, gin.H{"data": rows, "total": total, "page": page, "limit": limit})
	case adminListReports:
		writeSuccess(c, message, gin.H{"list": rows, "pagination": gin.H{"total": total, "page": page, "pageSize": limit, "pages": pagination["pages"]}})
	default:
		writeSuccess(c, message, gin.H{"data": rows, "pagination": pagination})
	}
}

func appVersionMap(version domain.AppVersion) gin.H {
	return gin.H{
		"id":           version.ID,
		"app_name":     version.AppName,
		"version_code": version.VersionCode,
		"version_name": version.VersionName,
		"platform":     version.Platform,
		"download_url": version.DownloadURL,
		"size_bytes":   version.SizeBytes,
		"size_mb":      appVersionSizeMB(version.SizeBytes),
		"update_log":   version.UpdateLog,
		"force_update": version.ForceUpdate,
		"is_active":    version.IsActive,
		"created_at":   version.CreatedAt,
		"updated_at":   version.UpdatedAt,
	}
}

func trimOptionalString(value any) *string {
	raw, exists := value.(string)
	if !exists {
		text := toString(value)
		if text == "" {
			return nil
		}
		trimmed := strings.TrimSpace(text)
		return &trimmed
	}
	if raw == "" {
		return nil
	}
	trimmed := strings.TrimSpace(raw)
	return &trimmed
}

func emptyAppVersionFormSnapshot() gin.H {
	return gin.H{
		"app_name":     "",
		"version_code": "",
		"version_name": "",
		"platform":     "",
		"download_url": "",
		"size_mb":      "",
		"update_log":   "",
		"force_update": false,
		"is_active":    true,
	}
}

func appVersionFormSnapshot(body map[string]any) gin.H {
	value := emptyAppVersionFormSnapshot()
	for _, key := range []string{"app_name", "version_name", "platform", "download_url", "update_log"} {
		value[key] = strings.TrimSpace(toString(body[key]))
	}
	if raw, exists := body["size_mb"]; exists {
		if parsed, ok := float64FromAny(raw); ok && parsed >= 0 {
			value["size_mb"] = parsed
		} else {
			value["size_mb"] = strings.TrimSpace(toString(raw))
		}
	}
	if raw, exists := body["version_code"]; exists {
		if parsed, ok := intFromAny(raw); ok {
			value["version_code"] = parsed
		} else {
			value["version_code"] = strings.TrimSpace(toString(raw))
		}
	}
	if raw, exists := body["force_update"]; exists {
		value["force_update"], _ = boolFromAny(raw)
	}
	if raw, exists := body["is_active"]; exists {
		value["is_active"], _ = boolFromAny(raw)
	}
	return value
}

func (h NativeHandlers) cacheLastAppVersionForm(c *gin.Context, value gin.H) {
	if h.Cache != nil {
		h.Cache.Set("app_version:last_form_data", value, 30*24*time.Hour)
	}
	if h.Redis != nil {
		h.Redis.Set(c.Request.Context(), "app_version:last_form_data", value, 30*24*time.Hour)
	}
}

func appVersionSizeMB(sizeBytes int64) float64 {
	if sizeBytes <= 0 {
		return 0
	}
	mb := float64(sizeBytes) / 1024 / 1024
	rounded, _ := strconv.ParseFloat(strconv.FormatFloat(mb, 'f', 2, 64), 64)
	return rounded
}

func settingType(value any) string {
	switch value.(type) {
	case bool:
		return "boolean"
	case int, int64, float64:
		return "number"
	default:
		return "text"
	}
}

func settingTypeForKey(key string, value any) string {
	if key == "video_center_account_cutoff" {
		return "datetime"
	}
	switch key {
	case "onboarding_custom_fields", "onboarding_points_intro_detail", "hidden_watermark_extract_user_ids", "hidden_watermark_extract_usernames", "oauth2_app_callback_urls":
		return "textarea"
	}
	if strings.HasPrefix(key, "app_download_") && (strings.HasSuffix(key, "_download_url") || strings.HasSuffix(key, "_release_notes")) {
		return "textarea"
	}
	if strings.HasPrefix(key, "app_download_") && strings.HasSuffix(key, "_size_bytes") {
		return "number"
	}
	return settingType(value)
}

func settingLabel(key string) string {
	switch key {
	case services.SiteTitleSetting:
		return "网站标题"
	case services.SiteDescriptionSetting:
		return "网站简介"
	case services.SiteAvatarURLSetting:
		return "网站头像 URL"
	case "onboarding_enabled":
		return "开启引导流程"
	case "onboarding_allow_skip":
		return "允许跳过引导"
	case "onboarding_interest_options":
		return "引导兴趣选项"
	case "onboarding_custom_fields":
		return "引导自定义字段"
	case "onboarding_avatar_enabled":
		return "显示头像设置"
	case "onboarding_avatar_required":
		return "头像必填"
	case "onboarding_background_enabled":
		return "显示背景设置"
	case "onboarding_background_required":
		return "背景必填"
	case "onboarding_name_enabled":
		return "显示名称设置"
	case "onboarding_name_required":
		return "名称必填"
	case "onboarding_signature_enabled":
		return "显示签名设置"
	case "onboarding_signature_required":
		return "签名必填"
	case "onboarding_interests_enabled":
		return "显示兴趣爱好"
	case "onboarding_interests_required":
		return "兴趣爱好必填"
	case "onboarding_min_interests":
		return "至少选择兴趣数"
	case "onboarding_points_intro_title":
		return "积分说明标题"
	case "onboarding_points_intro_summary":
		return "积分说明摘要"
	case "onboarding_points_intro_detail":
		return "积分说明详情"
	case "onboarding_result_title":
		return "完成弹窗标题"
	case "onboarding_result_saved_text":
		return "无积分保存文案"
	case "onboarding_points_wallet_label":
		return "积分入口按钮文案"
	case "onboarding_points_wallet_url":
		return "积分入口链接"
	case "paid_content_balance_max_price":
		return "月币收费上限"
	case "paid_content_points_max_price":
		return "积分收费上限"
	case services.FileRecycleRetentionDaysKey:
		return "文件回收站保留时间（天）"
	case services.FileRecycleCleanupIntervalHoursKey:
		return "文件回收站清理间隔（小时）"
	case "notification_interaction_suppression_enabled":
		return "互动通知抑制"
	case "notification_interaction_suppression_window_seconds":
		return "互动通知抑制窗口（秒）"
	case "notification_interaction_suppression_threshold":
		return "互动通知抑制阈值"
	default:
		return key
	}
}

func settingHint(key string) string {
	switch key {
	case services.SiteTitleSetting:
		return "首页和浏览器标题使用的站点名称。"
	case services.SiteDescriptionSetting:
		return "首页顶部和搜索引擎摘要使用的站点简介。"
	case services.SiteAvatarURLSetting:
		return "首页展示的站点头像，可填写 /api/file/... 站内路径或 http/https 图片地址。"
	case "onboarding_enabled":
		return "开启后，数据库未记录完成引导的登录用户会看到引导弹窗。"
	case "onboarding_allow_skip":
		return "关闭后，用户必须完成引导流程，不能直接跳过。"
	case "onboarding_interest_options":
		return "在引导设置页用列表添加，不需要手写 JSON。"
	case "onboarding_min_interests":
		return "仅在显示兴趣爱好且兴趣必填时生效。"
	case "onboarding_points_intro_detail":
		return "用户完成引导后点击查看详情时展示。"
	case "onboarding_points_wallet_url":
		return "完成弹窗中的积分入口跳转地址，可填写站内路径或完整 URL。"
	case "app_download_android_download_url":
		return "安卓安装包下载链接，可填写媒体库上传后的 /api/file/attachments/... 或完整 URL。"
	case "app_download_android_fast_download_url":
		return "安卓极速版安装包下载链接，可填写媒体库上传后的 /api/file/attachments/... 或完整 URL。"
	case "app_download_ios_download_url":
		return "iOS 下载链接，可填写 App Store、TestFlight、企业签名或描述文件地址。"
	case "app_download_android_version_code", "app_download_android_fast_version_code", "app_download_ios_version_code":
		return "用于展示和客户端比较的数字版本号。"
	case "app_download_android_size_bytes", "app_download_android_fast_size_bytes", "app_download_ios_size_bytes":
		return "后台按 MB 输入，保存时自动换算为字节；填写 0 时不展示。"
	case "app_download_android_size_label", "app_download_android_fast_size_label", "app_download_ios_size_label":
		return "可选展示文案；填写后优先于字节数显示，例如 ≤ 1 MB。"
	case "oauth2_app_callback_urls":
		return "App OAuth 唤起白名单，每行一个完整 callback，例如 xsewebfast://auth-return。后端按 scheme、host、path 精准匹配。"
	case "paid_content_balance_max_price":
		return "作者发布月币付费图片时允许设置的最高解锁价格。"
	case "paid_content_points_max_price":
		return "作者发布积分付费图片时允许设置的最高解锁价格。"
	case services.FileRecycleRetentionDaysKey:
		return "文件被移入回收站后自动清理前保留的天数，默认 30 天。"
	case services.FileRecycleCleanupIntervalHoursKey:
		return "后台扫描过期回收文件的间隔，默认每 24 小时执行一次。"
	case "notification_interaction_suppression_enabled":
		return "开启后，同一用户在时间窗口内对同一作者的同类点赞、评论点赞或收藏通知超过阈值时将静默抑制。"
	case "notification_interaction_suppression_window_seconds":
		return "滚动统计窗口，默认 600 秒，允许 60 到 86400 秒。"
	case "notification_interaction_suppression_threshold":
		return "窗口内允许发送的同类互动通知数量，默认 3，允许 1 到 100。"
	default:
		return ""
	}
}

func flattenSettings(body map[string]any) map[string]any {
	out := map[string]any{}
	var walk func(map[string]any)
	walk = func(m map[string]any) {
		for key, value := range m {
			if !isSafeColumn(key) {
				continue
			}
			if services.LocalizedSettingKeys[key] {
				out[key] = value
				continue
			}
			if nested, ok := value.(map[string]any); ok {
				if v, exists := nested["value"]; exists {
					out[key] = v
					continue
				}
				walk(nested)
				continue
			}
			out[key] = value
		}
	}
	walk(body)
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func isSafeColumn(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		return false
	}
	return true
}

func safeColumn(value string) string {
	if !isSafeColumn(value) {
		return "id"
	}
	return value
}

func adminUserIDFromPath(c *gin.Context) int64 {
	if id, ok := int64FromAny(matrixParam(c, "userId")); ok {
		return id
	}
	segments := adminSegments(c)
	for i, segment := range segments {
		if segment == "users" && i+1 < len(segments) {
			id, _ := strconv.ParseInt(segments[i+1], 10, 64)
			return id
		}
	}
	return 0
}

func defaultNotificationTemplates() []gin.H {
	return []gin.H{
		{"template_key": "system_notification", "name": "系统通知", "type": "system", "content": "{{content}}"},
		{"template_key": "email_test", "name": "邮件测试", "type": "email", "content": "{{content}}"},
	}
}

func positiveFloat(value float64) float64 {
	if value > 0 {
		return value
	}
	return 0
}

func nullInt64OrZero(value sql.NullInt64) int64 {
	if value.Valid {
		return value.Int64
	}
	return 0
}
