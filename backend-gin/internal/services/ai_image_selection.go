package services

import (
	"math/rand/v2"
	"strings"
)

const (
	AIImageSelectionOrdered = "ordered"
	AIImageSelectionRandom  = "random"
)

func normalizeAIImageSelectionMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AIImageSelectionRandom:
		return AIImageSelectionRandom
	default:
		return AIImageSelectionOrdered
	}
}

func validAIImageSelectionMode(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", AIImageSelectionOrdered, AIImageSelectionRandom:
		return true
	default:
		return false
	}
}

func selectAIImageSample[T any](items []T, maxImages int, mode string) []T {
	if len(items) == 0 || maxImages <= 0 {
		return nil
	}
	if maxImages >= len(items) {
		return append([]T(nil), items...)
	}
	if normalizeAIImageSelectionMode(mode) != AIImageSelectionRandom {
		return append([]T(nil), items[:maxImages]...)
	}
	shuffled := append([]T(nil), items...)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})
	return shuffled[:maxImages]
}
