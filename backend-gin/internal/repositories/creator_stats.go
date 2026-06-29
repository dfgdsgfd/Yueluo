package repositories

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
)

func getOrCreateCreatorEarningsTx(ctx context.Context, tx *gorm.DB, userID int64) (*domain.CreatorEarnings, error) {
	var earnings domain.CreatorEarnings
	err := tx.WithContext(ctx).Where("user_id = ?", userID).First(&earnings).Error
	if err == nil {
		return &earnings, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	earnings = domain.CreatorEarnings{UserID: userID, Balance: 0, TotalEarnings: 0, WithdrawnAmount: 0}
	if err := tx.WithContext(ctx).Create(&earnings).Error; err != nil {
		return nil, err
	}
	return &earnings, nil
}

func getOrCreateUserPointsTx(ctx context.Context, tx *gorm.DB, userID int64) (*domain.UserPoints, error) {
	var points domain.UserPoints
	err := tx.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&points).Error
	if err == nil {
		return &points, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	points = domain.UserPoints{UserID: userID, Points: 0}
	if err := tx.WithContext(ctx).Create(&points).Error; err != nil {
		return nil, err
	}
	return &points, nil
}

func (r CreatorRepository) earningsSum(ctx context.Context, userID int64, since time.Time) (float64, error) {
	var row struct{ Total float64 }
	err := r.db.WithContext(ctx).Model(&domain.CreatorEarningsLog{}).Select("COALESCE(SUM(amount),0) AS total").Where("user_id = ? AND amount > 0 AND created_at >= ?", userID, since).Scan(&row).Error
	return row.Total, err
}

func (r CreatorRepository) userPostIDs(ctx context.Context, userID int64) ([]int64, error) {
	var posts []domain.Post
	if err := r.db.WithContext(ctx).Where("user_id = ? AND is_draft = ?", userID, false).Select("id").Find(&posts).Error; err != nil {
		return nil, err
	}
	return paidPostIDs(posts), nil
}

func (r CreatorRepository) countDailyInteractions(ctx context.Context, userID int64, postIDs []int64, start, end time.Time) (int64, int64, int64, int64, int64, error) {
	views, err := r.countBrowsingByPosts(ctx, userID, start, &end)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	likes, err := r.countLikesByPosts(ctx, postIDs, start, &end)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	collects, err := r.countCollectionsByPosts(ctx, postIDs, start, &end)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	comments, err := r.countCommentsByPosts(ctx, postIDs, start, &end)
	if err != nil {
		return 0, 0, 0, 0, 0, err
	}
	followers, err := r.countFollows(ctx, userID, &start, &end)
	return views, likes, collects, comments, followers, err
}

func (r CreatorRepository) countBrowsingByPosts(ctx context.Context, userID int64, start time.Time, end *time.Time) (int64, error) {
	query := r.db.WithContext(ctx).Table("browsing_history").Joins("JOIN posts ON posts.id = browsing_history.post_id").Where("posts.user_id = ? AND browsing_history.created_at >= ?", userID, start)
	if end != nil {
		query = query.Where("browsing_history.created_at < ?", *end)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (r CreatorRepository) countLikesByPosts(ctx context.Context, postIDs []int64, start time.Time, end *time.Time) (int64, error) {
	if len(postIDs) == 0 {
		return 0, nil
	}
	query := r.db.WithContext(ctx).Model(&domain.Like{}).Where("target_type = ? AND target_id IN ? AND created_at >= ?", 1, uniqueInt64(postIDs), start)
	if end != nil {
		query = query.Where("created_at < ?", *end)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (r CreatorRepository) countCollectionsByPosts(ctx context.Context, postIDs []int64, start time.Time, end *time.Time) (int64, error) {
	if len(postIDs) == 0 {
		return 0, nil
	}
	query := r.db.WithContext(ctx).Model(&domain.Collection{}).Where("post_id IN ? AND created_at >= ?", uniqueInt64(postIDs), start)
	if end != nil {
		query = query.Where("created_at < ?", *end)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (r CreatorRepository) countCommentsByPosts(ctx context.Context, postIDs []int64, start time.Time, end *time.Time) (int64, error) {
	if len(postIDs) == 0 {
		return 0, nil
	}
	query := r.db.WithContext(ctx).Model(&domain.Comment{}).Where("post_id IN ? AND created_at >= ?", uniqueInt64(postIDs), start)
	if end != nil {
		query = query.Where("created_at < ?", *end)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (r CreatorRepository) countFollows(ctx context.Context, userID int64, start, end *time.Time) (int64, error) {
	query := r.db.WithContext(ctx).Model(&domain.Follow{}).Where("following_id = ?", userID)
	if start != nil {
		query = query.Where("created_at >= ?", *start)
	}
	if end != nil {
		query = query.Where("created_at < ?", *end)
	}
	var count int64
	err := query.Count(&count).Error
	return count, err
}

func (r CreatorRepository) postTotals(ctx context.Context, userID int64) (map[string]int64, []int64, error) {
	var rows []domain.Post
	if err := r.db.WithContext(ctx).Where("user_id = ? AND is_draft = ?", userID, false).Select("id", "view_count", "like_count", "collect_count", "comment_count").Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	totals := map[string]int64{"posts": int64(len(rows)), "view_count": 0, "like_count": 0, "collect_count": 0, "comment_count": 0}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
		totals["view_count"] += row.ViewCount
		totals["like_count"] += int64(row.LikeCount)
		totals["collect_count"] += int64(row.CollectCount)
		totals["comment_count"] += int64(row.CommentCount)
	}
	return totals, ids, nil
}

type rangeSpec struct {
	Key   string
	Start time.Time
	End   *time.Time
}

func (r CreatorRepository) rangeInteractions(ctx context.Context, userID int64, postIDs []int64, ranges []rangeSpec) (map[string]map[string]int64, error) {
	out := map[string]map[string]int64{
		"views": {}, "likes": {}, "collects": {}, "comments": {},
	}
	for _, item := range ranges {
		views, err := r.countBrowsingByPosts(ctx, userID, item.Start, item.End)
		if err != nil {
			return nil, err
		}
		likes, err := r.countLikesByPosts(ctx, postIDs, item.Start, item.End)
		if err != nil {
			return nil, err
		}
		collects, err := r.countCollectionsByPosts(ctx, postIDs, item.Start, item.End)
		if err != nil {
			return nil, err
		}
		comments, err := r.countCommentsByPosts(ctx, postIDs, item.Start, item.End)
		if err != nil {
			return nil, err
		}
		out["views"][item.Key] = views
		out["likes"][item.Key] = likes
		out["collects"][item.Key] = collects
		out["comments"][item.Key] = comments
	}
	return out, nil
}

func (r CreatorRepository) usersByID(ctx context.Context, ids []int64) (map[int64]*domain.User, error) {
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

func (r CreatorRepository) postsByID(ctx context.Context, ids []int64) (map[int64]*domain.Post, error) {
	out := map[int64]*domain.Post{}
	if len(ids) == 0 {
		return out, nil
	}
	var posts []domain.Post
	if err := r.db.WithContext(ctx).Where("id IN ?", uniqueInt64(ids)).Select("id", "title").Find(&posts).Error; err != nil {
		return nil, err
	}
	for i := range posts {
		out[posts[i].ID] = &posts[i]
	}
	return out, nil
}

func (r CreatorRepository) paymentSettingsByPostID(ctx context.Context, ids []int64) (map[int64]*domain.PostPaymentSetting, error) {
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

func (r CreatorRepository) postCovers(ctx context.Context, ids []int64) (map[int64]*string, error) {
	out := map[int64]*string{}
	if len(ids) == 0 {
		return out, nil
	}
	var images []domain.PostImage
	if err := r.db.WithContext(ctx).
		Where("post_id IN ? AND is_protected = ?", uniqueInt64(ids), false).
		Order("post_id ASC, sort_order ASC, id ASC").
		Find(&images).Error; err != nil {
		return nil, err
	}
	for i := range images {
		if _, exists := out[images[i].PostID]; !exists {
			out[images[i].PostID] = &images[i].ImageURL
		}
	}
	var videos []domain.PostVideo
	if err := r.db.WithContext(ctx).Where("post_id IN ?", uniqueInt64(ids)).Order("id ASC").Find(&videos).Error; err != nil {
		return nil, err
	}
	for i := range videos {
		if _, exists := out[videos[i].PostID]; !exists && videos[i].CoverURL != nil {
			out[videos[i].PostID] = videos[i].CoverURL
		}
	}
	return out, nil
}

type salesStat struct {
	Count int64
	Total float64
}

func (r CreatorRepository) salesStats(ctx context.Context, postIDs []int64) (map[int64]salesStat, error) {
	out := map[int64]salesStat{}
	if len(postIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		PostID int64
		Count  int64
		Total  float64
	}
	err := r.db.WithContext(ctx).Model(&domain.UserPurchasedContent{}).
		Select("post_id, COUNT(*) AS count, COALESCE(SUM(price),0) AS total").
		Where("post_id IN ?", uniqueInt64(postIDs)).
		Group("post_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.PostID] = salesStat{Count: row.Count, Total: row.Total}
	}
	return out, nil
}

func (r CreatorRepository) qualityPostsByID(ctx context.Context, ids []int64) (map[int64]*QualityRewardPost, error) {
	out := map[int64]*QualityRewardPost{}
	if len(ids) == 0 {
		return out, nil
	}
	var posts []domain.Post
	if err := r.db.WithContext(ctx).Where("id IN ?", uniqueInt64(ids)).Select("id", "title", "type", "quality_level", "created_at").Find(&posts).Error; err != nil {
		return nil, err
	}
	covers, err := r.postCovers(ctx, ids)
	if err != nil {
		return nil, err
	}
	for _, post := range posts {
		out[post.ID] = &QualityRewardPost{ID: post.ID, Title: post.Title, Type: post.Type, QualityLevel: post.QualityLevel, Cover: covers[post.ID], CreatedAt: post.CreatedAt}
	}
	return out, nil
}

func (r CreatorRepository) qualityRewardStats(ctx context.Context, userID int64) ([]QualityRewardStats, error) {
	var logs []domain.CreatorEarningsLog
	if err := r.db.WithContext(ctx).Where("user_id = ? AND type = ?", userID, "quality_reward").Select("reason", "amount").Find(&logs).Error; err != nil {
		return nil, err
	}
	stats := map[string]*QualityRewardStats{}
	for _, log := range logs {
		label := qualityRewardLabel(log.Reason)
		if stats[label] == nil {
			stats[label] = &QualityRewardStats{QualityLabel: label}
		}
		stats[label].Count++
		stats[label].TotalAmount += log.Amount
	}
	out := make([]QualityRewardStats, 0, len(stats))
	for _, stat := range stats {
		stat.TotalAmount = mathRound2(stat.TotalAmount)
		out = append(out, *stat)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].QualityLabel < out[j].QualityLabel })
	return out, nil
}

const (
	qualityRewardReasonPrefix = "\u7b14\u8bb0\u8d28\u91cf\u5956\u52b1: "
	qualityRewardFallback     = "\u5176\u4ed6"
)

func qualityRewardLabel(reason *string) string {
	if reason == nil {
		return qualityRewardFallback
	}
	label, ok := strings.CutPrefix(*reason, qualityRewardReasonPrefix)
	if !ok {
		return qualityRewardFallback
	}
	label = strings.TrimSpace(label)
	if label == "" {
		return qualityRewardFallback
	}
	return label
}

func purchasePostIDs(purchases []domain.UserPurchasedContent) []int64 {
	out := make([]int64, 0, len(purchases))
	for _, purchase := range purchases {
		out = append(out, purchase.PostID)
	}
	return out
}

func paidPostIDs(posts []domain.Post) []int64 {
	out := make([]int64, 0, len(posts))
	for _, post := range posts {
		out = append(out, post.ID)
	}
	return out
}

func earningBuyerIDs(logs []domain.CreatorEarningsLog) []int64 {
	out := []int64{}
	for _, log := range logs {
		if log.BuyerID != nil {
			out = append(out, *log.BuyerID)
		}
	}
	return out
}

func earningPostSourceIDs(logs []domain.CreatorEarningsLog) []int64 {
	out := []int64{}
	for _, log := range logs {
		if log.SourceID != nil && log.SourceType != nil && *log.SourceType == "post" {
			out = append(out, *log.SourceID)
		}
	}
	return out
}

func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func extendedDetails(extended ExtendedEarnings) []string {
	details := []string{}
	if extended.Views.Count > 0 {
		details = append(details, fmt.Sprintf("\u6d4f\u89c8%d\u6b21", extended.Views.Count))
	}
	if extended.Likes.Count > 0 {
		details = append(details, fmt.Sprintf("\u70b9\u8d5e%d\u6b21", extended.Likes.Count))
	}
	if extended.Collects.Count > 0 {
		details = append(details, fmt.Sprintf("\u6536\u85cf%d\u6b21", extended.Collects.Count))
	}
	if extended.Comments.Count > 0 {
		details = append(details, fmt.Sprintf("\u8bc4\u8bba%d\u6761", extended.Comments.Count))
	}
	if extended.Followers.Count > 0 {
		details = append(details, fmt.Sprintf("\u65b0\u7c89\u4e1d%d\u4f4d", extended.Followers.Count))
	}
	return details
}
