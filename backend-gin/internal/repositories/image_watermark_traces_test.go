package repositories

import (
	"context"
	"errors"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func TestReserveImageWatermarkTraceRetriesTokenCollision(t *testing.T) {
	db := watermarkTraceTestDB(t)
	if err := db.Create(&domain.ImageWatermarkTrace{Token: "0102030405060708", TraceType: domain.ImageWatermarkTraceUpload}).Error; err != nil {
		t.Fatal(err)
	}
	original := imageWatermarkTokenGenerator
	t.Cleanup(func() { imageWatermarkTokenGenerator = original })
	calls := 0
	imageWatermarkTokenGenerator = func() (string, error) {
		calls++
		if calls == 1 {
			return "0102030405060708", nil
		}
		return "1112131415161718", nil
	}
	trace, err := ReserveImageWatermarkTrace(context.Background(), db, ImageWatermarkTraceInput{TraceType: domain.ImageWatermarkTraceUpload})
	if err != nil {
		t.Fatalf("ReserveImageWatermarkTrace() error = %v", err)
	}
	if trace.Token != "1112131415161718" || calls != 2 {
		t.Fatalf("trace=%+v calls=%d", trace, calls)
	}
}

func TestReserveImageWatermarkTraceCreatesResolvableShortCode(t *testing.T) {
	db := watermarkTraceTestDB(t)
	originalToken := imageWatermarkTokenGenerator
	originalShortCode := imageWatermarkShortCodeGenerator
	t.Cleanup(func() {
		imageWatermarkTokenGenerator = originalToken
		imageWatermarkShortCodeGenerator = originalShortCode
	})
	imageWatermarkTokenGenerator = func() (string, error) {
		return "0102030405060708", nil
	}
	imageWatermarkShortCodeGenerator = func(_ int) (string, error) {
		return "a1b2c3d4", nil
	}
	trace, err := ReserveImageWatermarkTrace(context.Background(), db, ImageWatermarkTraceInput{
		TraceType:      domain.ImageWatermarkTraceProtected,
		ShortCodeBytes: domain.ImageWatermarkShortCodeBytes,
	})
	if err != nil {
		t.Fatalf("ReserveImageWatermarkTrace() error = %v", err)
	}
	if trace.ShortCode == nil || *trace.ShortCode != "a1b2c3d4" || trace.ShortCodeBytes != domain.ImageWatermarkShortCodeBytes {
		t.Fatalf("short code trace = %+v", trace)
	}
	resolved, err := ResolveImageWatermarkTraceByShortCode(context.Background(), db, "a1b2c3d4", domain.ImageWatermarkShortCodeBytes)
	if err != nil {
		t.Fatalf("ResolveImageWatermarkTraceByShortCode() error = %v", err)
	}
	if resolved.Token != trace.Token {
		t.Fatalf("resolved token = %q, want %q", resolved.Token, trace.Token)
	}
}

func TestReserveImageWatermarkTraceRetriesShortCodeCollision(t *testing.T) {
	db := watermarkTraceTestDB(t)
	existingCode := "a1b2c3d4"
	if err := db.Create(&domain.ImageWatermarkTrace{
		Token:          "0102030405060708",
		ShortCode:      &existingCode,
		ShortCodeBytes: domain.ImageWatermarkShortCodeBytes,
		TraceType:      domain.ImageWatermarkTraceProtected,
	}).Error; err != nil {
		t.Fatal(err)
	}
	originalToken := imageWatermarkTokenGenerator
	originalShortCode := imageWatermarkShortCodeGenerator
	t.Cleanup(func() {
		imageWatermarkTokenGenerator = originalToken
		imageWatermarkShortCodeGenerator = originalShortCode
	})
	tokenCalls := 0
	imageWatermarkTokenGenerator = func() (string, error) {
		tokenCalls++
		if tokenCalls == 1 {
			return "1112131415161718", nil
		}
		return "2122232425262728", nil
	}
	shortCodeCalls := 0
	imageWatermarkShortCodeGenerator = func(_ int) (string, error) {
		shortCodeCalls++
		if shortCodeCalls == 1 {
			return existingCode, nil
		}
		return "b1b2b3b4", nil
	}
	trace, err := ReserveImageWatermarkTrace(context.Background(), db, ImageWatermarkTraceInput{
		TraceType:      domain.ImageWatermarkTraceProtected,
		ShortCodeBytes: domain.ImageWatermarkShortCodeBytes,
	})
	if err != nil {
		t.Fatalf("ReserveImageWatermarkTrace() error = %v", err)
	}
	if trace.Token != "2122232425262728" || trace.ShortCode == nil || *trace.ShortCode != "b1b2b3b4" {
		t.Fatalf("trace=%+v", trace)
	}
	if tokenCalls != 2 || shortCodeCalls != 2 {
		t.Fatalf("calls token=%d short=%d, want 2/2", tokenCalls, shortCodeCalls)
	}
}

func TestListImageWatermarkRecoverDimensions(t *testing.T) {
	db := watermarkTraceTestDB(t)
	traces := []domain.ImageWatermarkTrace{
		{Token: "0102030405060708", TraceType: domain.ImageWatermarkTraceProtected, WatermarkWidth: 720, WatermarkHeight: 960},
		{Token: "1112131415161718", TraceType: domain.ImageWatermarkTraceProtected, WatermarkWidth: 720, WatermarkHeight: 960},
		{Token: "2122232425262728", TraceType: domain.ImageWatermarkTraceProtected, WatermarkWidth: 1280, WatermarkHeight: 720},
		{Token: "3132333435363738", TraceType: domain.ImageWatermarkTraceUpload, WatermarkWidth: 2048, WatermarkHeight: 2048},
	}
	if err := db.Create(&traces).Error; err != nil {
		t.Fatal(err)
	}
	dimensions, err := ListImageWatermarkRecoverDimensions(context.Background(), db, 10)
	if err != nil {
		t.Fatalf("ListImageWatermarkRecoverDimensions() error = %v", err)
	}
	want := [][2]int{{720, 960}, {1280, 720}}
	if len(dimensions) != len(want) {
		t.Fatalf("dimensions = %#v, want %#v", dimensions, want)
	}
	for index := range want {
		if dimensions[index] != want[index] {
			t.Fatalf("dimensions[%d] = %#v, want %#v", index, dimensions[index], want[index])
		}
	}
}

func TestBindAndPreparePostImageWatermarkTraces(t *testing.T) {
	ctx := context.Background()
	db := watermarkTraceTestDB(t)
	postID := int64(77)
	keptImageID := int64(9)
	removedImageID := int64(10)
	kept := domain.ImageWatermarkTrace{
		Token: "0102030405060708", TraceType: domain.ImageWatermarkTraceUpload,
		PostID: &postID, ImageID: &keptImageID, SourceURL: "/api/file/images/kept.webp",
	}
	removed := domain.ImageWatermarkTrace{
		Token: "1112131415161718", TraceType: domain.ImageWatermarkTraceProtected,
		PostID: &postID, ImageID: &removedImageID, SourceURL: "/api/file/images/removed.webp",
	}
	if err := db.Create(&[]domain.ImageWatermarkTrace{kept, removed}).Error; err != nil {
		t.Fatal(err)
	}
	if err := PreparePostImageWatermarkTraceRebind(ctx, db, postID, []PostImageInput{{URL: kept.SourceURL, WatermarkTraceToken: kept.Token}}); err != nil {
		t.Fatal(err)
	}
	if err := db.Where("token = ?", removed.Token).First(&domain.ImageWatermarkTrace{}).Error; !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("removed trace error = %v, want record not found", err)
	}
	var prepared domain.ImageWatermarkTrace
	if err := db.Where("token = ?", kept.Token).First(&prepared).Error; err != nil {
		t.Fatal(err)
	}
	if prepared.ImageID != nil {
		t.Fatalf("prepared image id = %v, want nil", prepared.ImageID)
	}
	image := domain.PostImage{PostID: postID, ImageURL: kept.SourceURL, WatermarkTraceToken: kept.Token}
	if err := db.Create(&image).Error; err != nil {
		t.Fatal(err)
	}
	if err := BindPostImageWatermarkTraces(ctx, db, postID, 0, []domain.PostImage{image}); err != nil {
		t.Fatal(err)
	}
	var rebound domain.ImageWatermarkTrace
	if err := db.Where("token = ?", kept.Token).First(&rebound).Error; err != nil {
		t.Fatal(err)
	}
	if rebound.ImageID == nil || *rebound.ImageID != image.ID {
		t.Fatalf("rebound trace = %+v, image id = %d", rebound, image.ID)
	}
}

func TestBindPostImageWatermarkTraceRejectsAnotherUsersToken(t *testing.T) {
	ctx := context.Background()
	db := watermarkTraceTestDB(t)
	trace := domain.ImageWatermarkTrace{
		Token:     "0102030405060708",
		TraceType: domain.ImageWatermarkTraceUpload,
		UserID:    7,
		SourceURL: "/api/file/images/private.webp",
	}
	if err := db.Create(&trace).Error; err != nil {
		t.Fatal(err)
	}
	image := domain.PostImage{PostID: 88, ImageURL: trace.SourceURL, WatermarkTraceToken: trace.Token}
	if err := db.Create(&image).Error; err != nil {
		t.Fatal(err)
	}
	err := BindPostImageWatermarkTraces(ctx, db, image.PostID, 9, []domain.PostImage{image})
	if !errors.Is(err, ErrContentInvalidArgument) {
		t.Fatalf("BindPostImageWatermarkTraces() error = %v, want ErrContentInvalidArgument", err)
	}
}

func TestBindPostImageWatermarkTraceRejectsWrongImageOrReusedPost(t *testing.T) {
	ctx := context.Background()
	db := watermarkTraceTestDB(t)
	originalPostID := int64(77)
	trace := domain.ImageWatermarkTrace{
		Token:     "0102030405060708",
		TraceType: domain.ImageWatermarkTraceUpload,
		UserID:    7,
		PostID:    &originalPostID,
		SourceURL: "/api/file/images/original.webp",
	}
	if err := db.Create(&trace).Error; err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name   string
		postID int64
		url    string
	}{
		{name: "wrong image URL", postID: originalPostID, url: "/api/file/images/other.webp"},
		{name: "reused on another post", postID: 88, url: trace.SourceURL},
	} {
		t.Run(test.name, func(t *testing.T) {
			image := domain.PostImage{PostID: test.postID, ImageURL: test.url, WatermarkTraceToken: trace.Token}
			if err := db.Create(&image).Error; err != nil {
				t.Fatal(err)
			}
			err := BindPostImageWatermarkTraces(ctx, db, test.postID, trace.UserID, []domain.PostImage{image})
			if !errors.Is(err, ErrContentInvalidArgument) {
				t.Fatalf("BindPostImageWatermarkTraces() error = %v, want ErrContentInvalidArgument", err)
			}
		})
	}
}

func watermarkTraceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.ImageWatermarkTrace{}, &domain.PostImage{}); err != nil {
		t.Fatal(err)
	}
	return db
}
