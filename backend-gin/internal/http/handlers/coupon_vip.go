package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"yuem-go/backend-gin/internal/domain"
)

const couponVIPLookupConcurrency = 8

var couponVIPHTTPClient = &http.Client{Timeout: 8 * time.Second}

func (h NativeHandlers) filterVIPUsers(ctx context.Context, users []domain.User, minLevel int) []int64 {
	matches := make([]bool, len(users))
	sem := make(chan struct{}, couponVIPLookupConcurrency)
	var wg sync.WaitGroup
	for i, user := range users {
		if user.OAuth2ID == nil {
			continue
		}
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			wg.Wait()
			return vipMatchesInUserOrder(users, matches)
		}
		wg.Add(1)
		go func(index int, candidate domain.User) {
			defer wg.Done()
			defer func() { <-sem }()
			matches[index] = h.userMeetsVIPLevel(ctx, candidate.OAuth2ID, minLevel)
		}(i, user)
	}
	wg.Wait()
	return vipMatchesInUserOrder(users, matches)
}

func (h NativeHandlers) userMeetsVIPLevel(ctx context.Context, oauth2ID *int64, minLevel int) bool {
	if oauth2ID == nil {
		return false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.Config.Balance.APIURL+"/api/external/user?user_id="+strconv.FormatInt(*oauth2ID, 10), nil)
	if err != nil {
		return false
	}
	req.Header.Set("X-API-Key", h.Config.Balance.APIKey)
	resp, err := couponVIPHTTPClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	var body struct {
		Success bool `json:"success"`
		Data    struct {
			VIPLevel int `json:"vip_level"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&body)
	return body.Success && body.Data.VIPLevel >= minLevel
}

func vipMatchesInUserOrder(users []domain.User, matches []bool) []int64 {
	out := make([]int64, 0, len(users))
	for i, user := range users {
		if i < len(matches) && matches[i] {
			out = append(out, user.ID)
		}
	}
	return out
}
