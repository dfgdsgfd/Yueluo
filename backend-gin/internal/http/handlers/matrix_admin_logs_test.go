package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/services"
)

func TestAdminLogBucketAndRankingExpressions(t *testing.T) {
	if got := adminLogBucketExpr(nil, "created_at", "hour"); got != "date_trunc('hour', created_at)" {
		t.Fatalf("postgres hour bucket = %q", got)
	}
	if got := rankingExpr(nil, "user_display_id"); got != "COALESCE(user_display_id, CAST(user_id AS TEXT))" {
		t.Fatalf("postgres user ranking = %q", got)
	}

	mysqlDB := &gorm.DB{Config: &gorm.Config{Dialector: namedDialector{name: "mysql"}}}
	if got := adminLogBucketExpr(mysqlDB, "created_at", "month"); got != "TIMESTAMP(DATE_FORMAT(created_at, '%Y-%m-01 00:00:00'))" {
		t.Fatalf("mysql month bucket = %q", got)
	}
	if got := rankingExpr(mysqlDB, "target_id"); got != "CAST(target_id AS CHAR)" {
		t.Fatalf("mysql target ranking = %q", got)
	}
	if got := rankingCastExpr(nil, "al.target_id"); got != "CAST(al.target_id AS TEXT)" {
		t.Fatalf("postgres aliased target ranking = %q", got)
	}
	if got := rankingCastExpr(mysqlDB, "al.target_id"); got != "CAST(al.target_id AS CHAR)" {
		t.Fatalf("mysql aliased target ranking = %q", got)
	}
}

type namedDialector struct {
	name string
}

func (d namedDialector) Name() string {
	return d.name
}

func (d namedDialector) Initialize(*gorm.DB) error {
	return nil
}

func (d namedDialector) Migrator(*gorm.DB) gorm.Migrator {
	return nil
}

func (d namedDialector) DataTypeOf(*schema.Field) string {
	return ""
}

func (d namedDialector) DefaultValueOf(*schema.Field) clause.Expression {
	return nil
}

func (d namedDialector) BindVarTo(clause.Writer, *gorm.Statement, any) {
}

func (d namedDialector) QuoteTo(writer clause.Writer, value string) {
	writer.WriteString(value)
}

func (d namedDialector) Explain(sql string, _ ...any) string {
	return sql
}

func TestParseAdminLogRangeCustomBucket(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/logs/access/analytics?range=custom&start=2026-01-01&end=2026-01-02&bucket=hour", nil)

	window := parseAdminLogRange(c)
	if window.Bucket != "hour" {
		t.Fatalf("bucket = %q, want hour", window.Bucket)
	}
	wantStart := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if !window.Start.Equal(wantStart) || !window.End.Equal(wantEnd) {
		t.Fatalf("range = %s to %s, want %s to %s", window.Start, window.End, wantStart, wantEnd)
	}
}

func TestParseAdminLogRangeAllStartsAtEpoch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/admin/logs/security?range=all", nil)

	window := parseAdminLogRange(c)
	if !window.Start.Equal(time.Unix(0, 0)) {
		t.Fatalf("all range start = %s, want unix epoch", window.Start)
	}
	if !window.Start.Before(window.End) {
		t.Fatalf("all range is invalid: %s to %s", window.Start, window.End)
	}
}

func TestAdminLogKeywordConditionsIncludeUserAgent(t *testing.T) {
	accessPredicate, accessArgs, ok := accessLogKeywordCondition(" Chrome ")
	if !ok {
		t.Fatalf("access keyword condition should be enabled")
	}
	if !strings.Contains(accessPredicate, "user_agent LIKE ?") || !strings.Contains(accessPredicate, "behavior LIKE ?") {
		t.Fatalf("access predicate should include UA and behavior: %s", accessPredicate)
	}
	if len(accessArgs) != 8 || accessArgs[0] != "%Chrome%" {
		t.Fatalf("access args = %#v, want 8 Chrome like args", accessArgs)
	}

	securityPredicate, securityArgs, ok := securityAuditLogKeywordCondition("bot")
	if !ok {
		t.Fatalf("security keyword condition should be enabled")
	}
	if !strings.Contains(securityPredicate, "user_agent LIKE ?") || !strings.Contains(securityPredicate, "category LIKE ?") {
		t.Fatalf("security predicate should include UA and category: %s", securityPredicate)
	}
	if len(securityArgs) != 11 || securityArgs[0] != "%bot%" {
		t.Fatalf("security args = %#v, want 11 bot like args", securityArgs)
	}

	if _, _, ok := accessLogKeywordCondition("  "); ok {
		t.Fatalf("blank access keyword should not build a condition")
	}
}

func TestBalanceAuditAnomaly(t *testing.T) {
	message := "remote response uncertain"
	tests := []struct {
		name string
		row  domain.ExternalBalanceTransaction
		want bool
	}{
		{name: "committed", row: domain.ExternalBalanceTransaction{Status: "local_committed", Attempts: 1}, want: false},
		{name: "unknown", row: domain.ExternalBalanceTransaction{Status: "unknown", Attempts: 1}, want: true},
		{name: "compensated", row: domain.ExternalBalanceTransaction{Status: "compensated", Attempts: 1}, want: true},
		{name: "retry", row: domain.ExternalBalanceTransaction{Status: "local_committed", Attempts: 2}, want: true},
		{name: "last error", row: domain.ExternalBalanceTransaction{Status: "local_committed", Attempts: 1, LastError: &message}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := balanceAuditAnomaly(tt.row); got != tt.want {
				t.Fatalf("balanceAuditAnomaly() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccessUserAgentLabels(t *testing.T) {
	iosCases := []struct {
		name string
		ua   string
		want string
	}{
		{name: "iphone safari", ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1", want: "Mobile / iOS / Safari"},
		{name: "ipad desktop mode", ua: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1", want: "Mobile / iOS / Safari"},
		{name: "chrome ios", ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 CriOS/126.0.0.0 Mobile/15E148 Safari/604.1", want: "Mobile / iOS / Chrome"},
		{name: "firefox ios", ua: "Mozilla/5.0 (iPad; CPU OS 18_0 like Mac OS X) AppleWebKit/605.1.15 FxiOS/128.0 Mobile/15E148 Safari/605.1.15", want: "Tablet / iOS / Firefox"},
		{name: "wechat ios", ua: "Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 Mobile/15E148 MicroMessenger/8.0.50", want: "Mobile / iOS / WeChat"},
	}
	for _, tt := range iosCases {
		t.Run(tt.name, func(t *testing.T) {
			if got := accessUserAgentLabel(tt.ua, "device_ua"); got != tt.want {
				t.Fatalf("label = %q, want %q", got, tt.want)
			}
		})
	}

	edgeWindows := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36 Edg/124.0"
	if got := accessUserAgentLabel(edgeWindows, "device_ua"); got != "Desktop / Windows / Edge" {
		t.Fatalf("edge label = %q", got)
	}

	bot := "Googlebot/2.1 (+http://www.google.com/bot.html)"
	if got := accessUserAgentLabel(bot, "device_ua"); got != "Bot / Other / Other" {
		t.Fatalf("bot label = %q", got)
	}
}

func TestUserAgentOSCaseExpressionClassifiesDesktopModeIPadBeforeMacOS(t *testing.T) {
	expr := userAgentOSCaseExpr("user_agent")
	iosPosition := strings.Index(expr, "THEN 'iOS'")
	macPosition := strings.Index(expr, "THEN 'macOS'")
	if iosPosition < 0 || macPosition < 0 || iosPosition >= macPosition {
		t.Fatalf("iOS classification must precede macOS classification: %s", expr)
	}
	if !strings.Contains(expr, "LIKE '%macintosh%'") || !strings.Contains(expr, "LIKE '%mobile%'") {
		t.Fatalf("desktop-mode iPad predicate missing: %s", expr)
	}
}

func TestUserAgentOSCaseExpressionAggregatesAllIOSVariants(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE ua_samples (user_agent TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	values := []string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) CriOS/126 Mobile",
		"Mozilla/5.0 (iPad; CPU OS 18_0 like Mac OS X) FxiOS/128 Mobile",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15) Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0) MicroMessenger/8.0",
	}
	for _, value := range values {
		if err := db.Exec("INSERT INTO ua_samples (user_agent) VALUES (?)", value).Error; err != nil {
			t.Fatal(err)
		}
	}
	var rows []rankingRow
	if err := db.Table("ua_samples").
		Select(userAgentOSCaseExpr("user_agent") + " AS label, COUNT(*) AS count").
		Group("label").
		Scan(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Label != "iOS" || rows[0].Count != int64(len(values)) {
		t.Fatalf("OS aggregation rows = %#v", rows)
	}
}

func TestAdminBalanceAuditLogsIncludesHistoricalCreatorEarnings(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.Post{},
		&domain.ExternalBalanceTransaction{},
		&domain.CreatorEarningsLog{},
	); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Add(-time.Minute)
	buyer := domain.User{ID: 1, UserID: "buyer", Nickname: "Buyer", CreatedAt: now}
	author := domain.User{ID: 2, UserID: "author", Nickname: "Author", CreatedAt: now}
	post := domain.Post{ID: 10, UserID: author.ID, Title: "Paid note", Type: 1, Visibility: "public", CreatedAt: now}
	for _, record := range []any{&buyer, &author, &post} {
		if err := db.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	sourceType := "post"
	reason := "content sale"
	if err := db.Create(&domain.CreatorEarningsLog{
		ID: 20, UserID: author.ID, EarningsID: 1, Amount: 8, BalanceAfter: 8,
		Type: "content_sale", SourceID: &post.ID, SourceType: &sourceType,
		BuyerID: &buyer.ID, Reason: &reason, PlatformFee: 2, CreatedAt: now,
	}).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/admin/logs/balance?range=all", nil)
	context.Set("user", &services.RequestUser{ID: 99, Type: "admin"})
	NativeHandlers{DB: db}.AdminBalanceAuditLogs(context)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			Items []struct {
				LedgerSource string  `json:"ledger_source"`
				EntryRole    string  `json:"entry_role"`
				PostTitle    *string `json:"post_title"`
				PlatformFee  float64 `json:"platform_fee"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Data.Items) != 1 || payload.Data.Items[0].LedgerSource != "creator_earnings" ||
		payload.Data.Items[0].EntryRole != "author_credit" || payload.Data.Items[0].PostTitle == nil ||
		*payload.Data.Items[0].PostTitle != "Paid note" || payload.Data.Items[0].PlatformFee != 2 {
		t.Fatalf("creator earning audit rows = %#v", payload.Data.Items)
	}
}
