package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

func TestBalanceCenterUserUsesNumericOAuth2IDAndAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/external/user" || r.URL.Query().Get("user_id") != "42" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.String())
		}
		if r.Header.Get("X-API-Key") != "secret" {
			t.Fatalf("X-API-Key = %q", r.Header.Get("X-API-Key"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{
			"id": 42, "username": "remote", "balance": "200.25", "vip_level": 3, "is_active": true,
		}})
	}))
	defer server.Close()

	db := newBalanceCenterTestDB(t)
	oauth2ID := int64(42)
	if err := db.Create(&domain.User{ID: 1, UserID: "local", Nickname: "Local", OAuth2ID: &oauth2ID}).Error; err != nil {
		t.Fatal(err)
	}
	service := NewBalanceCenterService(db, config.BalanceCenterConfig{APIURL: server.URL, APIKey: "secret"})
	user, gotOAuth2ID, err := service.UserByInternalID(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if gotOAuth2ID != 42 || user.Balance != 200.25 || user.VIPLevel != 3 {
		t.Fatalf("user = %#v, oauth2 = %d", user, gotOAuth2ID)
	}
}

func TestBalanceCenterUserRejectsMismatchedOAuth2ID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{
			"id": 99, "balance": 10,
		}})
	}))
	defer server.Close()

	service := NewBalanceCenterService(newBalanceCenterTestDB(t), config.BalanceCenterConfig{APIURL: server.URL, APIKey: "secret"})
	if _, err := service.User(context.Background(), 42); err == nil {
		t.Fatal("expected mismatched oauth2 id to be rejected")
	}
}

func TestBalanceCenterRequiresAPIKeyWithoutEnableSwitch(t *testing.T) {
	service := NewBalanceCenterService(newBalanceCenterTestDB(t), config.BalanceCenterConfig{APIURL: "https://user.yuelk.com"})
	if _, err := service.User(context.Background(), 42); !errors.Is(err, ErrBalanceCenterConfig) {
		t.Fatalf("error = %v, want configuration error", err)
	}
}

func TestBalanceMutationCommitsOnce(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["user_id"] != float64(42) || body["amount"] != float64(-6) {
			t.Fatalf("body = %#v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"balance": 14}})
	}))
	defer server.Close()

	db := newBalanceCenterTestDB(t)
	service := NewBalanceCenterService(db, config.BalanceCenterConfig{APIURL: server.URL, APIKey: "secret"})
	localCommits := 0
	input := BalanceMutationInput{
		OperationKey: "purchase:1", UserID: 1, OAuth2ID: 42, Amount: -6, Reason: "test",
		LocalCommit: func(tx *gorm.DB, _ *domain.ExternalBalanceTransaction) error {
			localCommits++
			return nil
		},
	}
	first, err := service.ChangeBalance(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.ChangeBalance(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || localCommits != 1 || first.BalanceAfter == nil || *first.BalanceAfter != 14 || !second.AlreadyDone {
		t.Fatalf("calls=%d commits=%d first=%#v second=%#v", calls, localCommits, first, second)
	}
}

func TestBalanceMutationCompensatesKnownLocalFailure(t *testing.T) {
	amounts := []float64{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		amount := body["amount"].(float64)
		amounts = append(amounts, amount)
		balance := 20.0
		if amount > 0 {
			balance = 26
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"balance": balance}})
	}))
	defer server.Close()

	db := newBalanceCenterTestDB(t)
	service := NewBalanceCenterService(db, config.BalanceCenterConfig{APIURL: server.URL, APIKey: "secret"})
	wantErr := errors.New("local commit failed")
	_, err := service.ChangeBalance(context.Background(), BalanceMutationInput{
		OperationKey: "purchase:2", UserID: 1, OAuth2ID: 42, Amount: -6, Reason: "test",
		LocalCommit: func(*gorm.DB, *domain.ExternalBalanceTransaction) error { return wantErr },
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("error = %v", err)
	}
	if fmt.Sprint(amounts) != "[-6 6]" {
		t.Fatalf("amounts = %v", amounts)
	}
	var record domain.ExternalBalanceTransaction
	if err := db.Where("operation_key = ?", "purchase:2").First(&record).Error; err != nil {
		t.Fatal(err)
	}
	if record.Status != ExternalBalanceCompensated {
		t.Fatalf("status = %q", record.Status)
	}
	if record.RemoteBalanceAfter == nil || *record.RemoteBalanceAfter != 26 {
		t.Fatalf("remote balance after compensation = %v", record.RemoteBalanceAfter)
	}
}

func TestManualCompensationStoresCompensatedBalance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body["amount"] != float64(5) {
			http.Error(w, fmt.Sprintf("unexpected compensation amount: %v", body["amount"]), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"balance": 55}})
	}))
	defer server.Close()

	db := newBalanceCenterTestDB(t)
	now := time.Now()
	transactionID := int64(1)
	remoteBalance := 50.0
	if err := db.Create(&domain.ExternalBalanceAccount{
		OAuth2ID: 42, UserID: 1, ActiveOperationID: &transactionID, CreatedAt: now, UpdatedAt: &now,
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&domain.ExternalBalanceTransaction{
		ID: transactionID, OperationKey: "manual-compensation", UserID: 1, OAuth2ID: 42,
		Amount: -5, Reason: "test", Status: ExternalBalanceApplied, RemoteBalanceAfter: &remoteBalance,
		CreatedAt: now, UpdatedAt: &now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	service := NewBalanceCenterService(db, config.BalanceCenterConfig{APIURL: server.URL, APIKey: "secret"})
	record, err := service.CompensateApplied(context.Background(), transactionID)
	if err != nil {
		t.Fatal(err)
	}
	if record.Status != ExternalBalanceCompensated || record.RemoteBalanceAfter == nil || *record.RemoteBalanceAfter != 55 {
		t.Fatalf("record = %#v", record)
	}
}

func TestBalanceMutationUnknownBlocksLaterOperations(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	}))
	defer server.Close()

	db := newBalanceCenterTestDB(t)
	service := NewBalanceCenterService(db, config.BalanceCenterConfig{APIURL: server.URL, APIKey: "secret"})
	_, err := service.ChangeBalance(context.Background(), BalanceMutationInput{OperationKey: "first", UserID: 1, OAuth2ID: 42, Amount: -1})
	if !errors.Is(err, ErrBalanceOperationUnknown) {
		t.Fatalf("first mutation error = %v, want unknown", err)
	}
	_, err = service.ChangeBalance(context.Background(), BalanceMutationInput{OperationKey: "second", UserID: 1, OAuth2ID: 42, Amount: 1})
	if !errors.Is(err, ErrBalanceOperationBusy) {
		t.Fatalf("second mutation error = %v, want busy", err)
	}
	if calls.Load() != 1 {
		t.Fatalf("remote calls = %d, want 1", calls.Load())
	}
}

func TestBalanceMutationsForSameOAuth2IDAreSerialized(t *testing.T) {
	var active atomic.Int32
	var maximum atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		current := active.Add(1)
		for current > maximum.Load() && !maximum.CompareAndSwap(maximum.Load(), current) {
		}
		time.Sleep(20 * time.Millisecond)
		active.Add(-1)
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "data": map[string]any{"balance": 10}})
	}))
	defer server.Close()

	service := NewBalanceCenterService(newBalanceCenterTestDB(t), config.BalanceCenterConfig{APIURL: server.URL, APIKey: "secret"})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for index, amount := range []float64{-1, 1} {
		wg.Add(1)
		go func(index int, amount float64) {
			defer wg.Done()
			_, err := service.ChangeBalance(context.Background(), BalanceMutationInput{
				OperationKey: fmt.Sprintf("parallel:%d", index), UserID: 1, OAuth2ID: 42, Amount: amount,
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

func newBalanceCenterTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.ExternalBalanceAccount{}, &domain.ExternalBalanceTransaction{}); err != nil {
		t.Fatal(err)
	}
	return db
}
