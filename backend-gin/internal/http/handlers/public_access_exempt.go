package handlers

import (
	"context"
	"errors"
	"net/http"
	pathpkg "path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/repositories"
)

const (
	publicAccessExemptOnlyContextKey = "public_access_exempt_only"
	fileAccessPositiveCacheTTL       = time.Minute
	fileAccessNegativeCacheTTL       = 10 * time.Second
)

type fileAccessCacheEntry struct {
	Allowed bool `json:"allowed"`
}

func publicAccessExemptOnly(c *gin.Context) bool {
	value, ok := c.Get(publicAccessExemptOnlyContextKey)
	enabled, _ := value.(bool)
	return ok && enabled
}

func (h NativeHandlers) allowRestrictedGuestAccess(c *gin.Context) bool {
	if c.Request.Method != http.MethodGet {
		return false
	}
	if postID, ok := postIDFromPostsPath(c.Request.URL.Path); ok {
		return h.publicAccessExemptPost(c.Request.Context(), postID)
	}
	switch c.Request.URL.Path {
	case "/api/categories", "/api/categories/hot", "/api/tags", "/api/tags/hot":
		return true
	case "/api/posts", "/api/posts/recommended", "/api/posts/hot", "/api/search":
		c.Set(publicAccessExemptOnlyContextKey, true)
		return true
	default:
		return false
	}
}

func postIDFromPostsPath(path string) (int64, bool) {
	const prefix = "/api/posts/"
	if !strings.HasPrefix(path, prefix) {
		return 0, false
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" {
		return 0, false
	}
	segment := rest
	if before, _, ok := strings.Cut(rest, "/"); ok {
		segment = before
	}
	id, err := strconv.ParseInt(segment, 10, 64)
	return id, err == nil && id > 0
}

func (h NativeHandlers) publicAccessExemptPost(ctx context.Context, postID int64) bool {
	if h.DB == nil || postID <= 0 {
		return false
	}
	var count int64
	err := h.DB.WithContext(ctx).Model(&domain.Post{}).
		Where("id = ? AND is_draft = ? AND visibility = ? AND public_access_exempt = ?", postID, false, repositories.VisibilityPublic, true).
		Count(&count).Error
	return err == nil && count > 0
}

func (h NativeHandlers) fileBelongsToPublicAccessExemptPost(ctx context.Context, canonicalPath string) bool {
	if h.DB == nil || canonicalPath == "" {
		return false
	}

	cacheKey := h.fileAccessCacheKey(canonicalPath)
	if h.Redis != nil {
		var cached fileAccessCacheEntry
		if h.Redis.CacheGetJSON(ctx, cacheKey, &cached) {
			return cached.Allowed
		}
	}

	loader := func() (any, error) {
		return h.fileBelongsToPublicAccessExemptPostDB(ctx, canonicalPath)
	}
	var value any
	var err error
	if h.FileAccessGroup != nil {
		value, err, _ = h.FileAccessGroup.Do(cacheKey, loader)
	} else {
		value, err = loader()
	}
	if err != nil {
		return false
	}
	allowed, _ := value.(bool)
	if h.Redis != nil {
		ttl := fileAccessNegativeCacheTTL
		if allowed {
			ttl = fileAccessPositiveCacheTTL
		}
		_ = h.Redis.CacheSet(ctx, cacheKey, fileAccessCacheEntry{Allowed: allowed}, ttl)
	}
	return allowed
}

func (h NativeHandlers) fileAccessCacheKey(canonicalPath string) string {
	return h.cacheKeyWithVersions(cacheScopeFileAccess, []string{cacheScopePosts}, canonicalPath)
}

func (h NativeHandlers) fileBelongsToPublicAccessExemptPostDB(ctx context.Context, canonicalPath string) (bool, error) {
	if h.DB == nil || canonicalPath == "" {
		return false, nil
	}
	imageAllowed, err := queryExists(h.DB.WithContext(ctx).Table("post_images pi").
		Joins("JOIN posts p ON p.id = pi.post_id").
		Joins("LEFT JOIN post_payment_settings pps ON pps.post_id = p.id AND pps.enabled = ?", true).
		Where("pi.image_url = ?", canonicalPath).
		Where("p.is_draft = ? AND p.visibility = ? AND p.public_access_exempt = ?", false, repositories.VisibilityPublic, true).
		Where("pi.is_protected = ? AND (pps.id IS NULL OR pps.enabled = ? OR (pps.hide_all = ? AND pi.is_free_preview = ?))", false, false, false, true))
	if err != nil {
		return false, err
	}
	if imageAllowed {
		return true, nil
	}

	candidates := []string{canonicalPath}
	if strings.HasPrefix(canonicalPath, "/api/file/videos/") && strings.EqualFold(pathpkg.Ext(canonicalPath), ".m4s") {
		candidates = append(candidates, pathpkg.Join(pathpkg.Dir(canonicalPath), "manifest.mpd"))
	}
	videoBase := h.DB.WithContext(ctx).Table("post_videos pv").
		Joins("JOIN posts p ON p.id = pv.post_id").
		Joins("LEFT JOIN post_payment_settings pps ON pps.post_id = p.id AND pps.enabled = ?", true).
		Where("p.is_draft = ? AND p.visibility = ? AND p.public_access_exempt = ?", false, repositories.VisibilityPublic, true)
	coverAllowed, err := queryExists(videoBase.Session(&gorm.Session{}).Where("pv.cover_url IN ?", candidates))
	if err != nil {
		return false, err
	}
	if coverAllowed {
		return true, nil
	}
	publicVideoAllowed, err := queryExists(videoBase.Session(&gorm.Session{}).
		Where("pps.id IS NULL OR pps.enabled = ?", false).
		Where("pv.video_url IN ? OR pv.dash_url IN ? OR pv.preview_video_url IN ?", candidates, candidates, candidates))
	if err != nil {
		return false, err
	}
	if publicVideoAllowed {
		return true, nil
	}
	previewAllowed, err := queryExists(videoBase.Session(&gorm.Session{}).
		Where("pps.enabled = ? AND pv.preview_video_url IN ?", true, candidates))
	if err != nil {
		return false, err
	}
	return previewAllowed, nil
}

func queryExists(query *gorm.DB) (bool, error) {
	var row struct {
		Found int `gorm:"column:found"`
	}
	err := query.Select("1 AS found").Limit(1).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return row.Found == 1, nil
}
