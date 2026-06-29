package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestClassifyAccessBehavior(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		behavior   string
		targetType string
		targetID   int64
	}{
		{name: "post detail", method: http.MethodGet, path: "/api/posts/42", behavior: "post_view", targetType: "post", targetID: 42},
		{name: "feed", method: http.MethodGet, path: "/api/posts/recommended", behavior: "feed_view"},
		{name: "collect", method: http.MethodPost, path: "/api/posts/42/collect", behavior: "collect", targetType: "post", targetID: 42},
		{name: "follow", method: http.MethodDelete, path: "/api/users/7/follow", behavior: "follow", targetType: "user", targetID: 7},
		{name: "admin", method: http.MethodPut, path: "/api/admin/settings", behavior: "admin_access"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classifyAccessBehavior(tt.method, tt.path)
			if got.Behavior != tt.behavior || got.TargetType != tt.targetType {
				t.Fatalf("classifyAccessBehavior() = %+v, want behavior=%q target=%q", got, tt.behavior, tt.targetType)
			}
			if tt.targetID > 0 {
				if got.TargetID == nil || *got.TargetID != tt.targetID {
					t.Fatalf("target id = %v, want %d", got.TargetID, tt.targetID)
				}
			}
		})
	}
}

func TestSystemSecurityEventClassification(t *testing.T) {
	if !shouldRecordSystemSecurityEvent("/api/posts/42", http.StatusInternalServerError) {
		t.Fatalf("500 API response should record a system security event")
	}
	if shouldRecordSystemSecurityEvent("/api/posts/42", http.StatusBadRequest) {
		t.Fatalf("4xx API response should not record a system security event")
	}
	if shouldRecordSystemSecurityEvent("/api/admin/logs/security", http.StatusInternalServerError) {
		t.Fatalf("log inspection routes should not recursively record system security events")
	}

	targetID := int64(42)
	actorID := int64(7)
	metadata := systemExceptionAuditMetadata(&actorID, "u7", "user", accessBehaviorMatch{Behavior: "post_view", TargetType: "post", TargetID: &targetID}, 123)
	if metadata["event_type"] != "http_exception" || metadata["latency_ms"] != int64(123) || metadata["target_type"] != "post" || metadata["target_id"] != targetID {
		t.Fatalf("metadata missing expected system exception fields: %#v", metadata)
	}
	triggeredBy, ok := metadata["triggered_by"].(map[string]any)
	if !ok || triggeredBy["actor_id"] != actorID || triggeredBy["actor_type"] != "user" {
		t.Fatalf("metadata missing triggered_by actor: %#v", metadata)
	}
}

func TestRequestClientIPUsesForwardedHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	c.Request.RemoteAddr = "172.18.0.13:43120"
	c.Request.Header.Set("X-Forwarded-For", "203.0.113.7, 172.18.0.13")

	if got := requestClientIP(c, nil); got != "203.0.113.7" {
		t.Fatalf("requestClientIP() = %q, want forwarded client", got)
	}
}

func TestRequestClientIPUsesConfiguredHeaderOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	c.Request.RemoteAddr = "172.18.0.13:43120"
	c.Request.Header.Set("X-Forwarded-For", "172.18.0.13")
	c.Request.Header.Set("X-Real-IP", "198.51.100.9")

	if got := requestClientIP(c, []string{"X-Real-IP", "X-Forwarded-For"}); got != "198.51.100.9" {
		t.Fatalf("requestClientIP() = %q, want configured real IP", got)
	}
}
