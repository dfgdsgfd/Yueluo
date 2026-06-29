package services

import (
	"context"
	"strings"
	"sync"
)

type aiJobEventBroker struct {
	mu          sync.Mutex
	subscribers map[string]map[chan AIStreamEvent]struct{}
}

func newAIJobEventBroker() *aiJobEventBroker {
	return &aiJobEventBroker{subscribers: map[string]map[chan AIStreamEvent]struct{}{}}
}

func (b *aiJobEventBroker) Subscribe(ctx context.Context, jobID string) (<-chan AIStreamEvent, func()) {
	jobID = strings.TrimSpace(jobID)
	ch := make(chan AIStreamEvent, 64)
	if b == nil || jobID == "" {
		close(ch)
		return ch, func() {}
	}
	b.mu.Lock()
	if b.subscribers[jobID] == nil {
		b.subscribers[jobID] = map[chan AIStreamEvent]struct{}{}
	}
	b.subscribers[jobID][ch] = struct{}{}
	b.mu.Unlock()
	var once sync.Once
	cancel := func() {
		once.Do(func() {
			b.mu.Lock()
			if subscribers := b.subscribers[jobID]; subscribers != nil {
				delete(subscribers, ch)
				if len(subscribers) == 0 {
					delete(b.subscribers, jobID)
				}
			}
			b.mu.Unlock()
			close(ch)
		})
	}
	go func() {
		<-ctx.Done()
		cancel()
	}()
	return ch, cancel
}

func (b *aiJobEventBroker) Publish(jobID string, event AIStreamEvent) {
	jobID = strings.TrimSpace(jobID)
	if b == nil || jobID == "" || event.Type == "" {
		return
	}
	event.JobID = jobID
	b.mu.Lock()
	subscribers := b.subscribers[jobID]
	for ch := range subscribers {
		select {
		case ch <- event:
		default:
		}
	}
	b.mu.Unlock()
}

func (s *AIService) SubscribeJobEvents(ctx context.Context, jobID string) (<-chan AIStreamEvent, func()) {
	if s == nil || s.jobEvents == nil {
		ch := make(chan AIStreamEvent)
		close(ch)
		return ch, func() {}
	}
	return s.jobEvents.Subscribe(ctx, jobID)
}
