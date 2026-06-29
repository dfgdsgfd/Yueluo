package repositories

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func TestQualityRewardLabel(t *testing.T) {
	good := "\u7b14\u8bb0\u8d28\u91cf\u5956\u52b1: \u4f18\u8d28\u7b14\u8bb0"
	blank := "\u7b14\u8bb0\u8d28\u91cf\u5956\u52b1:   "
	legacyOther := "\u7cfb\u7edf\u5956\u52b1: \u4f18\u8d28\u7b14\u8bb0"

	tests := []struct {
		name   string
		reason *string
		want   string
	}{
		{name: "extracts label from legacy chinese reason", reason: &good, want: "\u4f18\u8d28\u7b14\u8bb0"},
		{name: "nil falls back", reason: nil, want: qualityRewardFallback},
		{name: "blank label falls back", reason: &blank, want: qualityRewardFallback},
		{name: "unmatched reason falls back", reason: &legacyOther, want: qualityRewardFallback},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := qualityRewardLabel(tt.reason); got != tt.want {
				t.Fatalf("qualityRewardLabel() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPurchaseContentFinalizesRemoteMoonCoinPurchaseWithoutLocalWallet(t *testing.T) {
	db := newBalanceTestDB(t)
	seedBalancePaidPost(t, db, 6)

	result, err := NewBalanceRepository(db).PurchaseContent(context.Background(), PurchaseContentInput{
		UserID:          1,
		PostID:          100,
		PaymentMethod:   "balance",
		BalanceAfter:    4,
		PlatformFeeRate: 0.2,
	})
	if err != nil {
		t.Fatalf("PurchaseContent() error = %v", err)
	}
	if result.Already {
		t.Fatal("PurchaseContent() returned Already for a new purchase")
	}
	assertFloat(t, result.PaidAmount, 6)
	assertFloat(t, result.BalanceAfter, 4)
	assertFloat(t, result.PlatformFee, 1.2)
	assertFloat(t, result.AuthorEarnings, 4.8)

	var earnings domain.CreatorEarnings
	if err := db.Where("user_id = ?", 2).First(&earnings).Error; err != nil {
		t.Fatal(err)
	}
	assertFloat(t, earnings.Balance, 4.8)
	assertFloat(t, earnings.TotalEarnings, 4.8)

	var purchase domain.UserPurchasedContent
	if err := db.Where("user_id = ? AND post_id = ?", 1, 100).First(&purchase).Error; err != nil {
		t.Fatal(err)
	}
	if purchase.PaymentMethod != "balance" {
		t.Fatalf("purchase.PaymentMethod = %q, want balance", purchase.PaymentMethod)
	}
	assertFloat(t, purchase.PaidAmount, 6)
	var earningLog domain.CreatorEarningsLog
	if err := db.Where("user_id = ? AND type = ?", 2, "content_sale").First(&earningLog).Error; err != nil {
		t.Fatal(err)
	}
	if earningLog.SourceID == nil || *earningLog.SourceID != 100 || earningLog.BuyerID == nil || *earningLog.BuyerID != 1 {
		t.Fatalf("creator earning structured fields = %#v", earningLog)
	}
	assertFloat(t, earningLog.PlatformFee, 1.2)
}

func TestPurchaseContentWithPointsCreatesBuyerAndAuthorStructuredEntries(t *testing.T) {
	db := newBalanceTestDB(t)
	seedPointsPaidPost(t, db, 6)

	result, err := NewBalanceRepository(db).PurchaseContent(context.Background(), PurchaseContentInput{
		UserID:        1,
		PostID:        100,
		PaymentMethod: "points",
	})
	if err != nil {
		t.Fatalf("PurchaseContent() error = %v", err)
	}
	var logs []domain.PointsLog
	if err := db.Where("purchase_id = ?", result.PurchaseID).Order("amount ASC").Find(&logs).Error; err != nil {
		t.Fatal(err)
	}
	if len(logs) != 2 {
		t.Fatalf("points logs = %#v, want buyer and author entries", logs)
	}
	buyer, author := logs[0], logs[1]
	if buyer.Type != "paid_content_purchase" || buyer.EntryRole != "buyer_debit" || buyer.CounterpartyUserID == nil || *buyer.CounterpartyUserID != 2 {
		t.Fatalf("buyer points log = %#v", buyer)
	}
	if author.Type != "paid_content_sale" || author.EntryRole != "author_credit" || author.CounterpartyUserID == nil || *author.CounterpartyUserID != 1 {
		t.Fatalf("author points log = %#v", author)
	}
	for _, log := range logs {
		if log.PostID == nil || *log.PostID != 100 || log.PurchaseID == nil || *log.PurchaseID != result.PurchaseID || log.PaymentMethod != "points" {
			t.Fatalf("structured points log = %#v", log)
		}
	}
}

func TestReservePurchaseIntentRejectsDuplicateProcessing(t *testing.T) {
	db := newBalanceTestDB(t)
	seedBalancePaidPost(t, db, 6)
	repo := NewBalanceRepository(db)

	reservation, err := repo.ReservePurchaseIntent(context.Background(), PurchaseContentInput{
		UserID:        1,
		PostID:        100,
		PaymentMethod: "balance",
	}, 6, 6)
	if err != nil {
		t.Fatalf("ReservePurchaseIntent() error = %v", err)
	}
	if !reservation.Acquired || reservation.Completed {
		t.Fatalf("reservation = %#v, want acquired processing intent", reservation)
	}

	_, err = repo.ReservePurchaseIntent(context.Background(), PurchaseContentInput{
		UserID:        1,
		PostID:        100,
		PaymentMethod: "balance",
	}, 6, 6)
	if !errors.Is(err, ErrPurchaseInProgress) {
		t.Fatalf("second ReservePurchaseIntent() error = %v, want ErrPurchaseInProgress", err)
	}
}

func TestPostPurchaseUsersReturnsBuyersNewestFirst(t *testing.T) {
	db := newBalanceTestDB(t)
	now := time.Now()
	avatar := "/uploads/avatar.jpg"
	records := []any{
		&domain.User{ID: 1, UserID: "buyer-old", Nickname: "Old Buyer", CreatedAt: now},
		&domain.User{ID: 2, UserID: "author", Nickname: "Author", CreatedAt: now},
		&domain.User{ID: 3, UserID: "buyer-new", Nickname: "New Buyer", Avatar: &avatar, CreatedAt: now},
		&domain.User{ID: 4, UserID: "other-buyer", Nickname: "Other Buyer", CreatedAt: now},
		&domain.Post{ID: 100, UserID: 2, Title: "Paid Post", Content: "body", Type: 1, CreatedAt: now, Visibility: "public"},
		&domain.Post{ID: 101, UserID: 2, Title: "Other Post", Content: "body", Type: 1, CreatedAt: now, Visibility: "public"},
		&domain.UserPurchasedContent{ID: 1, UserID: 1, PostID: 100, AuthorID: 2, Price: 6, PaidAmount: 6, PaymentMethod: "points", PurchasedAt: now.Add(-time.Hour), CreatedAt: now.Add(-time.Hour)},
		&domain.UserPurchasedContent{ID: 2, UserID: 3, PostID: 100, AuthorID: 2, Price: 8, PaidAmount: 7, DiscountRate: 0.875, PaymentMethod: "balance", PurchasedAt: now, CreatedAt: now},
		&domain.UserPurchasedContent{ID: 3, UserID: 4, PostID: 101, AuthorID: 2, Price: 9, PaidAmount: 9, PaymentMethod: "balance", PurchasedAt: now.Add(time.Hour), CreatedAt: now.Add(time.Hour)},
	}
	for _, record := range records {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}

	total, purchases, err := NewBalanceRepository(db).PostPurchaseUsers(context.Background(), 100, 1, 20)
	if err != nil {
		t.Fatalf("PostPurchaseUsers() error = %v", err)
	}
	if total != 2 || len(purchases) != 2 {
		t.Fatalf("total=%d len=%d, want 2", total, len(purchases))
	}
	if purchases[0].Purchase.UserID != 3 || purchases[0].Buyer == nil || purchases[0].Buyer.Nickname != "New Buyer" {
		t.Fatalf("first purchase = %#v", purchases[0])
	}
	if purchases[1].Purchase.UserID != 1 || purchases[1].Buyer == nil || purchases[1].Buyer.Nickname != "Old Buyer" {
		t.Fatalf("second purchase = %#v", purchases[1])
	}
	assertFloat(t, purchases[0].Purchase.PaidAmount, 7)
}

func newBalanceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", name)), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.UserWallet{},
		&domain.UserPoints{},
		&domain.PointsLog{},
		&domain.Post{},
		&domain.PostPaymentSetting{},
		&domain.UserPurchasedContent{},
		&domain.CreatorEarnings{},
		&domain.CreatorEarningsLog{},
		&domain.ContentPurchaseIntent{},
	); err != nil {
		t.Fatal(err)
	}
	return db
}

func seedPointsPaidPost(t *testing.T, db *gorm.DB, price float64) {
	t.Helper()
	now := time.Now()
	records := []any{
		&domain.User{ID: 1, UserID: "buyer", Nickname: "Buyer", CreatedAt: now},
		&domain.User{ID: 2, UserID: "author", Nickname: "Author", CreatedAt: now},
		&domain.UserPoints{ID: 1, UserID: 1, Points: 10, CreatedAt: now},
		&domain.UserPoints{ID: 2, UserID: 2, Points: 1, CreatedAt: now},
		&domain.Post{ID: 100, UserID: 2, Title: "Paid Post", Content: "body", Type: 1, CreatedAt: now, Visibility: "public"},
		&domain.PostPaymentSetting{ID: 10, PostID: 100, Enabled: true, PaymentType: "single", PaymentMethod: "points", Price: price, CreatedAt: now},
	}
	for _, record := range records {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func seedBalancePaidPost(t *testing.T, db *gorm.DB, price float64) {
	t.Helper()
	now := time.Now()
	records := []any{
		&domain.User{ID: 1, UserID: "buyer", Nickname: "Buyer", CreatedAt: now},
		&domain.User{ID: 2, UserID: "author", Nickname: "Author", CreatedAt: now},
		&domain.Post{ID: 100, UserID: 2, Title: "Paid Post", Content: "body", Type: 1, CreatedAt: now, Visibility: "public"},
		&domain.PostPaymentSetting{ID: 10, PostID: 100, Enabled: true, PaymentType: "single", PaymentMethod: "balance", Price: price, CreatedAt: now},
	}
	for _, record := range records {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func assertFloat(t *testing.T, got, want float64) {
	t.Helper()
	if mathRound2(got) != mathRound2(want) {
		t.Fatalf("got %.4f, want %.4f", got, want)
	}
}
