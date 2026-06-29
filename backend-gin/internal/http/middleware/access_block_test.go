package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/services"
)

type accessBlockLogEvent struct {
	match  services.AccessBlockMatch
	status int
}

type recordingAccessBlockLogger struct {
	events []accessBlockLogEvent
}

func (l *recordingAccessBlockLogger) RecordAccessBlock(_ *gin.Context, match services.AccessBlockMatch, status int) {
	l.events = append(l.events, accessBlockLogEvent{match: match, status: status})
}

func TestAccessBlockMiddlewareMarksStatusBlocks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := newAccessBlockMiddlewareService(t, domain.AccessBlockRule{
		ID:         17,
		Kind:       services.AccessBlockKindIP,
		MatchType:  services.AccessBlockMatchCIDR,
		Pattern:    "203.0.113.0/24",
		Enabled:    true,
		Priority:   100,
		Action:     services.AccessBlockActionStatus,
		StatusCode: 444,
	})
	logger := &recordingAccessBlockLogger{}
	router := gin.New()
	router.Use(AccessBlock(service, func(*gin.Context) string { return "203.0.113.9" }, logger))
	router.GET("/api/posts/1", func(c *gin.Context) {
		c.String(http.StatusOK, "visible")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/posts/1", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != 444 {
		t.Fatalf("status = %d, want 444", rec.Code)
	}
	if got := rec.Header().Get(AccessBlockHeader); got != "1" {
		t.Fatalf("%s = %q, want 1", AccessBlockHeader, got)
	}
	if got := rec.Header().Get(AccessBlockRuleIDHeader); got != "17" {
		t.Fatalf("%s = %q, want 17", AccessBlockRuleIDHeader, got)
	}
	if got := rec.Header().Get(AccessBlockActionHeader); got != services.AccessBlockActionStatus {
		t.Fatalf("%s = %q, want status", AccessBlockActionHeader, got)
	}
	if got := rec.Header().Get(AccessBlockStatusCodeHeader); got != "444" {
		t.Fatalf("%s = %q, want 444", AccessBlockStatusCodeHeader, got)
	}
	if rec.Body.String() != "" {
		t.Fatalf("blocked response should not run the route handler, body=%q", rec.Body.String())
	}
	if len(logger.events) != 1 || logger.events[0].status != 444 || logger.events[0].match.Rule.ID != 17 {
		t.Fatalf("logger events = %#v, want one status block", logger.events)
	}
}

func TestAccessBlockMiddlewareMarksRedirectBlocks(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := newAccessBlockMiddlewareService(t, domain.AccessBlockRule{
		ID:          23,
		Kind:        services.AccessBlockKindUA,
		MatchType:   services.AccessBlockMatchUAContains,
		Pattern:     "BadBot",
		Enabled:     true,
		Priority:    50,
		Action:      services.AccessBlockActionRedirect,
		RedirectURL: "https://example.test/blocked",
	})
	logger := &recordingAccessBlockLogger{}
	router := gin.New()
	router.Use(AccessBlock(service, nil, logger))
	router.GET("/api/posts/1", func(c *gin.Context) {
		c.String(http.StatusOK, "visible")
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/posts/1", nil)
	req.Header.Set("User-Agent", "Friendly BadBot")
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want 302", rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "https://example.test/blocked" {
		t.Fatalf("Location = %q, want redirect URL", got)
	}
	if got := rec.Header().Get(AccessBlockHeader); got != "1" {
		t.Fatalf("%s = %q, want 1", AccessBlockHeader, got)
	}
	if got := rec.Header().Get(AccessBlockRuleIDHeader); got != "23" {
		t.Fatalf("%s = %q, want 23", AccessBlockRuleIDHeader, got)
	}
	if got := rec.Header().Get(AccessBlockActionHeader); got != services.AccessBlockActionRedirect {
		t.Fatalf("%s = %q, want redirect", AccessBlockActionHeader, got)
	}
	if got := rec.Header().Get(AccessBlockStatusCodeHeader); got != "302" {
		t.Fatalf("%s = %q, want 302", AccessBlockStatusCodeHeader, got)
	}
	if len(logger.events) != 1 || logger.events[0].status != http.StatusFound || logger.events[0].match.Rule.ID != 23 {
		t.Fatalf("logger events = %#v, want one redirect block", logger.events)
	}
}

func newAccessBlockMiddlewareService(t *testing.T, rules ...domain.AccessBlockRule) *services.AccessBlockService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.AccessBlockRule{}); err != nil {
		t.Fatal(err)
	}
	for _, rule := range rules {
		if err := db.Create(&rule).Error; err != nil {
			t.Fatal(err)
		}
	}
	service := services.NewAccessBlockService(db, nil, zap.NewNop(), false)
	if err := service.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	return service
}
