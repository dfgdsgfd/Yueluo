package routes

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/handlers"
)

type RegistrationReport struct {
	RegisteredHTTPRoutes int
	WebSocketEntries     int
	NativeHTTPRoutes     int
	ProxyHTTPRoutes      int
}

func RegisterMatrixRoutes(engine *gin.Engine, matrix Matrix, native handlers.NativeHandlers) RegistrationReport {
	registerNativeRoutes(engine, native)

	engine.NoRoute(func(c *gin.Context) {
		if route, ok := matchMatrixRoute(matrix, native, c.Request.Method, c.Request.URL.Path); ok {
			routeMetadata(route)(c)
			native.MatrixRoute(route.SourceFile, route.Method, materializeMatrixPath(route.Path, native), route.Auth)(c)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
	})

	engine.NoMethod(func(c *gin.Context) {
		if route, ok := matchMatrixRoute(matrix, native, c.Request.Method, c.Request.URL.Path); ok {
			routeMetadata(route)(c)
			native.MatrixRoute(route.SourceFile, route.Method, materializeMatrixPath(route.Path, native), route.Auth)(c)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "not found"})
	})

	return RegistrationReport{
		RegisteredHTTPRoutes: matrix.Summary.TotalAPIRoutes,
		WebSocketEntries:     len(matrix.WebSockets),
		NativeHTTPRoutes:     matrix.Summary.TotalAPIRoutes,
		ProxyHTTPRoutes:      0,
	}
}

func routeMetadata(route Route) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("express_source", route.SourceFile)
		c.Set("express_line", route.Line)
		c.Set("express_method", strings.ToUpper(route.Method))
		c.Set("express_path", route.Path)
		c.Set("auth_class", route.Auth)
		c.Header("X-Gin-Migration-Mode", "native")
		c.Next()
	}
}

func isAPIPath(path string) bool {
	return len(path) == 4 && path == "/api" || len(path) > 5 && path[:5] == "/api/"
}

func RegisterNativeHealth(engine *gin.Engine, matrix Matrix) {
	startedAt := time.Now()
	engine.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":      200,
			"message":   "OK",
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
			"uptime":    time.Since(startedAt).Seconds(),
		})
	})
}

func registerNativeRoutes(engine *gin.Engine, native handlers.NativeHandlers) int {
	engine.GET("/api/ua-block/check", native.UABlockCheck)
	engine.GET("/api/diagnostics/network", native.NetworkDiagnostics)
	engine.GET("/api/maintenance/status", native.MaintenanceStatus)
	engine.POST("/api/maintenance/enter", native.MaintenanceEnter)
	engine.GET("/api/app/download-config", native.AppDownloadConfig)
	engine.GET("/api/app/check-update", native.CheckAppUpdate)
	engine.POST("/api/app/report-event", native.ReportAppEvent)
	engine.GET("/api/"+native.Config.Debug.SwaggerDocsPath, native.SwaggerDocsPage)
	engine.GET("/api/swagger-ui/swagger-ui.css", native.SwaggerUICSS)
	engine.GET("/api/swagger-ui/swagger-ui-bundle.js", native.SwaggerUIJS)
	engine.GET("/api/swagger-ui/swagger-ui-standalone-preset.js", native.SwaggerUIStandalonePreset)
	engine.GET("/api/im/ws", native.IMWebSocket)

	noteRestricted := native.OptionalAuthWithNoteGuestRestriction()
	engine.GET("/api/tags", noteRestricted, native.Tags)
	engine.GET("/api/tags/hot", noteRestricted, native.HotTags)
	engine.GET("/api/categories", noteRestricted, native.Categories)
	engine.GET("/api/categories/hot", noteRestricted, native.HotCategories)
	engine.POST("/api/categories", native.RequireAuth(), native.CreateCategory)
	engine.GET("/api/search", noteRestricted, native.Search)

	auth := native.RequireAuth()
	engine.GET("/api/comments", noteRestricted, native.Comments)
	engine.POST("/api/comments", auth, native.CreateComment)
	engine.GET("/api/comments/:id/replies", noteRestricted, native.CommentReplies)
	engine.DELETE("/api/comments/:id", auth, native.DeleteComment)

	engine.GET("/api/posts/recommended", noteRestricted, native.PostsRecommended)
	engine.GET("/api/posts/hot", noteRestricted, native.PostsHot)
	engine.GET("/api/posts/video-center", native.OptionalAuthWithVideoGuestRestriction(), native.PostsVideoCenter)
	engine.GET("/api/posts/friends", auth, native.PostsFriends)
	engine.GET("/api/posts/following", auth, native.PostsFollowing)
	engine.GET("/api/posts", noteRestricted, native.Posts)
	engine.GET("/api/posts/protection-config", native.PostsProtectionConfig)
	engine.GET("/api/posts/:id/comments", noteRestricted, native.PostComments)
	engine.GET("/api/posts/:id/purchases", auth, native.PostPurchaseUsers)
	engine.GET("/api/posts/:id/image-archive", noteRestricted, native.PostImageArchiveStatus)
	engine.GET("/api/posts/:id", noteRestricted, native.PostDetail)
	engine.POST("/api/posts", auth, native.CreatePost)
	engine.PUT("/api/posts/:id", auth, native.UpdatePost)
	engine.DELETE("/api/posts/:id", auth, native.DeletePost)
	engine.POST("/api/posts/:id/collect", auth, native.ToggleCollect)
	engine.POST("/api/posts/:id/protected-package", auth, native.CreateProtectedPackage)
	engine.GET("/api/protected-packages/:jobId", auth, native.ProtectedPackageStatus)
	engine.GET("/api/protected-packages/:jobId/events", auth, native.ProtectedPackageEvents)
	engine.GET("/api/protected-packages/:jobId/download", auth, native.DownloadProtectedPackage)
	engine.GET("/api/image-archives/:jobId/download", noteRestricted, native.DownloadPostImageArchive)

	engine.POST("/api/likes", auth, native.ToggleLike)
	engine.DELETE("/api/likes", auth, native.RemoveLike)
	engine.POST("/api/dislikes", auth, native.ToggleDislike)
	engine.GET("/api/dislikes", auth, native.DislikeStatus)
	engine.POST("/api/reports", auth, native.CreateReport)
	engine.GET("/api/reports/check", auth, native.ReportStatus)
	engine.POST("/api/image-watermark/extract", auth, native.UserExtractImageWatermark)
	engine.POST("/api/ai/jobs", auth, native.CreateAIJob)
	engine.GET("/api/ai/jobs/active", auth, native.AIActiveJob)
	engine.GET("/api/ai/jobs/:jobId", auth, native.AIJobStatus)
	engine.GET("/api/ai/jobs/:jobId/stream", auth, native.AIJobStream)
	engine.POST("/api/ai/jobs/:jobId/cancel", auth, native.CancelAIJob)
	engine.GET("/api/ai/publish-generation/settings", auth, native.AIPublishGenerationSettings)
	engine.POST("/api/ai/publish-generation", auth, native.GeneratePublishContent)
	engine.POST("/api/ai/format-markdown/stream", auth, native.FormatMarkdownStream)

	openAPI := native.RequireOpenAPIKey()
	engine.GET("/api/open/posts", openAPI, native.OpenPosts)
	engine.GET("/api/open/posts/:id", openAPI, native.OpenPost)
	engine.GET("/api/open/posts/:id/images", openAPI, native.OpenPostImages)

	engine.GET("/api/pyvideo-api-proxy/*path", native.PyVideoProxy)
	engine.GET("/api/file/*filepath", native.FileAccess)

	engine.POST("/api/upload/single", auth, native.UploadSingle)
	engine.POST("/api/users/me/avatar", auth, native.UploadAvatar)
	engine.POST("/api/users/me/banner", auth, native.UploadBanner)
	engine.POST("/api/upload/multiple", auth, native.UploadMultiple)
	engine.POST("/api/upload/video", auth, native.UploadVideo)
	engine.GET("/api/upload/chunk/config", auth, native.UploadChunkConfig)
	engine.GET("/api/upload/chunk/verify", auth, native.UploadChunkVerify)
	engine.POST("/api/upload/chunk", auth, native.UploadChunk)
	engine.POST("/api/upload/chunk/merge", auth, native.UploadChunkMerge)
	engine.POST("/api/upload/chunk/merge/image", auth, native.UploadChunkMergeImage)
	engine.POST("/api/upload/chunk/merge/apk", auth, native.UploadChunkMergeAPK)
	engine.POST("/api/upload/attachment", auth, native.UploadAttachment)
	engine.POST("/api/upload/apk", auth, native.UploadAPK)

	engine.POST("/api/feedback/upload-image", auth, native.FeedbackUploadImage)
	engine.POST("/api/feedback/upload-video", auth, native.FeedbackUploadVideo)
	engine.POST("/api/feedback", auth, native.FeedbackCreate)
	engine.GET("/api/feedback/mine", auth, native.FeedbackMine)
	engine.GET("/api/feedback/:id", auth, native.FeedbackDetail)

	engine.GET("/api/stats", native.Stats)
	engine.GET("/api/announcements", native.Announcements)
	engine.GET("/api/announcements/:id", native.Announcement)
	engine.POST("/api/license/verify", native.VerifyLicense)

	engine.GET("/api/notifications/activities", native.OptionalAuth(), native.NotificationActivities)
	engine.GET("/api/notifications", auth, native.Notifications)
	engine.GET("/api/notifications/unread-count", auth, native.NotificationUnreadCount)
	engine.PUT("/api/notifications/read-all", auth, native.MarkAllNotificationsRead)
	engine.PUT("/api/notifications/:id/read", auth, native.MarkNotificationRead)
	engine.DELETE("/api/notifications/:id", auth, native.DeleteNotification)
	engine.GET("/api/notifications/system", auth, native.SystemNotifications)
	engine.GET("/api/notifications/system/popup", auth, native.PopupSystemNotifications)
	engine.POST("/api/notifications/system/:id/confirm", auth, native.ConfirmSystemNotification)
	engine.DELETE("/api/notifications/system/:id/dismiss", auth, native.DismissSystemNotification)

	engine.GET("/api/invite/my-code", auth, native.InviteMyCode)
	engine.POST("/api/invite/click/:code", native.OptionalAuth(), native.InviteClick)
	engine.GET("/api/invite/stats", auth, native.InviteStats)
	engine.GET("/api/invite/info/:code", native.InviteInfo)
	admin := native.RequireAdmin()
	engine.GET("/api/admin/system-logs", admin, native.AdminSystemLogs)
	engine.GET("/api/admin/performance", admin, native.AdminPerformance)
	engine.GET("/api/admin/observability/events", admin, native.AdminObservabilityEvents)
	engine.GET("/api/admin/observability/access-log", admin, native.AdminObservabilityAccessLog)
	engine.GET("/api/admin/logs/access", admin, native.AdminAccessLogs)
	engine.GET("/api/admin/logs/security", admin, native.AdminSecurityAuditLogs)
	engine.GET("/api/admin/logs/access/analytics", admin, native.AdminAccessLogAnalytics)
	engine.GET("/api/admin/logs/points", admin, native.AdminPointsAuditLogs)
	engine.GET("/api/admin/logs/balance", admin, native.AdminBalanceAuditLogs)
	engine.GET("/api/admin/access-block/rules", admin, native.AdminAccessBlockRules)
	engine.POST("/api/admin/access-block/rules", admin, native.AdminAccessBlockRules)
	engine.PUT("/api/admin/access-block/rules/:id", admin, native.AdminAccessBlockRule)
	engine.DELETE("/api/admin/access-block/rules/:id", admin, native.AdminAccessBlockRule)
	engine.POST("/api/admin/access-block/rules/batch", admin, native.AdminAccessBlockRulesBatch)
	engine.GET("/api/admin/access-block/import-sources", admin, native.AdminAccessBlockImportSources)
	engine.POST("/api/admin/access-block/import-sources", admin, native.AdminAccessBlockImportSources)
	engine.PUT("/api/admin/access-block/import-sources/:id", admin, native.AdminAccessBlockImportSource)
	engine.DELETE("/api/admin/access-block/import-sources/:id", admin, native.AdminAccessBlockImportSource)
	engine.POST("/api/admin/access-block/import-sources/:id/sync", admin, native.AdminAccessBlockImportSourceSync)
	engine.POST("/api/admin/ai/generate/stream", admin, native.AdminAIGenerateStream)
	engine.POST("/api/admin/ai/moderation/debug", admin, native.AdminAIModerationDebug)
	engine.GET("/api/admin/ai/jobs", admin, native.AdminAIJobs)
	engine.POST("/api/admin/ai/jobs", admin, native.AdminCreateAIJob)
	engine.GET("/api/admin/ai/jobs/:jobId", admin, native.AdminAIJobStatus)
	engine.GET("/api/admin/ai/jobs/:jobId/stream", admin, native.AdminAIJobStream)
	engine.POST("/api/admin/ai/jobs/:jobId/cancel", admin, native.AdminCancelAIJob)
	engine.GET("/api/admin/ai/settings", admin, native.AdminAISettings)
	engine.PUT("/api/admin/ai/settings", admin, native.AdminAISettings)
	engine.GET("/api/admin/ai/logs", admin, native.AdminAILogs)
	engine.POST("/api/admin/users/batch-generate", admin, native.AdminUsersBatchGenerate)
	engine.POST("/api/admin/users/:id/points", admin, native.AdminUserPointsUpdate)
	engine.GET("/api/admin/maintenance", admin, native.AdminMaintenance)
	engine.PUT("/api/admin/maintenance", admin, native.AdminMaintenanceUpdate)
	engine.POST("/api/admin/maintenance/rotate-entry", admin, native.AdminMaintenanceRotateEntry)
	engine.GET("/api/admin/database/overview", admin, native.AdminDatabaseOverview)
	engine.GET("/api/admin/database/tables", admin, native.AdminDatabaseTables)
	engine.GET("/api/admin/database/tables/:table/columns", admin, native.AdminDatabaseColumns)
	engine.GET("/api/admin/database/index-audit", admin, native.AdminDatabaseIndexAudit)
	engine.POST("/api/admin/database/repair", admin, native.AdminDatabaseRepair)
	engine.GET("/api/admin/database/vacuum-config", admin, native.AdminDatabaseVacuumConfig)
	engine.PUT("/api/admin/database/vacuum-config", admin, native.AdminDatabaseVacuumConfigUpdate)
	engine.POST("/api/admin/database/vacuum-analyze", admin, native.AdminDatabaseVacuumAnalyze)
	engine.GET("/api/admin/redis-maintenance", admin, native.AdminRedisMaintenance)
	engine.PUT("/api/admin/redis-maintenance", admin, native.AdminRedisMaintenanceUpdate)
	engine.POST("/api/admin/redis-maintenance/run", admin, native.AdminRedisMaintenanceRun)
	engine.GET("/api/admin/file-recycle-bin", admin, native.AdminFileRecycleBin)
	engine.GET("/api/admin/file-recycle-bin/:id/inspect", admin, native.AdminFileRecycleBin)
	engine.GET("/api/admin/file-recycle-bin/:id/preview", admin, native.AdminFileRecycleBin)
	engine.GET("/api/admin/file-recycle-bin/:id/download", admin, native.AdminFileRecycleBin)
	engine.DELETE("/api/admin/file-recycle-bin", admin, native.AdminFileRecycleBin)
	engine.DELETE("/api/admin/file-recycle-bin/:id", admin, native.AdminFileRecycleBin)
	engine.POST("/api/admin/file-recycle-bin/run-cleanup", admin, native.AdminFileRecycleBin)
	engine.GET("/api/invite/admin/list", admin, native.InviteAdminList)
	engine.PATCH("/api/invite/admin/:id/toggle", admin, native.InviteAdminToggle)
	engine.POST("/api/invite/admin/reward", admin, native.InviteAdminReward)
	engine.GET("/api/invite/admin/overview", admin, native.InviteAdminOverview)

	engine.GET("/api/coupon/my", auth, native.CouponMy)
	engine.POST("/api/coupon/claim", auth, native.CouponClaim)
	engine.POST("/api/coupon/validate", auth, native.CouponValidate)
	engine.POST("/api/coupon/use", auth, native.CouponUse)
	engine.GET("/api/coupon/admin/list", admin, native.CouponAdminList)
	engine.POST("/api/coupon/admin/create", admin, native.CouponAdminCreate)
	engine.GET("/api/coupon/admin/stats", admin, native.CouponAdminStats)
	engine.PUT("/api/coupon/admin/:id", admin, native.CouponAdminUpdate)
	engine.DELETE("/api/coupon/admin/:id", admin, native.CouponAdminDelete)
	engine.POST("/api/coupon/admin/:id/issue", admin, native.CouponAdminIssue)
	engine.GET("/api/coupon/admin/:id/usages", admin, native.CouponAdminUsages)

	engine.GET("/api/points/overview", auth, native.PointsOverview)
	engine.GET("/api/points/logs", auth, native.PointsLogs)
	engine.POST("/api/points/gift-cards/:productId/redeem", auth, native.PointsRedeemGiftCard)
	engine.GET("/api/points/redemptions", auth, native.PointsRedemptions)
	engine.GET("/api/points/admin/stats", admin, native.PointsAdminStats)
	engine.GET("/api/points/admin/settings", admin, native.PointsAdminSettings)
	engine.PUT("/api/points/admin/settings", admin, native.PointsAdminUpdateSettings)
	engine.GET("/api/points/admin/tasks", admin, native.PointsAdminTasks)
	engine.POST("/api/points/admin/tasks", admin, native.PointsAdminCreateTask)
	engine.PUT("/api/points/admin/tasks/:id", admin, native.PointsAdminUpdateTask)
	engine.DELETE("/api/points/admin/tasks/:id", admin, native.PointsAdminDeleteTask)
	engine.POST("/api/points/admin/clear-balances", admin, native.PointsAdminClearBalances)
	engine.POST("/api/points/admin/reset-task-progress", admin, native.PointsAdminResetTaskProgress)
	engine.GET("/api/points/admin/achievement-rules", admin, native.PointsAdminAchievementRules)
	engine.POST("/api/points/admin/achievement-rules", admin, native.PointsAdminCreateAchievementRule)
	engine.PUT("/api/points/admin/achievement-rules/:id", admin, native.PointsAdminUpdateAchievementRule)
	engine.DELETE("/api/points/admin/achievement-rules/:id", admin, native.PointsAdminDeleteAchievementRule)
	engine.GET("/api/points/admin/gift-card-products", admin, native.PointsAdminGiftCardProducts)
	engine.POST("/api/points/admin/gift-card-products", admin, native.PointsAdminCreateGiftCardProduct)
	engine.PUT("/api/points/admin/gift-card-products/:id", admin, native.PointsAdminUpdateGiftCardProduct)
	engine.DELETE("/api/points/admin/gift-card-products/:id", admin, native.PointsAdminDeleteGiftCardProduct)
	engine.POST("/api/points/admin/gift-card-products/:id/import-codes", admin, native.PointsAdminImportGiftCardCodes)
	engine.GET("/api/points/admin/gift-card-products/:id/codes", admin, native.PointsAdminGiftCardCodes)
	engine.GET("/api/points/admin/redemptions", admin, native.PointsAdminRedemptions)

	engine.GET("/api/admin/system-update/status", admin, native.AdminSystemUpdateStatus)
	engine.POST("/api/admin/system-update/check", admin, native.AdminSystemUpdateCheck)
	engine.GET("/api/admin/system-update/releases", admin, native.AdminSystemUpdateReleases)
	engine.PUT("/api/admin/system-update/config", admin, native.AdminSystemUpdateSaveConfig)
	engine.POST("/api/admin/system-update/run", admin, native.AdminSystemUpdateRun)

	engine.GET("/api/balance/config", native.BalanceConfig)
	engine.GET("/api/balance/recharge-config", native.BalanceRechargeConfig)
	engine.GET("/api/balance/local-points", auth, native.BalanceLocalPoints)
	engine.GET("/api/balance/user-balance", auth, native.BalanceUserBalance)
	engine.POST("/api/balance/purchase-content", auth, native.BalancePurchaseContent)
	engine.GET("/api/balance/orders", auth, native.BalanceOrders)
	engine.GET("/api/balance/check-purchase/:postId", auth, native.BalanceCheckPurchase)
	engine.GET("/api/admin/balance-transactions", admin, native.AdminBalanceTransactions)
	engine.POST("/api/admin/balance-transactions/:id/compensate", admin, native.AdminBalanceTransactionCompensate)

	engine.GET("/api/creator-center/config", native.CreatorConfig)
	engine.GET("/api/creator-center/overview", auth, native.CreatorOverview)
	engine.GET("/api/creator-center/trends", auth, native.CreatorTrends)
	engine.GET("/api/creator-center/stats", auth, native.CreatorStats)
	engine.GET("/api/creator-center/earnings-log", auth, native.CreatorEarningsLog)
	engine.GET("/api/creator-center/paid-content", auth, native.CreatorPaidContent)
	engine.POST("/api/creator-center/withdraw", auth, native.CreatorWithdraw)
	engine.POST("/api/creator-center/claim-incentive", auth, native.CreatorClaimIncentive)
	engine.GET("/api/creator-center/quality-rewards", auth, native.CreatorQualityRewards)

	engine.GET("/api/withdraw/wallet", auth, native.WithdrawWallet)
	engine.GET("/api/withdraw/payment-code", auth, native.WithdrawPaymentCode)
	engine.POST("/api/withdraw/payment-code", auth, native.WithdrawSavePaymentCode)
	engine.POST("/api/withdraw/apply", auth, native.WithdrawApply)
	engine.GET("/api/withdraw/orders", auth, native.WithdrawOrders)
	engine.GET("/api/withdraw/admin/orders", admin, native.WithdrawAdminOrders)
	engine.PUT("/api/withdraw/admin/orders/:id/approve", admin, native.WithdrawAdminApprove)
	engine.PUT("/api/withdraw/admin/orders/:id/reject", admin, native.WithdrawAdminReject)
	engine.PUT("/api/withdraw/admin/orders/:id/payout", admin, native.WithdrawAdminPayout)
	engine.GET("/api/admin/dashboard/overview", admin, native.AdminDashboardOverview)
	engine.GET("/api/admin/dashboard/trends", admin, native.AdminDashboardTrends)
	engine.GET("/api/admin/dashboard/hot-content", admin, native.AdminDashboardHotContent)
	return 185
}

func materializeMatrixPath(path string, native handlers.NativeHandlers) string {
	path = strings.ReplaceAll(path, "${JWT_TEST_TOKEN_PATH}", native.Config.Debug.JWTTestTokenPath)
	path = strings.ReplaceAll(path, "${SWAGGER_DOCS_PATH}", native.Config.Debug.SwaggerDocsPath)
	if before, ok := strings.CutSuffix(path, "/*"); ok {
		path = before + "/*path"
	}
	return path
}

func matchMatrixRoute(matrix Matrix, native handlers.NativeHandlers, method string, requestPath string) (Route, bool) {
	method = strings.ToUpper(method)
	bestScore := -1
	var best Route
	for _, route := range matrix.Routes {
		if strings.ToUpper(route.Method) != method && strings.ToUpper(route.Method) != "ALL" {
			continue
		}
		if score, ok := matrixPathMatchScore(materializeMatrixPath(route.Path, native), requestPath); ok && score > bestScore {
			best = route
			bestScore = score
		}
	}
	if bestScore < 0 {
		return Route{}, false
	}
	return best, true
}

func matrixPathMatches(pattern string, requestPath string) bool {
	_, ok := matrixPathMatchScore(pattern, requestPath)
	return ok
}

func matrixPathMatchScore(pattern string, requestPath string) (int, bool) {
	if pattern == requestPath {
		return 100000 + len(pattern), true
	}
	if strings.Contains(pattern, "${") {
		return 0, false
	}
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	requestParts := strings.Split(strings.Trim(requestPath, "/"), "/")
	score := 0
	for i := range patternParts {
		if strings.HasPrefix(patternParts[i], "*") {
			return score, len(requestParts) >= i
		}
		if i >= len(requestParts) {
			return 0, false
		}
		if strings.HasPrefix(patternParts[i], ":") {
			if requestParts[i] == "" {
				return 0, false
			}
			score++
			continue
		}
		if patternParts[i] != requestParts[i] {
			return 0, false
		}
		score += 10
	}
	return score, len(patternParts) == len(requestParts)
}
