package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/services"
)

func TestNormalizeImageSetting(t *testing.T) {
	tests := []struct {
		key   string
		value any
		ok    bool
		want  any
	}{
		{key: "image_webp_quality", value: 85, ok: true, want: 85},
		{key: "image_webp_quality", value: 101, ok: false},
		{key: "image_webp_method", value: "6", ok: true, want: 6},
		{key: "image_webp_method", value: -1, ok: false},
		{key: "image_webp_alpha_quality", value: 0, ok: true, want: 0},
		{key: "image_libvips_enabled", value: "true", ok: true, want: true},
		{key: "image_max_width", value: 16384, ok: true, want: 16384},
		{key: "image_max_height", value: 16385, ok: false},
		{key: "image_processing_concurrency", value: 1, ok: true, want: 1},
		{key: "image_processing_concurrency", value: 9, ok: false},
		{key: "image_post_max_count", value: 100, ok: true, want: 100},
		{key: "image_post_max_count", value: 501, ok: false},
		{key: "image_archive_enabled", value: "true", ok: true, want: true},
		{key: "image_archive_threshold", value: 25, ok: true, want: 25},
		{key: "image_archive_threshold", value: 0, ok: false},
		{key: "hidden_watermark_enabled", value: "true", ok: true, want: true},
		{key: "hidden_watermark_protected_only", value: "true", ok: true, want: true},
		{key: "hidden_watermark_engine", value: "auto", ok: true, want: "auto"},
		{key: "hidden_watermark_engine", value: "local", ok: true, want: "local"},
		{key: "hidden_watermark_engine", value: "remote", ok: true, want: "remote"},
		{key: "hidden_watermark_engine", value: "hybrid", ok: false},
		{key: "hidden_watermark_profile", value: "current", ok: true, want: "current"},
		{key: "hidden_watermark_profile", value: "author_recommended", ok: true, want: "author_recommended"},
		{key: "hidden_watermark_profile", value: "robust", ok: true, want: "robust"},
		{key: "hidden_watermark_profile", value: "legacy", ok: false},
		{key: "hidden_watermark_block_width", value: 5, ok: true, want: 6},
		{key: "hidden_watermark_block_height", value: "64", ok: true, want: 64},
		{key: "hidden_watermark_block_height", value: 65, ok: false},
		{key: "hidden_watermark_coefficient_mode", value: "d1", ok: true, want: "d1"},
		{key: "hidden_watermark_coefficient_mode", value: "d1d2", ok: true, want: "d1d2"},
		{key: "hidden_watermark_coefficient_mode", value: "d2", ok: false},
		{key: "hidden_watermark_d1", value: 21, ok: true, want: 21},
		{key: "hidden_watermark_d2", value: 9, ok: true, want: 9},
		{key: "hidden_watermark_ecc_mode", value: "golay", ok: true, want: "golay"},
		{key: "hidden_watermark_ecc_mode", value: "none", ok: true, want: "none"},
		{key: "hidden_watermark_ecc_mode", value: "reed_solomon", ok: false},
		{key: "hidden_watermark_golay_seed", value: 1234567890, ok: true, want: 1234567890},
		{key: "hidden_watermark_remote_password_wm", value: 1, ok: true, want: 1},
		{key: "hidden_watermark_remote_password_img", value: 9987, ok: true, want: 9987},
		{key: "hidden_watermark_remote_profile", value: "adaptive", ok: true, want: "adaptive"},
		{key: "hidden_watermark_remote_profile", value: "official", ok: true, want: "official"},
		{key: "hidden_watermark_remote_profile", value: "current", ok: false},
		{key: "hidden_watermark_remote_engine", value: "dwt_dct_svd", ok: true, want: "dwt_dct_svd"},
		{key: "hidden_watermark_remote_engine", value: "blind_watermark", ok: true, want: "blind_watermark"},
		{key: "hidden_watermark_remote_engine", value: "local", ok: false},
		{key: "hidden_watermark_remote_custom_d1", value: 18, ok: true, want: 18},
		{key: "hidden_watermark_remote_custom_d2", value: 8, ok: true, want: 8},
		{key: "hidden_watermark_remote_d1", value: 36, ok: false},
		{key: "hidden_watermark_remote_timeout_seconds", value: 50, ok: true, want: 50},
		{key: "hidden_watermark_remote_timeout_seconds", value: 9, ok: false},
		{key: "hidden_watermark_remote_operation_timeout_seconds", value: 45, ok: true, want: 45},
		{key: "hidden_watermark_remote_operation_timeout_seconds", value: 301, ok: false},
		{key: "image_protection_max_dimension", value: 0, ok: true, want: 0},
		{key: "image_protection_max_dimension", value: 2048, ok: true, want: 2048},
		{key: "image_protection_output_mode", value: "lossless_webp", ok: true, want: "lossless_webp"},
		{key: "image_protection_output_mode", value: "quality_webp", ok: true, want: "quality_webp"},
		{key: "image_protection_webp_quality", value: 95, ok: true, want: 95},
		{key: "hidden_watermark_extract_all_users", value: "true", ok: true, want: true},
		{key: "hidden_watermark_extract_user_ids", value: "7, account_7\nxise_7，7", ok: true, want: []string{"7", "account_7", "xise_7"}},
		{key: "hidden_watermark_extract_usernames", value: []any{"Alice", "alice", "Bob"}, ok: true, want: []string{"Alice", "Bob"}},
		{key: "image_protection_enabled", value: "true", ok: true, want: true},
		{key: "image_protection_notice_enabled", value: "false", ok: true, want: false},
		{key: "image_select_all_enabled", value: false, ok: true, want: false},
		{key: "paid_content_balance_enabled", value: "false", ok: true, want: false},
		{key: "paid_content_points_enabled", value: true, ok: true, want: true},
		{key: "paid_content_balance_max_price", value: "2000", ok: true, want: 2000},
		{key: "paid_content_points_max_price", value: 50000, ok: true, want: 50000},
		{key: "paid_content_balance_max_price", value: 0, ok: false},
		{key: "paid_content_points_max_price", value: 1000001, ok: false},
		{key: "hidden_watermark_custom_text", value: "yuem", ok: true, want: "yuem"},
		{key: "hidden_watermark_custom_text", value: "水印", ok: true, want: "水印"},
		{key: "hidden_watermark_custom_text", value: strings.Repeat("水", 256), ok: false},
		{key: "hidden_watermark_unknown", value: true, ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got, ok := normalizeImageSetting(tt.key, tt.value)
			if ok != tt.ok {
				t.Fatalf("normalizeImageSetting(%q, %v) ok = %v, want %v", tt.key, tt.value, ok, tt.ok)
			}
			if tt.ok && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("normalizeImageSetting(%q, %v) = %v, want %v", tt.key, tt.value, got, tt.want)
			}
		})
	}
}

func TestReadWatermarkUploadReadsScreenshotAndReference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range map[string]string{
		"file":           "screenshot",
		"reference_file": "reference",
	} {
		part, err := writer.CreateFormFile(name, name+".png")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/image-watermark/extract", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = request

	data, reference, meta, cleanup, err := (NativeHandlers{}).readWatermarkUpload(context)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		t.Fatalf("readWatermarkUpload() error = %v", err)
	}
	if string(data) != "screenshot" || string(reference) != "reference" {
		t.Fatalf("uploads = %q / %q", data, reference)
	}
	if meta.Filename != "file.png" || meta.Size != int64(len(data)) {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestExtractImageWatermarkStreamsRealProgress(t *testing.T) {
	gin.SetMode(gin.TestMode)
	token := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	tokenHex := hex.EncodeToString(token)
	shortCode := []byte{0xa1, 0xb2, 0xc3, 0xd4}
	shortCodeHex := hex.EncodeToString(shortCode)
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/watermark/extract-stream" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/x-ndjson")
		_, _ = w.Write([]byte(`{"type":"progress","stage":"extracting","percent":50,"completed":1,"total":2}` + "\n"))
		_, _ = w.Write([]byte(`{"type":"result","payload_b64":"` + base64.StdEncoding.EncodeToString(shortCode) + `","payload_bytes":4}` + "\n"))
	}))
	defer remote.Close()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "watermark.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.AutoMigrate(&domain.ImageWatermarkTrace{}, &domain.SecurityAuditLog{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.ImageWatermarkTrace{
		Token:          tokenHex,
		ShortCode:      &shortCodeHex,
		ShortCodeBytes: domain.ImageWatermarkShortCodeBytes,
		TraceType:      domain.ImageWatermarkTraceUpload,
		PayloadVersion: domain.ImageWatermarkPayloadVersion,
		PayloadBytes:   domain.ImageWatermarkPayloadBytes,
	}).Error; err != nil {
		t.Fatal(err)
	}

	var imageData bytes.Buffer
	if err := jpeg.Encode(&imageData, image.NewRGBA(image.Rect(0, 0, 512, 512)), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "image.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(imageData.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	settings := services.NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_remote_timeout_seconds", 10)
	processor := services.NewImageProcessorWithRemote(
		settings,
		"stream-secret",
		10<<20,
		nil,
		services.HiddenWatermarkRemoteClientConfig{URL: remote.URL, Timeout: time.Second},
	)
	handler := NativeHandlers{
		Settings: settings,
		Images:   processor,
		DB:       db,
		Config: config.Config{
			Upload: config.UploadConfig{Image: config.UploadImageConfig{MaxSizeBytes: 10 << 20}},
		},
	}
	request := httptest.NewRequest(http.MethodPost, "/api/admin/image-watermark/extract", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Accept", "application/x-ndjson")
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request

	handler.adminExtractImageWatermark(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), `"completed":1,"total":2`) ||
		!strings.Contains(recorder.Body.String(), `"type":"result"`) ||
		!strings.Contains(recorder.Body.String(), `"traceToken":"`+tokenHex+`"`) {
		t.Fatalf("unexpected stream: %s", recorder.Body.String())
	}
}

func TestUserCanExtractImageWatermark(t *testing.T) {
	ctx := context.Background()
	settings := services.NewSettingsService(nil, nil)
	handler := NativeHandlers{Settings: settings}
	xiseID := "xise_7"
	user := &services.RequestUser{
		ID:       7,
		UserID:   "account_7",
		XiseID:   &xiseID,
		Nickname: "Alice",
		Username: "alice_name",
		Type:     "user",
	}

	if handler.userCanExtractImageWatermark(user) {
		t.Fatal("default user access = true, want false")
	}

	_ = settings.Set(ctx, "hidden_watermark_extract_all_users", true)
	if !handler.userCanExtractImageWatermark(user) {
		t.Fatal("all users access = false, want true")
	}

	_ = settings.Set(ctx, "hidden_watermark_extract_all_users", false)
	_ = settings.Set(ctx, "hidden_watermark_extract_user_ids", []string{"8", "account_7"})
	if !handler.userCanExtractImageWatermark(user) {
		t.Fatal("public user id access = false, want true")
	}

	_ = settings.Set(ctx, "hidden_watermark_extract_user_ids", []string{"xise_7"})
	if !handler.userCanExtractImageWatermark(user) {
		t.Fatal("xise id access = false, want true")
	}

	_ = settings.Set(ctx, "hidden_watermark_extract_user_ids", []string{})
	_ = settings.Set(ctx, "hidden_watermark_extract_usernames", []string{"alice"})
	if !handler.userCanExtractImageWatermark(user) {
		t.Fatal("nickname case-insensitive access = false, want true")
	}

	_ = settings.Set(ctx, "hidden_watermark_extract_usernames", []string{"alice_name"})
	if !handler.userCanExtractImageWatermark(user) {
		t.Fatal("username access = false, want true")
	}

	admin := &services.RequestUser{ID: 1, Type: "admin"}
	if !handler.userCanExtractImageWatermark(admin) {
		t.Fatal("admin access = false, want true")
	}
}

func TestImageWatermarkExtractRequiresAdmin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	handler := NativeHandlers{}
	engine.POST(
		"/api/admin/image-watermark/extract",
		handler.MatrixRoute("backend/routes/admin.js", http.MethodPost, "/api/admin/image-watermark/extract", "admin"),
	)

	req := httptest.NewRequest(http.MethodPost, "/api/admin/image-watermark/extract", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401, body=%s", rec.Code, rec.Body.String())
	}
}

func TestImageWatermarkExtractRemovesRequestWorkDir(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tempRoot := filepath.Join(t.TempDir(), "uploads", "tmp")
	tempStorage := services.NewTempStorageService(config.UploadTempConfig{RootDir: tempRoot})
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.SecurityAuditLog{}); err != nil {
		t.Fatal(err)
	}
	handler := NativeHandlers{
		DB: db,
		Config: config.Config{Upload: config.UploadConfig{
			Image: config.UploadImageConfig{MaxSizeBytes: 1 << 20},
		}},
		TempStorage: tempStorage,
	}
	engine := gin.New()
	engine.POST("/extract", func(c *gin.Context) {
		c.Set("user", &services.RequestUser{ID: 7, UserID: "account_7", Type: "user"})
		handler.extractImageWatermark(c)
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "invalid.webp")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("not-an-image")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/extract", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}

	workRoot := filepath.Join(tempRoot, "watermark-inspector")
	entries, err := os.ReadDir(workRoot)
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("watermark inspector temporary work directories remain: %v", entries)
	}
	var audit domain.SecurityAuditLog
	if err := db.Where("category = ?", "hidden_watermark").Take(&audit).Error; err != nil {
		t.Fatalf("watermark usage audit missing: %v", err)
	}
	if audit.Action != "extract" || audit.Outcome != "failure" || audit.ActorID == nil || *audit.ActorID != 7 {
		t.Fatalf("unexpected watermark usage audit: %+v", audit)
	}
}

func TestImageWatermarkExtractRoutesUseRemoteEngine(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const secret = "handler-remote-secret"
	token := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	tokenHex := hex.EncodeToString(token)
	shortCode := []byte{0xa1, 0xb2, 0xc3, 0xd4}
	shortCodeHex := hex.EncodeToString(shortCode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "watermark.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.AutoMigrate(&domain.ImageWatermarkTrace{}, &domain.SecurityAuditLog{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.ImageWatermarkTrace{
		Token:          tokenHex,
		ShortCode:      &shortCodeHex,
		ShortCodeBytes: domain.ImageWatermarkShortCodeBytes,
		TraceType:      domain.ImageWatermarkTraceUpload,
		PayloadVersion: domain.ImageWatermarkPayloadVersion,
		PayloadBytes:   domain.ImageWatermarkPayloadBytes,
		FieldFlags:     domain.ImageWatermarkFieldUID | domain.ImageWatermarkFieldUserID,
		UserID:         42,
		UserDisplayID:  "remote-u",
	}).Error; err != nil {
		t.Fatal(err)
	}
	remoteCalls := 0
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/watermark/extract" {
			http.NotFound(w, r)
			return
		}
		if got := r.Header.Get("X-Internal-API-Key"); got != "remote-key" {
			t.Fatalf("X-Internal-API-Key = %q, want remote-key", got)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		remoteCalls++
		if got := r.FormValue("payload_bytes_candidates"); got != "4,3,2" {
			t.Fatalf("unexpected payload_bytes_candidates %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"payload_b64":   base64.StdEncoding.EncodeToString(shortCode),
			"payload_bytes": len(shortCode),
			"payload_groups": []map[string]any{
				{"payload_bytes": len(shortCode), "payload_candidates_b64": []string{base64.StdEncoding.EncodeToString(shortCode)}},
			},
		})
	}))
	defer remote.Close()

	settings := services.NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_engine", "remote")
	_ = settings.Set(context.Background(), "hidden_watermark_extract_all_users", true)
	handler := NativeHandlers{
		DB: db,
		Config: config.Config{
			Upload: config.UploadConfig{
				Image: config.UploadImageConfig{MaxSizeBytes: 1 << 20},
				Temp:  config.UploadTempConfig{RootDir: filepath.Join(t.TempDir(), "tmp")},
			},
		},
		Settings: settings,
		Images: services.NewImageProcessorWithRemote(
			settings,
			secret,
			1<<20,
			nil,
			services.HiddenWatermarkRemoteClientConfig{
				URL:     remote.URL,
				APIKey:  "remote-key",
				Timeout: time.Second,
			},
		),
	}
	handler.TempStorage = services.NewTempStorageService(handler.Config.Upload.Temp)

	tests := []struct {
		name string
		path string
		user *services.RequestUser
		call func(*gin.Context)
	}{
		{
			name: "admin detector",
			path: "/api/admin/image-watermark/extract",
			user: &services.RequestUser{ID: 1, Type: "admin"},
			call: handler.adminExtractImageWatermark,
		},
		{
			name: "frontend detector",
			path: "/api/image-watermark/extract",
			user: &services.RequestUser{ID: 7, UserID: "account_7", Type: "user"},
			call: handler.UserExtractImageWatermark,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := gin.New()
			engine.POST(tt.path, func(c *gin.Context) {
				c.Set("user", tt.user)
				tt.call(c)
			})
			body, contentType := multipartImageBody(t)
			req := httptest.NewRequest(http.MethodPost, tt.path, body)
			req.Header.Set("Content-Type", contentType)
			rec := httptest.NewRecorder()
			before := remoteCalls
			engine.ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
			}
			if remoteCalls-before != 1 {
				t.Fatalf("remote extract calls = %d, want 1", remoteCalls-before)
			}
			var envelope struct {
				Data services.HiddenWatermarkData `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if !envelope.Data.Found || !envelope.Data.Valid || envelope.Data.TraceToken != tokenHex || envelope.Data.UID != 42 || envelope.Data.UserID != "remote-u" {
				t.Fatalf("unexpected remote extraction result: %+v body=%s", envelope.Data, rec.Body.String())
			}
		})
	}
}

func TestImageWatermarkExtractResolvesRemoteShortCode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	token := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	tokenHex := hex.EncodeToString(token)
	shortCode := []byte{0xa1, 0xb2, 0xc3, 0xd4}
	shortCodeHex := hex.EncodeToString(shortCode)
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "watermark.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.AutoMigrate(&domain.ImageWatermarkTrace{}, &domain.SecurityAuditLog{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.ImageWatermarkTrace{
		Token:           tokenHex,
		ShortCode:       &shortCodeHex,
		ShortCodeBytes:  domain.ImageWatermarkShortCodeBytes,
		TraceType:       domain.ImageWatermarkTraceProtected,
		PayloadVersion:  domain.ImageWatermarkPayloadVersion,
		PayloadBytes:    domain.ImageWatermarkPayloadBytes,
		FieldFlags:      domain.ImageWatermarkFieldUID | domain.ImageWatermarkFieldUserID,
		UserID:          42,
		UserDisplayID:   "remote-u",
		WatermarkWidth:  720,
		WatermarkHeight: 960,
	}).Error; err != nil {
		t.Fatal(err)
	}
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/watermark/extract" {
			http.NotFound(w, r)
			return
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if got := r.FormValue("payload_bytes_candidates"); got != "4,3,2" {
			http.Error(w, "payload candidates = "+got, http.StatusBadRequest)
			return
		}
		if got := r.FormValue("recover_dimensions"); !strings.Contains(got, "720x960") {
			http.Error(w, "recover dimensions = "+got, http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"payload_b64":   base64.StdEncoding.EncodeToString(shortCode),
			"payload_bytes": len(shortCode),
			"payload_groups": []map[string]any{
				{"payload_bytes": len(shortCode), "payload_candidates_b64": []string{base64.StdEncoding.EncodeToString(shortCode)}},
			},
		})
	}))
	defer remote.Close()

	settings := services.NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "hidden_watermark_engine", "remote")
	_ = settings.Set(context.Background(), "hidden_watermark_extract_all_users", true)
	handler := NativeHandlers{
		DB: db,
		Config: config.Config{
			Upload: config.UploadConfig{
				Image: config.UploadImageConfig{MaxSizeBytes: 1 << 20},
				Temp:  config.UploadTempConfig{RootDir: filepath.Join(t.TempDir(), "tmp")},
			},
		},
		Settings: settings,
		Images: services.NewImageProcessorWithRemote(
			settings,
			"short-code-secret",
			1<<20,
			nil,
			services.HiddenWatermarkRemoteClientConfig{URL: remote.URL, Timeout: time.Second},
		),
	}
	handler.TempStorage = services.NewTempStorageService(handler.Config.Upload.Temp)
	engine := gin.New()
	engine.POST("/api/admin/image-watermark/extract", func(c *gin.Context) {
		c.Set("user", &services.RequestUser{ID: 1, Type: "admin"})
		handler.adminExtractImageWatermark(c)
	})
	body, contentType := multipartImageBody(t)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/image-watermark/extract", body)
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data services.HiddenWatermarkData `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Data.Found || !envelope.Data.Valid || envelope.Data.TraceToken != tokenHex || envelope.Data.PayloadBytes != domain.ImageWatermarkShortCodeBytes || envelope.Data.PayloadFormat != "short_code_v1" {
		t.Fatalf("unexpected short-code extraction result: %+v body=%s", envelope.Data, rec.Body.String())
	}
}

func multipartImageBody(t *testing.T) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	contentType := writer.FormDataContentType()
	part, err := writer.CreateFormFile("file", "remote.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(jpegFixtureForHandlerTest(t)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return &body, contentType
}

func jpegFixtureForHandlerTest(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := range 64 {
		for x := range 64 {
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 4), G: uint8(y * 4), B: 160, A: 255})
		}
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
