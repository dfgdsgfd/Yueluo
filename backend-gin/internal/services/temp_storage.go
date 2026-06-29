package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"yuem-go/backend-gin/internal/config"
)

type TempStorageService struct {
	cfg    config.UploadTempConfig
	root   string
	cancel context.CancelFunc
	mu     sync.Mutex
}

func NewTempStorageService(cfg config.UploadTempConfig) *TempStorageService {
	return &TempStorageService{cfg: cfg, root: serviceAbsPath(cfg.RootDir)}
}

func (s *TempStorageService) Start() error {
	if s == nil || strings.TrimSpace(s.root) == "" {
		return errors.New("temporary storage root is empty")
	}
	if err := os.MkdirAll(s.root, 0700); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	_ = s.Cleanup(time.Now())
	interval := s.cfg.CleanupInterval
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	go s.run(ctx, interval)
	return nil
}

func (s *TempStorageService) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.mu.Unlock()
}

func (s *TempStorageService) Root() string {
	if s == nil {
		return ""
	}
	return s.root
}

func (s *TempStorageService) NewWorkDir(kind string) (string, error) {
	if s == nil || s.root == "" {
		return "", errors.New("temporary storage is not configured")
	}
	kind = safeTempKind(kind)
	parent := filepath.Join(s.root, kind)
	if err := os.MkdirAll(parent, 0700); err != nil {
		return "", err
	}
	var suffix [12]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return "", err
	}
	dir := filepath.Join(parent, hex.EncodeToString(suffix[:]))
	if !pathWithinRoot(s.root, dir) {
		return "", errors.New("temporary path escaped configured root")
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

func (s *TempStorageService) Cleanup(now time.Time) error {
	if s == nil || s.root == "" {
		return nil
	}
	retention := s.cfg.Retention
	if retention <= 0 {
		retention = 24 * time.Hour
	}
	kinds, err := os.ReadDir(s.root)
	if errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(s.root, 0700)
	}
	if err != nil {
		return err
	}
	var joined error
	for _, kind := range kinds {
		if kind.Type()&os.ModeSymlink != 0 || !kind.IsDir() {
			continue
		}
		kindPath := filepath.Join(s.root, kind.Name())
		kindRetention := retention
		if kind.Name() == "protected-packages" && s.cfg.ProtectedPackageRetention > 0 {
			kindRetention = s.cfg.ProtectedPackageRetention
		}
		cutoff := now.Add(-kindRetention)
		entries, readErr := os.ReadDir(kindPath)
		if readErr != nil {
			joined = errors.Join(joined, readErr)
			continue
		}
		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
				continue
			}
			path := filepath.Join(kindPath, entry.Name())
			if !pathWithinRoot(s.root, path) {
				continue
			}
			info, statErr := entry.Info()
			if statErr != nil {
				joined = errors.Join(joined, statErr)
				continue
			}
			if info.ModTime().Before(cutoff) {
				joined = errors.Join(joined, os.RemoveAll(path))
			}
		}
	}
	return joined
}

func (s *TempStorageService) run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			_ = s.Cleanup(now)
		}
	}
}

func safeTempKind(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var out strings.Builder
	for _, r := range value {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			out.WriteRune(r)
		}
	}
	if out.Len() == 0 {
		return "jobs"
	}
	return out.String()
}

func pathWithinRoot(root, target string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
