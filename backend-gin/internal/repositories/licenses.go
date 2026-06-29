package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

type LicenseRepository struct {
	db *gorm.DB
}

func NewLicenseRepository(db *gorm.DB) LicenseRepository {
	return LicenseRepository{db: db}
}

func (r LicenseRepository) FindByKey(ctx context.Context, key string) (*domain.License, error) {
	var license domain.License
	if err := r.db.WithContext(ctx).Where("license_key = ?", key).First(&license).Error; err != nil {
		return nil, err
	}
	return &license, nil
}

func (r LicenseRepository) MarkVerified(ctx context.Context, id int64, machineID string, machineModel string, bindMachine bool) error {
	now := time.Now()
	update := map[string]any{
		"last_verified_at": now,
	}
	if bindMachine {
		update["machine_id"] = machineID
	}
	if machineModel != "" {
		update["machine_model"] = machineModel
	}
	return r.db.WithContext(ctx).Model(&domain.License{}).Where("id = ?", id).Updates(update).Error
}
