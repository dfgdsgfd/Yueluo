package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
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

func TestAdminFileRecycleBinInspectPreviewDownload(t *testing.T) {
	db := openAdminFileRecycleTestDB(t)
	root := t.TempDir()
	source := filepath.Join(root, "attachments", "note.txt")
	if err := writeAdminRecycleTestFile(source, "hello recycle"); err != nil {
		t.Fatalf("write source: %v", err)
	}
	cfg := adminFileRecycleTestConfig(root, filepath.Join(root, ".trash"))
	service := services.NewFileRecycleService(db, cfg, nil)
	summary, err := service.RecycleLocal(context.Background(), []services.FileRecycleInput{{
		ResourceType: "post",
		ResourceID:   9,
		Kind:         "attachment",
		Storage:      services.FileRecycleStorageLocal,
		OriginalURL:  "/api/file/attachments/note.txt",
		OriginalPath: source,
	}})
	if err != nil {
		t.Fatalf("RecycleLocal() error = %v", err)
	}
	id := summary.Results[0].ID
	handler := NativeHandlers{DB: db, Config: cfg, FileRecycle: service}

	inspect := performAdminFileRecycleRequest(handler, http.MethodGet, "/api/admin/file-recycle-bin/"+stringID(id)+"/inspect")
	if inspect.Code != http.StatusOK {
		t.Fatalf("inspect status = %d body=%s", inspect.Code, inspect.Body.String())
	}
	if body := inspect.Body.String(); !strings.Contains(body, `"previewable":true`) || !strings.Contains(body, `"downloadable":true`) {
		t.Fatalf("inspect body missing preview flags: %s", body)
	}

	preview := performAdminFileRecycleRequest(handler, http.MethodGet, "/api/admin/file-recycle-bin/"+stringID(id)+"/preview")
	if preview.Code != http.StatusOK {
		t.Fatalf("preview status = %d body=%s", preview.Code, preview.Body.String())
	}
	if got := preview.Body.String(); got != "hello recycle" {
		t.Fatalf("preview body = %q", got)
	}
	if disposition := preview.Header().Get("Content-Disposition"); !strings.HasPrefix(disposition, "inline") || !strings.Contains(disposition, "note.txt") {
		t.Fatalf("preview Content-Disposition = %q", disposition)
	}

	download := performAdminFileRecycleRequest(handler, http.MethodGet, "/api/admin/file-recycle-bin/"+stringID(id)+"/download")
	if download.Code != http.StatusOK {
		t.Fatalf("download status = %d body=%s", download.Code, download.Body.String())
	}
	if got := download.Body.String(); got != "hello recycle" {
		t.Fatalf("download body = %q", got)
	}
	if disposition := download.Header().Get("Content-Disposition"); !strings.HasPrefix(disposition, "attachment") || !strings.Contains(disposition, "note.txt") {
		t.Fatalf("download Content-Disposition = %q", disposition)
	}
}

func TestAdminFileRecycleBinDirectoryPreviewRejected(t *testing.T) {
	db := openAdminFileRecycleTestDB(t)
	root := t.TempDir()
	source := filepath.Join(root, "videos", "dash", "job")
	if err := writeAdminRecycleTestFile(filepath.Join(source, "manifest.mpd"), "<MPD></MPD>"); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	cfg := adminFileRecycleTestConfig(root, filepath.Join(root, ".trash"))
	service := services.NewFileRecycleService(db, cfg, nil)
	summary, err := service.RecycleLocal(context.Background(), []services.FileRecycleInput{{
		ResourceType: "post",
		ResourceID:   10,
		Kind:         "video_dash",
		OriginalPath: source,
		IsDir:        true,
	}})
	if err != nil {
		t.Fatalf("RecycleLocal() error = %v", err)
	}
	id := summary.Results[0].ID
	handler := NativeHandlers{DB: db, Config: cfg, FileRecycle: service}

	inspect := performAdminFileRecycleRequest(handler, http.MethodGet, "/api/admin/file-recycle-bin/"+stringID(id)+"/inspect")
	if inspect.Code != http.StatusOK {
		t.Fatalf("inspect status = %d body=%s", inspect.Code, inspect.Body.String())
	}
	if body := inspect.Body.String(); !strings.Contains(body, `"is_dir":true`) || !strings.Contains(body, "manifest.mpd") {
		t.Fatalf("inspect body missing directory details: %s", body)
	}

	preview := performAdminFileRecycleRequest(handler, http.MethodGet, "/api/admin/file-recycle-bin/"+stringID(id)+"/preview")
	if preview.Code != http.StatusBadRequest {
		t.Fatalf("preview status = %d body=%s", preview.Code, preview.Body.String())
	}
	if !strings.Contains(preview.Body.String(), "admin.fileRecycleBin.directoryPreviewUnsupported") {
		t.Fatalf("preview body missing i18n key: %s", preview.Body.String())
	}
}

func performAdminFileRecycleRequest(handler NativeHandlers, method string, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, path, nil)
	handler.AdminFileRecycleBin(context)
	return recorder
}

func openAdminFileRecycleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.FileRecycleItem{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func adminFileRecycleTestConfig(root string, trashDir string) config.Config {
	return config.Config{
		Upload: config.UploadConfig{
			RootDir: root,
			Recycle: config.UploadRecycleConfig{
				Enabled:         true,
				RootDir:         trashDir,
				Retention:       30 * 24 * time.Hour,
				CleanupInterval: time.Hour,
			},
			Image:      config.UploadImageConfig{LocalUploadDir: filepath.Join(root, "images")},
			Video:      config.UploadVideoConfig{LocalUploadDir: filepath.Join(root, "videos"), CoverDir: filepath.Join(root, "covers")},
			Attachment: config.UploadAttachmentConfig{LocalUploadDir: filepath.Join(root, "attachments")},
		},
	}
}

func writeAdminRecycleTestFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func stringID(id int64) string {
	return strconv.FormatInt(id, 10)
}
