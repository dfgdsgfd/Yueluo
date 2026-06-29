package handlers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

func hashAPIKey(apiKey string) string {
	sum := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(sum[:])
}

func (h NativeHandlers) lookupOpenAPIKey(ctx context.Context, apiKey string) (*domain.OpenAPI, error) {
	digest := hashAPIKey(apiKey)
	now := time.Now()
	if value, state := h.APIKeys.Lookup(openAPIKeyScope, digest, now); state != apiKeyCacheMiss {
		if state == apiKeyCacheNegative {
			return nil, gorm.ErrRecordNotFound
		}
		if cached, ok := value.(domain.OpenAPI); ok {
			return &cached, nil
		}
	}

	openAPI, err := repositories.NewOpenAPIRepository(h.DB).FindAPIKey(ctx, digest)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		h.APIKeys.StoreNegative(openAPIKeyScope, digest, 0, now)
		return nil, gorm.ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	if !openAPI.IsActive {
		h.APIKeys.StoreNegative(openAPIKeyScope, digest, openAPI.ID, now)
		return nil, gorm.ErrRecordNotFound
	}
	h.APIKeys.StorePositive(openAPIKeyScope, digest, openAPI.ID, *openAPI, now)
	return openAPI, nil
}

func (h NativeHandlers) lookupUserAPIKey(ctx context.Context, apiKey string) (*domain.UserAPIKey, error) {
	digest := hashAPIKey(apiKey)
	now := time.Now()
	if value, state := h.APIKeys.Lookup(userAPIKeyScope, digest, now); state != apiKeyCacheMiss {
		if state == apiKeyCacheNegative {
			return nil, gorm.ErrRecordNotFound
		}
		if cached, ok := value.(domain.UserAPIKey); ok {
			return &cached, nil
		}
	}

	var row domain.UserAPIKey
	err := h.DB.WithContext(ctx).Where("api_key = ?", digest).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		h.APIKeys.StoreNegative(userAPIKeyScope, digest, 0, now)
		return nil, gorm.ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	if !row.IsActive {
		h.APIKeys.StoreNegative(userAPIKeyScope, digest, row.ID, now)
		return nil, gorm.ErrRecordNotFound
	}
	h.APIKeys.StorePositive(userAPIKeyScope, digest, row.ID, row, now)
	return &row, nil
}

func (h NativeHandlers) apiKeyRequestBlocked(c *gin.Context, scope string) bool {
	if h.APIKeys == nil || !h.APIKeys.IsBlocked(h.apiKeyLimiterID(c, scope), time.Now()) {
		return false
	}
	h.writeAPIKeyRateLimited(c)
	return true
}

func (h NativeHandlers) rejectInvalidAPIKey(c *gin.Context, scope string, message string) {
	if h.APIKeys != nil && h.APIKeys.RecordInvalid(h.apiKeyLimiterID(c, scope), time.Now()) {
		h.writeAPIKeyRateLimited(c)
		return
	}
	response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, message, nil)
}

func (h NativeHandlers) acceptAPIKey(c *gin.Context, scope string) {
	if h.APIKeys != nil {
		h.APIKeys.ResetInvalid(h.apiKeyLimiterID(c, scope))
	}
}

func (h NativeHandlers) writeAPIKeyRateLimited(c *gin.Context) {
	c.Header("Retry-After", "300")
	response.JSON(c, http.StatusTooManyRequests, response.CodeTooManyRequests, "error.api_key_rate_limited", nil)
}

func (h NativeHandlers) apiKeyLimiterID(c *gin.Context, scope string) string {
	return scope + "\x00" + h.clientIP(c)
}

func (h NativeHandlers) touchOpenAPIKey(id int64) {
	h.touchAPIKey(openAPIKeyScope, id, func(ctx context.Context) error {
		return repositories.NewOpenAPIRepository(h.DB).TouchAPIKey(ctx, id)
	})
}

func (h NativeHandlers) touchUserAPIKey(id int64) {
	h.touchAPIKey(userAPIKeyScope, id, func(ctx context.Context) error {
		return h.DB.WithContext(ctx).Model(&domain.UserAPIKey{}).Where("id = ?", id).Update("last_used_at", time.Now()).Error
	})
}

func (h NativeHandlers) touchAPIKey(scope string, id int64, update func(context.Context) error) {
	if h.DB == nil || id <= 0 || update == nil {
		return
	}
	reservedAt := time.Now()
	if h.APIKeys != nil && !h.APIKeys.ReserveTouch(scope, id, reservedAt) {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := update(ctx); err != nil && h.APIKeys != nil {
			h.APIKeys.ReleaseTouch(scope, id, reservedAt)
		}
	}()
}

func (h NativeHandlers) invalidateOpenAPIKeyIDs(ids ...int64) {
	if h.APIKeys == nil {
		return
	}
	for _, id := range ids {
		h.APIKeys.InvalidateIdentity(openAPIKeyScope, id)
	}
}

func (h NativeHandlers) invalidateUserAPIKeyIDs(ids ...int64) {
	if h.APIKeys == nil {
		return
	}
	for _, id := range ids {
		h.APIKeys.InvalidateIdentity(userAPIKeyScope, id)
	}
}
