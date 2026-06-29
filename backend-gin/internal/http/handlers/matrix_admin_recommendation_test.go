package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestAdminRecommendationPostConfigCreateSuppliesRequiredTimestamps(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	statements := []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY, user_id TEXT, nickname TEXT, avatar TEXT)`,
		`CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER NOT NULL, title TEXT, content TEXT, type INTEGER NOT NULL DEFAULT 1, like_count INTEGER NOT NULL DEFAULT 0, collect_count INTEGER NOT NULL DEFAULT 0, view_count INTEGER NOT NULL DEFAULT 0)`,
		`CREATE TABLE post_images (id INTEGER PRIMARY KEY, post_id INTEGER NOT NULL, image_url TEXT)`,
		`CREATE TABLE post_videos (id INTEGER PRIMARY KEY, post_id INTEGER NOT NULL, cover_url TEXT)`,
		`CREATE TABLE post_recommend_configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			post_id INTEGER NOT NULL UNIQUE,
			boost_score REAL NOT NULL DEFAULT 0,
			is_pinned BOOLEAN NOT NULL DEFAULT FALSE,
			is_suppressed BOOLEAN NOT NULL DEFAULT FALSE,
			target_user_id INTEGER,
			reason TEXT,
			is_active BOOLEAN NOT NULL DEFAULT TRUE,
			start_time DATETIME,
			end_time DATETIME,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`INSERT INTO users (id, user_id, nickname) VALUES (1, 'user-1', 'User One')`,
		`INSERT INTO posts (id, user_id, title, content) VALUES (11, 1, 'Post', 'Content')`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("execute schema statement: %v", err)
		}
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/admin/recommendation/post-configs", strings.NewReader(`{"post_id":11,"boost_score":10,"is_pinned":true}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	NativeHandlers{DB: db}.adminRecommendationPostConfigCreate(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s, want 200", recorder.Code, recorder.Body.String())
	}
	var timestamps struct {
		CreatedAt time.Time
		UpdatedAt time.Time
	}
	if err := db.Table("post_recommend_configs").Select("created_at, updated_at").Where("post_id = ?", 11).Take(&timestamps).Error; err != nil {
		t.Fatalf("load timestamps: %v", err)
	}
	if timestamps.CreatedAt.IsZero() || timestamps.UpdatedAt.IsZero() {
		t.Fatalf("timestamps were not populated: %#v", timestamps)
	}
}
