package localization

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveRequestPrecedence(t *testing.T) {
	req := httptest.NewRequest("GET", "/?locale=ja", nil)
	req.Header.Set("X-App-Locale", "ko")
	req.Header.Set("Accept-Language", "vi")
	req.AddCookie(&http.Cookie{Name: "xse.locale", Value: "zh-TW"})
	if got := ResolveRequest(req); got != "ja" {
		t.Fatalf("ResolveRequest() = %q, want ja", got)
	}
}

func TestCompleteMapProvidesAllLocales(t *testing.T) {
	got := CompleteMap(map[string]any{"en": "Category", "zh-CN": "分类"})
	for _, locale := range Supported {
		if got[locale] == nil {
			t.Fatalf("locale %s is missing from %#v", locale, got)
		}
	}
}
