package repositories

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"yuem-go/backend-gin/internal/domain"
)

type queryCounterLogger struct {
	mu    sync.Mutex
	count int
}

func (l *queryCounterLogger) LogMode(logger.LogLevel) logger.Interface { return l }
func (l *queryCounterLogger) Info(context.Context, string, ...any)     {}
func (l *queryCounterLogger) Warn(context.Context, string, ...any)     {}
func (l *queryCounterLogger) Error(context.Context, string, ...any)    {}

func (l *queryCounterLogger) Trace(context.Context, time.Time, func() (string, int64), error) {
	l.mu.Lock()
	l.count++
	l.mu.Unlock()
}

func (l *queryCounterLogger) Reset() {
	l.mu.Lock()
	l.count = 0
	l.mu.Unlock()
}

func (l *queryCounterLogger) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.count
}

func TestContentListKeepsBatchQueryCount(t *testing.T) {
	db, counter := openPerformanceTestDB(t,
		&domain.User{},
		&domain.Category{},
		&domain.Post{},
		&domain.PostImage{},
		&domain.PostVideo{},
		&domain.PostAttachment{},
		&domain.PostPaymentSetting{},
		&domain.Tag{},
		&domain.PostTag{},
	)
	now := time.Now()
	category := domain.Category{ID: 1, Name: "photo"}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}
	tags := []domain.Tag{{ID: 1, Name: "tag-a"}, {ID: 2, Name: "tag-b"}}
	if err := db.Create(&tags).Error; err != nil {
		t.Fatalf("create tags: %v", err)
	}
	for i := 1; i <= 12; i++ {
		user := domain.User{ID: int64(i), UserID: fmt.Sprintf("u%d", i), Nickname: fmt.Sprintf("User %d", i), IsActive: true}
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user %d: %v", i, err)
		}
		post := domain.Post{ID: int64(i), UserID: user.ID, CategoryID: &category.ID, Title: fmt.Sprintf("Post %d", i), Type: PostTypeImage, Visibility: VisibilityPublic, CreatedAt: now.Add(time.Duration(i) * time.Minute)}
		if err := db.Create(&post).Error; err != nil {
			t.Fatalf("create post %d: %v", i, err)
		}
		if err := db.Create(&[]domain.PostImage{
			{PostID: post.ID, ImageURL: fmt.Sprintf("/api/file/images/%d-1.webp", i), SortOrder: 1},
			{PostID: post.ID, ImageURL: fmt.Sprintf("/api/file/images/%d-2.webp", i), SortOrder: 2},
		}).Error; err != nil {
			t.Fatalf("create images %d: %v", i, err)
		}
		if err := db.Create(&domain.PostVideo{PostID: post.ID, VideoURL: fmt.Sprintf("/api/file/videos/%d.mp4", i)}).Error; err != nil {
			t.Fatalf("create video %d: %v", i, err)
		}
		if err := db.Create(&domain.PostAttachment{PostID: post.ID, AttachmentURL: fmt.Sprintf("/api/file/attachments/%d.pdf", i), Filename: "file.pdf"}).Error; err != nil {
			t.Fatalf("create attachment %d: %v", i, err)
		}
		if err := db.Create(&domain.PostPaymentSetting{PostID: post.ID, Enabled: i%2 == 0}).Error; err != nil {
			t.Fatalf("create payment %d: %v", i, err)
		}
		if err := db.Create(&domain.PostTag{PostID: post.ID, TagID: tags[i%len(tags)].ID}).Error; err != nil {
			t.Fatalf("create post tag %d: %v", i, err)
		}
	}

	counter.Reset()
	result, err := NewContentRepository(db).ListPosts(context.Background(), PostListOptions{Page: 1, Limit: 12})
	if err != nil {
		t.Fatalf("ListPosts() error = %v", err)
	}
	if len(result.Posts) != 12 {
		t.Fatalf("post count = %d, want 12", len(result.Posts))
	}
	if got := counter.Count(); got > 10 {
		t.Fatalf("ListPosts query count = %d, want <= 10 to avoid per-post lookups", got)
	}
}

func TestCommentsKeepBatchQueryCount(t *testing.T) {
	db, counter := openPerformanceTestDB(t, &domain.User{}, &domain.Post{}, &domain.Comment{}, &domain.Like{})
	now := time.Now()
	post := domain.Post{ID: 1, UserID: 1, Title: "post", Type: PostTypeImage, Visibility: VisibilityPublic, CreatedAt: now}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post: %v", err)
	}
	for i := 1; i <= 8; i++ {
		user := domain.User{ID: int64(i), UserID: fmt.Sprintf("u%d", i), Nickname: fmt.Sprintf("User %d", i), IsActive: true}
		if err := db.Create(&user).Error; err != nil {
			t.Fatalf("create user %d: %v", i, err)
		}
		comment := domain.Comment{ID: int64(i), PostID: post.ID, UserID: user.ID, Content: "comment", IsPublic: true, CreatedAt: now.Add(time.Duration(i) * time.Minute)}
		if err := db.Create(&comment).Error; err != nil {
			t.Fatalf("create comment %d: %v", i, err)
		}
		replyParent := comment.ID
		if err := db.Create(&domain.Comment{ID: int64(100 + i), PostID: post.ID, ParentID: &replyParent, UserID: user.ID, Content: "reply", IsPublic: true, CreatedAt: now}).Error; err != nil {
			t.Fatalf("create reply %d: %v", i, err)
		}
		if i%2 == 0 {
			if err := db.Create(&domain.Like{UserID: 99, TargetType: 2, TargetID: comment.ID, CreatedAt: now}).Error; err != nil {
				t.Fatalf("create like %d: %v", i, err)
			}
		}
	}

	counter.Reset()
	total, comments, err := NewContentRepository(db).Comments(context.Background(), post.ID, nil, 99, 1, 8, false, true)
	if err != nil {
		t.Fatalf("Comments() error = %v", err)
	}
	if total != 8 || len(comments) != 8 {
		t.Fatalf("comments total=%d len=%d, want 8/8", total, len(comments))
	}
	if got := counter.Count(); got > 6 {
		t.Fatalf("Comments query count = %d, want <= 6 to avoid per-comment lookups", got)
	}
}

func TestNotificationsKeepBatchQueryCount(t *testing.T) {
	db, counter := openPerformanceTestDB(t,
		&domain.User{},
		&domain.Post{},
		&domain.PostImage{},
		&domain.Comment{},
		&domain.Notification{},
		&domain.Blacklist{},
	)
	now := time.Now()
	viewerID := int64(100)
	for i := 1; i <= 10; i++ {
		sender := domain.User{ID: int64(i), UserID: fmt.Sprintf("sender%d", i), Nickname: fmt.Sprintf("Sender %d", i), IsActive: true}
		if err := db.Create(&sender).Error; err != nil {
			t.Fatalf("create sender %d: %v", i, err)
		}
		post := domain.Post{ID: int64(i), UserID: sender.ID, Title: fmt.Sprintf("Post %d", i), Type: PostTypeImage, Visibility: VisibilityPublic, CreatedAt: now}
		if err := db.Create(&post).Error; err != nil {
			t.Fatalf("create post %d: %v", i, err)
		}
		if err := db.Create(&domain.PostImage{PostID: post.ID, ImageURL: fmt.Sprintf("/api/file/images/%d.webp", i), SortOrder: 1}).Error; err != nil {
			t.Fatalf("create image %d: %v", i, err)
		}
		comment := domain.Comment{ID: int64(i), PostID: post.ID, UserID: sender.ID, Content: "comment", IsPublic: true, CreatedAt: now}
		if err := db.Create(&comment).Error; err != nil {
			t.Fatalf("create comment %d: %v", i, err)
		}
		targetID := post.ID
		commentID := comment.ID
		notification := domain.Notification{ID: int64(i), UserID: viewerID, SenderID: sender.ID, Type: 3, Title: "commented", TargetID: &targetID, CommentID: &commentID, CreatedAt: now.Add(time.Duration(i) * time.Minute)}
		if err := db.Create(&notification).Error; err != nil {
			t.Fatalf("create notification %d: %v", i, err)
		}
	}

	counter.Reset()
	total, notifications, err := NewNotificationsRepository(db).List(context.Background(), viewerID, NotificationListParams{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("List notifications error = %v", err)
	}
	if total != 10 || len(notifications) != 10 {
		t.Fatalf("notifications total=%d len=%d, want 10/10", total, len(notifications))
	}
	if got := counter.Count(); got > 8 {
		t.Fatalf("Notifications query count = %d, want <= 8 to avoid per-notification lookups", got)
	}
}

func openPerformanceTestDB(t *testing.T, models ...any) (*gorm.DB, *queryCounterLogger) {
	t.Helper()
	counter := &queryCounterLogger{}
	name := fmt.Sprintf("performance_%d", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   counter,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	return db, counter
}
