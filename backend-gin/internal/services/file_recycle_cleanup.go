package services

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

type FileRecycleCleanupResult struct {
	Expired    FileRecycleSummary `json:"expired"`
	OrphanDASH FileRecycleSummary `json:"orphan_dash"`
}

type FileRecycleCleanupService struct {
	recycle *FileRecycleService
	logger  *zap.Logger
	done    chan struct{}
	close   sync.Once
	mu      sync.Mutex
	lastRun time.Time
	running atomic.Bool
}

func NewFileRecycleCleanupService(recycle *FileRecycleService, logger *zap.Logger) *FileRecycleCleanupService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &FileRecycleCleanupService{recycle: recycle, logger: logger, done: make(chan struct{})}
}

func (s *FileRecycleCleanupService) Start() {
	if s == nil || s.recycle == nil || s.recycle.db == nil || !s.recycle.Enabled() {
		return
	}
	go s.loop()
}

func (s *FileRecycleCleanupService) Close() {
	if s == nil {
		return
	}
	s.close.Do(func() {
		close(s.done)
	})
}

func (s *FileRecycleCleanupService) loop() {
	s.runIfDue()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.runIfDue()
		}
	}
}

func (s *FileRecycleCleanupService) runIfDue() {
	if s == nil || s.recycle == nil || !s.recycle.Enabled() || !s.due(time.Now()) || !s.running.CompareAndSwap(false, true) {
		return
	}
	defer s.running.Store(false)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	result, err := s.recycle.RunCleanup(ctx, 500, 100)
	s.markRun(time.Now())
	if err != nil {
		s.logger.Warn("file recycle cleanup failed", zap.Error(err))
		return
	}
	if result.Expired.Purged > 0 || result.Expired.Failed > 0 || result.OrphanDASH.Recycled > 0 || result.OrphanDASH.Failed > 0 {
		s.logger.Info("file recycle cleanup completed",
			zap.Int("purged", result.Expired.Purged),
			zap.Int("purge_failed", result.Expired.Failed),
			zap.Int("orphan_dash_recycled", result.OrphanDASH.Recycled),
			zap.Int("orphan_dash_failed", result.OrphanDASH.Failed),
		)
	}
}

func (s *FileRecycleCleanupService) due(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.lastRun.IsZero() {
		return true
	}
	return now.Sub(s.lastRun) >= s.recycle.CleanupInterval()
}

func (s *FileRecycleCleanupService) markRun(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastRun = now
}

func (s *FileRecycleService) RunCleanup(ctx context.Context, expiredLimit, orphanLimit int) (FileRecycleCleanupResult, error) {
	var result FileRecycleCleanupResult
	expired, err := s.PurgeExpired(ctx, expiredLimit)
	if err != nil {
		return result, err
	}
	result.Expired = expired
	orphanDASH, err := s.RecycleOrphanedDASH(ctx, 2*time.Hour, orphanLimit)
	if err != nil {
		return result, err
	}
	result.OrphanDASH = orphanDASH
	return result, nil
}

func clampFileRecycleRetentionDays(value int) int {
	if value <= 0 {
		return DefaultFileRecycleRetentionDays
	}
	if value > 3650 {
		return 3650
	}
	return value
}

func clampFileRecycleCleanupIntervalHours(value int) int {
	if value <= 0 {
		return DefaultFileRecycleCleanupIntervalHours
	}
	if value > 24*30 {
		return 24 * 30
	}
	return value
}
