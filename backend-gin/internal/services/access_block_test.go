package services

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

func TestAccessBlockServiceMatchesIPCIDRAndUserAgentRules(t *testing.T) {
	db := newAccessBlockTestDB(t)
	rows := []domain.AccessBlockRule{
		{ID: 1, Kind: AccessBlockKindIP, MatchType: AccessBlockMatchCIDR, Pattern: "10.0.0.0/8", Enabled: true, Priority: 100, Action: AccessBlockActionStatus, StatusCode: 403},
		{ID: 2, Kind: AccessBlockKindIP, MatchType: AccessBlockMatchCIDR, Pattern: "10.1.2.0/24", Enabled: true, Priority: 100, Action: AccessBlockActionStatus, StatusCode: 444},
		{ID: 3, Kind: AccessBlockKindUA, MatchType: AccessBlockMatchUAContains, Pattern: "BadBot", Enabled: true, Priority: 50, Action: AccessBlockActionStatus, StatusCode: 429},
		{ID: 4, Kind: AccessBlockKindUA, MatchType: AccessBlockMatchUARegex, Pattern: `(?i)evil-crawler/\d+`, Enabled: true, Priority: 40, Action: AccessBlockActionRedirect, RedirectURL: "https://example.test/blocked"},
		{ID: 5, Kind: AccessBlockKindIP, MatchType: AccessBlockMatchIP, Pattern: "192.0.2.10", Enabled: false, Priority: 1, Action: AccessBlockActionStatus, StatusCode: 403},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&domain.AccessBlockRule{}).Where("id = ?", 5).Update("enabled", false).Error; err != nil {
		t.Fatal(err)
	}

	service := NewAccessBlockService(db, nil, zap.NewNop(), false)
	if err := service.Load(context.Background()); err != nil {
		t.Fatal(err)
	}

	match, ok := service.Match(AccessBlockMatchInput{IP: "10.1.2.9", UserAgent: "Mozilla/5.0"})
	if !ok || match.Rule.ID != 2 || match.Rule.StatusCode != 444 {
		t.Fatalf("specific CIDR match = %#v, %v; want rule 2", match, ok)
	}

	match, ok = service.Match(AccessBlockMatchInput{IP: "198.51.100.7", UserAgent: "Friendly BadBot"})
	if !ok || match.Rule.ID != 3 || match.Rule.StatusCode != 429 {
		t.Fatalf("UA contains match = %#v, %v; want rule 3", match, ok)
	}

	match, ok = service.Match(AccessBlockMatchInput{IP: "198.51.100.7", UserAgent: "evil-crawler/12"})
	if !ok || match.Rule.ID != 4 || match.Rule.Action != AccessBlockActionRedirect {
		t.Fatalf("UA regex match = %#v, %v; want redirect rule 4", match, ok)
	}

	if match, ok := service.Match(AccessBlockMatchInput{IP: "192.0.2.10", UserAgent: "Mozilla/5.0"}); ok {
		t.Fatalf("disabled exact IP rule matched unexpectedly: %#v", match)
	}
}

func TestAccessBlockServiceRejectsSelfLockWithoutForce(t *testing.T) {
	db := newAccessBlockTestDB(t)
	service := NewAccessBlockService(db, nil, zap.NewNop(), false)

	_, err := service.Create(context.Background(), AccessBlockRuleInput{
		Kind:       AccessBlockKindIP,
		MatchType:  AccessBlockMatchCIDR,
		Pattern:    "203.0.113.0/24",
		Action:     AccessBlockActionStatus,
		StatusCode: 444,
	}, AccessBlockMatchInput{IP: "203.0.113.9"})
	if !errors.Is(err, ErrAccessBlockSelfLock) {
		t.Fatalf("Create() error = %v, want ErrAccessBlockSelfLock", err)
	}

	rule, err := service.Create(context.Background(), AccessBlockRuleInput{
		Kind:       AccessBlockKindIP,
		MatchType:  AccessBlockMatchCIDR,
		Pattern:    "203.0.113.0/24",
		Action:     AccessBlockActionStatus,
		StatusCode: 444,
		Force:      true,
	}, AccessBlockMatchInput{IP: "203.0.113.9"})
	if err != nil {
		t.Fatalf("forced Create() error = %v", err)
	}
	if rule.ID == 0 || !rule.Enabled {
		t.Fatalf("forced rule was not persisted: %#v", rule)
	}
}

func TestAccessBlockImportParsesIPAndCIDRLines(t *testing.T) {
	source := domain.AccessBlockImportSource{
		ID:         7,
		URL:        "https://example.test/list.txt",
		Enabled:    true,
		Priority:   500,
		Action:     AccessBlockActionStatus,
		StatusCode: 444,
	}
	rules, err := parseAccessBlockImportRules("\n# comment\n203.0.113.10\n198.51.100.0/24\n203.0.113.10\n", source)
	if err != nil {
		t.Fatalf("parseAccessBlockImportRules() error = %v", err)
	}
	if len(rules) != 2 {
		t.Fatalf("rule count = %d, want 2", len(rules))
	}
	if rules[0].MatchType != AccessBlockMatchIP || rules[0].Pattern != "203.0.113.10" {
		t.Fatalf("first rule = %#v", rules[0])
	}
	if rules[1].MatchType != AccessBlockMatchCIDR || rules[1].Pattern != "198.51.100.0/24" {
		t.Fatalf("second rule = %#v", rules[1])
	}
}

func TestAccessBlockImportSyncReplacesSourceRulesAndKeepsManualRules(t *testing.T) {
	db := newAccessBlockTestDB(t)
	body := "203.0.113.10\n198.51.100.0/24\n"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer upstream.Close()

	service := NewAccessBlockService(db, nil, zap.NewNop(), false)
	source, err := service.CreateImportSource(context.Background(), AccessBlockImportSourceInput{
		URL:                   upstream.URL,
		UpdateIntervalSeconds: 3600,
		Action:                AccessBlockActionStatus,
		StatusCode:            444,
	})
	if err != nil {
		t.Fatalf("CreateImportSource() error = %v", err)
	}
	_, err = service.Create(context.Background(), AccessBlockRuleInput{
		Kind:       AccessBlockKindIP,
		MatchType:  AccessBlockMatchIP,
		Pattern:    "192.0.2.44",
		Action:     AccessBlockActionStatus,
		StatusCode: 403,
		Force:      true,
	}, AccessBlockMatchInput{})
	if err != nil {
		t.Fatalf("Create() manual rule error = %v", err)
	}
	result, err := service.SyncImportSource(context.Background(), source.ID, AccessBlockImportSyncOptions{Manual: true, Force: true})
	if err != nil {
		t.Fatalf("SyncImportSource() error = %v", err)
	}
	if result.Count != 2 {
		t.Fatalf("sync count = %d, want 2", result.Count)
	}

	body = "198.51.100.0/24\n"
	result, err = service.SyncImportSource(context.Background(), source.ID, AccessBlockImportSyncOptions{Manual: true, Force: true})
	if err != nil {
		t.Fatalf("second SyncImportSource() error = %v", err)
	}
	if result.Count != 1 {
		t.Fatalf("second sync count = %d, want 1", result.Count)
	}
	rules, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 2 {
		t.Fatalf("stored rules = %#v, want manual + one imported", rules)
	}
	foundManual := false
	foundImported := false
	for _, rule := range rules {
		if rule.ImportSourceID == 0 && rule.Pattern == "192.0.2.44" {
			foundManual = true
		}
		if rule.ImportSourceID == source.ID && rule.Pattern == "198.51.100.0/24" {
			foundImported = true
		}
		if rule.Pattern == "203.0.113.10" {
			t.Fatalf("removed upstream rule is still present: %#v", rules)
		}
	}
	if !foundManual || !foundImported {
		t.Fatalf("manual/imported rules missing: manual=%v imported=%v rules=%#v", foundManual, foundImported, rules)
	}
}

func TestAccessBlockImportSyncFailurePreservesExistingRules(t *testing.T) {
	db := newAccessBlockTestDB(t)
	body := "203.0.113.10\n"
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	defer upstream.Close()

	service := NewAccessBlockService(db, nil, zap.NewNop(), false)
	source, err := service.CreateImportSource(context.Background(), AccessBlockImportSourceInput{
		URL:                   upstream.URL,
		UpdateIntervalSeconds: 3600,
		Action:                AccessBlockActionStatus,
		StatusCode:            444,
	})
	if err != nil {
		t.Fatalf("CreateImportSource() error = %v", err)
	}
	if _, err := service.SyncImportSource(context.Background(), source.ID, AccessBlockImportSyncOptions{Manual: true, Force: true}); err != nil {
		t.Fatalf("initial SyncImportSource() error = %v", err)
	}
	body = "not-an-ip\n"
	if _, err := service.SyncImportSource(context.Background(), source.ID, AccessBlockImportSyncOptions{Manual: true, Force: true}); !errors.Is(err, ErrAccessBlockValidation) {
		t.Fatalf("failed SyncImportSource() error = %v, want ErrAccessBlockValidation", err)
	}
	var count int64
	if err := db.Model(&domain.AccessBlockRule{}).Where("import_source_id = ? AND pattern = ?", source.ID, "203.0.113.10").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("old imported rule count = %d, want 1", count)
	}
}

func TestAccessBlockImportManualSyncRejectsSelfLockUnlessForced(t *testing.T) {
	db := newAccessBlockTestDB(t)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("203.0.113.0/24\n"))
	}))
	defer upstream.Close()

	service := NewAccessBlockService(db, nil, zap.NewNop(), false)
	source, err := service.CreateImportSource(context.Background(), AccessBlockImportSourceInput{
		URL:                   upstream.URL,
		UpdateIntervalSeconds: 3600,
		Action:                AccessBlockActionStatus,
		StatusCode:            444,
	})
	if err != nil {
		t.Fatalf("CreateImportSource() error = %v", err)
	}
	_, err = service.SyncImportSource(context.Background(), source.ID, AccessBlockImportSyncOptions{
		Manual:  true,
		Current: AccessBlockMatchInput{IP: "203.0.113.9"},
	})
	if !errors.Is(err, ErrAccessBlockSelfLock) {
		t.Fatalf("manual sync error = %v, want ErrAccessBlockSelfLock", err)
	}
	if _, err := service.SyncImportSource(context.Background(), source.ID, AccessBlockImportSyncOptions{
		Manual:  true,
		Force:   true,
		Current: AccessBlockMatchInput{IP: "203.0.113.9"},
	}); err != nil {
		t.Fatalf("forced manual sync error = %v", err)
	}
}

func newAccessBlockTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&domain.AccessBlockImportSource{}, &domain.AccessBlockRule{}); err != nil {
		t.Fatal(err)
	}
	return db
}
