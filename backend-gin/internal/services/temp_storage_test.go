package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"yuem-go/backend-gin/internal/config"
)

func TestTempStorageCreatesAndCleansWorkDirectories(t *testing.T) {
	root := filepath.Join(t.TempDir(), "uploads", "tmp")
	service := NewTempStorageService(config.UploadTempConfig{
		RootDir: root, Retention: 24 * time.Hour, ProtectedPackageRetention: 2 * time.Hour,
	})
	if err := service.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(service.Close)

	regular, err := service.NewWorkDir("uploads")
	if err != nil {
		t.Fatal(err)
	}
	protected, err := service.NewWorkDir("protected-packages")
	if err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-3 * time.Hour)
	if err := os.Chtimes(regular, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(protected, old, old); err != nil {
		t.Fatal(err)
	}
	if err := service.Cleanup(time.Now()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(regular); err != nil {
		t.Fatalf("regular work directory should be retained: %v", err)
	}
	if _, err := os.Stat(protected); !os.IsNotExist(err) {
		t.Fatalf("protected package should expire after two hours, stat error = %v", err)
	}
}

func TestTempStorageDoesNotFollowSymlinkOutsideRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "uploads", "tmp")
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0700); err != nil {
		t.Fatal(err)
	}
	service := NewTempStorageService(config.UploadTempConfig{RootDir: root, Retention: time.Hour})
	if err := service.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(service.Close)
	linkParent := filepath.Join(root, "jobs")
	if err := os.MkdirAll(linkParent, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(linkParent, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := service.Cleanup(time.Now().Add(48 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outside); err != nil {
		t.Fatalf("cleanup followed a symlink outside the root: %v", err)
	}
}
