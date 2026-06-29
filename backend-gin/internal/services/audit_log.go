package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/datatypes"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

const (
	TaskAccessLogBatch        = "audit-log:write-access-batch"
	TaskSecurityAuditLogBatch = "audit-log:write-security-batch"

	auditLogPruneInterval = time.Hour
)

type AccessLogEvent struct {
	UserID          *int64         `json:"user_id,omitempty"`
	UserDisplayID   string         `json:"user_display_id,omitempty"`
	PrincipalType   string         `json:"principal_type,omitempty"`
	IP              string         `json:"ip,omitempty"`
	UserAgent       string         `json:"user_agent,omitempty"`
	BrowserLanguage string         `json:"browser_language,omitempty"`
	Method          string         `json:"method,omitempty"`
	Path            string         `json:"path,omitempty"`
	Status          int            `json:"status,omitempty"`
	LatencyMS       int64          `json:"latency_ms,omitempty"`
	Behavior        string         `json:"behavior,omitempty"`
	TargetType      string         `json:"target_type,omitempty"`
	TargetID        *int64         `json:"target_id,omitempty"`
	RequestID       string         `json:"request_id,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

type SecurityAuditLogEvent struct {
	Category        string         `json:"category,omitempty"`
	Action          string         `json:"action,omitempty"`
	Outcome         string         `json:"outcome,omitempty"`
	ActorID         *int64         `json:"actor_id,omitempty"`
	ActorType       string         `json:"actor_type,omitempty"`
	ActorDisplayID  string         `json:"actor_display_id,omitempty"`
	IP              string         `json:"ip,omitempty"`
	UserAgent       string         `json:"user_agent,omitempty"`
	BrowserLanguage string         `json:"browser_language,omitempty"`
	Method          string         `json:"method,omitempty"`
	Path            string         `json:"path,omitempty"`
	Status          int            `json:"status,omitempty"`
	ReasonCode      string         `json:"reason_code,omitempty"`
	RequestID       string         `json:"request_id,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	CreatedAt       time.Time      `json:"created_at"`
}

type AuditLogService struct {
	queue            *QueueService
	cfg              config.Config
	logger           *zap.Logger
	accessCh         chan AccessLogEvent
	securityCh       chan SecurityAuditLogEvent
	done             chan struct{}
	behaviorSet      map[string]struct{}
	accessDropped    atomic.Int64
	securityDropped  atomic.Int64
	accessEnqueued   atomic.Int64
	securityEnqueued atomic.Int64
	enqueueFailures  atomic.Int64
}

type accessLogBatchPayload struct {
	Events     []AccessLogEvent `json:"events"`
	EnqueuedAt int64            `json:"enqueued_at"`
}

type securityAuditLogBatchPayload struct {
	Events     []SecurityAuditLogEvent `json:"events"`
	EnqueuedAt int64                   `json:"enqueued_at"`
}

func NewAuditLogService(queue *QueueService, cfg config.Config, logger *zap.Logger) *AuditLogService {
	if logger == nil {
		logger = zap.NewNop()
	}
	cfg.AccessLog = normalizeAccessLogConfig(cfg.AccessLog)
	cfg.SecurityAuditLog = normalizeSecurityAuditLogConfig(cfg.SecurityAuditLog)
	s := &AuditLogService{
		queue:       queue,
		cfg:         cfg,
		logger:      logger,
		done:        make(chan struct{}),
		behaviorSet: accessBehaviorSet(cfg.AccessLog.Behaviors),
	}
	if cfg.AccessLog.Enabled && queue != nil && queue.Enabled() {
		s.accessCh = make(chan AccessLogEvent, cfg.AccessLog.BufferSize)
		go s.runAccessFlusher()
	}
	if cfg.SecurityAuditLog.Enabled && queue != nil && queue.Enabled() {
		s.securityCh = make(chan SecurityAuditLogEvent, cfg.AccessLog.BufferSize)
		go s.runSecurityFlusher()
	}
	return s
}

func normalizeAccessLogConfig(cfg config.AccessLogConfig) config.AccessLogConfig {
	cfg.Scope = strings.ToLower(strings.TrimSpace(cfg.Scope))
	if cfg.Scope == "" {
		cfg.Scope = "key"
	}
	if cfg.Retention <= 0 {
		cfg.Retention = 8760 * time.Hour
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 4096
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 100
	}
	if cfg.BatchSize > 1000 {
		cfg.BatchSize = 1000
	}
	if cfg.FlushInterval <= 0 {
		cfg.FlushInterval = time.Second
	}
	return cfg
}

func normalizeSecurityAuditLogConfig(cfg config.SecurityAuditLogConfig) config.SecurityAuditLogConfig {
	if cfg.Retention <= 0 {
		cfg.Retention = 8760 * time.Hour
	}
	return cfg
}

func accessBehaviorSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			out[value] = struct{}{}
		}
	}
	return out
}

func (s *AuditLogService) Close() {
	if s == nil {
		return
	}
	close(s.done)
}

func (s *AuditLogService) RecordAccess(event AccessLogEvent) {
	if s == nil || !s.acceptAccessBehavior(event.Behavior) {
		return
	}
	if s.accessCh == nil {
		if s.cfg.AccessLog.Enabled {
			s.accessDropped.Add(1)
		}
		return
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	event = normalizeAccessLogEvent(event)
	select {
	case s.accessCh <- event:
	default:
		s.accessDropped.Add(1)
	}
}

func (s *AuditLogService) RecordSecurity(event SecurityAuditLogEvent) {
	if s == nil || s.securityCh == nil {
		if s != nil && s.cfg.SecurityAuditLog.Enabled {
			s.securityDropped.Add(1)
		}
		return
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	event = normalizeSecurityAuditLogEvent(event)
	select {
	case s.securityCh <- event:
	default:
		s.securityDropped.Add(1)
	}
}

func (s *AuditLogService) acceptAccessBehavior(behavior string) bool {
	if s == nil || !s.cfg.AccessLog.Enabled {
		return false
	}
	if s.cfg.AccessLog.Scope == "all" {
		return true
	}
	_, ok := s.behaviorSet[strings.ToLower(strings.TrimSpace(behavior))]
	return ok
}

func (s *AuditLogService) RuntimeStatus() map[string]any {
	status := map[string]any{
		"access_enabled":           s != nil && s.cfg.AccessLog.Enabled,
		"security_enabled":         s != nil && s.cfg.SecurityAuditLog.Enabled,
		"queue_enabled":            false,
		"available":                false,
		"access_buffered":          0,
		"security_buffered":        0,
		"access_buffer_capacity":   0,
		"security_buffer_capacity": 0,
		"access_dropped":           int64(0),
		"security_dropped":         int64(0),
		"access_enqueued":          int64(0),
		"security_enqueued":        int64(0),
		"enqueue_failures":         int64(0),
	}
	if s == nil {
		return status
	}
	status["queue_enabled"] = s.queue != nil && s.queue.Enabled()
	status["available"] = s.queue != nil && s.queue.Available()
	if s.accessCh != nil {
		status["access_buffered"] = len(s.accessCh)
		status["access_buffer_capacity"] = cap(s.accessCh)
	}
	if s.securityCh != nil {
		status["security_buffered"] = len(s.securityCh)
		status["security_buffer_capacity"] = cap(s.securityCh)
	}
	status["access_dropped"] = s.accessDropped.Load()
	status["security_dropped"] = s.securityDropped.Load()
	status["access_enqueued"] = s.accessEnqueued.Load()
	status["security_enqueued"] = s.securityEnqueued.Load()
	status["enqueue_failures"] = s.enqueueFailures.Load()
	return status
}

func (s *AuditLogService) runAccessFlusher() {
	ticker := time.NewTicker(s.cfg.AccessLog.FlushInterval)
	defer ticker.Stop()
	batch := make([]AccessLogEvent, 0, s.cfg.AccessLog.BatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		events := append([]AccessLogEvent(nil), batch...)
		batch = batch[:0]
		if err := s.queue.EnqueueAccessLogBatch(context.Background(), events); err != nil {
			s.enqueueFailures.Add(1)
			s.logger.Warn("enqueue access log batch failed", zap.Error(err), zap.Int("count", len(events)))
			return
		}
		s.accessEnqueued.Add(int64(len(events)))
	}
	for {
		select {
		case <-s.done:
			flush()
			return
		case event := <-s.accessCh:
			batch = append(batch, event)
			if len(batch) >= s.cfg.AccessLog.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (s *AuditLogService) runSecurityFlusher() {
	ticker := time.NewTicker(s.cfg.AccessLog.FlushInterval)
	defer ticker.Stop()
	batch := make([]SecurityAuditLogEvent, 0, s.cfg.AccessLog.BatchSize)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		events := append([]SecurityAuditLogEvent(nil), batch...)
		batch = batch[:0]
		if err := s.queue.EnqueueSecurityAuditLogBatch(context.Background(), events); err != nil {
			s.enqueueFailures.Add(1)
			s.logger.Warn("enqueue security audit log batch failed", zap.Error(err), zap.Int("count", len(events)))
			return
		}
		s.securityEnqueued.Add(int64(len(events)))
	}
	for {
		select {
		case <-s.done:
			flush()
			return
		case event := <-s.securityCh:
			batch = append(batch, event)
			if len(batch) >= s.cfg.AccessLog.BatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (s *QueueService) EnqueueAccessLogBatch(ctx context.Context, events []AccessLogEvent) error {
	if s == nil || !s.Enabled() {
		return errors.New("queue service disabled")
	}
	if len(events) == 0 {
		return nil
	}
	payload := accessLogBatchPayload{Events: events, EnqueuedAt: time.Now().UnixMilli()}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	options := []asynq.Option{
		asynq.Queue(QueueAuditLog),
		asynq.MaxRetry(maxInt(0, s.cfg.Queue.Retry.Attempts)),
		asynq.Timeout(30 * time.Second),
	}
	if retention := s.completedRetention(QueueAuditLog, 72*time.Hour); retention > 0 {
		options = append(options, asynq.Retention(retention))
	}
	info, err := s.client.EnqueueContext(ctx, newQueueTask(TaskAccessLogBatch, data, QueueAuditLog, payload.EnqueuedAt), options...)
	if err != nil {
		return err
	}
	s.recordQueueEvent(ctx, queueEvent{
		TaskID: info.ID,
		Queue:  info.Queue,
		Type:   TaskAccessLogBatch,
		Event:  "enqueued",
		State:  info.State.String(),
		At:     payload.EnqueuedAt,
		Detail: map[string]any{"count": len(events)},
	})
	return nil
}

func (s *QueueService) EnqueueSecurityAuditLogBatch(ctx context.Context, events []SecurityAuditLogEvent) error {
	if s == nil || !s.Enabled() {
		return errors.New("queue service disabled")
	}
	if len(events) == 0 {
		return nil
	}
	payload := securityAuditLogBatchPayload{Events: events, EnqueuedAt: time.Now().UnixMilli()}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	options := []asynq.Option{
		asynq.Queue(QueueAuditLog),
		asynq.MaxRetry(maxInt(0, s.cfg.Queue.Retry.Attempts)),
		asynq.Timeout(30 * time.Second),
	}
	if retention := s.completedRetention(QueueAuditLog, 72*time.Hour); retention > 0 {
		options = append(options, asynq.Retention(retention))
	}
	info, err := s.client.EnqueueContext(ctx, newQueueTask(TaskSecurityAuditLogBatch, data, QueueAuditLog, payload.EnqueuedAt), options...)
	if err != nil {
		return err
	}
	s.recordQueueEvent(ctx, queueEvent{
		TaskID: info.ID,
		Queue:  info.Queue,
		Type:   TaskSecurityAuditLogBatch,
		Event:  "enqueued",
		State:  info.State.String(),
		At:     payload.EnqueuedAt,
		Detail: map[string]any{"count": len(events)},
	})
	return nil
}

func (s *QueueService) processAccessLogBatch(ctx context.Context, task *asynq.Task) error {
	var payload accessLogBatchPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}
	rows := accessLogRows(payload.Events)
	if len(rows) > 0 {
		if err := s.db.WithContext(ctx).CreateInBatches(rows, maxInt(1, s.cfg.AccessLog.BatchSize)).Error; err != nil {
			return err
		}
	}
	s.pruneAccessLogs(ctx, time.Now())
	result, _ := json.Marshal(map[string]any{"inserted": len(rows)})
	_, _ = task.ResultWriter().Write(result)
	return nil
}

func (s *QueueService) processSecurityAuditLogBatch(ctx context.Context, task *asynq.Task) error {
	var payload securityAuditLogBatchPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		return err
	}
	rows := securityAuditLogRows(payload.Events)
	if len(rows) > 0 {
		if err := s.db.WithContext(ctx).CreateInBatches(rows, maxInt(1, s.cfg.AccessLog.BatchSize)).Error; err != nil {
			return err
		}
	}
	s.pruneSecurityAuditLogs(ctx, time.Now())
	result, _ := json.Marshal(map[string]any{"inserted": len(rows)})
	_, _ = task.ResultWriter().Write(result)
	return nil
}

func accessLogRows(events []AccessLogEvent) []domain.AccessLog {
	rows := make([]domain.AccessLog, 0, len(events))
	for _, event := range events {
		event = normalizeAccessLogEvent(event)
		rows = append(rows, domain.AccessLog{
			UserID:          event.UserID,
			UserDisplayID:   stringPtrOrNil(event.UserDisplayID),
			PrincipalType:   event.PrincipalType,
			IP:              event.IP,
			UserAgent:       event.UserAgent,
			BrowserLanguage: event.BrowserLanguage,
			Method:          event.Method,
			Path:            event.Path,
			Status:          event.Status,
			LatencyMS:       event.LatencyMS,
			Behavior:        event.Behavior,
			TargetType:      stringPtrOrNil(event.TargetType),
			TargetID:        event.TargetID,
			RequestID:       event.RequestID,
			Metadata:        metadataJSON(event.Metadata),
			CreatedAt:       event.CreatedAt,
		})
	}
	return rows
}

func securityAuditLogRows(events []SecurityAuditLogEvent) []domain.SecurityAuditLog {
	rows := make([]domain.SecurityAuditLog, 0, len(events))
	for _, event := range events {
		event = normalizeSecurityAuditLogEvent(event)
		rows = append(rows, domain.SecurityAuditLog{
			Category:        event.Category,
			Action:          event.Action,
			Outcome:         event.Outcome,
			ActorID:         event.ActorID,
			ActorType:       event.ActorType,
			ActorDisplayID:  stringPtrOrNil(event.ActorDisplayID),
			IP:              event.IP,
			UserAgent:       event.UserAgent,
			BrowserLanguage: event.BrowserLanguage,
			Method:          event.Method,
			Path:            event.Path,
			Status:          event.Status,
			ReasonCode:      event.ReasonCode,
			RequestID:       event.RequestID,
			Metadata:        metadataJSON(event.Metadata),
			CreatedAt:       event.CreatedAt,
		})
	}
	return rows
}

func normalizeAccessLogEvent(event AccessLogEvent) AccessLogEvent {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	event.PrincipalType = truncateString(firstNonEmptyAuditString(strings.ToLower(strings.TrimSpace(event.PrincipalType)), "guest"), 32)
	event.IP = truncateString(event.IP, 64)
	event.UserAgent = truncateString(event.UserAgent, 2048)
	event.BrowserLanguage = truncateString(event.BrowserLanguage, 128)
	event.Method = truncateString(strings.ToUpper(strings.TrimSpace(event.Method)), 16)
	event.Path = truncateString(event.Path, 255)
	event.Behavior = truncateString(strings.ToLower(strings.TrimSpace(event.Behavior)), 64)
	event.TargetType = truncateString(strings.ToLower(strings.TrimSpace(event.TargetType)), 32)
	event.UserDisplayID = truncateString(event.UserDisplayID, 128)
	event.RequestID = truncateString(event.RequestID, 128)
	return event
}

func normalizeSecurityAuditLogEvent(event SecurityAuditLogEvent) SecurityAuditLogEvent {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	event.Category = truncateString(strings.ToLower(strings.TrimSpace(event.Category)), 64)
	event.Action = truncateString(strings.ToLower(strings.TrimSpace(event.Action)), 64)
	event.Outcome = truncateString(firstNonEmptyAuditString(strings.ToLower(strings.TrimSpace(event.Outcome)), "unknown"), 32)
	event.ActorType = truncateString(firstNonEmptyAuditString(strings.ToLower(strings.TrimSpace(event.ActorType)), "unknown"), 32)
	event.ActorDisplayID = truncateString(event.ActorDisplayID, 128)
	event.IP = truncateString(event.IP, 64)
	event.UserAgent = truncateString(event.UserAgent, 2048)
	event.BrowserLanguage = truncateString(event.BrowserLanguage, 128)
	event.Method = truncateString(strings.ToUpper(strings.TrimSpace(event.Method)), 16)
	event.Path = truncateString(event.Path, 255)
	event.ReasonCode = truncateString(strings.ToLower(strings.TrimSpace(event.ReasonCode)), 128)
	event.RequestID = truncateString(event.RequestID, 128)
	return event
}

func metadataJSON(value map[string]any) datatypes.JSON {
	if len(value) == 0 {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return datatypes.JSON(data)
}

func stringPtrOrNil(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func firstNonEmptyAuditString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func truncateString(value string, limit int) string {
	value = strings.TrimSpace(value)
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}

func (s *QueueService) pruneAccessLogs(ctx context.Context, now time.Time) {
	if s == nil || s.db == nil || s.cfg.AccessLog.Retention <= 0 || !s.shouldPruneAccessLogs(now) {
		return
	}
	cutoff := now.Add(-s.cfg.AccessLog.Retention)
	_ = s.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&domain.AccessLog{}).Error
}

func (s *QueueService) pruneSecurityAuditLogs(ctx context.Context, now time.Time) {
	if s == nil || s.db == nil || s.cfg.SecurityAuditLog.Retention <= 0 || !s.shouldPruneSecurityAuditLogs(now) {
		return
	}
	cutoff := now.Add(-s.cfg.SecurityAuditLog.Retention)
	_ = s.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&domain.SecurityAuditLog{}).Error
}

func (s *QueueService) shouldPruneAccessLogs(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.lastAccessLogPrune.IsZero() && now.Sub(s.lastAccessLogPrune) < auditLogPruneInterval {
		return false
	}
	s.lastAccessLogPrune = now
	return true
}

func (s *QueueService) shouldPruneSecurityAuditLogs(now time.Time) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.lastSecurityAuditLogPrune.IsZero() && now.Sub(s.lastSecurityAuditLogPrune) < auditLogPruneInterval {
		return false
	}
	s.lastSecurityAuditLogPrune = now
	return true
}
