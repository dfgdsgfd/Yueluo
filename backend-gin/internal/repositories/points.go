package repositories

import (
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
)

const (
	PointsTaskComment       = "comment"
	PointsTaskClick         = "click"
	PointsTaskLike          = "like"
	PointsTaskCollect       = "collect"
	PointsTaskView          = "view"
	PointsTaskPost          = "post"
	PointsTaskSetAvatar     = "set_avatar"
	PointsTaskSetBackground = "set_background"
	PointsTaskSetSignature  = "set_signature"
	PointsTaskSetName       = "set_name"

	PointsAchievementTotalPosts       = "total_posts"
	PointsAchievementConsecutivePosts = "consecutive_posts"
	PointsAchievementTotalPoints      = "total_points"

	GiftCardCodeStatusAvailable = "available"
	GiftCardCodeStatusRedeemed  = "redeemed"
	RedemptionStatusCompleted   = "completed"

	PointsLogTypeWithdrawFromEarnings = "withdraw_from_earnings"
	PointsLogTypeTransferToEarnings   = "transfer_to_earnings"
	PointsLogTypeAdminAdd             = "admin_points_add"
	PointsLogTypeAdminDeduct          = "admin_points_deduct"
	PointsLogTypeAdminSet             = "admin_points_set"

	PointsAdjustmentAdd    = "add"
	PointsAdjustmentDeduct = "deduct"
	PointsAdjustmentSet    = "set"
)

var (
	ErrPointsInsufficient               = errors.New("points insufficient")
	ErrPointsAdjustmentInvalidOperation = errors.New("invalid points adjustment operation")
	ErrPointsAdjustmentInvalidAmount    = errors.New("invalid points adjustment amount")
	ErrPointsAdjustmentNoChange         = errors.New("points adjustment has no change")
	ErrGiftCardProductInactive          = errors.New("gift card product inactive")
	ErrGiftCardProductNotFound          = errors.New("gift card product not found")
	ErrGiftCardOutOfStock               = errors.New("gift card out of stock")
	ErrPointsTaskNotConfigured          = errors.New("points task not configured")
	ErrPointsNoAward                    = errors.New("points no award")
	ErrGiftCardImportEmpty              = errors.New("gift card import empty")
)

var hiddenUserPointsLogTypes = []string{
	PointsLogTypeWithdrawFromEarnings,
	PointsLogTypeTransferToEarnings,
}

type PointsRepository struct {
	db       *gorm.DB
	dailyCap float64
}

type AwardInput struct {
	UserID    int64
	TaskType  string
	TargetKey string
	Reason    string
}

type AwardResult struct {
	Awarded      bool
	Amount       float64
	BalanceAfter float64
	DailyAwarded float64
	Message      string
	Reason       string `json:"reason,omitempty"`
}

type PointsOverview struct {
	Points      float64
	TodayEarned float64
	DailyCap    float64
	Tasks       []PointsTaskProgress
	GiftCards   []GiftCardProductStock
	ActiveBonus *domain.UserCreatorBonus
	GeneratedAt time.Time
}

type PointsTaskProgress struct {
	Config         domain.PointsTaskConfig
	Completed      int
	AwardedPoints  float64
	RemainingCount int
	ReachedLimit   bool
}

type pointsTaskAggregate struct {
	CompletedCount int
	AwardedPoints  float64
}

type GiftCardProductStock struct {
	Product        domain.GiftCardProduct
	AvailableStock int64
	RedeemedStock  int64
}

type PointsLogBundle struct {
	Log domain.PointsLog
}

type GiftCardRedemptionBundle struct {
	Redemption domain.GiftCardRedemption
	Product    *domain.GiftCardProduct
}

type GiftCardImportResult struct {
	Imported int
	Skipped  int
	Batch    string
}

type PointsStats struct {
	TotalUsers       int64
	TotalPoints      float64
	TodayAwarded     float64
	TotalRedeemed    int64
	AvailableCards   int64
	ActiveTasks      int64
	ActiveBonusUsers int64
}

type PointsMaintenanceResult struct {
	AffectedUsers       int64
	DeletedEvents       int64
	DeletedStats        int64
	DeletedAchievements int64
	DeletedBonuses      int64
}

type AdminPointsAdjustmentResult struct {
	UserID          int64
	Operation       string
	PreviousBalance float64
	Amount          float64
	BalanceAfter    float64
}

func NewPointsRepository(db *gorm.DB, dailyCap float64) PointsRepository {
	return PointsRepository{db: db, dailyCap: positiveFloat(dailyCap)}
}

func (r PointsRepository) DailyCap() float64 {
	return positiveFloat(r.dailyCap)
}

func (r PointsRepository) Award(ctx context.Context, input AwardInput) (*AwardResult, error) {
	if input.UserID <= 0 {
		return nil, ErrPointsNoAward
	}
	taskType := normalizeTaskType(input.TaskType)
	if taskType == "" {
		return nil, ErrPointsTaskNotConfigured
	}
	targetKey := strings.TrimSpace(input.TargetKey)
	today := dateOnly(time.Now())
	var result AwardResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		config, err := pointsTaskConfigTx(ctx, tx, taskType)
		if err != nil {
			return err
		}
		if !config.IsActive || config.Points <= 0 {
			return ErrPointsNoAward
		}
		if targetKey == "" {
			if config.IsDailyTask {
				targetKey = taskType + ":daily"
			} else {
				targetKey = taskType + ":fixed"
			}
		}
		points, err := getOrCreateUserPointsTx(ctx, tx, input.UserID)
		if err != nil {
			return err
		}
		if !config.IsDailyTask {
			var existing int64
			if err := tx.Model(&domain.PointsTaskEvent{}).Where("user_id = ? AND task_type = ?", input.UserID, taskType).Count(&existing).Error; err != nil {
				return err
			}
			if existing > 0 {
				return ErrPointsNoAward
			}
		}

		stat, err := getOrCreateDailyStatTx(ctx, tx, input.UserID, taskType, today)
		if err != nil {
			return err
		}
		if config.IsDailyTask && config.DailyLimit > 0 && stat.CompletedCount >= config.DailyLimit {
			return ErrPointsNoAward
		}

		total, err := dailyPointsTotalTx(ctx, tx, input.UserID, today)
		if err != nil {
			return err
		}
		amount := config.Points
		if r.dailyCap > 0 {
			remaining := r.dailyCap - total
			if remaining <= 0 {
				return ErrPointsNoAward
			}
			if amount > remaining {
				amount = remaining
			}
		}
		amount = mathRound2(amount)
		if amount <= 0 {
			return ErrPointsNoAward
		}

		reason := strings.TrimSpace(input.Reason)
		if reason == "" {
			reason = config.Name
		}
		event := domain.PointsTaskEvent{UserID: input.UserID, TaskType: taskType, TargetKey: targetKey, EventDate: today, Points: amount, Reason: &reason}
		eventCreate := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "task_type"}, {Name: "target_key"}, {Name: "event_date"}},
			DoNothing: true,
		}).Create(&event)
		if eventCreate.Error != nil {
			return eventCreate.Error
		}
		if eventCreate.RowsAffected == 0 {
			return ErrPointsNoAward
		}

		newBalance := mathRound2(points.Points + amount)
		now := time.Now()
		if err := tx.Model(&domain.UserPoints{}).Where("user_id = ?", input.UserID).Updates(map[string]any{"points": newBalance, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Create(&domain.PointsLog{UserID: input.UserID, Amount: amount, BalanceAfter: newBalance, Type: "task_" + taskType, Reason: &reason}).Error; err != nil {
			return err
		}
		if err := tx.Model(&domain.PointsDailyStat{}).Where("id = ?", stat.ID).Updates(map[string]any{
			"completed_count": gorm.Expr("completed_count + ?", 1),
			"awarded_points":  gorm.Expr("awarded_points + ?", amount),
			"updated_at":      now,
		}).Error; err != nil {
			return err
		}
		result = AwardResult{Awarded: true, Amount: amount, BalanceAfter: newBalance, DailyAwarded: mathRound2(total + amount), Message: "积分已到账", Reason: reason}
		return nil
	})
	if errors.Is(err, ErrPointsNoAward) {
		return &AwardResult{Awarded: false, Message: "未触发积分奖励"}, nil
	}
	return &result, err
}

func (r PointsRepository) AwardBestEffort(ctx context.Context, input AwardInput) *AwardResult {
	result, err := r.Award(ctx, input)
	if err != nil {
		return &AwardResult{Awarded: false, Message: err.Error()}
	}
	return result
}

func (r PointsRepository) Overview(ctx context.Context, userID int64) (*PointsOverview, error) {
	points, err := r.GetOrCreateUserPoints(ctx, userID)
	if err != nil {
		return nil, err
	}
	today := dateOnly(time.Now())
	todayEarned, err := dailyPointsTotal(ctx, r.db, userID, today)
	if err != nil {
		return nil, err
	}
	configs, err := r.TaskConfigs(ctx, true)
	if err != nil {
		return nil, err
	}
	stats, err := r.dailyStatsByTask(ctx, userID, today)
	if err != nil {
		return nil, err
	}
	var aggregates map[string]pointsTaskAggregate
	if hasFixedTaskConfig(configs) {
		aggregates, err = r.taskEventAggregatesByTask(ctx, userID)
		if err != nil {
			return nil, err
		}
	}
	progress := make([]PointsTaskProgress, 0, len(configs))
	for _, config := range configs {
		if !config.IsDailyTask {
			aggregate := aggregates[config.TaskType]
			completed := aggregate.CompletedCount
			reached := completed > 0
			remaining := 1
			if reached {
				remaining = 0
			}
			progress = append(progress, PointsTaskProgress{Config: config, Completed: completed, AwardedPoints: mathRound2(aggregate.AwardedPoints), RemainingCount: remaining, ReachedLimit: reached})
			continue
		}
		stat := stats[config.TaskType]
		remaining := 0
		reached := false
		if config.DailyLimit > 0 {
			remaining = max(config.DailyLimit-stat.CompletedCount, 0)
			reached = remaining == 0
		}
		progress = append(progress, PointsTaskProgress{Config: config, Completed: stat.CompletedCount, AwardedPoints: mathRound2(stat.AwardedPoints), RemainingCount: remaining, ReachedLimit: reached})
	}
	giftCards, err := r.GiftCardProducts(ctx, true)
	if err != nil {
		return nil, err
	}
	activeBonus, err := r.ActiveCreatorBonus(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &PointsOverview{Points: points.Points, TodayEarned: todayEarned, DailyCap: r.dailyCap, Tasks: progress, GiftCards: giftCards, ActiveBonus: activeBonus, GeneratedAt: time.Now()}, nil
}

func (r PointsRepository) GetOrCreateUserPoints(ctx context.Context, userID int64) (*domain.UserPoints, error) {
	var points domain.UserPoints
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&points).Error
	if err == nil {
		return &points, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	points = domain.UserPoints{UserID: userID, Points: 0}
	if err := r.db.WithContext(ctx).Create(&points).Error; err != nil {
		return nil, err
	}
	return &points, nil
}

func (r PointsRepository) AdminAdjustBalance(ctx context.Context, userID int64, operation string, amount float64, reason string) (*AdminPointsAdjustmentResult, error) {
	operation = strings.ToLower(strings.TrimSpace(operation))
	amount = mathRound2(amount)
	if userID <= 0 {
		return nil, gorm.ErrRecordNotFound
	}
	switch operation {
	case PointsAdjustmentAdd, PointsAdjustmentDeduct:
		if amount <= 0 {
			return nil, ErrPointsAdjustmentInvalidAmount
		}
	case PointsAdjustmentSet:
		if amount < 0 {
			return nil, ErrPointsAdjustmentInvalidAmount
		}
	default:
		return nil, ErrPointsAdjustmentInvalidOperation
	}
	if amount > 1_000_000_000 {
		return nil, ErrPointsAdjustmentInvalidAmount
	}

	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "admin_manual_points_adjustment"
	}

	var result AdminPointsAdjustmentResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user domain.User
		if err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("id = ?", userID).First(&user).Error; err != nil {
			return err
		}
		points, err := getOrCreateUserPointsTx(ctx, tx, userID)
		if err != nil {
			return err
		}

		previous := mathRound2(points.Points)
		balanceAfter := previous
		delta := amount
		logType := PointsLogTypeAdminAdd
		switch operation {
		case PointsAdjustmentAdd:
			balanceAfter = mathRound2(previous + amount)
		case PointsAdjustmentDeduct:
			if previous < amount {
				return ErrPointsInsufficient
			}
			delta = -amount
			balanceAfter = mathRound2(previous - amount)
			logType = PointsLogTypeAdminDeduct
		case PointsAdjustmentSet:
			balanceAfter = amount
			delta = mathRound2(balanceAfter - previous)
			logType = PointsLogTypeAdminSet
		}
		if delta == 0 {
			return ErrPointsAdjustmentNoChange
		}

		now := time.Now()
		if err := tx.Model(&domain.UserPoints{}).
			Where("user_id = ?", userID).
			Updates(map[string]any{"points": balanceAfter, "updated_at": now}).Error; err != nil {
			return err
		}
		if err := tx.Create(&domain.PointsLog{
			UserID:       userID,
			Amount:       delta,
			BalanceAfter: balanceAfter,
			Type:         logType,
			Reason:       &reason,
			CreatedAt:    now,
		}).Error; err != nil {
			return err
		}
		result = AdminPointsAdjustmentResult{
			UserID:          userID,
			Operation:       operation,
			PreviousBalance: previous,
			Amount:          delta,
			BalanceAfter:    balanceAfter,
		}
		return nil
	})
	return &result, err
}
