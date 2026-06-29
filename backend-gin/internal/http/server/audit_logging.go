package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	appmiddleware "yuem-go/backend-gin/internal/http/middleware"
	"yuem-go/backend-gin/internal/services"
)

type accessBehaviorMatch struct {
	Behavior   string
	TargetType string
	TargetID   *int64
}

func auditLogMiddleware(audit *services.AuditLogService, clientIPHeaders []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		if audit == nil || !serverAPIPath(c.Request.URL.Path) {
			return
		}
		status := c.Writer.Status()
		requestID := c.Writer.Header().Get(appmiddleware.RequestIDHeader)
		actorID, actorDisplayID, actorType := requestActor(c)
		ip := requestClientIP(c, clientIPHeaders)
		latencyMS := time.Since(start).Milliseconds()
		match := classifyAccessBehavior(c.Request.Method, c.Request.URL.Path)
		applyAccessBehaviorOverride(c, &match)
		if match.Behavior != "internal" {
			audit.RecordAccess(services.AccessLogEvent{
				UserID:          actorID,
				UserDisplayID:   actorDisplayID,
				PrincipalType:   firstNonEmpty(actorType, "guest"),
				IP:              ip,
				UserAgent:       c.Request.UserAgent(),
				BrowserLanguage: firstAcceptLanguage(c.GetHeader("Accept-Language")),
				Method:          c.Request.Method,
				Path:            c.Request.URL.Path,
				Status:          status,
				LatencyMS:       latencyMS,
				Behavior:        match.Behavior,
				TargetType:      match.TargetType,
				TargetID:        match.TargetID,
				RequestID:       requestID,
				CreatedAt:       time.Now(),
			})
		}
		if shouldRecordSystemSecurityEvent(c.Request.URL.Path, status) {
			audit.RecordSecurity(services.SecurityAuditLogEvent{
				Category:        "system",
				Action:          "exception",
				Outcome:         "failure",
				ActorType:       "system",
				ActorDisplayID:  "system",
				IP:              ip,
				UserAgent:       c.Request.UserAgent(),
				BrowserLanguage: firstAcceptLanguage(c.GetHeader("Accept-Language")),
				Method:          c.Request.Method,
				Path:            c.Request.URL.Path,
				Status:          status,
				ReasonCode:      statusReasonCode(status),
				RequestID:       requestID,
				Metadata:        systemExceptionAuditMetadata(actorID, actorDisplayID, actorType, match, latencyMS),
				CreatedAt:       time.Now(),
			})
		}
		if shouldRecordAdminSecurityEvent(c.Request.Method, c.Request.URL.Path, status) {
			audit.RecordSecurity(services.SecurityAuditLogEvent{
				Category:        "admin_api",
				Action:          ternaryString(c.Request.Method == http.MethodGet, "access", "write"),
				Outcome:         ternaryString(status >= http.StatusBadRequest, "failure", "success"),
				ActorID:         actorID,
				ActorType:       firstNonEmpty(actorType, "unknown"),
				ActorDisplayID:  actorDisplayID,
				IP:              ip,
				UserAgent:       c.Request.UserAgent(),
				BrowserLanguage: firstAcceptLanguage(c.GetHeader("Accept-Language")),
				Method:          c.Request.Method,
				Path:            c.Request.URL.Path,
				Status:          status,
				ReasonCode:      statusReasonCode(status),
				RequestID:       requestID,
				CreatedAt:       time.Now(),
			})
		}
	}
}

func classifyAccessBehavior(method string, path string) accessBehaviorMatch {
	method = strings.ToUpper(strings.TrimSpace(method))
	path = strings.TrimSpace(path)
	if shouldSkipPersistentAccessLog(path) {
		return accessBehaviorMatch{Behavior: "internal"}
	}
	if strings.HasPrefix(path, "/api/admin/") {
		return accessBehaviorMatch{Behavior: "admin_access"}
	}
	if method == http.MethodGet && path == "/api/search" {
		return accessBehaviorMatch{Behavior: "search"}
	}
	if method == http.MethodPost && path == "/api/posts" {
		return accessBehaviorMatch{Behavior: "post_create", TargetType: "post"}
	}
	if method == http.MethodPut {
		if id, ok := numericPathID(path, "/api/posts/"); ok {
			return accessBehaviorMatch{Behavior: "post_update", TargetType: "post", TargetID: &id}
		}
	}
	if method == http.MethodDelete {
		if id, ok := numericPathID(path, "/api/posts/"); ok {
			return accessBehaviorMatch{Behavior: "post_delete", TargetType: "post", TargetID: &id}
		}
	}
	if method == http.MethodPost && path == "/api/comments" {
		return accessBehaviorMatch{Behavior: "comment_create", TargetType: "comment"}
	}
	if (method == http.MethodPost || method == http.MethodDelete) && path == "/api/likes" {
		return accessBehaviorMatch{Behavior: "like"}
	}
	if method == http.MethodPost && strings.HasPrefix(path, "/api/upload/") {
		return accessBehaviorMatch{Behavior: uploadAccessBehavior(path)}
	}
	if method == http.MethodGet && isFeedPath(path) {
		return accessBehaviorMatch{Behavior: "feed_view"}
	}
	if method == http.MethodGet {
		if id, ok := numericPathID(path, "/api/posts/"); ok {
			return accessBehaviorMatch{Behavior: "post_view", TargetType: "post", TargetID: &id}
		}
	}
	if method == http.MethodPost && strings.HasSuffix(path, "/collect") {
		if id, ok := numericPathID(strings.TrimSuffix(path, "/collect"), "/api/posts/"); ok {
			return accessBehaviorMatch{Behavior: "collect", TargetType: "post", TargetID: &id}
		}
	}
	if (method == http.MethodPost || method == http.MethodDelete) && strings.HasSuffix(path, "/follow") {
		if id, ok := numericPathID(strings.TrimSuffix(path, "/follow"), "/api/users/"); ok {
			return accessBehaviorMatch{Behavior: "follow", TargetType: "user", TargetID: &id}
		}
	}
	return accessBehaviorMatch{Behavior: "api_access"}
}

func applyAccessBehaviorOverride(c *gin.Context, match *accessBehaviorMatch) {
	if c == nil || match == nil {
		return
	}
	if value, ok := c.Get("access_behavior"); ok {
		if behavior := strings.TrimSpace(fmt.Sprint(value)); behavior != "" {
			match.Behavior = behavior
		}
	}
	if value, ok := c.Get("access_target_type"); ok {
		match.TargetType = strings.TrimSpace(fmt.Sprint(value))
	}
	if value, ok := c.Get("access_target_id"); ok {
		switch typed := value.(type) {
		case int64:
			id := typed
			match.TargetID = &id
		case int:
			id := int64(typed)
			match.TargetID = &id
		case string:
			if id, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64); err == nil && id > 0 {
				match.TargetID = &id
			}
		}
	}
}

func uploadAccessBehavior(path string) string {
	switch path {
	case "/api/upload/single", "/api/upload/multiple", "/api/upload/chunk/merge/image":
		return "image_upload"
	case "/api/upload/video", "/api/upload/chunk/merge":
		return "video_upload"
	case "/api/upload/attachment":
		return "attachment_upload"
	case "/api/upload/apk", "/api/upload/chunk/merge/apk":
		return "apk_upload"
	default:
		return "upload"
	}
}

func shouldSkipPersistentAccessLog(path string) bool {
	if path == "/api/health" {
		return true
	}
	return strings.HasPrefix(path, "/api/admin/logs") ||
		strings.HasPrefix(path, "/api/admin/observability") ||
		strings.HasPrefix(path, "/api/admin/system-logs")
}

func isFeedPath(path string) bool {
	switch path {
	case "/api/posts", "/api/posts/recommended", "/api/posts/hot", "/api/posts/friends", "/api/posts/following", "/api/posts/video-center":
		return true
	default:
		return false
	}
}

func numericPathID(path string, prefix string) (int64, bool) {
	raw := strings.TrimPrefix(path, prefix)
	if raw == path || raw == "" || strings.Contains(raw, "/") {
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	return id, err == nil && id > 0
}

func shouldRecordAdminSecurityEvent(method string, path string, status int) bool {
	if !strings.HasPrefix(path, "/api/admin/") || shouldSkipPersistentAccessLog(path) {
		return false
	}
	return method != http.MethodGet || status >= http.StatusBadRequest
}

func shouldRecordSystemSecurityEvent(path string, status int) bool {
	return status >= http.StatusInternalServerError && !shouldSkipPersistentAccessLog(path)
}

func systemExceptionAuditMetadata(actorID *int64, actorDisplayID string, actorType string, match accessBehaviorMatch, latencyMS int64) map[string]any {
	metadata := map[string]any{
		"event_type": "http_exception",
		"latency_ms": latencyMS,
	}
	if actorID != nil || strings.TrimSpace(actorDisplayID) != "" || strings.TrimSpace(actorType) != "" {
		triggeredBy := map[string]any{
			"actor_type":       firstNonEmpty(actorType, "unknown"),
			"actor_display_id": actorDisplayID,
		}
		if actorID != nil {
			triggeredBy["actor_id"] = *actorID
		}
		metadata["triggered_by"] = triggeredBy
	}
	if strings.TrimSpace(match.Behavior) != "" {
		metadata["behavior"] = match.Behavior
	}
	if strings.TrimSpace(match.TargetType) != "" {
		metadata["target_type"] = match.TargetType
	}
	if match.TargetID != nil {
		metadata["target_id"] = *match.TargetID
	}
	return metadata
}

func requestActor(c *gin.Context) (*int64, string, string) {
	raw, ok := c.Get("user")
	if !ok {
		return nil, "", "guest"
	}
	user, ok := raw.(*services.RequestUser)
	if !ok || user == nil {
		return nil, "", "guest"
	}
	id := user.ID
	display := firstNonEmpty(user.UserID, user.Username, strconv.FormatInt(user.ID, 10))
	return &id, display, firstNonEmpty(user.Type, "user")
}

func statusReasonCode(status int) string {
	switch {
	case status >= http.StatusInternalServerError:
		return "server_error"
	case status == http.StatusUnauthorized:
		return "unauthorized"
	case status == http.StatusForbidden:
		return "forbidden"
	case status >= http.StatusBadRequest:
		return "bad_request"
	default:
		return ""
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func ternaryString(cond bool, yes string, no string) string {
	if cond {
		return yes
	}
	return no
}

// firstAcceptLanguage extracts only the first language tag from an
// Accept-Language header value (e.g. "zh-CN,zh;q=0.9" → "zh-CN").
func firstAcceptLanguage(v string) string {
	if v == "" {
		return ""
	}
	if idx := strings.IndexByte(v, ','); idx > 0 {
		return strings.TrimSpace(v[:idx])
	}
	return strings.TrimSpace(v)
}
