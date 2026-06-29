package repositories

import (
	"context"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

type OpenAPIRepository struct {
	db *gorm.DB
}

type OpenPostListParams struct {
	Page       int
	Limit      int
	CategoryID *int
}

type OpenPostBundle struct {
	Post     domain.Post
	User     *domain.User
	Category *domain.Category
	Images   []domain.PostImage
	Videos   []domain.PostVideo
	Tags     []domain.Tag
}

func NewOpenAPIRepository(db *gorm.DB) OpenAPIRepository {
	return OpenAPIRepository{db: db}
}

func (r OpenAPIRepository) FindAPIKey(ctx context.Context, hashedKey string) (*domain.OpenAPI, error) {
	var api domain.OpenAPI
	err := r.db.WithContext(ctx).Where("api_key = ?", hashedKey).First(&api).Error
	if err != nil {
		return nil, err
	}
	return &api, nil
}

func (r OpenAPIRepository) TouchAPIKey(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&domain.OpenAPI{}).Where("id = ?", id).Update("last_used_at", time.Now()).Error
}

func (r OpenAPIRepository) ListPublicPosts(ctx context.Context, params OpenPostListParams) (int64, []OpenPostBundle, error) {
	query := r.db.WithContext(ctx).Model(&domain.Post{}).Where("is_draft = ? AND visibility = ?", false, "public")
	if params.CategoryID != nil {
		query = query.Where("category_id = ?", *params.CategoryID)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}
	var posts []domain.Post
	if err := query.
		Order("created_at DESC").
		Offset((params.Page - 1) * params.Limit).
		Limit(params.Limit).
		Find(&posts).Error; err != nil {
		return 0, nil, err
	}
	bundles, err := r.loadPostBundles(ctx, posts, true)
	return total, bundles, err
}

func (r OpenAPIRepository) FindPublicPost(ctx context.Context, id int64) (*OpenPostBundle, error) {
	var post domain.Post
	if err := r.db.WithContext(ctx).Where("id = ? AND is_draft = ? AND visibility = ?", id, false, "public").First(&post).Error; err != nil {
		return nil, err
	}
	bundles, err := r.loadPostBundles(ctx, []domain.Post{post}, true)
	if err != nil {
		return nil, err
	}
	return &bundles[0], nil
}

func (r OpenAPIRepository) PublicPostImages(ctx context.Context, id int64) ([]domain.PostImage, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.Post{}).Where("id = ? AND is_draft = ? AND visibility = ?", id, false, "public").Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, gorm.ErrRecordNotFound
	}
	var images []domain.PostImage
	err := r.db.WithContext(ctx).
		Where("post_id = ?", id).
		Order("sort_order ASC, id ASC").
		Find(&images).Error
	if err != nil {
		return nil, err
	}
	return publicPostImages(images, r.postPaymentEnabled(ctx, id)), nil
}

func (r OpenAPIRepository) loadPostBundles(ctx context.Context, posts []domain.Post, includeVideos bool) ([]OpenPostBundle, error) {
	if len(posts) == 0 {
		return []OpenPostBundle{}, nil
	}
	postIDs := make([]int64, 0, len(posts))
	userIDs := make([]int64, 0, len(posts))
	categoryIDs := make([]int, 0, len(posts))
	for _, post := range posts {
		postIDs = append(postIDs, post.ID)
		userIDs = append(userIDs, post.UserID)
		if post.CategoryID != nil {
			categoryIDs = append(categoryIDs, *post.CategoryID)
		}
	}

	users, err := r.usersByID(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	categories, err := r.categoriesByID(ctx, categoryIDs)
	if err != nil {
		return nil, err
	}
	images, err := r.imagesByPostID(ctx, postIDs)
	if err != nil {
		return nil, err
	}
	videos := map[int64][]domain.PostVideo{}
	if includeVideos {
		videos, err = r.videosByPostID(ctx, postIDs)
		if err != nil {
			return nil, err
		}
	}
	tags, err := r.tagsByPostID(ctx, postIDs)
	if err != nil {
		return nil, err
	}

	bundles := make([]OpenPostBundle, 0, len(posts))
	for _, post := range posts {
		bundle := OpenPostBundle{
			Post:   post,
			User:   users[post.UserID],
			Images: images[post.ID],
			Videos: videos[post.ID],
			Tags:   tags[post.ID],
		}
		if post.CategoryID != nil {
			bundle.Category = categories[*post.CategoryID]
		}
		bundles = append(bundles, bundle)
	}
	return bundles, nil
}

func (r OpenAPIRepository) usersByID(ctx context.Context, ids []int64) (map[int64]*domain.User, error) {
	var users []domain.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Select("id", "user_id", "nickname", "avatar").Find(&users).Error; err != nil {
		return nil, err
	}
	out := map[int64]*domain.User{}
	for i := range users {
		out[users[i].ID] = &users[i]
	}
	return out, nil
}

func (r OpenAPIRepository) categoriesByID(ctx context.Context, ids []int) (map[int]*domain.Category, error) {
	if len(ids) == 0 {
		return map[int]*domain.Category{}, nil
	}
	var categories []domain.Category
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Select("id", "name").Find(&categories).Error; err != nil {
		return nil, err
	}
	out := map[int]*domain.Category{}
	for i := range categories {
		out[categories[i].ID] = &categories[i]
	}
	return out, nil
}

func (r OpenAPIRepository) imagesByPostID(ctx context.Context, ids []int64) (map[int64][]domain.PostImage, error) {
	var images []domain.PostImage
	if err := r.db.WithContext(ctx).
		Where("post_id IN ?", ids).
		Select("id", "post_id", "image_url", "is_free_preview", "is_protected", "sort_order").
		Order("post_id ASC, sort_order ASC, id ASC").
		Find(&images).Error; err != nil {
		return nil, err
	}
	out := map[int64][]domain.PostImage{}
	for _, image := range images {
		out[image.PostID] = append(out[image.PostID], image)
	}
	var payments []domain.PostPaymentSetting
	if err := r.db.WithContext(ctx).Where("post_id IN ? AND enabled = ?", ids, true).Select("post_id").Find(&payments).Error; err != nil {
		return nil, err
	}
	paid := make(map[int64]bool, len(payments))
	for _, payment := range payments {
		paid[payment.PostID] = true
	}
	for postID, postImages := range out {
		out[postID] = publicPostImages(postImages, paid[postID])
	}
	return out, nil
}

func (r OpenAPIRepository) postPaymentEnabled(ctx context.Context, postID int64) bool {
	var count int64
	return r.db.WithContext(ctx).Model(&domain.PostPaymentSetting{}).Where("post_id = ? AND enabled = ?", postID, true).Count(&count).Error == nil && count > 0
}

func publicPostImages(images []domain.PostImage, paid bool) []domain.PostImage {
	normalized := NormalizePostImagesForAccess(images)
	visible := make([]domain.PostImage, 0, len(normalized))
	for _, image := range normalized {
		if image.IsProtected || (paid && !image.IsFreePreview) {
			continue
		}
		visible = append(visible, image)
	}
	return visible
}

func (r OpenAPIRepository) videosByPostID(ctx context.Context, ids []int64) (map[int64][]domain.PostVideo, error) {
	var videos []domain.PostVideo
	if err := r.db.WithContext(ctx).Where("post_id IN ?", ids).Select("id", "post_id", "cover_url", "video_url", "dash_url").Find(&videos).Error; err != nil {
		return nil, err
	}
	out := map[int64][]domain.PostVideo{}
	for _, video := range videos {
		out[video.PostID] = append(out[video.PostID], video)
	}
	return out, nil
}

type postTagRow struct {
	PostID int64
	ID     int
	Name   string
}

func (r OpenAPIRepository) tagsByPostID(ctx context.Context, ids []int64) (map[int64][]domain.Tag, error) {
	var rows []postTagRow
	err := r.db.WithContext(ctx).
		Table("post_tags").
		Select("post_tags.post_id, tags.id, tags.name").
		Joins("JOIN tags ON tags.id = post_tags.tag_id").
		Where("post_tags.post_id IN ?", ids).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := map[int64][]domain.Tag{}
	for _, row := range rows {
		out[row.PostID] = append(out[row.PostID], domain.Tag{ID: row.ID, Name: row.Name})
	}
	return out, nil
}
