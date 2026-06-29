package repositories

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
)

func (r PointsRepository) dailyStatsByTask(ctx context.Context, userID int64, date time.Time) (map[string]domain.PointsDailyStat, error) {
	var rows []domain.PointsDailyStat
	if err := r.db.WithContext(ctx).Where("user_id = ? AND stat_date = ?", userID, date).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := map[string]domain.PointsDailyStat{}
	for _, row := range rows {
		out[row.TaskType] = row
	}
	return out, nil
}

func (r PointsRepository) taskEventAggregatesByTask(ctx context.Context, userID int64) (map[string]pointsTaskAggregate, error) {
	var rows []struct {
		TaskType       string
		CompletedCount int64
		AwardedPoints  float64
	}
	err := r.db.WithContext(ctx).Model(&domain.PointsTaskEvent{}).
		Select("task_type, COUNT(*) AS completed_count, COALESCE(SUM(points),0) AS awarded_points").
		Where("user_id = ?", userID).
		Group("task_type").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[string]pointsTaskAggregate{}
	for _, row := range rows {
		out[row.TaskType] = pointsTaskAggregate{CompletedCount: int(row.CompletedCount), AwardedPoints: row.AwardedPoints}
	}
	return out, nil
}

func (r PointsRepository) productsByID(ctx context.Context, ids []int64) (map[int64]*domain.GiftCardProduct, error) {
	out := map[int64]*domain.GiftCardProduct{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []domain.GiftCardProduct
	if err := r.db.WithContext(ctx).Where("id IN ?", uniqueInt64(ids)).Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		out[rows[i].ID] = &rows[i]
	}
	return out, nil
}

func pointsTaskConfigTx(ctx context.Context, tx *gorm.DB, taskType string) (domain.PointsTaskConfig, error) {
	var config domain.PointsTaskConfig
	err := tx.WithContext(ctx).Where("task_type = ?", taskType).First(&config).Error
	if err == nil {
		return config, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return config, err
	}
	for _, item := range defaultPointsTaskConfigs() {
		if item.TaskType == taskType {
			return item, nil
		}
	}
	return config, ErrPointsTaskNotConfigured
}

func getOrCreateDailyStatTx(ctx context.Context, tx *gorm.DB, userID int64, taskType string, date time.Time) (*domain.PointsDailyStat, error) {
	var stat domain.PointsDailyStat
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ? AND task_type = ? AND stat_date = ?", userID, taskType, date).
		First(&stat).Error
	if err == nil {
		return &stat, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	stat = domain.PointsDailyStat{UserID: userID, TaskType: taskType, StatDate: date, CompletedCount: 0, AwardedPoints: 0}
	if err := tx.WithContext(ctx).Create(&stat).Error; err != nil {
		return nil, err
	}
	return &stat, nil
}

func dailyPointsTotal(ctx context.Context, db *gorm.DB, userID int64, date time.Time) (float64, error) {
	var row struct{ Total float64 }
	err := db.WithContext(ctx).Model(&domain.PointsDailyStat{}).Select("COALESCE(SUM(awarded_points),0) AS total").Where("user_id = ? AND stat_date = ?", userID, date).Scan(&row).Error
	return mathRound2(row.Total), err
}

func dailyPointsTotalTx(ctx context.Context, tx *gorm.DB, userID int64, date time.Time) (float64, error) {
	var row struct{ Total float64 }
	err := tx.WithContext(ctx).Model(&domain.PointsDailyStat{}).Select("COALESCE(SUM(awarded_points),0) AS total").Where("user_id = ? AND stat_date = ?", userID, date).Scan(&row).Error
	return mathRound2(row.Total), err
}

func achievementReachedTx(ctx context.Context, tx *gorm.DB, userID int64, rule domain.PointsAchievementRule) (bool, error) {
	threshold := rule.ThresholdValue
	if threshold <= 0 {
		return false, nil
	}
	switch normalizeAchievementTrigger(rule.TriggerType) {
	case PointsAchievementTotalPosts:
		var count int64
		err := tx.WithContext(ctx).Model(&domain.Post{}).Where("user_id = ? AND is_draft = ?", userID, false).Count(&count).Error
		return count >= int64(threshold), err
	case PointsAchievementConsecutivePosts:
		streak, err := consecutivePostDaysTx(ctx, tx, userID)
		return streak >= threshold, err
	case PointsAchievementTotalPoints:
		var points domain.UserPoints
		err := tx.WithContext(ctx).Where("user_id = ?", userID).First(&points).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return points.Points >= float64(threshold), err
	default:
		return false, nil
	}
}

func consecutivePostDaysTx(ctx context.Context, tx *gorm.DB, userID int64) (int, error) {
	var rows []struct {
		Date time.Time `gorm:"column:day"`
	}
	err := tx.WithContext(ctx).Raw(`
		SELECT DATE(created_at) AS day
		FROM posts
		WHERE user_id = ? AND is_draft = false
		GROUP BY DATE(created_at)
		ORDER BY day DESC
		LIMIT 366
	`, userID).Scan(&rows).Error
	if err != nil {
		return 0, err
	}
	if len(rows) == 0 {
		return 0, nil
	}
	streak := 0
	expected := dateOnly(time.Now())
	for _, row := range rows {
		day := dateOnly(row.Date)
		if streak == 0 && day.Before(expected) {
			yesterday := expected.AddDate(0, 0, -1)
			if day.Equal(yesterday) {
				expected = yesterday
			}
		}
		if day.Equal(expected) {
			streak++
			expected = expected.AddDate(0, 0, -1)
			continue
		}
		if day.Before(expected) {
			break
		}
	}
	return streak, nil
}

func defaultPointsTaskConfigs() []domain.PointsTaskConfig {
	now := time.Now()
	items := []domain.PointsTaskConfig{
		{TaskType: PointsTaskComment, Name: "评论", Description: stringPtr("发布评论自动获得积分"), Points: 2, DailyLimit: 10, IsDailyTask: true, IsActive: true, SortOrder: 10, CreatedAt: now},
		{TaskType: PointsTaskClick, Name: "点击", Description: stringPtr("点击进入内容详情自动获得积分"), Points: 1, DailyLimit: 30, IsDailyTask: true, IsActive: true, SortOrder: 20, CreatedAt: now},
		{TaskType: PointsTaskLike, Name: "点赞", Description: stringPtr("点赞内容自动获得积分"), Points: 1, DailyLimit: 20, IsDailyTask: true, IsActive: true, SortOrder: 30, CreatedAt: now},
		{TaskType: PointsTaskCollect, Name: "收藏", Description: stringPtr("收藏内容自动获得积分"), Points: 2, DailyLimit: 10, IsDailyTask: true, IsActive: true, SortOrder: 40, CreatedAt: now},
		{TaskType: PointsTaskView, Name: "浏览", Description: stringPtr("浏览内容自动获得积分"), Points: 1, DailyLimit: 30, IsDailyTask: true, IsActive: true, SortOrder: 50, CreatedAt: now},
		{TaskType: PointsTaskPost, Name: "发帖", Description: stringPtr("发布公开内容自动获得积分"), Points: 5, DailyLimit: 5, IsDailyTask: true, IsActive: true, SortOrder: 60, CreatedAt: now},
		{TaskType: PointsTaskSetAvatar, Name: "设置头像", Description: stringPtr("设置头像获得一次性积分"), Points: 2, DailyLimit: 1, IsDailyTask: false, IsActive: true, SortOrder: 70, CreatedAt: now},
		{TaskType: PointsTaskSetBackground, Name: "设置背景", Description: stringPtr("设置个人背景获得一次性积分"), Points: 2, DailyLimit: 1, IsDailyTask: false, IsActive: true, SortOrder: 80, CreatedAt: now},
		{TaskType: PointsTaskSetSignature, Name: "设置签名", Description: stringPtr("设置个人签名获得一次性积分"), Points: 2, DailyLimit: 1, IsDailyTask: false, IsActive: true, SortOrder: 90, CreatedAt: now},
		{TaskType: PointsTaskSetName, Name: "设置名称", Description: stringPtr("设置名称获得一次性积分"), Points: 2, DailyLimit: 1, IsDailyTask: false, IsActive: true, SortOrder: 100, CreatedAt: now},
	}
	for i := range items {
		items[i].UpdatedAt = &now
	}
	return items
}

func hasFixedTaskConfig(configs []domain.PointsTaskConfig) bool {
	for _, config := range configs {
		if !config.IsDailyTask {
			return true
		}
	}
	return false
}

func normalizeTaskType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "comment", "评论":
		return PointsTaskComment
	case "click", "点击":
		return PointsTaskClick
	case "like", "点赞":
		return PointsTaskLike
	case "collect", "favorite", "收藏":
		return PointsTaskCollect
	case "view", "browse", "浏览":
		return PointsTaskView
	case "post", "publish", "发帖", "发布":
		return PointsTaskPost
	case "set_avatar", "avatar", "设置头像":
		return PointsTaskSetAvatar
	case "set_background", "background", "设置背景":
		return PointsTaskSetBackground
	case "set_signature", "signature", "bio", "设置签名":
		return PointsTaskSetSignature
	case "set_name", "name", "nickname", "设置名称":
		return PointsTaskSetName
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func normalizeAchievementTrigger(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "total_posts", "posts", "post_count":
		return PointsAchievementTotalPosts
	case "consecutive_posts", "post_streak", "streak":
		return PointsAchievementConsecutivePosts
	case "total_points", "points":
		return PointsAchievementTotalPoints
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func defaultTaskName(taskType string) string {
	switch normalizeTaskType(taskType) {
	case PointsTaskComment:
		return "评论"
	case PointsTaskClick:
		return "点击"
	case PointsTaskLike:
		return "点赞"
	case PointsTaskCollect:
		return "收藏"
	case PointsTaskView:
		return "浏览"
	case PointsTaskPost:
		return "发帖"
	case PointsTaskSetAvatar:
		return "设置头像"
	case PointsTaskSetBackground:
		return "设置背景"
	case PointsTaskSetSignature:
		return "设置签名"
	case PointsTaskSetName:
		return "设置名称"
	default:
		return strings.TrimSpace(taskType)
	}
}

func dateOnly(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func uniqueCodeLines(text string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, raw := range strings.FieldsFunc(text, func(r rune) bool { return r == '\n' || r == '\r' }) {
		code := strings.TrimSpace(raw)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		out = append(out, code)
	}
	sort.Strings(out)
	return out
}

func productIDs(products []domain.GiftCardProduct) []int64 {
	out := make([]int64, 0, len(products))
	for _, product := range products {
		out = append(out, product.ID)
	}
	return out
}

func redemptionProductIDs(rows []domain.GiftCardRedemption) []int64 {
	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ProductID)
	}
	return out
}

func stringPtr(value string) *string {
	return &value
}

func positiveFloat(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}

func PointsTaskTarget(taskType string, id any) string {
	return fmt.Sprintf("%s:%v", normalizeTaskType(taskType), id)
}
