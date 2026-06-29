package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	appmiddleware "yuem-go/backend-gin/internal/http/middleware"
	"yuem-go/backend-gin/internal/http/response"
)

func (h NativeHandlers) UABlockCheck(c *gin.Context) {
	if h.Settings == nil || !h.Settings.IsUaBlockEnabled() {
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "blocked": false})
		return
	}
	blockedDevices := h.Settings.UaBlockDevices()
	if len(blockedDevices) == 0 || !appmiddleware.ShouldBlockUserAgent(c.GetHeader("User-Agent"), blockedDevices) {
		c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "blocked": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":         response.CodeSuccess,
		"blocked":      true,
		"redirect_url": nullableString(appmiddleware.ValidRedirectURL(h.Settings.UaBlockRedirectURL())),
		"page_html":    nullableString(h.Settings.UaBlockPageHTML()),
	})
}

func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
