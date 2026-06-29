package server

import (
	"net"
	"strings"

	"github.com/gin-gonic/gin"
)

var defaultClientIPHeaders = []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"}

func requestClientIP(c *gin.Context, headers []string) string {
	if c == nil || c.Request == nil {
		return ""
	}
	for _, header := range normalizedClientIPHeaders(headers) {
		if ip := clientIPFromHeaderValue(header, c.GetHeader(header)); ip != "" {
			return ip
		}
	}
	if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		if ip := normalizeIPCandidate(host); ip != "" {
			return ip
		}
	}
	return c.ClientIP()
}

func normalizedClientIPHeaders(headers []string) []string {
	if len(headers) == 0 {
		return defaultClientIPHeaders
	}
	out := make([]string, 0, len(headers))
	for _, header := range headers {
		header = strings.TrimSpace(header)
		if header != "" {
			out = append(out, header)
		}
	}
	if len(out) == 0 {
		return defaultClientIPHeaders
	}
	return out
}

func clientIPFromHeaderValue(header string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.EqualFold(header, "Forwarded") {
		return clientIPFromForwardedHeader(value)
	}
	for part := range strings.SplitSeq(value, ",") {
		if ip := normalizeIPCandidate(part); ip != "" {
			return ip
		}
	}
	return ""
}

func clientIPFromForwardedHeader(value string) string {
	for group := range strings.SplitSeq(value, ",") {
		for part := range strings.SplitSeq(group, ";") {
			key, raw, ok := strings.Cut(strings.TrimSpace(part), "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(key), "for") {
				continue
			}
			if ip := normalizeIPCandidate(raw); ip != "" {
				return ip
			}
		}
	}
	return ""
}

func normalizeIPCandidate(value string) string {
	value = strings.TrimSpace(strings.Trim(value, `"`))
	if value == "" || strings.EqualFold(value, "unknown") {
		return ""
	}
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		host = strings.Trim(host, "[]")
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
	}
	value = strings.Trim(value, "[]")
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	return ""
}
