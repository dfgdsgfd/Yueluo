package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/services"
)

type AccessBlockLogger interface {
	RecordAccessBlock(c *gin.Context, match services.AccessBlockMatch, status int)
}

type AccessBlockClientIPFunc func(*gin.Context) string

const (
	AccessBlockHeader           = "X-Access-Block"
	AccessBlockRuleIDHeader     = "X-Access-Block-Rule-ID"
	AccessBlockActionHeader     = "X-Access-Block-Action"
	AccessBlockStatusCodeHeader = "X-Access-Block-Status-Code"
)

func AccessBlock(service *services.AccessBlockService, clientIP AccessBlockClientIPFunc, logger AccessBlockLogger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if service == nil || service.Disabled() {
			c.Next()
			return
		}
		ip := ""
		if clientIP != nil {
			ip = clientIP(c)
		}
		match, ok := service.Match(services.AccessBlockMatchInput{
			IP:        ip,
			UserAgent: c.Request.UserAgent(),
		})
		if !ok {
			c.Next()
			return
		}
		status := match.Rule.StatusCode
		if match.Rule.Action == services.AccessBlockActionRedirect {
			status = http.StatusFound
			applyAccessBlockHeaders(c, match, status)
			if logger != nil {
				logger.RecordAccessBlock(c, match, status)
			}
			c.Redirect(status, match.Rule.RedirectURL)
			c.Abort()
			return
		}
		if status == 0 {
			status = 444
		}
		applyAccessBlockHeaders(c, match, status)
		if logger != nil {
			logger.RecordAccessBlock(c, match, status)
		}
		c.Status(status)
		c.Abort()
	}
}

func applyAccessBlockHeaders(c *gin.Context, match services.AccessBlockMatch, status int) {
	c.Header(AccessBlockHeader, "1")
	c.Header(AccessBlockRuleIDHeader, strconv.FormatInt(match.Rule.ID, 10))
	c.Header(AccessBlockActionHeader, match.Rule.Action)
	c.Header(AccessBlockStatusCodeHeader, strconv.Itoa(status))
}
