package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func TestLikeNotificationsRespectSuppressionByReceiverSenderAndType(t *testing.T) {
	db := newNotificationSuppressionTestDB(t)
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	createSuppressionUsers(t, db, 10, 20, 30)
	createSuppressionPost(t, db, 101, 10)
	createSuppressionPost(t, db, 102, 30)
	cfg := NotificationSuppressionConfig{Enabled: true, WindowSeconds: 600, Threshold: 3, Now: func() time.Time { return now }}
	repo := NewInteractionsRepository(db, cfg)

	for range 3 {
		id := createSuppressionPost(t, db, 0, 10)
		result, err := repo.ToggleLike(context.Background(), 20, 1, id)
		if err != nil {
			t.Fatalf("ToggleLike seed: %v", err)
		}
		if !result.Liked {
			t.Fatal("seed like should create a like")
		}
	}
	if got := countSuppressionNotifications(t, db, 10, 20, NotificationLikePost); got != 3 {
		t.Fatalf("seed notifications = %d, want 3", got)
	}

	result, err := repo.ToggleLike(context.Background(), 20, 1, 101)
	if err != nil {
		t.Fatalf("ToggleLike suppressed: %v", err)
	}
	if !result.Liked {
		t.Fatal("suppressed like should still create the like")
	}
	if got := countSuppressionNotifications(t, db, 10, 20, NotificationLikePost); got != 3 {
		t.Fatalf("suppressed notifications = %d, want 3", got)
	}

	result, err = repo.ToggleLike(context.Background(), 20, 1, 102)
	if err != nil {
		t.Fatalf("ToggleLike other receiver: %v", err)
	}
	if !result.Liked {
		t.Fatal("other receiver like should still create the like")
	}
	if got := countSuppressionNotifications(t, db, 30, 20, NotificationLikePost); got != 1 {
		t.Fatalf("other receiver notifications = %d, want 1", got)
	}
}

func TestCommentLikeAndCollectionNotificationsHaveIndependentSuppressionTypes(t *testing.T) {
	db := newNotificationSuppressionTestDB(t)
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	createSuppressionUsers(t, db, 10, 20, 30)
	postID := createSuppressionPost(t, db, 0, 10)
	commentID := createSuppressionComment(t, db, postID, 10)
	cfg := NotificationSuppressionConfig{Enabled: true, WindowSeconds: 600, Threshold: 1, Now: func() time.Time { return now }}

	if _, err := NewInteractionsRepository(db, cfg).ToggleLike(context.Background(), 20, 1, postID); err != nil {
		t.Fatalf("post like: %v", err)
	}
	if _, err := NewInteractionsRepository(db, cfg).ToggleLike(context.Background(), 20, 2, commentID); err != nil {
		t.Fatalf("comment like: %v", err)
	}
	if _, err := NewContentRepository(db, cfg).ToggleCollection(context.Background(), 20, postID); err != nil {
		t.Fatalf("collection: %v", err)
	}

	if got := countSuppressionNotifications(t, db, 10, 20, NotificationLikePost); got != 1 {
		t.Fatalf("post like notifications = %d, want 1", got)
	}
	if got := countSuppressionNotifications(t, db, 10, 20, NotificationLikeComment); got != 1 {
		t.Fatalf("comment like notifications = %d, want 1", got)
	}
	if got := countSuppressionNotifications(t, db, 10, 20, 6); got != 1 {
		t.Fatalf("collection notifications = %d, want 1", got)
	}

	otherCommentID := createSuppressionComment(t, db, postID, 10)
	if _, err := NewInteractionsRepository(db, cfg).ToggleLike(context.Background(), 20, 2, otherCommentID); err != nil {
		t.Fatalf("suppressed comment like: %v", err)
	}
	otherPostID := createSuppressionPost(t, db, 0, 10)
	if _, err := NewContentRepository(db, cfg).ToggleCollection(context.Background(), 20, otherPostID); err != nil {
		t.Fatalf("suppressed collection: %v", err)
	}
	if got := countSuppressionNotifications(t, db, 10, 20, NotificationLikeComment); got != 1 {
		t.Fatalf("suppressed comment like notifications = %d, want 1", got)
	}
	if got := countSuppressionNotifications(t, db, 10, 20, 6); got != 1 {
		t.Fatalf("suppressed collection notifications = %d, want 1", got)
	}

	thirdPostID := createSuppressionPost(t, db, 0, 10)
	if _, err := NewContentRepository(db, cfg).ToggleCollection(context.Background(), 30, thirdPostID); err != nil {
		t.Fatalf("different sender collection: %v", err)
	}
	if got := countSuppressionNotifications(t, db, 10, 30, 6); got != 1 {
		t.Fatalf("different sender collection notifications = %d, want 1", got)
	}
}

func TestNotificationSuppressionIgnoresOldRowsAndCanBeDisabled(t *testing.T) {
	db := newNotificationSuppressionTestDB(t)
	now := time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC)
	createSuppressionUsers(t, db, 10, 20)
	oldPostID := createSuppressionPost(t, db, 0, 10)
	oldTargetID := oldPostID
	if err := db.Create(&domain.Notification{
		UserID:    10,
		SenderID:  20,
		Type:      NotificationLikePost,
		Title:     "赞了你的笔记",
		TargetID:  &oldTargetID,
		CreatedAt: now.Add(-11 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("create old notification: %v", err)
	}

	cfg := NotificationSuppressionConfig{Enabled: true, WindowSeconds: 600, Threshold: 1, Now: func() time.Time { return now }}
	if _, err := NewInteractionsRepository(db, cfg).ToggleLike(context.Background(), 20, 1, createSuppressionPost(t, db, 0, 10)); err != nil {
		t.Fatalf("window like: %v", err)
	}
	if got := countSuppressionNotifications(t, db, 10, 20, NotificationLikePost); got != 2 {
		t.Fatalf("notifications after old row ignored = %d, want 2", got)
	}

	disabled := NotificationSuppressionConfig{Enabled: false, WindowSeconds: 600, Threshold: 1, Now: func() time.Time { return now }}
	if _, err := NewInteractionsRepository(db, disabled).ToggleLike(context.Background(), 20, 1, createSuppressionPost(t, db, 0, 10)); err != nil {
		t.Fatalf("disabled suppression like: %v", err)
	}
	if got := countSuppressionNotifications(t, db, 10, 20, NotificationLikePost); got != 3 {
		t.Fatalf("disabled suppression notifications = %d, want 3", got)
	}
}

func TestSelfInteractionsDoNotCreateNotifications(t *testing.T) {
	db := newNotificationSuppressionTestDB(t)
	createSuppressionUsers(t, db, 10)
	postID := createSuppressionPost(t, db, 0, 10)
	commentID := createSuppressionComment(t, db, postID, 10)
	cfg := NotificationSuppressionConfig{Enabled: true, WindowSeconds: 600, Threshold: 1}

	if _, err := NewInteractionsRepository(db, cfg).ToggleLike(context.Background(), 10, 1, postID); err != nil {
		t.Fatalf("self post like: %v", err)
	}
	if _, err := NewInteractionsRepository(db, cfg).ToggleLike(context.Background(), 10, 2, commentID); err != nil {
		t.Fatalf("self comment like: %v", err)
	}
	if _, err := NewContentRepository(db, cfg).ToggleCollection(context.Background(), 10, postID); err != nil {
		t.Fatalf("self collection: %v", err)
	}
	var total int64
	if err := db.Model(&domain.Notification{}).Count(&total).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	if total != 0 {
		t.Fatalf("self interaction notifications = %d, want 0", total)
	}
}

func newNotificationSuppressionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	if err := db.AutoMigrate(&domain.User{}, &domain.Post{}, &domain.Comment{}, &domain.Like{}, &domain.Collection{}, &domain.Notification{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	return db
}

func createSuppressionUsers(t *testing.T, db *gorm.DB, ids ...int64) {
	t.Helper()
	for _, id := range ids {
		if err := db.Create(&domain.User{ID: id, UserID: "user", Nickname: "User", IsActive: true}).Error; err != nil {
			t.Fatalf("create user %d: %v", id, err)
		}
	}
}

func createSuppressionPost(t *testing.T, db *gorm.DB, id int64, userID int64) int64 {
	t.Helper()
	post := domain.Post{ID: id, UserID: userID, Title: "post", Type: PostTypeImage, Visibility: VisibilityPublic, QualityLevel: PostQualityNone}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	return post.ID
}

func createSuppressionComment(t *testing.T, db *gorm.DB, postID int64, userID int64) int64 {
	t.Helper()
	comment := domain.Comment{PostID: postID, UserID: userID, Content: "comment", AuditStatus: 1, IsPublic: true}
	if err := db.Create(&comment).Error; err != nil {
		t.Fatalf("create comment: %v", err)
	}
	return comment.ID
}

func countSuppressionNotifications(t *testing.T, db *gorm.DB, userID int64, senderID int64, notificationType int) int64 {
	t.Helper()
	var total int64
	if err := db.Model(&domain.Notification{}).Where("user_id = ? AND sender_id = ? AND type = ?", userID, senderID, notificationType).Count(&total).Error; err != nil {
		t.Fatalf("count notifications: %v", err)
	}
	return total
}
