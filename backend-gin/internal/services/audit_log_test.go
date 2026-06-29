package services

import (
	"testing"
	"time"

	"yuem-go/backend-gin/internal/config"
)

func TestAuditLogRecordAccessDropsWhenBufferFull(t *testing.T) {
	service := &AuditLogService{
		cfg:         config.Config{AccessLog: config.AccessLogConfig{Enabled: true, Scope: "all"}},
		accessCh:    make(chan AccessLogEvent, 1),
		behaviorSet: map[string]struct{}{},
	}
	service.RecordAccess(AccessLogEvent{Behavior: "post_view", Path: "/api/posts/1"})
	service.RecordAccess(AccessLogEvent{Behavior: "post_view", Path: "/api/posts/2"})
	if got := len(service.accessCh); got != 1 {
		t.Fatalf("buffer len = %d, want 1", got)
	}
	if got := service.accessDropped.Load(); got != 1 {
		t.Fatalf("dropped = %d, want 1", got)
	}
}

func TestAuditLogRecordSecurityUsesQueueBuffer(t *testing.T) {
	service := &AuditLogService{
		cfg:        config.Config{SecurityAuditLog: config.SecurityAuditLogConfig{Enabled: true}},
		securityCh: make(chan SecurityAuditLogEvent, 1),
	}
	service.RecordSecurity(SecurityAuditLogEvent{Category: "system", Action: "exception", Outcome: "failure"})
	if got := len(service.securityCh); got != 1 {
		t.Fatalf("security buffer len = %d, want 1", got)
	}
	service.RecordSecurity(SecurityAuditLogEvent{Category: "system", Action: "exception", Outcome: "failure"})
	if got := service.securityDropped.Load(); got != 1 {
		t.Fatalf("security dropped = %d, want 1", got)
	}
}

func TestAccessLogRowsNormalizePayload(t *testing.T) {
	userID := int64(42)
	targetID := int64(99)
	rows := accessLogRows([]AccessLogEvent{{
		UserID:          &userID,
		UserDisplayID:   "u42",
		PrincipalType:   "USER",
		IP:              "127.0.0.1",
		UserAgent:       "agent",
		BrowserLanguage: "zh-CN",
		Method:          "get",
		Path:            "/api/posts/99",
		Behavior:        "POST_VIEW",
		TargetType:      "POST",
		TargetID:        &targetID,
		Metadata:        map[string]any{"source": "test"},
	}})
	if len(rows) != 1 {
		t.Fatalf("rows len = %d, want 1", len(rows))
	}
	row := rows[0]
	if row.UserID == nil || *row.UserID != userID || row.Method != "GET" || row.Behavior != "post_view" || row.TargetType == nil || *row.TargetType != "post" || len(row.Metadata) == 0 {
		t.Fatalf("unexpected row: %+v", row)
	}
}

func TestQueueServicePruneThrottle(t *testing.T) {
	service := &QueueService{}
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	if !service.shouldPruneAccessLogs(now) {
		t.Fatalf("first prune should be allowed")
	}
	if service.shouldPruneAccessLogs(now.Add(10 * time.Minute)) {
		t.Fatalf("second prune inside interval should be throttled")
	}
	if !service.shouldPruneAccessLogs(now.Add(2 * time.Hour)) {
		t.Fatalf("prune after interval should be allowed")
	}
}
