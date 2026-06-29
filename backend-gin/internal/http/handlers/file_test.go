package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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

func TestFileAccessServesConfiguredUploadTypeDirOutsideRoot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	temp := t.TempDir()
	rootDir := filepath.Join(temp, "upload-root")
	imageDir := filepath.Join(temp, "image-local")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatalf("mkdir image dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "sample.webp"), []byte("webp-data"), 0644); err != nil {
		t.Fatalf("write sample: %v", err)
	}

	router := gin.New()
	cfg := config.Config{Upload: config.UploadConfig{
		RootDir: rootDir,
		Image:   config.UploadImageConfig{LocalUploadDir: imageDir},
	}}
	handler := NativeHandlers{Config: cfg, UploadPaths: NewUploadPathResolver(cfg)}
	router.GET("/api/file/*filepath", handler.FileAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/file/images/sample.webp", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != "webp-data" {
		t.Fatalf("body = %q, want file bytes", got)
	}
	if got := rec.Header().Get("Content-Type"); got != "image/webp" {
		t.Fatalf("Content-Type = %q, want image/webp", got)
	}
}

func TestFileAccessServesAttachmentsPubliclyWhenNoteGuestsAreRestricted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	temp := t.TempDir()
	attachmentDir := filepath.Join(temp, "attachment-local")
	if err := os.MkdirAll(attachmentDir, 0755); err != nil {
		t.Fatalf("mkdir attachment dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(attachmentDir, "sample.pdf"), []byte("pdf-data"), 0644); err != nil {
		t.Fatalf("write attachment: %v", err)
	}

	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), "guest_access_note_restricted", true) {
		t.Fatalf("failed to restrict note guest access")
	}
	cfg := config.Config{Upload: config.UploadConfig{
		RootDir:    filepath.Join(temp, "upload-root"),
		Attachment: config.UploadAttachmentConfig{LocalUploadDir: attachmentDir},
	}}
	handler := NativeHandlers{
		Config:      cfg,
		Settings:    settings,
		UploadPaths: NewUploadPathResolver(cfg),
	}
	router := gin.New()
	router.GET("/api/file/*filepath", handler.FileAccess)

	req := httptest.NewRequest(http.MethodGet, "/api/file/attachments/sample.pdf", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != "pdf-data" {
		t.Fatalf("body = %q, want file bytes", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "public, max-age=31536000" {
		t.Fatalf("Cache-Control = %q, want public long-lived cache", got)
	}
}

func TestFileAccessServesProfileHashAliasFromLegacyFilename(t *testing.T) {
	gin.SetMode(gin.TestMode)
	temp := t.TempDir()
	avatarDir := filepath.Join(temp, "avatars")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		t.Fatalf("mkdir avatar dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(avatarDir, "2068.webp"), []byte("avatar-data"), 0644); err != nil {
		t.Fatalf("write avatar: %v", err)
	}

	cfg := config.Config{Upload: config.UploadConfig{
		RootDir:   filepath.Join(temp, "upload-root"),
		AvatarDir: avatarDir,
	}}
	handler := NativeHandlers{Config: cfg, UploadPaths: NewUploadPathResolver(cfg)}
	router := gin.New()
	router.GET("/api/file/*filepath", handler.FileAccess)

	hash := profileImageXXH3([]byte("avatar-data"))
	for _, target := range []string{
		"/api/file/avatar/2068.webp",
		"/api/file/avatar/2068-" + hash + ".webp",
	} {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want 200, body=%s", target, rec.Code, rec.Body.String())
		}
		if got := rec.Body.String(); got != "avatar-data" {
			t.Fatalf("%s body = %q, want avatar-data", target, got)
		}
	}
}

func TestProfileImageStorageValueBackfillsLegacyURL(t *testing.T) {
	temp := t.TempDir()
	bannerDir := filepath.Join(temp, "banners")
	if err := os.MkdirAll(bannerDir, 0755); err != nil {
		t.Fatalf("mkdir banner dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bannerDir, "2068.webp"), []byte("banner-data"), 0644); err != nil {
		t.Fatalf("write banner: %v", err)
	}

	cfg := config.Config{Upload: config.UploadConfig{
		RootDir:   filepath.Join(temp, "upload-root"),
		BannerDir: bannerDir,
	}}
	handler := NativeHandlers{Config: cfg, UploadPaths: NewUploadPathResolver(cfg)}
	hash := profileImageXXH3([]byte("banner-data"))

	url, hashValue := handler.profileImageStorageValue("/api/file/banner/2068.webp", profileImageBackgroundColumn)
	if url != "/api/file/banner/2068-"+hash+".webp" {
		t.Fatalf("url = %q, want hash URL", url)
	}
	if hashValue != hash {
		t.Fatalf("hash = %#v, want %q", hashValue, hash)
	}
}

func TestSignFileURLPtrBackfillsLegacyProfileImageInDB(t *testing.T) {
	temp := t.TempDir()
	avatarDir := filepath.Join(temp, "avatars")
	if err := os.MkdirAll(avatarDir, 0755); err != nil {
		t.Fatalf("mkdir avatar dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(avatarDir, "2068.webp"), []byte("avatar-data"), 0644); err != nil {
		t.Fatalf("write avatar: %v", err)
	}
	db, err := gorm.Open(sqlite.Open("file:profile-image-hash-backfill?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	legacyURL := "/api/file/avatar/2068.webp"
	if err := db.Create(&domain.User{ID: 2068, UserID: "user-2068", Nickname: "User 2068", Avatar: &legacyURL}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	cfg := config.Config{
		Auth: config.AuthConfig{JWTSecret: "jwt-secret"},
		Upload: config.UploadConfig{
			RootDir:   filepath.Join(temp, "upload-root"),
			AvatarDir: avatarDir,
		},
	}
	handler := NativeHandlers{DB: db, Config: cfg, UploadPaths: NewUploadPathResolver(cfg)}
	hash := profileImageXXH3([]byte("avatar-data"))

	signed := handler.signFileURLPtr(&legacyURL)
	if signed == nil || !strings.Contains(*signed, "/api/file/avatar/2068-"+hash+".webp?") {
		t.Fatalf("signed URL = %#v, want hash URL", signed)
	}
	var updated domain.User
	if err := db.Where("id = ?", int64(2068)).First(&updated).Error; err != nil {
		t.Fatalf("load updated user: %v", err)
	}
	if got := stringPtrValue(updated.Avatar); got != "/api/file/avatar/2068-"+hash+".webp" {
		t.Fatalf("avatar = %q, want hash URL", got)
	}
	if got := stringPtrValue(updated.AvatarXXH3); got != hash {
		t.Fatalf("avatar_xxh3 = %q, want %q", got, hash)
	}
}

func TestUploadPathResolverPrecomputesAndDeduplicatesCandidates(t *testing.T) {
	rootDir := t.TempDir()
	imageDir := filepath.Join(rootDir, "images")
	cfg := config.Config{Upload: config.UploadConfig{
		RootDir: rootDir,
		Image:   config.UploadImageConfig{LocalUploadDir: imageDir},
	}}

	resolver := NewUploadPathResolver(cfg)
	handler := NativeHandlers{Config: cfg, UploadPaths: resolver}
	candidates := handler.uploadCandidateDirs("images")
	if len(candidates) != 1 {
		t.Fatalf("image candidate dirs = %#v, want one deduplicated directory", candidates)
	}
	if candidates[0] != filepath.Clean(imageDir) {
		t.Fatalf("image candidate dir = %q, want %q", candidates[0], filepath.Clean(imageDir))
	}
}

func TestOpenUploadFileCachesResolvedPathAndFallsBackWhenStale(t *testing.T) {
	temp := t.TempDir()
	rootDir := filepath.Join(temp, "upload-root")
	firstDir := filepath.Join(temp, "image-local")
	secondDir := filepath.Join(rootDir, "images")
	if err := os.MkdirAll(firstDir, 0755); err != nil {
		t.Fatalf("mkdir first dir: %v", err)
	}
	if err := os.MkdirAll(secondDir, 0755); err != nil {
		t.Fatalf("mkdir second dir: %v", err)
	}
	secondPath := filepath.Join(secondDir, "sample.webp")
	if err := os.WriteFile(secondPath, []byte("second"), 0644); err != nil {
		t.Fatalf("write second file: %v", err)
	}
	cfg := config.Config{Upload: config.UploadConfig{
		RootDir: rootDir,
		Image:   config.UploadImageConfig{LocalUploadDir: firstDir},
	}}
	handler := NativeHandlers{
		Config:      cfg,
		Cache:       services.NewCache(),
		UploadPaths: NewUploadPathResolver(cfg),
	}

	file, _, filePath, ok := handler.openUploadFile("images", "sample.webp")
	if !ok {
		t.Fatalf("openUploadFile did not find file in fallback dir")
	}
	_ = file.Close()
	if filePath != filepath.Clean(secondPath) {
		t.Fatalf("filePath = %q, want %q", filePath, filepath.Clean(secondPath))
	}
	if cached, ok := handler.Cache.Get(uploadPathCacheKey("images", "sample.webp")); !ok || cached != filepath.Clean(secondPath) {
		t.Fatalf("cached path = %#v, ok=%v, want %q", cached, ok, filepath.Clean(secondPath))
	}

	if err := os.Remove(secondPath); err != nil {
		t.Fatalf("remove stale cached file: %v", err)
	}
	firstPath := filepath.Join(firstDir, "sample.webp")
	if err := os.WriteFile(firstPath, []byte("first"), 0644); err != nil {
		t.Fatalf("write first file: %v", err)
	}
	file, _, filePath, ok = handler.openUploadFile("images", "sample.webp")
	if !ok {
		t.Fatalf("openUploadFile did not fall back after stale cache")
	}
	_ = file.Close()
	if filePath != filepath.Clean(firstPath) {
		t.Fatalf("filePath after stale cache = %q, want %q", filePath, filepath.Clean(firstPath))
	}
}

func TestAllowedAttachmentTypeAcceptsAudio(t *testing.T) {
	for _, contentType := range []string{
		"audio/mpeg",
		"audio/mp4",
		"audio/wav",
		"audio/flac",
		"audio/aac",
	} {
		if !allowedAttachmentType(contentType) {
			t.Fatalf("allowedAttachmentType(%q) = false, want true", contentType)
		}
	}

	if allowedAttachmentType("video/mp4") {
		t.Fatalf("allowedAttachmentType(%q) = true, want false", "video/mp4")
	}
}

func TestAllowedAttachmentFileAcceptsMobileconfigFallbackTypes(t *testing.T) {
	for _, contentType := range []string{
		"application/x-apple-aspen-config",
		"application/octet-stream",
		"text/xml",
		"application/xml",
		"",
	} {
		if !allowedAttachmentFile("profile.mobileconfig", contentType) {
			t.Fatalf("allowedAttachmentFile(profile.mobileconfig, %q) = false, want true", contentType)
		}
	}

	if allowedAttachmentFile("profile.bin", "application/octet-stream") {
		t.Fatalf("allowedAttachmentFile(profile.bin, application/octet-stream) = true, want false")
	}
}

func TestContentTypeForPathHandlesMobileconfig(t *testing.T) {
	if got := contentTypeForPath("profile.mobileconfig"); got != "application/x-apple-aspen-config" {
		t.Fatalf("contentTypeForPath(profile.mobileconfig) = %q, want application/x-apple-aspen-config", got)
	}
}

func TestFileAccessAcceptsShortSignatureWhenGuestRestricted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	temp := t.TempDir()
	imageDir := filepath.Join(temp, "image-local")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatalf("mkdir image dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(imageDir, "sample.webp"), []byte("webp-data"), 0644); err != nil {
		t.Fatalf("write sample: %v", err)
	}

	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), "guest_access_note_restricted", true) {
		t.Fatalf("failed to restrict note guest access")
	}
	handler := NativeHandlers{Config: config.Config{
		Auth: config.AuthConfig{JWTSecret: "jwt-secret"},
		Upload: config.UploadConfig{
			FileSigning: config.UploadFileSigningConfig{Secret: "file-secret", TTL: time.Hour},
			Image:       config.UploadImageConfig{LocalUploadDir: imageDir},
			RootDir:     filepath.Join(temp, "upload-root"),
		},
	}, Settings: settings}
	router := gin.New()
	router.GET("/api/file/*filepath", handler.FileAccess)

	signedPath := handler.signFileURL("/api/file/images/sample.webp")
	req := httptest.NewRequest(http.MethodGet, signedPath, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != "webp-data" {
		t.Fatalf("body = %q, want file bytes", got)
	}
}

func TestDASHManifestRewritesSegmentTemplateWithReusableManifestSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)
	temp := t.TempDir()
	videoDir := filepath.Join(temp, "video-local")
	dashSubPath := filepath.Join("dash", "2026-05-13", "1497", "1778703092050")
	dashDir := filepath.Join(videoDir, dashSubPath)
	if err := os.MkdirAll(dashDir, 0755); err != nil {
		t.Fatalf("mkdir dash dir: %v", err)
	}
	manifest := `<?xml version="1.0" encoding="UTF-8"?>
<MPD xmlns="urn:mpeg:dash:schema:mpd:2011" type="static">
  <Period>
    <AdaptationSet>
      <Representation id="4" bandwidth="800000">
        <SegmentTemplate timescale="1000" initialization="init-stream$RepresentationID$.m4s" media="chunk-stream$RepresentationID$-$Number%05d$.m4s" startNumber="1" />
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`
	if err := os.WriteFile(filepath.Join(dashDir, "manifest.mpd"), []byte(manifest), 0644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dashDir, "init-stream4.m4s"), []byte("init-data"), 0644); err != nil {
		t.Fatalf("write init segment: %v", err)
	}

	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), "guest_access_video_restricted", true) {
		t.Fatalf("failed to restrict video guest access")
	}
	handler := NativeHandlers{Config: config.Config{
		Auth: config.AuthConfig{JWTSecret: "jwt-secret"},
		Upload: config.UploadConfig{
			FileSigning: config.UploadFileSigningConfig{Secret: "file-secret", TTL: time.Hour},
			Video:       config.UploadVideoConfig{LocalUploadDir: videoDir},
			RootDir:     filepath.Join(temp, "upload-root"),
		},
	}, Settings: settings}
	router := gin.New()
	router.GET("/api/file/*filepath", handler.FileAccess)

	signedManifest := handler.signFileURL("/api/file/videos/dash/2026-05-13/1497/1778703092050/manifest.mpd")
	req := httptest.NewRequest(http.MethodGet, signedManifest, nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("manifest status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, `initialization="init-stream$RepresentationID$.m4s?pvimg_exp=`) {
		t.Fatalf("manifest initialization URL was not signed: %s", body)
	}
	if !strings.Contains(body, `media="chunk-stream$RepresentationID$-$Number%05d$.m4s?pvimg_exp=`) {
		t.Fatalf("manifest media URL was not signed: %s", body)
	}

	query := extractEscapedDASHQuery(t, body, `init-stream$RepresentationID$.m4s?`)
	segmentURL := "/api/file/videos/dash/2026-05-13/1497/1778703092050/init-stream4.m4s?" + query
	req = httptest.NewRequest(http.MethodGet, segmentURL, nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("segment status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if got := rec.Body.String(); got != "init-data" {
		t.Fatalf("segment body = %q, want init-data", got)
	}
}

func TestDASHManifestRewritesSegmentURLAndPreservesExistingQuery(t *testing.T) {
	handler := NativeHandlers{Config: config.Config{
		Auth: config.AuthConfig{JWTSecret: "jwt-secret"},
		Upload: config.UploadConfig{
			FileSigning: config.UploadFileSigningConfig{Secret: "file-secret", TTL: time.Hour},
		},
	}}
	manifestPath := "/api/file/videos/dash/2026-05-13/1497/1778703092050/manifest.mpd"
	manifest := `<MPD xmlns="urn:mpeg:dash:schema:mpd:2011">
  <Period>
    <AdaptationSet>
      <Representation>
        <SegmentList>
          <Initialization sourceURL="/api/file/videos/dash/2026-05-13/1497/1778703092050/init-stream4.m4s?token=keep#frag" />
          <SegmentURL media="chunk-stream4-00001.m4s?range=0-99" />
          <SegmentURL media="https://cdn.example.com/chunk-stream4-00002.m4s" />
        </SegmentList>
      </Representation>
    </AdaptationSet>
  </Period>
</MPD>`

	rewritten, ok := handler.signDASHManifestReferences([]byte(manifest), manifestPath, time.Unix(1000, 0))
	if !ok {
		t.Fatalf("manifest was not rewritten")
	}
	body := string(rewritten)
	if !strings.Contains(body, `sourceURL="/api/file/videos/dash/2026-05-13/1497/1778703092050/init-stream4.m4s?token=keep&amp;pvimg_exp=`) {
		t.Fatalf("absolute local sourceURL was not signed while preserving query: %s", body)
	}
	if !strings.Contains(body, `#frag"`) {
		t.Fatalf("sourceURL fragment was not preserved: %s", body)
	}
	if !strings.Contains(body, `media="chunk-stream4-00001.m4s?range=0-99&amp;pvimg_exp=`) {
		t.Fatalf("SegmentURL media was not signed while preserving query: %s", body)
	}
	if strings.Contains(body, `cdn.example.com/chunk-stream4-00002.m4s?pvimg_exp=`) {
		t.Fatalf("external CDN URL should not be rewritten: %s", body)
	}
}

func TestFileAccessRejectsMissingExpiredAndTamperedSignatureWhenGuestRestricted(t *testing.T) {
	gin.SetMode(gin.TestMode)
	temp := t.TempDir()
	imageDir := filepath.Join(temp, "image-local")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatalf("mkdir image dir: %v", err)
	}
	for _, name := range []string{"sample.webp", "other.webp"} {
		if err := os.WriteFile(filepath.Join(imageDir, name), []byte(name), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), "guest_access_note_restricted", true) {
		t.Fatalf("failed to restrict note guest access")
	}
	handler := NativeHandlers{Config: config.Config{
		Auth: config.AuthConfig{JWTSecret: "jwt-secret"},
		Upload: config.UploadConfig{
			FileSigning: config.UploadFileSigningConfig{Secret: "file-secret", TTL: time.Minute},
			Image:       config.UploadImageConfig{LocalUploadDir: imageDir},
			RootDir:     filepath.Join(temp, "upload-root"),
		},
	}, Settings: settings}
	router := gin.New()
	router.GET("/api/file/*filepath", handler.FileAccess)

	for _, path := range []string{
		"/api/file/images/sample.webp",
		handler.signFileURLAt("/api/file/images/sample.webp", time.Now().Add(-2*time.Minute)),
		strings.Replace(handler.signFileURL("/api/file/images/sample.webp"), "sample.webp", "other.webp", 1),
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("path %q status = %d, want 401, body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func extractEscapedDASHQuery(t *testing.T, body string, marker string) string {
	t.Helper()
	start := strings.Index(body, marker)
	if start < 0 {
		t.Fatalf("marker %q missing from manifest: %s", marker, body)
	}
	start += len(marker)
	end := strings.Index(body[start:], `"`)
	if end < 0 {
		t.Fatalf("signed query for marker %q is unterminated: %s", marker, body)
	}
	return strings.ReplaceAll(body[start:start+end], "&amp;", "&")
}

func TestNormalizeFileURLForStorageStripsLocalSignature(t *testing.T) {
	handler := NativeHandlers{Config: config.Config{
		Auth: config.AuthConfig{JWTSecret: "jwt-secret"},
		Upload: config.UploadConfig{
			FileSigning: config.UploadFileSigningConfig{Secret: "file-secret", TTL: time.Hour},
			LocalBase:   "https://xse.example.com",
		},
	}}
	signed := handler.signFileURL("https://xse.example.com/api/file/images/sample.webp")
	if !strings.Contains(signed, "pvimg_exp=") || !strings.Contains(signed, "sign=") {
		t.Fatalf("signed URL missing signature params: %s", signed)
	}
	if got := handler.normalizeFileURLForStorage(signed); got != "/api/file/images/sample.webp" {
		t.Fatalf("normalized = %q, want canonical path", got)
	}
	if got := handler.normalizeFileURLForStorage("https://cdn.example.com/image.webp?x=1"); got != "https://cdn.example.com/image.webp?x=1" {
		t.Fatalf("external URL changed: %q", got)
	}
}
