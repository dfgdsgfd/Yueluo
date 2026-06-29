package repositories

import (
	"sort"
	"strings"
	"unicode"
)

func recommendationCandidatePoolSize(_ int, _ int, weights RecommendationWeights) int {
	return weights.MaxRecommended
}

func contentTypeBoost(postType int, weights RecommendationWeights) float64 {
	switch postType {
	case PostTypeImage:
		return weights.ContentTypeBoostImage
	case PostTypeVideo:
		return weights.ContentTypeBoostVideo
	case PostTypeVideoCenter:
		return weights.ContentTypeBoostExternal
	default:
		return 1
	}
}

func addWeightedTokens(target map[string]float64, text string, weight float64) {
	for _, token := range tokenizeText(text) {
		target[token] += weight
	}
}

func tokenizeText(text string) []string {
	text = strings.ToLower(strings.TrimSpace(text))
	if text == "" {
		return []string{}
	}
	out := []string{}
	seen := map[string]bool{}
	word := strings.Builder{}
	flush := func() {
		if word.Len() == 0 {
			return
		}
		token := word.String()
		word.Reset()
		if len(token) >= 2 && !seen[token] {
			seen[token] = true
			out = append(out, token)
		}
	}
	var previousCJK string
	for _, r := range text {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			if isCJK(r) {
				flush()
				token := string(r)
				if !seen[token] {
					seen[token] = true
					out = append(out, token)
				}
				if previousCJK != "" {
					bigram := previousCJK + token
					if !seen[bigram] {
						seen[bigram] = true
						out = append(out, bigram)
					}
				}
				previousCJK = token
			} else {
				word.WriteRune(r)
				previousCJK = ""
			}
		default:
			flush()
			previousCJK = ""
		}
	}
	flush()
	return out
}

func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) || (r >= 0x3040 && r <= 0x30ff) || (r >= 0xac00 && r <= 0xd7af)
}

func termScore(tokens []string, weights map[string]float64) float64 {
	score := 0.0
	seen := map[string]bool{}
	for _, token := range tokens {
		if seen[token] {
			continue
		}
		seen[token] = true
		score += weights[token]
	}
	return score
}

func addInterestValue(target map[string]float64, value any) {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			addInterestValue(target, item)
		}
	case map[string]any:
		for _, item := range typed {
			addInterestValue(target, item)
		}
	case string:
		addWeightedTokens(target, typed, 1)
	}
}

func limitTermMap(values map[string]float64, limit int) {
	if limit <= 0 || len(values) <= limit {
		return
	}
	type item struct {
		Key   string
		Value float64
	}
	items := make([]item, 0, len(values))
	for key, value := range values {
		items = append(items, item{Key: key, Value: value})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Value > items[j].Value
	})
	keep := map[string]bool{}
	for _, item := range items[:limit] {
		keep[item.Key] = true
	}
	for key := range values {
		if !keep[key] {
			delete(values, key)
		}
	}
}

func postBundleIDsLocal(bundles []PostBundle) []int64 {
	out := make([]int64, 0, len(bundles))
	for _, bundle := range bundles {
		out = append(out, bundle.Post.ID)
	}
	return out
}

func int64MapKeys(values map[int64]bool) []int64 {
	out := make([]int64, 0, len(values))
	for id := range values {
		out = append(out, id)
	}
	return out
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}
