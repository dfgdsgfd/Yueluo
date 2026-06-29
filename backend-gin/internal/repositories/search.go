package repositories

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

type SearchRepository struct {
	db *gorm.DB
}

type SearchPostParams struct {
	Keyword            string
	Tag                string
	Type               string
	Page               int
	Limit              int
	PublicAccessOnly   bool
	ExcludeVideoGuests bool
}

type SearchPostBundle struct {
	Post           domain.Post
	User           *domain.User
	Images         []domain.PostImage
	Videos         []domain.PostVideo
	Tags           []domain.Tag
	PaymentSetting *domain.PostPaymentSetting
}

type SearchUserParams struct {
	Keyword string
	Page    int
	Limit   int
}

type SearchUserBundle struct {
	User      domain.User
	PostCount int64
}

type SearchTagStat struct {
	Name  string
	Count int64
}

func NewSearchRepository(db *gorm.DB) SearchRepository {
	return SearchRepository{db: db}
}

func (r SearchRepository) SearchPosts(ctx context.Context, params SearchPostParams) (int64, []SearchPostBundle, error) {
	query := r.searchPostQuery(ctx, params)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var posts []domain.Post
	if err := query.
		Select("posts.*").
		Order("posts.created_at DESC").
		Offset((params.Page - 1) * params.Limit).
		Limit(params.Limit).
		Find(&posts).Error; err != nil {
		return 0, nil, err
	}
	bundles, err := r.loadSearchPostBundles(ctx, posts)
	return total, bundles, err
}

func (r SearchRepository) TagStats(ctx context.Context, params SearchPostParams) ([]SearchTagStat, error) {
	keyword := strings.TrimSpace(params.Keyword)
	if keyword == "" {
		return []SearchTagStat{}, nil
	}
	like := "%" + keyword + "%"
	query := r.db.WithContext(ctx).
		Table("post_tags").
		Select("post_tags.tag_id, COUNT(post_tags.tag_id) AS count").
		Joins("JOIN posts ON posts.id = post_tags.post_id").
		Joins("LEFT JOIN users ON users.id = posts.user_id").
		Where("posts.is_draft = ? AND posts.visibility = ?", false, "public").
		Where(`posts.title LIKE ? OR posts.content LIKE ? OR users.nickname LIKE ? OR users.user_id LIKE ? OR EXISTS (
			SELECT 1 FROM post_tags pt_kw JOIN tags t_kw ON t_kw.id = pt_kw.tag_id
			WHERE pt_kw.post_id = posts.id AND t_kw.name LIKE ?
		)`, like, like, like, like, like)
	if params.Type == "posts" {
		query = query.Where("posts.type = ?", 1)
	} else if params.Type == "videos" {
		query = query.Where("posts.type = ?", 2)
	}
	if params.PublicAccessOnly {
		query = query.Where("posts.public_access_exempt = ?", true)
	}
	if params.ExcludeVideoGuests {
		query = query.Where("(posts.type <> ? OR posts.public_access_exempt = ?)", PostTypeVideo, true)
	}

	var rows []struct {
		TagID int   `gorm:"column:tag_id"`
		Count int64 `gorm:"column:count"`
	}
	if err := query.Group("post_tags.tag_id").Order("post_tags.tag_id ASC").Limit(10).Scan(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []SearchTagStat{}, nil
	}

	tagIDs := make([]int, 0, len(rows))
	counts := map[int]int64{}
	for _, row := range rows {
		tagIDs = append(tagIDs, row.TagID)
		counts[row.TagID] = row.Count
	}
	var tags []domain.Tag
	if err := r.db.WithContext(ctx).Where("id IN ?", tagIDs).Select("id", "name").Order("name ASC").Find(&tags).Error; err != nil {
		return nil, err
	}
	stats := make([]SearchTagStat, 0, len(tags))
	for _, tag := range tags {
		stats = append(stats, SearchTagStat{Name: tag.Name, Count: counts[tag.ID]})
	}
	return stats, nil
}

func (r SearchRepository) SearchUsers(ctx context.Context, params SearchUserParams) (int64, []SearchUserBundle, error) {
	like := "%" + strings.TrimSpace(params.Keyword) + "%"
	query := r.db.WithContext(ctx).Model(&domain.User{}).
		Where("nickname LIKE ? OR user_id LIKE ?", like, like)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return 0, nil, err
	}

	var users []domain.User
	if err := query.
		Select("id", "user_id", "nickname", "avatar", "bio", "location", "follow_count", "fans_count", "like_count", "created_at", "verified").
		Order("created_at DESC").
		Offset((params.Page - 1) * params.Limit).
		Limit(params.Limit).
		Find(&users).Error; err != nil {
		return 0, nil, err
	}

	counts, err := r.postCountsByUser(ctx, searchUserIDs(users))
	if err != nil {
		return 0, nil, err
	}
	out := make([]SearchUserBundle, 0, len(users))
	for _, user := range users {
		out = append(out, SearchUserBundle{User: user, PostCount: counts[user.ID]})
	}
	return total, out, nil
}

func (r SearchRepository) RecordSearchHistory(ctx context.Context, userID int64, keyword string) error {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil
	}
	if len([]rune(keyword)) > 100 {
		keyword = string([]rune(keyword)[:100])
	}
	return r.db.WithContext(ctx).Create(&domain.UserSearchHistory{UserID: userID, Keyword: keyword, CreatedAt: time.Now()}).Error
}

func (r SearchRepository) PurchasedPostIDs(ctx context.Context, userID int64, postIDs []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	if userID == 0 || len(postIDs) == 0 {
		return out, nil
	}
	var rows []domain.UserPurchasedContent
	if err := r.db.WithContext(ctx).Where("user_id = ? AND post_id IN ?", userID, uniqueInt64(postIDs)).Select("post_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PostID] = true
	}
	return out, nil
}

func (r SearchRepository) LikedPostIDs(ctx context.Context, userID int64, postIDs []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	if userID == 0 || len(postIDs) == 0 {
		return out, nil
	}
	var rows []domain.Like
	if err := r.db.WithContext(ctx).Where("user_id = ? AND target_type = ? AND target_id IN ?", userID, 1, uniqueInt64(postIDs)).Select("target_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.TargetID] = true
	}
	return out, nil
}

func (r SearchRepository) CollectedPostIDs(ctx context.Context, userID int64, postIDs []int64) (map[int64]bool, error) {
	out := map[int64]bool{}
	if userID == 0 || len(postIDs) == 0 {
		return out, nil
	}
	var rows []domain.Collection
	if err := r.db.WithContext(ctx).Where("user_id = ? AND post_id IN ?", userID, uniqueInt64(postIDs)).Select("post_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PostID] = true
	}
	return out, nil
}

func (r SearchRepository) FollowSets(ctx context.Context, currentUserID int64, userIDs []int64) (map[int64]bool, map[int64]bool, error) {
	following := map[int64]bool{}
	followers := map[int64]bool{}
	if currentUserID == 0 || len(userIDs) == 0 {
		return following, followers, nil
	}
	ids := uniqueInt64(userIDs)
	if err := runParallel(
		func() error {
			var followingRows []domain.Follow
			if err := r.db.WithContext(ctx).Where("follower_id = ? AND following_id IN ?", currentUserID, ids).Select("following_id").Find(&followingRows).Error; err != nil {
				return err
			}
			for _, row := range followingRows {
				following[row.FollowingID] = true
			}
			return nil
		},
		func() error {
			var followerRows []domain.Follow
			if err := r.db.WithContext(ctx).Where("follower_id IN ? AND following_id = ?", ids, currentUserID).Select("follower_id").Find(&followerRows).Error; err != nil {
				return err
			}
			for _, row := range followerRows {
				followers[row.FollowerID] = true
			}
			return nil
		},
	); err != nil {
		return nil, nil, err
	}
	return following, followers, nil
}

func (r SearchRepository) searchPostQuery(ctx context.Context, params SearchPostParams) *gorm.DB {
	query := r.db.WithContext(ctx).Model(&domain.Post{}).
		Joins("LEFT JOIN users ON users.id = posts.user_id").
		Where("posts.is_draft = ? AND posts.visibility = ?", false, "public")
	keyword := strings.TrimSpace(params.Keyword)
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(`posts.title LIKE ? OR posts.content LIKE ? OR users.nickname LIKE ? OR users.user_id LIKE ? OR EXISTS (
			SELECT 1 FROM post_tags pt_kw JOIN tags t_kw ON t_kw.id = pt_kw.tag_id
			WHERE pt_kw.post_id = posts.id AND t_kw.name LIKE ?
		)`, like, like, like, like, like)
	}
	if tag := strings.TrimSpace(params.Tag); tag != "" {
		query = query.Where(`EXISTS (
			SELECT 1 FROM post_tags pt_tag JOIN tags t_tag ON t_tag.id = pt_tag.tag_id
			WHERE pt_tag.post_id = posts.id AND t_tag.name = ?
		)`, tag)
	}
	if params.Type == "posts" {
		query = query.Where("posts.type = ?", 1)
	} else if params.Type == "videos" {
		query = query.Where("posts.type = ?", 2)
	}
	if params.PublicAccessOnly {
		query = query.Where("posts.public_access_exempt = ?", true)
	}
	if params.ExcludeVideoGuests {
		query = query.Where("(posts.type <> ? OR posts.public_access_exempt = ?)", PostTypeVideo, true)
	}
	return query
}

func (r SearchRepository) loadSearchPostBundles(ctx context.Context, posts []domain.Post) ([]SearchPostBundle, error) {
	if len(posts) == 0 {
		return []SearchPostBundle{}, nil
	}
	postIDs := searchPostIDs(posts)
	userIDs := searchPostUserIDs(posts)

	var users map[int64]*domain.User
	var images map[int64][]domain.PostImage
	var videos map[int64][]domain.PostVideo
	var tags map[int64][]domain.Tag
	var paymentSettings map[int64]*domain.PostPaymentSetting
	if err := runParallel(
		func() error {
			var err error
			users, err = r.searchUsersByID(ctx, userIDs)
			return err
		},
		func() error {
			var err error
			images, err = r.searchImagesByPostID(ctx, postIDs)
			return err
		},
		func() error {
			var err error
			videos, err = r.searchVideosByPostID(ctx, postIDs)
			return err
		},
		func() error {
			var err error
			tags, err = r.searchTagsByPostID(ctx, postIDs)
			return err
		},
		func() error {
			var err error
			paymentSettings, err = r.paymentSettingsByPostID(ctx, postIDs)
			return err
		},
	); err != nil {
		return nil, err
	}

	out := make([]SearchPostBundle, 0, len(posts))
	for _, post := range posts {
		out = append(out, SearchPostBundle{
			Post:           post,
			User:           users[post.UserID],
			Images:         images[post.ID],
			Videos:         videos[post.ID],
			Tags:           tags[post.ID],
			PaymentSetting: paymentSettings[post.ID],
		})
	}
	return out, nil
}

func (r SearchRepository) searchUsersByID(ctx context.Context, ids []int64) (map[int64]*domain.User, error) {
	out := map[int64]*domain.User{}
	if len(ids) == 0 {
		return out, nil
	}
	var users []domain.User
	if err := r.db.WithContext(ctx).Where("id IN ?", uniqueInt64(ids)).Select("id", "user_id", "nickname", "avatar", "location").Find(&users).Error; err != nil {
		return nil, err
	}
	for i := range users {
		out[users[i].ID] = &users[i]
	}
	return out, nil
}

func (r SearchRepository) searchImagesByPostID(ctx context.Context, ids []int64) (map[int64][]domain.PostImage, error) {
	out := map[int64][]domain.PostImage{}
	if len(ids) == 0 {
		return out, nil
	}
	var images []domain.PostImage
	if err := r.db.WithContext(ctx).
		Where("post_id IN ?", uniqueInt64(ids)).
		Select("id", "post_id", "image_url", "is_free_preview", "is_protected", "sort_order").
		Order("post_id ASC, sort_order ASC, id ASC").
		Find(&images).Error; err != nil {
		return nil, err
	}
	for _, image := range images {
		out[image.PostID] = append(out[image.PostID], image)
	}
	return out, nil
}

func (r SearchRepository) searchVideosByPostID(ctx context.Context, ids []int64) (map[int64][]domain.PostVideo, error) {
	out := map[int64][]domain.PostVideo{}
	if len(ids) == 0 {
		return out, nil
	}
	var videos []domain.PostVideo
	if err := r.db.WithContext(ctx).Where("post_id IN ?", uniqueInt64(ids)).Select("id", "post_id", "cover_url", "video_url", "dash_url", "preview_video_url").Find(&videos).Error; err != nil {
		return nil, err
	}
	for _, video := range videos {
		out[video.PostID] = append(out[video.PostID], video)
	}
	return out, nil
}

func (r SearchRepository) searchTagsByPostID(ctx context.Context, ids []int64) (map[int64][]domain.Tag, error) {
	out := map[int64][]domain.Tag{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		PostID int64  `gorm:"column:post_id"`
		ID     int    `gorm:"column:id"`
		Name   string `gorm:"column:name"`
	}
	err := r.db.WithContext(ctx).
		Table("post_tags").
		Select("post_tags.post_id, tags.id, tags.name").
		Joins("JOIN tags ON tags.id = post_tags.tag_id").
		Where("post_tags.post_id IN ?", uniqueInt64(ids)).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PostID] = append(out[row.PostID], domain.Tag{ID: row.ID, Name: row.Name})
	}
	return out, nil
}

func (r SearchRepository) paymentSettingsByPostID(ctx context.Context, ids []int64) (map[int64]*domain.PostPaymentSetting, error) {
	out := map[int64]*domain.PostPaymentSetting{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []domain.PostPaymentSetting
	if err := r.db.WithContext(ctx).Where("post_id IN ?", uniqueInt64(ids)).Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		out[rows[i].PostID] = &rows[i]
	}
	return out, nil
}

func (r SearchRepository) postCountsByUser(ctx context.Context, ids []int64) (map[int64]int64, error) {
	out := map[int64]int64{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		UserID int64 `gorm:"column:user_id"`
		Count  int64 `gorm:"column:count"`
	}
	if err := r.db.WithContext(ctx).Model(&domain.Post{}).Select("user_id, COUNT(*) AS count").Where("user_id IN ? AND is_draft = ?", uniqueInt64(ids), false).Group("user_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.UserID] = row.Count
	}
	return out, nil
}

func searchPostIDs(posts []domain.Post) []int64 {
	out := make([]int64, 0, len(posts))
	for _, post := range posts {
		out = append(out, post.ID)
	}
	return out
}

func searchPostUserIDs(posts []domain.Post) []int64 {
	out := make([]int64, 0, len(posts))
	for _, post := range posts {
		out = append(out, post.UserID)
	}
	return out
}

func searchUserIDs(users []domain.User) []int64 {
	out := make([]int64, 0, len(users))
	for _, user := range users {
		out = append(out, user.ID)
	}
	return out
}
