package repositories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func TestEffectiveRecommendationWeightsParsesDatabaseConfig(t *testing.T) {
	weights := effectiveRecommendationWeights(`{"title_weight":9,"content_weight":"3.5","candidate_pool_multiplier":2,"max_recommended":7,"new_post_boost_max":0}`)

	if weights.TitleWeight != 9 {
		t.Fatalf("TitleWeight = %v, want 9", weights.TitleWeight)
	}
	if weights.ContentWeight != 3.5 {
		t.Fatalf("ContentWeight = %v, want 3.5", weights.ContentWeight)
	}
	if weights.CandidatePoolMultiplier != 2 {
		t.Fatalf("CandidatePoolMultiplier = %v, want 2", weights.CandidatePoolMultiplier)
	}
	if weights.MaxRecommended != 7 {
		t.Fatalf("MaxRecommended = %v, want 7", weights.MaxRecommended)
	}
	if weights.NewPostBoostMax != 0 {
		t.Fatalf("NewPostBoostMax = %v, want 0", weights.NewPostBoostMax)
	}
}

func TestScoreRecommendedPostUsesPersonalizedProfile(t *testing.T) {
	weights := defaultRecommendationWeights
	weights.NewPostBoostMax = 0
	weights.TimeDecayHalfLife = 3650
	weights.FreshnessWeight = 0
	weights.ContentTypeBoostImage = 1

	now := time.Now()
	photo := PostBundle{
		Post: domain.Post{ID: 1, UserID: 10, Title: "photo composition tips", Content: "night city camera tutorial", Type: PostTypeImage, CreatedAt: now},
		Tags: []domain.Tag{{ID: 1, Name: "photo"}},
	}
	food := PostBundle{
		Post: domain.Post{ID: 2, UserID: 11, Title: "home cooking notes", Content: "simple dinner and dessert", Type: PostTypeImage, CreatedAt: now},
		Tags: []domain.Tag{{ID: 2, Name: "food"}},
	}

	photoProfile := recommendationProfile{
		PreferredCategories: map[int]float64{},
		PreferredTags:       map[int]float64{1: 10},
		PreferredTitleTerms: map[string]float64{},
		PreferredBodyTerms:  map[string]float64{},
		InterestTerms:       map[string]float64{},
		FollowingIDs:        map[int64]bool{},
		MutualFollowIDs:     map[int64]bool{},
		DislikedPostIDs:     map[int64]bool{},
	}
	addWeightedTokens(photoProfile.PreferredTitleTerms, "photo camera composition", 5)
	foodProfile := recommendationProfile{
		PreferredCategories: map[int]float64{},
		PreferredTags:       map[int]float64{2: 10},
		PreferredTitleTerms: map[string]float64{},
		PreferredBodyTerms:  map[string]float64{},
		InterestTerms:       map[string]float64{},
		FollowingIDs:        map[int64]bool{},
		MutualFollowIDs:     map[int64]bool{},
		DislikedPostIDs:     map[int64]bool{},
	}
	addWeightedTokens(foodProfile.PreferredTitleTerms, "cooking dinner dessert", 5)

	photoScoreForPhotoUser, _ := scoreRecommendedPost(photo, photoProfile, weights, postRecommendRule{})
	foodScoreForPhotoUser, _ := scoreRecommendedPost(food, photoProfile, weights, postRecommendRule{})
	if photoScoreForPhotoUser <= foodScoreForPhotoUser {
		t.Fatalf("photo user score photo=%v food=%v, want photo higher", photoScoreForPhotoUser, foodScoreForPhotoUser)
	}

	photoScoreForFoodUser, _ := scoreRecommendedPost(photo, foodProfile, weights, postRecommendRule{})
	foodScoreForFoodUser, _ := scoreRecommendedPost(food, foodProfile, weights, postRecommendRule{})
	if foodScoreForFoodUser <= photoScoreForFoodUser {
		t.Fatalf("food user score food=%v photo=%v, want food higher", foodScoreForFoodUser, photoScoreForFoodUser)
	}
}

func TestScoreRecommendedPostAppliesAdminRules(t *testing.T) {
	weights := defaultRecommendationWeights
	weights.NewPostBoostMax = 0
	weights.PinnedBoost = 500
	weights.SuppressedPenalty = -99
	bundle := PostBundle{Post: domain.Post{ID: 1, UserID: 10, Title: "test", Type: PostTypeImage, CreatedAt: time.Now()}}
	profile := recommendationProfile{
		PreferredCategories: map[int]float64{},
		PreferredTags:       map[int]float64{},
		PreferredTitleTerms: map[string]float64{},
		PreferredBodyTerms:  map[string]float64{},
		InterestTerms:       map[string]float64{},
		FollowingIDs:        map[int64]bool{},
		MutualFollowIDs:     map[int64]bool{},
		DislikedPostIDs:     map[int64]bool{},
	}

	suppressed, suppressedBreakdown := scoreRecommendedPost(bundle, profile, weights, postRecommendRule{IsSuppressed: true})
	if suppressed != -99 || suppressedBreakdown["finalScore"] != -99 {
		t.Fatalf("suppressed score = %v breakdown=%#v, want -99", suppressed, suppressedBreakdown)
	}

	pinned, breakdown := scoreRecommendedPost(bundle, profile, weights, postRecommendRule{IsPinned: true, BoostScore: 10})
	if breakdown["adminBoost"] != 510 {
		t.Fatalf("adminBoost = %v, want 510", breakdown["adminBoost"])
	}
	if pinned < 500 {
		t.Fatalf("pinned score = %v, want pinned boost applied", pinned)
	}
}

func TestScoreRecommendedPostBoostsOriginalIncentive(t *testing.T) {
	weights := defaultRecommendationWeights
	weights.NewPostBoostMax = 0
	weights.FreshnessWeight = 0
	weights.TimeDecayHalfLife = 3650
	weights.ContentTypeBoostImage = 1
	weights.OriginalIncentiveBoost = 80
	profile := recommendationProfile{
		PreferredCategories: map[int]float64{},
		PreferredTags:       map[int]float64{},
		PreferredTitleTerms: map[string]float64{},
		PreferredBodyTerms:  map[string]float64{},
		InterestTerms:       map[string]float64{},
		FollowingIDs:        map[int64]bool{},
		MutualFollowIDs:     map[int64]bool{},
		DislikedPostIDs:     map[int64]bool{},
	}
	now := time.Now()
	plain := PostBundle{Post: domain.Post{ID: 1, UserID: 10, Title: "test", Type: PostTypeImage, CreatedAt: now}}
	amount := 6.0
	rewarded := plain
	rewarded.Post.ID = 2
	rewarded.Post.QualityReward = &amount

	plainScore, plainBreakdown := scoreRecommendedPost(plain, profile, weights, postRecommendRule{})
	rewardedScore, rewardedBreakdown := scoreRecommendedPost(rewarded, profile, weights, postRecommendRule{})

	if rewardedBreakdown["originalIncentiveBoost"] <= 0 {
		t.Fatalf("originalIncentiveBoost = %v, want positive", rewardedBreakdown["originalIncentiveBoost"])
	}
	if plainBreakdown["originalIncentiveBoost"] != 0 {
		t.Fatalf("plain originalIncentiveBoost = %v, want 0", plainBreakdown["originalIncentiveBoost"])
	}
	if rewardedScore <= plainScore {
		t.Fatalf("rewarded score = %v, plain score = %v, want rewarded higher", rewardedScore, plainScore)
	}
}

func TestSortScoredRecommendedPostsKeepsPinnedFirst(t *testing.T) {
	now := time.Now()
	scored := []scoredPostBundle{
		{Bundle: PostBundle{Post: domain.Post{ID: 1, CreatedAt: now}}, Score: 100000},
		{Bundle: PostBundle{Post: domain.Post{ID: 2, CreatedAt: now.Add(-time.Hour)}}, Score: 1, Rule: postRecommendRule{IsPinned: true}},
		{Bundle: PostBundle{Post: domain.Post{ID: 3, CreatedAt: now.Add(time.Hour)}}, Score: -100, Rule: postRecommendRule{IsPinned: true, IsSuppressed: true}},
	}

	sortScoredRecommendedPosts(scored)

	if got := scored[0].Bundle.Post.ID; got != 2 {
		t.Fatalf("first post = %d, want pinned post 2", got)
	}
	if got := scored[1].Bundle.Post.ID; got != 1 {
		t.Fatalf("second post = %d, want high-scoring unpinned post 1", got)
	}
}

func TestMergeOriginalIncentiveCandidatesIncludesRewardedPost(t *testing.T) {
	db := openRecommendationTestDB(t)
	now := time.Now()
	categoryID := 9
	reward := 8.0
	posts := []domain.Post{
		{ID: 1, UserID: 10, CategoryID: &categoryID, Title: "popular", Type: PostTypeImage, Visibility: VisibilityPublic, CreatedAt: now, LikeCount: 100},
		{ID: 2, UserID: 11, CategoryID: &categoryID, Title: "rewarded", Type: PostTypeImage, Visibility: VisibilityPublic, QualityReward: &reward, CreatedAt: now.AddDate(-1, 0, 0)},
		{ID: 3, UserID: 12, Title: "other category", Type: PostTypeImage, Visibility: VisibilityPublic, QualityReward: &reward, CreatedAt: now},
	}
	if err := db.Create(&posts).Error; err != nil {
		t.Fatalf("create posts: %v", err)
	}

	repo := NewContentRepository(db)
	profile := recommendationProfile{DislikedPostIDs: map[int64]bool{}}
	merged, err := repo.mergeOriginalIncentiveCandidates(context.Background(), posts[:1], PostListOptions{CategoryID: &categoryID, Limit: 20}, profile, defaultRecommendationWeights)
	if err != nil {
		t.Fatalf("merge original incentive candidates: %v", err)
	}
	if len(merged) != 2 || merged[1].ID != 2 {
		t.Fatalf("merged candidate ids = %#v, want [1 2]", []int64{merged[0].ID, merged[len(merged)-1].ID})
	}
}

func TestMergeAdminRecommendationCandidatesIncludesEligibleOldPinnedPost(t *testing.T) {
	db := openRecommendationTestDB(t)
	now := time.Now()
	categoryID := 7
	posts := []domain.Post{
		{ID: 1, UserID: 10, CategoryID: &categoryID, Title: "popular", Type: PostTypeImage, Visibility: VisibilityPublic, CreatedAt: now, LikeCount: 100},
		{ID: 2, UserID: 11, CategoryID: &categoryID, Title: "old pinned", Type: PostTypeImage, Visibility: VisibilityPublic, CreatedAt: now.AddDate(-1, 0, 0)},
		{ID: 3, UserID: 12, Title: "other category", Type: PostTypeImage, Visibility: VisibilityPublic, CreatedAt: now},
	}
	if err := db.Create(&posts).Error; err != nil {
		t.Fatalf("create posts: %v", err)
	}
	if err := db.Create(&domain.PostRecommendConfig{PostID: 2, IsPinned: true, IsActive: true, CreatedAt: now}).Error; err != nil {
		t.Fatalf("create recommendation config: %v", err)
	}

	repo := NewContentRepository(db)
	profile := recommendationProfile{DislikedPostIDs: map[int64]bool{}}
	merged, err := repo.mergeAdminRecommendationCandidates(context.Background(), posts[:1], PostListOptions{CategoryID: &categoryID}, profile, defaultRecommendationWeights)
	if err != nil {
		t.Fatalf("merge admin candidates: %v", err)
	}
	if len(merged) != 2 || merged[1].ID != 2 {
		t.Fatalf("merged candidate ids = %#v, want [1 2]", []int64{merged[0].ID, merged[len(merged)-1].ID})
	}
}

func TestRecommendationCandidatePoolIsStableAcrossPages(t *testing.T) {
	weights := defaultRecommendationWeights
	weights.MaxRecommended = 321
	if first, later := recommendationCandidatePoolSize(1, 24, weights), recommendationCandidatePoolSize(8, 24, weights); first != 321 || later != first {
		t.Fatalf("candidate pool sizes = first:%d later:%d, want stable 321", first, later)
	}
}

func openRecommendationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := fmt.Sprintf("recommendation_%d", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.Post{}, &domain.PostRecommendConfig{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	return db
}
