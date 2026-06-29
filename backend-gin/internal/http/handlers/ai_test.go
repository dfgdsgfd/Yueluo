package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/services"
)

func TestShouldEmitAIStreamEventToActorFiltersUpstreamDiagnostics(t *testing.T) {
	event := services.AIStreamEvent{Type: "upstream_event"}
	if shouldEmitAIStreamEventToActor(event, services.AIActor{Type: "user"}) {
		t.Fatal("user stream should not emit upstream diagnostics")
	}
	if !shouldEmitAIStreamEventToActor(event, services.AIActor{Type: "admin"}) {
		t.Fatal("admin stream should emit upstream diagnostics")
	}
	if !shouldEmitAIStreamEventToActor(services.AIStreamEvent{Type: "progress"}, services.AIActor{Type: "user"}) {
		t.Fatal("user stream should still emit normal progress events")
	}
}

func TestAIReachableImagesSignsLocalFilesWithAbsoluteBaseURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NativeHandlers{Config: config.Config{
		Auth: config.AuthConfig{JWTSecret: "jwt-secret"},
		Upload: config.UploadConfig{
			FileSigning: config.UploadFileSigningConfig{Secret: "file-secret", TTL: time.Hour},
			LocalBase:   "https://xse.example.com",
		},
	}}

	images := handler.aiReachableImages(nil, []services.AIImageInput{
		{URL: "/api/file/images/sample.webp", Mime: " image/webp ", Alt: " sample.webp "},
		{URL: "https://cdn.example.com/public.jpg?x=1", Mime: "image/jpeg"},
	})
	if len(images) != 2 {
		t.Fatalf("images = %d, want 2", len(images))
	}
	if got := images[0].URL; !strings.HasPrefix(got, "https://xse.example.com/api/file/images/sample.webp?") ||
		!strings.Contains(got, "pvimg_exp=") || !strings.Contains(got, "sign=") {
		t.Fatalf("local AI image URL was not signed as absolute URL: %q", got)
	}
	if images[0].Mime != "image/webp" || images[0].Alt != "sample.webp" {
		t.Fatalf("image metadata not trimmed: %#v", images[0])
	}
	if got := images[1].URL; got != "https://cdn.example.com/public.jpg?x=1" {
		t.Fatalf("external AI image URL changed: %q", got)
	}
}

func TestAIReachableImagesFallsBackToForwardedHostWhenConfiguredBaseIsLocal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/ai/publish-generation", nil)
	context.Request.Host = "localhost:3001"
	context.Request.Header.Set("X-Forwarded-Proto", "https")
	context.Request.Header.Set("X-Forwarded-Host", "api.example.com")
	handler := NativeHandlers{Config: config.Config{
		Auth: config.AuthConfig{JWTSecret: "jwt-secret"},
		Upload: config.UploadConfig{
			FileSigning: config.UploadFileSigningConfig{Secret: "file-secret", TTL: time.Hour},
			LocalBase:   "http://localhost:3001",
		},
	}}

	images := handler.aiReachableImages(context, []services.AIImageInput{{URL: "/api/file/images/sample.webp"}})
	if len(images) != 1 {
		t.Fatalf("images = %d, want 1", len(images))
	}
	if got := images[0].URL; !strings.HasPrefix(got, "https://api.example.com/api/file/images/sample.webp?") ||
		!strings.Contains(got, "pvimg_exp=") || !strings.Contains(got, "sign=") {
		t.Fatalf("forwarded AI image URL was not signed as absolute URL: %q", got)
	}

	images = handler.aiReachableImages(context, []services.AIImageInput{{URL: "http://localhost:3001/api/file/images/local.webp"}})
	if got := images[0].URL; !strings.HasPrefix(got, "https://api.example.com/api/file/images/local.webp?") ||
		!strings.Contains(got, "pvimg_exp=") || !strings.Contains(got, "sign=") {
		t.Fatalf("local absolute AI image URL was not rewritten to forwarded host: %q", got)
	}
}

func TestAIJobResponseIncludesMetadataQueueJob(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{
		"queueJob": map[string]any{
			"id":                   "job-queue-ticket",
			"jobId":                "job-123",
			"queue":                "ai-concurrency",
			"state":                "queued",
			"queuePosition":        3,
			"queueCount":           5,
			"estimatedWaitSeconds": 42,
		},
	})
	job := domain.AIJob{
		JobID:    "job-123",
		Status:   services.AIJobStatusRunning,
		Stage:    "queued",
		Metadata: datatypes.JSON(raw),
	}

	payload := (NativeHandlers{}).aiJobResponse(context.Background(), job)
	queueJob, ok := payload["queueJob"].(gin.H)
	if !ok {
		t.Fatalf("queueJob missing from payload: %#v", payload["queueJob"])
	}
	if queueJob["queuePosition"] != 3 || queueJob["queueCount"] != 5 || queueJob["estimatedWaitSeconds"] != 42 {
		t.Fatalf("queueJob = %#v, want metadata queue values", queueJob)
	}
}

func TestMergeAIQueueJobPayloadPreservesActiveJob(t *testing.T) {
	existing := map[string]any{
		"id":                   "job-queue-ticket",
		"jobId":                "job-123",
		"queuePosition":        3,
		"queueCount":           5,
		"estimatedWaitSeconds": 42,
		"activeJob": map[string]any{
			"jobId":           "active-job",
			"actorId":         7,
			"actorDisplayId":  "xise-7",
			"generatedTokens": 99,
			"tokensPerSecond": 4.2,
		},
	}
	next := map[string]any{
		"id":                   "task-id",
		"jobId":                "job-123",
		"queue":                "ai-concurrency",
		"state":                "queued",
		"queuePosition":        2,
		"queueCount":           6,
		"estimatedWaitSeconds": 30,
	}

	merged := mergeAIQueueJobPayload(existing, next)
	activeJob, ok := merged["activeJob"].(map[string]any)
	if !ok {
		t.Fatalf("merged queueJob missing activeJob: %#v", merged)
	}
	if activeJob["jobId"] != "active-job" || activeJob["generatedTokens"] != 99 {
		t.Fatalf("merged activeJob = %#v, want preserved active job snapshot", activeJob)
	}
	if merged["queuePosition"] != 2 || merged["queueCount"] != 6 || merged["queue"] != "ai-concurrency" {
		t.Fatalf("merged queueJob = %#v, want latest queue fields", merged)
	}
}

func TestAdminAIJobsListsActiveJobsWithStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&domain.AIJob{}); err != nil {
		t.Fatalf("migrate ai jobs: %v", err)
	}
	now := time.Now()
	display := "xise-100"
	jobs := []domain.AIJob{
		{
			JobID:          "job-queued",
			RequestHash:    "hash-queued",
			TaskType:       services.AITaskFormatMarkdown,
			TemplateKey:    "markdown_format",
			ActorType:      "user",
			ActorID:        int64Ptr(100),
			ActorDisplayID: &display,
			Status:         services.AIJobStatusQueued,
			Stage:          "queued",
			UpdatedAt:      &now,
			CreatedAt:      now.Add(-time.Minute),
		},
		{
			JobID:       "job-running",
			RequestHash: "hash-running",
			TaskType:    services.AITaskPostPolish,
			TemplateKey: "post_polish",
			ActorType:   "user",
			ActorID:     int64Ptr(101),
			Status:      services.AIJobStatusRunning,
			Stage:       "running",
			UpdatedAt:   &now,
			CreatedAt:   now.Add(-2 * time.Minute),
		},
		{
			JobID:       "job-completed",
			RequestHash: "hash-completed",
			TaskType:    services.AITaskFormatMarkdown,
			TemplateKey: "markdown_format",
			ActorType:   "user",
			ActorID:     int64Ptr(102),
			Status:      services.AIJobStatusCompleted,
			Stage:       "completed",
			UpdatedAt:   &now,
			CreatedAt:   now.Add(-3 * time.Minute),
		},
	}
	if err := db.Create(&jobs).Error; err != nil {
		t.Fatalf("create jobs: %v", err)
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/admin/ai/jobs", nil)

	NativeHandlers{DB: db}.AdminAIJobs(context)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body=%s", recorder.Code, recorder.Body.String())
	}
	var body struct {
		Data struct {
			Items []struct {
				JobID          string `json:"jobId"`
				ActorDisplayID string `json:"actorDisplayId"`
				Status         string `json:"status"`
			} `json:"items"`
			Stats map[string]int `json:"stats"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data.Items) != 2 {
		t.Fatalf("items = %d, want 2: %#v", len(body.Data.Items), body.Data.Items)
	}
	if body.Data.Stats["queued"] != 1 || body.Data.Stats["running"] != 1 || body.Data.Stats["active"] != 2 {
		t.Fatalf("stats = %#v, want queued/running/active = 1/1/2", body.Data.Stats)
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
