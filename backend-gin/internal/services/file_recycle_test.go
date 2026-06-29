package services

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

func TestFileRecycleMovesLocalFileAndRecordsMissing(t *testing.T) {
	db := openFileRecycleTestDB(t)
	root := t.TempDir()
	imageDir := filepath.Join(root, "images")
	trashDir := filepath.Join(root, ".trash")
	imagePath := filepath.Join(imageDir, "2026", "06", "cat.jpg")
	if err := writeTestFile(imagePath, "image-bytes"); err != nil {
		t.Fatalf("write image: %v", err)
	}
	missingPath := filepath.Join(imageDir, "missing.jpg")
	postID := int64(42)
	userID := int64(1497)
	service := NewFileRecycleService(db, fileRecycleTestConfig(root, trashDir), nil)

	summary, err := service.RecycleLocal(context.Background(), []FileRecycleInput{
		{
			ResourceType: "post",
			ResourceID:   postID,
			PostID:       &postID,
			UserID:       &userID,
			Kind:         "image",
			Storage:      FileRecycleStorageLocal,
			OriginalURL:  "/api/file/images/2026/06/cat.jpg",
			OriginalPath: imagePath,
		},
		{
			ResourceType: "post",
			ResourceID:   postID,
			PostID:       &postID,
			UserID:       &userID,
			Kind:         "image",
			Storage:      FileRecycleStorageLocal,
			OriginalURL:  "/api/file/images/missing.jpg",
			OriginalPath: missingPath,
		},
	})
	if err != nil {
		t.Fatalf("RecycleLocal() error = %v", err)
	}
	if summary.Recycled != 1 || summary.Missing != 1 || summary.Failed != 0 {
		t.Fatalf("summary = %+v, want one recycled and one missing", summary)
	}
	if _, err := os.Stat(imagePath); !os.IsNotExist(err) {
		t.Fatalf("source file should be moved, stat error = %v", err)
	}
	var items []domain.FileRecycleItem
	if err := db.Order("status ASC").Find(&items).Error; err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("rows = %d, want 2", len(items))
	}
	var recycled domain.FileRecycleItem
	if err := db.Where("status = ?", FileRecycleStatusRecycled).Take(&recycled).Error; err != nil {
		t.Fatal(err)
	}
	if recycled.RecycledPath == "" || !strings.Contains(filepath.ToSlash(recycled.RecycledPath), "/.trash/post/") {
		t.Fatalf("unexpected recycled path: %q", recycled.RecycledPath)
	}
	if _, err := os.Stat(recycled.RecycledPath); err != nil {
		t.Fatalf("recycled file missing: %v", err)
	}
	if recycled.PurgeAfter.Sub(recycled.DeletedAt) != 30*24*time.Hour {
		t.Fatalf("retention = %s, want 720h", recycled.PurgeAfter.Sub(recycled.DeletedAt))
	}
	manifestMatches, err := filepath.Glob(filepath.Join(trashDir, "post", "*", "*", "*", "post-42", "*", "manifest.json"))
	if err != nil || len(manifestMatches) != 1 {
		t.Fatalf("manifest matches = %v err=%v, want one manifest", manifestMatches, err)
	}
}

func TestFileRecycleUsesSettingsRetentionDays(t *testing.T) {
	db := openFileRecycleTestDB(t)
	root := t.TempDir()
	imageDir := filepath.Join(root, "images")
	imagePath := filepath.Join(imageDir, "cat.jpg")
	if err := writeTestFile(imagePath, "image-bytes"); err != nil {
		t.Fatalf("write image: %v", err)
	}
	settings := NewSettingsService(nil, nil)
	if !settings.Set(context.Background(), FileRecycleRetentionDaysKey, 3) {
		t.Fatal("set file recycle retention days")
	}
	service := NewFileRecycleServiceWithSettings(db, fileRecycleTestConfig(root, filepath.Join(root, ".trash")), nil, settings)

	summary, err := service.RecycleLocal(context.Background(), []FileRecycleInput{{
		ResourceType: "post",
		ResourceID:   1,
		Kind:         "image",
		Storage:      FileRecycleStorageLocal,
		OriginalURL:  "/api/file/images/cat.jpg",
		OriginalPath: imagePath,
	}})
	if err != nil {
		t.Fatalf("RecycleLocal() error = %v", err)
	}
	if summary.Recycled != 1 {
		t.Fatalf("summary = %+v, want one recycled", summary)
	}
	var item domain.FileRecycleItem
	if err := db.Take(&item).Error; err != nil {
		t.Fatal(err)
	}
	if item.PurgeAfter.Sub(item.DeletedAt) != 3*24*time.Hour {
		t.Fatalf("retention = %s, want 72h", item.PurgeAfter.Sub(item.DeletedAt))
	}
}

func TestFileRecycleSkipsUnsafePath(t *testing.T) {
	db := openFileRecycleTestDB(t)
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.jpg")
	if err := writeTestFile(outside, "outside"); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	service := NewFileRecycleService(db, fileRecycleTestConfig(root, filepath.Join(root, ".trash")), nil)

	summary, err := service.RecycleLocal(context.Background(), []FileRecycleInput{{ResourceType: "post", ResourceID: 1, Kind: "image", OriginalPath: outside}})
	if err != nil {
		t.Fatalf("RecycleLocal() error = %v", err)
	}
	if summary.Skipped != 1 || summary.Recycled != 0 {
		t.Fatalf("summary = %+v, want skipped", summary)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside file should remain: %v", err)
	}
	var item domain.FileRecycleItem
	if err := db.Take(&item).Error; err != nil {
		t.Fatal(err)
	}
	if item.Status != FileRecycleStatusSkipped || item.Error != "unsafe_upload_path" {
		t.Fatalf("item = %+v, want unsafe skipped", item)
	}
}

func TestFileRecycleMovesDASHDirectoryWithSegments(t *testing.T) {
	db := openFileRecycleTestDB(t)
	root := t.TempDir()
	videoDir := filepath.Join(root, "videos")
	dashDir := filepath.Join(videoDir, "dash", "2026-06-27", "1497", "1778703092050")
	for name, content := range map[string]string{
		"manifest.mpd":      "<MPD></MPD>",
		"chunk-stream0.m4s": "segment",
		"init-stream0.m4s":  "init",
	} {
		if err := writeTestFile(filepath.Join(dashDir, name), content); err != nil {
			t.Fatalf("write dash %s: %v", name, err)
		}
	}
	postID := int64(42)
	userID := int64(1497)
	service := NewFileRecycleService(db, fileRecycleTestConfig(root, filepath.Join(root, ".trash")), nil)

	summary, err := service.RecycleLocal(context.Background(), []FileRecycleInput{{
		ResourceType: "post",
		ResourceID:   postID,
		PostID:       &postID,
		UserID:       &userID,
		Kind:         "video_dash",
		Storage:      FileRecycleStorageLocal,
		OriginalURL:  "/api/file/videos/dash/2026-06-27/1497/1778703092050/manifest.mpd",
		OriginalPath: dashDir,
		IsDir:        true,
	}})
	if err != nil {
		t.Fatalf("RecycleLocal() error = %v", err)
	}
	if summary.Recycled != 1 || summary.Results[0].FileCount != 3 {
		t.Fatalf("summary = %+v, want dash dir with 3 files", summary)
	}
	if _, err := os.Stat(dashDir); !os.IsNotExist(err) {
		t.Fatalf("dash dir should be moved, stat error = %v", err)
	}
	recycledDir := summary.Results[0].RecycledPath
	for _, name := range []string{"manifest.mpd", "chunk-stream0.m4s", "init-stream0.m4s"} {
		if _, err := os.Stat(filepath.Join(recycledDir, name)); err != nil {
			t.Fatalf("recycled dash file %s missing: %v", name, err)
		}
	}
}

func TestFileRecycleInspectAndOpenRecycledFile(t *testing.T) {
	db := openFileRecycleTestDB(t)
	root := t.TempDir()
	textPath := filepath.Join(root, "attachments", "note.txt")
	if err := writeTestFile(textPath, "hello recycle"); err != nil {
		t.Fatalf("write text: %v", err)
	}
	service := NewFileRecycleService(db, fileRecycleTestConfig(root, filepath.Join(root, ".trash")), nil)

	summary, err := service.RecycleLocal(context.Background(), []FileRecycleInput{{
		ResourceType: "post",
		ResourceID:   7,
		Kind:         "attachment",
		Storage:      FileRecycleStorageLocal,
		OriginalURL:  "/api/file/attachments/note.txt",
		OriginalPath: textPath,
	}})
	if err != nil {
		t.Fatalf("RecycleLocal() error = %v", err)
	}
	id := summary.Results[0].ID
	inspection, err := service.InspectItem(context.Background(), id)
	if err != nil {
		t.Fatalf("InspectItem() error = %v", err)
	}
	if inspection.Original.Exists || !inspection.Recycled.Exists || inspection.Recycled.Unsafe || !inspection.Downloadable || !inspection.Previewable {
		t.Fatalf("inspection = %+v, want safe previewable recycled file and missing original", inspection)
	}
	if inspection.PreviewKind != "text" {
		t.Fatalf("preview kind = %q, want text", inspection.PreviewKind)
	}
	opened, err := service.OpenRecycledFile(context.Background(), id)
	if err != nil {
		t.Fatalf("OpenRecycledFile() error = %v", err)
	}
	defer opened.File.Close()
	content, err := io.ReadAll(opened.File)
	if err != nil {
		t.Fatalf("read opened file: %v", err)
	}
	if string(content) != "hello recycle" {
		t.Fatalf("opened content = %q", content)
	}
}

func TestFileRecycleInspectDirectoryAndRejectOpen(t *testing.T) {
	db := openFileRecycleTestDB(t)
	root := t.TempDir()
	dashDir := filepath.Join(root, "videos", "dash", "job")
	if err := writeTestFile(filepath.Join(dashDir, "manifest.mpd"), "<MPD></MPD>"); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	service := NewFileRecycleService(db, fileRecycleTestConfig(root, filepath.Join(root, ".trash")), nil)
	summary, err := service.RecycleLocal(context.Background(), []FileRecycleInput{{
		ResourceType: "post",
		ResourceID:   8,
		Kind:         "video_dash",
		OriginalPath: dashDir,
		IsDir:        true,
	}})
	if err != nil {
		t.Fatalf("RecycleLocal() error = %v", err)
	}
	inspection, err := service.InspectItem(context.Background(), summary.Results[0].ID)
	if err != nil {
		t.Fatalf("InspectItem() error = %v", err)
	}
	if !inspection.Recycled.Exists || !inspection.Recycled.IsDir || len(inspection.Files) != 1 || inspection.Previewable || inspection.Downloadable {
		t.Fatalf("inspection = %+v, want directory listing without preview/download", inspection)
	}
	if _, err := service.OpenRecycledFile(context.Background(), summary.Results[0].ID); !errors.Is(err, ErrFileRecycleDirectory) {
		t.Fatalf("OpenRecycledFile() error = %v, want directory error", err)
	}
}

func TestFileRecycleRejectsUnsafeRecycledPathOnInspectAndPurge(t *testing.T) {
	db := openFileRecycleTestDB(t)
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "victim.txt")
	if err := writeTestFile(outside, "keep"); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	service := NewFileRecycleService(db, fileRecycleTestConfig(root, filepath.Join(root, ".trash")), nil)
	item := domain.FileRecycleItem{
		GroupID:      "unsafe",
		ResourceType: "post",
		ResourceID:   1,
		Kind:         "image",
		Storage:      FileRecycleStorageLocal,
		RecycledPath: outside,
		Status:       FileRecycleStatusRecycled,
		DeletedAt:    time.Now(),
		PurgeAfter:   time.Now().Add(time.Hour),
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	inspection, err := service.InspectItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("InspectItem() error = %v", err)
	}
	if !inspection.Recycled.Unsafe || inspection.Downloadable {
		t.Fatalf("inspection = %+v, want unsafe recycled path", inspection)
	}
	summary, err := service.PurgeIDs(context.Background(), []int64{item.ID})
	if err != nil {
		t.Fatalf("PurgeIDs() error = %v", err)
	}
	if summary.Failed != 1 || summary.Purged != 0 {
		t.Fatalf("summary = %+v, want purge failure", summary)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside file should remain: %v", err)
	}
}

func TestFileRecycleRejectsSymlinkComponentOnInspectAndPurge(t *testing.T) {
	db := openFileRecycleTestDB(t)
	root := t.TempDir()
	trashDir := filepath.Join(root, ".trash")
	outsideDir := t.TempDir()
	outside := filepath.Join(outsideDir, "victim.txt")
	if err := writeTestFile(outside, "keep"); err != nil {
		t.Fatalf("write outside: %v", err)
	}
	if err := os.MkdirAll(trashDir, 0755); err != nil {
		t.Fatalf("mkdir trash: %v", err)
	}
	linkDir := filepath.Join(trashDir, "linked")
	if err := os.Symlink(outsideDir, linkDir); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	service := NewFileRecycleService(db, fileRecycleTestConfig(root, trashDir), nil)
	item := domain.FileRecycleItem{
		GroupID:      "unsafe-link",
		ResourceType: "post",
		ResourceID:   1,
		Kind:         "image",
		Storage:      FileRecycleStorageLocal,
		RecycledPath: filepath.Join(linkDir, "victim.txt"),
		Status:       FileRecycleStatusRecycled,
		DeletedAt:    time.Now(),
		PurgeAfter:   time.Now().Add(time.Hour),
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	inspection, err := service.InspectItem(context.Background(), item.ID)
	if err != nil {
		t.Fatalf("InspectItem() error = %v", err)
	}
	if !inspection.Recycled.Unsafe || inspection.Recycled.Exists || inspection.Downloadable {
		t.Fatalf("inspection = %+v, want unsafe symlink path", inspection)
	}
	summary, err := service.PurgeIDs(context.Background(), []int64{item.ID})
	if err != nil {
		t.Fatalf("PurgeIDs() error = %v", err)
	}
	if summary.Failed != 1 || summary.Purged != 0 {
		t.Fatalf("summary = %+v, want purge failure", summary)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("outside file should remain: %v", err)
	}
}

func openFileRecycleTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.FileRecycleItem{}, &domain.Post{}, &domain.PostVideo{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func fileRecycleTestConfig(root, trashDir string) config.Config {
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
