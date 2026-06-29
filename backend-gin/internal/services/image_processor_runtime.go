package services

import (
	"context"
	"time"

	"go.uber.org/zap"
)

func (p *ImageProcessor) boolSetting(key string, fallback bool) bool {
	if p.settings == nil {
		return fallback
	}
	value := p.settings.Get(key)
	if value == nil {
		return fallback
	}
	return settingBool(value)
}

func (p *ImageProcessor) intSetting(key string, fallback, minValue, maxValue int) int {
	value := fallback
	if p.settings != nil {
		value = p.settings.Int(key, fallback)
	}
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func (p *ImageProcessor) stringSetting(key string) string {
	if p.settings == nil {
		return ""
	}
	return p.settings.String(key)
}

func (p *ImageProcessor) warn(message string, fields ...zap.Field) {
	if p.logger != nil {
		p.logger.Warn(message, fields...)
	}
}

func (g *imageProcessingGate) acquire(ctx context.Context, limit int) error {
	if limit < 1 {
		limit = 1
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		g.mu.Lock()
		if g.active < limit {
			g.active++
			g.mu.Unlock()
			return nil
		}
		g.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (g *imageProcessingGate) release() {
	g.mu.Lock()
	if g.active > 0 {
		g.active--
	}
	g.mu.Unlock()
}
