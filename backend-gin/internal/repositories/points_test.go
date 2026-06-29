package repositories

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func TestHiddenUserPointsLogTypesExcludeEarningsTransfers(t *testing.T) {
	hidden := map[string]bool{}
	for _, logType := range hiddenUserPointsLogTypes {
		hidden[logType] = true
	}

	for _, logType := range []string{PointsLogTypeWithdrawFromEarnings, PointsLogTypeTransferToEarnings} {
		if !hidden[logType] {
			t.Fatalf("hiddenUserPointsLogTypes missing %q", logType)
		}
	}
}

func TestAdminAdjustBalanceSupportsAddDeductAndSet(t *testing.T) {
	db := newPointsTestDB(t, "admin-adjust")
	user := domain.User{UserID: "user-1", Nickname: "Tester", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	repo := NewPointsRepository(db, 50)
	ctx := context.Background()

	added, err := repo.AdminAdjustBalance(ctx, user.ID, PointsAdjustmentAdd, 25, "manual add")
	if err != nil {
		t.Fatalf("add points: %v", err)
	}
	if added.PreviousBalance != 0 || added.Amount != 25 || added.BalanceAfter != 25 {
		t.Fatalf("unexpected add result: %#v", added)
	}

	deducted, err := repo.AdminAdjustBalance(ctx, user.ID, PointsAdjustmentDeduct, 5, "manual deduct")
	if err != nil {
		t.Fatalf("deduct points: %v", err)
	}
	if deducted.PreviousBalance != 25 || deducted.Amount != -5 || deducted.BalanceAfter != 20 {
		t.Fatalf("unexpected deduct result: %#v", deducted)
	}

	set, err := repo.AdminAdjustBalance(ctx, user.ID, PointsAdjustmentSet, 7.5, "manual set")
	if err != nil {
		t.Fatalf("set points: %v", err)
	}
	if set.PreviousBalance != 20 || set.Amount != -12.5 || set.BalanceAfter != 7.5 {
		t.Fatalf("unexpected set result: %#v", set)
	}

	var points domain.UserPoints
	if err := db.Where("user_id = ?", user.ID).First(&points).Error; err != nil {
		t.Fatalf("load points: %v", err)
	}
	if points.Points != 7.5 {
		t.Fatalf("points balance = %v, want 7.5", points.Points)
	}
	var logs []domain.PointsLog
	if err := db.Where("user_id = ?", user.ID).Order("id ASC").Find(&logs).Error; err != nil {
		t.Fatalf("load logs: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("points logs = %d, want 3", len(logs))
	}
	if logs[0].Type != PointsLogTypeAdminAdd || logs[1].Type != PointsLogTypeAdminDeduct || logs[2].Type != PointsLogTypeAdminSet {
		t.Fatalf("unexpected log types: %#v", logs)
	}
}

func TestAdminAdjustBalanceRejectsInsufficientAndNoChange(t *testing.T) {
	db := newPointsTestDB(t, "admin-adjust-errors")
	user := domain.User{UserID: "user-2", Nickname: "Tester", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	repo := NewPointsRepository(db, 50)
	ctx := context.Background()
	if _, err := repo.AdminAdjustBalance(ctx, user.ID, PointsAdjustmentAdd, 3, "seed"); err != nil {
		t.Fatalf("seed points: %v", err)
	}
	if _, err := repo.AdminAdjustBalance(ctx, user.ID, PointsAdjustmentDeduct, 4, "too much"); !errors.Is(err, ErrPointsInsufficient) {
		t.Fatalf("deduct error = %v, want insufficient", err)
	}
	if _, err := repo.AdminAdjustBalance(ctx, user.ID, PointsAdjustmentSet, 3, "same"); !errors.Is(err, ErrPointsAdjustmentNoChange) {
		t.Fatalf("set error = %v, want no change", err)
	}
	var logs int64
	if err := db.Model(&domain.PointsLog{}).Where("user_id = ?", user.ID).Count(&logs).Error; err != nil {
		t.Fatalf("count logs: %v", err)
	}
	if logs != 1 {
		t.Fatalf("points logs = %d, want 1", logs)
	}
}

func TestAwardDuplicateDailyTargetIsIdempotent(t *testing.T) {
	db := newPointsTestDB(t, "award-daily-idempotent")
	repo := NewPointsRepository(db, 50)
	ctx := context.Background()
	input := AwardInput{UserID: 2136, TaskType: PointsTaskClick, TargetKey: "click:799", Reason: "click_reward"}

	first, err := repo.Award(ctx, input)
	if err != nil {
		t.Fatalf("first award: %v", err)
	}
	if first == nil || !first.Awarded || first.Amount != 1 {
		t.Fatalf("first award result = %#v, want awarded 1", first)
	}

	second, err := repo.Award(ctx, input)
	if err != nil {
		t.Fatalf("second award: %v", err)
	}
	if second == nil || second.Awarded {
		t.Fatalf("second award result = %#v, want no award", second)
	}

	var events int64
	if err := db.Model(&domain.PointsTaskEvent{}).Where("user_id = ? AND task_type = ? AND target_key = ?", input.UserID, input.TaskType, input.TargetKey).Count(&events).Error; err != nil {
		t.Fatalf("count events: %v", err)
	}
	if events != 1 {
		t.Fatalf("points task events = %d, want 1", events)
	}
	var logs int64
	if err := db.Model(&domain.PointsLog{}).Where("user_id = ? AND type = ?", input.UserID, "task_"+input.TaskType).Count(&logs).Error; err != nil {
		t.Fatalf("count logs: %v", err)
	}
	if logs != 1 {
		t.Fatalf("points logs = %d, want 1", logs)
	}
	var points domain.UserPoints
	if err := db.Where("user_id = ?", input.UserID).First(&points).Error; err != nil {
		t.Fatalf("load points: %v", err)
	}
	if points.Points != 1 {
		t.Fatalf("points balance = %v, want 1", points.Points)
	}
}

func TestRedeemGiftCardCreatesAtomicNotificationAndUserHistory(t *testing.T) {
	db := newGiftCardTestDB(t, "gift-card-atomic", true)
	product, code := seedGiftCardRedemption(t, db, 101)
	repo := NewPointsRepository(db, 50)

	bundle, err := repo.RedeemGiftCard(context.Background(), 101, product.ID)
	if err != nil {
		t.Fatalf("redeem gift card: %v", err)
	}
	if bundle.Redemption.CodeSnapshot != code.Code {
		t.Fatalf("code snapshot = %q, want %q", bundle.Redemption.CodeSnapshot, code.Code)
	}

	var notification domain.Notification
	if err := db.Where("user_id = ? AND type = ? AND target_id = ?", 101, NotificationTypeGiftCardRedeemed, bundle.Redemption.ID).First(&notification).Error; err != nil {
		t.Fatalf("load gift card notification: %v", err)
	}
	if notification.Title != "notification.giftCardRedeemed.title" {
		t.Fatalf("notification title = %q", notification.Title)
	}
	if notification.SenderID != 101 {
		t.Fatalf("notification sender_id = %d, want current user id", notification.SenderID)
	}

	total, rows, err := repo.Redemptions(context.Background(), ptrInt64(101), 1, 8)
	if err != nil {
		t.Fatalf("load user redemptions: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].Redemption.CodeSnapshot != code.Code {
		t.Fatalf("unexpected redemption history: total=%d rows=%#v", total, rows)
	}
	otherTotal, otherRows, err := repo.Redemptions(context.Background(), ptrInt64(202), 1, 8)
	if err != nil {
		t.Fatalf("load other user redemptions: %v", err)
	}
	if otherTotal != 0 || len(otherRows) != 0 {
		t.Fatalf("other user can see redemption: total=%d rows=%#v", otherTotal, otherRows)
	}
}

func TestRedeemGiftCardKeepsRedemptionWhenNotificationCannotBeCreated(t *testing.T) {
	db := newGiftCardTestDB(t, "gift-card-notification-rollback", false)
	product, code := seedGiftCardRedemption(t, db, 303)
	repo := NewPointsRepository(db, 50)

	bundle, err := repo.RedeemGiftCard(context.Background(), 303, product.ID)
	if err != nil {
		t.Fatalf("redeem gift card without notifications table: %v", err)
	}
	if bundle.Redemption.ID == 0 || bundle.Redemption.CodeSnapshot != code.Code {
		t.Fatalf("unexpected redemption bundle: %#v", bundle)
	}

	var points domain.UserPoints
	if err := db.Where("user_id = ?", 303).First(&points).Error; err != nil {
		t.Fatalf("load points after redemption: %v", err)
	}
	if points.Points != 70 {
		t.Fatalf("points after redemption = %v, want 70", points.Points)
	}
	var persistedCode domain.GiftCardCode
	if err := db.First(&persistedCode, code.ID).Error; err != nil {
		t.Fatalf("load code after redemption: %v", err)
	}
	if persistedCode.Status != GiftCardCodeStatusRedeemed || persistedCode.RedemptionID == nil || *persistedCode.RedemptionID != bundle.Redemption.ID || persistedCode.UserID == nil || *persistedCode.UserID != 303 {
		t.Fatalf("code not marked redeemed: %#v", persistedCode)
	}
	var redemptionCount int64
	if err := db.Model(&domain.GiftCardRedemption{}).Count(&redemptionCount).Error; err != nil {
		t.Fatalf("count redemptions: %v", err)
	}
	if redemptionCount != 1 {
		t.Fatalf("redemption count = %d, want 1", redemptionCount)
	}
}

func TestGiftCardNotificationDetailsEnforceRedemptionOwner(t *testing.T) {
	db := newGiftCardTestDB(t, "gift-card-notification-owner", true)
	product, _ := seedGiftCardRedemption(t, db, 404)
	bundle, err := NewPointsRepository(db, 50).RedeemGiftCard(context.Background(), 404, product.ID)
	if err != nil {
		t.Fatalf("redeem gift card: %v", err)
	}

	repo := NewNotificationsRepository(db)
	details, err := repo.giftCardRedemptionDetails(context.Background(), []int64{bundle.Redemption.ID}, map[int64]int64{bundle.Redemption.ID: 404})
	if err != nil {
		t.Fatalf("load owned detail: %v", err)
	}
	if details[bundle.Redemption.ID] == nil || details[bundle.Redemption.ID].Code != bundle.Redemption.CodeSnapshot {
		t.Fatalf("owned detail missing code: %#v", details)
	}

	foreignDetails, err := repo.giftCardRedemptionDetails(context.Background(), []int64{bundle.Redemption.ID}, map[int64]int64{bundle.Redemption.ID: 405})
	if err != nil {
		t.Fatalf("load foreign detail: %v", err)
	}
	if foreignDetails[bundle.Redemption.ID] != nil {
		t.Fatalf("foreign user received gift card detail: %#v", foreignDetails)
	}
}

func newPointsTestDB(t *testing.T, name string) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", name)), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.UserPoints{}, &domain.PointsLog{}, &domain.PointsTaskConfig{}, &domain.PointsTaskEvent{}, &domain.PointsDailyStat{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	return db
}

func newGiftCardTestDB(t *testing.T, name string, migrateNotifications bool) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", name)), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	models := []any{
		&domain.UserPoints{},
		&domain.PointsLog{},
		&domain.GiftCardProduct{},
		&domain.GiftCardCode{},
		&domain.GiftCardRedemption{},
	}
	if migrateNotifications {
		models = append(models, &domain.Notification{})
	}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate gift card sqlite: %v", err)
	}
	return db
}

func seedGiftCardRedemption(t *testing.T, db *gorm.DB, userID int64) (domain.GiftCardProduct, domain.GiftCardCode) {
	t.Helper()
	points := domain.UserPoints{UserID: userID, Points: 100}
	if err := db.Create(&points).Error; err != nil {
		t.Fatalf("create user points: %v", err)
	}
	faceValue := "10 USD"
	product := domain.GiftCardProduct{Name: "Test Gift Card", FaceValue: &faceValue, PointsRequired: 30, IsActive: true}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create gift card product: %v", err)
	}
	code := domain.GiftCardCode{ProductID: product.ID, Code: fmt.Sprintf("CARD-%d-SECRET", userID), Status: GiftCardCodeStatusAvailable}
	if err := db.Create(&code).Error; err != nil {
		t.Fatalf("create gift card code: %v", err)
	}
	return product, code
}

func ptrInt64(value int64) *int64 {
	return &value
}

func TestDefaultPointsTaskConfigsIncludeProfileFixedTasks(t *testing.T) {
	configs := defaultPointsTaskConfigs()
	byType := map[string]bool{}
	for _, config := range configs {
		byType[config.TaskType] = true
	}

	for _, taskType := range []string{PointsTaskSetAvatar, PointsTaskSetBackground, PointsTaskSetSignature, PointsTaskSetName} {
		if !byType[taskType] {
			t.Fatalf("defaultPointsTaskConfigs() missing %q", taskType)
		}
	}

	for _, config := range configs {
		switch config.TaskType {
		case PointsTaskSetAvatar, PointsTaskSetBackground, PointsTaskSetSignature, PointsTaskSetName:
			if config.IsDailyTask {
				t.Fatalf("%q should be a fixed task", config.TaskType)
			}
			if config.DailyLimit != 1 {
				t.Fatalf("%q DailyLimit = %d, want 1", config.TaskType, config.DailyLimit)
			}
			if config.Points <= 0 {
				t.Fatalf("%q Points = %v, want positive", config.TaskType, config.Points)
			}
		}
	}
}

func TestNormalizeTaskTypeProfileAliases(t *testing.T) {
	tests := map[string]string{
		"avatar":     PointsTaskSetAvatar,
		"设置头像":       PointsTaskSetAvatar,
		"background": PointsTaskSetBackground,
		"设置背景":       PointsTaskSetBackground,
		"bio":        PointsTaskSetSignature,
		"设置签名":       PointsTaskSetSignature,
		"nickname":   PointsTaskSetName,
		"设置名称":       PointsTaskSetName,
	}

	for input, want := range tests {
		if got := normalizeTaskType(input); got != want {
			t.Fatalf("normalizeTaskType(%q) = %q, want %q", input, got, want)
		}
	}
}
