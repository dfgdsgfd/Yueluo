package config

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func getEnv(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func int64Env(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func optionalIntEnv(key string) *int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil
	}
	return &parsed
}

func optionalRangedIntEnv(key string, min int, max int) *int {
	parsed := optionalIntEnv(key)
	if parsed == nil || *parsed < min || *parsed > max {
		return nil
	}
	return parsed
}

func millisecondsEnvAsSeconds(key string, fallbackMS int) int {
	ms := intEnv(key, fallbackMS)
	if ms <= 0 {
		ms = fallbackMS
	}
	seconds := ms / 1000
	if ms%1000 != 0 {
		seconds++
	}
	if seconds <= 0 {
		return 1
	}
	return seconds
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(raw)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func boolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "true" || value == "1" || value == "yes"
}

func floatEnv(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func parseSizeBytes(value string, fallback int64) int64 {
	text := strings.TrimSpace(strings.ToLower(value))
	if text == "" {
		return fallback
	}
	multipliers := []struct {
		suffix string
		value  int64
	}{
		{"gb", 1024 * 1024 * 1024},
		{"g", 1024 * 1024 * 1024},
		{"mb", 1024 * 1024},
		{"m", 1024 * 1024},
		{"kb", 1024},
		{"k", 1024},
		{"b", 1},
	}
	for _, item := range multipliers {
		if before, ok := strings.CutSuffix(text, item.suffix); ok {
			raw := strings.TrimSpace(before)
			parsed, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				return fallback
			}
			return int64(parsed * float64(item.value))
		}
	}
	parsed, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseDASHResolutions(value string) []DASHResolution {
	parts := splitCSV(value)
	out := make([]DASHResolution, 0, len(parts))
	for _, part := range parts {
		resolution, bitrateText, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		bitrate, err := strconv.Atoi(strings.TrimSpace(bitrateText))
		if err != nil || bitrate <= 0 {
			continue
		}
		resolution = strings.TrimSpace(strings.ToLower(resolution))
		item := DASHResolution{Bitrate: bitrate}
		if before, ok0 := strings.CutSuffix(resolution, "p"); ok0 {
			height, err := strconv.Atoi(before)
			if err != nil || height <= 0 {
				continue
			}
			item.Height = height
			item.Label = resolution
			out = append(out, item)
			continue
		}
		widthText, heightText, ok := strings.Cut(resolution, "x")
		if !ok {
			continue
		}
		width, widthErr := strconv.Atoi(strings.TrimSpace(widthText))
		height, heightErr := strconv.Atoi(strings.TrimSpace(heightText))
		if widthErr != nil || heightErr != nil || width <= 0 || height <= 0 {
			continue
		}
		item.Width = width
		item.Height = height
		item.Label = strconv.Itoa(height) + "p"
		out = append(out, item)
	}
	return out
}

func databaseDriver(databaseURL string) string {
	parsed, err := url.Parse(databaseURL)
	if err != nil {
		return ""
	}
	switch parsed.Scheme {
	case "mysql", "mariadb":
		return "mysql"
	case "postgres", "postgresql":
		return "postgres"
	default:
		return parsed.Scheme
	}
}
