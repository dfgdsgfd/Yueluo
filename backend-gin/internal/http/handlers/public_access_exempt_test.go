package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/services"
)

func TestNoteGuestRestrictionAllowsPublicAccessExemptPost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openPublicAccessExemptTestDB(t, &domain.Post{})
	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), "guest_access_note_restricted", true) {
		t.Fatalf("failed to restrict note guest access")
	}
	post := domain.Post{
		UserID:             1,
		Title:              "Public exempt",
		Type:               1,
		Visibility:         "public",
		PublicAccessExempt: true,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	handler := NativeHandlers{DB: db, Settings: settings}
	router := gin.New()
	router.GET("/api/posts/:id", handler.OptionalAuthWithNoteGuestRestriction(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/posts/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", rec.Code, rec.Body.String())
	}
}

func TestNoteGuestRestrictionRejectsNonExemptPost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openPublicAccessExemptTestDB(t, &domain.Post{})
	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), "guest_access_note_restricted", true) {
		t.Fatalf("failed to restrict note guest access")
	}
	post := domain.Post{UserID: 1, Title: "Restricted", Type: 1, Visibility: "public"}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}

	handler := NativeHandlers{DB: db, Settings: settings}
	router := gin.New()
	router.GET("/api/posts/:id", handler.OptionalAuthWithNoteGuestRestriction(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/posts/1", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body=%s", rec.Code, rec.Body.String())
	}
}

func TestNoteGuestRestrictionAllowsPublicAccessExemptFeedRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), "guest_access_note_restricted", true) {
		t.Fatalf("failed to restrict note guest access")
	}

	handler := NativeHandlers{Settings: settings}
	router := gin.New()
	router.GET("/api/posts/recommended", handler.OptionalAuthWithNoteGuestRestriction(), func(c *gin.Context) {
		if !publicAccessExemptOnly(c) {
			t.Fatalf("public access exempt list flag was not set")
		}
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/posts/recommended", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", rec.Code, rec.Body.String())
	}
}

func TestNoteGuestRestrictionAllowsTaxonomyForPublicAccessPreview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), "guest_access_note_restricted", true) {
		t.Fatalf("failed to restrict note guest access")
	}

	handler := NativeHandlers{Settings: settings}
	router := gin.New()
	router.GET("/api/categories", handler.OptionalAuthWithNoteGuestRestriction(), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/categories", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204, body=%s", rec.Code, rec.Body.String())
	}
}

func TestFileAccessAllowsPublicAccessExemptPostImageWithoutSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	temp := t.TempDir()
	imageDir := filepath.Join(temp, "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatalf("mkdir image dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "sample.webp"), []byte("public-image"), 0644); err != nil {
		t.Fatalf("write sample: %v", err)
	}

	db := openPublicAccessExemptTestDB(t, &domain.Post{}, &domain.PostImage{}, &domain.PostVideo{}, &domain.PostPaymentSetting{})
	post := domain.Post{
		UserID:             1,
		Title:              "Public image",
		Type:               1,
		Visibility:         "public",
		PublicAccessExempt: true,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Create(&domain.PostImage{PostID: post.ID, ImageURL: "/api/file/images/sample.webp"}).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}
	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), "guest_access_note_restricted", true) {
		t.Fatalf("failed to restrict note guest access")
	}
	cfg := config.Config{Upload: config.UploadConfig{
		RootDir: filepath.Join(temp, "upload-root"),
		Image:   config.UploadImageConfig{LocalUploadDir: imageDir},
	}}
	handler := NativeHandlers{
		Config:      cfg,
		DB:          db,
		Settings:    settings,
		UploadPaths: NewUploadPathResolver(cfg),
	}
	router := gin.New()
	router.GET("/api/file/*filepath", handler.FileAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/file/images/sample.webp", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != "public-image" {
		t.Fatalf("body = %q, want public image bytes", got)
	}
}

func TestFileAccessDoesNotExposeHiddenPaidImagesForPublicAccessExemptPost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	temp := t.TempDir()
	imageDir := filepath.Join(temp, "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatalf("mkdir image dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "paid.webp"), []byte("paid-image"), 0644); err != nil {
		t.Fatalf("write sample: %v", err)
	}

	db := openPublicAccessExemptTestDB(t, &domain.Post{}, &domain.PostImage{}, &domain.PostVideo{}, &domain.PostPaymentSetting{})
	post := domain.Post{
		UserID:             1,
		Title:              "Paid image",
		Type:               1,
		Visibility:         "public",
		PublicAccessExempt: true,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Create(&domain.PostImage{PostID: post.ID, ImageURL: "/api/file/images/paid.webp", IsFreePreview: false}).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}
	if err := db.Create(&domain.PostPaymentSetting{PostID: post.ID, Enabled: true, Price: 9.9, PaymentMethod: "balance"}).Error; err != nil {
		t.Fatalf("create payment setting: %v", err)
	}
	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), "guest_access_note_restricted", true) {
		t.Fatalf("failed to restrict note guest access")
	}
	cfg := config.Config{Upload: config.UploadConfig{
		RootDir: filepath.Join(temp, "upload-root"),
		Image:   config.UploadImageConfig{LocalUploadDir: imageDir},
	}}
	handler := NativeHandlers{
		Config:      cfg,
		DB:          db,
		Settings:    settings,
		UploadPaths: NewUploadPathResolver(cfg),
	}
	router := gin.New()
	router.GET("/api/file/*filepath", handler.FileAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/file/images/paid.webp", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body=%s", rec.Code, rec.Body.String())
	}
}

func TestFileAccessSignedURLSkipsPublicAccessExemptDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	temp := t.TempDir()
	imageDir := filepath.Join(temp, "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatalf("mkdir image dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "signed.webp"), []byte("signed-image"), 0644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	db := openPublicAccessExemptTestDB(t, &domain.Post{}, &domain.PostImage{}, &domain.PostVideo{}, &domain.PostPaymentSetting{})
	var queries atomic.Int64
	registerQueryCounter(t, db, &queries, 0)

	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), "guest_access_note_restricted", true) {
		t.Fatalf("failed to restrict note guest access")
	}
	cfg := config.Config{
		Auth: config.AuthConfig{JWTSecret: "jwt-secret"},
		Upload: config.UploadConfig{
			FileSigning: config.UploadFileSigningConfig{Secret: "file-secret", TTL: time.Hour},
			RootDir:     filepath.Join(temp, "upload-root"),
			Image:       config.UploadImageConfig{LocalUploadDir: imageDir},
		},
	}
	handler := NativeHandlers{
		Config:          cfg,
		DB:              db,
		Settings:        settings,
		UploadPaths:     NewUploadPathResolver(cfg),
		FileAccessGroup: &singleflight.Group{},
	}
	router := gin.New()
	router.GET("/api/file/*filepath", handler.FileAccess)

	req := httptest.NewRequest(http.MethodGet, handler.signFileURL("/api/file/images/signed.webp"), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if queries.Load() != 0 {
		t.Fatalf("signed file access executed %d DB queries, want 0", queries.Load())
	}
}

func TestFileAccessPublicAccessExemptUsesRedisCacheAndVersionInvalidation(t *testing.T) {
	db := openPublicAccessExemptTestDB(t, &domain.Post{}, &domain.PostImage{}, &domain.PostVideo{}, &domain.PostPaymentSetting{})
	path := seedPublicAccessExemptImage(t, db, "/api/file/images/cache.webp")
	var queries atomic.Int64
	registerQueryCounter(t, db, &queries, 0)

	redisServer := miniredis.RunT(t)
	store := services.NewRedisStore(config.RedisConfig{
		Addr:         redisServer.Addr(),
		CacheEnabled: true,
		CacheL1TTL:   time.Second,
	})
	handler := NativeHandlers{DB: db, Redis: store, FileAccessGroup: &singleflight.Group{}}
	ctx := context.Background()

	if !handler.fileBelongsToPublicAccessExemptPost(ctx, path) {
		t.Fatalf("first access should be allowed")
	}
	firstQueries := queries.Load()
	if firstQueries == 0 {
		t.Fatalf("first access did not query DB")
	}
	if !handler.fileBelongsToPublicAccessExemptPost(ctx, path) {
		t.Fatalf("second access should be allowed from cache")
	}
	if got := queries.Load(); got != firstQueries {
		t.Fatalf("second access queried DB: got %d queries, want %d", got, firstQueries)
	}

	if err := db.Where("image_url = ?", path).Delete(&domain.PostImage{}).Error; err != nil {
		t.Fatalf("delete image: %v", err)
	}
	handler.bumpCacheVersions(cacheScopeFileAccess)
	if handler.fileBelongsToPublicAccessExemptPost(ctx, path) {
		t.Fatalf("access should be denied after version invalidation and DB source removal")
	}
	if got := queries.Load(); got <= firstQueries {
		t.Fatalf("version invalidation did not force DB reload: got %d, first %d", got, firstQueries)
	}
}

func TestFileAccessPublicAccessExemptColdMissUsesSingleflight(t *testing.T) {
	db := openSharedPublicAccessExemptTestDB(t, &domain.Post{}, &domain.PostImage{}, &domain.PostVideo{}, &domain.PostPaymentSetting{})
	path := seedPublicAccessExemptImage(t, db, "/api/file/images/singleflight.webp")
	var queries atomic.Int64
	registerQueryCounter(t, db, &queries, 50*time.Millisecond)
	handler := NativeHandlers{DB: db, FileAccessGroup: &singleflight.Group{}}

	const workers = 16
	start := make(chan struct{})
	var wg sync.WaitGroup
	var failures atomic.Int64
	for range workers {
		wg.Go(func() {
			<-start
			if !handler.fileBelongsToPublicAccessExemptPost(context.Background(), path) {
				failures.Add(1)
			}
		})
	}
	close(start)
	wg.Wait()

	if failures.Load() != 0 {
		t.Fatalf("%d workers were denied", failures.Load())
	}
	if got := queries.Load(); got != 1 {
		t.Fatalf("DB query count = %d, want 1 singleflight loader", got)
	}
}

func openPublicAccessExemptTestDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	return db
}

func openSharedPublicAccessExemptTestDB(t *testing.T, models ...any) *gorm.DB {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(4)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	return db
}

func seedPublicAccessExemptImage(t *testing.T, db *gorm.DB, imageURL string) string {
	t.Helper()
	post := domain.Post{
		UserID:             1,
		Title:              "Public image",
		Type:               1,
		Visibility:         "public",
		PublicAccessExempt: true,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	if err := db.Create(&domain.PostImage{PostID: post.ID, ImageURL: imageURL}).Error; err != nil {
		t.Fatalf("create image: %v", err)
	}
	return imageURL
}

func registerQueryCounter(t *testing.T, db *gorm.DB, queries *atomic.Int64, delay time.Duration) {
	t.Helper()
	name := "test_query_counter:" + strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	if err := db.Callback().Query().Before("gorm:query").Register(name, func(tx *gorm.DB) {
		queries.Add(1)
		if delay > 0 {
			time.Sleep(delay)
		}
	}); err != nil {
		t.Fatalf("register query callback: %v", err)
	}
}
