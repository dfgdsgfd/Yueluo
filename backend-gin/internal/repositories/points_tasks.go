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

func (r PointsRepository) Logs(ctx context.Context, userID int64, page, limit int) (int64, []PointsLogBundle, error) {
	query := r.db.WithContext(ctx).
		Model(&domain.PointsLog{}).
		Where("user_id = ?", userID).
		Where("type NOT IN ?", hiddenUserPointsLogTypes)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var logs []domain.PointsLog
	if err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&logs).Error; err != nil {
		return 0, nil, err
	}
	out := make([]PointsLogBundle, 0, len(logs))
	for _, log := range logs {
		out = append(out, PointsLogBundle{Log: log})
	}
	return total, out, nil
}

func (r PointsRepository) TaskConfigs(ctx context.Context, activeOnly bool) ([]domain.PointsTaskConfig, error) {
	query := r.db.WithContext(ctx).Model(&domain.PointsTaskConfig{})
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	var rows []domain.PointsTaskConfig
	if err := query.Order("sort_order ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 && !activeOnly {
		return defaultPointsTaskConfigs(), nil
	}
	if len(rows) == 0 && activeOnly {
		for _, row := range defaultPointsTaskConfigs() {
			if row.IsActive {
				rows = append(rows, row)
			}
		}
	}
	return rows, nil
}

func (r PointsRepository) SaveTaskConfig(ctx context.Context, id int64, input domain.PointsTaskConfig) (*domain.PointsTaskConfig, error) {
	input.TaskType = normalizeTaskType(input.TaskType)
	input.Points = mathRound2(input.Points)
	if strings.TrimSpace(input.Name) == "" {
		input.Name = defaultTaskName(input.TaskType)
	}
	now := time.Now()
	if id <= 0 {
		input.CreatedAt = now
		input.UpdatedAt = &now
		if err := r.db.WithContext(ctx).Create(&input).Error; err != nil {
			return nil, err
		}
		return &input, nil
	}
	updates := map[string]any{
		"task_type":     input.TaskType,
		"name":          input.Name,
		"description":   input.Description,
		"points":        input.Points,
		"daily_limit":   input.DailyLimit,
		"is_daily_task": input.IsDailyTask,
		"is_active":     input.IsActive,
		"sort_order":    input.SortOrder,
		"updated_at":    now,
	}
	if err := r.db.WithContext(ctx).Model(&domain.PointsTaskConfig{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	input.ID = id
	return &input, nil
}

func (r PointsRepository) DeleteTaskConfig(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&domain.PointsTaskConfig{}, id).Error
}

func (r PointsRepository) ClearAllBalances(ctx context.Context, reason string) (*PointsMaintenanceResult, error) {
	result := PointsMaintenanceResult{}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "管理员清空全部积分"
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rows []domain.UserPoints
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("points <> ?", 0).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		now := time.Now()
		logs := make([]domain.PointsLog, 0, len(rows))
		ids := make([]int64, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
			logs = append(logs, domain.PointsLog{
				UserID:       row.UserID,
				Amount:       -mathRound2(row.Points),
				BalanceAfter: 0,
				Type:         "admin_clear_points",
				Reason:       &reason,
				CreatedAt:    now,
			})
		}
		if err := tx.Create(&logs).Error; err != nil {
			return err
		}
		res := tx.Model(&domain.UserPoints{}).Where("id IN ?", ids).Updates(map[string]any{"points": 0, "updated_at": now})
		if res.Error != nil {
			return res.Error
		}
		result.AffectedUsers = res.RowsAffected
		return nil
	})
	return &result, err
}

func (r PointsRepository) ResetTaskProgress(ctx context.Context) (*PointsMaintenanceResult, error) {
	result := PointsMaintenanceResult{}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("1 = 1").Delete(&domain.PointsTaskEvent{})
		if res.Error != nil {
			return res.Error
		}
		result.DeletedEvents = res.RowsAffected

		res = tx.Where("1 = 1").Delete(&domain.PointsDailyStat{})
		if res.Error != nil {
			return res.Error
		}
		result.DeletedStats = res.RowsAffected

		res = tx.Where("1 = 1").Delete(&domain.UserAchievementReward{})
		if res.Error != nil {
			return res.Error
		}
		result.DeletedAchievements = res.RowsAffected

		res = tx.Where("1 = 1").Delete(&domain.UserCreatorBonus{})
		if res.Error != nil {
			return res.Error
		}
		result.DeletedBonuses = res.RowsAffected
		return nil
	})
	return &result, err
}

func (r PointsRepository) AchievementRules(ctx context.Context, activeOnly bool) ([]domain.PointsAchievementRule, error) {
	query := r.db.WithContext(ctx).Model(&domain.PointsAchievementRule{})
	if activeOnly {
		query = query.Where("is_active = ?", true)
	}
	var rows []domain.PointsAchievementRule
	err := query.Order("id ASC").Find(&rows).Error
	return rows, err
}

func (r PointsRepository) SaveAchievementRule(ctx context.Context, id int64, input domain.PointsAchievementRule) (*domain.PointsAchievementRule, error) {
	input.TriggerType = normalizeAchievementTrigger(input.TriggerType)
	input.PointsReward = mathRound2(input.PointsReward)
	input.CreatorBonusPercent = mathRound2(input.CreatorBonusPercent)
	now := time.Now()
	if id <= 0 {
		input.CreatedAt = now
		input.UpdatedAt = &now
		if err := r.db.WithContext(ctx).Create(&input).Error; err != nil {
			return nil, err
		}
		return &input, nil
	}
	updates := map[string]any{
		"name":                  input.Name,
		"trigger_type":          input.TriggerType,
		"threshold_value":       input.ThresholdValue,
		"points_reward":         input.PointsReward,
		"creator_bonus_percent": input.CreatorBonusPercent,
		"bonus_days":            input.BonusDays,
		"description":           input.Description,
		"is_active":             input.IsActive,
		"updated_at":            now,
	}
	if err := r.db.WithContext(ctx).Model(&domain.PointsAchievementRule{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	input.ID = id
	return &input, nil
}

func (r PointsRepository) DeleteAchievementRule(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&domain.PointsAchievementRule{}, id).Error
}

func (r PointsRepository) EvaluateAchievements(ctx context.Context, userID int64) ([]AwardResult, error) {
	if userID <= 0 {
		return nil, nil
	}
	rules, err := r.AchievementRules(ctx, true)
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, nil
	}
	awards := make([]AwardResult, 0)
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, rule := range rules {
			reached, err := achievementReachedTx(ctx, tx, userID, rule)
			if err != nil {
				return err
			}
			if !reached {
				continue
			}
			var existing int64
			if err := tx.Model(&domain.UserAchievementReward{}).Where("user_id = ? AND rule_id = ?", userID, rule.ID).Count(&existing).Error; err != nil {
				return err
			}
			if existing > 0 {
				continue
			}
			pointsAwarded := mathRound2(rule.PointsReward)
			var newBalance float64
			if pointsAwarded > 0 {
				points, err := getOrCreateUserPointsTx(ctx, tx, userID)
				if err != nil {
					return err
				}
				newBalance = mathRound2(points.Points + pointsAwarded)
				reason := "成就奖励: " + rule.Name
				if err := tx.Model(&domain.UserPoints{}).Where("user_id = ?", userID).Updates(map[string]any{"points": newBalance, "updated_at": time.Now()}).Error; err != nil {
					return err
				}
				if err := tx.Create(&domain.PointsLog{UserID: userID, Amount: pointsAwarded, BalanceAfter: newBalance, Type: "achievement", Reason: &reason}).Error; err != nil {
					return err
				}
				awards = append(awards, AwardResult{Awarded: true, Amount: pointsAwarded, BalanceAfter: newBalance, Message: "积分已到账", Reason: reason})
			}
			if err := tx.Create(&domain.UserAchievementReward{UserID: userID, RuleID: rule.ID, PointsAwarded: pointsAwarded, CreatorBonusPercent: rule.CreatorBonusPercent}).Error; err != nil {
				return err
			}
			if rule.CreatorBonusPercent > 0 {
				now := time.Now()
				var expires *time.Time
				if rule.BonusDays > 0 {
					value := now.AddDate(0, 0, rule.BonusDays)
					expires = &value
				}
				ruleID := rule.ID
				bonus := domain.UserCreatorBonus{UserID: userID, RuleID: &ruleID, BonusPercent: rule.CreatorBonusPercent, IsActive: true, StartsAt: now, ExpiresAt: expires, UpdatedAt: &now}
				if err := tx.Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "user_id"}, {Name: "rule_id"}},
					DoUpdates: clause.Assignments(map[string]any{
						"bonus_percent": bonus.BonusPercent,
						"is_active":     true,
						"starts_at":     bonus.StartsAt,
						"expires_at":    bonus.ExpiresAt,
						"updated_at":    now,
					}),
				}).Create(&bonus).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	return awards, err
}

func (r PointsRepository) ActiveCreatorBonus(ctx context.Context, userID int64) (*domain.UserCreatorBonus, error) {
	var bonus domain.UserCreatorBonus
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_active = ? AND starts_at <= ? AND (expires_at IS NULL OR expires_at > ?)", userID, true, now, now).
		Order("bonus_percent DESC, id DESC").
		First(&bonus).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &bonus, nil
}

func (r PointsRepository) CreatorWithdrawBonus(ctx context.Context, userID int64, amount float64) (float64, *domain.UserCreatorBonus, error) {
	if amount <= 0 {
		return 0, nil, nil
	}
	bonus, err := r.ActiveCreatorBonus(ctx, userID)
	if err != nil || bonus == nil || bonus.BonusPercent <= 0 {
		return 0, bonus, err
	}
	return mathRound2(amount * bonus.BonusPercent / 100), bonus, nil
}
