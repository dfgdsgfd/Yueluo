package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDatabaseDriver(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "mysql", url: "mysql://user:pass@localhost:3306/app", want: "mysql"},
		{name: "mariadb", url: "mariadb://user:pass@localhost:3306/app", want: "mysql"},
		{name: "postgres", url: "postgres://user:pass@localhost:5432/app", want: "postgres"},
		{name: "postgresql", url: "postgresql://user:pass@localhost:5432/app", want: "postgres"},
		{name: "empty", url: "", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := databaseDriver(tt.url); got != tt.want {
				t.Fatalf("databaseDriver(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestLoadMapsGoGinEnvVariablesAndRuntimeAliases(t *testing.T) {
	env := map[string]string{
		"GIN_PORT":                                "3001",
		"GIN_MODE":                                "release",
		"APP_ENV":                                 "production",
		"CLIENT_IP_HEADERS":                       "X-Forwarded-For,X-Real-IP,CF-Connecting-IP",
		"LOG_LEVEL":                               "error",
		"LOG_FILE_ENABLED":                        "false",
		"LOG_FILE_DIR":                            "runtime-logs",
		"LOG_FILE_RETENTION_DAYS":                 "14",
		"GEOIP_COUNTRY_MMDB_PATH":                 "/data/geoip/Country.mmdb",
		"GEOIP_COUNTRY_MMDB_URL":                  "https://example.com/Country.mmdb",
		"GEOIP_COUNTRY_MMDB_SHA256_URL":           "https://example.com/Country.mmdb.sha256sum",
		"DATABASE_URL":                            "postgresql://u:p@db:5432/app?schema=public",
		"DB_POOL_SIZE":                            "24",
		"DB_MAX_IDLE_CONNS":                       "12",
		"DB_CONN_MAX_LIFETIME_MS":                 "300000",
		"DB_CONNECTION_TIMEOUT":                   "15000",
		"DB_IDLE_TIMEOUT":                         "45000",
		"DB_SCHEMA":                               "app_schema",
		"DB_SSL_MODE":                             "require",
		"DB_STATEMENT_TIMEOUT":                    "3000",
		"DB_LOCK_TIMEOUT":                         "2000",
		"DB_LOG_LEVEL":                            "error",
		"DB_IGNORE_RECORD_NOT_FOUND_ERROR":        "false",
		"DB_PREPARE_STMT":                         "true",
		"DB_CREATE_BATCH_SIZE":                    "200",
		"DB_SKIP_DEFAULT_TRANSACTION":             "true",
		"CORS_ORIGINS":                            "https://xse.example.com",
		"CORS_ORIGIN":                             "https://xse.example.com",
		"API_BASE_URL":                            "https://xse.example.com/api",
		"FRONTEND_URL":                            "https://xse.example.com",
		"LOCAL_BASE_URL":                          "https://xse.example.com",
		"PYVIDEO_API_KEY":                         "proxy-key",
		"VIDEO_CENTER_API_KEY":                    "video-center-key",
		"PYVIDEO_UPSTREAM_URL":                    "https://video.example.com/",
		"QUEUE_ENABLED":                           "true",
		"QUEUE_IP_LOCATION_CONCURRENCY":           "2",
		"QUEUE_CONTENT_AUDIT_CONCURRENCY":         "1",
		"QUEUE_GENERAL_TASK_CONCURRENCY":          "3",
		"QUEUE_AI_TASK_CONCURRENCY":               "7",
		"QUEUE_VIDEO_TRANSCODING_CONCURRENCY":     "1",
		"QUEUE_BATCH_NOTE_CREATE_CONCURRENCY":     "4",
		"QUEUE_AUDIT_LOG_CONCURRENCY":             "6",
		"QUEUE_RETRY_ATTEMPTS":                    "5",
		"QUEUE_RETRY_DELAY":                       "2500",
		"REDIS_POOL_SIZE":                         "64",
		"REDIS_MIN_IDLE_CONNS":                    "8",
		"REDIS_CACHE_ENABLED":                     "true",
		"REDIS_CACHE_COMMAND_TIMEOUT_MS":          "75",
		"REDIS_CACHE_DEFAULT_TTL_SECONDS":         "45",
		"REDIS_CACHE_L1_TTL_SECONDS":              "3",
		"REDIS_CACHE_L1_MAX_ENTRIES":              "4096",
		"ACCESS_LOG_ENABLED":                      "true",
		"ACCESS_LOG_SCOPE":                        "all",
		"ACCESS_LOG_BEHAVIORS":                    "post_view,admin_access",
		"ACCESS_LOG_RETENTION_HOURS":              "720",
		"ACCESS_LOG_BUFFER_SIZE":                  "2048",
		"ACCESS_LOG_BATCH_SIZE":                   "50",
		"ACCESS_LOG_FLUSH_INTERVAL_MS":            "250",
		"SECURITY_AUDIT_LOG_ENABLED":              "true",
		"SECURITY_AUDIT_LOG_RETENTION_HOURS":      "1440",
		"UPLOAD_MAX_SIZE":                         "99mb",
		"DELETE_POST_UPLOADS_ON_DELETE":           "true",
		"UPLOAD_RECYCLE_ENABLED":                  "true",
		"UPLOAD_RECYCLE_ROOT":                     "uploads/custom-trash",
		"UPLOAD_RECYCLE_RETENTION":                "48h",
		"UPLOAD_RECYCLE_CLEANUP_INTERVAL":         "30m",
		"FILE_SIGNING_SECRET":                     "file-signing-secret",
		"FILE_SIGNING_TTL_SECONDS":                "120",
		"IMAGEHOST_TIMEOUT":                       "60000",
		"IMAGE_UPLOAD_STRATEGY":                   "local",
		"VIDEO_UPLOAD_STRATEGY":                   "local",
		"VIDEO_UPLOAD_DIR":                        "uploads/videos-legacy",
		"VIDEO_LOCAL_UPLOAD_DIR":                  "uploads/videos-legacy",
		"VIDEO_COVER_DIR":                         "uploads/covers",
		"UPLOAD_TEMP_DIR":                         "uploads/tmp-tests",
		"UPLOAD_TEMP_CLEANUP_INTERVAL":            "3m",
		"UPLOAD_TEMP_RETENTION":                   "12h",
		"PROTECTED_PACKAGE_RETENTION":             "90m",
		"AVATAR_UPLOAD_DIR":                       "uploads/avatar-tests",
		"BANNER_UPLOAD_DIR":                       "uploads/banner-tests",
		"VIDEO_TRANSCODING_ENABLED":               "true",
		"VIDEO_TRANSCODING_MAX_THREADS":           "6",
		"VIDEO_TRANSCODING_MAX_CONCURRENT":        "2",
		"DASH_RESOLUTIONS":                        "1080p:1800,1280x720:1500",
		"FFMPEG_CRF":                              "25",
		"WEBP_QUALITY":                            "86",
		"AVATAR_WEBP_QUALITY":                     "73",
		"WEBP_METHOD":                             "5",
		"WEBP_MAX_WIDTH":                          "1920",
		"HIDDEN_WATERMARK_SECRET":                 "hidden-watermark-secret",
		"HIDDEN_WATERMARK_REMOTE_URL":             "https://watermark.internal/",
		"HIDDEN_WATERMARK_REMOTE_API_KEY":         "remote-watermark-key",
		"HIDDEN_WATERMARK_REMOTE_SKIP_TLS_VERIFY": "true",
		"WATERMARK_ENABLED":                       "true",
		"WATERMARK_TYPE":                          "image",
		"WATERMARK_IMAGE_PATH":                    "/images/watermark.webp",
		"WATERMARK_TILE_MODE":                     "true",
		"USERNAME_WATERMARK_ENABLED":              "true",
		"USERNAME_WATERMARK_FONT_PATH":            "/fonts/NotoSansCJKsc-Regular.otf",
		"CONTENT_AUDIT_ENABLED":                   "true",
		"DIFY_API_URL":                            "https://audit.example.com/v1/chat-messages",
		"DIFY_API_KEY":                            "audit-key",
		"CHAT_WS_ENABLED":                         "true",
		"CHAT_WS_BASE_URL":                        "https://chat.example.com/",
		"CHAT_WS_TICKET_TTL":                      "30",
		"CHAT_WS_HTTP_TIMEOUT":                    "2000",
		"CHAT_MESSAGE_MAX_LENGTH":                 "3000",
		"IM_BASE":                                 "https://im-server.example.com/",
		"IM_HMAC_SECRET":                          "server-im-secret",
		"IM_MESSAGE_KEY":                          "message-key",
		"ALLOWED_MEDIA_DOMAINS":                   "cdn.example.com, images.example.com",
		"RECOMMENDATION_DEBUG":                    "true",
		"VERIFICATION_COLLECT_SENSITIVE_INFO":     "true",
		"OAUTH2_ENABLED":                          "true",
		"OAUTH2_ONLY_LOGIN":                       "true",
		"OAUTH2_LOGIN_URL":                        "https://user.example.com/",
		"OAUTH2_CLIENT_ID":                        "client-id",
		"OAUTH2_CLIENT_SECRET":                    "client-secret",
		"OAUTH2_REDIRECT_BASE_URL":                "https://app.example.com/",
		"OAUTH2_CALLBACK_PATH":                    "oauth2/callback",
		"OAUTH2_APP_CALLBACK_URL":                 "https://app.example.com/app/oauth/callback",
		"OAUTH2_DPOP_ENABLED":                     "true",
		"BALANCE_API_URL":                         "https://balance.example.com",
		"BALANCE_API_KEY":                         "balance-secret",
		"BALANCE_API_TIMEOUT":                     "7s",
		"CREATOR_WITHDRAW_ENABLED":                "true",
		"CREATOR_MIN_WITHDRAW_AMOUNT":             "1",
		"CREATOR_DAILY_EXTENDED_EARNINGS_CAP":     "50",
		"GEETEST_ENABLED":                         "true",
		"GEETEST_CAPTCHA_ID":                      "captcha-id",
		"GEETEST_CAPTCHA_KEY":                     "captcha-key",
		"JWT_EXPIRES_IN":                          "3600",
		"ADMIN_JWT_EXPIRES_IN":                    "12h",
		"REFRESH_TOKEN_EXPIRES_IN":                "7776000",
		"NOTIFICATION_EMAIL_ENABLED":              "true",
		"SYSTEM_NOTIFICATION_EMAIL_ENABLED":       "true",
		"DISCORD_WEBHOOK_ENABLED":                 "true",
		"DISCORD_WEBHOOK_URL":                     "https://discord.example.com/webhook",
	}
	for key, value := range env {
		t.Setenv(key, value)
	}

	cfg := Load()

	if cfg.Server.Port != "3001" || cfg.Server.Env != "production" || strings.Join(cfg.Server.ClientIPHeaders, ",") != "X-Forwarded-For,X-Real-IP,CF-Connecting-IP" {
		t.Fatalf("server config not mapped: %+v", cfg.Server)
	}
	if cfg.Log.Level != "error" || cfg.Log.FileEnabled || cfg.Log.FileDir != "runtime-logs" || cfg.Log.FileRetentionDays != 14 || cfg.Server.RequestLogLevel != "error" {
		t.Fatalf("log config not mapped: log=%+v server=%+v", cfg.Log, cfg.Server)
	}
	if cfg.GeoIP.CountryMMDBPath != "/data/geoip/Country.mmdb" || cfg.GeoIP.CountryMMDBURL != "https://example.com/Country.mmdb" || cfg.GeoIP.SHA256URL != "https://example.com/Country.mmdb.sha256sum" {
		t.Fatalf("geoip config not mapped: %+v", cfg.GeoIP)
	}
	if cfg.Database.Driver != "postgres" || cfg.Database.PoolSize != 24 || cfg.Database.MaxIdleConns != 12 || cfg.Database.ConnMaxLifetimeMS != 300000 || cfg.Database.LogLevel != "error" || cfg.Database.IgnoreRecordNotFound || !cfg.Database.PrepareStmt || cfg.Database.CreateBatchSize != 200 || !cfg.Database.SkipDefaultTransaction {
		t.Fatalf("database config not mapped: %+v", cfg.Database)
	}
	if len(cfg.CORS.Origins) != 1 || cfg.CORS.Origins[0] != "https://xse.example.com" {
		t.Fatalf("CORS_ORIGIN alias not mapped: %+v", cfg.CORS.Origins)
	}
	if cfg.Frontend.BaseURL != "https://xse.example.com" || cfg.API.BaseURL != "https://xse.example.com/api" {
		t.Fatalf("frontend/api base URL not mapped: frontend=%q api=%q", cfg.Frontend.BaseURL, cfg.API.BaseURL)
	}
	if cfg.Auth.JWTExpiresIn != "3600" || cfg.Auth.AdminJWTExpiresIn != "12h" || cfg.Auth.RefreshTokenExpiresIn != "7776000" {
		t.Fatalf("auth config not mapped: %+v", cfg.Auth)
	}
	if cfg.PyVideo.APIKey != "proxy-key" || cfg.PyVideo.VideoCenterAPIKey != "video-center-key" || cfg.PyVideo.UpstreamURL != "https://video.example.com" {
		t.Fatalf("pyvideo config not mapped: %+v", cfg.PyVideo)
	}
	if !cfg.Queue.Enabled || cfg.Queue.Concurrency.BatchNoteCreate != 4 || cfg.Queue.Concurrency.AuditLog != 6 || cfg.Queue.Concurrency.AITask != 7 || cfg.Queue.Retry.BackoffDelay != 2500*time.Millisecond {
		t.Fatalf("queue config not mapped: %+v", cfg.Queue)
	}
	if cfg.Redis.PoolSize != 64 || cfg.Redis.MinIdleConns != 8 || !cfg.Redis.CacheEnabled || cfg.Redis.CacheCommandTimeout != 75*time.Millisecond || cfg.Redis.CacheDefaultTTL != 45*time.Second || cfg.Redis.CacheL1TTL != 3*time.Second || cfg.Redis.CacheL1MaxEntries != 4096 {
		t.Fatalf("redis cache config not mapped: %+v", cfg.Redis)
	}
	if !cfg.AccessLog.Enabled || cfg.AccessLog.Scope != "all" || cfg.AccessLog.Retention != 720*time.Hour || cfg.AccessLog.BufferSize != 2048 || cfg.AccessLog.BatchSize != 50 || cfg.AccessLog.FlushInterval != 250*time.Millisecond || strings.Join(cfg.AccessLog.Behaviors, ",") != "post_view,admin_access" {
		t.Fatalf("access log config not mapped: %+v", cfg.AccessLog)
	}
	if !cfg.SecurityAuditLog.Enabled || cfg.SecurityAuditLog.Retention != 1440*time.Hour {
		t.Fatalf("security audit log config not mapped: %+v", cfg.SecurityAuditLog)
	}
	if cfg.Upload.MaxSize != 99*1024*1024 || !cfg.Upload.DeletePostUploadsOnDelete || cfg.Upload.FileSigning.Secret != "file-signing-secret" || cfg.Upload.FileSigning.TTL != 120*time.Second || cfg.Upload.Image.ImageHostTimeoutSeconds != 60 || cfg.Upload.Video.LocalUploadDir != "uploads/videos-legacy" {
		t.Fatalf("upload config not mapped: %+v", cfg.Upload)
	}
	if !cfg.Upload.Recycle.Enabled || cfg.Upload.Recycle.RootDir != "uploads/custom-trash" || cfg.Upload.Recycle.Retention != 48*time.Hour || cfg.Upload.Recycle.CleanupInterval != 30*time.Minute {
		t.Fatalf("upload recycle config not mapped: %+v", cfg.Upload.Recycle)
	}
	if cfg.Upload.Temp.RootDir != "uploads/tmp-tests" || cfg.Upload.Temp.CleanupInterval != 3*time.Minute || cfg.Upload.Temp.Retention != 12*time.Hour || cfg.Upload.Temp.ProtectedPackageRetention != 90*time.Minute || cfg.Upload.AvatarDir != "uploads/avatar-tests" || cfg.Upload.BannerDir != "uploads/banner-tests" {
		t.Fatalf("upload temporary/profile config not mapped: %+v", cfg.Upload)
	}
	if cfg.Balance.APIURL != "https://balance.example.com" || cfg.Balance.APIKey != "balance-secret" || cfg.Balance.Timeout != 7*time.Second {
		t.Fatalf("balance center config not mapped: %+v", cfg.Balance)
	}
	if !cfg.Video.Enabled || cfg.Video.MaxThreads != 6 || len(cfg.Video.DASH.Resolutions) != 2 || cfg.Video.FFmpeg.CRF == nil || *cfg.Video.FFmpeg.CRF != 25 {
		t.Fatalf("video transcoding config not mapped: %+v", cfg.Video)
	}
	if cfg.WebP.Quality != 86 || cfg.WebP.AvatarQuality != 73 || cfg.WebP.Method != 5 || cfg.WebP.MaxWidth == nil || *cfg.WebP.MaxWidth != 1920 || cfg.WebP.HiddenWatermark.Secret != "hidden-watermark-secret" || cfg.WebP.HiddenWatermark.Remote.URL != "https://watermark.internal" || cfg.WebP.HiddenWatermark.Remote.APIKey != "remote-watermark-key" || !cfg.WebP.HiddenWatermark.Remote.SkipTLSVerify || !cfg.WebP.Watermark.Enabled || !cfg.WebP.Watermark.TileMode || !cfg.WebP.UsernameWatermark.Enabled {
		t.Fatalf("webp/watermark config not mapped: %+v", cfg.WebP)
	}
	if !cfg.Audit.Enabled || cfg.Audit.APIKey != "audit-key" {
		t.Fatalf("content audit config not mapped: %+v", cfg.Audit)
	}
	if !cfg.Chat.Enabled || cfg.Chat.HTTPTimeout != 2*time.Second || cfg.Chat.MessageMaxLength != 3000 {
		t.Fatalf("chat config not mapped: %+v", cfg.Chat)
	}
	if cfg.IM.BaseURL != "https://im-server.example.com" || cfg.IM.HMACSecret != "server-im-secret" || cfg.IM.MessageKey != "message-key" {
		t.Fatalf("IM config not mapped: %+v", cfg.IM)
	}
	if got := strings.Join(cfg.Media.AllowedDomains, ","); got != "cdn.example.com,images.example.com" || !cfg.Media.RecommendationDebug {
		t.Fatalf("media config not mapped: %+v", cfg.Media)
	}
	if !cfg.Verify.CollectSensitiveInfo || !cfg.OAuth2.Enabled || !cfg.Creator.WithdrawEnabled || !cfg.Geetest.Enabled || !cfg.Notify.Discord.Enabled {
		t.Fatalf("integration flags not mapped: verify=%+v oauth=%+v balance=%+v creator=%+v geetest=%+v notify=%+v", cfg.Verify, cfg.OAuth2, cfg.Balance, cfg.Creator, cfg.Geetest, cfg.Notify)
	}
	if cfg.Balance.APIURL != "https://balance.example.com" || cfg.Balance.APIKey != "balance-secret" || cfg.Balance.Timeout != 7*time.Second {
		t.Fatalf("balance center config not mapped: %+v", cfg.Balance)
	}
	if cfg.OAuth2.RedirectBaseURL != "https://app.example.com" || cfg.OAuth2.CallbackPath != "oauth2/callback" || cfg.OAuth2.AppCallbackURL != "https://app.example.com/app/oauth/callback" {
		t.Fatalf("oauth2 redirect config not mapped: %+v", cfg.OAuth2)
	}
}

func TestDatabaseURLFromLegacyParts(t *testing.T) {
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_USER", "app")
	t.Setenv("DB_PASSWORD", "secret")
	t.Setenv("DB_NAME", "appdb")
	t.Setenv("DB_PORT", "5432")

	got := databaseURLFromEnv()
	if got != "postgresql://app:secret@db:5432/appdb" {
		t.Fatalf("databaseURLFromEnv() = %q", got)
	}
}

func TestLoadEnvFilesPreservesProcessEnvAndUsesLastDuplicate(t *testing.T) {
	temp := t.TempDir()
	envPath := filepath.Join(temp, ".env")
	content := strings.Join([]string{
		"TEST_DUPLICATE=value1",
		"TEST_EXISTING=file",
		"TEST_DUPLICATE=value2",
		"export TEST_EXPORTED=ok",
		"go test ./...",
	}, "\n")
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	withUnsetEnv(t, "TEST_DUPLICATE", "TEST_EXPORTED", "ENV_FILE", "GIN_ENV_FILE")
	t.Setenv("ENV_FILE", envPath)
	t.Setenv("TEST_EXISTING", "process")

	LoadEnvFiles()

	if got := os.Getenv("TEST_DUPLICATE"); got != "value2" {
		t.Fatalf("duplicate key = %q, want value2", got)
	}
	if got := os.Getenv("TEST_EXISTING"); got != "process" {
		t.Fatalf("process env was overwritten: %q", got)
	}
	if got := os.Getenv("TEST_EXPORTED"); got != "ok" {
		t.Fatalf("export syntax was not parsed: %q", got)
	}
}

func TestBackendGinEnvExampleContainsMigratedKeys(t *testing.T) {
	required := []string{
		"GIN_PORT", "GIN_MODE", "APP_ENV", "CLIENT_IP_HEADERS", "LOG_LEVEL", "LOG_FILE_ENABLED", "LOG_FILE_DIR", "LOG_FILE_RETENTION_DAYS", "REQUEST_LOG_LEVEL", "DATABASE_URL", "DB_HOST", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_PORT",
		"DB_POOL_SIZE", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME_MS", "DB_CONNECTION_TIMEOUT", "DB_IDLE_TIMEOUT", "DB_CHARSET", "DB_TIMEZONE", "DB_SCHEMA",
		"DB_SSL_MODE", "DB_STATEMENT_TIMEOUT", "DB_LOCK_TIMEOUT", "DB_LOG_LEVEL", "DB_IGNORE_RECORD_NOT_FOUND_ERROR", "DB_PREPARE_STMT", "DB_CREATE_BATCH_SIZE", "DB_SKIP_DEFAULT_TRANSACTION",
		"JWT_SECRET", "JWT_EXPIRES_IN", "ADMIN_JWT_EXPIRES_IN", "REFRESH_TOKEN_EXPIRES_IN", "API_BASE_URL", "FRONTEND_URL",
		"UPLOAD_MAX_SIZE", "DELETE_POST_UPLOADS_ON_DELETE", "FILE_SIGNING_SECRET", "FILE_SIGNING_TTL_SECONDS", "IMAGE_MAX_SIZE", "IMAGE_UPLOAD_STRATEGY", "VIDEO_UPLOAD_STRATEGY", "LOCAL_UPLOAD_DIR",
		"LOCAL_BASE_URL", "VIDEO_UPLOAD_DIR", "VIDEO_LOCAL_UPLOAD_DIR", "VIDEO_COVER_DIR", "UPLOAD_TEMP_DIR", "UPLOAD_TEMP_CLEANUP_INTERVAL", "UPLOAD_TEMP_RETENTION", "PROTECTED_PACKAGE_RETENTION", "AVATAR_UPLOAD_DIR", "BANNER_UPLOAD_DIR",
		"IMAGEHOST_API_URL", "IMAGEHOST_TIMEOUT", "R2_ACCESS_KEY_ID", "R2_SECRET_ACCESS_KEY", "R2_ENDPOINT",
		"R2_BUCKET_NAME", "R2_ACCOUNT_ID", "R2_REGION", "R2_PUBLIC_URL", "CORS_ORIGINS",
		"EMAIL_ENABLED", "SMTP_HOST", "SMTP_PORT", "SMTP_SECURE", "SMTP_USER", "SMTP_PASSWORD", "EMAIL_FROM",
		"EMAIL_FROM_NAME", "NOTIFICATION_EMAIL_ENABLED", "SYSTEM_NOTIFICATION_EMAIL_ENABLED",
		"DISCORD_WEBHOOK_ENABLED", "DISCORD_WEBHOOK_URL", "DISCORD_SITE_NAME", "DISCORD_SITE_URL",
		"OAUTH2_ENABLED", "OAUTH2_ONLY_LOGIN", "OAUTH2_LOGIN_URL", "OAUTH2_CLIENT_ID", "OAUTH2_CLIENT_SECRET",
		"OAUTH2_REDIRECT_URI", "OAUTH2_REDIRECT_BASE_URL", "OAUTH2_CALLBACK_PATH", "OAUTH2_APP_CALLBACK_URL", "OAUTH2_APP_CALLBACK_URLS", "OAUTH2_DPOP_ENABLED", "BALANCE_API_URL", "BALANCE_API_KEY", "BALANCE_API_TIMEOUT",
		"BALANCE_EXCHANGE_RATE_IN", "BALANCE_EXCHANGE_RATE_OUT", "FFMPEG_PATH", "FFPROBE_PATH",
		"VIDEO_TRANSCODING_ENABLED", "VIDEO_TRANSCODING_MAX_THREADS", "VIDEO_TRANSCODING_MAX_CONCURRENT",
		"VIDEO_DASH_OUTPUT_FORMAT", "DASH_SEGMENT_DURATION", "DASH_MIN_BITRATE", "DASH_MAX_BITRATE",
		"DASH_RESOLUTIONS", "ORIGINAL_VIDEO_MAX_BITRATE", "DELETE_ORIGINAL_VIDEO", "VIDEO_CHUNK_TEMP_DIR",
		"VIDEO_CHUNK_SIZE", "VIDEO_CHUNK_CLEANUP_INTERVAL", "VIDEO_CHUNK_EXPIRE_TIME", "VIDEO_MAX_SIZE",
		"FFMPEG_PRESET", "FFMPEG_PROFILE", "FFMPEG_CRF", "FFMPEG_GOP_SIZE", "FFMPEG_B_FRAMES",
		"FFMPEG_REF_FRAMES", "FFMPEG_COMPLEXITY", "FFMPEG_AUDIO_BITRATE", "FFMPEG_AUDIO_SAMPLE_RATE",
		"FFMPEG_PIXEL_FORMAT", "FFMPEG_HARDWARE_ACCEL", "FFMPEG_HARDWARE_ACCEL_TYPE", "WEBP_ENABLE_CONVERSION",
		"WEBP_QUALITY", "AVATAR_WEBP_QUALITY", "WEBP_METHOD", "WEBP_CONVERT_JPEG", "WEBP_CONVERT_PNG", "WEBP_KEEP_ORIGINAL", "WEBP_MAX_WIDTH",
		"WEBP_MAX_HEIGHT", "WEBP_LOSSLESS", "WEBP_ALPHA_QUALITY", "HIDDEN_WATERMARK_REMOTE_URL",
		"HIDDEN_WATERMARK_REMOTE_API_KEY", "HIDDEN_WATERMARK_REMOTE_SKIP_TLS_VERIFY",
		"WATERMARK_ENABLED", "WATERMARK_TYPE",
		"WATERMARK_TEXT", "WATERMARK_FONT_SIZE", "WATERMARK_FONT_PATH", "WATERMARK_IMAGE_PATH",
		"WATERMARK_OPACITY", "WATERMARK_POSITION", "WATERMARK_POSITION_MODE", "WATERMARK_PRECISE_X",
		"WATERMARK_PRECISE_Y", "WATERMARK_IMAGE_RATIO", "WATERMARK_COLOR", "WATERMARK_TILE_MODE",
		"USERNAME_WATERMARK_ENABLED", "USERNAME_WATERMARK_FONT_SIZE", "USERNAME_WATERMARK_FONT_PATH",
		"USERNAME_WATERMARK_OPACITY", "USERNAME_WATERMARK_POSITION", "USERNAME_WATERMARK_POSITION_MODE",
		"USERNAME_WATERMARK_PRECISE_X", "USERNAME_WATERMARK_PRECISE_Y", "USERNAME_WATERMARK_COLOR",
		"USERNAME_WATERMARK_TEXT", "HIDDEN_WATERMARK_ENABLED", "HIDDEN_WATERMARK_SECRET", "CONTENT_AUDIT_ENABLED", "DIFY_API_URL", "DIFY_API_KEY", "GEETEST_ENABLED",
		"GEETEST_CAPTCHA_ID", "GEETEST_CAPTCHA_KEY", "CREATOR_EXTENDED_EARNINGS_ENABLED",
		"CREATOR_EARNINGS_PER_VIEW", "CREATOR_EARNINGS_PER_LIKE", "CREATOR_EARNINGS_PER_COLLECT",
		"CREATOR_EARNINGS_PER_COMMENT", "CREATOR_EARNINGS_PER_FOLLOWER", "CREATOR_DAILY_EXTENDED_EARNINGS_CAP",
		"CREATOR_PLATFORM_FEE_RATE", "CREATOR_WITHDRAW_ENABLED", "CREATOR_MIN_WITHDRAW_AMOUNT",
		"VERIFICATION_COLLECT_SENSITIVE_INFO", "QUEUE_ENABLED", "REDIS_HOST", "REDIS_PORT", "REDIS_PASSWORD",
		"REDIS_DB", "REDIS_POOL_SIZE", "REDIS_MIN_IDLE_CONNS", "REDIS_CACHE_ENABLED", "REDIS_CACHE_COMMAND_TIMEOUT_MS", "REDIS_CACHE_DEFAULT_TTL_SECONDS",
		"REDIS_CACHE_L1_TTL_SECONDS", "REDIS_CACHE_L1_MAX_ENTRIES", "QUEUE_IP_LOCATION_CONCURRENCY", "QUEUE_CONTENT_AUDIT_CONCURRENCY",
		"QUEUE_GENERAL_TASK_CONCURRENCY", "QUEUE_AI_TASK_CONCURRENCY", "QUEUE_VIDEO_TRANSCODING_CONCURRENCY", "QUEUE_BATCH_NOTE_CREATE_CONCURRENCY",
		"QUEUE_AUDIT_LOG_CONCURRENCY", "QUEUE_RETRY_ATTEMPTS", "QUEUE_RETRY_DELAY",
		"ACCESS_LOG_ENABLED", "ACCESS_LOG_SCOPE", "ACCESS_LOG_BEHAVIORS", "ACCESS_LOG_RETENTION_HOURS",
		"ACCESS_LOG_BUFFER_SIZE", "ACCESS_LOG_BATCH_SIZE", "ACCESS_LOG_FLUSH_INTERVAL_MS",
		"SECURITY_AUDIT_LOG_ENABLED", "SECURITY_AUDIT_LOG_RETENTION_HOURS", "PYVIDEO_API_KEY", "VIDEO_CENTER_API_KEY",
		"PYVIDEO_UPSTREAM_URL", "CHAT_WS_ENABLED", "CHAT_WS_BASE_URL", "CHAT_WS_PROVISION_SECRET",
		"CHAT_WS_ENCRYPTION_KEY", "CHAT_WS_TICKET_TTL", "CHAT_WS_HTTP_TIMEOUT", "CHAT_MESSAGE_MAX_LENGTH",
		"IM_BASE", "IM_HMAC_SECRET",
		"IM_MESSAGE_KEY", "ALLOWED_MEDIA_DOMAINS", "RECOMMENDATION_DEBUG", "SWAGGER_DOCS_PATH", "JWT_TEST_TOKEN_PATH",
	}
	data, err := os.ReadFile(filepath.Join("..", "..", ".env.example"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, key := range required {
		if !strings.Contains(text, key+"=") && !strings.Contains(text, "# "+key+"=") {
			t.Fatalf(".env.example missing %s", key)
		}
	}
}

func withUnsetEnv(t *testing.T, keys ...string) {
	t.Helper()
	for _, key := range keys {
		old, had := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() {
			if had {
				_ = os.Setenv(key, old)
				return
			}
			_ = os.Unsetenv(key)
		})
	}
}
