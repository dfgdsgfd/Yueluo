package repositories

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

type InviteRepository struct {
	db *gorm.DB
}

type InviteStats struct {
	Code          domain.InviteCode
	Invitees      []InviteeBundle
	InviteesTotal int64
	EarningsLogs  []domain.InviteEarningsLog
}

type InviteeBundle struct {
	Code domain.InviteCode
	User *domain.User
}

type InviteAdminBundle struct {
	Code      domain.InviteCode
	User      *domain.User
	InvitedBy *domain.User
}

type InviteOverview struct {
	TotalCodes     int64
	TotalClicks    int64
	TotalRegisters int64
	TotalEarnings  float64
}

func NewInviteRepository(db *gorm.DB) InviteRepository {
	return InviteRepository{db: db}
}

func (r InviteRepository) GetOrCreateCode(ctx context.Context, userID int64) (*domain.InviteCode, error) {
	var record domain.InviteCode
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&record).Error
	if err == nil {
		return &record, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	code, err := r.generateUniqueCode(ctx, r.db.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	record = domain.InviteCode{UserID: userID, Code: code, IsActive: true}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

func (r InviteRepository) RecordClick(ctx context.Context, code, ipHash, userAgent string) (bool, error) {
	var record domain.InviteCode
	if err := r.db.WithContext(ctx).Where("code = ? AND is_active = ?", code, true).First(&record).Error; err != nil {
		return false, err
	}
	todayStart := time.Now()
	todayStart = time.Date(todayStart.Year(), todayStart.Month(), todayStart.Day(), 0, 0, 0, 0, todayStart.Location())
	var existing domain.InviteClick
	err := r.db.WithContext(ctx).Where("code = ? AND ip_hash = ? AND created_at >= ?", code, ipHash, todayStart).First(&existing).Error
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}
	return true, r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ua := truncate(userAgent, 500)
		click := domain.InviteClick{Code: code, IPHash: ipHash, UserAgent: &ua}
		if err := tx.Create(&click).Error; err != nil {
			return err
		}
		return tx.Model(&domain.InviteCode{}).Where("code = ?", code).UpdateColumn("click_count", gorm.Expr("click_count + ?", 1)).Error
	})
}

func (r InviteRepository) Stats(ctx context.Context, userID int64, page, limit int) (*InviteStats, error) {
	code, err := r.GetOrCreateCode(ctx, userID)
	if err != nil {
		return nil, err
	}
	var invitees []domain.InviteCode
	query := r.db.WithContext(ctx).Where("invited_by_id = ?", userID).Model(&domain.InviteCode{})
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := query.Order("created_at DESC").Offset((page - 1) * limit).Limit(limit).Find(&invitees).Error; err != nil {
		return nil, err
	}
	users, err := r.usersByID(ctx, inviteCodeUserIDs(invitees))
	if err != nil {
		return nil, err
	}
	bundles := make([]InviteeBundle, 0, len(invitees))
	for _, invitee := range invitees {
		bundles = append(bundles, InviteeBundle{Code: invitee, User: users[invitee.UserID]})
	}
	var logs []domain.InviteEarningsLog
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at DESC").Limit(20).Find(&logs).Error; err != nil {
		return nil, err
	}
	return &InviteStats{Code: *code, Invitees: bundles, InviteesTotal: total, EarningsLogs: logs}, nil
}

func (r InviteRepository) Info(ctx context.Context, code string) (*domain.User, error) {
	var record domain.InviteCode
	if err := r.db.WithContext(ctx).Where("code = ? AND is_active = ?", code, true).First(&record).Error; err != nil {
		return nil, err
	}
	var user domain.User
	if err := r.db.WithContext(ctx).Where("id = ?", record.UserID).Select("id", "nickname", "avatar").First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

func (r InviteRepository) AdminList(ctx context.Context, page, limit int, keyword string) (int64, []InviteAdminBundle, error) {
	query := r.db.WithContext(ctx).Model(&domain.InviteCode{})
	if strings.TrimSpace(keyword) != "" {
		like := "%" + strings.TrimSpace(keyword) + "%"
		query = query.Joins("JOIN users ON users.id = invite_codes.user_id").
			Where("users.nickname LIKE ? OR users.user_id LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var records []domain.InviteCode
	if err := query.Order("invite_codes.click_count DESC").Offset((page - 1) * limit).Limit(limit).Find(&records).Error; err != nil {
		return 0, nil, err
	}
	userIDs := inviteCodeUserIDs(records)
	for _, record := range records {
		if record.InvitedByID != nil {
			userIDs = append(userIDs, *record.InvitedByID)
		}
	}
	users, err := r.usersByID(ctx, userIDs)
	if err != nil {
		return 0, nil, err
	}
	bundles := make([]InviteAdminBundle, 0, len(records))
	for _, record := range records {
		var invitedBy *domain.User
		if record.InvitedByID != nil {
			invitedBy = users[*record.InvitedByID]
		}
		bundles = append(bundles, InviteAdminBundle{Code: record, User: users[record.UserID], InvitedBy: invitedBy})
	}
	return total, bundles, nil
}

func (r InviteRepository) Toggle(ctx context.Context, id int64) (bool, error) {
	var record domain.InviteCode
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&record).Error; err != nil {
		return false, err
	}
	next := !record.IsActive
	return next, r.db.WithContext(ctx).Model(&domain.InviteCode{}).Where("id = ?", id).Update("is_active", next).Error
}

func (r InviteRepository) Reward(ctx context.Context, userID int64, amount float64, rewardType string, reason *string) error {
	var user domain.User
	if err := r.db.WithContext(ctx).Where("id = ?", userID).Select("id").First(&user).Error; err != nil {
		return err
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		log := domain.InviteEarningsLog{UserID: userID, Amount: amount, Type: rewardType, Reason: reason}
		if err := tx.Create(&log).Error; err != nil {
			return err
		}
		var record domain.InviteCode
		err := tx.Where("user_id = ?", userID).First(&record).Error
		if err == nil {
			return tx.Model(&domain.InviteCode{}).Where("user_id = ?", userID).
				UpdateColumn("total_earnings", gorm.Expr("total_earnings + ?", amount)).Error
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		code, err := r.generateUniqueCode(ctx, tx)
		if err != nil {
			return err
		}
		return tx.Create(&domain.InviteCode{UserID: userID, Code: code, TotalEarnings: amount, IsActive: true}).Error
	})
}

func (r InviteRepository) Overview(ctx context.Context) (*InviteOverview, error) {
	var overview InviteOverview
	if err := r.db.WithContext(ctx).Model(&domain.InviteCode{}).Count(&overview.TotalCodes).Error; err != nil {
		return nil, err
	}
	var sums struct {
		TotalClicks    int64
		TotalRegisters int64
		TotalEarnings  float64
	}
	err := r.db.WithContext(ctx).Model(&domain.InviteCode{}).
		Select("COALESCE(SUM(click_count),0) AS total_clicks, COALESCE(SUM(register_count),0) AS total_registers, COALESCE(SUM(total_earnings),0) AS total_earnings").
		Scan(&sums).Error
	if err != nil {
		return nil, err
	}
	overview.TotalClicks = sums.TotalClicks
	overview.TotalRegisters = sums.TotalRegisters
	overview.TotalEarnings = sums.TotalEarnings
	return &overview, nil
}

func (r InviteRepository) generateUniqueCode(ctx context.Context, db *gorm.DB) (string, error) {
	for range 32 {
		buf := make([]byte, 4)
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		code := strings.ToUpper(hex.EncodeToString(buf))
		var count int64
		if err := db.WithContext(ctx).Model(&domain.InviteCode{}).Where("code = ?", code).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return code, nil
		}
	}
	return "", errors.New("failed to generate invite code")
}

func (r InviteRepository) usersByID(ctx context.Context, ids []int64) (map[int64]*domain.User, error) {
	out := map[int64]*domain.User{}
	if len(ids) == 0 {
		return out, nil
	}
	var users []domain.User
	if err := r.db.WithContext(ctx).Where("id IN ?", uniqueInt64(ids)).Select("id", "user_id", "nickname", "avatar").Find(&users).Error; err != nil {
		return nil, err
	}
	for i := range users {
		out[users[i].ID] = &users[i]
	}
	return out, nil
}

func inviteCodeUserIDs(records []domain.InviteCode) []int64 {
	out := make([]int64, 0, len(records))
	for _, record := range records {
		out = append(out, record.UserID)
	}
	return out
}

func uniqueInt64(values []int64) []int64 {
	seen := map[int64]bool{}
	out := make([]int64, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	return out
}

func truncate(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}
