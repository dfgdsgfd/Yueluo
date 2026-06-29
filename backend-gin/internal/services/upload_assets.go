package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
)

const (
	UploadAssetStatusTemp         = "temp"
	UploadAssetStatusBound        = "bound"
	UploadAssetStatusDeleted      = "deleted"
	UploadAssetStatusExpired      = "expired"
	UploadAssetStatusDeleteFailed = "delete_failed"

	uploadAssetKindImage      = "image"
	uploadAssetStorageLocal   = "local"
	uploadAssetStorageR2      = "r2"
	uploadAssetStorageUnknown = "unknown"
)

type UploadAssetRecordInput struct {
	UserID       int64
	Purpose      string
	Kind         string
	URL          string
	Storage      string
	ObjectKey    string
	LocalPath    string
	OriginalName string
	Size         int64
	MimeType     string
}

type UploadAssetService struct {
	db     *gorm.DB
	cfg    config.Config
	logger *zap.Logger
	cancel context.CancelFunc
	mu     sync.Mutex
}

func NewUploadAssetService(db *gorm.DB, cfg config.Config, logger *zap.Logger) *UploadAssetService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &UploadAssetService{db: db, cfg: cfg, logger: logger}
}

func (s *UploadAssetService) Start() {
	if s == nil || s.db == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	interval := s.cfg.Upload.Temp.CleanupInterval
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	go s.runCleanup(ctx, interval)
}

func (s *UploadAssetService) Close() {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
}

func (s *UploadAssetService) runCleanup(ctx context.Context, interval time.Duration) {
	_ = s.CleanupExpired(ctx, 200)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = s.CleanupExpired(ctx, 200)
		}
	}
}

func (s *UploadAssetService) Record(ctx context.Context, input UploadAssetRecordInput) (*domain.UploadAsset, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	now := time.Now()
	urlValue := strings.TrimSpace(input.URL)
	if urlValue == "" {
		return nil, nil
	}
	purpose := strings.TrimSpace(input.Purpose)
	if purpose == "" {
		purpose = string(ImagePurposeContent)
	}
	kind := strings.TrimSpace(input.Kind)
	if kind == "" {
		kind = uploadAssetKindImage
	}
	storage := strings.TrimSpace(input.Storage)
	if storage == "" {
		storage = uploadAssetStorageUnknown
	}
	expiresAt := now.Add(s.tempRetention())
	objectKey := strings.TrimSpace(input.ObjectKey)
	if objectKey == "" && storage == uploadAssetStorageR2 {
		objectKey = r2ObjectKeyFromURL(s.cfg.Upload.Image.R2, urlValue)
	}
	row := domain.UploadAsset{
		UserID:       input.UserID,
		Purpose:      purpose,
		Kind:         kind,
		URL:          urlValue,
		Storage:      storage,
		ObjectKey:    objectKey,
		LocalPath:    strings.TrimSpace(input.LocalPath),
		OriginalName: strings.TrimSpace(input.OriginalName),
		Size:         input.Size,
		MimeType:     strings.TrimSpace(input.MimeType),
		Status:       UploadAssetStatusTemp,
		ExpiresAt:    &expiresAt,
		LastUsedAt:   &now,
		CreatedAt:    now,
		UpdatedAt:    &now,
	}
	err := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "url"}},
		DoUpdates: clause.Assignments(uploadAssetRecordConflictAssignments(row)),
	}).Create(&row).Error
	if err != nil {
		return nil, err
	}
	var saved domain.UploadAsset
	if err := s.db.WithContext(ctx).Where("url = ?", urlValue).First(&saved).Error; err != nil {
		return &row, nil
	}
	return &saved, nil
}

func uploadAssetRecordConflictAssignments(row domain.UploadAsset) map[string]any {
	return map[string]any{
		"user_id":       row.UserID,
		"purpose":       row.Purpose,
		"kind":          row.Kind,
		"storage":       row.Storage,
		"object_key":    row.ObjectKey,
		"local_path":    row.LocalPath,
		"original_name": row.OriginalName,
		"size":          row.Size,
		"mime_type":     row.MimeType,
		"status":        gorm.Expr(`CASE WHEN "upload_assets"."status" = ? THEN "upload_assets"."status" ELSE ? END`, UploadAssetStatusBound, UploadAssetStatusTemp),
		"expires_at":    gorm.Expr(`CASE WHEN "upload_assets"."status" = ? THEN "upload_assets"."expires_at" ELSE ? END`, UploadAssetStatusBound, row.ExpiresAt),
		"last_used_at":  row.LastUsedAt,
		"cleanup_error": "",
		"updated_at":    row.UpdatedAt,
	}
}

func (s *UploadAssetService) TouchAIAnalysis(ctx context.Context, userID int64, urls []string) error {
	if s == nil || s.db == nil || userID <= 0 {
		return nil
	}
	normalized := uniqueUploadAssetStrings(urls)
	if len(normalized) == 0 {
		return nil
	}
	now := time.Now()
	expiresAt := now.Add(s.tempRetention())
	return s.db.WithContext(ctx).Model(&domain.UploadAsset{}).
		Where("user_id = ? AND url IN ? AND status = ?", userID, normalized, UploadAssetStatusTemp).
		Updates(map[string]any{
			"purpose":       string(ImagePurposeAIAnalysis),
			"expires_at":    &expiresAt,
			"last_used_at":  &now,
			"cleanup_error": "",
			"updated_at":    &now,
		}).Error
}

func (s *UploadAssetService) CleanupExpired(ctx context.Context, limit int) error {
	if s == nil || s.db == nil {
		return nil
	}
	if limit <= 0 {
		limit = 200
	}
	var rows []domain.UploadAsset
	now := time.Now()
	if err := s.db.WithContext(ctx).
		Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?", UploadAssetStatusTemp, now).
		Order("expires_at ASC").
		Limit(limit).
		Find(&rows).Error; err != nil {
		return err
	}
	for _, asset := range rows {
		if err := s.cleanupExpiredAsset(ctx, asset, now); err != nil && s.logger != nil {
			s.logger.Warn("upload asset cleanup failed", zap.Int64("asset_id", asset.ID), zap.String("url", asset.URL), zap.Error(err))
		}
	}
	return nil
}

func (s *UploadAssetService) cleanupExpiredAsset(ctx context.Context, asset domain.UploadAsset, now time.Time) error {
	if postID, referenced, err := s.referencedPostID(ctx, asset.URL); err != nil {
		return err
	} else if referenced {
		return s.markBound(ctx, asset.ID, postID, now)
	}
	deleted, reason, err := s.deleteAssetObject(ctx, asset)
	status := UploadAssetStatusDeleted
	cleanupError := ""
	deletedAt := &now
	if err != nil {
		status = UploadAssetStatusDeleteFailed
		cleanupError = err.Error()
		deletedAt = nil
	} else if !deleted {
		status = UploadAssetStatusExpired
		cleanupError = reason
	}
	return s.db.WithContext(ctx).Model(&domain.UploadAsset{}).
		Where("id = ? AND status = ?", asset.ID, UploadAssetStatusTemp).
		Updates(map[string]any{
			"status":        status,
			"deleted_at":    deletedAt,
			"cleanup_error": cleanupError,
			"updated_at":    &now,
		}).Error
}

func (s *UploadAssetService) referencedPostID(ctx context.Context, rawURL string) (int64, bool, error) {
	urlValue := strings.TrimSpace(rawURL)
	if urlValue == "" {
		return 0, false, nil
	}
	var postID int64
	err := s.db.WithContext(ctx).Model(&domain.PostImage{}).
		Where("image_url = ?", urlValue).
		Select("post_id").
		Limit(1).
		Scan(&postID).Error
	if err != nil || postID > 0 {
		return postID, postID > 0, err
	}
	err = s.db.WithContext(ctx).Model(&domain.PostVideo{}).
		Where("video_url = ? OR cover_url = ? OR dash_url = ? OR preview_video_url = ?", urlValue, urlValue, urlValue, urlValue).
		Select("post_id").
		Limit(1).
		Scan(&postID).Error
	if err != nil || postID > 0 {
		return postID, postID > 0, err
	}
	err = s.db.WithContext(ctx).Model(&domain.PostAttachment{}).
		Where("attachment_url = ?", urlValue).
		Select("post_id").
		Limit(1).
		Scan(&postID).Error
	return postID, postID > 0, err
}

func (s *UploadAssetService) markBound(ctx context.Context, assetID, postID int64, now time.Time) error {
	return s.db.WithContext(ctx).Model(&domain.UploadAsset{}).
		Where("id = ?", assetID).
		Updates(map[string]any{
			"status":        UploadAssetStatusBound,
			"bound_post_id": postID,
			"expires_at":    nil,
			"last_used_at":  &now,
			"cleanup_error": "",
			"updated_at":    &now,
		}).Error
}

func (s *UploadAssetService) deleteAssetObject(ctx context.Context, asset domain.UploadAsset) (bool, string, error) {
	switch strings.TrimSpace(asset.Storage) {
	case uploadAssetStorageLocal:
		return s.deleteLocalAsset(asset)
	case uploadAssetStorageR2:
		key := strings.TrimSpace(asset.ObjectKey)
		if key == "" {
			key = r2ObjectKeyFromURL(s.cfg.Upload.Image.R2, asset.URL)
		}
		if key == "" {
			return false, "r2_object_key_missing", nil
		}
		return true, "", deleteR2Object(ctx, s.cfg.Upload.Image.R2, key)
	default:
		return false, "delete_api_unavailable", nil
	}
}

func (s *UploadAssetService) deleteLocalAsset(asset domain.UploadAsset) (bool, string, error) {
	path := strings.TrimSpace(asset.LocalPath)
	if path == "" {
		path = s.localPathFromURL(asset.URL)
	}
	if path == "" {
		return false, "local_path_missing", nil
	}
	path = filepath.Clean(path)
	if !s.safeUploadFilePath(path) {
		return false, "unsafe_upload_path", nil
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, "", nil
		}
		return true, "", err
	}
	return true, "", nil
}

func (s *UploadAssetService) localPathFromURL(raw string) string {
	pathValue := uploadAssetCanonicalFilePath(raw)
	if pathValue == "" {
		return ""
	}
	rest := strings.TrimPrefix(pathValue, "/api/file/")
	parts := strings.SplitN(rest, "/", 2)
	if len(parts) != 2 {
		return ""
	}
	for _, dir := range s.uploadTypeDirs(parts[0]) {
		candidate := filepath.Clean(filepath.Join(dir, filepath.FromSlash(parts[1])))
		rel, err := filepath.Rel(filepath.Clean(dir), candidate)
		if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return candidate
		}
	}
	return ""
}

func uploadAssetCanonicalFilePath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	var pathValue string
	if err == nil && parsed.Scheme != "" && parsed.Host != "" {
		pathValue = parsed.Path
	} else if err == nil && strings.HasPrefix(parsed.Path, "/") {
		pathValue = parsed.Path
	} else {
		pathValue = trimmed
		if strings.HasPrefix(pathValue, "api/file/") {
			pathValue = "/" + pathValue
		}
	}
	if !strings.HasPrefix(pathValue, "/api/file/") || strings.Contains(pathValue, "..") {
		return ""
	}
	return pathValue
}

func (s *UploadAssetService) safeUploadFilePath(path string) bool {
	for _, root := range s.uploadAssetRoots() {
		rel, err := filepath.Rel(root, path)
		if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func (s *UploadAssetService) uploadAssetRoots() []string {
	dirs := []string{
		s.cfg.Upload.Image.LocalUploadDir,
		s.cfg.Upload.Video.CoverDir,
		s.cfg.Upload.Attachment.LocalUploadDir,
		s.cfg.Upload.RootDir,
	}
	return uniqueCleanAbsPaths(dirs)
}

func (s *UploadAssetService) uploadTypeDirs(fileType string) []string {
	switch fileType {
	case "images":
		return uniqueCleanAbsPaths([]string{s.cfg.Upload.Image.LocalUploadDir, filepath.Join(s.cfg.Upload.RootDir, "images")})
	case "covers":
		return uniqueCleanAbsPaths([]string{s.cfg.Upload.Video.CoverDir, filepath.Join(s.cfg.Upload.RootDir, "covers")})
	case "attachments":
		return uniqueCleanAbsPaths([]string{s.cfg.Upload.Attachment.LocalUploadDir, filepath.Join(s.cfg.Upload.RootDir, "attachments")})
	default:
		return uniqueCleanAbsPaths([]string{filepath.Join(s.cfg.Upload.RootDir, fileType)})
	}
}

func (s *UploadAssetService) tempRetention() time.Duration {
	if s.cfg.Upload.Temp.Retention > 0 {
		return s.cfg.Upload.Temp.Retention
	}
	return 24 * time.Hour
}

func uniqueCleanAbsPaths(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		abs, err := filepath.Abs(value)
		if err != nil {
			continue
		}
		abs = filepath.Clean(abs)
		if !seen[abs] {
			seen[abs] = true
			out = append(out, abs)
		}
	}
	return out
}

func uniqueUploadAssetStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func r2ObjectKeyFromURL(cfg config.R2Config, raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}
	prefixes := []string{}
	if cfg.PublicURL != "" {
		prefixes = append(prefixes, strings.TrimRight(cfg.PublicURL, "/")+"/")
	}
	if cfg.AccountID != "" {
		prefixes = append(prefixes, "https://pub-"+cfg.AccountID+".r2.dev/")
	}
	for _, prefix := range prefixes {
		if after, ok := strings.CutPrefix(value, prefix); ok {
			key, err := url.PathUnescape(after)
			if err == nil {
				return strings.TrimLeft(key, "/")
			}
		}
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Path == "" {
		return ""
	}
	pathValue := strings.TrimLeft(parsed.Path, "/")
	if cfg.BucketName != "" {
		bucketPrefix := strings.Trim(cfg.BucketName, "/") + "/"
		pathValue = strings.TrimPrefix(pathValue, bucketPrefix)
	}
	key, err := url.PathUnescape(pathValue)
	if err != nil {
		return pathValue
	}
	return strings.TrimLeft(key, "/")
}

func deleteR2Object(ctx context.Context, cfg config.R2Config, key string) error {
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" || cfg.BucketName == "" || cfg.Endpoint == "" {
		return errors.New("r2 config is incomplete")
	}
	if cfg.Region == "" {
		cfg.Region = "auto"
	}
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	objectURL := endpoint + "/" + uploadAssetPathEscapeSegments(cfg.BucketName) + "/" + uploadAssetPathEscapeSegments(strings.TrimLeft(key, "/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, objectURL, bytes.NewReader(nil))
	if err != nil {
		return err
	}
	payloadHash := uploadAssetSHA256HexBytes(nil)
	now := time.Now().UTC()
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	req.Header.Set("X-Amz-Date", now.Format("20060102T150405Z"))
	signUploadAssetAWSV4(req, payloadHash, cfg.AccessKeyID, cfg.SecretAccessKey, cfg.Region, "s3", now)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("r2 delete failed: %s", resp.Status)
	}
	return nil
}

func signUploadAssetAWSV4(req *http.Request, payloadHash string, accessKey string, secretKey string, region string, service string, now time.Time) {
	date := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	host := req.URL.Host
	req.Header.Set("Host", host)

	signedHeaders := []string{"host", "x-amz-content-sha256", "x-amz-date"}
	headers := map[string]string{
		"host":                 host,
		"x-amz-content-sha256": payloadHash,
		"x-amz-date":           amzDate,
	}
	sort.Strings(signedHeaders)
	var canonicalHeaders strings.Builder
	for _, header := range signedHeaders {
		canonicalHeaders.WriteString(header)
		canonicalHeaders.WriteString(":")
		canonicalHeaders.WriteString(strings.TrimSpace(headers[header]))
		canonicalHeaders.WriteString("\n")
	}
	signedHeadersText := strings.Join(signedHeaders, ";")
	canonicalRequest := strings.Join([]string{
		req.Method,
		req.URL.EscapedPath(),
		req.URL.RawQuery,
		canonicalHeaders.String(),
		signedHeadersText,
		payloadHash,
	}, "\n")
	scope := date + "/" + region + "/" + service + "/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		uploadAssetSHA256HexString(canonicalRequest),
	}, "\n")
	signingKey := uploadAssetAWSV4SigningKey(secretKey, date, region, service)
	signature := hex.EncodeToString(uploadAssetHMACSHA256(signingKey, stringToSign))
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", accessKey, scope, signedHeadersText, signature))
}

func uploadAssetAWSV4SigningKey(secret, date, region, service string) []byte {
	kDate := uploadAssetHMACSHA256([]byte("AWS4"+secret), date)
	kRegion := uploadAssetHMACSHA256(kDate, region)
	kService := uploadAssetHMACSHA256(kRegion, service)
	return uploadAssetHMACSHA256(kService, "aws4_request")
}

func uploadAssetHMACSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(data))
	return mac.Sum(nil)
}

func uploadAssetSHA256HexBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func uploadAssetSHA256HexString(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

func uploadAssetPathEscapeSegments(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
