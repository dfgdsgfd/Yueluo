package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"yuem-go/backend-gin/internal/config"
)

func TestBalanceCenterPostUsesSupportedDebitContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/external/balance" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if _, exists := body["request_id"]; exists {
			t.Fatalf("unsupported request_id sent: %#v", body)
		}
		for _, key := range []string{"user_id", "amount", "reason"} {
			if _, exists := body[key]; !exists {
				t.Fatalf("missing %s in %#v", key, body)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"data":{"balance":"12.50","vip_level":"2"}}`))
	}))
	defer server.Close()

	handler := NativeHandlers{Config: config.Config{Balance: config.BalanceCenterConfig{APIURL: server.URL, APIKey: "test"}}}
	result, err := handler.balanceCenterPost(context.Background(), "/api/external/balance", map[string]any{
		"user_id": "42",
		"amount":  -3.5,
		"reason":  "test",
	})
	if err != nil || !result.Success {
		t.Fatalf("result = %#v, err = %v", result, err)
	}
	if got := floatFromMap(result.Data, "balance"); got != 12.5 {
		t.Fatalf("normalized balance = %v", got)
	}
	if got := floatFromMap(result.Data, "vip_level"); got != 2 {
		t.Fatalf("normalized vip level = %v", got)
	}
}

func TestBalanceCenterConfiguredUsesRequiredConnectionSettings(t *testing.T) {
	if balanceCenterConfigured(config.BalanceCenterConfig{APIURL: "https://user.yuelk.com"}) {
		t.Fatal("balance center must require an API key")
	}
	if balanceCenterConfigured(config.BalanceCenterConfig{APIKey: "secret"}) {
		t.Fatal("balance center must require an API URL")
	}
	if !balanceCenterConfigured(config.BalanceCenterConfig{APIURL: "https://user.yuelk.com", APIKey: "secret"}) {
		t.Fatal("balance center should be available when URL and API key are configured")
	}
}
