package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/http/routes"
	"yuem-go/backend-gin/internal/http/swaggerui"
)

type fatalPanicTestExit struct {
	report fatalPanicReport
}

func captureFatalPanics(t *testing.T) {
	t.Helper()
	original := fatalPanicExit
	fatalPanicExit = func(report fatalPanicReport) {
		panic(fatalPanicTestExit{report: report})
	}
	t.Cleanup(func() {
		fatalPanicExit = original
	})
}

func serveHTTP(t *testing.T, router http.Handler, rec *httptest.ResponseRecorder, req *http.Request) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			if fatal, ok := recovered.(fatalPanicTestExit); ok {
				t.Fatalf("%s %s triggered fatal panic: %v", fatal.report.Method, fatal.report.Path, fatal.report.Value)
			}
			panic(recovered)
		}
	}()
	router.ServeHTTP(rec, req)
}

func TestGinLoggerIncludesContextErrorsForInternalServerErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	core, observed := observer.New(zapcore.ErrorLevel)
	router := gin.New()
	router.Use(ginLogger(zap.New(core), zapcore.DebugLevel, nil))
	router.GET("/api/posts", func(c *gin.Context) {
		_ = c.Error(fmt.Errorf("load posts failed: database connection reset"))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "error.internal"})
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/posts", nil)
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	entries := observed.FilterMessage("request").All()
	if len(entries) != 1 {
		t.Fatalf("logged entries = %d, want 1", len(entries))
	}
	fields := entries[0].ContextMap()
	errorsValue, ok := fields["errors"].([]any)
	if !ok || len(errorsValue) != 1 || !strings.Contains(fmt.Sprint(errorsValue[0]), "database connection reset") {
		t.Fatalf("errors field = %#v, want database error", fields["errors"])
	}
	if fields["route"] != "/api/posts" {
		t.Fatalf("route field = %#v, want /api/posts", fields["route"])
	}
}

func TestMigrationStatusReportsCurrentNativeCoverage(t *testing.T) {
	captureFatalPanics(t)
	router, err := NewRouter(testConfig(), zap.NewNop())
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/_gin_migration/status", nil)
	rec := httptest.NewRecorder()
	serveHTTP(t, router, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			ExpressRoutes    int    `json:"express_routes"`
			RegisteredRoutes int    `json:"registered_http_routes"`
			NativeHTTPRoutes int    `json:"native_http_routes"`
			ProxyHTTPRoutes  int    `json:"proxy_http_routes"`
			WebSocketEntries int    `json:"websocket_entries"`
			FinalGate        string `json:"final_gate"`
			SchemaPolicy     string `json:"database_schema_policy"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode status body: %v", err)
	}
	if body.Data.ExpressRoutes != 508 || body.Data.RegisteredRoutes != 508 {
		t.Fatalf("route totals = express %d registered %d, want 508/508", body.Data.ExpressRoutes, body.Data.RegisteredRoutes)
	}
	if body.Data.NativeHTTPRoutes != 508 {
		t.Fatalf("native_http_routes = %d, want 508", body.Data.NativeHTTPRoutes)
	}
	if body.Data.ProxyHTTPRoutes != 0 {
		t.Fatalf("proxy_http_routes = %d, want 0", body.Data.ProxyHTTPRoutes)
	}
	if body.Data.WebSocketEntries != 1 {
		t.Fatalf("websocket_entries = %d, want 1", body.Data.WebSocketEntries)
	}
	if body.Data.FinalGate == "" || body.Data.SchemaPolicy == "" {
		t.Fatalf("status body missing migration gate details: %+v", body.Data)
	}
}

func TestHealthEndpointIsNative(t *testing.T) {
	captureFatalPanics(t)
	router, err := NewRouter(testConfig(), zap.NewNop())
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	serveHTTP(t, router, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health body: %v", err)
	}
	if body["code"].(float64) != 200 || body["message"].(string) != "OK" {
		t.Fatalf("unexpected health body: %#v", body)
	}
	if rec.Header().Get("X-Gin-Migration-Mode") == "compat-proxy" {
		t.Fatalf("/api/health should not be served by compatibility proxy")
	}
}

func TestNetworkDiagnosticsEndpointIsNative(t *testing.T) {
	captureFatalPanics(t)
	router, err := NewRouter(testConfig(), zap.NewNop())
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/diagnostics/network?sample=1", nil)
	req.Host = "app.example.test"
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.4")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "app.example.test")
	req.Header.Set("X-Forwarded-Port", "443")
	req.Header.Set("CF-Ray", "test-ray-LAX")
	rec := httptest.NewRecorder()
	serveHTTP(t, router, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			CDN struct {
				Provider string `json:"provider"`
				RayID    string `json:"rayId"`
			} `json:"cdn"`
			ClientIP string `json:"clientIp"`
			Request  struct {
				Host       string `json:"host"`
				Method     string `json:"method"`
				Path       string `json:"path"`
				Scheme     string `json:"scheme"`
				RequestURI string `json:"requestUri"`
			} `json:"request"`
			Forwarded struct {
				Host     string   `json:"host"`
				Port     string   `json:"port"`
				Protocol string   `json:"protocol"`
				Chain    []string `json:"chain"`
			} `json:"forwarded"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode diagnostics body: %v", err)
	}
	if body.Code != 200 || body.Data.Request.Method != http.MethodGet || body.Data.Request.Path != "/api/diagnostics/network" {
		t.Fatalf("unexpected diagnostics body: %#v", body)
	}
	if body.Data.ClientIP != "203.0.113.10" || body.Data.Request.Scheme != "https" || body.Data.Forwarded.Port != "443" {
		t.Fatalf("forwarding details not preserved: %#v", body.Data)
	}
	if body.Data.CDN.Provider != "cloudflare" || body.Data.CDN.RayID != "test-ray-LAX" {
		t.Fatalf("cdn details not detected: %#v", body.Data.CDN)
	}
}

func TestDatabaseOptionalAuthBootstrapRoutes(t *testing.T) {
	captureFatalPanics(t)
	cfg := testConfig()
	cfg.Email.Enabled = true
	cfg.OAuth2.Enabled = true
	cfg.OAuth2.OnlyOAuth2 = true
	cfg.OAuth2.ClientID = "oauth-client-id"
	cfg.OAuth2.LoginURL = "https://auth.example.test/login"
	cfg.OAuth2.RedirectURI = ""
	cfg.OAuth2.RedirectBaseURL = "http://localhost:3000"
	cfg.Geetest.Enabled = true
	cfg.Geetest.CaptchaID = "captcha-id"
	router, err := NewRouter(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	tests := []struct {
		name       string
		path       string
		assertData func(t *testing.T, data map[string]any)
	}{
		{
			name: "auth config",
			path: "/api/auth/auth-config",
			assertData: func(t *testing.T, data map[string]any) {
				t.Helper()
				if data["emailEnabled"] != true || data["oauth2Enabled"] != true || data["oauth2OnlyLogin"] != true || data["geetestEnabled"] != true {
					t.Fatalf("auth config flags = %#v", data)
				}
				if data["oauth2LoginUrl"] != "https://auth.example.test/login" || data["geetestCaptchaId"] != "captcha-id" {
					t.Fatalf("auth config URLs/ids = %#v", data)
				}
				if data["oauth2StartUrl"] != "/api/auth/oauth2/login" {
					t.Fatalf("auth config start URL = %#v", data)
				}
			},
		},
		{
			name: "email config",
			path: "/api/auth/email-config",
			assertData: func(t *testing.T, data map[string]any) {
				t.Helper()
				if data["emailEnabled"] != true {
					t.Fatalf("email config = %#v", data)
				}
			},
		},
		{
			name: "captcha",
			path: "/api/auth/captcha",
			assertData: func(t *testing.T, data map[string]any) {
				t.Helper()
				if data["captchaId"] == "" || !strings.Contains(fmt.Sprint(data["captchaSvg"]), "<svg") {
					t.Fatalf("captcha data = %#v", data)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			serveHTTP(t, router, rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
			}
			var body struct {
				Code int            `json:"code"`
				Data map[string]any `json:"data"`
			}
			if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response body: %v", err)
			}
			if body.Code != 200 {
				t.Fatalf("code = %d, want 200, body=%s", body.Code, rec.Body.String())
			}
			tt.assertData(t, body.Data)
		})
	}

	t.Run("oauth2 login start", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth2/login", nil)
		rec := httptest.NewRecorder()
		serveHTTP(t, router, rec, req)

		if rec.Code != http.StatusFound {
			t.Fatalf("status = %d, want 302, body=%s", rec.Code, rec.Body.String())
		}
		location := rec.Header().Get("Location")
		parsed, err := url.Parse(location)
		if err != nil {
			t.Fatalf("parse redirect location %q: %v", location, err)
		}
		if parsed.Scheme != "https" || parsed.Host != "auth.example.test" || parsed.Path != "/login/oauth2.1/authorize" {
			t.Fatalf("redirect location = %q", location)
		}
		query := parsed.Query()
		if query.Get("response_type") != "code" || query.Get("client_id") != "oauth-client-id" || query.Get("code_challenge_method") != "S256" {
			t.Fatalf("redirect query missing OAuth2 parameters: %s", parsed.RawQuery)
		}
		if query.Get("state") == "" || query.Get("code_challenge") == "" || query.Get("redirect_uri") != "http://localhost:3000/api/auth/oauth2/callback" {
			t.Fatalf("redirect query missing state/challenge/callback: %s", parsed.RawQuery)
		}
	})

	t.Run("oauth2 callback stays public", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/auth/oauth2/callback?code=sample&state=missing", nil)
		rec := httptest.NewRecorder()
		serveHTTP(t, router, rec, req)

		if rec.Code != http.StatusFound {
			t.Fatalf("status = %d, want 302, body=%s", rec.Code, rec.Body.String())
		}
		if location := rec.Header().Get("Location"); !strings.Contains(location, "error=invalid_state") {
			t.Fatalf("callback should reach OAuth handler and redirect invalid state, location=%q", location)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/api/posts?page=1&limit=1", nil)
	rec := httptest.NewRecorder()
	serveHTTP(t, router, rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("business status = %d, want 500, body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "数据库未配置") {
		t.Fatalf("business response should still report missing database: %s", rec.Body.String())
	}
}

func TestSwaggerJSONRouteServesGeneratedSpec(t *testing.T) {
	captureFatalPanics(t)
	cfg := testConfig()
	router, err := NewRouter(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/"+cfg.Debug.SwaggerDocsPath+".json", nil)
	rec := httptest.NewRecorder()
	serveHTTP(t, router, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q, want application/json", contentType)
	}
	var body struct {
		OpenAPI    string         `json:"openapi"`
		Paths      map[string]any `json:"paths"`
		RouteCount int            `json:"x-route-count"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode swagger body: %v", err)
	}
	if body.OpenAPI != "3.0.0" {
		t.Fatalf("openapi = %q, want 3.0.0", body.OpenAPI)
	}
	if body.RouteCount != 508 {
		t.Fatalf("x-route-count = %d, want 508", body.RouteCount)
	}
	if len(body.Paths) == 0 {
		t.Fatalf("swagger paths should not be empty")
	}
	if _, ok := body.Paths["/api/health"]; !ok {
		t.Fatalf("swagger spec missing /api/health path")
	}
}

func TestSwaggerDocsPageServesBrowsableHTML(t *testing.T) {
	captureFatalPanics(t)
	cfg := testConfig()
	router, err := NewRouter(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/"+cfg.Debug.SwaggerDocsPath, nil)
	rec := httptest.NewRecorder()
	serveHTTP(t, router, rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	if contentType := rec.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/html") {
		t.Fatalf("Content-Type = %q, want text/html", contentType)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Yuem Go-Gin API") || !strings.Contains(body, cfg.Debug.SwaggerDocsPath+".json") {
		t.Fatalf("swagger docs page missing expected content: %s", body)
	}
	if !strings.Contains(body, "Scalar.createApiReference") {
		t.Fatalf("swagger docs page should initialize Scalar: %s", body)
	}
	if !strings.Contains(body, "?ui="+swaggerui.ScalarUIQuery()) {
		t.Fatalf("swagger docs page should load versioned Scalar assets: %s", body)
	}
	if !strings.Contains(body, `"hiddenClients":[]`) || !strings.Contains(body, `credentials: "same-origin"`) || !strings.Contains(body, "removeEmptyAuthorizationHeader") {
		t.Fatalf("swagger docs page should enable Scalar clients and same-origin cookie auth: %s", body)
	}
	for _, hiddenOption := range []string{`"hideTestRequestButton":true`, `"hideClientButton":true`, `"hiddenClients":true`} {
		if strings.Contains(body, hiddenOption) {
			t.Fatalf("swagger docs page should not disable Scalar test clients with %s: %s", hiddenOption, body)
		}
	}
	if strings.Contains(body, "readYuemAccessToken") || strings.Contains(body, `preferredSecurityScheme: "bearerAuth"`) {
		t.Fatalf("swagger docs page should not inject stale readable JWT over HttpOnly cookies: %s", body)
	}
	if strings.Contains(body, "SwaggerUIBundle") {
		t.Fatalf("swagger docs page should not initialize Swagger UI: %s", body)
	}
}

func TestSwaggerUIStaticRoutesPreserveLegacyAssetsAndServeScalarAssets(t *testing.T) {
	captureFatalPanics(t)
	cfg := testConfig()
	router, err := NewRouter(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}

	legacyReq := httptest.NewRequest(http.MethodGet, "/api/swagger-ui/swagger-ui-bundle.js", nil)
	legacyRec := httptest.NewRecorder()
	serveHTTP(t, router, legacyRec, legacyReq)

	if legacyRec.Code != http.StatusOK {
		t.Fatalf("legacy status = %d, want 200, body=%s", legacyRec.Code, legacyRec.Body.String())
	}
	if !strings.Contains(legacyRec.Body.String(), "SwaggerUIBundle") {
		t.Fatalf("legacy asset should still serve Swagger UI bundle")
	}

	scalarReq := httptest.NewRequest(http.MethodGet, "/api/swagger-ui/swagger-ui-bundle.js?ui="+swaggerui.ScalarUIQuery(), nil)
	scalarRec := httptest.NewRecorder()
	serveHTTP(t, router, scalarRec, scalarReq)

	if scalarRec.Code != http.StatusOK {
		t.Fatalf("scalar status = %d, want 200, body=%s", scalarRec.Code, scalarRec.Body.String())
	}
	scalarBody := scalarRec.Body.String()
	if !strings.Contains(scalarBody, "@scalar/api-reference@") || !strings.Contains(scalarBody, "window.Scalar") {
		t.Fatalf("versioned asset should serve Scalar bundle")
	}
}

func TestEveryMatrixRouteHasNativeGinEntry(t *testing.T) {
	captureFatalPanics(t)
	cfg := testConfig()
	router, err := NewRouter(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	matrix, err := routes.LoadMatrix()
	if err != nil {
		t.Fatalf("LoadMatrix() error = %v", err)
	}

	for _, route := range matrix.Routes {
		method := strings.ToUpper(route.Method)
		if method == "ALL" {
			method = http.MethodGet
		}
		path := sampleMatrixPath(route.Path, cfg)
		req := httptest.NewRequest(method, path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					if fatal, ok := recovered.(fatalPanicTestExit); ok {
						t.Fatalf("%s %s triggered fatal panic: %v\n%s", fatal.report.Method, fatal.report.Path, fatal.report.Value, stackSnippet(fatal.report.Stack))
					}
					panic(recovered)
				}
			}()
			router.ServeHTTP(rec, req)
		}()

		if rec.Code == http.StatusNotImplemented {
			t.Fatalf("%s %s returned 501: %s", method, path, rec.Body.String())
		}
		if rec.Header().Get("X-Gin-Migration-Mode") == "compat-proxy" {
			t.Fatalf("%s %s was served by compatibility proxy", method, path)
		}
		body := strings.ToLower(rec.Body.String())
		for _, marker := range []string{"route not implemented", "route not found", "not implemented"} {
			if strings.Contains(body, marker) {
				t.Fatalf("%s %s hit unfinished matrix handler: %s", method, path, rec.Body.String())
			}
		}
	}
}

func TestRouteMatrixSourcesHaveNativeOwners(t *testing.T) {
	captureFatalPanics(t)
	cfg := testConfig()
	router, err := NewRouter(cfg, zap.NewNop())
	if err != nil {
		t.Fatalf("NewRouter() error = %v", err)
	}
	matrix, err := routes.LoadMatrix()
	if err != nil {
		t.Fatalf("LoadMatrix() error = %v", err)
	}

	explicitRoutes := map[string]struct{}{}
	for _, route := range router.Routes() {
		explicitRoutes[strings.ToUpper(route.Method)+" "+normalizeGinPath(route.Path)] = struct{}{}
	}

	for _, route := range matrix.Routes {
		if matrixSourceHasDispatcher(route.SourceFile) {
			continue
		}
		key := strings.ToUpper(route.Method) + " " + normalizeGinPath(materializeMatrixPattern(route.Path, cfg))
		if _, ok := explicitRoutes[key]; !ok {
			t.Fatalf("%s has no explicit Gin route and source %s has no matrix dispatcher", key, route.SourceFile)
		}
	}
}

func stackSnippet(stack []byte) string {
	lines := strings.Split(string(stack), "\n")
	if len(lines) > 20 {
		lines = lines[:20]
	}
	return fmt.Sprintf("stack:\n%s", strings.Join(lines, "\n"))
}

func TestShouldSuppressRequestInfo(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		path    string
		status  int
		latency time.Duration
		want    bool
	}{
		{name: "successful delivered receipt", method: http.MethodPost, path: "/api/im/messages/226/delivered", status: http.StatusOK, latency: 20 * time.Millisecond, want: true},
		{name: "successful read receipt", method: http.MethodPost, path: "/api/im/messages/226/read", status: http.StatusOK, latency: 20 * time.Millisecond, want: true},
		{name: "receipt error stays logged", method: http.MethodPost, path: "/api/im/messages/226/read", status: http.StatusInternalServerError, latency: 20 * time.Millisecond, want: false},
		{name: "slow receipt stays logged", method: http.MethodPost, path: "/api/im/messages/226/read", status: http.StatusOK, latency: 500 * time.Millisecond, want: false},
		{name: "normal im route stays logged", method: http.MethodPost, path: "/api/im/messages", status: http.StatusOK, latency: 20 * time.Millisecond, want: false},
		{name: "read suffix outside im messages stays logged", method: http.MethodPost, path: "/api/notifications/1/read", status: http.StatusOK, latency: 20 * time.Millisecond, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldSuppressRequestInfo(tt.method, tt.path, tt.status, tt.latency)
			if got != tt.want {
				t.Fatalf("shouldSuppressRequestInfo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequestLogLevelForStatus(t *testing.T) {
	tests := []struct {
		status int
		want   zapcore.Level
	}{
		{status: http.StatusOK, want: zapcore.InfoLevel},
		{status: http.StatusNotFound, want: zapcore.WarnLevel},
		{status: http.StatusInternalServerError, want: zapcore.ErrorLevel},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.status), func(t *testing.T) {
			if got := requestLogLevelForStatus(tt.status); got != tt.want {
				t.Fatalf("requestLogLevelForStatus(%d) = %s, want %s", tt.status, got, tt.want)
			}
		})
	}
}

func TestMaintenanceProtectedPathRules(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: "/api/posts", want: true},
		{path: "/api/upload/single", want: true},
		{path: "/api/health", want: false},
		{path: "/api/diagnostics/network", want: false},
		{path: "/api/admin/performance", want: false},
		{path: "/api/auth/admin/login", want: false},
		{path: "/api/maintenance/status", want: false},
		{path: "/api/maintenance/enter", want: false},
		{path: "/api/file/images/sample.png", want: false},
		{path: "/api/pyvideo-api-proxy/posts", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := maintenanceProtectedPath(tt.path); got != tt.want {
				t.Fatalf("maintenanceProtectedPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func matrixSourceHasDispatcher(source string) bool {
	for _, suffix := range []string{
		"auth.js",
		"users.js",
		"im.js",
		"admin.js",
		"app.js",
		"file.js",
		"pyvideoProxy.js",
	} {
		if strings.HasSuffix(source, suffix) {
			return true
		}
	}
	return false
}

func normalizeGinPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, "*") {
			parts[i] = "*"
		}
	}
	return strings.Join(parts, "/")
}

func materializeMatrixPattern(pattern string, cfg config.Config) string {
	pattern = strings.ReplaceAll(pattern, "${JWT_TEST_TOKEN_PATH}", cfg.Debug.JWTTestTokenPath)
	pattern = strings.ReplaceAll(pattern, "${SWAGGER_DOCS_PATH}", cfg.Debug.SwaggerDocsPath)
	return pattern
}

func testConfig() config.Config {
	cfg := config.Load()
	cfg.Database.URL = ""
	cfg.Database.Driver = ""
	cfg.Redis.Addr = "127.0.0.1:1"
	cfg.Observe.SystemLogEnabled = false
	cfg.Observe.MetricsEnabled = false
	return cfg
}

func sampleMatrixPath(pattern string, cfg config.Config) string {
	path := strings.ReplaceAll(pattern, "${JWT_TEST_TOKEN_PATH}", cfg.Debug.JWTTestTokenPath)
	path = strings.ReplaceAll(path, "${SWAGGER_DOCS_PATH}", cfg.Debug.SwaggerDocsPath)
	parts := strings.Split(strings.Trim(path, "/"), "/")
	for i, part := range parts {
		switch {
		case part == "*":
			parts[i] = "sample"
		case strings.HasPrefix(part, "*"):
			parts[i] = "sample"
		case strings.HasPrefix(part, ":"):
			name := strings.TrimPrefix(part, ":")
			switch name {
			case "type":
				parts[i] = "images"
			case "filename":
				parts[i] = "sample.jpg"
			case "code":
				parts[i] = "TESTCODE"
			default:
				parts[i] = "1"
			}
		}
	}
	if len(parts) == 0 {
		return "/"
	}
	return "/" + strings.Join(parts, "/")
}
