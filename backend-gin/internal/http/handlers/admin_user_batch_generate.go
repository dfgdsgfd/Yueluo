package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/security"
)

const batchAvatarSampleCount = 50

type batchGenerateUsersRequest struct {
	Count int `json:"count"`
}

func (h NativeHandlers) AdminUsersBatchGenerate(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	var body batchGenerateUsersRequest
	_ = c.ShouldBindJSON(&body)
	count := body.Count
	if count <= 0 {
		count = 1
	}
	if count > 500 {
		count = 500
	}
	avatars, err := h.ensureBatchAvatarSamples(c.Request.Context())
	if err != nil {
		response.JSON(c, http.StatusBadGateway, response.CodeError, "error.avatar_samples_download_failed", gin.H{"detail": err.Error()})
		return
	}
	passwordHash, err := security.HashPassword("123456")
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	now := time.Now()
	users := make([]domain.User, 0, count)
	for index := 0; index < count; index++ {
		userID := h.nextBatchUserID(c.Request.Context(), index)
		avatar := avatars[index%len(avatars)]
		users = append(users, domain.User{
			UserID:    userID,
			Nickname:  fmt.Sprintf("AI User %s", userID[len(userID)-6:]),
			Password:  &passwordHash,
			Avatar:    &avatar,
			IsActive:  true,
			CreatedAt: now,
			UpdatedAt: &now,
		})
	}
	if err := h.DB.WithContext(c.Request.Context()).Create(&users).Error; err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, gin.H{"detail": err.Error()})
		return
	}
	items := make([]gin.H, 0, len(users))
	for _, user := range users {
		items = append(items, gin.H{"id": user.ID, "user_id": user.UserID, "nickname": user.Nickname, "avatar": h.signFileURLPtr(user.Avatar), "is_active": user.IsActive})
	}
	h.bumpCacheVersions(cacheScopeUsers)
	writeSuccess(c, matrixMsgOK, gin.H{"count": len(items), "items": items})
}

func (h NativeHandlers) ensureBatchAvatarSamples(ctx context.Context) ([]string, error) {
	dir := filepath.Join("static", "upload", "avatars", "dicebear")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	urls := make([]string, 0, batchAvatarSampleCount)
	client := &http.Client{Timeout: 20 * time.Second}
	for index := 1; index <= batchAvatarSampleCount; index++ {
		filename := fmt.Sprintf("yuem-ai-avatar-%02d.png", index)
		path := filepath.Join(dir, filename)
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			urls = append(urls, "/api/file/static/upload/avatars/dicebear/"+filename)
			continue
		}
		seed := fmt.Sprintf("yuem-ai-user-%02d", index)
		apiURL := "https://api.dicebear.com/10.x/adventurer/png?seed=" + seed + "&size=256"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		closeErr := resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 || len(data) == 0 {
			return nil, fmt.Errorf("dicebear returned HTTP %d", resp.StatusCode)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			return nil, err
		}
		urls = append(urls, "/api/file/static/upload/avatars/dicebear/"+filename)
	}
	return urls, nil
}

func (h NativeHandlers) nextBatchUserID(ctx context.Context, index int) string {
	for attempt := range 20 {
		candidate := fmt.Sprintf("ai_%s_%02d", strings.ToLower(randomHex(5)), index+attempt)
		var count int64
		_ = h.DB.WithContext(ctx).Model(&domain.User{}).Where("user_id = ?", candidate).Count(&count).Error
		if count == 0 {
			return candidate
		}
	}
	return "ai_" + strings.ToLower(randomHex(8))
}
