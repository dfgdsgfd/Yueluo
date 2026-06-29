package handlers

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

var pyVideoProxyHTTPClient = &http.Client{Timeout: 30 * time.Second}

func (h NativeHandlers) PyVideoProxy(c *gin.Context) {
	if h.Auth == nil || !h.authenticatePyVideoAccess(c) {
		c.Header("Cache-Control", "no-store")
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "访问令牌缺失", nil)
		return
	}

	targetPath := strings.TrimPrefix(c.Param("path"), "/")
	if strings.Contains(targetPath, "..") {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效的路径", nil)
		return
	}
	upstreamBase := h.Config.PyVideo.UpstreamURL
	if upstreamBase == "" {
		upstreamBase = "https://v.yuelk.com"
	}
	targetURL, err := url.Parse(upstreamBase + "/" + targetPath)
	if err != nil {
		response.JSON(c, http.StatusBadGateway, http.StatusBadGateway, "代理请求失败", nil)
		return
	}
	targetURL.RawQuery = c.Request.URL.RawQuery

	req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodGet, targetURL.String(), nil)
	if err != nil {
		response.JSON(c, http.StatusBadGateway, http.StatusBadGateway, "代理请求失败", nil)
		return
	}
	if h.Config.PyVideo.APIKey != "" {
		req.Header.Set("X-API-Key", h.Config.PyVideo.APIKey)
	}

	resp, err := pyVideoProxyHTTPClient.Do(req)
	if err != nil {
		response.JSON(c, http.StatusBadGateway, http.StatusBadGateway, "代理请求失败", nil)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		status := resp.StatusCode
		if status < 400 || status >= 600 {
			status = http.StatusBadGateway
		}
		response.JSON(c, status, status, "上游服务请求失败", nil)
		return
	}
	copyHeaderIfPresent(c, resp.Header, "Content-Type")
	copyHeaderIfPresent(c, resp.Header, "Content-Length")
	copyHeaderIfPresent(c, resp.Header, "Cache-Control")
	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}

func (h NativeHandlers) authenticatePyVideoAccess(c *gin.Context) bool {
	if h.Auth == nil {
		return false
	}
	if cookieToken, err := c.Cookie("file_token"); err == nil && cookieToken != "" {
		if _, ok := h.Auth.VerifyFileToken(cookieToken); ok {
			return true
		}
	}
	bearer := services.ExtractBearerToken(c.GetHeader("Authorization"))
	if bearer == "" {
		return false
	}
	_, ok := h.Auth.VerifyTokenClaims(bearer)
	return ok
}

func copyHeaderIfPresent(c *gin.Context, source http.Header, key string) {
	if value := source.Get(key); value != "" {
		c.Header(key, value)
	}
}
