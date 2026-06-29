package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

type AnnouncementRepository struct {
	db *gorm.DB
}

type AnnouncementListParams struct {
	Page     int
	PageSize int
	Type     string
}

func NewAnnouncementRepository(db *gorm.DB) AnnouncementRepository {
	return AnnouncementRepository{db: db}
}

func (r AnnouncementRepository) ListPublished(ctx context.Context, params AnnouncementListParams) (int64, []domain.Announcement, error) {
	now := time.Now()
	query := r.db.WithContext(ctx).Model(&domain.Announcement{}).Where("is_published = ? AND (expires_at IS NULL OR expires_at > ?)", true, now)
	if params.Type != "" {
		query = query.Where("type = ?", params.Type)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var list []domain.Announcement
	err := query.
		Order("created_at DESC").
		Offset((params.Page - 1) * params.PageSize).
		Limit(params.PageSize).
		Find(&list).Error
	if err != nil {
		return 0, nil, err
	}
	return total, list, nil
}

func (r AnnouncementRepository) FindPublished(ctx context.Context, id int64) (*domain.Announcement, error) {
	var announcement domain.Announcement
	now := time.Now()
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_published = ? AND (expires_at IS NULL OR expires_at > ?)", id, true, now).
		First(&announcement).Error
	if err != nil {
		return nil, err
	}
	return &announcement, nil
}
