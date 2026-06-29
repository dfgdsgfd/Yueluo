package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/security"
	"yuem-go/backend-gin/internal/services"
)

type adminQueryCaptureLogger struct {
	mu      sync.Mutex
	queries []string
}

func (l *adminQueryCaptureLogger) LogMode(logger.LogLevel) logger.Interface { return l }
func (l *adminQueryCaptureLogger) Info(context.Context, string, ...any)     {}
func (l *adminQueryCaptureLogger) Warn(context.Context, string, ...any)     {}
func (l *adminQueryCaptureLogger) Error(context.Context, string, ...any)    {}

func (l *adminQueryCaptureLogger) Trace(_ context.Context, _ time.Time, sql func() (string, int64), _ error) {
	query, _ := sql()
	l.mu.Lock()
	l.queries = append(l.queries, query)
	l.mu.Unlock()
}

func (l *adminQueryCaptureLogger) Queries() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.queries...)
}

func TestNormalizeRowHidesSensitiveAdminFields(t *testing.T) {
	tests := []struct {
		name     string
		resource adminResource
		row      map[string]any
		hidden   []string
	}{
		{
			name:     "admins",
			resource: adminResources["admins"],
			row:      map[string]any{"id": 1, "username": "root", "password": "secret"},
			hidden:   []string{"password"},
		},
		{
			name:     "users",
			resource: adminResources["users"],
			row:      map[string]any{"id": 2, "user_id": "u1", "password": "secret"},
			hidden:   []string{"password"},
		},
		{
			name:     "open-apis",
			resource: adminResources["open-apis"],
			row:      map[string]any{"id": 3, "name": "client", "api_key": "hashed", "api_key_prefix": "oapi_abc"},
			hidden:   []string{"api_key"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := (NativeHandlers{}).normalizeRow(tt.row, tt.resource)
			for _, key := range tt.hidden {
				if _, exists := got[key]; exists {
					t.Fatalf("normalizeRow exposed %q in %#v", key, got)
				}
			}
		})
	}
}

func TestAdminPreprocessCreateOpenAPIReturnsRawKeyOnce(t *testing.T) {
	body := map[string]any{"name": "client"}
	extra, err := adminPreprocessCreate(adminResources["open-apis"], body)
	if err != nil {
		t.Fatalf("adminPreprocessCreate returned error: %v", err)
	}

	rawKey, ok := extra["api_key"].(string)
	if !ok || len(rawKey) <= len("oapi_") || rawKey[:5] != "oapi_" {
		t.Fatalf("raw api key missing from one-time response: %#v", extra)
	}
	if body["api_key"] == rawKey {
		t.Fatalf("stored api_key should be hashed, got raw key")
	}
	if got := body["api_key_prefix"]; got != rawKey[:12] {
		t.Fatalf("api_key_prefix = %#v, want %q", got, rawKey[:12])
	}
	if _, exists := body["raw_api_key"]; exists {
		t.Fatalf("raw_api_key must not be persisted: %#v", body)
	}
	if got := body["api_key"].(string); len(got) != 64 {
		t.Fatalf("stored api_key hash length = %d, want 64", len(got))
	}
}

func TestAdminPreprocessCreateUserDefaultsPasswordAndActive(t *testing.T) {
	body := map[string]any{"user_id": "u1", "password": ""}
	if _, err := adminPreprocessCreate(adminResources["users"], body); err != nil {
		t.Fatalf("adminPreprocessCreate returned error: %v", err)
	}

	got, ok := body["password"].(string)
	if !ok || !security.IsArgon2idHash(got) || !security.VerifyPassword("123456", got) {
		t.Fatalf("password hash = %#v, want Argon2id default password hash", body["password"])
	}
	if got := body["is_active"]; got != true {
		t.Fatalf("is_active = %#v, want true", got)
	}
}

func TestAdminPreprocessAnnouncementTimestamps(t *testing.T) {
	body := map[string]any{"title": "公告", "content": "正文", "is_published": true, "duration_days": 3}
	if _, err := adminPreprocessCreate(adminResources["announcements"], body); err != nil {
		t.Fatalf("adminPreprocessCreate returned error: %v", err)
	}

	createdAt, hasCreatedAt := body["created_at"].(time.Time)
	updatedAt, hasUpdatedAt := body["updated_at"].(time.Time)
	publishedAt, hasPublishedAt := body["published_at"].(time.Time)
	expiresAt, hasExpiresAt := body["expires_at"].(time.Time)
	if !hasCreatedAt || createdAt.IsZero() {
		t.Fatalf("created_at missing or zero: %#v", body)
	}
	if !hasUpdatedAt || updatedAt.IsZero() {
		t.Fatalf("updated_at missing or zero: %#v", body)
	}
	if !hasPublishedAt || publishedAt.IsZero() {
		t.Fatalf("published_at missing or zero: %#v", body)
	}
	if !hasExpiresAt || expiresAt.Before(publishedAt.Add(70*time.Hour)) || expiresAt.After(publishedAt.Add(74*time.Hour)) {
		t.Fatalf("expires_at = %#v, want roughly 3 days after published_at %#v", expiresAt, publishedAt)
	}
	if _, exists := body["duration_days"]; exists {
		t.Fatalf("duration_days should not be persisted: %#v", body)
	}

	update := map[string]any{"title": "更新公告", "duration_days": 1}
	if err := adminPreprocessUpdate(adminResources["announcements"], update); err != nil {
		t.Fatalf("adminPreprocessUpdate returned error: %v", err)
	}
	if updated, ok := update["updated_at"].(time.Time); !ok || updated.IsZero() {
		t.Fatalf("update updated_at missing or zero: %#v", update)
	}
	if expires, ok := update["expires_at"].(time.Time); !ok || expires.Before(time.Now().Add(23*time.Hour)) {
		t.Fatalf("update expires_at missing or too early: %#v", update)
	}
	if _, exists := update["duration_days"]; exists {
		t.Fatalf("update duration_days should not be persisted: %#v", update)
	}
}

func TestAdminSanitizeBodyFiltersPostsToWritableColumns(t *testing.T) {
	body := map[string]any{
		"id":              849,
		"user_id":         802,
		"title":           "video title",
		"content":         "vc_external_id:10452",
		"category":        nil,
		"category_id":     nil,
		"images":          []any{"/api/file/plsc/a.webp"},
		"tags":            []any{nil},
		"nickname":        "video center",
		"user_display_id": "vc_system_user",
		"video_url":       nil,
		"cover_url":       nil,
		"quality_level":   "",
	}

	got := (NativeHandlers{}).adminSanitizeBody(adminResources["posts"], body)

	for _, key := range []string{"id", "category", "images", "tags", "nickname", "user_display_id", "video_url", "cover_url"} {
		if _, ok := got[key]; ok {
			t.Fatalf("adminSanitizeBody kept non-writable post field %q: %#v", key, got)
		}
	}
	if got["user_id"] != 802 || got["title"] != "video title" || got["content"] != "vc_external_id:10452" {
		t.Fatalf("adminSanitizeBody dropped writable post fields: %#v", got)
	}
	if got["quality_level"] != "none" {
		t.Fatalf("quality_level = %#v, want none", got["quality_level"])
	}
	if _, ok := got["category_id"]; !ok {
		t.Fatalf("category_id should remain writable even when nil: %#v", got)
	}
}

func TestAdminSanitizeBodyStoresMarkdownForConfiguredFields(t *testing.T) {
	body := map[string]any{
		"content": `<h2 style="text-align:center">公告</h2><p onclick="x()">正文 <strong>加粗</strong><script>alert(1)</script></p><p><a href="https://example.com/a">link</a><a href="javascript:alert(1)">bad</a></p>`,
		"title":   `<script>alert(1)</script>标题`,
	}

	got := (NativeHandlers{}).adminSanitizeBody(adminResources["announcements"], body)
	content := got["content"].(string)

	for _, want := range []string{
		`## 公告`,
		`正文 **加粗**`,
		`[link](https://example.com/a)`,
		`bad`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("markdown content missing %q in %s", want, content)
		}
	}
	for _, forbidden := range []string{"<h2", "<script", "onclick", "javascript:", "style="} {
		if strings.Contains(content, forbidden) {
			t.Fatalf("markdown content kept unsafe or legacy fragment %q in %s", forbidden, content)
		}
	}
	if got["title"] != `标题` {
		t.Fatalf("plain title should strip HTML/JS: %#v", got["title"])
	}
}

func TestSanitizeMarkdownDropsUnsafeHTML(t *testing.T) {
	got := sanitizeMarkdownSubmittedText(`<p style="color:#123456;background-image:url(javascript:1);text-align:right" onclick="x()">Hi<script>alert(1)</script><a href="javascript:alert(1)">bad</a><img src="data:text/html;base64,xx" onerror="x"><span style="color:rgb(var(--x))">x</span></p>`)

	for _, forbidden := range []string{"<", "script", "onclick", "javascript:", "data:text", "background-image", "rgb(var", "style="} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("sanitized markdown kept unsafe fragment %q in %s", forbidden, got)
		}
	}
	if got != "Hibadx" {
		t.Fatalf("sanitized markdown = %q, want unsafe markup stripped", got)
	}
}

func TestSanitizePostContentConvertsLegacyHTMLToMarkdown(t *testing.T) {
	rich := sanitizePostContent(`<h2>标题</h2><p onclick="x()">正文 <strong>加粗</strong><script>alert(1)</script></p>`)
	for _, want := range []string{`## 标题`, `正文 **加粗**`} {
		if !strings.Contains(rich, want) {
			t.Fatalf("markdown post content missing %q in %s", want, rich)
		}
	}
	for _, forbidden := range []string{"onclick", "<script", "</script", "<h2", "<strong"} {
		if strings.Contains(rich, forbidden) {
			t.Fatalf("markdown post content kept unsafe or legacy fragment %q in %s", forbidden, rich)
		}
	}

	plain := sanitizePostContent("第一行\n<script>alert(1)</script>")
	if plain != "第一行" {
		t.Fatalf("plain post content = %q", plain)
	}

	table := sanitizePostContent(`<table><thead><tr><th>A</th><th>B</th></tr></thead><tbody><tr><td>1</td><td>2</td></tr></tbody></table>`)
	if !strings.Contains(table, "| A") || !strings.Contains(table, "| 1") {
		t.Fatalf("table HTML should migrate to markdown table, got %s", table)
	}
}

func TestWriteAdminListLegacyShapes(t *testing.T) {
	tests := []struct {
		name      string
		resource  adminResource
		assertion func(t *testing.T, body map[string]any)
	}{
		{
			name:     "categories top-level",
			resource: adminResources["categories"],
			assertion: func(t *testing.T, body map[string]any) {
				if _, ok := body["data"].([]any); !ok {
					t.Fatalf("categories data should be top-level array: %#v", body["data"])
				}
				pagination := body["pagination"].(map[string]any)
				if pagination["totalPages"].(float64) != 3 {
					t.Fatalf("totalPages = %#v, want 3", pagination["totalPages"])
				}
			},
		},
		{
			name:     "media items",
			resource: adminResources["media-library"],
			assertion: func(t *testing.T, body map[string]any) {
				data := body["data"].(map[string]any)
				if _, ok := data["items"].([]any); !ok {
					t.Fatalf("media data.items missing: %#v", data)
				}
				if data["pageSize"].(float64) != 5 {
					t.Fatalf("media pageSize = %#v, want 5", data["pageSize"])
				}
			},
		},
		{
			name:     "reports list",
			resource: adminResources["reports"],
			assertion: func(t *testing.T, body map[string]any) {
				data := body["data"].(map[string]any)
				if _, ok := data["list"].([]any); !ok {
					t.Fatalf("reports data.list missing: %#v", data)
				}
				pagination := data["pagination"].(map[string]any)
				if pagination["pageSize"].(float64) != 5 {
					t.Fatalf("reports pageSize = %#v, want 5", pagination["pageSize"])
				}
			},
		},
		{
			name:     "audit list",
			resource: adminResources["audit"],
			assertion: func(t *testing.T, body map[string]any) {
				data := body["data"].(map[string]any)
				if _, ok := data["data"].([]any); !ok {
					t.Fatalf("audit data.data missing: %#v", data)
				}
				if data["limit"].(float64) != 5 {
					t.Fatalf("audit limit = %#v, want 5", data["limit"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rec)
			writeAdminList(ctx, tt.resource, []gin.H{{"id": 1}}, 2, 5, 11)
			var body map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			tt.assertion(t, body)
		})
	}
}

func TestAdminOrderUsesWhitelistedColumnsOnly(t *testing.T) {
	ctx := testGinContextWithQuery("/?sortField=password&sortOrder=ASC")
	if got, want := adminOrder(ctx, adminResources["users"]), "u.created_at DESC"; got != want {
		t.Fatalf("unsafe sort order = %q, want %q", got, want)
	}

	ctx = testGinContextWithQuery("/?sortField=user_id&sortOrder=ASC")
	if got, want := adminOrder(ctx, adminResources["users"]), "u.user_id ASC"; got != want {
		t.Fatalf("safe sort order = %q, want %q", got, want)
	}
}

func TestAdminAuditListCountSkipsUserJoinWhenUnfiltered(t *testing.T) {
	gin.SetMode(gin.TestMode)
	queryLogger := &adminQueryCaptureLogger{}
	db, err := gorm.Open(sqlite.Open("file:admin-audit-count?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   queryLogger,
	})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}, &domain.Audit{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	user := domain.User{ID: 1, UserID: "u1", Nickname: "User 1", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	status := 0
	audit := domain.Audit{ID: 1, UserID: user.ID, Type: 1, Content: "verification", Status: &status, CreatedAt: time.Now()}
	if err := db.Create(&audit).Error; err != nil {
		t.Fatalf("create audit: %v", err)
	}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/admin/audit?page=1&limit=5", nil)
	NativeHandlers{DB: db}.adminGenericList(ctx, adminResources["audit"])
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", rec.Code, rec.Body.String())
	}

	var countSQL string
	for _, query := range queryLogger.Queries() {
		if strings.Contains(strings.ToLower(query), "count(") {
			countSQL = query
			break
		}
	}
	if countSQL == "" {
		t.Fatalf("count query not captured: %#v", queryLogger.Queries())
	}
	if strings.Contains(strings.ToLower(countSQL), "join users") {
		t.Fatalf("audit count query should not join users when unfiltered: %s", countSQL)
	}
}

func TestAdminUsersResourceIncludesIdentityAndPointsFields(t *testing.T) {
	resource := adminResources["users"]
	for _, fragment := range []string{"u.id AS uid", "u.oauth2_id", "COALESCE(up.points, 0) AS points"} {
		if !strings.Contains(resource.Select, fragment) {
			t.Fatalf("users select missing %q: %s", fragment, resource.Select)
		}
	}
	if len(resource.Joins) != 1 || resource.Joins[0] != "LEFT JOIN user_points up ON up.user_id = u.id" {
		t.Fatalf("users points join = %#v", resource.Joins)
	}
	if resource.SortFields["points"] != "up.points" {
		t.Fatalf("users points sort = %q", resource.SortFields["points"])
	}
}

func TestAdminSessionHelpersPreserveLegacyRedisShape(t *testing.T) {
	raw := `{"id":7,"user_id":"42","token":"access","refresh_token":"refresh","user_agent":"ua","client_ip":"127.0.0.1","fingerprint":"fp","is_active":false,"expires_at":"2030-01-01T00:00:00Z","created_at":"2026-01-01T00:00:00Z"}`
	session, ok := adminParseRedisSession(raw)
	if !ok {
		t.Fatalf("adminParseRedisSession failed")
	}
	if session.ID != 7 || session.UserID != "42" || session.RefreshToken != "refresh" || session.IsActive {
		t.Fatalf("unexpected session: %#v", session)
	}

	row := adminSessionListRow(session, adminSessionUser{ID: 42, UserID: "display42", Nickname: "nick"})
	if row["user_id"] != int64(42) || row["user_display_id"] != "display42" || row["nickname"] != "nick" {
		t.Fatalf("unexpected session list row: %#v", row)
	}
	if row["last_active_at"] != session.CreatedAt {
		t.Fatalf("legacy session last_active_at = %#v, want created_at fallback", row["last_active_at"])
	}
}

func TestDefaultQualityRewardSettingsKeepExpressShape(t *testing.T) {
	defaults := defaultQualityRewardSettings()
	if len(defaults) != 3 {
		t.Fatalf("len(defaultQualityRewardSettings) = %d, want 3", len(defaults))
	}
	if defaults[0]["quality_level"] != "low" || defaults[2]["reward_amount"] != 5.00 {
		t.Fatalf("unexpected default settings: %#v", defaults)
	}
}

func TestAdminResetUserOnboardingResetsRequestedUserOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:admin-reset-user-onboarding?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	users := []domain.User{
		{ID: 1, UserID: "target-user", Nickname: "Target", ProfileCompleted: true, IsActive: true},
		{ID: 2, UserID: "other-user", Nickname: "Other", ProfileCompleted: true, IsActive: true},
	}
	if err := db.Create(&users).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/admin/users/target-user/reset-onboarding", nil)

	NativeHandlers{DB: db}.adminResetUserOnboarding(ctx, "target-user")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var responseBody struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if responseBody.Data["user_id"] != "target-user" || responseBody.Data["was_completed"] != true {
		t.Fatalf("unexpected response data: %#v", responseBody.Data)
	}

	var target domain.User
	if err := db.Where("user_id = ?", "target-user").First(&target).Error; err != nil {
		t.Fatalf("reload target: %v", err)
	}
	if target.ProfileCompleted {
		t.Fatal("target user profile_completed was not reset")
	}
	var other domain.User
	if err := db.Where("user_id = ?", "other-user").First(&other).Error; err != nil {
		t.Fatalf("reload other: %v", err)
	}
	if !other.ProfileCompleted {
		t.Fatal("other user profile_completed was changed")
	}
}

func TestOnboardingSettingMetadata(t *testing.T) {
	if got := settingTypeForKey("onboarding_points_intro_detail", ""); got != "textarea" {
		t.Fatalf("settingTypeForKey(detail) = %q, want textarea", got)
	}
	if got := settingTypeForKey("onboarding_enabled", true); got != "boolean" {
		t.Fatalf("settingTypeForKey(enabled) = %q, want boolean", got)
	}
	if got := settingLabel("onboarding_allow_skip"); got != "允许跳过引导" {
		t.Fatalf("settingLabel(allow_skip) = %q", got)
	}
	if got := settingHint("onboarding_points_intro_detail"); got == "" {
		t.Fatal("settingHint(detail) should not be empty")
	}
}

func TestAdminUpdateSettingsValidatesNotificationSuppression(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := services.NewSettingsService(nil, nil)
	handler := NativeHandlers{Settings: settings}

	for _, tt := range []struct {
		name   string
		body   string
		status int
	}{
		{
			name:   "valid",
			body:   `{"notification_interaction_suppression_enabled":false,"notification_interaction_suppression_window_seconds":120,"notification_interaction_suppression_threshold":4}`,
			status: http.StatusOK,
		},
		{
			name:   "window too low",
			body:   `{"notification_interaction_suppression_window_seconds":59}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "threshold too high",
			body:   `{"notification_interaction_suppression_threshold":101}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "invalid enabled",
			body:   `{"notification_interaction_suppression_enabled":"sometimes"}`,
			status: http.StatusBadRequest,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rec)
			ctx.Request = httptest.NewRequest(http.MethodPut, "/api/admin/system-settings", strings.NewReader(tt.body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			handler.adminUpdateSettings(ctx)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d, body=%s", rec.Code, tt.status, rec.Body.String())
			}
		})
	}

	if settings.Bool("notification_interaction_suppression_enabled") {
		t.Fatal("valid update should save enabled=false")
	}
	if got := settings.Int("notification_interaction_suppression_window_seconds", 0); got != 120 {
		t.Fatalf("saved window = %d, want 120", got)
	}
	if got := settings.Int("notification_interaction_suppression_threshold", 0); got != 4 {
		t.Fatalf("saved threshold = %d, want 4", got)
	}
}

func TestNotificationSettingMetadata(t *testing.T) {
	if got := settingLabel("notification_interaction_suppression_enabled"); got != "互动通知抑制" {
		t.Fatalf("notification suppression label = %q", got)
	}
	if got := settingHint("notification_interaction_suppression_window_seconds"); got == "" {
		t.Fatal("notification suppression window hint should not be empty")
	}
}

func TestAdminUpdateSettingsValidatesFileRecycleControls(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := services.NewSettingsService(nil, nil)
	handler := NativeHandlers{Settings: settings}

	for _, tt := range []struct {
		name   string
		body   string
		status int
	}{
		{
			name:   "valid",
			body:   `{"file_recycle_retention_days":45,"file_recycle_cleanup_interval_hours":12}`,
			status: http.StatusOK,
		},
		{
			name:   "retention too low",
			body:   `{"file_recycle_retention_days":0}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "interval too high",
			body:   `{"file_recycle_cleanup_interval_hours":721}`,
			status: http.StatusBadRequest,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(rec)
			ctx.Request = httptest.NewRequest(http.MethodPut, "/api/admin/system-settings", strings.NewReader(tt.body))
			ctx.Request.Header.Set("Content-Type", "application/json")
			handler.adminUpdateSettings(ctx)
			if rec.Code != tt.status {
				t.Fatalf("status = %d, want %d, body=%s", rec.Code, tt.status, rec.Body.String())
			}
		})
	}

	if got := settings.Int(services.FileRecycleRetentionDaysKey, 0); got != 45 {
		t.Fatalf("saved retention days = %d, want 45", got)
	}
	if got := settings.Int(services.FileRecycleCleanupIntervalHoursKey, 0); got != 12 {
		t.Fatalf("saved cleanup interval hours = %d, want 12", got)
	}
}

func TestFileRecycleSettingMetadata(t *testing.T) {
	if got := settingLabel(services.FileRecycleRetentionDaysKey); got != "文件回收站保留时间（天）" {
		t.Fatalf("file recycle retention label = %q", got)
	}
	if got := settingHint(services.FileRecycleCleanupIntervalHoursKey); got == "" {
		t.Fatal("file recycle cleanup interval hint should not be empty")
	}
}

func TestAdminUpdateSettingsValidatesSiteProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := services.NewSettingsService(nil, nil)
	handler := NativeHandlers{Settings: settings}

	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/admin/system-settings", strings.NewReader(`{
		"site_title":{"en":"Moon","zh-CN":"月站"},
		"site_description":{"en":"A calm creator feed","zh-CN":"安静的创作信息流"},
		"site_avatar_url":"/api/file/attachments/site-avatar.png"
	}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler.adminUpdateSettings(ctx)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	profile := services.ReadSiteProfileForLocale(settings, "zh-CN")
	if profile.Title != "月站" {
		t.Fatalf("localized site title = %q, want 月站", profile.Title)
	}
	if profile.Description != "安静的创作信息流" {
		t.Fatalf("localized site description = %q, want 安静的创作信息流", profile.Description)
	}
	if profile.AvatarURL != "/api/file/attachments/site-avatar.png" {
		t.Fatalf("site avatar = %q", profile.AvatarURL)
	}

	rec = httptest.NewRecorder()
	ctx, _ = gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/admin/system-settings", strings.NewReader(`{"site_avatar_url":"javascript:alert(1)"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	handler.adminUpdateSettings(ctx)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid avatar status = %d, want 400, body=%s", rec.Code, rec.Body.String())
	}
}

func TestSiteSettingMetadata(t *testing.T) {
	if got := settingLabel(services.SiteTitleSetting); got != "网站标题" {
		t.Fatalf("site title label = %q", got)
	}
	if got := settingHint(services.SiteAvatarURLSetting); got == "" {
		t.Fatal("site avatar hint should not be empty")
	}
}

func testGinContextWithQuery(target string) *gin.Context {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	ctx.Request = httptest.NewRequest(http.MethodGet, target, nil)
	return ctx
}
