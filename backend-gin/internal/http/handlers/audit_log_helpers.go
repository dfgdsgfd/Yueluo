package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	appmiddleware "yuem-go/backend-gin/internal/http/middleware"
	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) recordSecurityAudit(c *gin.Context, category string, action string, outcome string, reasonCode string, status int, actorID *int64, actorType string, actorDisplayID string, metadata map[string]any) {
	if h.AuditLog == nil || c == nil || c.Request == nil {
		return
	}
	if status == 0 {
		status = http.StatusOK
	}
	h.AuditLog.RecordSecurity(services.SecurityAuditLogEvent{
		Category:        category,
		Action:          action,
		Outcome:         outcome,
		ActorID:         actorID,
		ActorType:       firstNonEmptyHandler(actorType, "unknown"),
		ActorDisplayID:  actorDisplayID,
		IP:              h.clientIP(c),
		UserAgent:       c.Request.UserAgent(),
		BrowserLanguage: firstAcceptLanguage(c.GetHeader("Accept-Language")),
		Method:          c.Request.Method,
		Path:            c.Request.URL.Path,
		Status:          status,
		ReasonCode:      reasonCode,
		RequestID:       c.Writer.Header().Get(appmiddleware.RequestIDHeader),
		Metadata:        metadata,
		CreatedAt:       time.Now(),
	})
}

func (h NativeHandlers) recordSystemFileDeletionAudit(c *gin.Context, reasonCode string, postIDs []int64, summary postFileDeletionSummary, metadata map[string]any) {
	if summary.attempted() == 0 {
		return
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["resource_type"] = "post"
	metadata["post_ids"] = postIDs
	metadata["file_delete"] = summary.auditMetadata()
	if actorID, actorDisplayID, actorType := requestUserActor(c); actorID != nil || actorDisplayID != "" || actorType != "" {
		triggeredBy := map[string]any{"actor_type": actorType, "actor_display_id": actorDisplayID}
		if actorID != nil {
			triggeredBy["actor_id"] = *actorID
		}
		metadata["triggered_by"] = triggeredBy
	}
	h.recordSecurityAudit(c, "system", "file_delete", summary.outcome(), reasonCode, http.StatusOK, nil, "system", "system", metadata)
}

func requestUserActor(c *gin.Context) (*int64, string, string) {
	raw, ok := c.Get("user")
	if !ok {
		return nil, "", "unknown"
	}
	user, ok := raw.(*services.RequestUser)
	if !ok || user == nil {
		return nil, "", "unknown"
	}
	id := user.ID
	display := firstNonEmptyHandler(user.UserID, user.Username, strconv.FormatInt(user.ID, 10))
	return &id, display, firstNonEmptyHandler(user.Type, "user")
}

func firstNonEmptyHandler(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func firstAcceptLanguage(v string) string {
	if v == "" {
		return ""
	}
	if idx := strings.IndexByte(v, ','); idx > 0 {
		return strings.TrimSpace(v[:idx])
	}
	return strings.TrimSpace(v)
}
