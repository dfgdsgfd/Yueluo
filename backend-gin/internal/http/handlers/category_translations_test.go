package handlers

import (
	"encoding/json"
	"testing"

	"gorm.io/datatypes"
)

func TestNormalizedCategoryTranslationsKeepsSixSupportedLocales(t *testing.T) {
	value := normalizedCategoryTranslations(map[string]any{
		"en":    "Photography",
		"zh-CN": "摄影",
		"zh-TW": "攝影",
		"vi":    "Nhiếp ảnh",
		"ja":    "写真",
		"ko":    "사진",
		"other": "ignored",
	}, "photography", "")

	raw, ok := value.(datatypes.JSON)
	if !ok {
		t.Fatalf("expected datatypes.JSON, got %T", value)
	}
	var translations map[string]string
	if err := json.Unmarshal(raw, &translations); err != nil {
		t.Fatal(err)
	}
	if len(translations) != 6 {
		t.Fatalf("expected six translations, got %#v", translations)
	}
	if translations["en"] != "Photography" || translations["zh-CN"] != "摄影" || translations["ko"] != "사진" {
		t.Fatalf("unexpected translations: %#v", translations)
	}
	if _, exists := translations["other"]; exists {
		t.Fatalf("unsupported locale must be discarded: %#v", translations)
	}
}

func TestNormalizedCategoryTranslationsFillsMissingLocales(t *testing.T) {
	value := normalizedCategoryTranslations(map[string]any{"zh-CN": "科技"}, "technology", "")
	raw := value.(datatypes.JSON)
	var translations map[string]string
	if err := json.Unmarshal(raw, &translations); err != nil {
		t.Fatal(err)
	}
	if translations["en"] != "technology" || translations["ja"] != "technology" || translations["zh-CN"] != "科技" {
		t.Fatalf("unexpected fallback translations: %#v", translations)
	}
}
