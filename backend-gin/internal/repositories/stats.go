package repositories

import (
	"context"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

type StatsRepository struct {
	db *gorm.DB
}

func NewStatsRepository(db *gorm.DB) StatsRepository {
	return StatsRepository{db: db}
}

func (r StatsRepository) Counts(ctx context.Context) (map[string]int64, error) {
	result := map[string]int64{}
	tables := []struct {
		alias string
		model any
	}{
		{alias: "users", model: &domain.User{}},
		{alias: "posts", model: &domain.Post{}},
		{alias: "comments", model: &domain.Comment{}},
		{alias: "likes", model: &domain.Like{}},
	}
	for _, table := range tables {
		var count int64
		if err := r.db.WithContext(ctx).Model(table.model).Count(&count).Error; err != nil {
			return nil, err
		}
		result[table.alias] = count
	}
	return result, nil
}
