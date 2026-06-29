package localization

import (
	"net/http"
	"strings"
)

const Default = "en"

var Supported = []string{"en", "zh-CN", "zh-TW", "vi", "ja", "ko"}

func Normalize(value string) string {
	value = strings.TrimSpace(value)
	for _, locale := range Supported {
		if strings.EqualFold(value, locale) {
			return locale
		}
	}
	lower := strings.ToLower(value)
	switch {
	case strings.HasPrefix(lower, "zh-tw"), strings.HasPrefix(lower, "zh-hk"), strings.HasPrefix(lower, "zh-hant"):
		return "zh-TW"
	case strings.HasPrefix(lower, "zh"):
		return "zh-CN"
	case strings.HasPrefix(lower, "vi"):
		return "vi"
	case strings.HasPrefix(lower, "ja"):
		return "ja"
	case strings.HasPrefix(lower, "ko"):
		return "ko"
	case strings.HasPrefix(lower, "en"):
		return "en"
	default:
		return ""
	}
}

func ResolveRequest(r *http.Request) string {
	if r == nil {
		return Default
	}
	for _, candidate := range []string{r.URL.Query().Get("locale"), r.Header.Get("X-App-Locale")} {
		if locale := Normalize(candidate); locale != "" {
			return locale
		}
	}
	if cookie, err := r.Cookie("xse.locale"); err == nil {
		if locale := Normalize(cookie.Value); locale != "" {
			return locale
		}
	}
	for item := range strings.SplitSeq(r.Header.Get("Accept-Language"), ",") {
		candidate := strings.TrimSpace(strings.SplitN(item, ";", 2)[0])
		if locale := Normalize(candidate); locale != "" {
			return locale
		}
	}
	return Default
}

func CompleteMap(value any) map[string]any {
	out := map[string]any{}
	if typed, ok := value.(map[string]any); ok {
		for _, locale := range Supported {
			if localeValue, exists := typed[locale]; exists {
				out[locale] = localeValue
			}
		}
	}
	fallback := value
	if len(out) > 0 {
		fallback = out[Default]
		if fallback == nil {
			for _, locale := range Supported {
				if out[locale] != nil {
					fallback = out[locale]
					break
				}
			}
		}
	}
	for _, locale := range Supported {
		if _, exists := out[locale]; !exists {
			out[locale] = fallback
		}
	}
	return out
}

func Value(value any, locale string) any {
	locale = Normalize(locale)
	if locale == "" {
		locale = Default
	}
	if typed, ok := value.(map[string]any); ok {
		if localized, exists := typed[locale]; exists {
			return localized
		}
		if fallback, exists := typed[Default]; exists {
			return fallback
		}
		for _, candidate := range Supported {
			if fallback, exists := typed[candidate]; exists {
				return fallback
			}
		}
	}
	return value
}
