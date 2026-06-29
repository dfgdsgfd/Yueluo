package storage

import (
	"reflect"
	"testing"

	"yuem-go/backend-gin/internal/domain"
)

func TestRemoteMoonCoinSchemaContract(t *testing.T) {
	if _, exists := reflect.TypeFor[domain.UserWallet]().FieldByName("MoonCoin"); exists {
		t.Fatal("UserWallet must not contain a local MoonCoin field")
	}
	wanted := map[reflect.Type]bool{
		reflect.TypeFor[*domain.ExternalBalanceAccount]():     false,
		reflect.TypeFor[*domain.ExternalBalanceTransaction](): false,
		reflect.TypeFor[*domain.Category]():                   false,
		reflect.TypeFor[*domain.ImageWatermarkTrace]():        false,
		reflect.TypeFor[*domain.AIGenerationLog]():            false,
		reflect.TypeFor[*domain.AIJob]():                      false,
		reflect.TypeFor[*domain.AIModerationLog]():            false,
		reflect.TypeFor[*domain.FileRecycleItem]():            false,
	}
	for _, model := range AutoMigrateModels() {
		if _, exists := wanted[reflect.TypeOf(model)]; exists {
			wanted[reflect.TypeOf(model)] = true
		}
	}
	for model, found := range wanted {
		if !found {
			t.Fatalf("AutoMigrateModels is missing %v", model)
		}
	}
}
