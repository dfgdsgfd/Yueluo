package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/services"
)

func parseAdminLogRange(c *gin.Context) adminLogRange {
	now := time.Now()
	value := strings.ToLower(strings.TrimSpace(c.DefaultQuery("range", "7d")))
	var start time.Time
	end := now
	switch value {
	case "today", "day":
		start = startOfDay(now)
	case "1h":
		start = now.Add(-1 * time.Hour)
	case "3h":
		start = now.Add(-3 * time.Hour)
	case "6h":
		start = now.Add(-6 * time.Hour)
	case "12h":
		start = now.Add(-12 * time.Hour)
	case "3d":
		start = startOfDay(now).AddDate(0, 0, -2)
	case "7d", "week":
		start = startOfDay(now).AddDate(0, 0, -6)
	case "30d", "month":
		start = startOfDay(now).AddDate(0, 0, -29)
	case "365d", "year":
		start = startOfDay(now).AddDate(-1, 0, 1)
	case "all":
		start = time.Unix(0, 0)
	case "custom":
		start = parseLogTime(c.Query("start"), startOfDay(now).AddDate(0, 0, -6))
		end = parseLogTime(c.Query("end"), now)
	default:
		start = startOfDay(now).AddDate(0, 0, -6)
	}
	if !start.Before(end) {
		start = end.Add(-24 * time.Hour)
	}
	bucket := strings.ToLower(strings.TrimSpace(c.DefaultQuery("bucket", "auto")))
	if bucket == "auto" || bucket == "" {
		bucket = autoLogBucket(start, end)
	}
	if bucket != "hour" && bucket != "day" && bucket != "month" {
		bucket = autoLogBucket(start, end)
	}
	return adminLogRange{Start: start, End: end, Bucket: bucket}
}

func parseLogTime(raw string, fallback time.Time) time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed
	}
	if parsed, err := time.Parse("2006-01-02", raw); err == nil {
		return parsed
	}
	return fallback
}

func autoLogBucket(start time.Time, end time.Time) string {
	duration := end.Sub(start)
	switch {
	case duration <= 48*time.Hour:
		return "hour"
	case duration <= 92*24*time.Hour:
		return "day"
	default:
		return "month"
	}
}

func adminLogBucketExpr(db *gorm.DB, column string, bucket string) string {
	unit := "day"
	switch bucket {
	case "hour", "month":
		unit = bucket
	}
	if db != nil && db.Dialector != nil && db.Dialector.Name() == "mysql" {
		switch unit {
		case "hour":
			return "TIMESTAMP(DATE_FORMAT(" + column + ", '%Y-%m-%d %H:00:00'))"
		case "month":
			return "TIMESTAMP(DATE_FORMAT(" + column + ", '%Y-%m-01 00:00:00'))"
		default:
			return "DATE(" + column + ")"
		}
	}
	return "date_trunc('" + unit + "', " + column + ")"
}

func rankingExpr(db *gorm.DB, field string) string {
	castUser := "CAST(user_id AS TEXT)"
	if db != nil && db.Dialector != nil && db.Dialector.Name() == "mysql" {
		castUser = "CAST(user_id AS CHAR)"
	}
	switch field {
	case "path":
		return "path"
	case "target_id":
		return rankingCastExpr(db, "target_id")
	case "user_display_id":
		return "COALESCE(user_display_id, " + castUser + ")"
	case "ip":
		return "ip"
	case "browser_language":
		return "browser_language"
	case "behavior":
		return "behavior"
	default:
		return ""
	}
}

func rankingCastExpr(db *gorm.DB, column string) string {
	if db != nil && db.Dialector != nil && db.Dialector.Name() == "mysql" {
		return "CAST(" + column + " AS CHAR)"
	}
	return "CAST(" + column + " AS TEXT)"
}

func mergeLogSeries(window adminLogRange, accessRows []accessLogSeriesRow, securityRows []securityLogSeriesRow) []gin.H {
	type point struct {
		pv             int64
		activeUsers    int64
		uniqueIPs      int64
		postViews      int64
		securityEvents int64
	}
	points := map[string]point{}
	for _, row := range accessRows {
		key := bucketKey(row.Bucket, window.Bucket)
		value := points[key]
		value.pv = row.PV
		value.activeUsers = row.ActiveUsers
		value.uniqueIPs = row.UniqueIPs
		value.postViews = row.PostViews
		points[key] = value
	}
	for _, row := range securityRows {
		key := bucketKey(row.Bucket, window.Bucket)
		value := points[key]
		value.securityEvents = row.SecurityEvents
		points[key] = value
	}
	out := []gin.H{}
	for cursor := bucketStart(window.Start, window.Bucket); cursor.Before(window.End); cursor = nextBucket(cursor, window.Bucket) {
		key := bucketKey(cursor, window.Bucket)
		value := points[key]
		out = append(out, gin.H{
			"ts":              cursor.UTC().Format(time.RFC3339),
			"pv":              value.pv,
			"active_users":    value.activeUsers,
			"unique_ips":      value.uniqueIPs,
			"post_views":      value.postViews,
			"security_events": value.securityEvents,
		})
	}
	return out
}

func aggregateLogSeries(series []gin.H) gin.H {
	var pv, activeUsers, uniqueIPs, postViews, securityEvents int64
	for _, row := range series {
		pv += logInt64(row["pv"])
		activeUsers += logInt64(row["active_users"])
		uniqueIPs += logInt64(row["unique_ips"])
		postViews += logInt64(row["post_views"])
		securityEvents += logInt64(row["security_events"])
	}
	return gin.H{"pv": pv, "active_users": activeUsers, "unique_ips": uniqueIPs, "post_views": postViews, "security_events": securityEvents}
}

func bucketStart(value time.Time, bucket string) time.Time {
	switch bucket {
	case "hour":
		return time.Date(value.Year(), value.Month(), value.Day(), value.Hour(), 0, 0, 0, value.Location())
	case "month":
		return time.Date(value.Year(), value.Month(), 1, 0, 0, 0, 0, value.Location())
	default:
		return startOfDay(value)
	}
}

func nextBucket(value time.Time, bucket string) time.Time {
	switch bucket {
	case "hour":
		return value.Add(time.Hour)
	case "month":
		return value.AddDate(0, 1, 0)
	default:
		return value.AddDate(0, 0, 1)
	}
}

func bucketKey(value time.Time, bucket string) string {
	return bucketStart(value, bucket).UTC().Format(time.RFC3339)
}

func (h NativeHandlers) accessLogMap(row domain.AccessLog) gin.H {
	country := h.countryForIP(row.IP)
	return gin.H{
		"id":               row.ID,
		"user_id":          row.UserID,
		"user_display_id":  row.UserDisplayID,
		"principal_type":   row.PrincipalType,
		"ip":               row.IP,
		"country_code":     country.Code,
		"country_name":     country.Name,
		"country_flag":     country.Flag,
		"user_agent":       row.UserAgent,
		"browser_language": row.BrowserLanguage,
		"method":           row.Method,
		"path":             row.Path,
		"status":           row.Status,
		"latency_ms":       row.LatencyMS,
		"behavior":         row.Behavior,
		"target_type":      row.TargetType,
		"target_id":        row.TargetID,
		"request_id":       row.RequestID,
		"metadata":         jsonValue(row.Metadata),
		"created_at":       row.CreatedAt,
	}
}

func (h NativeHandlers) securityAuditLogMap(row domain.SecurityAuditLog) gin.H {
	country := h.countryForIP(row.IP)
	return gin.H{
		"id":               row.ID,
		"category":         row.Category,
		"action":           row.Action,
		"outcome":          row.Outcome,
		"actor_id":         row.ActorID,
		"actor_type":       row.ActorType,
		"actor_display_id": row.ActorDisplayID,
		"ip":               row.IP,
		"country_code":     country.Code,
		"country_name":     country.Name,
		"country_flag":     country.Flag,
		"user_agent":       row.UserAgent,
		"browser_language": row.BrowserLanguage,
		"method":           row.Method,
		"path":             row.Path,
		"status":           row.Status,
		"reason_code":      row.ReasonCode,
		"request_id":       row.RequestID,
		"metadata":         jsonValue(row.Metadata),
		"created_at":       row.CreatedAt,
	}
}

func (h NativeHandlers) countryForIP(ip string) services.GeoIPCountry {
	if h.GeoIP == nil {
		return services.GeoIPCountry{}
	}
	return h.GeoIP.CountryForIP(ip)
}

func operationAccessBehaviors() []string {
	return []string{
		"post_create",
		"draft_save",
		"post_update",
		"post_delete",
		"draft_delete",
		"comment_create",
		"like",
		"collect",
		"follow",
		"image_upload",
		"video_upload",
		"attachment_upload",
		"apk_upload",
		"upload",
	}
}

func (h NativeHandlers) auditLogRuntimeStatus() gin.H {
	if h.AuditLog == nil {
		return gin.H{"access_enabled": false, "security_enabled": false, "available": false}
	}
	return gin.H(h.AuditLog.RuntimeStatus())
}

func int64FromString(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func logInt64(value any) int64 {
	if parsed, ok := int64FromAny(value); ok {
		return parsed
	}
	return 0
}
