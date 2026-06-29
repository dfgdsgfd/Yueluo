package handlers

import (
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) AdminAccessLogAnalytics(c *gin.Context) {
	if _, ok := h.requireMatrixAdmin(c); !ok || !h.requireDB(c) {
		return
	}
	window := parseAdminLogRange(c)
	series, err := h.accessLogSeries(c, window)
	if writeDBError(c, err, "") {
		return
	}
	securitySeries, err := h.securityLogSeries(c, window)
	if writeDBError(c, err, "") {
		return
	}
	merged := mergeLogSeries(window, series, securitySeries)
	totals := aggregateLogSeries(merged)
	limit := min(positiveIntQuery(c, "rankingLimit", 10), 50)
	if limit <= 0 {
		limit = 10
	}
	rankings := gin.H{
		"paths":             h.accessRanking(c, "path", "", nil, limit),
		"posts":             h.accessPostRanking(c, limit),
		"users":             h.accessRanking(c, "user_display_id", "user_id IS NOT NULL", nil, limit),
		"ips":               h.accessRanking(c, "ip", "", nil, limit),
		"countries":         h.accessCountryRanking(c, limit),
		"languages":         h.accessRanking(c, "browser_language", "", nil, limit),
		"behaviors":         h.accessRanking(c, "behavior", "", nil, limit),
		"device_uas":        h.accessUserAgentRanking(c, "device_ua", limit),
		"devices":           h.accessUserAgentRanking(c, "device", limit),
		"browsers":          h.accessUserAgentRanking(c, "browser", limit),
		"operating_systems": h.accessUserAgentRanking(c, "os", limit),
	}
	writeSuccess(c, matrixMsgOK, gin.H{
		"range": gin.H{
			"start":  window.Start.UTC().Format(time.RFC3339),
			"end":    window.End.UTC().Format(time.RFC3339),
			"bucket": window.Bucket,
		},
		"totals":   totals,
		"series":   merged,
		"rankings": rankings,
		"status":   h.auditLogRuntimeStatus(),
	})
}

func applyAccessLogFilters(c *gin.Context, query *gorm.DB) *gorm.DB {
	window := parseAdminLogRange(c)
	query = query.Where("created_at >= ? AND created_at < ?", window.Start, window.End)
	if behavior := strings.TrimSpace(c.Query("behavior")); behavior != "" {
		query = query.Where("behavior = ?", behavior)
	}
	if strings.EqualFold(strings.TrimSpace(c.Query("behavior_group")), "operation") {
		query = query.Where("behavior IN ?", operationAccessBehaviors())
		query = query.Where("path NOT LIKE ?", "/api/admin/%")
	}
	if userID := strings.TrimSpace(c.Query("user_id")); userID != "" {
		query = query.Where("user_display_id = ? OR user_id = ?", userID, int64FromString(userID))
	}
	if ip := strings.TrimSpace(c.Query("ip")); ip != "" {
		query = query.Where("ip = ?", ip)
	}
	if predicate, args, ok := accessLogKeywordCondition(c.Query("keyword")); ok {
		query = query.Where(predicate, args...)
	}
	query = applyVisitorFilter(c, query, "principal_type")
	return query
}

func applyVisitorFilter(c *gin.Context, query *gorm.DB, column string) *gorm.DB {
	switch strings.ToLower(strings.TrimSpace(c.Query("visitor"))) {
	case "exclude":
		query = query.Where(column+" IS DISTINCT FROM ?", "guest")
	case "only":
		query = query.Where(column+" = ?", "guest")
	}
	return query
}

func applySecurityAuditLogFilters(c *gin.Context, query *gorm.DB) *gorm.DB {
	window := parseAdminLogRange(c)
	query = query.Where("created_at >= ? AND created_at < ?", window.Start, window.End)
	if category := strings.TrimSpace(c.Query("category")); category != "" {
		query = query.Where("category = ?", category)
	}
	if excludeCategory := strings.TrimSpace(c.Query("exclude_category")); excludeCategory != "" {
		query = query.Where("category <> ?", excludeCategory)
	}
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		query = query.Where("action = ?", action)
	}
	if outcome := strings.TrimSpace(c.Query("outcome")); outcome != "" {
		query = query.Where("outcome = ?", outcome)
	}
	if predicate, args, ok := securityAuditLogKeywordCondition(c.Query("keyword")); ok {
		query = query.Where(predicate, args...)
	}
	return query
}

func accessLogKeywordCondition(keyword string) (string, []any, bool) {
	like, ok := logKeywordLike(keyword)
	if !ok {
		return "", nil, false
	}
	return "path LIKE ? OR user_display_id LIKE ? OR ip LIKE ? OR request_id LIKE ? OR user_agent LIKE ? OR browser_language LIKE ? OR behavior LIKE ? OR target_type LIKE ?",
		repeatAny(like, 8),
		true
}

func securityAuditLogKeywordCondition(keyword string) (string, []any, bool) {
	like, ok := logKeywordLike(keyword)
	if !ok {
		return "", nil, false
	}
	return "path LIKE ? OR actor_display_id LIKE ? OR ip LIKE ? OR request_id LIKE ? OR reason_code LIKE ? OR user_agent LIKE ? OR browser_language LIKE ? OR category LIKE ? OR action LIKE ? OR outcome LIKE ? OR actor_type LIKE ?",
		repeatAny(like, 11),
		true
}

func logKeywordLike(keyword string) (string, bool) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return "", false
	}
	return "%" + keyword + "%", true
}

func repeatAny(value any, count int) []any {
	if count <= 0 {
		return nil
	}
	out := make([]any, count)
	for i := range out {
		out[i] = value
	}
	return out
}

type accessLogSeriesRow struct {
	Bucket      time.Time `gorm:"column:bucket"`
	PV          int64     `gorm:"column:pv"`
	ActiveUsers int64     `gorm:"column:active_users"`
	UniqueIPs   int64     `gorm:"column:unique_ips"`
	PostViews   int64     `gorm:"column:post_views"`
}

type securityLogSeriesRow struct {
	Bucket         time.Time `gorm:"column:bucket"`
	SecurityEvents int64     `gorm:"column:security_events"`
}

type rankingRow struct {
	Label string `gorm:"column:label"`
	Title string `gorm:"column:title"`
	Count int64  `gorm:"column:count"`
}

type userAgentRankingRow struct {
	UserAgent string `gorm:"column:user_agent"`
	Count     int64  `gorm:"column:count"`
}

func (h NativeHandlers) accessLogSeries(c *gin.Context, window adminLogRange) ([]accessLogSeriesRow, error) {
	rows := []accessLogSeriesRow{}
	query := h.DB.WithContext(c.Request.Context()).
		Table("access_logs").
		Select(adminLogBucketExpr(h.DB, "created_at", window.Bucket)+` AS bucket,
			COUNT(*) AS pv,
			COUNT(DISTINCT user_id) AS active_users,
			COUNT(DISTINCT ip) AS unique_ips,
			COALESCE(SUM(CASE WHEN behavior = 'post_view' THEN 1 ELSE 0 END), 0) AS post_views`).
		Where("created_at >= ? AND created_at < ?", window.Start, window.End)
	if behavior := strings.TrimSpace(c.Query("behavior")); behavior != "" {
		query = query.Where("behavior = ?", behavior)
	}
	if strings.EqualFold(strings.TrimSpace(c.Query("behavior_group")), "operation") {
		query = query.Where("behavior IN ?", operationAccessBehaviors())
		query = query.Where("path NOT LIKE ?", "/api/admin/%")
	}
	query = applyVisitorFilter(c, query, "principal_type")
	err := query.Group("bucket").Order("bucket ASC").Scan(&rows).Error
	return rows, err
}

func (h NativeHandlers) securityLogSeries(c *gin.Context, window adminLogRange) ([]securityLogSeriesRow, error) {
	rows := []securityLogSeriesRow{}
	query := h.DB.WithContext(c.Request.Context()).
		Table("security_audit_logs").
		Select(adminLogBucketExpr(h.DB, "created_at", window.Bucket)+" AS bucket, COUNT(*) AS security_events").
		Where("created_at >= ? AND created_at < ?", window.Start, window.End)
	if category := strings.TrimSpace(c.Query("category")); category != "" {
		query = query.Where("category = ?", category)
	}
	err := query.Group("bucket").Order("bucket ASC").Scan(&rows).Error
	return rows, err
}

func (h NativeHandlers) accessRanking(c *gin.Context, field string, extra string, args []any, limit int) []gin.H {
	window := parseAdminLogRange(c)
	expr := rankingExpr(h.DB, field)
	if expr == "" {
		return []gin.H{}
	}
	query := h.DB.WithContext(c.Request.Context()).
		Table("access_logs").
		Select(expr+" AS label, COUNT(*) AS count").
		Where("created_at >= ? AND created_at < ?", window.Start, window.End)
	if behavior := strings.TrimSpace(c.Query("behavior")); behavior != "" {
		query = query.Where("behavior = ?", behavior)
	}
	if strings.EqualFold(strings.TrimSpace(c.Query("behavior_group")), "operation") {
		query = query.Where("behavior IN ?", operationAccessBehaviors())
		query = query.Where("path NOT LIKE ?", "/api/admin/%")
	}
	if extra != "" {
		query = query.Where(extra, args...)
	}
	query = applyVisitorFilter(c, query, "principal_type")
	query = query.Where(expr + " IS NOT NULL AND " + expr + " <> ''")
	rows := []rankingRow{}
	err := query.Group("label").Order("count DESC").Limit(limit).Scan(&rows).Error
	if err != nil {
		return []gin.H{}
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"label": row.Label, "key": row.Label, "count": row.Count})
	}
	return out
}

func (h NativeHandlers) accessPostRanking(c *gin.Context, limit int) []gin.H {
	window := parseAdminLogRange(c)
	labelExpr := rankingCastExpr(h.DB, "al.target_id")
	query := h.DB.WithContext(c.Request.Context()).
		Table("access_logs AS al").
		Select(labelExpr+" AS label, COALESCE(MAX(p.title), '') AS title, COUNT(*) AS count").
		Joins("LEFT JOIN posts p ON p.id = al.target_id").
		Where("al.created_at >= ? AND al.created_at < ?", window.Start, window.End).
		Where("al.target_type = ?", "post")
	if behavior := strings.TrimSpace(c.Query("behavior")); behavior != "" {
		query = query.Where("al.behavior = ?", behavior)
	}
	if strings.EqualFold(strings.TrimSpace(c.Query("behavior_group")), "operation") {
		query = query.Where("al.behavior IN ?", operationAccessBehaviors())
		query = query.Where("al.path NOT LIKE ?", "/api/admin/%")
	}
	query = applyVisitorFilter(c, query, "al.principal_type")
	query = query.Where("al.target_id IS NOT NULL")
	rows := []rankingRow{}
	err := query.Group("label").Order("count DESC").Limit(limit).Scan(&rows).Error
	if err != nil {
		return []gin.H{}
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		item := gin.H{"label": row.Label, "key": row.Label, "count": row.Count}
		if title := strings.TrimSpace(row.Title); title != "" {
			item["title"] = title
			item["post_title"] = title
		}
		out = append(out, item)
	}
	return out
}

func (h NativeHandlers) accessUserAgentRanking(c *gin.Context, dimension string, limit int) []gin.H {
	if dimension == "os" {
		return h.accessOperatingSystemRanking(c, limit)
	}
	window := parseAdminLogRange(c)
	query := h.DB.WithContext(c.Request.Context()).
		Table("access_logs").
		Select("user_agent, COUNT(*) AS count").
		Where("created_at >= ? AND created_at < ?", window.Start, window.End).
		Where("user_agent IS NOT NULL AND user_agent <> ''")
	if behavior := strings.TrimSpace(c.Query("behavior")); behavior != "" {
		query = query.Where("behavior = ?", behavior)
	}
	if strings.EqualFold(strings.TrimSpace(c.Query("behavior_group")), "operation") {
		query = query.Where("behavior IN ?", operationAccessBehaviors())
		query = query.Where("path NOT LIKE ?", "/api/admin/%")
	}
	query = applyVisitorFilter(c, query, "principal_type")
	rows := []userAgentRankingRow{}
	sampleLimit := max(limit*100, 500)
	err := query.Group("user_agent").Order("count DESC").Limit(sampleLimit).Scan(&rows).Error
	if err != nil {
		return []gin.H{}
	}
	counts := map[string]int64{}
	for _, row := range rows {
		label := accessUserAgentLabel(row.UserAgent, dimension)
		if strings.TrimSpace(label) == "" {
			label = "Other"
		}
		counts[label] += row.Count
	}
	items := make([]rankingRow, 0, len(counts))
	for label, count := range counts {
		items = append(items, rankingRow{Label: label, Count: count})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Label < items[j].Label
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		out = append(out, gin.H{"label": item.Label, "key": item.Label, "count": item.Count})
	}
	return out
}

func (h NativeHandlers) accessOperatingSystemRanking(c *gin.Context, limit int) []gin.H {
	window := parseAdminLogRange(c)
	query := h.DB.WithContext(c.Request.Context()).
		Table("access_logs").
		Select(userAgentOSCaseExpr("user_agent")+" AS label, COUNT(*) AS count").
		Where("created_at >= ? AND created_at < ?", window.Start, window.End).
		Where("user_agent IS NOT NULL AND user_agent <> ''")
	if behavior := strings.TrimSpace(c.Query("behavior")); behavior != "" {
		query = query.Where("behavior = ?", behavior)
	}
	if strings.EqualFold(strings.TrimSpace(c.Query("behavior_group")), "operation") {
		query = query.Where("behavior IN ?", operationAccessBehaviors())
		query = query.Where("path NOT LIKE ?", "/api/admin/%")
	}
	query = applyVisitorFilter(c, query, "principal_type")
	rows := []rankingRow{}
	if err := query.Group("label").Order("count DESC, label ASC").Limit(limit).Scan(&rows).Error; err != nil {
		return []gin.H{}
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		out = append(out, gin.H{"label": row.Label, "key": row.Label, "count": row.Count})
	}
	return out
}

func userAgentOSCaseExpr(column string) string {
	ua := "LOWER(COALESCE(" + column + ", ''))"
	return "CASE" +
		" WHEN " + ua + " LIKE '%harmonyos%' THEN 'HarmonyOS'" +
		" WHEN " + ua + " LIKE '%iphone%' OR " + ua + " LIKE '%ipad%' OR " + ua + " LIKE '%ipod%' OR " + ua + " LIKE '%ios%' OR (" + ua + " LIKE '%macintosh%' AND " + ua + " LIKE '%mobile%') THEN 'iOS'" +
		" WHEN " + ua + " LIKE '%android%' THEN 'Android'" +
		" WHEN " + ua + " LIKE '%windows%' THEN 'Windows'" +
		" WHEN " + ua + " LIKE '%mac os x%' OR " + ua + " LIKE '%macintosh%' THEN 'macOS'" +
		" WHEN " + ua + " LIKE '%linux%' OR " + ua + " LIKE '%x11%' THEN 'Linux'" +
		" ELSE 'Other' END"
}

func (h NativeHandlers) accessCountryRanking(c *gin.Context, limit int) []gin.H {
	window := parseAdminLogRange(c)
	query := h.DB.WithContext(c.Request.Context()).
		Table("access_logs").
		Select("ip AS label, COUNT(*) AS count").
		Where("created_at >= ? AND created_at < ?", window.Start, window.End).
		Where("ip IS NOT NULL AND ip <> ''")
	if behavior := strings.TrimSpace(c.Query("behavior")); behavior != "" {
		query = query.Where("behavior = ?", behavior)
	}
	if strings.EqualFold(strings.TrimSpace(c.Query("behavior_group")), "operation") {
		query = query.Where("behavior IN ?", operationAccessBehaviors())
		query = query.Where("path NOT LIKE ?", "/api/admin/%")
	}
	query = applyVisitorFilter(c, query, "principal_type")
	rows := []rankingRow{}
	sampleLimit := max(limit*100, 500)
	if err := query.Group("ip").Order("count DESC").Limit(sampleLimit).Scan(&rows).Error; err != nil {
		return []gin.H{}
	}
	type countryCount struct {
		Country services.GeoIPCountry
		Count   int64
	}
	counts := map[string]countryCount{}
	for _, row := range rows {
		country := h.countryForIP(row.Label)
		key := country.Code
		if key == "" {
			key = "Unknown"
			country = services.GeoIPCountry{Code: "Unknown", Name: "Unknown"}
		}
		item := counts[key]
		item.Country = country
		item.Count += row.Count
		counts[key] = item
	}
	items := make([]countryCount, 0, len(counts))
	for _, item := range counts {
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Country.Name < items[j].Country.Name
		}
		return items[i].Count > items[j].Count
	})
	if len(items) > limit {
		items = items[:limit]
	}
	out := make([]gin.H, 0, len(items))
	for _, item := range items {
		label := item.Country.Name
		if item.Country.Code != "" && item.Country.Code != "Unknown" {
			label = strings.TrimSpace(item.Country.Flag + " [" + item.Country.Code + "] " + item.Country.Name)
		}
		out = append(out, gin.H{"label": label, "key": item.Country.Code, "count": item.Count, "country_code": item.Country.Code, "country_name": item.Country.Name, "country_flag": item.Country.Flag})
	}
	return out
}

func accessUserAgentLabel(userAgent string, dimension string) string {
	device := userAgentDeviceLabel(userAgent)
	browser := userAgentBrowserLabel(userAgent)
	os := userAgentOSLabel(userAgent)
	switch dimension {
	case "device":
		return device
	case "browser":
		return browser
	case "os":
		return os
	default:
		return device + " / " + os + " / " + browser
	}
}

func userAgentDeviceLabel(userAgent string) string {
	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "bot") || strings.Contains(ua, "crawler") || strings.Contains(ua, "spider") || strings.Contains(ua, "slurp"):
		return "Bot"
	case strings.Contains(ua, "ipad") || strings.Contains(ua, "tablet"):
		return "Tablet"
	case strings.Contains(ua, "mobile") || strings.Contains(ua, "iphone") || strings.Contains(ua, "android"):
		return "Mobile"
	case strings.Contains(ua, "windows") || strings.Contains(ua, "macintosh") || strings.Contains(ua, "x11") || strings.Contains(ua, "linux"):
		return "Desktop"
	default:
		return "Other"
	}
}

func userAgentBrowserLabel(userAgent string) string {
	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "micromessenger"):
		return "WeChat"
	case strings.Contains(ua, "edg/") || strings.Contains(ua, "edge/"):
		return "Edge"
	case strings.Contains(ua, "opr/") || strings.Contains(ua, "opera"):
		return "Opera"
	case strings.Contains(ua, "firefox/") || strings.Contains(ua, "fxios/"):
		return "Firefox"
	case strings.Contains(ua, "crios/") || strings.Contains(ua, "chrome/") || strings.Contains(ua, "chromium/"):
		return "Chrome"
	case strings.Contains(ua, "safari/"):
		return "Safari"
	default:
		return "Other"
	}
}

func userAgentOSLabel(userAgent string) string {
	ua := strings.ToLower(userAgent)
	switch {
	case strings.Contains(ua, "harmonyos"):
		return "HarmonyOS"
	case strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod") || strings.Contains(ua, "ios") ||
		(strings.Contains(ua, "macintosh") && strings.Contains(ua, "mobile")):
		return "iOS"
	case strings.Contains(ua, "android"):
		return "Android"
	case strings.Contains(ua, "windows"):
		return "Windows"
	case strings.Contains(ua, "mac os x") || strings.Contains(ua, "macintosh"):
		return "macOS"
	case strings.Contains(ua, "linux") || strings.Contains(ua, "x11"):
		return "Linux"
	default:
		return "Other"
	}
}
