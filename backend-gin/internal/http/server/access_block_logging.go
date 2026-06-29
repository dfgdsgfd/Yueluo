package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	appmiddleware "yuem-go/backend-gin/internal/http/middleware"
	"yuem-go/backend-gin/internal/services"
)

type accessBlockLogger struct {
	audit           *services.AuditLogService
	observe         *services.ObservabilityService
	clientIPHeaders []string
}

func (l accessBlockLogger) RecordAccessBlock(c *gin.Context, match services.AccessBlockMatch, status int) {
	if c == nil || c.Request == nil {
		return
	}
	if status <= 0 {
		status = http.StatusForbidden
	}
	ip := requestClientIP(c, l.clientIPHeaders)
	requestID := c.Writer.Header().Get(appmiddleware.RequestIDHeader)
	metadata := map[string]any{
		"rule_id":       match.Rule.ID,
		"kind":          match.Rule.Kind,
		"match_type":    match.Rule.MatchType,
		"pattern":       match.Rule.Pattern,
		"action":        match.Rule.Action,
		"status_code":   match.Rule.StatusCode,
		"redirect_url":  match.Rule.RedirectURL,
		"client_ip":     ip,
		"matched_value": match.MatchedValue,
	}
	if l.audit != nil {
		l.audit.RecordSecurity(services.SecurityAuditLogEvent{
			Category:        "access_block",
			Action:          match.Rule.Kind,
			Outcome:         "blocked",
			ActorType:       "guest",
			IP:              ip,
			UserAgent:       c.Request.UserAgent(),
			BrowserLanguage: firstAcceptLanguage(c.GetHeader("Accept-Language")),
			Method:          c.Request.Method,
			Path:            c.Request.URL.Path,
			Status:          status,
			ReasonCode:      "access_block_rule",
			RequestID:       requestID,
			Metadata:        metadata,
			CreatedAt:       time.Now(),
		})
	}
	if l.observe != nil {
		l.observe.RecordAccess(c.Request.Context(), services.RecentAccessLogEvent{
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Status:    status,
			LatencyMS: 0,
			IP:        ip,
			UserAgent: c.Request.UserAgent(),
			RequestID: requestID,
			CreatedAt: time.Now(),
		})
		l.observe.Log(services.SystemLogEvent{
			Type:      "access_block",
			Level:     "warn",
			Message:   c.Request.Method + " " + c.Request.URL.Path,
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Status:    status,
			IP:        ip,
			UserAgent: c.Request.UserAgent(),
			RequestID: requestID,
			Detail:    metadata,
			CreatedAt: time.Now(),
		})
	}
}
