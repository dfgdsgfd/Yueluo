package services

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

const (
	accessBlockImportStatusPending = "pending"
	accessBlockImportStatusSuccess = "success"
	accessBlockImportStatusFailed  = "failed"
)

func defaultAccessBlockImportConfig() AccessBlockImportConfig {
	return AccessBlockImportConfig{
		MinInterval:           5 * time.Minute,
		HTTPTimeout:           10 * time.Second,
		MaxBytes:              2 * 1024 * 1024,
		SchedulerInterval:     time.Minute,
		DefaultUpdateInterval: time.Hour,
	}
}

func normalizeAccessBlockImportConfig(cfg AccessBlockImportConfig) AccessBlockImportConfig {
	defaults := defaultAccessBlockImportConfig()
	if cfg.MinInterval <= 0 {
		cfg.MinInterval = defaults.MinInterval
	}
	if cfg.HTTPTimeout <= 0 {
		cfg.HTTPTimeout = defaults.HTTPTimeout
	}
	if cfg.MaxBytes <= 0 {
		cfg.MaxBytes = defaults.MaxBytes
	}
	if cfg.SchedulerInterval <= 0 {
		cfg.SchedulerInterval = defaults.SchedulerInterval
	}
	if cfg.DefaultUpdateInterval <= 0 {
		cfg.DefaultUpdateInterval = defaults.DefaultUpdateInterval
	}
	if cfg.DefaultUpdateInterval < cfg.MinInterval {
		cfg.DefaultUpdateInterval = cfg.MinInterval
	}
	return cfg
}

func (s *AccessBlockService) ListImportSources(ctx context.Context) ([]AccessBlockImportSourceDTO, error) {
	if s == nil || s.db == nil {
		return []AccessBlockImportSourceDTO{}, nil
	}
	var rows []domain.AccessBlockImportSource
	if err := s.db.WithContext(ctx).Order("id DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]AccessBlockImportSourceDTO, 0, len(rows))
	for _, row := range rows {
		out = append(out, accessBlockImportSourceDTO(row))
	}
	return out, nil
}

func (s *AccessBlockService) CreateImportSource(ctx context.Context, input AccessBlockImportSourceInput) (AccessBlockImportSourceDTO, error) {
	if s == nil || s.db == nil {
		return AccessBlockImportSourceDTO{}, gorm.ErrInvalidDB
	}
	source, err := s.normalizeImportSourceInput(input, nil)
	if err != nil {
		return AccessBlockImportSourceDTO{}, err
	}
	now := time.Now()
	source.CreatedAt = now
	source.UpdatedAt = &now
	source.LastStatus = accessBlockImportStatusPending
	source.NextSyncAt = accessBlockNextSyncAt(now, source.Enabled, source.UpdateIntervalSeconds)
	if err := s.db.WithContext(ctx).Create(&source).Error; err != nil {
		return AccessBlockImportSourceDTO{}, err
	}
	return accessBlockImportSourceDTO(source), nil
}

func (s *AccessBlockService) UpdateImportSource(ctx context.Context, id int64, input AccessBlockImportSourceInput) (AccessBlockImportSourceDTO, error) {
	if s == nil || s.db == nil {
		return AccessBlockImportSourceDTO{}, gorm.ErrInvalidDB
	}
	var existing domain.AccessBlockImportSource
	if err := s.db.WithContext(ctx).Where("id = ?", id).Take(&existing).Error; err != nil {
		return AccessBlockImportSourceDTO{}, err
	}
	next, err := s.normalizeImportSourceInput(input, &existing)
	if err != nil {
		return AccessBlockImportSourceDTO{}, err
	}
	now := time.Now()
	next.ID = existing.ID
	next.CreatedAt = existing.CreatedAt
	next.UpdatedAt = &now
	next.LastSyncAt = existing.LastSyncAt
	next.LastStatus = existing.LastStatus
	next.LastError = existing.LastError
	next.LastCount = existing.LastCount
	next.NextSyncAt = accessBlockNextSyncAt(now, next.Enabled, next.UpdateIntervalSeconds)
	if err := s.db.WithContext(ctx).Model(&domain.AccessBlockImportSource{}).Where("id = ?", id).Updates(map[string]any{
		"url":                     next.URL,
		"enabled":                 next.Enabled,
		"priority":                next.Priority,
		"action":                  next.Action,
		"status_code":             next.StatusCode,
		"redirect_url":            next.RedirectURL,
		"note":                    next.Note,
		"update_interval_seconds": next.UpdateIntervalSeconds,
		"next_sync_at":            next.NextSyncAt,
		"updated_at":              now,
	}).Error; err != nil {
		return AccessBlockImportSourceDTO{}, err
	}
	if err := s.db.WithContext(ctx).Model(&domain.AccessBlockRule{}).Where("import_source_id = ?", id).Updates(map[string]any{
		"enabled":      next.Enabled,
		"priority":     next.Priority,
		"action":       next.Action,
		"status_code":  next.StatusCode,
		"redirect_url": next.RedirectURL,
		"note":         next.Note,
		"updated_at":   now,
	}).Error; err != nil {
		return AccessBlockImportSourceDTO{}, err
	}
	s.afterMutation(ctx)
	return accessBlockImportSourceDTO(next), nil
}

func (s *AccessBlockService) DeleteImportSource(ctx context.Context, id int64) error {
	if s == nil || s.db == nil {
		return gorm.ErrInvalidDB
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("import_source_id = ?", id).Delete(&domain.AccessBlockRule{}).Error; err != nil {
			return err
		}
		return tx.Delete(&domain.AccessBlockImportSource{}, id).Error
	}); err != nil {
		return err
	}
	s.afterMutation(ctx)
	return nil
}

func (s *AccessBlockService) SyncImportSource(ctx context.Context, id int64, options AccessBlockImportSyncOptions) (AccessBlockImportSyncResult, error) {
	if s == nil || s.db == nil {
		return AccessBlockImportSyncResult{}, gorm.ErrInvalidDB
	}
	if s.importConfig.Disabled {
		return AccessBlockImportSyncResult{}, ErrAccessBlockImportDisabled
	}
	s.importMu.Lock()
	defer s.importMu.Unlock()

	var source domain.AccessBlockImportSource
	if err := s.db.WithContext(ctx).Where("id = ?", id).Take(&source).Error; err != nil {
		return AccessBlockImportSyncResult{}, err
	}
	rules, err := s.fetchImportRules(ctx, source)
	if err != nil {
		updated := s.markImportSourceSynced(ctx, source, accessBlockImportStatusFailed, err.Error(), source.LastCount)
		return AccessBlockImportSyncResult{Source: accessBlockImportSourceDTO(updated)}, err
	}
	if options.Manual && !options.Force {
		for _, rule := range rules {
			if rule.Enabled && accessBlockRuleMatchesInput(rule, options.Current) {
				err := ErrAccessBlockSelfLock
				updated := s.markImportSourceSynced(ctx, source, accessBlockImportStatusFailed, err.Error(), source.LastCount)
				return AccessBlockImportSyncResult{Source: accessBlockImportSourceDTO(updated)}, err
			}
		}
	}
	if err := s.replaceImportedRules(ctx, source, rules); err != nil {
		updated := s.markImportSourceSynced(ctx, source, accessBlockImportStatusFailed, err.Error(), source.LastCount)
		return AccessBlockImportSyncResult{Source: accessBlockImportSourceDTO(updated)}, err
	}
	updated := s.markImportSourceSynced(ctx, source, accessBlockImportStatusSuccess, "", len(rules))
	s.afterMutation(ctx)
	return AccessBlockImportSyncResult{Source: accessBlockImportSourceDTO(updated), Count: len(rules)}, nil
}

func (s *AccessBlockService) StartImportScheduler() {
	if s == nil || s.db == nil || s.importConfig.Disabled {
		return
	}
	interval := s.importConfig.SchedulerInterval
	if interval <= 0 {
		interval = time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				s.syncDueImportSources()
			}
		}
	}()
}

func (s *AccessBlockService) syncDueImportSources() {
	if s == nil || s.db == nil {
		return
	}
	now := time.Now()
	var sources []domain.AccessBlockImportSource
	if err := s.db.WithContext(context.Background()).
		Where("enabled = ? AND next_sync_at IS NOT NULL AND next_sync_at <= ?", true, now).
		Order("next_sync_at ASC, id ASC").
		Limit(10).
		Find(&sources).Error; err != nil {
		s.logger.Warn("load due access block import sources failed", zap.Error(err))
		return
	}
	for _, source := range sources {
		ctx, cancel := context.WithTimeout(context.Background(), s.importConfig.HTTPTimeout+5*time.Second)
		_, err := s.SyncImportSource(ctx, source.ID, AccessBlockImportSyncOptions{})
		cancel()
		if err != nil {
			s.logger.Warn("sync access block import source failed", zap.Int64("source_id", source.ID), zap.Error(err))
		}
	}
}

func (s *AccessBlockService) normalizeImportSourceInput(input AccessBlockImportSourceInput, existing *domain.AccessBlockImportSource) (domain.AccessBlockImportSource, error) {
	source := domain.AccessBlockImportSource{}
	if existing != nil {
		source = *existing
	}
	source.URL = strings.TrimSpace(firstAccessBlockNonEmptyString(input.URL, source.URL))
	if !validAccessBlockImportURL(source.URL) {
		return domain.AccessBlockImportSource{}, fmt.Errorf("%w: invalid import url", ErrAccessBlockValidation)
	}
	if input.Enabled != nil {
		source.Enabled = *input.Enabled
	} else if existing == nil {
		source.Enabled = true
	}
	if input.Priority != nil {
		source.Priority = *input.Priority
	} else if existing == nil {
		source.Priority = 1000
	}
	source.Action = normalizeAccessBlockAction(firstAccessBlockNonEmptyString(input.Action, source.Action))
	if source.Action == "" {
		source.Action = AccessBlockActionStatus
	}
	if input.StatusCode != 0 {
		source.StatusCode = input.StatusCode
	} else if source.StatusCode == 0 {
		source.StatusCode = 444
	}
	source.RedirectURL = strings.TrimSpace(firstAccessBlockNonEmptyString(input.RedirectURL, source.RedirectURL))
	source.Note = strings.TrimSpace(firstAccessBlockNonEmptyString(input.Note, source.Note))
	source.UpdateIntervalSeconds = s.normalizeImportIntervalSeconds(input.UpdateIntervalSeconds, source.UpdateIntervalSeconds)
	probe := domain.AccessBlockRule{
		Kind:        AccessBlockKindIP,
		MatchType:   AccessBlockMatchIP,
		Pattern:     "127.0.0.1",
		Enabled:     true,
		Priority:    source.Priority,
		Action:      source.Action,
		StatusCode:  source.StatusCode,
		RedirectURL: source.RedirectURL,
		Note:        source.Note,
	}
	if err := validateAccessBlockRule(probe); err != nil {
		return domain.AccessBlockImportSource{}, err
	}
	return source, nil
}

func (s *AccessBlockService) normalizeImportIntervalSeconds(next int, current int) int {
	if next <= 0 {
		next = current
	}
	if next <= 0 {
		next = int(s.importConfig.DefaultUpdateInterval.Seconds())
	}
	minimum := int(s.importConfig.MinInterval.Seconds())
	if minimum <= 0 {
		minimum = int((5 * time.Minute).Seconds())
	}
	if next < minimum {
		next = minimum
	}
	return next
}

func (s *AccessBlockService) fetchImportRules(ctx context.Context, source domain.AccessBlockImportSource) ([]domain.AccessBlockRule, error) {
	text, err := s.downloadImportText(ctx, source.URL)
	if err != nil {
		return nil, err
	}
	return parseAccessBlockImportRules(text, source)
}

func (s *AccessBlockService) downloadImportText(ctx context.Context, rawURL string) (string, error) {
	if !validAccessBlockImportURL(rawURL) {
		return "", fmt.Errorf("%w: invalid import url", ErrAccessBlockValidation)
	}
	timeout := s.importConfig.HTTPTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrAccessBlockImportFetch, err)
	}
	req.Header.Set("Accept", "text/plain,*/*;q=0.8")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrAccessBlockImportFetch, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("%w: status %d", ErrAccessBlockImportFetch, resp.StatusCode)
	}
	limit := s.importConfig.MaxBytes
	if limit <= 0 {
		limit = 2 * 1024 * 1024
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrAccessBlockImportFetch, err)
	}
	if int64(len(data)) > limit {
		return "", fmt.Errorf("%w: response too large", ErrAccessBlockImportFetch)
	}
	return string(data), nil
}

func parseAccessBlockImportRules(text string, source domain.AccessBlockImportSource) ([]domain.AccessBlockRule, error) {
	scanner := bufio.NewScanner(strings.NewReader(text))
	scanner.Buffer(make([]byte, 1024), accessBlockPatternMaxLength*4)
	seen := map[string]struct{}{}
	rules := []domain.AccessBlockRule{}
	line := 0
	now := time.Now()
	for scanner.Scan() {
		line++
		pattern := strings.TrimSpace(scanner.Text())
		if pattern == "" || strings.HasPrefix(pattern, "#") {
			continue
		}
		matchType := AccessBlockMatchIP
		if strings.Contains(pattern, "/") {
			matchType = AccessBlockMatchCIDR
		}
		rule := domain.AccessBlockRule{
			ImportSourceID: source.ID,
			Kind:           AccessBlockKindIP,
			MatchType:      matchType,
			Pattern:        pattern,
			Enabled:        source.Enabled,
			Priority:       source.Priority,
			Action:         source.Action,
			StatusCode:     source.StatusCode,
			RedirectURL:    source.RedirectURL,
			Note:           source.Note,
			CreatedAt:      now,
			UpdatedAt:      &now,
		}
		normalized, err := normalizeAccessBlockRuleInput(AccessBlockRuleInput{
			Kind:        rule.Kind,
			MatchType:   rule.MatchType,
			Pattern:     rule.Pattern,
			Enabled:     &rule.Enabled,
			Priority:    &rule.Priority,
			Action:      rule.Action,
			StatusCode:  rule.StatusCode,
			RedirectURL: rule.RedirectURL,
			Note:        rule.Note,
			Force:       true,
		}, &rule)
		if err != nil {
			return nil, fmt.Errorf("%w: line %d", err, line)
		}
		key := normalized.MatchType + "\x00" + normalized.Pattern
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		rules = append(rules, normalized)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAccessBlockImportFetch, err)
	}
	return rules, nil
}

func (s *AccessBlockService) replaceImportedRules(ctx context.Context, source domain.AccessBlockImportSource, rules []domain.AccessBlockRule) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("import_source_id = ?", source.ID).Delete(&domain.AccessBlockRule{}).Error; err != nil {
			return err
		}
		if len(rules) == 0 {
			return nil
		}
		return tx.CreateInBatches(rules, 500).Error
	})
}

func (s *AccessBlockService) markImportSourceSynced(ctx context.Context, source domain.AccessBlockImportSource, status string, lastError string, count int) domain.AccessBlockImportSource {
	now := time.Now()
	source.LastSyncAt = &now
	source.LastStatus = status
	source.LastError = lastError
	source.LastCount = count
	source.NextSyncAt = accessBlockNextSyncAt(now, source.Enabled, source.UpdateIntervalSeconds)
	source.UpdatedAt = &now
	_ = s.db.WithContext(ctx).Model(&domain.AccessBlockImportSource{}).Where("id = ?", source.ID).Updates(map[string]any{
		"last_sync_at": &now,
		"last_status":  status,
		"last_error":   lastError,
		"last_count":   count,
		"next_sync_at": source.NextSyncAt,
		"updated_at":   now,
	}).Error
	return source
}

func accessBlockNextSyncAt(now time.Time, enabled bool, intervalSeconds int) *time.Time {
	if !enabled {
		return nil
	}
	if intervalSeconds <= 0 {
		intervalSeconds = int(time.Hour.Seconds())
	}
	next := now.Add(time.Duration(intervalSeconds) * time.Second)
	return &next
}

func accessBlockImportSourceDTO(row domain.AccessBlockImportSource) AccessBlockImportSourceDTO {
	return AccessBlockImportSourceDTO{
		ID:                    row.ID,
		URL:                   row.URL,
		Enabled:               row.Enabled,
		Priority:              row.Priority,
		Action:                row.Action,
		StatusCode:            row.StatusCode,
		RedirectURL:           row.RedirectURL,
		Note:                  row.Note,
		UpdateIntervalSeconds: row.UpdateIntervalSeconds,
		LastSyncAt:            row.LastSyncAt,
		NextSyncAt:            row.NextSyncAt,
		LastStatus:            row.LastStatus,
		LastError:             row.LastError,
		LastCount:             row.LastCount,
		CreatedAt:             row.CreatedAt,
		UpdatedAt:             row.UpdatedAt,
	}
}

func validAccessBlockImportURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed == nil || !parsed.IsAbs() || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}
