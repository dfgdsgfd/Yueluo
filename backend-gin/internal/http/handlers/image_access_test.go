package handlers

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

func TestImageAccessForViewerFourCombinations(t *testing.T) {
	images := []domain.PostImage{
		{ID: 1, ImageURL: "free-direct", IsFreePreview: true, SortOrder: 1},
		{ID: 2, ImageURL: "paid-direct", IsFreePreview: false, SortOrder: 2},
		{ID: 3, ImageURL: "free-protected", IsFreePreview: true, IsProtected: true, SortOrder: 3},
		{ID: 4, ImageURL: "paid-protected", IsFreePreview: false, IsProtected: true, SortOrder: 4},
	}
	payment := &domain.PostPaymentSetting{Enabled: true, PaymentMethod: "points", Price: 12}

	guest := imageAccessForViewer(images, payment, false, false)
	if got := imageURLs(guest.DirectImages); len(got) != 1 || got[0] != "free-direct" {
		t.Fatalf("guest direct images = %#v", got)
	}
	if guest.ProtectedPackageImageCount != 1 || guest.LockedProtectedImagesCount != 1 || guest.HiddenPaidImagesCount != 2 {
		t.Fatalf("guest access summary = %#v", guest)
	}

	purchased := imageAccessForViewer(images, payment, true, false)
	if got := imageURLs(purchased.DirectImages); len(got) != 2 || got[0] != "free-direct" || got[1] != "paid-direct" {
		t.Fatalf("purchased direct images = %#v", got)
	}
	if purchased.ProtectedPackageImageCount != 2 || purchased.HiddenPaidImagesCount != 0 {
		t.Fatalf("purchased access summary = %#v", purchased)
	}

	author := imageAccessForViewer(images, payment, true, true)
	if got := imageURLs(author.DirectImages); len(got) != 4 {
		t.Fatalf("author must receive every image for editing: %#v", got)
	}
	if author.ProtectedPackageImageCount != 2 {
		t.Fatalf("author protected package images = %d", author.ProtectedPackageImageCount)
	}
}

func TestRejectPostContentOverConfiguredLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(t.Context(), "post_content_max_length", 4) {
		t.Fatal("set post content max length")
	}
	handler := NativeHandlers{Settings: settings}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/posts", nil)

	if !handler.rejectPostContentOverLimit(context, "你好世界五") {
		t.Fatal("expected content over the configured limit to be rejected")
	}
	if recorder.Code != http.StatusBadRequest || !bytes.Contains(recorder.Body.Bytes(), []byte("error.post_content_limit")) {
		t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestImageAccessForViewerForcesLegacyCoverToFreeDirect(t *testing.T) {
	images := []domain.PostImage{
		{ID: 1, ImageURL: "legacy-cover", IsFreePreview: false, IsProtected: true, SortOrder: 1},
		{ID: 2, ImageURL: "paid-protected", IsFreePreview: false, IsProtected: true, SortOrder: 2},
	}
	payment := &domain.PostPaymentSetting{Enabled: true, PaymentMethod: "balance", Price: 8}

	guest := imageAccessForViewer(images, payment, false, false)
	if got := imageURLs(guest.DirectImages); len(got) != 1 || got[0] != "legacy-cover" {
		t.Fatalf("legacy guest direct images = %#v", got)
	}
	if guest.ProtectedImagesCount != 1 || guest.LockedProtectedImagesCount != 1 || guest.ProtectedPackageImageCount != 0 {
		t.Fatalf("legacy guest access summary = %#v", guest)
	}

	purchased := imageAccessForViewer(images, payment, true, false)
	if purchased.ProtectedPackageImageCount != 1 || len(purchased.ProtectedPackageImages) != 1 || purchased.ProtectedPackageImages[0].ID != 2 {
		t.Fatalf("legacy purchased access summary = %#v", purchased)
	}
}

func TestProtectedPackageCurrentImageIndex(t *testing.T) {
	job := domain.ImageProtectionJob{
		Status:              protectedPackageProcessing,
		ProtectedImageCount: 4,
		ProcessedImageCount: 0,
		CurrentStep:         "embedding",
	}
	if got := protectedPackageCurrentImageIndex(job); got != 1 {
		t.Fatalf("current image index = %d, want 1", got)
	}
	job.ProcessedImageCount = 2
	job.CurrentStep = "image_completed"
	if got := protectedPackageCurrentImageIndex(job); got != 2 {
		t.Fatalf("completed image index = %d, want 2", got)
	}
	job.CurrentStep = "writing_archive"
	if got := protectedPackageCurrentImageIndex(job); got != 3 {
		t.Fatalf("active image index = %d, want 3", got)
	}
}

func TestImageInputCountAndPaymentValidation(t *testing.T) {
	values := make([]any, defaultMaxPostImages+1)
	if got := imageInputCount(values); got != defaultMaxPostImages+1 {
		t.Fatalf("imageInputCount() = %d", got)
	}
	if validImagePaymentSettings(nil) {
		t.Fatal("nil payment settings should be invalid")
	}
	if validImagePaymentSettings(&repositories.PaymentSettingsInput{Enabled: true, PaymentMethod: "crypto", Price: 1}) {
		t.Fatal("unsupported payment method should be invalid")
	}
	if !validImagePaymentSettings(&repositories.PaymentSettingsInput{Enabled: true, PaymentMethod: "points", Price: 1}) {
		t.Fatal("points payment settings should be valid")
	}
}

func TestPaidContentPaymentMethodsCanBeEnabledIndependently(t *testing.T) {
	settings := services.NewSettingsService(nil, nil)
	handler := NativeHandlers{Settings: settings}
	if !handler.paidContentPaymentMethodEnabled("balance") || !handler.paidContentPaymentMethodEnabled("points") {
		t.Fatal("both paid content methods should default to enabled")
	}
	if !settings.Set(t.Context(), "paid_content_balance_enabled", false) {
		t.Fatal("disable balance setting")
	}
	if handler.paidContentPaymentMethodEnabled("balance") || !handler.paidContentPaymentMethodEnabled("points") {
		t.Fatal("balance and points switches should be independent")
	}
	if !settings.Set(t.Context(), "paid_content_points_enabled", false) {
		t.Fatal("disable points setting")
	}
	if handler.paidContentPaymentMethodEnabled("points") {
		t.Fatal("points should be disabled")
	}
}

func TestRejectInvalidImagePaymentRejectsPriceOverConfiguredLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := services.NewSettingsService(nil, nil)
	if !settings.Set(t.Context(), "paid_content_points_max_price", 88) {
		t.Fatal("set points max price")
	}
	handler := NativeHandlers{Settings: settings}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/posts", nil)

	rejected := handler.rejectInvalidImagePayment(
		context,
		[]repositories.PostImageInput{{URL: "free", IsFreePreview: true}, {URL: "paid", IsFreePreview: false}},
		&repositories.PaymentSettingsInput{Enabled: true, PaymentMethod: "points", Price: 89},
	)
	if !rejected {
		t.Fatal("expected over-limit payment to be rejected")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("error.paid_content_points_price_limit")) ||
		!bytes.Contains(recorder.Body.Bytes(), []byte(`"maxPrice":88`)) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestPostPurchaseUsersRequiresAuthorAndReturnsPurchasers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.Post{}, &domain.UserPurchasedContent{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	records := []any{
		&domain.User{ID: 1, UserID: "buyer", Nickname: "Buyer", CreatedAt: now},
		&domain.User{ID: 2, UserID: "author", Nickname: "Author", CreatedAt: now},
		&domain.User{ID: 3, UserID: "viewer", Nickname: "Viewer", CreatedAt: now},
		&domain.Post{ID: 100, UserID: 2, Title: "Paid Post", Type: 1, Visibility: "public", CreatedAt: now},
		&domain.UserPurchasedContent{ID: 10, UserID: 1, PostID: 100, AuthorID: 2, Price: 12, PaidAmount: 12, PaymentMethod: "points", PurchaseType: "single", PurchasedAt: now, CreatedAt: now},
	}
	for _, record := range records {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	handler := NativeHandlers{DB: db}

	forbiddenRecorder := httptest.NewRecorder()
	forbiddenContext, _ := gin.CreateTestContext(forbiddenRecorder)
	forbiddenContext.Params = gin.Params{{Key: "id", Value: "100"}}
	forbiddenContext.Request = httptest.NewRequest(http.MethodGet, "/api/posts/100/purchases", nil)
	forbiddenContext.Set("user", &services.RequestUser{ID: 3, UserID: "viewer"})
	handler.PostPurchaseUsers(forbiddenContext)
	if forbiddenRecorder.Code != http.StatusForbidden {
		t.Fatalf("forbidden status = %d, body = %s", forbiddenRecorder.Code, forbiddenRecorder.Body.String())
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Params = gin.Params{{Key: "id", Value: "100"}}
	context.Request = httptest.NewRequest(http.MethodGet, "/api/posts/100/purchases", nil)
	context.Set("user", &services.RequestUser{ID: 2, UserID: "author"})
	handler.PostPurchaseUsers(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	body := recorder.Body.String()
	if !bytes.Contains([]byte(body), []byte(`"nickname":"Buyer"`)) ||
		!bytes.Contains([]byte(body), []byte(`"payment_method":"points"`)) ||
		!bytes.Contains([]byte(body), []byte(`"total":1`)) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestMissingCompletedProtectedPackageExpiresForRegeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.ImageProtectionJob{}); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	job := domain.ImageProtectionJob{
		JobID:               "missing-package",
		PostID:              7,
		UserID:              9,
		Status:              protectedPackageCompleted,
		PackagePath:         filepath.Join(t.TempDir(), "missing.zip"),
		ProtectedImageCount: 2,
		CreatedAt:           now,
		UpdatedAt:           &now,
	}
	if err := db.Create(&job).Error; err != nil {
		t.Fatal(err)
	}
	handler := NativeHandlers{DB: db}
	if handler.protectedPackageJobReusable(&job) {
		t.Fatal("missing completed package must not be reused")
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/protected-packages/missing-package/download", nil)
	handler.expireProtectedPackageJob(context, &job, "package_missing")

	var updated domain.ImageProtectionJob
	if err := db.First(&updated, job.ID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Status != protectedPackageExpired || updated.PackagePath != "" || !updated.Retryable ||
		updated.ErrorCode == nil || *updated.ErrorCode != "package_missing" {
		t.Fatalf("expired job = %+v", updated)
	}

	readyPath := filepath.Join(t.TempDir(), "ready.zip")
	if err := os.WriteFile(readyPath, []byte("zip"), 0600); err != nil {
		t.Fatal(err)
	}
	ready := domain.ImageProtectionJob{Status: protectedPackageCompleted, PackagePath: readyPath}
	if !handler.protectedPackageJobReusable(&ready) {
		t.Fatal("existing completed package should be reusable")
	}
}

func TestUploadMultipleRejectsMoreThanMaximum(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for range defaultMaxPostImages + 1 {
		part, err := writer.CreateFormFile("files", "image.jpg")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := part.Write([]byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/upload/multiple", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = request

	NativeHandlers{}.UploadMultiple(context)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("error.upload_image_limit")) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestStoreImageCreatesEightByteWatermarkTrace(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.ImageWatermarkTrace{}); err != nil {
		t.Fatal(err)
	}
	var source bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 512, 512))
	for y := range 512 {
		for x := range 512 {
			img.SetRGBA(x, y, color.RGBA{R: uint8(x % 255), G: uint8(y % 255), B: uint8((x + y) % 255), A: 255})
		}
	}
	if err := jpeg.Encode(&source, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	settings := services.NewSettingsService(nil, nil)
	handler := NativeHandlers{
		DB:       db,
		Settings: settings,
		Config: config.Config{Upload: config.UploadConfig{
			FileSigning: config.UploadFileSigningConfig{Secret: "trace-secret"},
			Image: config.UploadImageConfig{
				Strategy:       "local",
				LocalUploadDir: filepath.Join(root, "images"),
				MaxSizeBytes:   10 << 20,
			},
			Temp: config.UploadTempConfig{RootDir: filepath.Join(root, "tmp")},
		}},
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/upload/single", nil)
	context.Set("user", &services.RequestUser{ID: 7, UserID: "account_7", Nickname: "Trace User"})
	clean, errMessage, ok := handler.storeImage(context, services.ImagePurposeContent, source.Bytes(), "clean.jpg", "image/jpeg")
	if !ok {
		t.Fatalf("protected-only storeImage() failed: %s", errMessage)
	}
	if clean.WatermarkTraceToken != "" || clean.Processed.WatermarkApplied {
		t.Fatalf("protected-only upload unexpectedly added a watermark: %+v", clean)
	}
	if !settings.Set(context.Request.Context(), "hidden_watermark_protected_only", false) {
		t.Fatal("disable protected-only watermark setting failed")
	}
	stored, errMessage, ok := handler.storeImage(context, services.ImagePurposeContent, source.Bytes(), "trace.jpg", "image/jpeg")
	if !ok {
		t.Fatalf("storeImage() failed: %s", errMessage)
	}
	if len(stored.WatermarkTraceToken) != domain.ImageWatermarkTraceTokenBytes*2 ||
		stored.WatermarkPayloadBytes != domain.ImageWatermarkPayloadBytes ||
		stored.WatermarkEngine != "local" {
		t.Fatalf("stored trace metadata = %+v", stored)
	}
	var trace domain.ImageWatermarkTrace
	if err := db.Where("token = ?", stored.WatermarkTraceToken).First(&trace).Error; err != nil {
		t.Fatal(err)
	}
	if trace.UserID != 7 || trace.SourceURL != stored.URL || trace.PayloadBytes != domain.ImageWatermarkPayloadBytes || trace.ShortCode == nil || trace.ShortCodeBytes != domain.ImageWatermarkShortCodeBytes {
		t.Fatalf("trace = %+v", trace)
	}
	handler.Config.Upload.Image.Strategy = "unsupported"
	if _, _, ok := handler.storeImage(context, services.ImagePurposeContent, source.Bytes(), "rollback.jpg", "image/jpeg"); ok {
		t.Fatal("unsupported strategy should fail")
	}
	var count int64
	if err := db.Model(&domain.ImageWatermarkTrace{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("trace count after failed upload = %d, want 1", count)
	}
}

func imageURLs(images []domain.PostImage) []string {
	out := make([]string, 0, len(images))
	for _, image := range images {
		out = append(out, image.ImageURL)
	}
	return out
}
