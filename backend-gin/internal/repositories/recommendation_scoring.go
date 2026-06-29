package repositories

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

func scoreRecommendedPost(bundle PostBundle, profile recommendationProfile, weights RecommendationWeights, rule postRecommendRule) (float64, map[string]float64) {
	if rule.IsSuppressed {
		return weights.SuppressedPenalty, map[string]float64{"finalScore": weights.SuppressedPenalty, "adminBoost": weights.SuppressedPenalty}
	}
	post := bundle.Post
	breakdown := map[string]float64{
		"baseScore":              1,
		"categoryMatch":          0,
		"tagMatch":               0,
		"titleMatch":             0,
		"contentMatch":           0,
		"socialBoost":            0,
		"popularityScore":        0,
		"interestMatch":          0,
		"freshnessScore":         0,
		"timeDecay":              timeDecay(post.CreatedAt, weights.TimeDecayHalfLife),
		"contentTypeBoost":       contentTypeBoost(post.Type, weights),
		"adminBoost":             0,
		"newPostBoost":           0,
		"originalIncentiveBoost": 0,
	}
	score := breakdown["baseScore"]
	if post.CategoryID != nil {
		breakdown["categoryMatch"] = clamp(profile.PreferredCategories[*post.CategoryID]*weights.CategoryWeight*0.1, 0, 5)
		score += breakdown["categoryMatch"]
	}
	tagScore := 0.0
	for _, tag := range bundle.Tags {
		tagScore += profile.PreferredTags[tag.ID] * weights.TagWeight * 0.1
		tagScore += termScore(tokenizeText(tag.Name), profile.InterestTerms) * weights.InterestWeight * 0.2
	}
	breakdown["tagMatch"] = clamp(tagScore, 0, 8)
	score += breakdown["tagMatch"]

	titleTokens := tokenizeText(post.Title)
	contentTokens := tokenizeText(post.Content)
	breakdown["titleMatch"] = clamp(termScore(titleTokens, profile.PreferredTitleTerms)*weights.TitleWeight*0.2+termScore(titleTokens, profile.PreferredBodyTerms)*weights.TitleWeight*0.08, 0, 8)
	breakdown["contentMatch"] = clamp(termScore(contentTokens, profile.PreferredBodyTerms)*weights.ContentWeight*0.08+termScore(contentTokens, profile.PreferredTitleTerms)*weights.ContentWeight*0.05, 0, 6)
	breakdown["interestMatch"] = clamp((termScore(titleTokens, profile.InterestTerms)*0.4+termScore(contentTokens, profile.InterestTerms)*0.2)*weights.InterestWeight, 0, 6)
	score += breakdown["titleMatch"] + breakdown["contentMatch"] + breakdown["interestMatch"]

	if profile.MutualFollowIDs[post.UserID] {
		breakdown["socialBoost"] = weights.MutualFollowWeight
	} else if profile.FollowingIDs[post.UserID] {
		breakdown["socialBoost"] = weights.FollowingWeight
	}
	score += breakdown["socialBoost"]

	rawPopularity := float64(post.LikeCount)*weights.PopularityLikeWeight +
		float64(post.CollectCount)*weights.PopularityCollectWeight +
		float64(post.CommentCount)*weights.PopularityCommentWeight +
		math.Log10(float64(post.ViewCount)+1)*weights.PopularityViewWeight
	breakdown["popularityScore"] = math.Log10(rawPopularity+1) * weights.PopularityWeight
	score += breakdown["popularityScore"]
	breakdown["freshnessScore"] = weights.FreshnessWeight * breakdown["timeDecay"]
	score += breakdown["freshnessScore"]
	score *= breakdown["contentTypeBoost"]
	score *= breakdown["timeDecay"]

	if weights.NewPostBoostMax > 0 {
		breakdown["newPostBoost"] = weights.NewPostBoostMax * timeDecay(post.CreatedAt, weights.NewPostBoostHalfLife)
		score += breakdown["newPostBoost"]
	}
	if weights.OriginalIncentiveBoost > 0 && post.QualityReward != nil && *post.QualityReward > 0 {
		breakdown["originalIncentiveBoost"] = weights.OriginalIncentiveBoost * (1 + math.Log10(*post.QualityReward+1))
		score += breakdown["originalIncentiveBoost"]
	}
	if rule.BoostScore != 0 {
		breakdown["adminBoost"] += rule.BoostScore * weights.AdminBoostWeight
	}
	if rule.IsPinned {
		breakdown["adminBoost"] += weights.PinnedBoost
	}
	score += breakdown["adminBoost"]
	breakdown["finalScore"] = score
	return score, breakdown
}

func recommendationDebug(opts PostListOptions, profile recommendationProfile, weights RecommendationWeights, candidateCount, scoredCount, returnedCount int) map[string]any {
	return map[string]any{
		"enabled": true,
		"userId":  opts.CurrentUserID,
		"statistics": map[string]any{
			"candidatePosts":              candidateCount,
			"scoredPosts":                 scoredCount,
			"returnedPosts":               returnedCount,
			"preferredCategories":         len(profile.PreferredCategories),
			"preferredTags":               len(profile.PreferredTags),
			"preferredTitleTerms":         len(profile.PreferredTitleTerms),
			"preferredContentTerms":       len(profile.PreferredBodyTerms),
			"interestTerms":               len(profile.InterestTerms),
			"likeSignals":                 profile.Signals["likes"],
			"collectionSignals":           profile.Signals["collections"],
			"historyClickSignals":         profile.Signals["browsing_history"],
			"candidatePoolMultiplier":     weights.CandidatePoolMultiplier,
			"maxRecommendedCandidatePool": weights.MaxRecommended,
		},
	}
}

func recommendationConfigMap(raw any) map[string]any {
	switch typed := raw.(type) {
	case map[string]any:
		return typed
	case string:
		var out map[string]any
		if json.Unmarshal([]byte(typed), &out) == nil {
			return out
		}
	case []byte:
		var out map[string]any
		if json.Unmarshal(typed, &out) == nil {
			return out
		}
	}
	return map[string]any{}
}

func applyFloatWeight(config map[string]any, key string, target *float64) {
	if value, ok := numberFromAny(config[key]); ok {
		*target = value
	}
}

func applyIntWeight(config map[string]any, key string, target *int) {
	if value, ok := numberFromAny(config[key]); ok {
		*target = int(value)
	}
}

func numberFromAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, !math.IsNaN(typed) && !math.IsInf(typed, 0)
	case float32:
		value := float64(typed)
		return value, !math.IsNaN(value) && !math.IsInf(value, 0)
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0)
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(text, 64)
		return parsed, err == nil && !math.IsNaN(parsed) && !math.IsInf(parsed, 0)
	default:
		return 0, false
	}
}

func normalizeRecommendationWeights(weights *RecommendationWeights) {
	if weights.TimeDecayHalfLife <= 0 {
		weights.TimeDecayHalfLife = 7
	}
	if weights.BehaviorDecayHalfLife <= 0 {
		weights.BehaviorDecayHalfLife = 14
	}
	if weights.NewPostBoostHalfLife <= 0 {
		weights.NewPostBoostHalfLife = 0.5
	}
	if weights.CandidatePoolMultiplier < 1 {
		weights.CandidatePoolMultiplier = 1
	}
	if weights.MaxRecommended < 1 {
		weights.MaxRecommended = 100
	}
	if weights.MaxHot < 1 {
		weights.MaxHot = 100
	}
	if weights.FreshPoolHours < 0 {
		weights.FreshPoolHours = 0
	}
	if weights.FreshPoolSize < 0 {
		weights.FreshPoolSize = 0
	}
	if weights.ContentTypeBoostImage <= 0 {
		weights.ContentTypeBoostImage = 1
	}
	if weights.ContentTypeBoostVideo <= 0 {
		weights.ContentTypeBoostVideo = 1
	}
	if weights.ContentTypeBoostExternal <= 0 {
		weights.ContentTypeBoostExternal = 1
	}
}
