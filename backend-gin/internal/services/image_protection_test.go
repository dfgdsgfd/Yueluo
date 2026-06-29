package services

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

func TestSelectProtectedPackageImagesSupportsPinnedFullArchiveSelection(t *testing.T) {
	images := []domain.PostImage{
		{ID: 1, IsFreePreview: false, IsProtected: true},
		{ID: 2, IsFreePreview: true, IsProtected: true},
		{ID: 3, IsFreePreview: false, IsProtected: true},
	}

	guest, err := selectProtectedPackageImages(images, false, []int64{2})
	if err != nil || len(guest) != 1 || guest[0].ID != 2 {
		t.Fatalf("guest selection = %#v, err = %v", guest, err)
	}
	purchased, err := selectProtectedPackageImages(images, true, []int64{2, 3})
	if err != nil || len(purchased) != 2 {
		t.Fatalf("purchased selection = %#v, err = %v", purchased, err)
	}
	fullArchive, err := selectProtectedPackageImages(images, true, []int64{1, 2, 3})
	if err != nil || len(fullArchive) != 3 || fullArchive[0].IsProtected {
		t.Fatalf("full archive selection = %#v, err = %v", fullArchive, err)
	}
	if _, err := selectProtectedPackageImages(images, false, []int64{2, 3}); err == nil {
		t.Fatal("locked paid image in pinned selection should be rejected")
	}
}

func TestImageProtectionSecondsPerImageUsesRecentCompletedJobs(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.ImageProtectionJob{}); err != nil {
		t.Fatal(err)
	}
	started := time.Now().Add(-30 * time.Second)
	finished := started.Add(30 * time.Second)
	if err := db.Create(&domain.ImageProtectionJob{
		JobID: "eta-sample", Status: imageProtectionStatusCompleted,
		ProtectedImageCount: 3, StartedAt: &started, FinishedAt: &finished,
	}).Error; err != nil {
		t.Fatal(err)
	}
	service := &QueueService{db: db}
	if got := service.imageProtectionSecondsPerImage(context.Background()); got < 9.5 || got > 10.5 {
		t.Fatalf("seconds per image = %.2f, want about 10", got)
	}
}

func TestImageProtectionAllowedFailures(t *testing.T) {
	tests := []struct {
		name    string
		total   int
		percent int
		want    int
	}{
		{name: "two_of_ten", total: 10, percent: 20, want: 2},
		{name: "floors_small_batches", total: 4, percent: 20, want: 0},
		{name: "keeps_one_required_success", total: 1, percent: 100, want: 0},
		{name: "all_but_one_on_full_tolerance", total: 10, percent: 100, want: 9},
		{name: "disabled", total: 10, percent: 0, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := imageProtectionAllowedFailures(tt.total, tt.percent); got != tt.want {
				t.Fatalf("imageProtectionAllowedFailures(%d, %d) = %d, want %d", tt.total, tt.percent, got, tt.want)
			}
		})
	}
}

func TestReportImageProtectionProgressIsMonotonic(t *testing.T) {
	ctx := context.Background()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.ImageProtectionJob{}); err != nil {
		t.Fatal(err)
	}
	job := domain.ImageProtectionJob{
		JobID:               "monotonic-progress-job",
		Status:              imageProtectionStatusProcessing,
		Progress:            60,
		ProtectedImageCount: 4,
		ProcessedImageCount: 2,
		CurrentStep:         "embedding",
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	service := &QueueService{db: db}
	if err := service.reportImageProtectionProgress(ctx, job.JobID, "reading_source", "", 0, 4, 5, time.Now(), 20); err != nil {
		t.Fatal(err)
	}
	var updated domain.ImageProtectionJob
	if err := db.Where("job_id = ?", job.JobID).First(&updated).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Progress < 60 {
		t.Fatalf("progress moved backward to %d", updated.Progress)
	}
	if updated.ProcessedImageCount < 2 {
		t.Fatalf("processed count moved backward to %d", updated.ProcessedImageCount)
	}
}

func TestProcessProtectedImageWithRetryUpscalesSmallImageAndVerifiesWatermark(t *testing.T) {
	var source bytes.Buffer
	if err := jpeg.Encode(&source, gradientImage(48, 64), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	const secret = "protection-retry-secret"
	const traceToken = "0102030405060708"
	settings := NewSettingsService(nil, nil)
	_ = settings.Set(context.Background(), "image_webp_enabled", false)
	processor := NewImageProcessor(settings, secret, 10<<20, nil)
	configureTestShortCodeResolver(processor, traceToken, testShortCode)
	processed, extracted, err := processProtectedImageWithRetry(context.Background(), processor, ProcessImageInput{
		Data:                  source.Bytes(),
		Filename:              "small.jpg",
		ContentType:           "image/jpeg",
		Purpose:               ImagePurposeContent,
		User:                  &RequestUser{ID: 42, UserID: "viewer-account"},
		ForceWatermark:        true,
		WatermarkTraceToken:   traceToken,
		WatermarkPayloadToken: testShortCode,
	}, traceToken)
	if err != nil {
		t.Fatalf("process protected image: %v", err)
	}
	if !processed.WatermarkApplied || !extracted.Valid || processed.Format != "webp" || processed.Width <= 48 || processed.Height <= 64 {
		t.Fatalf("unexpected protected image result: processed=%+v extracted=%+v", processed, extracted)
	}
}

func TestImageProtectionErrorCodeKeepsRemoteFailuresSpecific(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "unavailable",
			err:  fmt.Errorf("embedding image 2: %w: connection refused", ErrHiddenWatermarkRemoteUnavailable),
			want: "watermark_server_unavailable",
		},
		{
			name: "timeout",
			err:  fmt.Errorf("embedding image 2: %w: context deadline exceeded", ErrHiddenWatermarkRemoteTimeout),
			want: "watermark_server_timeout",
		},
		{
			name: "rejected",
			err:  fmt.Errorf("embedding image 2: %w: upstream status 422", ErrHiddenWatermarkRemoteRejected),
			want: "watermark_server_rejected",
		},
		{
			name: "wrapped stage unavailable",
			err: imageProtectionStageError{
				stage: "embedding", imageIndex: 2, err: ErrHiddenWatermarkRemoteUnavailable,
			},
			want: "watermark_server_unavailable",
		},
		{
			name: "watermark verification",
			err:  errors.New("image watermark verification failed after 3 attempts"),
			want: "watermark_verification_failed",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := imageProtectionErrorCode(test.err); got != test.want {
				t.Fatalf("imageProtectionErrorCode(%v) = %q, want %q", test.err, got, test.want)
			}
		})
	}
}

func TestRunImageProtectionPackageRequiresRemoteWatermarkServer(t *testing.T) {
	ctx := context.Background()
	tempRoot := t.TempDir()
	imageDir := filepath.Join(tempRoot, "uploads", "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatal(err)
	}
	var source bytes.Buffer
	if err := jpeg.Encode(&source, gradientImage(512, 512), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	imagePath := filepath.Join(imageDir, "protected-source.jpg")
	if err := os.WriteFile(imagePath, source.Bytes(), 0644); err != nil {
		t.Fatalf("write source image: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Post{},
		&domain.PostImage{},
		&domain.PostPaymentSetting{},
		&domain.UserPurchasedContent{},
		&domain.ImageProtectionJob{},
		&domain.ImageWatermarkTrace{},
		&domain.SystemSetting{},
	); err != nil {
		t.Fatal(err)
	}
	author := domain.User{UserID: "author"}
	viewer := domain.User{UserID: "viewer", Nickname: "Viewer"}
	if err := db.Create(&author).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	post := domain.Post{UserID: author.ID, Title: "protected", Visibility: "public", Type: 1}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	cover := domain.PostImage{PostID: post.ID, ImageURL: "/api/file/images/cover.jpg", SortOrder: 1, IsProtected: true}
	protected := domain.PostImage{PostID: post.ID, ImageURL: "/api/file/images/protected-source.jpg", SortOrder: 2, IsProtected: true}
	if err := db.Create(&cover).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&protected).Error; err != nil {
		t.Fatal(err)
	}
	job := domain.ImageProtectionJob{
		JobID:               "remote-required-job",
		PostID:              post.ID,
		UserID:              viewer.ID,
		AuthorID:            author.ID,
		Status:              imageProtectionStatusQueued,
		ProtectedImageCount: 1,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}

	settings := NewSettingsService(db, nil)
	settings.Load(ctx)
	service := &QueueService{
		cfg: config.Config{Upload: config.UploadConfig{
			FileSigning: config.UploadFileSigningConfig{Secret: "remote-required-secret"},
			Image:       config.UploadImageConfig{LocalUploadDir: imageDir, MaxSizeBytes: 10 << 20},
			Temp:        config.UploadTempConfig{RootDir: filepath.Join(tempRoot, "tmp")},
		}},
		db:       db,
		settings: settings,
	}
	err = service.runImageProtectionPackage(ctx, job.JobID, []int64{protected.ID})
	if !errors.Is(err, ErrHiddenWatermarkRemoteUnavailable) {
		t.Fatalf("runImageProtectionPackage() error = %v, want ErrHiddenWatermarkRemoteUnavailable", err)
	}
	if got := imageProtectionErrorCode(err); got != "watermark_server_unavailable" {
		t.Fatalf("imageProtectionErrorCode() = %q, want watermark_server_unavailable", got)
	}
}

func TestRunImageProtectionPackageDoesNotIncludeJSONManifest(t *testing.T) {
	ctx := context.Background()
	tempRoot := t.TempDir()
	imageDir := filepath.Join(tempRoot, "uploads", "images")
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		t.Fatal(err)
	}
	var source bytes.Buffer
	if err := jpeg.Encode(&source, gradientImage(512, 512), &jpeg.Options{Quality: 95}); err != nil {
		t.Fatalf("encode fixture: %v", err)
	}
	imagePath := filepath.Join(imageDir, "protected-source.jpg")
	if err := os.WriteFile(imagePath, source.Bytes(), 0644); err != nil {
		t.Fatalf("write source image: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Post{},
		&domain.PostImage{},
		&domain.PostPaymentSetting{},
		&domain.UserPurchasedContent{},
		&domain.ImageProtectionJob{},
		&domain.ImageWatermarkTrace{},
		&domain.Notification{},
		&domain.SystemSetting{},
	); err != nil {
		t.Fatal(err)
	}
	author := domain.User{UserID: "author"}
	viewer := domain.User{UserID: "viewer", Nickname: "Viewer"}
	if err := db.Create(&author).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	post := domain.Post{UserID: author.ID, Title: "protected", Visibility: "public", Type: 1}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	cover := domain.PostImage{PostID: post.ID, ImageURL: "/api/file/images/cover.jpg", SortOrder: 1, IsProtected: true}
	protected := domain.PostImage{PostID: post.ID, ImageURL: "/api/file/images/protected-source.jpg", SortOrder: 2, IsProtected: true}
	if err := db.Create(&cover).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&protected).Error; err != nil {
		t.Fatal(err)
	}
	job := domain.ImageProtectionJob{
		JobID:               "0123456789abcdef0123456789abcdef",
		PostID:              post.ID,
		UserID:              viewer.ID,
		AuthorID:            author.ID,
		Status:              imageProtectionStatusQueued,
		ProtectedImageCount: 1,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}

	settings := NewSettingsService(db, nil)
	settings.Load(ctx)
	remote := newImageProtectionWatermarkTestServer(t)
	defer remote.Close()
	service := &QueueService{
		cfg: config.Config{
			Upload: config.UploadConfig{
				FileSigning: config.UploadFileSigningConfig{Secret: "manifest-secret"},
				Image:       config.UploadImageConfig{LocalUploadDir: imageDir, MaxSizeBytes: 10 << 20},
				Temp:        config.UploadTempConfig{RootDir: filepath.Join(tempRoot, "tmp")},
			},
			WebP: config.WebPConfig{HiddenWatermark: config.HiddenWatermarkConfig{
				Remote: config.HiddenWatermarkRemoteConfig{URL: remote.URL},
			}},
		},
		db:       db,
		settings: settings,
	}
	if err := service.runImageProtectionPackage(ctx, job.JobID, []int64{protected.ID}); err != nil {
		t.Fatalf("runImageProtectionPackage: %v", err)
	}
	var completed domain.ImageProtectionJob
	if err := db.Where("job_id = ?", job.JobID).First(&completed).Error; err != nil {
		t.Fatal(err)
	}
	reader, err := zip.OpenReader(completed.PackagePath)
	if err != nil {
		t.Fatalf("open package: %v", err)
	}
	defer reader.Close()
	imageEntries := 0
	for _, file := range reader.File {
		if filepath.Ext(file.Name) == ".json" {
			t.Fatalf("protected package must not include JSON files, found %s", file.Name)
		}
		imageEntries++
	}
	if imageEntries != 1 {
		t.Fatalf("zip image entries = %d, want 1", imageEntries)
	}
	var trace domain.ImageWatermarkTrace
	if err := db.Where("job_id = ? AND image_id = ?", job.JobID, protected.ID).First(&trace).Error; err != nil {
		t.Fatalf("protected watermark trace missing: %v", err)
	}
	if trace.PayloadBytes != domain.ImageWatermarkPayloadBytes || trace.Token == "" {
		t.Fatalf("protected watermark trace = %+v", trace)
	}
	expired := time.Now().Add(-time.Minute)
	if err := db.Model(&completed).Updates(map[string]any{"expires_at": expired}).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.cleanupExpiredImageProtectionPackages(ctx); err != nil {
		t.Fatal(err)
	}
	if err := db.Where("id = ?", trace.ID).First(&domain.ImageWatermarkTrace{}).Error; err != nil {
		t.Fatalf("trace must survive package expiry: %v", err)
	}
}

func newImageProtectionWatermarkTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	var embeddedPayload []byte
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/watermark/embed":
			raw, err := base64.StdEncoding.DecodeString(r.FormValue("payload_b64"))
			if err != nil {
				t.Fatal(err)
			}
			embeddedPayload = append([]byte(nil), raw...)
			var out bytes.Buffer
			if err := png.Encode(&out, gradientImage(512, 512)); err != nil {
				t.Fatal(err)
			}
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(out.Bytes())
		case "/v1/watermark/extract":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"payload_b64":            base64.StdEncoding.EncodeToString(embeddedPayload),
				"payload_candidates_b64": []string{base64.StdEncoding.EncodeToString(embeddedPayload)},
				"payload_bytes":          len(embeddedPayload),
			})
		default:
			http.NotFound(w, r)
		}
	}))
}
