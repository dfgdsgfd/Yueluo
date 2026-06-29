package handlers

import (
	"context"
	"fmt"
	"time"

	"yuem-go/backend-gin/internal/services"
)

const (
	cacheScopePosts               = "posts"
	cacheScopeComments            = "comments"
	cacheScopeSearch              = "search"
	cacheScopeUsers               = "users"
	cacheScopeInteractions        = "interactions"
	cacheScopeNotifications       = "notifications"
	cacheScopeSystemNotifications = "system_notifications"
	cacheScopeIM                  = "im"
	cacheScopeSettings            = "settings"
	cacheScopeFileAccess          = "file_access"
	cacheScopeCreator             = "creator"
	cacheScopeFileRecycle         = "file_recycle"
)

func (h NativeHandlers) cacheVersion(scope string) int64 {
	if h.Redis == nil {
		return 0
	}
	return h.Redis.CacheVersion(context.Background(), scope)
}

func (h NativeHandlers) cacheVersions(scopes ...string) map[string]int64 {
	if h.Redis == nil {
		return map[string]int64{}
	}
	return h.Redis.CacheVersions(context.Background(), scopes...)
}

func (h NativeHandlers) cacheKey(scope string, parts ...any) string {
	version := int64(0)
	if h.Redis != nil {
		version = h.cacheVersions(scope)[scope]
	}
	return services.CacheKey(scope, version, parts...)
}

func (h NativeHandlers) cacheKeyWithVersions(scope string, versions []string, parts ...any) string {
	scopes := make([]string, 0, len(versions)+1)
	scopes = append(scopes, scope)
	scopes = append(scopes, versions...)
	versionValues := h.cacheVersions(scopes...)
	all := make([]any, 0, len(versions)+len(parts))
	for _, versionScope := range versions {
		all = append(all, fmt.Sprintf("%s=%d", versionScope, versionValues[versionScope]))
	}
	all = append(all, parts...)
	return services.CacheKey(scope, versionValues[scope], all...)
}

func (h NativeHandlers) bumpCacheVersions(scopes ...string) {
	if h.Redis == nil {
		return
	}
	h.Redis.BumpCacheVersion(context.Background(), scopes...)
}

func (h NativeHandlers) bumpAdminResourceCacheVersions(table string) {
	switch table {
	case "posts", "post_images", "post_videos", "post_payment_settings":
		h.bumpCacheVersions(cacheScopePosts, cacheScopeFileAccess)
	case "file_recycle_items":
		h.bumpCacheVersions(cacheScopeFileRecycle)
	}
}

func cacheTTL(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}
