package services

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
)

const (
	QueueIPLocation       = "ip-location-update"
	QueueContentAudit     = "content-audit"
	QueueAuditLog         = "audit-log"
	QueueGeneralTask      = "general-task"
	QueueAITask           = "ai-task"
	QueueBrowsingHistory  = "browsing-history"
	QueueVideoTranscoding = "video-transcoding"
	QueueBatchNoteCreate  = "batch-note-create"
	QueueImageProtection  = "image-protection"

	TaskBatchNoteCreate      = "batch-note-create:create-note"
	TaskVideoTranscoding     = "video-transcoding:transcode"
	TaskVideoTranscodingName = TaskVideoTranscoding
	TaskImageProtection      = "image-protection:package"
	TaskPostImageArchive     = "image-protection:post-archive"
	TaskAIPostAutoComment    = "ai:post-auto-comment"
	TaskAICommentReply       = "ai:comment-reply"
	TaskAIJobRun             = "ai:job-run"
	TaskAIModerateContent    = "ai:moderate-content"
)

var QueueNames = []string{
	QueueIPLocation,
	QueueContentAudit,
	QueueAuditLog,
	QueueGeneralTask,
	QueueAITask,
	QueueBrowsingHistory,
	QueueVideoTranscoding,
	QueueBatchNoteCreate,
	QueueImageProtection,
}

const (
	taskHeaderEnqueuedAt = "yuem-enqueued-at-ms"
	taskHeaderQueueKind  = "yuem-queue-kind"
	queueEventZSetKey    = "queue:events"
)

type queueMeta struct {
	Kind        string
	TaskTypes   []string
	Worker      bool
	Concurrency int
}

type QueueService struct {
	cfg                       config.Config
	db                        *gorm.DB
	settings                  *SettingsService
	ai                        *AIService
	cache                     *RedisStore
	client                    *asynq.Client
	inspector                 *asynq.Inspector
	server                    *asynq.Server
	aiServer                  *asynq.Server
	imageProtectionServer     *asynq.Server
	redis                     *redis.Client
	started                   bool
	startError                error
	lastAccessLogPrune        time.Time
	lastSecurityAuditLogPrune time.Time
	cleanupCancel             context.CancelFunc
	mu                        sync.Mutex
}

type queueEvent struct {
	TaskID     string         `json:"task_id"`
	Queue      string         `json:"queue"`
	Type       string         `json:"type"`
	Event      string         `json:"event"`
	State      string         `json:"state,omitempty"`
	At         int64          `json:"at"`
	LatencyMS  int64          `json:"latency_ms,omitempty"`
	DurationMS int64          `json:"duration_ms,omitempty"`
	Error      string         `json:"error,omitempty"`
	Detail     map[string]any `json:"detail,omitempty"`
}

type QueueBatchFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type QueueBatchNote struct {
	Title    string           `json:"title"`
	Content  string           `json:"content"`
	Files    []QueueBatchFile `json:"files"`
	CoverURL string           `json:"cover_url"`
}

type batchNoteTaskPayload struct {
	UserID     int64            `json:"user_id"`
	PostType   int              `json:"post_type"`
	Title      string           `json:"title"`
	Content    string           `json:"content"`
	IsDraft    bool             `json:"is_draft"`
	Files      []QueueBatchFile `json:"files"`
	Tags       []string         `json:"tags"`
	CoverURL   string           `json:"cover_url"`
	BatchID    string           `json:"batch_id"`
	NoteIndex  int              `json:"note_index"`
	TotalNotes int              `json:"total_notes"`
	EnqueuedAt int64            `json:"enqueued_at"`
}

type imageProtectionTaskPayload struct {
	JobID      string  `json:"job_id"`
	ImageIDs   []int64 `json:"image_ids,omitempty"`
	Kind       string  `json:"kind,omitempty"`
	EnqueuedAt int64   `json:"enqueued_at"`
}

type aiPostAutoCommentTaskPayload struct {
	PostID     int64 `json:"post_id"`
	AuthorID   int64 `json:"author_id,omitempty"`
	BotUserID  int64 `json:"bot_user_id"`
	EnqueuedAt int64 `json:"enqueued_at"`
}

type aiCommentReplyTaskPayload struct {
	TriggerCommentID int64 `json:"trigger_comment_id"`
	EnqueuedAt       int64 `json:"enqueued_at"`
}

type aiJobRunTaskPayload struct {
	JobID      string `json:"job_id"`
	EnqueuedAt int64  `json:"enqueued_at"`
}

type aiModerateContentTaskPayload struct {
	TargetType         string `json:"target_type"`
	TargetID           int64  `json:"target_id"`
	UserID             int64  `json:"user_id"`
	OriginalVisibility string `json:"original_visibility,omitempty"`
	EnqueuedAt         int64  `json:"enqueued_at"`
}

func NewQueueService(db *gorm.DB, cfg config.Config, settings *SettingsService, ai ...*AIService) *QueueService {
	s := &QueueService{cfg: cfg, db: db, settings: settings, cache: NewRedisStore(cfg.Redis)}
	if len(ai) > 0 {
		s.ai = ai[0]
	}
	if !cfg.Queue.Enabled {
		return s
	}
	opt := asynq.RedisClientOpt{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  250 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	}
	s.client = asynq.NewClient(opt)
	s.inspector = asynq.NewInspector(opt)
	s.redis = redis.NewClient(&redis.Options{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  250 * time.Millisecond,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
		MaxRetries:   0,
	})
	s.server = asynq.NewServer(opt, asynq.Config{
		Concurrency: maxInt(1,
			cfg.Queue.Concurrency.BatchNoteCreate+
				cfg.Queue.Concurrency.VideoTranscoding+
				cfg.Queue.Concurrency.GeneralTask+
				cfg.Queue.Concurrency.AuditLog,
		),
		Queues: map[string]int{
			QueueBatchNoteCreate:  1,
			QueueVideoTranscoding: 1,
			QueueGeneralTask:      maxInt(1, cfg.Queue.Concurrency.GeneralTask),
			QueueAuditLog:         maxInt(1, cfg.Queue.Concurrency.AuditLog),
		},
		ShutdownTimeout:     15 * time.Second,
		RetryDelayFunc:      retryDelayFunc(cfg.Queue.Retry.BackoffDelay),
		HealthCheckInterval: 30 * time.Second,
	})
	s.aiServer = asynq.NewServer(opt, asynq.Config{
		Concurrency:         maxInt(1, cfg.Queue.Concurrency.AITask),
		Queues:              map[string]int{QueueAITask: 1},
		ShutdownTimeout:     15 * time.Second,
		RetryDelayFunc:      retryDelayFunc(cfg.Queue.Retry.BackoffDelay),
		HealthCheckInterval: 30 * time.Second,
	})
	s.imageProtectionServer = asynq.NewServer(opt, asynq.Config{
		Concurrency:         maxInt(1, cfg.Queue.Concurrency.ImageProtection),
		Queues:              map[string]int{QueueImageProtection: 1},
		ShutdownTimeout:     15 * time.Second,
		RetryDelayFunc:      retryDelayFunc(cfg.Queue.Retry.BackoffDelay),
		HealthCheckInterval: 30 * time.Second,
	})
	return s
}

func (s *QueueService) Start() error {
	if s == nil || s.server == nil || s.db == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.started {
		return s.startError
	}
	mux := asynq.NewServeMux()
	mux.Use(s.queueEventMiddleware)
	mux.HandleFunc(TaskBatchNoteCreate, s.processBatchNoteCreate)
	mux.HandleFunc(TaskVideoTranscoding, s.processVideoTranscoding)
	mux.HandleFunc(TaskImageProtection, s.processImageProtectionPackage)
	mux.HandleFunc(TaskPostImageArchive, s.processImageProtectionPackage)
	mux.HandleFunc(TaskAIPostAutoComment, s.processAIPostAutoComment)
	mux.HandleFunc(TaskAICommentReply, s.processAICommentReply)
	mux.HandleFunc(TaskAIJobRun, s.processAIJobRun)
	mux.HandleFunc(TaskAIModerateContent, s.processAIModerateContent)
	mux.HandleFunc(TaskAccessLogBatch, s.processAccessLogBatch)
	mux.HandleFunc(TaskSecurityAuditLogBatch, s.processSecurityAuditLogBatch)
	s.startError = s.server.Start(mux)
	if s.startError == nil && s.imageProtectionServer != nil {
		s.startError = s.imageProtectionServer.Start(mux)
		if s.startError != nil {
			s.server.Shutdown()
		}
	}
	if s.startError == nil && s.aiServer != nil {
		s.startError = s.aiServer.Start(mux)
		if s.startError != nil {
			s.server.Shutdown()
			if s.imageProtectionServer != nil {
				s.imageProtectionServer.Shutdown()
			}
		}
	}
	s.started = s.startError == nil
	if s.started && s.cleanupCancel == nil {
		cleanupContext, cancel := context.WithCancel(context.Background())
		s.cleanupCancel = cancel
		go s.runImageProtectionCleanup(cleanupContext)
	}
	return s.startError
}

func (s *QueueService) Close() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	if s.cleanupCancel != nil {
		s.cleanupCancel()
		s.cleanupCancel = nil
	}
	s.mu.Unlock()
	if s.server != nil {
		s.server.Shutdown()
	}
	if s.imageProtectionServer != nil {
		s.imageProtectionServer.Shutdown()
	}
	if s.aiServer != nil {
		s.aiServer.Shutdown()
	}
	var err error
	if s.inspector != nil {
		err = errors.Join(err, s.inspector.Close())
	}
	if s.client != nil {
		err = errors.Join(err, s.client.Close())
	}
	if s.redis != nil {
		err = errors.Join(err, s.redis.Close())
	}
	if s.cache != nil && s.cache.client != nil {
		err = errors.Join(err, s.cache.client.Close())
	}
	return err
}

func (s *QueueService) runImageProtectionCleanup(ctx context.Context) {
	_ = s.cleanupExpiredImageProtectionPackages(ctx)
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.cleanupExpiredImageProtectionPackages(ctx)
		}
	}
}

func (s *QueueService) Enabled() bool {
	return s != nil && s.cfg.Queue.Enabled && s.client != nil && s.inspector != nil
}

func (s *QueueService) Available() bool {
	if !s.Enabled() {
		return false
	}
	return s.client.Ping() == nil
}

func (s *QueueService) RuntimeStatus() map[string]any {
	status := map[string]any{
		"enabled":          false,
		"available":        false,
		"queueEnabled":     false,
		"redisConfigured":  false,
		"redisAvailable":   false,
		"redis":            map[string]any{"configured": false, "available": false},
		"workerStarted":    false,
		"redisAddr":        "",
		"redisDb":          0,
		"configuredQueues": []string{},
		"checkedAt":        timeMillis(time.Now()),
	}
	if s == nil {
		status["message"] = "queue service is not initialized"
		return status
	}
	status["queueEnabled"] = s.cfg.Queue.Enabled
	status["redisConfigured"] = strings.TrimSpace(s.cfg.Redis.Addr) != ""
	status["redisAddr"] = s.cfg.Redis.Addr
	status["redisDb"] = s.cfg.Redis.DB
	status["workerStarted"] = s.started
	status["configuredQueues"] = QueueNames
	status["redis"] = (&RedisStore{client: s.redis}).Status(context.Background(), s.cfg.Redis)
	if s.startError != nil {
		status["startError"] = s.startError.Error()
	}
	if !s.Enabled() {
		status["message"] = "queue service is disabled or Redis client is not initialized"
		return status
	}
	available := s.client.Ping() == nil
	status["enabled"] = available
	status["available"] = available
	status["redisAvailable"] = available
	if !available {
		status["message"] = "Redis ping failed for Asynq"
	}
	return status
}

func (s *QueueService) Names() []string {
	names := append([]string{}, QueueNames...)
	if s == nil || !s.Enabled() {
		return names
	}
	discovered, err := s.inspector.Queues()
	if err != nil {
		return names
	}
	seen := make(map[string]bool, len(names)+len(discovered))
	for _, name := range names {
		seen[name] = true
	}
	sort.Strings(discovered)
	for _, name := range discovered {
		if !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}
	return names
}
