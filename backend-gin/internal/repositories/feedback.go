package repositories

import (
	"context"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

type FeedbackRepository struct {
	db *gorm.DB
}

func NewFeedbackRepository(db *gorm.DB) FeedbackRepository {
	return FeedbackRepository{db: db}
}

func (r FeedbackRepository) Create(ctx context.Context, feedback domain.Feedback) (*domain.Feedback, error) {
	if feedback.Status == "" {
		feedback.Status = "pending"
	}
	if err := r.db.WithContext(ctx).Create(&feedback).Error; err != nil {
		return nil, err
	}
	return &feedback, nil
}

func (r FeedbackRepository) ListMine(ctx context.Context, userID int64, page, limit int) (int64, []domain.Feedback, error) {
	query := r.db.WithContext(ctx).Model(&domain.Feedback{}).Where("user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var list []domain.Feedback
	if err := query.
		Select("id", "content", "images", "video_url", "status", "admin_reply", "replied_at", "created_at").
		Order("created_at DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&list).Error; err != nil {
		return 0, nil, err
	}
	return total, list, nil
}

func (r FeedbackRepository) FindOwned(ctx context.Context, id, userID int64) (*domain.Feedback, bool, error) {
	var feedback domain.Feedback
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&feedback).Error; err != nil {
		return nil, false, err
	}
	return &feedback, feedback.UserID == userID, nil
}
