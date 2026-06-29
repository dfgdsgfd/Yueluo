package handlers

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/services"
)

func (h NativeHandlers) shouldHideVideoCenterForRequest(c *gin.Context, user *services.RequestUser) bool {
	if user == nil {
		return false
	}
	return shouldHideVideoCenterWithCountry(h.Settings, user.CreatedAt, h.videoCenterRequestCountryCode(c))
}

func shouldHideVideoCenterWithCountry(settings *services.SettingsService, createdAt time.Time, countryCode string) bool {
	if settings == nil || !settings.ShouldHideVideoCenterForUser(createdAt) {
		return false
	}

	countryCode = strings.ToUpper(strings.TrimSpace(countryCode))
	return countryCode == "" || countryCode == "CN"
}

func (h NativeHandlers) videoCenterRequestCountryCode(c *gin.Context) string {
	return h.videoCenterRequestCountry(c).Code
}

func (h NativeHandlers) videoCenterRequestCountry(c *gin.Context) services.GeoIPCountry {
	if h.GeoIP == nil {
		return services.GeoIPCountry{}
	}
	return h.GeoIP.CountryForIP(h.clientIP(c))
}
