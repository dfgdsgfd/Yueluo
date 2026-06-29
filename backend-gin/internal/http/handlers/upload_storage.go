package handlers

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/repositories"
	"yuem-go/backend-gin/internal/services"
)

type storedImage struct {
	URL                   string
	UploadAssetID         int64
	UploadAssetExpiresAt  *time.Time
	UploadAssetPurpose    string
	Storage               string
	LocalPath             string
	ObjectKey             string
	Processed             services.ProcessedImage
	WatermarkTraceToken   string
	WatermarkPayloadBytes int
	WatermarkEngine       string
}

func (h NativeHandlers) storeImage(c *gin.Context, purpose services.ImagePurpose, data []byte, filename string, contentType string) (storedImage, string, bool) {
	processor := h.imageProcessor()
	var user *services.RequestUser
	if c != nil {
		user, _ = currentUser(c)
	}
	traceToken := ""
	tracePayloadToken := ""
	if h.DB != nil && imagePurposeUsesHiddenWatermark(purpose) &&
		(h.Settings == nil || h.Settings.Bool("hidden_watermark_enabled")) &&
		(h.Settings == nil || !h.Settings.Bool("hidden_watermark_protected_only")) {
		sum := sha256.Sum256(data)
		traceInput := repositories.ImageWatermarkTraceInput{
			TraceType:      domain.ImageWatermarkTraceUpload,
			ShortCodeBytes: domain.ImageWatermarkShortCodeBytes,
			SourceHash:     hex.EncodeToString(sum[:]),
			FieldFlags:     imageWatermarkFieldFlags(h.Settings),
		}
		if h.Settings != nil && traceInput.FieldFlags&domain.ImageWatermarkFieldCustom != 0 {
			traceInput.CustomText = strings.TrimSpace(toString(h.Settings.Get("hidden_watermark_custom_text")))
		}
		if user != nil {
			traceInput.UserID = user.ID
			traceInput.UserDisplayID = user.UserID
			traceInput.Username = firstNonEmptyHandler(user.Nickname, user.Username)
		}
		trace, err := repositories.ReserveImageWatermarkTrace(c.Request.Context(), h.DB, traceInput)
		if err != nil {
			return storedImage{}, err.Error(), false
		}
		traceToken = trace.Token
		tracePayloadToken = imageWatermarkTraceShortCode(trace)
	}
	processed, err := processor.Process(c.Request.Context(), services.ProcessImageInput{
		Data:                  data,
		Filename:              filename,
		ContentType:           contentType,
		Purpose:               purpose,
		User:                  user,
		WatermarkTraceToken:   traceToken,
		WatermarkPayloadToken: tracePayloadToken,
	})
	if err != nil {
		repositories.DeleteImageWatermarkTrace(c.Request.Context(), h.DB, traceToken)
		return storedImage{}, err.Error(), false
	}
	if traceToken != "" && !processed.WatermarkApplied {
		repositories.DeleteImageWatermarkTrace(c.Request.Context(), h.DB, traceToken)
		traceToken = ""
	}
	var url string
	var localPath string
	var errMsg string
	var ok bool
	storage := h.Config.Upload.Image.Strategy
	switch h.Config.Upload.Image.Strategy {
	case "local":
		url, localPath, errMsg, ok = h.storeLocalFile(processed.Data, h.Config.Upload.Image.LocalUploadDir, "images", processed.Filename)
	case "imagehost":
		url, errMsg, ok = h.uploadToImageHost(processed.Data, processed.Filename, processed.ContentType)
	case "r2":
		url, errMsg, ok = h.uploadToR2(processed.Data, processed.Filename, processed.ContentType, "images", h.Config.Upload.Image.R2)
	default:
		repositories.DeleteImageWatermarkTrace(c.Request.Context(), h.DB, traceToken)
		return storedImage{}, msgUploadUnsupported, false
	}
	if !ok {
		repositories.DeleteImageWatermarkTrace(c.Request.Context(), h.DB, traceToken)
		return storedImage{}, errMsg, false
	}
	if traceToken != "" {
		now := time.Now()
		if err := h.DB.WithContext(c.Request.Context()).Model(&domain.ImageWatermarkTrace{}).Where("token = ?", traceToken).Updates(map[string]any{
			"source_url":       url,
			"engine":           processed.WatermarkEngine,
			"watermark_width":  processed.Width,
			"watermark_height": processed.Height,
			"updated_at":       now,
		}).Error; err != nil {
			repositories.DeleteImageWatermarkTrace(c.Request.Context(), h.DB, traceToken)
			return storedImage{}, err.Error(), false
		}
	}
	stored := storedImage{
		URL:                   url,
		Storage:               storage,
		LocalPath:             localPath,
		Processed:             processed,
		WatermarkTraceToken:   traceToken,
		WatermarkPayloadBytes: processed.WatermarkPayloadBytes,
		WatermarkEngine:       processed.WatermarkEngine,
	}
	if h.UploadAssets != nil && uploadAssetPurposeIsTemporary(purpose) {
		asset, err := h.UploadAssets.Record(c.Request.Context(), services.UploadAssetRecordInput{
			UserID:       requestUserID(user),
			Purpose:      string(purpose),
			Kind:         "image",
			URL:          url,
			Storage:      storage,
			LocalPath:    localPath,
			OriginalName: filename,
			Size:         int64(len(data)),
			MimeType:     processed.ContentType,
		})
		if err != nil {
			return storedImage{}, err.Error(), false
		}
		if asset != nil {
			stored.UploadAssetID = asset.ID
			stored.UploadAssetExpiresAt = asset.ExpiresAt
			stored.UploadAssetPurpose = asset.Purpose
			stored.ObjectKey = asset.ObjectKey
		}
	}
	return stored, "", true
}

func (h NativeHandlers) imageUploadResponse(stored storedImage, originalName string, originalSize int64) gin.H {
	out := gin.H{
		"originalname":          originalName,
		"size":                  originalSize,
		"url":                   stored.URL,
		"signedUrl":             h.signFileURL(stored.URL),
		"format":                stored.Processed.Format,
		"contentType":           stored.Processed.ContentType,
		"processedSize":         stored.Processed.ProcessedSize,
		"width":                 stored.Processed.Width,
		"height":                stored.Processed.Height,
		"watermarkApplied":      stored.Processed.WatermarkApplied,
		"watermarkTraceToken":   stored.WatermarkTraceToken,
		"watermarkPayloadBytes": stored.WatermarkPayloadBytes,
		"watermarkEngine":       stored.WatermarkEngine,
	}
	if stored.UploadAssetID > 0 {
		out["uploadAssetId"] = stored.UploadAssetID
		out["uploadAssetPurpose"] = stored.UploadAssetPurpose
		out["uploadAssetExpiresAt"] = stored.UploadAssetExpiresAt
	}
	return out
}

func requestUserID(user *services.RequestUser) int64 {
	if user == nil {
		return 0
	}
	return user.ID
}

func imagePurposeUsesHiddenWatermark(purpose services.ImagePurpose) bool {
	return purpose != services.ImagePurposeAvatar && purpose != services.ImagePurposeBackground
}

func uploadAssetPurposeIsTemporary(purpose services.ImagePurpose) bool {
	switch purpose {
	case services.ImagePurposeContent, services.ImagePurposeCover, services.ImagePurposeAIAnalysis:
		return true
	default:
		return false
	}
}

func imageWatermarkFieldFlags(settings *services.SettingsService) int {
	flags := 0
	if settings == nil || settings.Bool("hidden_watermark_include_uid") {
		flags |= domain.ImageWatermarkFieldUID
	}
	if settings == nil || settings.Bool("hidden_watermark_include_user_id") {
		flags |= domain.ImageWatermarkFieldUserID
	}
	if settings != nil && settings.Bool("hidden_watermark_include_username") {
		flags |= domain.ImageWatermarkFieldUsername
	}
	if settings == nil || settings.Bool("hidden_watermark_include_time") {
		flags |= domain.ImageWatermarkFieldTime
	}
	if settings == nil || settings.Bool("hidden_watermark_include_file_hash") {
		flags |= domain.ImageWatermarkFieldSourceHash
	}
	if settings != nil && settings.Bool("hidden_watermark_include_custom") {
		flags |= domain.ImageWatermarkFieldCustom
	}
	return flags
}

func (h NativeHandlers) storeVideo(data []byte, filename string, contentType string) (string, string, string, bool) {
	switch h.Config.Upload.Video.Strategy {
	case "local":
		return h.storeLocalFile(data, h.Config.Upload.Video.LocalUploadDir, "videos", filename)
	case "r2":
		url, errMsg, ok := h.uploadToR2(data, filename, contentType, "videos", h.Config.Upload.Video.R2)
		return url, "", errMsg, ok
	default:
		return "", "", msgUploadUnsupported, false
	}
}

func (h NativeHandlers) storeLocalFile(data []byte, uploadDir, urlType, filename string) (string, string, string, bool) {
	return h.storeLocalFileAs(data, uploadDir, urlType, uniqueUploadFilename(data, filename))
}

func (h NativeHandlers) storeLocalFilePreservingName(data []byte, uploadDir, urlType, filename string) (string, string, string, bool) {
	if invalidSingleFilename(filename) {
		return "", "", msgFileInvalidName, false
	}
	return h.storeLocalFileAs(data, uploadDir, urlType, filename)
}

func (h NativeHandlers) storeLocalFileAs(data []byte, uploadDir, urlType, storedFilename string) (string, string, string, bool) {
	workDir, err := h.localUploadWorkDir()
	if err != nil {
		return "", "", err.Error(), false
	}
	defer os.RemoveAll(workDir)
	stagedPath := filepath.Join(workDir, "payload")
	if err := os.WriteFile(stagedPath, data, 0600); err != nil {
		return "", "", err.Error(), false
	}
	dir := absPath(uploadDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", err.Error(), false
	}
	path := filepath.Join(dir, storedFilename)
	partPath := path + ".part-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := copyFileContents(stagedPath, partPath); err != nil {
		_ = os.Remove(partPath)
		return "", "", err.Error(), false
	}
	if err := os.Rename(partPath, path); err != nil {
		_ = os.Remove(partPath)
		return "", "", err.Error(), false
	}
	return "/api/file/" + urlType + "/" + pathEscapeSegments(storedFilename), path, "", true
}

func (h NativeHandlers) localUploadWorkDir() (string, error) {
	if h.TempStorage != nil {
		return h.TempStorage.NewWorkDir("local-upload")
	}
	root := absPath(h.Config.Upload.Temp.RootDir)
	if err := os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	return os.MkdirTemp(root, "local-upload-")
}

func (h NativeHandlers) uploadToR2(data []byte, filename string, contentType string, prefix string, cfg config.R2Config) (string, string, bool) {
	if cfg.AccessKeyID == "" || cfg.SecretAccessKey == "" || cfg.BucketName == "" || cfg.Endpoint == "" {
		return "", "R2 config is incomplete", false
	}
	if cfg.Region == "" {
		cfg.Region = "auto"
	}
	key := strings.Trim(prefix, "/") + "/" + uniqueUploadFilename(data, filename)
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	objectURL := endpoint + "/" + pathEscapeSegments(cfg.BucketName) + "/" + pathEscapeSegments(key)
	req, err := http.NewRequest(http.MethodPut, objectURL, bytes.NewReader(data))
	if err != nil {
		return "", err.Error(), false
	}
	if contentType == "" {
		contentType = contentTypeForPath(filename)
	}
	payloadHash := sha256HexBytes(data)
	now := time.Now().UTC()
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	req.Header.Set("X-Amz-Date", now.Format("20060102T150405Z"))
	signAWSV4(req, payloadHash, cfg.AccessKeyID, cfg.SecretAccessKey, cfg.Region, "s3", now)
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", err.Error(), false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = resp.Status
		}
		return "", msg, false
	}
	return r2PublicURL(cfg, key), "", true
}

func signAWSV4(req *http.Request, payloadHash string, accessKey string, secretKey string, region string, service string, now time.Time) {
	date := now.Format("20060102")
	amzDate := now.Format("20060102T150405Z")
	host := req.URL.Host
	req.Header.Set("Host", host)

	signedHeaders := []string{"content-type", "host", "x-amz-content-sha256", "x-amz-date"}
	headers := map[string]string{
		"content-type":         req.Header.Get("Content-Type"),
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
		sha256HexString(canonicalRequest),
	}, "\n")
	signingKey := awsV4SigningKey(secretKey, date, region, service)
	signature := hex.EncodeToString(hmacSHA256(signingKey, stringToSign))
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s", accessKey, scope, signedHeadersText, signature))
}

func awsV4SigningKey(secret, date, region, service string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), date)
	kRegion := hmacSHA256(kDate, region)
	kService := hmacSHA256(kRegion, service)
	return hmacSHA256(kService, "aws4_request")
}

func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(data))
	return mac.Sum(nil)
}

func sha256HexBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sha256HexString(data string) string {
	sum := sha256.Sum256([]byte(data))
	return hex.EncodeToString(sum[:])
}

func pathEscapeSegments(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

func r2PublicURL(cfg config.R2Config, key string) string {
	if cfg.PublicURL != "" {
		return strings.TrimRight(cfg.PublicURL, "/") + "/" + key
	}
	return "https://pub-" + cfg.AccountID + ".r2.dev/" + key
}

func (h NativeHandlers) uploadToImageHost(data []byte, filename string, contentType string) (string, string, bool) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", err.Error(), false
	}
	if _, err := part.Write(data); err != nil {
		return "", err.Error(), false
	}
	if err := writer.Close(); err != nil {
		return "", err.Error(), false
	}
	client := &http.Client{Timeout: time.Duration(h.Config.Upload.Image.ImageHostTimeoutSeconds) * time.Second}
	req, err := http.NewRequest(http.MethodPost, h.Config.Upload.Image.ImageHostAPIURL, body)
	if err != nil {
		return "", err.Error(), false
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Content-Length", strconv.Itoa(body.Len()))
	resp, err := client.Do(req)
	if err != nil {
		return "", err.Error(), false
	}
	defer resp.Body.Close()
	var parsed struct {
		Errno int `json:"errno"`
		Data  struct {
			URL string `json:"url"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", msgUploadImagehostFailed, false
	}
	if parsed.Errno == 0 && strings.TrimSpace(parsed.Data.URL) != "" {
		return strings.TrimSpace(parsed.Data.URL), "", true
	}
	return "", msgUploadImagehostFailed, false
}
