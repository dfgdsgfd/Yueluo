package repositories

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func (r ContentRepository) recommendationProfile(ctx context.Context, userID int64, weights RecommendationWeights) (recommendationProfile, error) {
	profile := recommendationProfile{
		PreferredCategories: map[int]float64{},
		PreferredTags:       map[int]float64{},
		PreferredTitleTerms: map[string]float64{},
		PreferredBodyTerms:  map[string]float64{},
		InterestTerms:       map[string]float64{},
		FollowingIDs:        map[int64]bool{},
		MutualFollowIDs:     map[int64]bool{},
		DislikedPostIDs:     map[int64]bool{},
		Signals:             map[string]int{},
	}
	if userID == 0 {
		return profile, nil
	}
	signals, err := r.userRecommendationSignals(ctx, userID, weights)
	if err != nil {
		return profile, err
	}
	for _, signal := range signals {
		profile.Signals[signal.Source]++
	}
	following, err := r.followingIDs(ctx, userID)
	if err != nil {
		return profile, err
	}
	for _, id := range following {
		profile.FollowingIDs[id] = true
	}
	mutual, err := r.mutualFollowerIDs(ctx, userID)
	if err != nil {
		return profile, err
	}
	for _, id := range mutual {
		profile.MutualFollowIDs[id] = true
	}
	var dislikes []domain.Dislike
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Select("post_id").Limit(1000).Find(&dislikes).Error; err != nil {
		return profile, err
	}
	for _, row := range dislikes {
		profile.DislikedPostIDs[row.PostID] = true
	}
	if err := r.addUserInterestTerms(ctx, userID, &profile); err != nil {
		return profile, err
	}
	if len(signals) == 0 {
		return profile, nil
	}
	ids := make([]int64, 0, len(signals))
	signalWeights := map[int64]float64{}
	for _, signal := range signals {
		ids = append(ids, signal.PostID)
		signalWeights[signal.PostID] += signal.Weight * timeDecay(signal.Timestamp, weights.BehaviorDecayHalfLife)
	}
	var posts []domain.Post
	if err := r.db.WithContext(ctx).Where("id IN ?", uniqueInt64(ids)).Find(&posts).Error; err != nil {
		return profile, err
	}
	bundles, err := r.postBundles(ctx, posts)
	if err != nil {
		return profile, err
	}
	for _, bundle := range bundles {
		weight := signalWeights[bundle.Post.ID]
		if weight <= 0 {
			continue
		}
		if bundle.Post.CategoryID != nil {
			profile.PreferredCategories[*bundle.Post.CategoryID] += weight
		}
		for _, tag := range bundle.Tags {
			profile.PreferredTags[tag.ID] += weight
			addWeightedTokens(profile.PreferredTitleTerms, tag.Name, weight)
		}
		addWeightedTokens(profile.PreferredTitleTerms, bundle.Post.Title, weight)
		addWeightedTokens(profile.PreferredBodyTerms, bundle.Post.Content, weight)
	}
	limitTermMap(profile.PreferredTitleTerms, 300)
	limitTermMap(profile.PreferredBodyTerms, 300)
	return profile, nil
}

func (r ContentRepository) userRecommendationSignals(ctx context.Context, userID int64, weights RecommendationWeights) ([]recommendationSignal, error) {
	signals := []recommendationSignal{}
	var likes []domain.Like
	if err := r.db.WithContext(ctx).Where("user_id = ? AND target_type = ?", userID, 1).Select("target_id", "created_at").Order("created_at DESC").Limit(200).Find(&likes).Error; err != nil {
		return nil, err
	}
	for _, row := range likes {
		signals = append(signals, recommendationSignal{PostID: row.TargetID, Weight: weights.LikeWeight, Timestamp: row.CreatedAt, Source: "likes"})
	}
	var collections []domain.Collection
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Select("post_id", "created_at").Order("created_at DESC").Limit(200).Find(&collections).Error; err != nil {
		return nil, err
	}
	for _, row := range collections {
		signals = append(signals, recommendationSignal{PostID: row.PostID, Weight: weights.CollectWeight, Timestamp: row.CreatedAt, Source: "collections"})
	}
	var history []domain.BrowsingHistory
	if err := r.db.WithContext(ctx).Where("user_id = ?", userID).Select("post_id", "created_at", "updated_at").Order("updated_at DESC").Limit(300).Find(&history).Error; err != nil {
		return nil, err
	}
	historyWeight := weights.BrowseWeight + weights.HistoryWeight + weights.ClickWeight + weights.ViewWeight
	if historyWeight <= 0 {
		historyWeight = weights.ViewWeight
	}
	for _, row := range history {
		at := row.CreatedAt
		if row.UpdatedAt != nil {
			at = *row.UpdatedAt
		}
		signals = append(signals, recommendationSignal{PostID: row.PostID, Weight: historyWeight, Timestamp: at, Source: "browsing_history"})
	}
	return signals, nil
}

func (r ContentRepository) addUserInterestTerms(ctx context.Context, userID int64, profile *recommendationProfile) error {
	var user domain.User
	err := r.db.WithContext(ctx).Where("id = ?", userID).Select("id", "interests").Take(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	var raw any
	if len(user.Interests) == 0 || json.Unmarshal(user.Interests, &raw) != nil {
		return nil
	}
	addInterestValue(profile.InterestTerms, raw)
	limitTermMap(profile.InterestTerms, 100)
	return nil
}

func (r ContentRepository) postRecommendRules(ctx context.Context, postIDs []int64, userID int64) (map[int64]postRecommendRule, error) {
	out := map[int64]postRecommendRule{}
	if len(postIDs) == 0 {
		return out, nil
	}
	now := time.Now()
	var rows []domain.PostRecommendConfig
	err := r.db.WithContext(ctx).Where("post_id IN ? AND is_active = ?", uniqueInt64(postIDs), true).
		Where("(start_time IS NULL OR start_time <= ?) AND (end_time IS NULL OR end_time >= ?)", now, now).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		if row.TargetUserID != nil && (userID == 0 || *row.TargetUserID != userID) {
			continue
		}
		out[row.PostID] = postRecommendRule{BoostScore: row.BoostScore, IsPinned: row.IsPinned, IsSuppressed: row.IsSuppressed}
	}
	return out, nil
}
