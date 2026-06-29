package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

func TestFilterVIPUsersUsesBoundedConcurrencyAndKeepsUserOrder(t *testing.T) {
	var active atomic.Int32
	var maxActive atomic.Int32
	var keyMu sync.Mutex
	seenAPIKeys := map[string]int{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := active.Add(1)
		for {
			maxSeen := maxActive.Load()
			if current <= maxSeen || maxActive.CompareAndSwap(maxSeen, current) {
				break
			}
		}
		defer active.Add(-1)
		time.Sleep(10 * time.Millisecond)

		keyMu.Lock()
		seenAPIKeys[r.Header.Get("X-API-Key")]++
		keyMu.Unlock()

		oauthID, _ := strconv.ParseInt(r.URL.Query().Get("user_id"), 10, 64)
		vipLevel := 1
		if oauthID%2 == 0 {
			vipLevel = 2
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"success": true,
			"data": map[string]any{
				"vip_level": vipLevel,
			},
		})
	}))
	defer server.Close()

	users := make([]domain.User, 0, couponVIPLookupConcurrency*2)
	for i := 1; i <= couponVIPLookupConcurrency*2; i++ {
		oauthID := int64(i)
		users = append(users, domain.User{ID: int64(i), OAuth2ID: &oauthID})
	}
	handler := NativeHandlers{Config: config.Config{Balance: config.BalanceCenterConfig{APIURL: server.URL, APIKey: "test-key"}}}

	got := handler.filterVIPUsers(context.Background(), users, 2)
	want := []int64{2, 4, 6, 8, 10, 12, 14, 16}
	if len(got) != len(want) {
		t.Fatalf("vip users = %#v, want %#v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("vip users = %#v, want %#v", got, want)
		}
	}
	if maxActive.Load() > couponVIPLookupConcurrency {
		t.Fatalf("max concurrent lookups = %d, want <= %d", maxActive.Load(), couponVIPLookupConcurrency)
	}
	if maxActive.Load() < 2 {
		t.Fatalf("expected concurrent lookups, max active = %d", maxActive.Load())
	}
	if seenAPIKeys["test-key"] != len(users) {
		t.Fatalf("api key usage = %#v, want %d requests with test-key", seenAPIKeys, len(users))
	}
}
