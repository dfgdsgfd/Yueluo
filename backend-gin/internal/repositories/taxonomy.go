package repositories

import (
	"context"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

type TaxonomyRepository struct {
	db *gorm.DB
}

func NewTaxonomyRepository(db *gorm.DB) TaxonomyRepository {
	return TaxonomyRepository{db: db}
}

func (r TaxonomyRepository) ListTags(ctx context.Context) ([]domain.Tag, error) {
	var tags []domain.Tag
	err := r.db.WithContext(ctx).Order("name ASC").Find(&tags).Error
	return tags, err
}

func (r TaxonomyRepository) HotTags(ctx context.Context, limit int) ([]domain.Tag, error) {
	var tags []domain.Tag
	err := r.db.WithContext(ctx).
		Where("use_count > ?", 0).
		Order("use_count DESC").
		Order("name ASC").
		Limit(limit).
		Find(&tags).Error
	return tags, err
}

func (r TaxonomyRepository) ListCategories(ctx context.Context) ([]domain.Category, error) {
	var categories []domain.Category
	err := r.db.WithContext(ctx).
		Order("use_count DESC").
		Order("name ASC").
		Find(&categories).Error
	return categories, err
}

func (r TaxonomyRepository) HotCategories(ctx context.Context, limit int) ([]domain.Category, error) {
	var categories []domain.Category
	err := r.db.WithContext(ctx).
		Where("use_count > ?", 0).
		Order("use_count DESC").
		Order("name ASC").
		Limit(limit).
		Find(&categories).Error
	return categories, err
}

func (r TaxonomyRepository) CategoryNameExists(ctx context.Context, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Category{}).Where("name = ?", name).Count(&count).Error
	return count > 0, err
}

func (r TaxonomyRepository) CategoryTitleExists(ctx context.Context, title string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Category{}).Where("category_title = ?", title).Count(&count).Error
	return count > 0, err
}

func (r TaxonomyRepository) CreateCategory(ctx context.Context, category domain.Category) (*domain.Category, error) {
	err := r.db.WithContext(ctx).Create(&category).Error
	if err != nil {
		return nil, err
	}
	return &category, nil
}
