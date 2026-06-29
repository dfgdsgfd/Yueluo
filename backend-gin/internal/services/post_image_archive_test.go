package services

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

func TestRunPostImageArchiveWritesEverySelectedImage(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.Post{}, &domain.PostImage{}, &domain.ImageProtectionJob{}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	first := filepath.Join(root, "first.png")
	second := filepath.Join(root, "second.jpg")
	if err := os.WriteFile(first, []byte("first-image"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("second-image"), 0600); err != nil {
		t.Fatal(err)
	}
	post := domain.Post{UserID: 7, Title: "archive", Visibility: "public"}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	images := []domain.PostImage{
		{PostID: post.ID, ImageURL: first, SortOrder: 0},
		{PostID: post.ID, ImageURL: second, SortOrder: 1},
	}
	if err := db.Create(&images).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	job := domain.ImageProtectionJob{
		JobID:               "post-archive-job",
		PostID:              post.ID,
		AuthorID:            post.UserID,
		PackageKind:         "post_archive",
		SourceSignature:     postArchiveImageSignature(images),
		Status:              imageProtectionStatusQueued,
		ProtectedImageCount: len(images),
		CreatedAt:           now,
		UpdatedAt:           &now,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	service := &QueueService{
		db: db,
		cfg: config.Config{Upload: config.UploadConfig{
			RootDir:   root,
			LocalBase: "http://localhost:3001",
		}},
	}
	if err := service.runPostImageArchive(context.Background(), job.JobID, []int64{images[0].ID, images[1].ID}); err != nil {
		t.Fatal(err)
	}
	var completed domain.ImageProtectionJob
	if err := db.Where("job_id = ?", job.JobID).First(&completed).Error; err != nil {
		t.Fatal(err)
	}
	if completed.Status != imageProtectionStatusCompleted || completed.Progress != 100 || completed.ProcessedImageCount != 2 {
		t.Fatalf("completed job = %#v", completed)
	}
	reader, err := zip.OpenReader(completed.PackagePath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	if len(reader.File) != 2 {
		t.Fatalf("archive entries = %d, want 2", len(reader.File))
	}
}
