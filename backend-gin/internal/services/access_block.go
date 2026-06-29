package services

import (
	"context"
	"errors"
	"net/netip"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

const (
	AccessBlockKindIP = "ip"
	AccessBlockKindUA = "ua"

	AccessBlockMatchIP         = "ip"
	AccessBlockMatchCIDR       = "cidr"
	AccessBlockMatchUAContains = "ua_contains"
	AccessBlockMatchUARegex    = "ua_regex"

	AccessBlockActionStatus   = "status"
	AccessBlockActionRedirect = "redirect"

	AccessBlockCacheScope = "access_block"

	accessBlockPatternMaxLength = 512
)

var (
	ErrAccessBlockValidation     = errors.New("access block validation failed")
	ErrAccessBlockSelfLock       = errors.New("access block rule matches current request")
	ErrAccessBlockImportDisabled = errors.New("access block import is disabled")
	ErrAccessBlockImportFetch    = errors.New("access block import fetch failed")
)

type AccessBlockService struct {
	db     *gorm.DB
	redis  *RedisStore
	logger *zap.Logger

	snapshot     atomic.Value
	disabled     atomic.Bool
	lastVer      atomic.Int64
	importConfig AccessBlockImportConfig
	importMu     sync.Mutex
	done         chan struct{}
	closeOnce    sync.Once
}

type AccessBlockMatchInput struct {
	IP        string
	UserAgent string
}

type AccessBlockMatch struct {
	Rule         AccessBlockRuleDTO
	MatchedValue string
}

type AccessBlockRuleDTO struct {
	ID             int64      `json:"id"`
	ImportSourceID int64      `json:"import_source_id"`
	Kind           string     `json:"kind"`
	MatchType      string     `json:"match_type"`
	Pattern        string     `json:"pattern"`
	Enabled        bool       `json:"enabled"`
	Priority       int        `json:"priority"`
	Action         string     `json:"action"`
	StatusCode     int        `json:"status_code"`
	RedirectURL    string     `json:"redirect_url"`
	Note           string     `json:"note"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

type AccessBlockRuleInput struct {
	Kind        string
	MatchType   string
	Pattern     string
	Enabled     *bool
	Priority    *int
	Action      string
	StatusCode  int
	RedirectURL string
	Note        string
	Force       bool
}

type AccessBlockBatchItem struct {
	Input AccessBlockRuleInput
	Line  int
}

type AccessBlockBatchResult struct {
	Line  int                 `json:"line"`
	Rule  *AccessBlockRuleDTO `json:"rule,omitempty"`
	Error string              `json:"error,omitempty"`
}

type AccessBlockImportConfig struct {
	Disabled              bool
	MinInterval           time.Duration
	HTTPTimeout           time.Duration
	MaxBytes              int64
	SchedulerInterval     time.Duration
	DefaultUpdateInterval time.Duration
}

type AccessBlockImportSourceDTO struct {
	ID                    int64      `json:"id"`
	URL                   string     `json:"url"`
	Enabled               bool       `json:"enabled"`
	Priority              int        `json:"priority"`
	Action                string     `json:"action"`
	StatusCode            int        `json:"status_code"`
	RedirectURL           string     `json:"redirect_url"`
	Note                  string     `json:"note"`
	UpdateIntervalSeconds int        `json:"update_interval_seconds"`
	LastSyncAt            *time.Time `json:"last_sync_at,omitempty"`
	NextSyncAt            *time.Time `json:"next_sync_at,omitempty"`
	LastStatus            string     `json:"last_status"`
	LastError             string     `json:"last_error"`
	LastCount             int        `json:"last_count"`
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             *time.Time `json:"updated_at,omitempty"`
}

type AccessBlockImportSourceInput struct {
	URL                   string
	Enabled               *bool
	Priority              *int
	Action                string
	StatusCode            int
	RedirectURL           string
	Note                  string
	UpdateIntervalSeconds int
}

type AccessBlockImportSyncOptions struct {
	Current AccessBlockMatchInput
	Force   bool
	Manual  bool
}

type AccessBlockImportSyncResult struct {
	Source AccessBlockImportSourceDTO `json:"source"`
	Count  int                        `json:"count"`
}

type accessBlockSnapshot struct {
	rules      []AccessBlockRuleDTO
	ipExact    map[netip.Addr][]compiledAccessBlockRule
	ipPrefixes map[int][]compiledAccessBlockRule
	uaContains []compiledAccessBlockRule
	uaRegex    []compiledAccessBlockRule
}

type compiledAccessBlockRule struct {
	dto       AccessBlockRuleDTO
	addr      netip.Addr
	prefix    netip.Prefix
	contains  string
	regex     *regexp.Regexp
	specific  int
	sortIndex int
}

func NewAccessBlockService(db *gorm.DB, redis *RedisStore, logger *zap.Logger, disabled bool) *AccessBlockService {
	if logger == nil {
		logger = zap.NewNop()
	}
	s := &AccessBlockService{
		db:           db,
		redis:        redis,
		logger:       logger,
		importConfig: defaultAccessBlockImportConfig(),
		done:         make(chan struct{}),
	}
	s.disabled.Store(disabled)
	s.snapshot.Store(emptyAccessBlockSnapshot())
	return s
}

func (s *AccessBlockService) ConfigureImports(cfg AccessBlockImportConfig) {
	if s == nil {
		return
	}
	s.importConfig = normalizeAccessBlockImportConfig(cfg)
}

func (s *AccessBlockService) Load(ctx context.Context) error {
	if s == nil || s.db == nil {
		return nil
	}
	var rows []domain.AccessBlockRule
	if err := s.db.WithContext(ctx).Order("priority ASC, id ASC").Find(&rows).Error; err != nil {
		return err
	}
	s.snapshot.Store(buildAccessBlockSnapshot(rows))
	return nil
}

func (s *AccessBlockService) StartVersionPolling(interval time.Duration) {
	if s == nil || s.redis == nil || interval <= 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
				version := s.redis.CacheVersion(ctx, AccessBlockCacheScope)
				cancel()
				if version <= 0 || version == s.lastVer.Load() {
					continue
				}
				if err := s.Load(context.Background()); err != nil {
					s.logger.Warn("reload access block rules failed", zap.Error(err))
					continue
				}
				s.lastVer.Store(version)
			}
		}
	}()
}

func (s *AccessBlockService) Close() {
	if s == nil {
		return
	}
	s.closeOnce.Do(func() {
		close(s.done)
	})
}

func (s *AccessBlockService) Disabled() bool {
	return s == nil || s.disabled.Load()
}

func (s *AccessBlockService) Match(input AccessBlockMatchInput) (AccessBlockMatch, bool) {
	if s == nil || s.Disabled() {
		return AccessBlockMatch{}, false
	}
	snapshot := s.currentSnapshot()
	candidates := make([]compiledAccessBlockRule, 0, 8)
	if ip, ok := parseAccessBlockAddr(input.IP); ok {
		if rules := snapshot.ipExact[ip]; len(rules) > 0 {
			candidates = append(candidates, rules...)
		}
		for bits := ip.BitLen(); bits >= 0; bits-- {
			rules := snapshot.ipPrefixes[bits]
			for _, rule := range rules {
				if rule.prefix.Contains(ip) {
					candidates = append(candidates, rule)
				}
			}
		}
	}
	ua := strings.ToLower(strings.TrimSpace(input.UserAgent))
	if ua != "" {
		for _, rule := range snapshot.uaContains {
			if strings.Contains(ua, rule.contains) {
				candidates = append(candidates, rule)
			}
		}
		for _, rule := range snapshot.uaRegex {
			if rule.regex != nil && rule.regex.MatchString(input.UserAgent) {
				candidates = append(candidates, rule)
			}
		}
	}
	if len(candidates) == 0 {
		return AccessBlockMatch{}, false
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return accessBlockRuleLess(candidates[i], candidates[j])
	})
	winner := candidates[0]
	return AccessBlockMatch{Rule: winner.dto, MatchedValue: matchedAccessBlockValue(winner, input)}, true
}

func (s *AccessBlockService) List(ctx context.Context) ([]AccessBlockRuleDTO, error) {
	if s == nil || s.db == nil {
		return []AccessBlockRuleDTO{}, nil
	}
	var rows []domain.AccessBlockRule
	if err := s.db.WithContext(ctx).Order("priority ASC, id ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]AccessBlockRuleDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, accessBlockRuleDTO(row))
	}
	return out, nil
}

func (s *AccessBlockService) Create(ctx context.Context, input AccessBlockRuleInput, current AccessBlockMatchInput) (AccessBlockRuleDTO, error) {
	if s == nil || s.db == nil {
		return AccessBlockRuleDTO{}, gorm.ErrInvalidDB
	}
	rule, err := normalizeAccessBlockRuleInput(input, nil)
	if err != nil {
		return AccessBlockRuleDTO{}, err
	}
	if !input.Force && rule.Enabled && accessBlockRuleMatchesInput(rule, current) {
		return AccessBlockRuleDTO{}, ErrAccessBlockSelfLock
	}
	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = &now
	if err := s.db.WithContext(ctx).Select(
		"Kind",
		"ImportSourceID",
		"MatchType",
		"Pattern",
		"Enabled",
		"Priority",
		"Action",
		"StatusCode",
		"RedirectURL",
		"Note",
		"CreatedAt",
		"UpdatedAt",
	).Create(&rule).Error; err != nil {
		return AccessBlockRuleDTO{}, err
	}
	s.afterMutation(ctx)
	return accessBlockRuleDTO(rule), nil
}

func (s *AccessBlockService) Update(ctx context.Context, id int64, input AccessBlockRuleInput, current AccessBlockMatchInput) (AccessBlockRuleDTO, error) {
	if s == nil || s.db == nil {
		return AccessBlockRuleDTO{}, gorm.ErrInvalidDB
	}
	var existing domain.AccessBlockRule
	if err := s.db.WithContext(ctx).Where("id = ?", id).Take(&existing).Error; err != nil {
		return AccessBlockRuleDTO{}, err
	}
	next, err := normalizeAccessBlockRuleInput(input, &existing)
	if err != nil {
		return AccessBlockRuleDTO{}, err
	}
	if !input.Force && next.Enabled && accessBlockRuleMatchesInput(next, current) {
		return AccessBlockRuleDTO{}, ErrAccessBlockSelfLock
	}
	now := time.Now()
	next.ID = existing.ID
	next.CreatedAt = existing.CreatedAt
	next.UpdatedAt = &now
	if err := s.db.WithContext(ctx).Model(&domain.AccessBlockRule{}).Where("id = ?", id).Updates(map[string]any{
		"kind":         next.Kind,
		"match_type":   next.MatchType,
		"pattern":      next.Pattern,
		"enabled":      next.Enabled,
		"priority":     next.Priority,
		"action":       next.Action,
		"status_code":  next.StatusCode,
		"redirect_url": next.RedirectURL,
		"note":         next.Note,
		"updated_at":   now,
	}).Error; err != nil {
		return AccessBlockRuleDTO{}, err
	}
	s.afterMutation(ctx)
	return accessBlockRuleDTO(next), nil
}

func (s *AccessBlockService) Delete(ctx context.Context, id int64) error {
	if s == nil || s.db == nil {
		return gorm.ErrInvalidDB
	}
	if err := s.db.WithContext(ctx).Delete(&domain.AccessBlockRule{}, id).Error; err != nil {
		return err
	}
	s.afterMutation(ctx)
	return nil
}

func (s *AccessBlockService) BatchCreate(ctx context.Context, items []AccessBlockBatchItem, current AccessBlockMatchInput, force bool) ([]AccessBlockBatchResult, error) {
	results := make([]AccessBlockBatchResult, 0, len(items))
	for _, item := range items {
		item.Input.Force = force || item.Input.Force
		rule, err := s.Create(ctx, item.Input, current)
		result := AccessBlockBatchResult{Line: item.Line}
		if err != nil {
			result.Error = AccessBlockErrorKey(err)
		} else {
			result.Rule = &rule
		}
		results = append(results, result)
	}
	return results, nil
}

func AccessBlockErrorKey(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrAccessBlockSelfLock):
		return "error.access_block_self_lock"
	case errors.Is(err, ErrAccessBlockImportDisabled):
		return "error.access_block_import_disabled"
	case errors.Is(err, ErrAccessBlockImportFetch):
		return "error.access_block_import_fetch_failed"
	case errors.Is(err, ErrAccessBlockValidation):
		return "error.access_block_invalid_rule"
	default:
		return "error.access_block_save_failed"
	}
}

func (s *AccessBlockService) afterMutation(ctx context.Context) {
	_ = s.Load(ctx)
	if s != nil && s.redis != nil {
		s.redis.BumpCacheVersion(ctx, AccessBlockCacheScope)
		s.lastVer.Store(s.redis.CacheVersion(ctx, AccessBlockCacheScope))
	}
}

func (s *AccessBlockService) currentSnapshot() *accessBlockSnapshot {
	if s == nil {
		return emptyAccessBlockSnapshot()
	}
	if loaded := s.snapshot.Load(); loaded != nil {
		if snapshot, ok := loaded.(*accessBlockSnapshot); ok && snapshot != nil {
			return snapshot
		}
	}
	return emptyAccessBlockSnapshot()
}
