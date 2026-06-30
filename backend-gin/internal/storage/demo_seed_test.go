package storage

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/security"
)

func TestSeedDemoDataCreatesAndIsIdempotent(t *testing.T) {
	db := openDemoSeedTestDB(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 30, 10, 0, 0, 0, time.UTC)

	result, err := SeedDemoData(ctx, db, DemoSeedOptions{
		Password:  "TestDemo123!",
		UserLimit: 6,
		PostLimit: 7,
		Now:       now,
	})
	if err != nil {
		t.Fatalf("SeedDemoData() error = %v", err)
	}
	if result.UserCount != 6 || result.PostCount != 7 {
		t.Fatalf("SeedDemoData() targets = users %d posts %d, want 6 and 7", result.UserCount, result.PostCount)
	}
	if len(result.LoginAccounts) != 6 {
		t.Fatalf("login accounts = %d, want 6", len(result.LoginAccounts))
	}

	var alice domain.User
	if err := db.Where("user_id = ?", "demo_alice").First(&alice).Error; err != nil {
		t.Fatalf("find demo_alice: %v", err)
	}
	if alice.Email == nil || *alice.Email != "demo-alice@example.test" {
		t.Fatalf("demo_alice email = %v, want demo-alice@example.test", alice.Email)
	}
	if alice.Password == nil || !security.VerifyPassword("TestDemo123!", *alice.Password) {
		t.Fatal("demo_alice password was not seeded with a verifiable Argon2id hash")
	}
	assertChineseDemoRows(t, db)

	assertRowCount(t, db, &domain.User{}, 6)
	assertRowCount(t, db, &domain.Category{}, int64(len(demoCategorySeeds)))
	assertRowCount(t, db, &domain.Tag{}, int64(len(demoTagSeeds)))
	assertRowCount(t, db, &domain.Post{}, 7)
	assertMinimumRows(t, db, &domain.PostImage{}, 10)
	assertRowCount(t, db, &domain.PostVideo{}, 1)
	assertRowCount(t, db, &domain.PostAttachment{}, 1)
	assertRowCount(t, db, &domain.PostPaymentSetting{}, 1)
	assertMinimumRows(t, db, &domain.Comment{}, 7)
	assertMinimumRows(t, db, &domain.Like{}, 7)
	assertMinimumRows(t, db, &domain.Collection{}, 1)
	assertMinimumRows(t, db, &domain.Notification{}, 1)
	assertRowCount(t, db, &domain.UserPoints{}, 6)
	assertRowCount(t, db, &domain.UserWallet{}, 6)
	assertRowCount(t, db, &domain.CreatorEarnings{}, 6)
	assertRowCount(t, db, &domain.GiftCardProduct{}, 2)
	assertRowCount(t, db, &domain.GiftCardCode{}, 6)
	assertRowCount(t, db, &domain.Announcement{}, 1)
	assertRowCount(t, db, &domain.SystemNotification{}, 1)
	assertRowCount(t, db, &domain.IMConversation{}, 1)
	assertRowCount(t, db, &domain.IMMessage{}, 3)

	simulateLegacyEnglishDemoRows(t, db, alice.ID)
	before := demoSeedRowCounts(t, db)
	second, err := SeedDemoData(ctx, db, DemoSeedOptions{
		Password:  "ChangedDemo123!",
		UserLimit: 6,
		PostLimit: 7,
		Now:       now.Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("SeedDemoData() second run error = %v", err)
	}
	if second.UsersCreated != 0 || second.PostsCreated != 0 {
		t.Fatalf("second run created users=%d posts=%d, want 0 and 0", second.UsersCreated, second.PostsCreated)
	}
	after := demoSeedRowCounts(t, db)
	for table, count := range before {
		if after[table] != count {
			t.Fatalf("second run changed row count for %s: got %d, want %d", table, after[table], count)
		}
	}
	if err := db.Where("user_id = ?", "demo_alice").First(&alice).Error; err != nil {
		t.Fatalf("reload demo_alice: %v", err)
	}
	if alice.Password == nil || !security.VerifyPassword("ChangedDemo123!", *alice.Password) {
		t.Fatal("second run did not reset demo_alice password")
	}
	assertChineseDemoRows(t, db)
}

func assertChineseDemoRows(t *testing.T, db *gorm.DB) {
	t.Helper()
	var alice domain.User
	if err := db.Where("user_id = ?", "demo_alice").First(&alice).Error; err != nil {
		t.Fatalf("find demo_alice: %v", err)
	}
	if alice.Nickname != "林晓雨" || alice.Location == nil || !strings.Contains(*alice.Location, "中国") {
		t.Fatalf("demo_alice profile not Chinese: nickname=%q location=%v", alice.Nickname, alice.Location)
	}

	var category domain.Category
	if err := db.Where("name = ?", "photography").First(&category).Error; err != nil {
		t.Fatalf("find photography category: %v", err)
	}
	if category.CategoryTitle == nil || *category.CategoryTitle != "摄影" {
		t.Fatalf("photography category title = %v, want 摄影", category.CategoryTitle)
	}

	var tag domain.Tag
	if err := db.Where("name = ?", "摄影").First(&tag).Error; err != nil {
		t.Fatalf("find 摄影 tag: %v", err)
	}

	var post domain.Post
	if err := db.Where("title = ?", "图书馆清晨的第一束光").First(&post).Error; err != nil {
		t.Fatalf("find Chinese post: %v", err)
	}
	if !strings.Contains(post.Content, "上海高校") {
		t.Fatalf("Chinese post content = %q", post.Content)
	}

	var oldPostCount int64
	if err := db.Model(&domain.Post{}).Where("title = ?", "Morning light in the library").Count(&oldPostCount).Error; err != nil {
		t.Fatalf("count legacy English post: %v", err)
	}
	if oldPostCount != 0 {
		t.Fatalf("legacy English post rows = %d, want 0", oldPostCount)
	}

	assertTextRowExists(t, db, &domain.Comment{}, "content = ?", "这条中文演示内容让信息流更接近真实使用场景。")
	assertTextRowExists(t, db, &domain.Notification{}, "title = ?", "点赞了你的帖子")
	assertTextRowExists(t, db, &domain.GiftCardProduct{}, "name = ?", "演示咖啡兑换卡")
	assertTextRowExists(t, db, &domain.Announcement{}, "title = ?", "中文演示数据已准备好")
	assertTextRowExists(t, db, &domain.SystemNotification{}, "title = ?", "中文演示环境已就绪")
	assertTextRowExists(t, db, &domain.IMConversation{}, "name = ?", "中文演示会话")
	assertTextRowExists(t, db, &domain.IMMessage{}, "content = ?", "中文演示信息流已经准备好，可以开始验收。")
}

func simulateLegacyEnglishDemoRows(t *testing.T, db *gorm.DB, aliceID int64) {
	t.Helper()
	if err := db.Model(&domain.User{}).Where("id = ?", aliceID).Updates(map[string]any{
		"nickname": "Demo Alice",
		"bio":      "Campus photographer and product tester.",
		"location": "Demo City",
	}).Error; err != nil {
		t.Fatalf("simulate legacy user: %v", err)
	}
	if err := db.Model(&domain.Category{}).Where("name = ?", "photography").Update("category_title", "Photography").Error; err != nil {
		t.Fatalf("simulate legacy category: %v", err)
	}
	if err := db.Model(&domain.Tag{}).Where("name = ?", "摄影").Update("name", "photography").Error; err != nil {
		t.Fatalf("simulate legacy tag: %v", err)
	}
	if err := db.Model(&domain.Post{}).Where("user_id = ? AND title = ?", aliceID, "图书馆清晨的第一束光").Updates(map[string]any{
		"title":   "Morning light in the library",
		"content": "A quiet set of reference images for feed layout, image preview, and comment QA.",
	}).Error; err != nil {
		t.Fatalf("simulate legacy post: %v", err)
	}
	updateOneTextRow(t, db, &domain.Comment{}, "content", "这条中文演示内容让信息流更接近真实使用场景。", "This demo post makes the feed feel alive.")
	updateOneTextRow(t, db, &domain.Notification{}, "title", "点赞了你的帖子", "liked your post")
	updateOneTextRow(t, db, &domain.GiftCardProduct{}, "name", "演示咖啡兑换卡", "Demo Coffee Card")
	updateOneTextRow(t, db, &domain.Announcement{}, "title", "中文演示数据已准备好", "Demo data is available")
	updateOneTextRow(t, db, &domain.SystemNotification{}, "title", "中文演示环境已就绪", "Demo workspace ready")
	updateOneTextRow(t, db, &domain.IMConversation{}, "name", "中文演示会话", "Demo chat")
	updateOneTextRow(t, db, &domain.IMMessage{}, "content", "中文演示信息流已经准备好，可以开始验收。", "The demo feed is ready for review.")
}

func updateOneTextRow(t *testing.T, db *gorm.DB, model any, column string, from string, to string) {
	t.Helper()
	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil {
		t.Fatalf("parse model: %v", err)
	}
	var id int64
	if err := db.Table(stmt.Schema.Table).Select("id").Where(column+" = ?", from).Limit(1).Scan(&id).Error; err != nil {
		t.Fatalf("find %s legacy row: %v", stmt.Schema.Table, err)
	}
	if id == 0 {
		t.Fatalf("missing row in %s where %s = %q", stmt.Schema.Table, column, from)
	}
	if err := db.Table(stmt.Schema.Table).Where("id = ?", id).Update(column, to).Error; err != nil {
		t.Fatalf("update %s legacy row: %v", stmt.Schema.Table, err)
	}
}

func assertTextRowExists(t *testing.T, db *gorm.DB, model any, query string, args ...any) {
	t.Helper()
	var count int64
	if err := db.Model(model).Where(query, args...).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	if count == 0 {
		t.Fatalf("%T missing row for %s %v", model, query, args)
	}
}

func openDemoSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+strings.ReplaceAll(t.Name(), "/", "_")+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(demoSeedTestModels()...); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return db
}

func demoSeedTestModels() []any {
	return []any{
		&domain.User{},
		&domain.Category{},
		&domain.Tag{},
		&domain.UserPoints{},
		&domain.PointsLog{},
		&domain.UserWallet{},
		&domain.CreatorEarnings{},
		&domain.CreatorEarningsLog{},
		&domain.Post{},
		&domain.Comment{},
		&domain.Like{},
		&domain.Collection{},
		&domain.Follow{},
		&domain.Notification{},
		&domain.PostImage{},
		&domain.PostAttachment{},
		&domain.PostVideo{},
		&domain.PostPaymentSetting{},
		&domain.PostTag{},
		&domain.GiftCardProduct{},
		&domain.GiftCardCode{},
		&domain.Announcement{},
		&domain.SystemNotification{},
		&domain.IMConversation{},
		&domain.IMConversationMember{},
		&domain.IMMessage{},
		&domain.IMMessageReceipt{},
		&domain.UserSearchHistory{},
		&domain.BrowsingHistory{},
	}
}

func demoSeedRowCounts(t *testing.T, db *gorm.DB) map[string]int64 {
	t.Helper()
	counts := map[string]int64{}
	for _, model := range demoSeedTestModels() {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			t.Fatalf("parse model: %v", err)
		}
		counts[stmt.Schema.Table] = countRows(t, db, model)
	}
	return counts
}

func assertRowCount(t *testing.T, db *gorm.DB, model any, want int64) {
	t.Helper()
	if got := countRows(t, db, model); got != want {
		t.Fatalf("%T row count = %d, want %d", model, got, want)
	}
}

func assertMinimumRows(t *testing.T, db *gorm.DB, model any, want int64) {
	t.Helper()
	if got := countRows(t, db, model); got < want {
		t.Fatalf("%T row count = %d, want at least %d", model, got, want)
	}
}

func countRows(t *testing.T, db *gorm.DB, model any) int64 {
	t.Helper()
	var count int64
	if err := db.Model(model).Count(&count).Error; err != nil {
		t.Fatalf("count %T: %v", model, err)
	}
	return count
}
