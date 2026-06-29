package handlers

import (
	"os"
	"path/filepath"
	"testing"

	"yuem-go/backend-gin/internal/config"
)

func TestDeleteLocalPostFilesReturnsAuditSummary(t *testing.T) {
	root := t.TempDir()
	imageDir := filepath.Join(root, "images")
	if err := os.MkdirAll(imageDir, 0o755); err != nil {
		t.Fatalf("mkdir image dir: %v", err)
	}
	filePath := filepath.Join(imageDir, "sample.jpg")
	if err := os.WriteFile(filePath, []byte("sample"), 0o644); err != nil {
		t.Fatalf("write image: %v", err)
	}
	missingPath := filepath.Join(imageDir, "missing.jpg")

	handler := NativeHandlers{Config: config.Config{
		Server: config.ServerConfig{Env: "test"},
		Upload: config.UploadConfig{
			RootDir: root,
			Image:   config.UploadImageConfig{LocalUploadDir: imageDir},
		},
	}}

	summary := handler.deleteLocalPostFiles([]string{filePath, missingPath})
	if summary.Deleted != 1 || summary.Missing != 1 || summary.Failed != 0 || summary.attempted() != 2 {
		t.Fatalf("summary = %+v, want one deleted and one missing", summary)
	}
	if summary.outcome() != "success" {
		t.Fatalf("outcome = %q, want success", summary.outcome())
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Fatalf("file should be removed, stat err = %v", err)
	}
	metadata := summary.auditMetadata()
	if metadata["deleted_count"] != 1 || metadata["missing_count"] != 1 {
		t.Fatalf("metadata = %#v, want deleted/missing counts", metadata)
	}
}

func TestDeleteLocalPostFilesSkipsUnsafeDirectory(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	handler := NativeHandlers{Config: config.Config{
		Server: config.ServerConfig{Env: "test"},
		Upload: config.UploadConfig{RootDir: root},
	}}

	summary := handler.deleteLocalPostFiles([]string{outside})
	if summary.Skipped != 1 || summary.Deleted != 0 || summary.outcome() != "skipped" {
		t.Fatalf("summary = %+v, want skipped unsafe directory", summary)
	}
	if stat, err := os.Stat(outside); err != nil || !stat.IsDir() {
		t.Fatalf("unsafe directory should remain, stat = %+v err = %v", stat, err)
	}
}
