package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

func TestUploadAssetCleanupDeletesExpiredLocalTempImage(t *testing.T) {
	db := newUploadAssetTestDB(t)
	imageDir := filepath.Join(t.TempDir(), "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(imageDir, "sample.webp")
	if err := os.WriteFile(filePath, []byte("image"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := uploadAssetTestConfig(imageDir)
	service := NewUploadAssetService(db, cfg, nil)
	asset, err := service.Record(context.Background(), UploadAssetRecordInput{
		UserID:       7,
		Purpose:      string(ImagePurposeAIAnalysis),
		Kind:         "image",
		URL:          "/api/file/images/sample.webp",
		Storage:      "local",
		LocalPath:    filePath,
		OriginalName: "sample.jpg",
		Size:         5,
		MimeType:     "image/webp",
	})
	if err != nil {
		t.Fatalf("Record() error = %v", err)
	}
	past := time.Now().Add(-time.Minute)
	if err := db.Model(&domain.UploadAsset{}).Where("id = ?", asset.ID).Update("expires_at", &past).Error; err != nil {
		t.Fatal(err)
	}

	if err := service.CleanupExpired(context.Background(), 10); err != nil {
		t.Fatalf("CleanupExpired() error = %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("expired file still exists or stat failed unexpectedly: %v", err)
	}
	var saved domain.UploadAsset
	if err := db.First(&saved, asset.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != UploadAssetStatusDeleted || saved.DeletedAt == nil {
		t.Fatalf("asset status = %q deletedAt=%v, want deleted with timestamp", saved.Status, saved.DeletedAt)
	}
}

func TestUploadAssetCleanupKeepsReferencedImageAndMarksBound(t *testing.T) {
	db := newUploadAssetTestDB(t)
	imageDir := filepath.Join(t.TempDir(), "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatal(err)
	}
	filePath := filepath.Join(imageDir, "kept.webp")
	if err := os.WriteFile(filePath, []byte("image"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := uploadAssetTestConfig(imageDir)
	service := NewUploadAssetService(db, cfg, nil)
	past := time.Now().Add(-time.Minute)
	asset := domain.UploadAsset{
		UserID:    7,
		Purpose:   string(ImagePurposeAIAnalysis),
		Kind:      "image",
		URL:       "/api/file/images/kept.webp",
		Storage:   "local",
		LocalPath: filePath,
		Status:    UploadAssetStatusTemp,
		ExpiresAt: &past,
		CreatedAt: past,
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.PostImage{PostID: 88, ImageURL: asset.URL, IsFreePreview: true, SortOrder: 1}).Error; err != nil {
		t.Fatal(err)
	}

	if err := service.CleanupExpired(context.Background(), 10); err != nil {
		t.Fatalf("CleanupExpired() error = %v", err)
	}
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("referenced file should remain: %v", err)
	}
	var saved domain.UploadAsset
	if err := db.First(&saved, asset.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != UploadAssetStatusBound || saved.BoundPostID == nil || *saved.BoundPostID != 88 {
		t.Fatalf("asset binding = status %q post %v, want bound to post 88", saved.Status, saved.BoundPostID)
	}
}

func TestTouchAIAnalysisExtendsTemporaryUploadAsset(t *testing.T) {
	db := newUploadAssetTestDB(t)
	cfg := uploadAssetTestConfig(t.TempDir())
	service := NewUploadAssetService(db, cfg, nil)
	past := time.Now().Add(-time.Minute)
	asset := domain.UploadAsset{
		UserID:    7,
		Purpose:   string(ImagePurposeContent),
		Kind:      "image",
		URL:       "/api/file/images/touched.webp",
		Storage:   "local",
		Status:    UploadAssetStatusTemp,
		ExpiresAt: &past,
		CreatedAt: past,
	}
	if err := db.Create(&asset).Error; err != nil {
		t.Fatal(err)
	}

	if err := service.TouchAIAnalysis(context.Background(), 7, []string{asset.URL}); err != nil {
		t.Fatalf("TouchAIAnalysis() error = %v", err)
	}
	var saved domain.UploadAsset
	if err := db.First(&saved, asset.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Purpose != string(ImagePurposeAIAnalysis) || saved.ExpiresAt == nil || !saved.ExpiresAt.After(time.Now()) {
		t.Fatalf("asset after touch = purpose %q expires %v, want ai purpose and future expiry", saved.Purpose, saved.ExpiresAt)
	}
}

func TestUploadAssetRecordUpsertQualifiesPostgresStatusColumns(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1700000000, 0).UTC()
	expiresAt := now.Add(time.Hour)
	row := domain.UploadAsset{
		UserID:     7,
		Purpose:    string(ImagePurposeAIAnalysis),
		Kind:       "image",
		URL:        "/api/file/images/sample.webp",
		Storage:    "local",
		Status:     UploadAssetStatusTemp,
		ExpiresAt:  &expiresAt,
		LastUsedAt: &now,
		CreatedAt:  now,
		UpdatedAt:  &now,
	}
	tx := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "url"}},
		DoUpdates: clause.Assignments(uploadAssetRecordConflictAssignments(row)),
	}).Create(&row)
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	sql := tx.Statement.SQL.String()
	if strings.Contains(sql, "CASE WHEN status") {
		t.Fatalf("upsert SQL has ambiguous status reference: %s", sql)
	}
	for _, want := range []string{`"upload_assets"."status"`, `"upload_assets"."expires_at"`} {
		if !strings.Contains(sql, want) {
			t.Fatalf("upsert SQL missing %s qualification: %s", want, sql)
		}
	}
}

func newUploadAssetTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.UploadAsset{}, &domain.PostImage{}, &domain.PostVideo{}, &domain.PostAttachment{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func uploadAssetTestConfig(imageDir string) config.Config {
	root := filepath.Dir(imageDir)
	return config.Config{
		Upload: config.UploadConfig{
			RootDir: root,
			Temp: config.UploadTempConfig{
				Retention:       time.Hour,
				CleanupInterval: time.Hour,
			},
			Image: config.UploadImageConfig{
				LocalUploadDir: imageDir,
				Strategy:       "local",
			},
		},
	}
}
