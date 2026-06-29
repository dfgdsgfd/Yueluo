package routes

import "testing"

func TestLoadMatrixCounts(t *testing.T) {
	matrix, err := LoadMatrix()
	if err != nil {
		t.Fatalf("LoadMatrix() error = %v", err)
	}
	if matrix.Summary.TotalAPIRoutes != 508 {
		t.Fatalf("TotalAPIRoutes = %d, want 508", matrix.Summary.TotalAPIRoutes)
	}
	if len(matrix.Routes) != 508 {
		t.Fatalf("len(Routes) = %d, want 508", len(matrix.Routes))
	}
	if len(matrix.WebSockets) != 1 {
		t.Fatalf("len(WebSockets) = %d, want 1", len(matrix.WebSockets))
	}
	if matrix.WebSockets[0].Path != "/api/im/ws" {
		t.Fatalf("websocket path = %q, want /api/im/ws", matrix.WebSockets[0].Path)
	}
	if matrix.Summary.ModuleCounts["backend/routes/admin.js"] != 269 {
		t.Fatalf("admin route count = %d, want 269", matrix.Summary.ModuleCounts["backend/routes/admin.js"])
	}
	if matrix.Summary.ModuleCounts["backend/routes/ai.js"] != 8 {
		t.Fatalf("AI route count = %d, want 8", matrix.Summary.ModuleCounts["backend/routes/ai.js"])
	}
	foundImageWatermarkExtract := false
	foundUserImageWatermarkExtract := false
	foundProtectedPackageCreate := false
	foundProtectedPackageEvents := false
	foundPostImageArchiveStatus := false
	foundPostImageArchiveDownload := false
	foundPostPurchases := false
	foundBalanceAudit := false
	foundPointsLedgerAudit := false
	foundMoonCoinLedgerAudit := false
	foundAvatarUpload := false
	foundBannerUpload := false
	foundAdminPointsAdjustment := false
	foundAdminSingleOnboardingReset := false
	foundRedisMaintenanceGet := false
	foundRedisMaintenancePut := false
	foundRedisMaintenanceRun := false
	foundNetworkDiagnostics := false
	foundAIFormatMarkdown := false
	foundAdminAIGenerate := false
	foundAdminAISettingsGet := false
	foundAdminAISettingsPut := false
	foundAdminAILogs := false
	foundAdminAIModerationDebug := false
	foundAccessBlockList := false
	foundAccessBlockCreate := false
	foundAccessBlockUpdate := false
	foundAccessBlockDelete := false
	foundAccessBlockBatch := false
	foundAccessBlockImportList := false
	foundAccessBlockImportCreate := false
	foundAccessBlockImportUpdate := false
	foundAccessBlockImportDelete := false
	foundAccessBlockImportSync := false
	foundAIJobCreate := false
	foundAIJobActive := false
	foundAIJobStatus := false
	foundAIJobStream := false
	foundAIJobCancel := false
	foundAIPublishGenerationSettings := false
	foundAIPublishGeneration := false
	foundAdminAIJobList := false
	foundAdminAIJobCreate := false
	foundAdminAIJobStatus := false
	foundAdminAIJobStream := false
	foundAdminAIJobCancel := false
	foundAdminUserBatchGenerate := false
	foundAIModerationLogs := false
	foundFileRecycleInspect := false
	foundFileRecyclePreview := false
	foundFileRecycleDownload := false
	for _, route := range matrix.Routes {
		if route.Method == "POST" && route.Path == "/api/admin/image-watermark/extract" && route.Auth == "admin" {
			foundImageWatermarkExtract = true
		}
		if route.Method == "POST" && route.Path == "/api/image-watermark/extract" && route.Auth == "user" {
			foundUserImageWatermarkExtract = true
		}
		if route.Method == "POST" && route.Path == "/api/posts/:id/protected-package" && route.Auth == "user" {
			foundProtectedPackageCreate = true
		}
		if route.Method == "GET" && route.Path == "/api/protected-packages/:jobId/events" && route.Auth == "user" {
			foundProtectedPackageEvents = true
		}
		if route.Method == "GET" && route.Path == "/api/posts/:id/image-archive" && route.Auth == "optional-note-guest-restricted" {
			foundPostImageArchiveStatus = true
		}
		if route.Method == "GET" && route.Path == "/api/image-archives/:jobId/download" && route.Auth == "optional-note-guest-restricted" {
			foundPostImageArchiveDownload = true
		}
		if route.Method == "GET" && route.Path == "/api/posts/:id/purchases" && route.Auth == "user" {
			foundPostPurchases = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/balance-transactions" && route.Auth == "admin" {
			foundBalanceAudit = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/logs/points" && route.Auth == "admin" {
			foundPointsLedgerAudit = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/logs/balance" && route.Auth == "admin" {
			foundMoonCoinLedgerAudit = true
		}
		if route.Method == "POST" && route.Path == "/api/users/me/avatar" && route.Auth == "user" {
			foundAvatarUpload = true
		}
		if route.Method == "POST" && route.Path == "/api/users/me/banner" && route.Auth == "user" {
			foundBannerUpload = true
		}
		if route.Method == "POST" && route.Path == "/api/admin/users/:id/points" && route.Auth == "admin" {
			foundAdminPointsAdjustment = true
		}
		if route.Method == "POST" && route.Path == "/api/admin/users/:id/reset-onboarding" && route.Auth == "admin" {
			foundAdminSingleOnboardingReset = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/redis-maintenance" && route.Auth == "admin" {
			foundRedisMaintenanceGet = true
		}
		if route.Method == "PUT" && route.Path == "/api/admin/redis-maintenance" && route.Auth == "admin" {
			foundRedisMaintenancePut = true
		}
		if route.Method == "POST" && route.Path == "/api/admin/redis-maintenance/run" && route.Auth == "admin" {
			foundRedisMaintenanceRun = true
		}
		if route.Method == "GET" && route.Path == "/api/diagnostics/network" && route.Auth == "public" {
			foundNetworkDiagnostics = true
		}
		if route.Method == "POST" && route.Path == "/api/ai/format-markdown/stream" && route.Auth == "user" {
			foundAIFormatMarkdown = true
		}
		if route.Method == "POST" && route.Path == "/api/admin/ai/generate/stream" && route.Auth == "admin" {
			foundAdminAIGenerate = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/ai/settings" && route.Auth == "admin" {
			foundAdminAISettingsGet = true
		}
		if route.Method == "PUT" && route.Path == "/api/admin/ai/settings" && route.Auth == "admin" {
			foundAdminAISettingsPut = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/ai/logs" && route.Auth == "admin" {
			foundAdminAILogs = true
		}
		if route.Method == "POST" && route.Path == "/api/admin/ai/moderation/debug" && route.Auth == "admin" {
			foundAdminAIModerationDebug = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/access-block/rules" && route.Auth == "admin" {
			foundAccessBlockList = true
		}
		if route.Method == "POST" && route.Path == "/api/admin/access-block/rules" && route.Auth == "admin" {
			foundAccessBlockCreate = true
		}
		if route.Method == "PUT" && route.Path == "/api/admin/access-block/rules/:id" && route.Auth == "admin" {
			foundAccessBlockUpdate = true
		}
		if route.Method == "DELETE" && route.Path == "/api/admin/access-block/rules/:id" && route.Auth == "admin" {
			foundAccessBlockDelete = true
		}
		if route.Method == "POST" && route.Path == "/api/admin/access-block/rules/batch" && route.Auth == "admin" {
			foundAccessBlockBatch = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/access-block/import-sources" && route.Auth == "admin" {
			foundAccessBlockImportList = true
		}
		if route.Method == "POST" && route.Path == "/api/admin/access-block/import-sources" && route.Auth == "admin" {
			foundAccessBlockImportCreate = true
		}
		if route.Method == "PUT" && route.Path == "/api/admin/access-block/import-sources/:id" && route.Auth == "admin" {
			foundAccessBlockImportUpdate = true
		}
		if route.Method == "DELETE" && route.Path == "/api/admin/access-block/import-sources/:id" && route.Auth == "admin" {
			foundAccessBlockImportDelete = true
		}
		if route.Method == "POST" && route.Path == "/api/admin/access-block/import-sources/:id/sync" && route.Auth == "admin" {
			foundAccessBlockImportSync = true
		}
		if route.Method == "POST" && route.Path == "/api/ai/jobs" && route.Auth == "user" {
			foundAIJobCreate = true
		}
		if route.Method == "GET" && route.Path == "/api/ai/jobs/active" && route.Auth == "user" {
			foundAIJobActive = true
		}
		if route.Method == "GET" && route.Path == "/api/ai/jobs/:jobId" && route.Auth == "user" {
			foundAIJobStatus = true
		}
		if route.Method == "GET" && route.Path == "/api/ai/jobs/:jobId/stream" && route.Auth == "user" {
			foundAIJobStream = true
		}
		if route.Method == "POST" && route.Path == "/api/ai/jobs/:jobId/cancel" && route.Auth == "user" {
			foundAIJobCancel = true
		}
		if route.Method == "GET" && route.Path == "/api/ai/publish-generation/settings" && route.Auth == "user" {
			foundAIPublishGenerationSettings = true
		}
		if route.Method == "POST" && route.Path == "/api/ai/publish-generation" && route.Auth == "user" {
			foundAIPublishGeneration = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/ai/jobs" && route.Auth == "admin" {
			foundAdminAIJobList = true
		}
		if route.Method == "POST" && route.Path == "/api/admin/ai/jobs" && route.Auth == "admin" {
			foundAdminAIJobCreate = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/ai/jobs/:jobId" && route.Auth == "admin" {
			foundAdminAIJobStatus = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/ai/jobs/:jobId/stream" && route.Auth == "admin" {
			foundAdminAIJobStream = true
		}
		if route.Method == "POST" && route.Path == "/api/admin/ai/jobs/:jobId/cancel" && route.Auth == "admin" {
			foundAdminAIJobCancel = true
		}
		if route.Method == "POST" && route.Path == "/api/admin/users/batch-generate" && route.Auth == "admin" {
			foundAdminUserBatchGenerate = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/ai-moderation-logs" && route.Auth == "admin" {
			foundAIModerationLogs = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/file-recycle-bin/:id/inspect" && route.Auth == "admin" {
			foundFileRecycleInspect = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/file-recycle-bin/:id/preview" && route.Auth == "admin" {
			foundFileRecyclePreview = true
		}
		if route.Method == "GET" && route.Path == "/api/admin/file-recycle-bin/:id/download" && route.Auth == "admin" {
			foundFileRecycleDownload = true
		}
	}
	if !foundImageWatermarkExtract {
		t.Fatal("admin hidden watermark extraction route is missing")
	}
	if !foundUserImageWatermarkExtract {
		t.Fatal("user hidden watermark extraction route is missing")
	}
	if !foundProtectedPackageCreate {
		t.Fatal("protected image package route is missing")
	}
	if !foundProtectedPackageEvents {
		t.Fatal("protected image package event stream route is missing")
	}
	if !foundPostImageArchiveStatus || !foundPostImageArchiveDownload {
		t.Fatal("post image archive routes are missing")
	}
	if !foundPostPurchases {
		t.Fatal("post purchase user list route is missing")
	}
	if !foundBalanceAudit || !foundPointsLedgerAudit || !foundMoonCoinLedgerAudit || !foundAvatarUpload || !foundBannerUpload || !foundAdminPointsAdjustment || !foundAdminSingleOnboardingReset {
		t.Fatal("new balance audit or profile upload route is missing")
	}
	if !foundRedisMaintenanceGet || !foundRedisMaintenancePut || !foundRedisMaintenanceRun {
		t.Fatal("redis maintenance routes are missing")
	}
	if !foundNetworkDiagnostics {
		t.Fatal("network diagnostics route is missing")
	}
	if !foundAIFormatMarkdown || !foundAdminAIGenerate || !foundAdminAISettingsGet || !foundAdminAISettingsPut || !foundAdminAILogs || !foundAdminAIModerationDebug {
		t.Fatal("AI agent routes are missing")
	}
	if !foundAccessBlockList || !foundAccessBlockCreate || !foundAccessBlockUpdate || !foundAccessBlockDelete || !foundAccessBlockBatch {
		t.Fatal("admin access block routes are missing")
	}
	if !foundAccessBlockImportList || !foundAccessBlockImportCreate || !foundAccessBlockImportUpdate || !foundAccessBlockImportDelete || !foundAccessBlockImportSync {
		t.Fatal("admin access block import source routes are missing")
	}
	if !foundAIJobCreate || !foundAIJobActive || !foundAIJobStatus || !foundAIJobStream || !foundAIJobCancel || !foundAIPublishGenerationSettings || !foundAIPublishGeneration || !foundAdminAIJobList || !foundAdminAIJobCreate || !foundAdminAIJobStatus || !foundAdminAIJobStream || !foundAdminAIJobCancel {
		t.Fatal("persistent AI job routes are missing")
	}
	if !foundAdminUserBatchGenerate || !foundAIModerationLogs {
		t.Fatal("AI moderation log or batch user generation route is missing")
	}
	if !foundFileRecycleInspect || !foundFileRecyclePreview || !foundFileRecycleDownload {
		t.Fatal("file recycle inspect, preview, or download route is missing")
	}
}
