package repositories

import (
	"context"
	"sort"
	"sync"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func (r ContentRepository) standardPosts(ctx context.Context, opts PostListOptions) (*PostListResult, error) {
	if opts.PublicAccessOnly {
		opts.IsDraft = false
	}
	query := r.db.WithContext(ctx).Model(&domain.Post{}).Where("is_draft = ?", opts.IsDraft)
	if opts.IsDraft {
		if opts.CurrentUserID == 0 {
			return nil, ErrContentForbidden
		}
		query = query.Where("user_id = ?", opts.CurrentUserID)
	} else if opts.UserID != nil {
		query = query.Where("user_id = ?", *opts.UserID)
		if opts.CurrentUserID == *opts.UserID {
			// author can see all non-draft posts
		} else {
			mutual, err := r.areMutualFollowers(ctx, opts.CurrentUserID, *opts.UserID)
			if err != nil {
				return nil, err
			}
			if mutual {
				query = query.Where("visibility IN ?", []string{VisibilityPublic, VisibilityFriendsOnly})
			} else {
				query = query.Where("visibility = ?", VisibilityPublic)
			}
		}
	} else {
		mutualIDs, err := r.mutualFollowerIDs(ctx, opts.CurrentUserID)
		if err != nil {
			return nil, err
		}
		if opts.CurrentUserID != 0 && len(mutualIDs) > 0 {
			query = query.Where("(visibility = ? OR (visibility = ? AND user_id IN ?) OR user_id = ?)", VisibilityPublic, VisibilityFriendsOnly, mutualIDs, opts.CurrentUserID)
		} else if opts.CurrentUserID != 0 {
			query = query.Where("(visibility = ? OR user_id = ?)", VisibilityPublic, opts.CurrentUserID)
		} else {
			query = query.Where("visibility = ?", VisibilityPublic)
		}
		blocked, err := r.blockedUserIDs(ctx, opts.CurrentUserID)
		if err != nil {
			return nil, err
		}
		if len(blocked) > 0 {
			query = query.Where("user_id NOT IN ?", blocked)
		}
	}
	if opts.CategoryID != nil {
		query = query.Where("category_id = ?", *opts.CategoryID)
	}
	if opts.PublicAccessOnly {
		query = query.Where("visibility = ? AND public_access_exempt = ?", VisibilityPublic, true)
	}
	if opts.Type != nil {
		query = query.Where("type = ?", *opts.Type)
	} else if opts.ExcludeVideoGuests && opts.CurrentUserID == 0 {
		query = query.Where("(type <> ? OR public_access_exempt = ?)", PostTypeVideo, true)
	}
	return r.finishPostList(ctx, query, opts, "created_at DESC")
}

func (r ContentRepository) hotPosts(ctx context.Context, opts PostListOptions) (*PostListResult, error) {
	since := time.Now().AddDate(0, 0, -opts.TimeRangeDays)
	query := r.db.WithContext(ctx).Model(&domain.Post{}).
		Where("is_draft = ? AND visibility = ? AND created_at >= ?", false, VisibilityPublic, since)
	if opts.CategoryID != nil {
		query = query.Where("category_id = ?", *opts.CategoryID)
	}
	if opts.PublicAccessOnly {
		query = query.Where("public_access_exempt = ?", true)
	}
	if opts.Type != nil {
		query = query.Where("type = ?", *opts.Type)
	} else {
		query = query.Where("type <> ?", PostTypeVideoCenter)
	}
	if opts.ExcludeVideoGuests && opts.CurrentUserID == 0 {
		query = query.Where("(type <> ? OR public_access_exempt = ?)", PostTypeVideo, true)
	}
	blocked, err := r.blockedUserIDs(ctx, opts.CurrentUserID)
	if err != nil {
		return nil, err
	}
	if len(blocked) > 0 {
		query = query.Where("user_id NOT IN ?", blocked)
	}
	result, err := r.finishPostList(ctx, query, opts, "like_count DESC, collect_count DESC, comment_count DESC, view_count DESC")
	if err != nil {
		return nil, err
	}
	sort.SliceStable(result.Posts, func(i, j int) bool {
		return popularityScore(result.Posts[i].Post) > popularityScore(result.Posts[j].Post)
	})
	return result, nil
}

func (r ContentRepository) friendPosts(ctx context.Context, opts PostListOptions) (*PostListResult, error) {
	friends, err := r.mutualFollowerIDs(ctx, opts.CurrentUserID)
	if err != nil {
		return nil, err
	}
	hasFriends := len(friends) > 0
	if !hasFriends {
		users, err := r.recommendedUsers(ctx, opts.CurrentUserID)
		return &PostListResult{Total: 0, Posts: []PostBundle{}, HasFriends: &hasFriends, RecommendedUsers: users}, err
	}
	query := r.db.WithContext(ctx).Model(&domain.Post{}).
		Where("user_id IN ? AND is_draft = ? AND visibility IN ?", friends, false, []string{VisibilityPublic, VisibilityFriendsOnly})
	if opts.Type != nil {
		query = query.Where("type = ?", *opts.Type)
	}
	blocked, err := r.blockedUserIDs(ctx, opts.CurrentUserID)
	if err != nil {
		return nil, err
	}
	if len(blocked) > 0 {
		query = query.Where("user_id NOT IN ?", blocked)
	}
	result, err := r.finishPostList(ctx, query, opts, sortClause(opts.Sort))
	if result != nil {
		result.HasFriends = &hasFriends
	}
	return result, err
}

func (r ContentRepository) followingPosts(ctx context.Context, opts PostListOptions) (*PostListResult, error) {
	following, err := r.followingIDs(ctx, opts.CurrentUserID)
	if err != nil {
		return nil, err
	}
	hasFollowing := len(following) > 0
	if !hasFollowing {
		users, err := r.recommendedUsers(ctx, opts.CurrentUserID)
		return &PostListResult{Total: 0, Posts: []PostBundle{}, HasFollowing: &hasFollowing, RecommendedUsers: users}, err
	}
	mutual, err := r.mutualFollowerIDs(ctx, opts.CurrentUserID)
	if err != nil {
		return nil, err
	}
	query := r.db.WithContext(ctx).Model(&domain.Post{}).
		Where("user_id IN ? AND is_draft = ?", following, false).
		Where("(visibility = ? OR (visibility = ? AND user_id IN ?))", VisibilityPublic, VisibilityFriendsOnly, mutual)
	if opts.Type != nil {
		query = query.Where("type = ?", *opts.Type)
	}
	blocked, err := r.blockedUserIDs(ctx, opts.CurrentUserID)
	if err != nil {
		return nil, err
	}
	if len(blocked) > 0 {
		query = query.Where("user_id NOT IN ?", blocked)
	}
	result, err := r.finishPostList(ctx, query, opts, sortClause(opts.Sort))
	if result != nil {
		result.HasFollowing = &hasFollowing
	}
	return result, err
}

func (r ContentRepository) finishPostList(ctx context.Context, query *gorm.DB, opts PostListOptions, order string) (*PostListResult, error) {
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	var posts []domain.Post
	if err := query.Order(order).Offset((opts.Page - 1) * opts.Limit).Limit(opts.Limit).Find(&posts).Error; err != nil {
		return nil, err
	}
	bundles, err := r.postBundles(ctx, posts)
	if err != nil {
		return nil, err
	}
	return &PostListResult{Total: total, Posts: bundles}, nil
}

func (r ContentRepository) visibleCommentsQuery(ctx context.Context, currentUserID int64) *gorm.DB {
	query := r.db.WithContext(ctx).Model(&domain.Comment{})
	if currentUserID != 0 {
		return query.Where("(is_public = ? OR user_id = ?)", true, currentUserID)
	}
	return query.Where("is_public = ?", true)
}

func (r ContentRepository) commentBundles(ctx context.Context, comments []domain.Comment, currentUserID int64, withReplyCount bool) ([]CommentBundle, error) {
	if len(comments) == 0 {
		return []CommentBundle{}, nil
	}
	userIDs := make([]int64, 0, len(comments))
	commentIDs := make([]int64, 0, len(comments))
	for _, comment := range comments {
		userIDs = append(userIDs, comment.UserID)
		commentIDs = append(commentIDs, comment.ID)
	}
	users, err := r.usersByID(ctx, userIDs)
	if err != nil {
		return nil, err
	}
	liked := map[int64]bool{}
	if currentUserID != 0 {
		var likes []domain.Like
		if err := r.db.WithContext(ctx).Where("user_id = ? AND target_type = ? AND target_id IN ?", currentUserID, 2, uniqueInt64(commentIDs)).Select("target_id").Find(&likes).Error; err != nil {
			return nil, err
		}
		for _, like := range likes {
			liked[like.TargetID] = true
		}
	}
	replyCounts := map[int64]int64{}
	if withReplyCount {
		var rows []struct {
			ParentID int64 `gorm:"column:parent_id"`
			Count    int64 `gorm:"column:count"`
		}
		query := r.visibleCommentsQuery(ctx, currentUserID).Select("parent_id, COUNT(*) AS count").Where("parent_id IN ?", uniqueInt64(commentIDs)).Group("parent_id")
		if err := query.Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			replyCounts[row.ParentID] = row.Count
		}
	}
	out := make([]CommentBundle, 0, len(comments))
	for _, comment := range comments {
		out = append(out, CommentBundle{
			Comment:    comment,
			User:       users[comment.UserID],
			Liked:      liked[comment.ID],
			ReplyCount: replyCounts[comment.ID],
		})
	}
	return out, nil
}

func (r ContentRepository) postBundles(ctx context.Context, posts []domain.Post) ([]PostBundle, error) {
	if len(posts) == 0 {
		return []PostBundle{}, nil
	}
	postIDs := make([]int64, 0, len(posts))
	userIDs := make([]int64, 0, len(posts))
	categoryIDs := []int{}
	for _, post := range posts {
		postIDs = append(postIDs, post.ID)
		userIDs = append(userIDs, post.UserID)
		if post.CategoryID != nil {
			categoryIDs = append(categoryIDs, *post.CategoryID)
		}
	}
	var users map[int64]*domain.User
	var categories map[int]*domain.Category
	var images map[int64][]domain.PostImage
	var videos map[int64][]domain.PostVideo
	var attachments map[int64][]domain.PostAttachment
	var tags map[int64][]domain.Tag
	var payment map[int64]*domain.PostPaymentSetting
	if err := runParallel(
		func() error {
			var err error
			users, err = r.usersByID(ctx, userIDs)
			return err
		},
		func() error {
			var err error
			categories, err = r.categoriesByID(ctx, categoryIDs)
			return err
		},
		func() error {
			var err error
			images, err = r.imagesByPostID(ctx, postIDs)
			return err
		},
		func() error {
			var err error
			videos, err = r.videosByPostID(ctx, postIDs)
			return err
		},
		func() error {
			var err error
			attachments, err = r.attachmentsByPostID(ctx, postIDs)
			return err
		},
		func() error {
			var err error
			tags, err = r.tagsByPostID(ctx, postIDs)
			return err
		},
		func() error {
			var err error
			payment, err = r.paymentByPostID(ctx, postIDs)
			return err
		},
	); err != nil {
		return nil, err
	}
	out := make([]PostBundle, 0, len(posts))
	for _, post := range posts {
		var category *domain.Category
		if post.CategoryID != nil {
			category = categories[*post.CategoryID]
		}
		out = append(out, PostBundle{
			Post:           post,
			User:           users[post.UserID],
			Category:       category,
			Images:         images[post.ID],
			Videos:         videos[post.ID],
			Attachments:    attachments[post.ID],
			Tags:           tags[post.ID],
			PaymentSetting: payment[post.ID],
		})
	}
	return out, nil
}

func runParallel(tasks ...func() error) error {
	errs := make(chan error, len(tasks))
	var wg sync.WaitGroup
	for _, task := range tasks {
		wg.Go(func() {
			errs <- task()
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func (r ContentRepository) usersByID(ctx context.Context, ids []int64) (map[int64]*domain.User, error) {
	out := map[int64]*domain.User{}
	if len(ids) == 0 {
		return out, nil
	}
	var users []domain.User
	if err := r.db.WithContext(ctx).Where("id IN ?", uniqueInt64(ids)).Select("id", "user_id", "nickname", "avatar", "bio", "location", "follow_count", "fans_count", "like_count", "created_at", "verified").Find(&users).Error; err != nil {
		return nil, err
	}
	for i := range users {
		out[users[i].ID] = &users[i]
	}
	return out, nil
}

func (r ContentRepository) categoriesByID(ctx context.Context, ids []int) (map[int]*domain.Category, error) {
	out := map[int]*domain.Category{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []domain.Category
	if err := r.db.WithContext(ctx).Where("id IN ?", uniqueInt(ids)).Select("id", "name").Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		out[rows[i].ID] = &rows[i]
	}
	return out, nil
}

func (r ContentRepository) imagesByPostID(ctx context.Context, ids []int64) (map[int64][]domain.PostImage, error) {
	out := map[int64][]domain.PostImage{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []domain.PostImage
	if err := r.db.WithContext(ctx).Where("post_id IN ?", uniqueInt64(ids)).Order("sort_order ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PostID] = append(out[row.PostID], row)
	}
	return out, nil
}

func (r ContentRepository) videosByPostID(ctx context.Context, ids []int64) (map[int64][]domain.PostVideo, error) {
	out := map[int64][]domain.PostVideo{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []domain.PostVideo
	if err := r.db.WithContext(ctx).Where("post_id IN ?", uniqueInt64(ids)).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PostID] = append(out[row.PostID], row)
	}
	return out, nil
}

func (r ContentRepository) attachmentsByPostID(ctx context.Context, ids []int64) (map[int64][]domain.PostAttachment, error) {
	out := map[int64][]domain.PostAttachment{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []domain.PostAttachment
	if err := r.db.WithContext(ctx).Where("post_id IN ?", uniqueInt64(ids)).Order("id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PostID] = append(out[row.PostID], row)
	}
	return out, nil
}

func (r ContentRepository) tagsByPostID(ctx context.Context, ids []int64) (map[int64][]domain.Tag, error) {
	out := map[int64][]domain.Tag{}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []struct {
		PostID int64  `gorm:"column:post_id"`
		ID     int    `gorm:"column:id"`
		Name   string `gorm:"column:name"`
	}
	err := r.db.WithContext(ctx).Table("post_tags").
		Select("post_tags.post_id, tags.id, tags.name").
		Joins("JOIN tags ON tags.id = post_tags.tag_id").
		Where("post_tags.post_id IN ?", uniqueInt64(ids)).
		Order("post_tags.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PostID] = append(out[row.PostID], domain.Tag{ID: row.ID, Name: row.Name})
	}
	return out, nil
}

func (r ContentRepository) paymentByPostID(ctx context.Context, ids []int64) (map[int64]*domain.PostPaymentSetting, error) {
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

func (r ContentRepository) followingIDs(ctx context.Context, userID int64) ([]int64, error) {
	if userID == 0 {
		return []int64{}, nil
	}
	var rows []domain.Follow
	if err := r.db.WithContext(ctx).Where("follower_id = ?", userID).Select("following_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.FollowingID)
	}
	return out, nil
}

func (r ContentRepository) mutualFollowerIDs(ctx context.Context, userID int64) ([]int64, error) {
	following, err := r.followingIDs(ctx, userID)
	if err != nil || len(following) == 0 {
		return []int64{}, err
	}
	var rows []domain.Follow
	if err := r.db.WithContext(ctx).Where("follower_id IN ? AND following_id = ?", uniqueInt64(following), userID).Select("follower_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.FollowerID)
	}
	return out, nil
}

func (r ContentRepository) areMutualFollowers(ctx context.Context, userID1, userID2 int64) (bool, error) {
	if userID1 == 0 || userID2 == 0 || userID1 == userID2 {
		return false, nil
	}
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Follow{}).
		Where("(follower_id = ? AND following_id = ?) OR (follower_id = ? AND following_id = ?)", userID1, userID2, userID2, userID1).
		Count(&count).Error
	return count >= 2, err
}

func (r ContentRepository) blockedUserIDs(ctx context.Context, userID int64) ([]int64, error) {
	if userID == 0 {
		return []int64{}, nil
	}
	var rows []domain.Blacklist
	if err := r.db.WithContext(ctx).Where("blocker_id = ? OR blocked_id = ?", userID, userID).Select("blocker_id", "blocked_id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		if row.BlockerID == userID {
			out = append(out, row.BlockedID)
		} else {
			out = append(out, row.BlockerID)
		}
	}
	return uniqueInt64(out), nil
}

func (r ContentRepository) recommendedUsers(ctx context.Context, currentUserID int64) ([]RecommendedUser, error) {
	query := r.db.WithContext(ctx).Model(&domain.User{})
	if currentUserID != 0 {
		query = query.Where("id <> ?", currentUserID)
	}
	var users []domain.User
	if err := query.Where("is_active = ?", true).Select("id", "user_id", "nickname", "avatar", "bio", "fans_count", "verified").Order("fans_count DESC").Limit(10).Find(&users).Error; err != nil {
		return nil, err
	}
	counts, err := r.postCountsByUser(ctx, userIDs(users))
	if err != nil {
		return nil, err
	}
	out := make([]RecommendedUser, 0, len(users))
	for _, user := range users {
		out = append(out, RecommendedUser{User: user, PostCount: counts[user.ID]})
	}
	return out, nil
}

func (r ContentRepository) postCountsByUser(ctx context.Context, ids []int64) (map[int64]int64, error) {
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
