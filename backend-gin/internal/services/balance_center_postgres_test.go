package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

func TestPostgresBalanceMutationsAreSerialized(t *testing.T) {
	dsn := os.Getenv("TEST_POSTGRES_URL")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_URL is not configured")
	}
	baseURL, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	baseQuery := baseURL.Query()
	baseQuery.Set("connect_timeout", "5")
	baseURL.RawQuery = baseQuery.Encode()
	dsn = baseURL.String()
	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("balance_test_%d", time.Now().UnixNano())
	if err := adminDB.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		t.Fatal(err)
	}
	defer adminDB.Exec(`DROP SCHEMA "` + schema + `" CASCADE`)

	parsed, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("search_path", schema)
	parsed.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(parsed.String()), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.ExternalBalanceAccount{}, &domain.ExternalBalanceTransaction{}); err != nil {
		t.Fatal(err)
	}

	var active atomic.Int32
	var maximum atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		current := active.Add(1)
		for {
			old := maximum.Load()
			if current <= old || maximum.CompareAndSwap(old, current) {
				break
			}
		}
		time.Sleep(25 * time.Millisecond)
		active.Add(-1)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"balance": 20}})
	}))
	defer server.Close()

	service := NewBalanceCenterService(db, config.BalanceCenterConfig{APIURL: server.URL, APIKey: "test"})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for index, amount := range []float64{-2, 2} {
		wg.Add(1)
		go func(index int, amount float64) {
			defer wg.Done()
			_, err := service.ChangeBalance(context.Background(), BalanceMutationInput{
				OperationKey: fmt.Sprintf("postgres:%d", index), UserID: 9, OAuth2ID: 99, Amount: amount,
			})
			errs <- err
		}(index, amount)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	if maximum.Load() != 1 {
		t.Fatalf("maximum concurrent remote mutations = %d, want 1", maximum.Load())
	}
}
