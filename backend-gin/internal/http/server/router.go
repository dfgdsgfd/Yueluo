package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/http/handlers"
	appmiddleware "yuem-go/backend-gin/internal/http/middleware"
	"yuem-go/backend-gin/internal/http/routes"
	"yuem-go/backend-gin/internal/localization"
	"yuem-go/backend-gin/internal/services"
	"yuem-go/backend-gin/internal/storage"
)

func NewRouter(cfg config.Config, logger *zap.Logger) (*gin.Engine, error) {
	engine, _, err := NewRouterWithShutdown(cfg, logger)
	return engine, err
}

func NewRouterWithShutdown(cfg config.Config, logger *zap.Logger) (*gin.Engine, func(), error) {
	requestLogLevel := parseLogLevel(cfg.Server.RequestLogLevel)
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	matrix, err := routes.LoadMatrix()
	if err != nil {
		return nil, nil, fmt.Errorf("load route matrix: %w", err)
	}
	if matrix.Summary.TotalAPIRoutes != 508 {
		return nil, nil, fmt.Errorf("route matrix mismatch: got %d routes, want 508", matrix.Summary.TotalAPIRoutes)
	}
	db, err := storage.OpenDatabase(cfg.Database)
	if err != nil {
		return nil, nil, fmt.Errorf("open database: %w", err)
	}
	redisStore := services.NewRedisStore(cfg.Redis)
	settings := services.NewSettingsService(db, redisStore)
	settingsCtx, cancelSettingsLoad := context.WithTimeout(context.Background(), time.Second)
	settings.Load(settingsCtx)
	cancelSettingsLoad()
	aiService := services.NewAIService(db, settings)
	auth := services.NewAuthService(db, redisStore, cfg.Auth, settings)
	cache := services.NewCache()
	apiKeys := handlers.NewAPIKeyAuthCache()
	uploadPaths := handlers.NewUploadPathResolver(cfg)
	hiddenWatermarkSecret := cfg.WebP.HiddenWatermark.Secret
	if hiddenWatermarkSecret == "" {
		hiddenWatermarkSecret = cfg.Upload.FileSigning.Secret
	}
	imageProcessor := services.NewImageProcessorWithRemote(
		settings,
		hiddenWatermarkSecret,
		cfg.Upload.Image.MaxSizeBytes,
		logger,
		services.HiddenWatermarkRemoteClientConfigFromConfig(cfg),
	)
	queue := services.NewQueueService(db, cfg, settings, aiService)
	balanceCenter := services.NewBalanceCenterService(db, cfg.Balance)
	tempStorage := services.NewTempStorageService(cfg.Upload.Temp)
	if err := tempStorage.Start(); err != nil {
		return nil, nil, fmt.Errorf("start temporary storage: %w", err)
	}
	uploadAssets := services.NewUploadAssetService(db, cfg, logger)
	uploadAssets.Start()
	fileRecycle := services.NewFileRecycleServiceWithSettings(db, cfg, logger, settings)
	fileRecycleCleanup := services.NewFileRecycleCleanupService(fileRecycle, logger)
	fileRecycleCleanup.Start()
	if err := queue.Start(); err != nil && logger != nil {
		logger.Warn("asynq queue service did not start", zap.Error(err))
	}
	auditLog := services.NewAuditLogService(queue, cfg, logger)
	observe := services.NewObservabilityService(redisStore, cfg.Observe, logger, settings)
	accessBlock := services.NewAccessBlockService(db, redisStore, logger, cfg.AccessBlock.Disabled)
	accessBlock.ConfigureImports(services.AccessBlockImportConfig{
		Disabled:              cfg.AccessBlock.ImportDisabled,
		MinInterval:           cfg.AccessBlock.ImportMinInterval,
		HTTPTimeout:           cfg.AccessBlock.ImportHTTPTimeout,
		MaxBytes:              cfg.AccessBlock.ImportMaxBytes,
		SchedulerInterval:     cfg.AccessBlock.ImportSchedulerInterval,
		DefaultUpdateInterval: cfg.AccessBlock.ImportDefaultInterval,
	})
	accessBlockCtx, cancelAccessBlockLoad := context.WithTimeout(context.Background(), time.Second)
	if err := accessBlock.Load(accessBlockCtx); err != nil && logger != nil {
		logger.Warn("load access block rules failed", zap.Error(err))
	}
	cancelAccessBlockLoad()
	accessBlock.StartVersionPolling(cfg.AccessBlock.VersionPollInterval)
	accessBlock.StartImportScheduler()
	if db != nil {
		db.Logger = services.NewObservabilityGormLogger(db.Logger, observe, cfg.Database.IgnoreRecordNotFound)
	}
	dbMaintenance := services.NewDatabaseMaintenanceService(db, settings, logger)
	dbMaintenance.Start()
	redisMaintenance := services.NewRedisMaintenanceService(redisStore, queue, settings, logger)
	redisMaintenance.Start()
	cleanup := func() {
		tempStorage.Close()
		uploadAssets.Close()
		auditLog.Close()
		_ = queue.Close()
		observe.Close()
		accessBlock.Close()
		dbMaintenance.Close()
		redisMaintenance.Close()
		fileRecycleCleanup.Close()
	}

	engine := gin.New()
	engine.Use(fatalPanicRecovery(logger))
	engine.Use(appmiddleware.RequestID())
	engine.Use(ginLogger(logger, requestLogLevel, cfg.Server.RequestLogSilence))
	engine.Use(observabilityMiddleware(observe, cfg.Server.ClientIPHeaders))
	engine.Use(auditLogMiddleware(auditLog, cfg.Server.ClientIPHeaders))
	engine.Use(appmiddleware.AccessBlock(accessBlock, func(c *gin.Context) string {
		return requestClientIP(c, cfg.Server.ClientIPHeaders)
	}, accessBlockLogger{audit: auditLog, observe: observe, clientIPHeaders: cfg.Server.ClientIPHeaders}))
	engine.Use(cors.New(cors.Config{
		AllowOrigins: cfg.CORS.Origins,
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodPatch},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "X-API-Key", "X-App-Locale", "Range"},
		ExposeHeaders: []string{
			"Content-Length",
			"Content-Range",
			"Accept-Ranges",
			"ETag",
			"Location",
			appmiddleware.RequestIDHeader,
			appmiddleware.AccessBlockHeader,
			appmiddleware.AccessBlockRuleIDHeader,
			appmiddleware.AccessBlockActionHeader,
			appmiddleware.AccessBlockStatusCodeHeader,
		},
		AllowCredentials: true,
		CustomSchemas:    []string{"capacitor://", "ionic://"},
		MaxAge:           12 * time.Hour,
	}))
	engine.Use(gzip.Gzip(gzip.DefaultCompression))
	engine.Use(securityHeaders())
	engine.Use(databaseAvailability(db, cfg))
	engine.Use(maintenanceMode(settings, cfg))
	engine.Use(appmiddleware.UABlock(settings))

	routes.RegisterNativeHealth(engine, matrix)
	native := handlers.NativeHandlers{
		DB:           db,
		Config:       cfg,
		Cache:        cache,
		Redis:        redisStore,
		APIKeys:      apiKeys,
		UploadPaths:  uploadPaths,
		Queue:        queue,
		Settings:     settings,
		AI:           aiService,
		Images:       imageProcessor,
		Auth:         auth,
		Observe:      observe,
		AuditLog:     auditLog,
		AccessBlock:  accessBlock,
		GeoIP:        services.NewGeoIPService(cfg.GeoIP),
		Balance:      balanceCenter,
		TempStorage:  tempStorage,
		UploadAssets: uploadAssets,
		FileRecycle:  fileRecycle,
		RedisCare:    redisMaintenance,

		FileAccessGroup: &singleflight.Group{},
		PostListGroup:   &singleflight.Group{},
		SearchGroup:     &singleflight.Group{},
	}
	report := routes.RegisterMatrixRoutes(engine, matrix, native)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		changed, err := native.BackfillProfileImageHashes(ctx, 200)
		if err != nil {
			if logger != nil {
				logger.Warn("profile image hash backfill failed", zap.Error(err))
			}
			return
		}
		if changed > 0 && logger != nil {
			logger.Info("profile image hash backfill completed", zap.Int64("updated_users", changed))
		}
	}()
	registerCompatibilityStatus(engine, matrix, report)
	return engine, cleanup, nil
}

type fatalPanicReport struct {
	Value     any
	Method    string
	Path      string
	RequestID string
	Stack     []byte
	ExitCode  int
}

var fatalPanicExit = func(report fatalPanicReport) {
	os.Exit(report.ExitCode)
}

func fatalPanicRecovery(logger *zap.Logger) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				report := fatalPanicReport{
					Value:     recovered,
					Method:    c.Request.Method,
					Path:      c.Request.URL.Path,
					RequestID: c.Writer.Header().Get(appmiddleware.RequestIDHeader),
					Stack:     debug.Stack(),
					ExitCode:  2,
				}
				logger.Error("fatal panic in request",
					zap.Any("panic", report.Value),
					zap.String("method", report.Method),
					zap.String("path", report.Path),
					zap.String("request_id", report.RequestID),
					zap.ByteString("stack", report.Stack),
				)
				_ = logger.Sync()
				fatalPanicExit(report)
			}
		}()
		c.Next()
	}
}

func ginLogger(logger *zap.Logger, threshold zapcore.Level, silencePaths []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		method := c.Request.Method
		path := c.Request.URL.Path
		status := c.Writer.Status()
		if shouldSuppressRequestInfo(method, path, status, latency) {
			return
		}
		intended := requestLogLevelForStatus(status)
		for _, prefix := range silencePaths {
			if status < http.StatusBadRequest && strings.HasPrefix(path, prefix) {
				intended = zapcore.DebugLevel
				break
			}
		}
		if intended < threshold {
			return
		}
		if ce := logger.Check(intended, "request"); ce != nil {
			fields := []zap.Field{
				zap.String("method", method),
				zap.String("path", path),
				zap.Int("status", status),
				zap.Duration("latency", latency),
				zap.String("request_id", c.Writer.Header().Get(appmiddleware.RequestIDHeader)),
			}
			if route := c.FullPath(); route != "" {
				fields = append(fields, zap.String("route", route))
			}
			if errors := requestErrorMessages(c); len(errors) > 0 {
				fields = append(fields, zap.Strings("errors", errors))
			}
			ce.Write(fields...)
		}
	}
}

func requestErrorMessages(c *gin.Context) []string {
	if c == nil || len(c.Errors) == 0 {
		return nil
	}
	messages := make([]string, 0, len(c.Errors))
	for _, item := range c.Errors {
		if item == nil || item.Err == nil {
			continue
		}
		messages = append(messages, item.Err.Error())
	}
	return messages
}

func requestLogLevelForStatus(status int) zapcore.Level {
	if status >= http.StatusInternalServerError {
		return zapcore.ErrorLevel
	}
	if status >= http.StatusBadRequest {
		return zapcore.WarnLevel
	}
	return zapcore.InfoLevel
}

func observabilityMiddleware(observe *services.ObservabilityService, clientIPHeaders []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		var finish func()
		ip := requestClientIP(c, clientIPHeaders)
		if observe != nil && serverAPIPath(c.Request.URL.Path) {
			finish = observe.BeginRequest(ip)
		}
		c.Next()
		if finish != nil {
			finish()
		}
		if observe == nil || !serverAPIPath(c.Request.URL.Path) {
			return
		}
		latency := time.Since(start)
		requestID := c.Writer.Header().Get(appmiddleware.RequestIDHeader)
		observe.RecordRequest(c.Request.Context(), services.RequestMetric{
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Route:     c.FullPath(),
			Status:    c.Writer.Status(),
			LatencyMS: latency.Milliseconds(),
			RequestID: requestID,
			CreatedAt: time.Now(),
		})
		access := services.RecentAccessLogEvent{
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Status:    c.Writer.Status(),
			LatencyMS: latency.Milliseconds(),
			IP:        ip,
			UserAgent: c.Request.UserAgent(),
			RequestID: requestID,
			CreatedAt: time.Now(),
		}
		if !shouldWriteSystemLog(c.Request.Method, c.Request.URL.Path, c.Writer.Status()) {
			if rawUser, ok := c.Get("user"); ok {
				if user, ok := rawUser.(*services.RequestUser); ok && user != nil {
					access.UserID = user.UserID
					if access.UserID == "" {
						access.UserID = strconv.FormatInt(user.ID, 10)
					}
					access.Username = user.Username
					access.Nickname = user.Nickname
					access.UserType = user.Type
				}
			}
			observe.RecordAccess(c.Request.Context(), access)
			return
		}
		event := services.SystemLogEvent{
			Type:      systemLogType(c.Request.Method, c.Request.URL.Path, c.Writer.Status()),
			Level:     systemLogLevel(c.Writer.Status()),
			Message:   c.Request.Method + " " + c.Request.URL.Path,
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Status:    c.Writer.Status(),
			LatencyMS: latency.Milliseconds(),
			IP:        ip,
			UserAgent: c.Request.UserAgent(),
			RequestID: requestID,
			CreatedAt: time.Now(),
		}
		if rawUser, ok := c.Get("user"); ok {
			if user, ok := rawUser.(*services.RequestUser); ok && user != nil {
				event.ActorID = user.ID
				event.ActorType = user.Type
				access.UserID = user.UserID
				if access.UserID == "" {
					access.UserID = strconv.FormatInt(user.ID, 10)
				}
				access.Username = user.Username
				access.Nickname = user.Nickname
				access.UserType = user.Type
			}
		}
		observe.RecordAccess(c.Request.Context(), access)
		observe.Log(event)
	}
}

func serverAPIPath(path string) bool {
	return path == "/api" || strings.HasPrefix(path, "/api/")
}

func shouldWriteSystemLog(method, path string, status int) bool {
	return status >= http.StatusInternalServerError ||
		strings.HasPrefix(path, "/api/admin/") ||
		(method == http.MethodPost && path == "/api/auth/admin/login")
}

func systemLogType(method, path string, status int) string {
	if status >= http.StatusInternalServerError {
		return "system_error"
	}
	if method == http.MethodPost && path == "/api/auth/admin/login" {
		return "admin_login"
	}
	if strings.HasPrefix(path, "/api/admin/") {
		return "admin_api_access"
	}
	return "api_access"
}

func systemLogLevel(status int) string {
	if status >= http.StatusInternalServerError {
		return "error"
	}
	if status >= http.StatusBadRequest {
		return "warn"
	}
	return "info"
}

func parseLogLevel(level string) zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return zapcore.DebugLevel
	case "warn", "warning":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "dpanic":
		return zapcore.DPanicLevel
	case "panic":
		return zapcore.PanicLevel
	case "fatal":
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

func shouldSuppressRequestInfo(method, path string, status int, latency time.Duration) bool {
	if status >= http.StatusBadRequest || latency >= 500*time.Millisecond {
		return false
	}
	if method != http.MethodPost || !strings.HasPrefix(path, "/api/im/messages/") {
		return false
	}
	return strings.HasSuffix(path, "/delivered") || strings.HasSuffix(path, "/read")
}

func databaseAvailability(db *gorm.DB, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db != nil || databaseOptionalPath(c.Request.URL.Path, cfg) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"code":    http.StatusInternalServerError,
			"message": "\u6570\u636e\u5e93\u672a\u914d\u7f6e",
		})
	}
}

func databaseOptionalPath(path string, cfg config.Config) bool {
	if path == "/api/health" || path == "/api/diagnostics/network" || path == "/_gin_migration/status" {
		return true
	}
	switch path {
	case "/api/auth/auth-config", "/api/auth/email-config", "/api/auth/captcha", "/api/auth/oauth2/login", "/api/auth/oauth2/callback", "/api/maintenance/status", "/api/maintenance/enter":
		return true
	}
	swaggerPath := "/api/" + cfg.Debug.SwaggerDocsPath
	if path == swaggerPath || path == swaggerPath+".json" {
		return true
	}
	return strings.HasPrefix(path, "/api/swagger-ui/")
}

func maintenanceMode(settings *services.SettingsService, cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		state := services.ReadMaintenanceStateForLocale(settings, localization.ResolveRequest(c.Request))
		if !state.Enabled || !maintenanceProtectedPath(c.Request.URL.Path) {
			c.Next()
			return
		}
		if raw, err := c.Cookie(services.MaintenanceBypassCookie); err == nil && services.ValidMaintenanceBypass(raw, state.EntryCode, cfg.Auth.JWTSecret) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"code":    http.StatusServiceUnavailable,
			"message": "maintenance.enabled",
			"data": gin.H{
				"enabled":          true,
				"estimated_end_at": state.EstimatedEndAt,
				"message":          state.Message,
				"now":              time.Now().UTC().Format(time.RFC3339Nano),
			},
		})
	}
}

func maintenanceProtectedPath(path string) bool {
	if !serverAPIPath(path) {
		return false
	}
	if path == "/api/health" ||
		path == "/api/diagnostics/network" ||
		path == "/api/maintenance/status" ||
		path == "/api/maintenance/enter" ||
		strings.HasPrefix(path, "/api/admin/") ||
		strings.HasPrefix(path, "/api/auth/admin/") ||
		strings.HasPrefix(path, "/api/file/") ||
		strings.HasPrefix(path, "/api/pyvideo-api-proxy/") ||
		strings.HasPrefix(path, "/api/swagger-ui/") {
		return false
	}
	return true
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

func registerCompatibilityStatus(engine *gin.Engine, matrix routes.Matrix, report routes.RegistrationReport) {
	engine.GET("/_gin_migration/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "OK",
			"data": gin.H{
				"mode":                   "native-gin",
				"express_routes":         matrix.Summary.TotalAPIRoutes,
				"registered_http_routes": report.RegisteredHTTPRoutes,
				"native_http_routes":     report.NativeHTTPRoutes,
				"proxy_http_routes":      report.ProxyHTTPRoutes,
				"websocket_entries":      report.WebSocketEntries,
				"database_schema_policy": "gorm auto migrate on startup when DB_AUTO_MIGRATE=true",
				"final_gate":             "proxy_http_routes is 0; Express is not required at runtime. Functional parity is guarded by route and module tests.",
			},
		})
	})
}
