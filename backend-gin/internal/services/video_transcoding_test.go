package services

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"yuem-go/backend-gin/internal/config"
)

func TestVideoTranscodingLocalPathAndDashOutput(t *testing.T) {
	temp := t.TempDir()
	videoDir := filepath.Join(temp, "videos")
	sourcePath := filepath.Join(videoDir, "1778703092050_sample.mp4")
	cfg := config.Config{
		Upload: config.UploadConfig{
			LocalBase: "https://xse.example.com",
			Video:     config.UploadVideoConfig{LocalUploadDir: videoDir},
		},
		Video: config.VideoTranscodingConfig{
			OutputFormat: "{date}/{userId}/{postId}/{timestamp}",
		},
	}

	if got := canonicalVideoURLFromPath(sourcePath, cfg); got != "/api/file/videos/1778703092050_sample.mp4" {
		t.Fatalf("canonicalVideoURLFromPath() = %q", got)
	}
	gotPath, ok := localVideoPathFromURL("https://xse.example.com/api/file/videos/1778703092050_sample.mp4", cfg)
	if !ok {
		t.Fatalf("localVideoPathFromURL() ok = false")
	}
	if gotPath != sourcePath {
		t.Fatalf("localVideoPathFromURL() = %q, want %q", gotPath, sourcePath)
	}

	outputDir, dashURL, err := videoDashOutput(sourcePath, videoTranscodingTaskPayload{
		UserID: 1497,
		PostID: 42,
	}, cfg)
	if err != nil {
		t.Fatalf("videoDashOutput() error = %v", err)
	}
	if !strings.Contains(filepath.ToSlash(outputDir), "/videos/dash/") {
		t.Fatalf("outputDir should be inside dash root, got %q", outputDir)
	}
	if !strings.HasSuffix(dashURL, "/1497/42/1778703092050/manifest.mpd") {
		t.Fatalf("dashURL = %q, want user/post/upload timestamp path", dashURL)
	}
}

func TestVideoTranscodingTaskIDIsStableForVideoRows(t *testing.T) {
	payload := videoTranscodingTaskPayload{VideoID: 99, VideoURL: "/api/file/videos/a.mp4"}
	if got := videoTranscodingTaskID(payload); got != "video-transcoding:video:99" {
		t.Fatalf("videoTranscodingTaskID() = %q", got)
	}
	payload.VideoID = 0
	first := videoTranscodingTaskID(payload)
	second := videoTranscodingTaskID(payload)
	if first == "" || first != second || !strings.HasPrefix(first, "video-transcoding:source:") {
		t.Fatalf("source task id is not stable: first=%q second=%q", first, second)
	}
}

func TestTranscodeVideoUsesExistingManifestWithoutFFmpeg(t *testing.T) {
	temp := t.TempDir()
	videoDir := filepath.Join(temp, "videos")
	sourcePath := filepath.Join(videoDir, "1778703092050_sample.mp4")
	if err := writeTestFile(sourcePath, "video-bytes"); err != nil {
		t.Fatalf("write source: %v", err)
	}
	mtime := time.Date(2026, 5, 13, 10, 0, 0, 0, time.UTC)
	if err := touchTestFile(sourcePath, mtime); err != nil {
		t.Fatalf("touch source: %v", err)
	}
	cfg := config.Config{
		Upload: config.UploadConfig{
			Video: config.UploadVideoConfig{LocalUploadDir: videoDir},
		},
		Video: config.VideoTranscodingConfig{
			FFmpegPath:   filepath.Join(temp, "missing-ffmpeg"),
			OutputFormat: "{date}/{userId}/{timestamp}",
		},
	}
	manifestDir := filepath.Join(videoDir, "dash", "2026-05-13", "1497", "1778703092050")
	if err := writeTestFile(filepath.Join(manifestDir, "manifest.mpd"), "<MPD></MPD>"); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	service := &QueueService{cfg: cfg}
	result, err := service.transcodeVideo(context.Background(), videoTranscodingTaskPayload{
		UserID:     1497,
		VideoURL:   "/api/file/videos/1778703092050_sample.mp4",
		SourcePath: sourcePath,
		CoverURL:   "/api/file/thumbnails/existing.jpg",
	})
	if err != nil {
		t.Fatalf("transcodeVideo() error = %v", err)
	}
	if result.DashURL != "/api/file/videos/dash/2026-05-13/1497/1778703092050/manifest.mpd" {
		t.Fatalf("DashURL = %q", result.DashURL)
	}
	if result.CoverURL != "/api/file/thumbnails/existing.jpg" {
		t.Fatalf("CoverURL = %q", result.CoverURL)
	}
}

func TestTranscodeVideoSkipsRemoteSource(t *testing.T) {
	service := &QueueService{}
	result, err := service.transcodeVideo(context.Background(), videoTranscodingTaskPayload{
		VideoURL: "https://cdn.example.com/video.mp4",
	})
	if err != nil {
		t.Fatalf("transcodeVideo() error = %v", err)
	}
	if !result.Skipped || result.Reason == "" {
		t.Fatalf("remote source should be skipped, got %+v", result)
	}
}

func writeTestFile(path string, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func touchTestFile(path string, ts time.Time) error {
	return os.Chtimes(path, ts, ts)
}
