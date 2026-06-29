package middleware

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/services"
)

var (
	androidUAPattern = regexp.MustCompile(`(?i)Android`)
	deviceUAPatterns = map[string]*regexp.Regexp{
		"android": androidUAPattern,
		"ios":     regexp.MustCompile(`(?i)(?:iPhone|iPad|iPod|iOS)`),
		"windows": regexp.MustCompile(`(?i)Windows NT`),
		"mac":     regexp.MustCompile(`(?i)Macintosh|Mac OS X`),
		"linux":   regexp.MustCompile(`(?i)Linux`),
	}
)

func UABlock(settings *services.SettingsService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if shouldSkipUABlock(c.Request.URL.Path) || settings == nil || !settings.IsUaBlockEnabled() {
			c.Next()
			return
		}
		blockedDevices := settings.UaBlockDevices()
		if len(blockedDevices) == 0 || !ShouldBlockUserAgent(c.GetHeader("User-Agent"), blockedDevices) {
			c.Next()
			return
		}

		redirectURL := validRedirectURL(settings.UaBlockRedirectURL())
		customHTML := settings.UaBlockPageHTML()
		isAPIRequest := strings.HasPrefix(c.Request.URL.Path, "/api/") || strings.Contains(c.GetHeader("Accept"), "application/json")
		if isAPIRequest {
			c.JSON(http.StatusForbidden, gin.H{
				"code":         403,
				"blocked":      true,
				"message":      "当前设备暂不支持访问",
				"redirect_url": nullableString(redirectURL),
				"page_html":    nullableString(customHTML),
			})
			c.Abort()
			return
		}
		if redirectURL != "" {
			c.Redirect(http.StatusFound, redirectURL)
			c.Abort()
			return
		}
		c.Header("Content-Type", "text/html; charset=utf-8")
		if customHTML != "" {
			c.String(http.StatusForbidden, customHTML)
			c.Abort()
			return
		}
		c.String(http.StatusForbidden, defaultUABlockHTML)
		c.Abort()
	}
}

func ShouldBlockUserAgent(userAgent string, blockedDevices []string) bool {
	if userAgent == "" || len(blockedDevices) == 0 {
		return false
	}
	for _, device := range blockedDevices {
		pattern, ok := deviceUAPatterns[device]
		if !ok {
			continue
		}
		if device == "linux" && androidUAPattern.MatchString(userAgent) {
			continue
		}
		if pattern.MatchString(userAgent) {
			return true
		}
	}
	return false
}

func ValidRedirectURL(value string) string {
	return validRedirectURL(value)
}

func shouldSkipUABlock(path string) bool {
	return strings.HasPrefix(path, "/api/admin") ||
		strings.HasPrefix(path, "/api/file/apk") ||
		path == "/api/app/check-update" ||
		path == "/api/health" ||
		path == "/api/ua-block/check"
}

func validRedirectURL(value string) string {
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || !parsed.IsAbs() {
		return ""
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	return value
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

const defaultUABlockHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>访问受限</title>
<style>
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; background: #f5f5f5; color: #333; }
.container { text-align: center; padding: 40px; }
h1 { font-size: 24px; margin-bottom: 12px; }
p { color: #666; font-size: 16px; }
</style>
</head>
<body>
<div class="container">
<h1>访问受限</h1>
<p>当前设备暂不支持访问，请使用其他设备。</p>
</div>
</body>
</html>`
