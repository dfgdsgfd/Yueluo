package repositories

import (
	"context"
	"errors"
	"sort"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

type RecommendationWeights struct {
	BrowseWeight             float64
	HistoryWeight            float64
	ClickWeight              float64
	LikeWeight               float64
	CollectWeight            float64
	ViewWeight               float64
	CategoryWeight           float64
	TagWeight                float64
	TitleWeight              float64
	ContentWeight            float64
	FollowingWeight          float64
	MutualFollowWeight       float64
	PopularityWeight         float64
	PopularityLikeWeight     float64
	PopularityCollectWeight  float64
	PopularityCommentWeight  float64
	PopularityViewWeight     float64
	InterestWeight           float64
	FreshnessWeight          float64
	TimeDecayHalfLife        float64
	BehaviorDecayHalfLife    float64
	ContentTypeBoostImage    float64
	ContentTypeBoostVideo    float64
	ContentTypeBoostExternal float64
	CandidatePoolMultiplier  int
	MaxRecommended           int
	MaxHot                   int
	FreshPoolHours           int
	FreshPoolSize            int
	NewPostBoostMax          float64
	NewPostBoostHalfLife     float64
	OriginalIncentiveBoost   float64
	AdminBoostWeight         float64
	SuppressedPenalty        float64
	PinnedBoost              float64
}

type recommendationProfile struct {
	PreferredCategories map[int]float64
	PreferredTags       map[int]float64
	PreferredTitleTerms map[string]float64
	PreferredBodyTerms  map[string]float64
	InterestTerms       map[string]float64
	FollowingIDs        map[int64]bool
	MutualFollowIDs     map[int64]bool
	DislikedPostIDs     map[int64]bool
	Signals             map[string]int
}

type recommendationSignal struct {
	PostID    int64
	Weight    float64
	Timestamp time.Time
	Source    string
}

type postRecommendRule struct {
	BoostScore   float64
	IsPinned     bool
	IsSuppressed bool
}

type scoredPostBundle struct {
	Bundle    PostBundle
	Score     float64
	Breakdown map[string]float64
	Rule      postRecommendRule
}

var defaultRecommendationWeights = RecommendationWeights{
	BrowseWeight:             1,
	HistoryWeight:            1,
	ClickWeight:              1,
	LikeWeight:               3,
	CollectWeight:            4,
	ViewWeight:               1,
	CategoryWeight:           2,
	TagWeight:                2.5,
	TitleWeight:              2,
	ContentWeight:            1.2,
	FollowingWeight:          3,
	MutualFollowWeight:       4,
	PopularityWeight:         1.5,
	PopularityLikeWeight:     1,
	PopularityCollectWeight:  1.5,
	PopularityCommentWeight:  2,
	PopularityViewWeight:     0.5,
	InterestWeight:           2,
	FreshnessWeight:          1,
	TimeDecayHalfLife:        7,
	BehaviorDecayHalfLife:    14,
	ContentTypeBoostImage:    1.5,
	ContentTypeBoostVideo:    1.5,
	ContentTypeBoostExternal: 0.8,
	CandidatePoolMultiplier:  5,
	MaxRecommended:           500,
	MaxHot:                   300,
	FreshPoolHours:           48,
	FreshPoolSize:            50,
	NewPostBoostMax:          1000,
	NewPostBoostHalfLife:     0.5,
	OriginalIncentiveBoost:   120,
	AdminBoostWeight:         1,
	SuppressedPenalty:        -2000,
	PinnedBoost:              10000,
}

func effectiveRecommendationWeights(raw any) RecommendationWeights {
	weights := defaultRecommendationWeights
	config := recommendationConfigMap(raw)
	if len(config) == 0 {
		return weights
	}
	applyFloatWeight(config, "browse_weight", &weights.BrowseWeight)
	applyFloatWeight(config, "history_weight", &weights.HistoryWeight)
	applyFloatWeight(config, "click_weight", &weights.ClickWeight)
	applyFloatWeight(config, "like_weight", &weights.LikeWeight)
	applyFloatWeight(config, "collect_weight", &weights.CollectWeight)
	applyFloatWeight(config, "view_weight", &weights.ViewWeight)
	applyFloatWeight(config, "category_weight", &weights.CategoryWeight)
	applyFloatWeight(config, "tag_weight", &weights.TagWeight)
	applyFloatWeight(config, "title_weight", &weights.TitleWeight)
	applyFloatWeight(config, "content_weight", &weights.ContentWeight)
	applyFloatWeight(config, "following_weight", &weights.FollowingWeight)
	applyFloatWeight(config, "mutual_follow_weight", &weights.MutualFollowWeight)
	applyFloatWeight(config, "popularity_weight", &weights.PopularityWeight)
	applyFloatWeight(config, "popularity_like_weight", &weights.PopularityLikeWeight)
	applyFloatWeight(config, "popularity_collect_weight", &weights.PopularityCollectWeight)
	applyFloatWeight(config, "popularity_comment_weight", &weights.PopularityCommentWeight)
	applyFloatWeight(config, "popularity_view_weight", &weights.PopularityViewWeight)
	applyFloatWeight(config, "interest_weight", &weights.InterestWeight)
	applyFloatWeight(config, "freshness_weight", &weights.FreshnessWeight)
	applyFloatWeight(config, "time_decay_half_life", &weights.TimeDecayHalfLife)
	applyFloatWeight(config, "behavior_decay_half_life", &weights.BehaviorDecayHalfLife)
	applyFloatWeight(config, "content_type_boost_image", &weights.ContentTypeBoostImage)
	applyFloatWeight(config, "content_type_boost_video", &weights.ContentTypeBoostVideo)
	applyFloatWeight(config, "content_type_boost_external", &weights.ContentTypeBoostExternal)
	applyIntWeight(config, "candidate_pool_multiplier", &weights.CandidatePoolMultiplier)
	applyIntWeight(config, "max_recommended", &weights.MaxRecommended)
	applyIntWeight(config, "max_hot", &weights.MaxHot)
	applyIntWeight(config, "fresh_pool_hours", &weights.FreshPoolHours)
	applyIntWeight(config, "fresh_pool_size", &weights.FreshPoolSize)
	applyFloatWeight(config, "new_post_boost_max", &weights.NewPostBoostMax)
	applyFloatWeight(config, "new_post_boost_half_life", &weights.NewPostBoostHalfLife)
	applyFloatWeight(config, "original_incentive_boost", &weights.OriginalIncentiveBoost)
	applyFloatWeight(config, "admin_boost_weight", &weights.AdminBoostWeight)
	applyFloatWeight(config, "suppressed_penalty", &weights.SuppressedPenalty)
	applyFloatWeight(config, "pinned_boost", &weights.PinnedBoost)
	normalizeRecommendationWeights(&weights)
	return weights
}

func (r ContentRepository) recommendedPosts(ctx context.Context, opts PostListOptions) (*PostListResult, error) {
	weights := effectiveRecommendationWeights(opts.RecommendConfig)
	if err := r.applyUserRecommendationWeights(ctx, opts.CurrentUserID, &weights); err != nil {
		return nil, err
	}
	if len(opts.BlockedUserIDs) == 0 {
		blocked, err := r.blockedUserIDs(ctx, opts.CurrentUserID)
		if err != nil {
			return nil, err
		}
		opts.BlockedUserIDs = blocked
	}
	profile, err := r.recommendationProfile(ctx, opts.CurrentUserID, weights)
	if err != nil {
		return nil, err
	}
	query := r.recommendationBaseQuery(ctx, opts, profile)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	candidatePoolSize := recommendationCandidatePoolSize(opts.Page, opts.Limit, weights)
	var candidates []domain.Post
	if err := query.Order("like_count DESC, collect_count DESC, comment_count DESC, view_count DESC, created_at DESC").Limit(candidatePoolSize).Find(&candidates).Error; err != nil {
		return nil, err
	}
	candidates, err = r.mergeFreshRecommendationCandidates(ctx, candidates, opts, profile, weights)
	if err != nil {
		return nil, err
	}
	candidates, err = r.mergeOriginalIncentiveCandidates(ctx, candidates, opts, profile, weights)
	if err != nil {
		return nil, err
	}
	candidates, err = r.mergeAdminRecommendationCandidates(ctx, candidates, opts, profile, weights)
	if err != nil {
		return nil, err
	}
	bundles, err := r.postBundles(ctx, candidates)
	if err != nil {
		return nil, err
	}
	rules, err := r.postRecommendRules(ctx, postBundleIDsLocal(bundles), opts.CurrentUserID)
	if err != nil {
		return nil, err
	}
	scored := make([]scoredPostBundle, 0, len(bundles))
	for _, bundle := range bundles {
		score, breakdown := scoreRecommendedPost(bundle, profile, weights, rules[bundle.Post.ID])
		bundle.RecommendationScore = &score
		bundle.ScoreBreakdown = breakdown
		scored = append(scored, scoredPostBundle{Bundle: bundle, Score: score, Breakdown: breakdown, Rule: rules[bundle.Post.ID]})
	}
	sortScoredRecommendedPosts(scored)
	start := (opts.Page - 1) * opts.Limit
	end := start + opts.Limit
	posts := []PostBundle{}
	if start < len(scored) {
		if end > len(scored) {
			end = len(scored)
		}
		posts = make([]PostBundle, 0, end-start)
		for _, item := range scored[start:end] {
			posts = append(posts, item.Bundle)
		}
	}
	recommendedTotal := min(total, int64(len(scored)))
	return &PostListResult{
		Total: recommendedTotal,
		Posts: posts,
		Debug: recommendationDebug(opts, profile, weights, len(candidates), len(scored), len(posts)),
	}, nil
}

func (r ContentRepository) recommendationBaseQuery(ctx context.Context, opts PostListOptions, profile recommendationProfile) *gorm.DB {
	query := r.db.WithContext(ctx).Model(&domain.Post{}).
		Where("is_draft = ? AND visibility = ?", false, VisibilityPublic)
	if opts.PublicAccessOnly {
		query = query.Where("public_access_exempt = ?", true)
	}
	if opts.CategoryID != nil {
		query = query.Where("category_id = ?", *opts.CategoryID)
	}
	if opts.Type != nil {
		query = query.Where("type = ?", *opts.Type)
	} else {
		query = query.Where("type <> ?", PostTypeVideoCenter)
	}
	if opts.ExcludeVideoGuests && opts.CurrentUserID == 0 {
		query = query.Where("(type <> ? OR public_access_exempt = ?)", PostTypeVideo, true)
	}
	blocked := opts.BlockedUserIDs
	if len(blocked) > 0 {
		query = query.Where("user_id NOT IN ?", blocked)
	}
	if len(profile.DislikedPostIDs) > 0 {
		query = query.Where("id NOT IN ?", int64MapKeys(profile.DislikedPostIDs))
	}
	return query
}

func (r ContentRepository) mergeFreshRecommendationCandidates(ctx context.Context, candidates []domain.Post, opts PostListOptions, profile recommendationProfile, weights RecommendationWeights) ([]domain.Post, error) {
	if weights.FreshPoolHours <= 0 || weights.FreshPoolSize <= 0 {
		return candidates, nil
	}
	query := r.recommendationBaseQuery(ctx, opts, profile).Where("created_at >= ?", time.Now().Add(-time.Duration(weights.FreshPoolHours)*time.Hour))
	var fresh []domain.Post
	if err := query.Order("created_at DESC").Limit(weights.FreshPoolSize).Find(&fresh).Error; err != nil {
		return nil, err
	}
	seen := map[int64]bool{}
	for _, post := range candidates {
		seen[post.ID] = true
	}
	for _, post := range fresh {
		if !seen[post.ID] {
			candidates = append(candidates, post)
			seen[post.ID] = true
		}
	}
	return candidates, nil
}

func (r ContentRepository) mergeOriginalIncentiveCandidates(ctx context.Context, candidates []domain.Post, opts PostListOptions, profile recommendationProfile, weights RecommendationWeights) ([]domain.Post, error) {
	if weights.OriginalIncentiveBoost <= 0 {
		return candidates, nil
	}
	limit := min(max(weights.FreshPoolSize, opts.Limit), weights.MaxRecommended)
	var rewarded []domain.Post
	err := r.recommendationBaseQuery(ctx, opts, profile).
		Where("quality_reward > ?", 0).
		Order("quality_reward DESC, created_at DESC").
		Limit(limit).
		Find(&rewarded).Error
	if err != nil {
		return nil, err
	}
	return mergeRecommendationCandidates(candidates, rewarded), nil
}

func (r ContentRepository) mergeAdminRecommendationCandidates(ctx context.Context, candidates []domain.Post, opts PostListOptions, profile recommendationProfile, weights RecommendationWeights) ([]domain.Post, error) {
	now := time.Now()
	configQuery := r.db.WithContext(ctx).Model(&domain.PostRecommendConfig{}).
		Where("is_active = ? AND is_suppressed = ?", true, false).
		Where("(start_time IS NULL OR start_time <= ?) AND (end_time IS NULL OR end_time >= ?)", now, now).
		Where("(is_pinned = ? OR boost_score <> ?)", true, 0)
	if opts.CurrentUserID == 0 {
		configQuery = configQuery.Where("target_user_id IS NULL")
	} else {
		configQuery = configQuery.Where("target_user_id IS NULL OR target_user_id = ?", opts.CurrentUserID)
	}
	var configs []domain.PostRecommendConfig
	if err := configQuery.Order("is_pinned DESC, boost_score DESC, id ASC").Limit(weights.MaxRecommended).Find(&configs).Error; err != nil {
		return nil, err
	}
	if len(configs) == 0 {
		return candidates, nil
	}
	postIDs := make([]int64, 0, len(configs))
	for _, config := range configs {
		postIDs = append(postIDs, config.PostID)
	}
	var configured []domain.Post
	if err := r.recommendationBaseQuery(ctx, opts, profile).Where("id IN ?", uniqueInt64(postIDs)).Find(&configured).Error; err != nil {
		return nil, err
	}
	return mergeRecommendationCandidates(candidates, configured), nil
}

func mergeRecommendationCandidates(candidates, additional []domain.Post) []domain.Post {
	seen := make(map[int64]bool, len(candidates)+len(additional))
	for _, post := range candidates {
		seen[post.ID] = true
	}
	for _, post := range additional {
		if !seen[post.ID] {
			candidates = append(candidates, post)
			seen[post.ID] = true
		}
	}
	return candidates
}

func sortScoredRecommendedPosts(scored []scoredPostBundle) {
	sort.SliceStable(scored, func(i, j int) bool {
		leftPinned := scored[i].Rule.IsPinned && !scored[i].Rule.IsSuppressed
		rightPinned := scored[j].Rule.IsPinned && !scored[j].Rule.IsSuppressed
		if leftPinned != rightPinned {
			return leftPinned
		}
		if scored[i].Score == scored[j].Score {
			return scored[i].Bundle.Post.CreatedAt.After(scored[j].Bundle.Post.CreatedAt)
		}
		return scored[i].Score > scored[j].Score
	})
}

func (r ContentRepository) applyUserRecommendationWeights(ctx context.Context, userID int64, weights *RecommendationWeights) error {
	if userID == 0 {
		return nil
	}
	var row domain.RecommendConfig
	err := r.db.WithContext(ctx).Where("user_id = ? AND is_active = ?", userID, true).Take(&row).Error
	if err == nil {
		weights.LikeWeight = row.LikeWeight
		weights.CollectWeight = row.CollectWeight
		weights.ViewWeight = row.ViewWeight
		weights.CategoryWeight = row.CategoryWeight
		weights.TagWeight = row.TagWeight
		weights.FollowingWeight = row.FollowingWeight
		weights.MutualFollowWeight = row.MutualFollowWeight
		weights.PopularityWeight = row.PopularityWeight
		weights.InterestWeight = row.InterestWeight
		weights.TimeDecayHalfLife = float64(row.TimeDecayHalfLife)
		normalizeRecommendationWeights(weights)
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	return err
}
