package handlers

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func TestPostImageArchiveContextRequiresPaidPurchase(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&domain.Post{},
		&domain.PostImage{},
		&domain.PostPaymentSetting{},
		&domain.UserPurchasedContent{},
	); err != nil {
		t.Fatal(err)
	}
	post := domain.Post{UserID: 7, Title: "paid archive", Visibility: "public"}
	if err := db.Create(&post).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.PostImage{PostID: post.ID, ImageURL: "/api/file/images/one.webp"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.PostPaymentSetting{PostID: post.ID, Enabled: true, Price: 10}).Error; err != nil {
		t.Fatal(err)
	}
	handler := NativeHandlers{DB: db}
	if _, _, _, canViewPaid, err := handler.postImageArchiveContext(context.Background(), post.ID, 0); err != nil || canViewPaid {
		t.Fatalf("guest canViewPaid = %v, err = %v", canViewPaid, err)
	}
	if _, _, _, canViewPaid, err := handler.postImageArchiveContext(context.Background(), post.ID, 9); err != nil || canViewPaid {
		t.Fatalf("unpaid user canViewPaid = %v, err = %v", canViewPaid, err)
	}
	now := time.Now()
	if err := db.Create(&domain.UserPurchasedContent{
		UserID: 9, PostID: post.ID, AuthorID: post.UserID, Price: 10, PaidAmount: 10, PurchasedAt: now, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if _, _, _, canViewPaid, err := handler.postImageArchiveContext(context.Background(), post.ID, 9); err != nil || !canViewPaid {
		t.Fatalf("purchased user canViewPaid = %v, err = %v", canViewPaid, err)
	}
	if _, _, _, canViewPaid, err := handler.postImageArchiveContext(context.Background(), post.ID, post.UserID); err != nil || !canViewPaid {
		t.Fatalf("author canViewPaid = %v, err = %v", canViewPaid, err)
	}
}

func TestPostImageArchiveSignatureChangesWithImageOrderOrURL(t *testing.T) {
	images := []domain.PostImage{
		{ID: 1, ImageURL: "one.webp", SortOrder: 0},
		{ID: 2, ImageURL: "two.webp", SortOrder: 1},
	}
	original := postImageArchiveSignature(images)
	images[1].ImageURL = "two-updated.webp"
	if updated := postImageArchiveSignature(images); updated == original {
		t.Fatal("signature must change when an image URL changes")
	}
	images[1].ImageURL = "two.webp"
	images[0].SortOrder = 2
	if reordered := postImageArchiveSignature(images); reordered == original {
		t.Fatal("signature must change when image order changes")
	}
}

func TestImageArchiveEligibilityRequiresCountAboveThreshold(t *testing.T) {
	handler := NativeHandlers{}
	body := gin.H{}
	handler.applyImageArchiveMetadata(body, imageAccessSummary{TotalImagesCount: 25}, nil, true)
	if eligible, _ := body["imageArchiveEligible"].(bool); eligible {
		t.Fatal("25 images must not exceed the default threshold of 25")
	}
	handler.applyImageArchiveMetadata(body, imageAccessSummary{TotalImagesCount: 26}, nil, true)
	if eligible, _ := body["imageArchiveEligible"].(bool); !eligible {
		t.Fatal("26 images must exceed the default threshold of 25")
	}
}
