package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/services"
)

func TestAppDownloadConfigUsesSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	settings := services.NewSettingsService(nil, nil)
	ctx := context.Background()
	if !settings.Set(ctx, "app_download_android_name", "Yuem Android Beta") ||
		!settings.Set(ctx, "app_download_android_version_name", "1.2.3") ||
		!settings.Set(ctx, "app_download_android_version_code", 123) ||
		!settings.Set(ctx, "app_download_android_download_url", "/api/file/attachments/yuem.apk") ||
		!settings.Set(ctx, "app_download_android_size_bytes", 7340032) ||
		!settings.Set(ctx, "app_download_android_fast_name", "Yuem Fast") ||
		!settings.Set(ctx, "app_download_android_fast_download_url", "/api/file/attachments/yuem-fast.apk") ||
		!settings.Set(ctx, "app_download_android_fast_size_label", "≤ 1 MB") ||
		!settings.Set(ctx, "app_download_ios_enabled", false) {
		t.Fatal("failed to seed app download settings")
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/app/download-config", nil)
	NativeHandlers{Settings: settings}.AppDownloadConfig(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var responseBody struct {
		Data struct {
			Android struct {
				Name        string `json:"name"`
				VersionName string `json:"version_name"`
				VersionCode int    `json:"version_code"`
				DownloadURL string `json:"download_url"`
				SizeBytes   int    `json:"size_bytes"`
			} `json:"android"`
			AndroidFast struct {
				Name        string `json:"name"`
				DownloadURL string `json:"download_url"`
				SizeLabel   string `json:"size_label"`
				PackageName string `json:"package_name"`
			} `json:"android_fast"`
			IOS struct {
				Enabled bool `json:"enabled"`
			} `json:"ios"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if responseBody.Data.Android.Name != "Yuem Android Beta" ||
		responseBody.Data.Android.VersionName != "1.2.3" ||
		responseBody.Data.Android.VersionCode != 123 ||
		responseBody.Data.Android.DownloadURL != "/api/file/attachments/yuem.apk" ||
		responseBody.Data.Android.SizeBytes != 7340032 {
		t.Fatalf("unexpected android config: %+v", responseBody.Data.Android)
	}
	if responseBody.Data.AndroidFast.Name != "Yuem Fast" ||
		responseBody.Data.AndroidFast.DownloadURL != "/api/file/attachments/yuem-fast.apk" ||
		responseBody.Data.AndroidFast.SizeLabel != "≤ 1 MB" ||
		responseBody.Data.AndroidFast.PackageName != "com.yuelk.xsewebfast" {
		t.Fatalf("unexpected android fast config: %+v", responseBody.Data.AndroidFast)
	}
	if responseBody.Data.IOS.Enabled {
		t.Fatalf("ios enabled = true, want false")
	}
}

func TestAppVersionSizeMBBytes(t *testing.T) {
	got, ok := appVersionSizeMBBytes("0.95")
	if !ok {
		t.Fatal("appVersionSizeMBBytes should parse decimal MB")
	}
	const want = int64(996147)
	if got != want {
		t.Fatalf("appVersionSizeMBBytes(0.95) = %d, want %d", got, want)
	}
	if mb := appVersionSizeMB(1048576); mb != 1 {
		t.Fatalf("appVersionSizeMB(1048576) = %v, want 1", mb)
	}
}

func TestReportAppEventPersistsUsageLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file:app-usage-report?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.AppVersion{}, &domain.AppUsageLog{}); err != nil {
		t.Fatalf("migrate sqlite: %v", err)
	}
	version := domain.AppVersion{
		AppName:     "Yuem",
		VersionCode: 6621,
		VersionName: "2.22.0",
		Platform:    "android",
		DownloadURL: "/api/file/attachments/yuem.apk",
		IsActive:    true,
	}
	if err := db.Create(&version).Error; err != nil {
		t.Fatalf("seed app version: %v", err)
	}

	handler := NativeHandlers{DB: db}
	postAppEvent(t, handler, `{"device_id":"device-1","event_type":"app_open","platform":"android","version_code":6621}`)
	postAppEvent(t, handler, `{"device_id":"device-1","event_type":"update_complete","platform":"android","version_code":"6621"}`)
	postAppEvent(t, handler, `{"device_id":"device-1","event_type":"usage_duration","platform":"android","version_code":6621,"duration":75}`)

	var logs []domain.AppUsageLog
	if err := db.Order("id asc").Find(&logs).Error; err != nil {
		t.Fatalf("query usage logs: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("usage log count = %d, want 3", len(logs))
	}
	if logs[0].EventType != "app_open" || logs[0].DeviceID != "device-1" || logs[0].Platform != "android" {
		t.Fatalf("unexpected app_open log: %+v", logs[0])
	}
	if logs[1].EventType != "update_complete" || logs[1].VersionCode == nil || *logs[1].VersionCode != 6621 ||
		logs[1].VersionID == nil || *logs[1].VersionID != version.ID {
		t.Fatalf("unexpected update_complete log: %+v", logs[1])
	}
	if logs[2].EventType != "usage_duration" || logs[2].Duration == nil || *logs[2].Duration != 75 {
		t.Fatalf("unexpected usage_duration log: %+v", logs[2])
	}
}

func postAppEvent(t *testing.T, handler NativeHandlers, body string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/app/report-event", strings.NewReader(body))
	context.Request.Header.Set("Content-Type", "application/json")
	handler.ReportAppEvent(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
}
