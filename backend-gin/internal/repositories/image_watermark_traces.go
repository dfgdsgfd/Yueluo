package repositories

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"time"

	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
)

const imageWatermarkTokenCreateAttempts = 8

var imageWatermarkTokenGenerator = func() (string, error) {
	var raw [domain.ImageWatermarkTraceTokenBytes]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw[:]), nil
}

var imageWatermarkShortCodeGenerator = func(byteLen int) (string, error) {
	if byteLen <= 0 {
		byteLen = domain.ImageWatermarkShortCodeBytes
	}
	raw := make([]byte, byteLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}

type ImageWatermarkTraceInput struct {
	TraceType      string
	FieldFlags     int
	ShortCodeBytes int
	UserID         int64
	UserDisplayID  string
	Username       string
	PostID         *int64
	ImageID        *int64
	JobID          string
	SourceURL      string
	SourceHash     string
	CustomText     string
}

func ReserveImageWatermarkTrace(ctx context.Context, db *gorm.DB, input ImageWatermarkTraceInput) (domain.ImageWatermarkTrace, error) {
	if db == nil {
		return domain.ImageWatermarkTrace{}, errors.New("image watermark trace database is not configured")
	}
	traceType := strings.TrimSpace(input.TraceType)
	if traceType == "" {
		traceType = domain.ImageWatermarkTraceUpload
	}
	for range imageWatermarkTokenCreateAttempts {
		token, err := imageWatermarkTokenGenerator()
		if err != nil {
			return domain.ImageWatermarkTrace{}, err
		}
		shortCodeBytes := normalizeImageWatermarkShortCodeBytes(input.ShortCodeBytes)
		code, codeErr := imageWatermarkShortCodeGenerator(shortCodeBytes)
		if codeErr != nil {
			return domain.ImageWatermarkTrace{}, codeErr
		}
		code = strings.ToLower(strings.TrimSpace(code))
		shortCode := &code
		now := time.Now()
		trace := domain.ImageWatermarkTrace{
			Token:          token,
			ShortCode:      shortCode,
			ShortCodeBytes: shortCodeBytes,
			TraceType:      traceType,
			PayloadVersion: domain.ImageWatermarkPayloadVersion,
			PayloadBytes:   shortCodeBytes,
			FieldFlags:     input.FieldFlags,
			UserID:         input.UserID,
			UserDisplayID:  strings.TrimSpace(input.UserDisplayID),
			Username:       strings.TrimSpace(input.Username),
			PostID:         input.PostID,
			ImageID:        input.ImageID,
			JobID:          strings.TrimSpace(input.JobID),
			SourceURL:      strings.TrimSpace(input.SourceURL),
			SourceHash:     strings.TrimSpace(input.SourceHash),
			CustomText:     strings.TrimSpace(input.CustomText),
			CreatedAt:      now,
			UpdatedAt:      &now,
		}
		if err := db.WithContext(ctx).Create(&trace).Error; err == nil {
			return trace, nil
		} else if !isUniqueConstraintError(err) {
			return domain.ImageWatermarkTrace{}, err
		}
	}
	return domain.ImageWatermarkTrace{}, errors.New("image watermark token collision retry limit exceeded")
}

func ResolveImageWatermarkTrace(ctx context.Context, db *gorm.DB, token string) (domain.ImageWatermarkTrace, error) {
	var trace domain.ImageWatermarkTrace
	if db == nil {
		return trace, gorm.ErrRecordNotFound
	}
	normalized := strings.ToLower(strings.TrimSpace(token))
	if len(normalized) != domain.ImageWatermarkTraceTokenBytes*2 {
		return trace, gorm.ErrRecordNotFound
	}
	err := db.WithContext(ctx).Where("token = ?", normalized).First(&trace).Error
	return trace, err
}

func normalizeImageWatermarkShortCodeBytes(value int) int {
	if value < domain.ImageWatermarkMinShortCodeBytes || value > domain.ImageWatermarkShortCodeBytes {
		return domain.ImageWatermarkShortCodeBytes
	}
	return value
}

func ResolveImageWatermarkTraceByShortCode(ctx context.Context, db *gorm.DB, shortCode string, byteLen int) (domain.ImageWatermarkTrace, error) {
	var trace domain.ImageWatermarkTrace
	if db == nil {
		return trace, gorm.ErrRecordNotFound
	}
	normalized := strings.ToLower(strings.TrimSpace(shortCode))
	if byteLen <= 0 {
		byteLen = len(normalized) / 2
	}
	if len(normalized) != byteLen*2 {
		return trace, gorm.ErrRecordNotFound
	}
	err := db.WithContext(ctx).Where("short_code = ?", normalized).First(&trace).Error
	return trace, err
}

func ListImageWatermarkRecoverDimensions(ctx context.Context, db *gorm.DB, limit int) ([][2]int, error) {
	if db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 32
	}
	type dimensionRow struct {
		Width  int `gorm:"column:watermark_width"`
		Height int `gorm:"column:watermark_height"`
	}
	var rows []dimensionRow
	err := db.WithContext(ctx).
		Model(&domain.ImageWatermarkTrace{}).
		Select("watermark_width, watermark_height").
		Where("trace_type = ? AND watermark_width > 0 AND watermark_height > 0", domain.ImageWatermarkTraceProtected).
		Group("watermark_width, watermark_height").
		Order("watermark_width ASC, watermark_height ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	dimensions := make([][2]int, 0, len(rows))
	for _, row := range rows {
		if row.Width <= 0 || row.Height <= 0 {
			continue
		}
		dimension := [2]int{row.Width, row.Height}
		seen := slices.Contains(dimensions, dimension)
		if !seen {
			dimensions = append(dimensions, dimension)
		}
	}
	return dimensions, nil
}

func DeleteImageWatermarkTrace(ctx context.Context, db *gorm.DB, token string) {
	if db == nil || strings.TrimSpace(token) == "" {
		return
	}
	_ = db.WithContext(ctx).Where("token = ?", strings.ToLower(strings.TrimSpace(token))).Delete(&domain.ImageWatermarkTrace{}).Error
}

func PreparePostImageWatermarkTraceRebind(ctx context.Context, tx *gorm.DB, postID int64, images []PostImageInput) error {
	if tx == nil || postID <= 0 {
		return nil
	}
	urls := make([]string, 0, len(images))
	seen := map[string]bool{}
	for _, image := range images {
		url := strings.TrimSpace(image.URL)
		if url != "" && !seen[url] {
			seen[url] = true
			urls = append(urls, url)
		}
	}
	query := tx.WithContext(ctx).Where("post_id = ?", postID)
	if len(urls) == 0 {
		return query.Delete(&domain.ImageWatermarkTrace{}).Error
	}
	if err := query.Where("source_url NOT IN ?", urls).Delete(&domain.ImageWatermarkTrace{}).Error; err != nil {
		return err
	}
	return tx.WithContext(ctx).Model(&domain.ImageWatermarkTrace{}).
		Where("post_id = ? AND source_url IN ?", postID, urls).
		Update("image_id", nil).Error
}

func BindPostImageWatermarkTraces(ctx context.Context, tx *gorm.DB, postID, userID int64, images []domain.PostImage) error {
	if tx == nil || postID <= 0 {
		return nil
	}
	now := time.Now()
	for index := range images {
		image := &images[index]
		token := strings.ToLower(strings.TrimSpace(image.WatermarkTraceToken))
		if token != "" {
			query := tx.WithContext(ctx).Model(&domain.ImageWatermarkTrace{}).
				Where("token = ? AND source_url = ? AND (post_id IS NULL OR post_id = ?)", token, image.ImageURL, postID)
			if userID > 0 {
				query = query.Where("user_id = ?", userID)
			}
			result := query.
				Updates(map[string]any{
					"post_id":    postID,
					"image_id":   image.ID,
					"source_url": image.ImageURL,
					"updated_at": now,
				})
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return ErrContentInvalidArgument
			}
		}
		if err := tx.WithContext(ctx).Model(&domain.ImageWatermarkTrace{}).
			Where("post_id = ? AND source_url = ?", postID, image.ImageURL).
			Updates(map[string]any{"image_id": image.ID, "updated_at": now}).Error; err != nil {
			return err
		}
		if token != "" {
			continue
		}
		var trace domain.ImageWatermarkTrace
		if userID <= 0 {
			continue
		}
		err := tx.WithContext(ctx).
			Where("trace_type = ? AND user_id = ? AND post_id IS NULL AND source_url = ?", domain.ImageWatermarkTraceUpload, userID, image.ImageURL).
			Order("created_at DESC").
			First(&trace).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		image.WatermarkTraceToken = trace.Token
		if err := tx.WithContext(ctx).Model(&domain.PostImage{}).Where("id = ?", image.ID).
			Update("watermark_trace_token", trace.Token).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Model(&trace).Updates(map[string]any{
			"post_id":    postID,
			"image_id":   image.ID,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}

func isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unique") || strings.Contains(message, "duplicate")
}
