package repositories

import (
	"context"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

type AppRepository struct {
	db *gorm.DB
}

func NewAppRepository(db *gorm.DB) AppRepository {
	return AppRepository{db: db}
}

func (r AppRepository) ActiveVersions(ctx context.Context, platform string) ([]domain.AppVersion, error) {
	var versions []domain.AppVersion
	err := r.db.WithContext(ctx).
		Where("platform = ? AND is_active = ?", platform, true).
		Find(&versions).Error
	return versions, err
}

func (r AppRepository) VersionByCode(ctx context.Context, platform string, versionCode int) (*domain.AppVersion, error) {
	var version domain.AppVersion
	err := r.db.WithContext(ctx).
		Where("platform = ? AND version_code = ?", platform, versionCode).
		Select("id").
		First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func (r AppRepository) CreateUsageLog(ctx context.Context, log domain.AppUsageLog) error {
	return r.db.WithContext(ctx).Create(&log).Error
}
