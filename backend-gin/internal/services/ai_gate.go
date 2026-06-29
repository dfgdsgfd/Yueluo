package services

import (
	"context"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const defaultAITokensPerSecond = 60.0

type aiConcurrencyGate struct {
	mu              sync.Mutex
	active          int
	activeTickets   map[string]*aiQueueTicket
	queue           []*aiQueueTicket
	nextSeq         int64
	tokensPerSecond float64
}

type aiQueueTicket struct {
	jobID           string
	seq             int64
	ready           chan struct{}
	estimatedTokens int
	acquired        bool
	canceled        bool
	released        bool
}

type aiQueueSnapshot struct {
	acquired        bool
	position        int
	total           int
	etaSeconds      int
	estimatedTokens int
	tokensPerSecond float64
}

func newAIConcurrencyGate() *aiConcurrencyGate {
	return &aiConcurrencyGate{
		activeTickets:   map[string]*aiQueueTicket{},
		tokensPerSecond: defaultAITokensPerSecond,
	}
}

func (g *aiConcurrencyGate) enqueue(jobID string, estimatedTokens int, limit int) *aiQueueTicket {
	if g == nil {
		return nil
	}
	ticket := &aiQueueTicket{
		jobID:           jobID,
		ready:           make(chan struct{}),
		estimatedTokens: maxAIInt(1, estimatedTokens),
	}
	g.mu.Lock()
	g.nextSeq++
	ticket.seq = g.nextSeq
	g.queue = append(g.queue, ticket)
	g.promoteLocked(limit)
	g.mu.Unlock()
	return ticket
}

func (g *aiConcurrencyGate) snapshot(ticket *aiQueueTicket, limit int) aiQueueSnapshot {
	if g == nil || ticket == nil {
		return aiQueueSnapshot{acquired: true, tokensPerSecond: defaultAITokensPerSecond}
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.promoteLocked(limit)
	return g.snapshotLocked(ticket, limit)
}

func (g *aiConcurrencyGate) cancel(ticket *aiQueueTicket, limit int) {
	if g == nil || ticket == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if ticket.released {
		return
	}
	ticket.canceled = true
	if ticket.acquired {
		ticket.released = true
		if g.active > 0 {
			g.active--
		}
		delete(g.activeTickets, ticket.jobID)
	} else {
		g.removeQueuedLocked(ticket)
	}
	g.promoteLocked(limit)
}

func (g *aiConcurrencyGate) release(ticket *aiQueueTicket, limit int) {
	if g == nil || ticket == nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if ticket.released {
		return
	}
	ticket.released = true
	if ticket.acquired && g.active > 0 {
		g.active--
	}
	delete(g.activeTickets, ticket.jobID)
	g.promoteLocked(limit)
}

func (g *aiConcurrencyGate) recordThroughput(tokens int, duration time.Duration) {
	if g == nil || tokens <= 0 || duration <= 0 {
		return
	}
	next := float64(tokens) / duration.Seconds()
	if next <= 0 {
		return
	}
	if next < 10 {
		next = 10
	}
	if next > 1000 {
		next = 1000
	}
	g.mu.Lock()
	if g.tokensPerSecond <= 0 {
		g.tokensPerSecond = next
	} else {
		g.tokensPerSecond = g.tokensPerSecond*0.8 + next*0.2
	}
	g.mu.Unlock()
}

func (g *aiConcurrencyGate) promoteLocked(limit int) {
	limit = normalizeAIConcurrency(limit)
	for g.active < limit && len(g.queue) > 0 {
		ticket := g.queue[0]
		g.queue = g.queue[1:]
		if ticket == nil || ticket.canceled || ticket.released {
			continue
		}
		ticket.acquired = true
		g.active++
		g.activeTickets[ticket.jobID] = ticket
		close(ticket.ready)
	}
}

func (g *aiConcurrencyGate) snapshotLocked(ticket *aiQueueTicket, limit int) aiQueueSnapshot {
	speed := g.tokensPerSecond
	if speed <= 0 {
		speed = defaultAITokensPerSecond
	}
	if ticket.acquired {
		return aiQueueSnapshot{
			acquired:        true,
			estimatedTokens: ticket.estimatedTokens,
			tokensPerSecond: speed,
		}
	}
	position := 0
	tokensAhead := 0
	for _, active := range g.activeTickets {
		tokensAhead += maxAIInt(1, active.estimatedTokens)
	}
	for index, queued := range g.queue {
		if queued == nil || queued.canceled || queued.released {
			continue
		}
		if queued == ticket {
			position = index + 1
			break
		}
		tokensAhead += maxAIInt(1, queued.estimatedTokens)
	}
	if position == 0 {
		position = len(g.queue)
	}
	parallelSpeed := speed * float64(normalizeAIConcurrency(limit))
	etaSeconds := 0
	if tokensAhead > 0 && parallelSpeed > 0 {
		etaSeconds = int(float64(tokensAhead)/parallelSpeed + 0.999)
	}
	if etaSeconds < 0 {
		etaSeconds = 0
	}
	return aiQueueSnapshot{
		position:        position,
		total:           len(g.queue),
		etaSeconds:      etaSeconds,
		estimatedTokens: ticket.estimatedTokens,
		tokensPerSecond: speed,
	}
}

func (g *aiConcurrencyGate) removeQueuedLocked(ticket *aiQueueTicket) {
	for index, queued := range g.queue {
		if queued != ticket {
			continue
		}
		g.queue = append(g.queue[:index], g.queue[index+1:]...)
		return
	}
}

func (s *AIService) acquireAIConcurrency(
	ctx context.Context,
	cfg AIConfig,
	tmpl AITemplateConfig,
	templateKey string,
	jobID string,
	totalChunks int,
	estimatedTokens int,
	emit func(AIStreamEvent) error,
) (func(), error) {
	if s == nil {
		return func() {}, nil
	}
	releases := make([]func(), 0, 2)
	released := false
	releaseAll := func() {
		if released {
			return
		}
		released = true
		for i := len(releases) - 1; i >= 0; i-- {
			releases[i]()
		}
	}
	if limit := aiTemplateConcurrencyLimit(tmpl); limit > 0 {
		release, err := acquireAIConcurrencyGateTicket(ctx, s.aiProjectGate(templateKey, tmpl), limit, jobID, totalChunks, estimatedTokens, emit)
		if err != nil {
			releaseAll()
			return func() {}, err
		}
		releases = append(releases, release)
	}
	if s.gate != nil {
		release, err := acquireAIConcurrencyGateTicket(ctx, s.gate, normalizeAIConcurrency(cfg.Concurrency), jobID, totalChunks, estimatedTokens, emit)
		if err != nil {
			releaseAll()
			return func() {}, err
		}
		releases = append(releases, release)
	}
	return releaseAll, nil
}

func acquireAIConcurrencyGateTicket(
	ctx context.Context,
	gate *aiConcurrencyGate,
	limit int,
	jobID string,
	totalChunks int,
	estimatedTokens int,
	emit func(AIStreamEvent) error,
) (func(), error) {
	if gate == nil {
		return func() {}, nil
	}
	limit = normalizeAIConcurrency(limit)
	ticket := gate.enqueue(jobID, estimatedTokens, limit)
	released := false
	release := func() {
		if released {
			return
		}
		released = true
		gate.release(ticket, limit)
	}
	emitQueued := func(snapshot aiQueueSnapshot) error {
		if snapshot.acquired {
			return nil
		}
		return emit(AIStreamEvent{
			Type:            "progress",
			JobID:           jobID,
			Percent:         0,
			CurrentChunk:    0,
			TotalChunks:     totalChunks,
			Stage:           "queued",
			QueuePosition:   snapshot.position,
			QueueTotal:      snapshot.total,
			ETASeconds:      snapshot.etaSeconds,
			EstimatedTokens: snapshot.estimatedTokens,
			TokensPerSecond: snapshot.tokensPerSecond,
		})
	}

	if snapshot := gate.snapshot(ticket, limit); snapshot.acquired {
		return release, nil
	} else if err := emitQueued(snapshot); err != nil {
		gate.cancel(ticket, limit)
		return func() {}, err
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticket.ready:
			return release, nil
		case <-ctx.Done():
			gate.cancel(ticket, limit)
			return func() {}, AIError{Code: "error.ai_request_canceled", Err: ctx.Err()}
		case <-ticker.C:
			snapshot := gate.snapshot(ticket, limit)
			if snapshot.acquired {
				return release, nil
			}
			if err := emitQueued(snapshot); err != nil {
				gate.cancel(ticket, limit)
				return func() {}, err
			}
		}
	}
}

func (s *AIService) aiProjectGate(templateKey string, tmpl AITemplateConfig) *aiConcurrencyGate {
	key := aiProjectConcurrencyKey(templateKey, tmpl)
	if key == "" {
		key = "default"
	}
	s.projectGatesMu.Lock()
	defer s.projectGatesMu.Unlock()
	if s.projectGates == nil {
		s.projectGates = map[string]*aiConcurrencyGate{}
	}
	gate := s.projectGates[key]
	if gate == nil {
		gate = newAIConcurrencyGate()
		s.projectGates[key] = gate
	}
	return gate
}

func aiProjectConcurrencyKey(templateKey string, tmpl AITemplateConfig) string {
	if key := strings.TrimSpace(templateKey); key != "" {
		return "template:" + key
	}
	if taskType := strings.TrimSpace(tmpl.TaskType); taskType != "" {
		return "task:" + taskType
	}
	return ""
}

func aiTemplateConcurrencyLimit(tmpl AITemplateConfig) int {
	if tmpl.Concurrency <= 0 {
		return 0
	}
	return boundedInt(tmpl.Concurrency, 1, 50, 0)
}

func (s *AIService) recordAIThroughput(usage AIUsage, estimatedTokens int, duration time.Duration) {
	if s == nil || s.gate == nil {
		return
	}
	s.gate.recordThroughput(aiCompletionTokenCount(usage, estimatedTokens), duration)
}

func calculateAITokensPerSecond(usage AIUsage, estimatedTokens int, duration time.Duration) float64 {
	tokens := aiCompletionTokenCount(usage, estimatedTokens)
	if tokens <= 0 || duration <= 0 {
		return 0
	}
	return float64(tokens) / duration.Seconds()
}

func aiCompletionTokenCount(usage AIUsage, estimatedTokens int) int {
	tokens := usage.CompletionTokens
	if tokens <= 0 {
		tokens = usage.TotalTokens
	}
	if tokens <= 0 {
		tokens = estimatedTokens
	}
	return tokens
}

func estimateAIRequestTokens(req AIRequest, cfg AIConfig, tmpl AITemplateConfig, chunks []string) int {
	total := 0
	for _, chunk := range chunks {
		total += estimateAITokens(chunk)
	}
	promptTokens := estimateAITokens(tmpl.SystemPrompt) + estimateAITokens(tmpl.UserPrompt)
	if promptTokens == 0 {
		promptTokens = estimateAITokens(tmpl.Prompt)
	}
	total += promptTokens * maxAIInt(1, len(chunks))
	maxOutputTokens := cfg.MaxOutputTokens
	if tmpl.MaxOutputTokens > 0 || tmpl.maxOutputSet {
		maxOutputTokens = tmpl.MaxOutputTokens
	}
	if req.Options.MaxOutputTokens != nil && *req.Options.MaxOutputTokens > 0 {
		maxOutputTokens = *req.Options.MaxOutputTokens
	}
	total += maxOutputTokens * maxAIInt(1, len(chunks))
	if len(req.Images) > 0 {
		total += len(req.Images) * 1000
	}
	if target, ok := contentFormatTargetForTask(cfg.ContentFormat, req.Type); ok {
		if plan := aiCustomContinuationPlanForRequest(req, target); plan.Enabled {
			total *= plan.estimatedTokenMultiplier()
		}
	}
	return maxAIInt(1, total)
}

func estimateAITokens(text string) int {
	runes := utf8.RuneCountInString(text)
	if runes <= 0 {
		return 0
	}
	return runes/4 + 1
}

func normalizeAIConcurrency(value int) int {
	return boundedInt(value, 1, 50, 5)
}

func maxAIInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
