package handlers

import (
	"errors"
	"io"
	"mime"
	"net/http"
	"os"
	pathpkg "path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

var allowedFileTypes = map[string]string{
	"images":      "note",
	"avatar":      "open",
	"banner":      "open",
	"covers":      "note",
	"attachments": "open",
	"plsc":        "note",
	"videos":      "video",
	"thumbnails":  "video",
	"apk":         "open",
	"bfiu":        "open",
	"static":      "open",
}

var fileMIMETypes = map[string]string{
	".jpg":          "image/jpeg",
	".jpeg":         "image/jpeg",
	".png":          "image/png",
	".gif":          "image/gif",
	".webp":         "image/webp",
	".bmp":          "image/bmp",
	".svg":          "image/svg+xml",
	".mp4":          "video/mp4",
	".avi":          "video/x-msvideo",
	".mov":          "video/quicktime",
	".wmv":          "video/x-ms-wmv",
	".flv":          "video/x-flv",
	".webm":         "video/webm",
	".mkv":          "video/x-matroska",
	".pdf":          "application/pdf",
	".zip":          "application/zip",
	".rar":          "application/x-rar-compressed",
	".7z":           "application/x-7z-compressed",
	".gz":           "application/gzip",
	".tar":          "application/x-tar",
	".doc":          "application/msword",
	".docx":         "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":          "application/vnd.ms-excel",
	".xlsx":         "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".ppt":          "application/vnd.ms-powerpoint",
	".pptx":         "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".txt":          "text/plain",
	".apk":          "application/vnd.android.package-archive",
	".apks":         "application/octet-stream",
	".mobileconfig": "application/x-apple-aspen-config",
	".mpd":          "application/dash+xml",
	".m4s":          "video/iso.segment",
}

type UploadPathResolver struct {
	roots         []string
	typeDirs      map[string][]string
	candidateDirs map[string][]string
}

func NewUploadPathResolver(cfg config.Config) *UploadPathResolver {
	roots := configuredUploadRoots(cfg)
	typeDirs := make(map[string][]string, len(allowedFileTypes)+1)
	candidateDirs := make(map[string][]string, len(allowedFileTypes)+1)
	fileTypes := make([]string, 0, len(allowedFileTypes)+1)
	for fileType := range allowedFileTypes {
		fileTypes = append(fileTypes, fileType)
	}
	fileTypes = append(fileTypes, "media")
	for _, fileType := range fileTypes {
		dirs := configuredUploadTypeDirs(cfg, fileType)
		typeDirs[fileType] = dirs
		candidates := append([]string{}, dirs...)
		for _, root := range roots {
			candidates = append(candidates, filepath.Join(root, fileType))
		}
		candidateDirs[fileType] = uniqueStrings(candidates)
	}
	return &UploadPathResolver{
		roots:         roots,
		typeDirs:      typeDirs,
		candidateDirs: candidateDirs,
	}
}

const (
	msgFileInvalidName = "\u65e0\u6548\u7684\u6587\u4ef6\u540d"
	msgFileInvalidPath = "\u65e0\u6548\u7684\u6587\u4ef6\u8def\u5f84"
	msgFileForbidden   = "\u7981\u6b62\u8bbf\u95ee"
	msgFileNotFound    = "\u6587\u4ef6\u4e0d\u5b58\u5728"
	msgFileNoResource  = "\u8d44\u6e90\u4e0d\u5b58\u5728"
	msgFileReadFailed  = "\u6587\u4ef6\u8bfb\u53d6\u5931\u8d25"

	uploadPathCachePrefix = "file:path:"
	uploadPathCacheTTL    = 5 * time.Second
)

func (h NativeHandlers) FileAccess(c *gin.Context) {
	requestPath := strings.TrimPrefix(c.Param("filepath"), "/")
	if requestPath == "" {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgFileNoResource, nil)
		return
	}
	parts := strings.SplitN(requestPath, "/", 2)
	if len(parts) != 2 {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgFileNoResource, nil)
		return
	}
	if parts[0] == "public" {
		if strings.Contains(parts[1], "/") {
			response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgFileInvalidName, nil)
			return
		}
		h.servePublicFile(c, parts[1])
		return
	}
	h.serveProtectedFile(c, parts[0], parts[1])
}

func (h NativeHandlers) PublicFile(c *gin.Context) {
	h.servePublicFile(c, c.Param("filename"))
}

func (h NativeHandlers) servePublicFile(c *gin.Context, filename string) {
	if invalidSingleFilename(filename) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgFileInvalidName, nil)
		return
	}
	h.serveLocalFile(c, "media", filename, true)
}

func (h NativeHandlers) ProtectedFile(c *gin.Context) {
	fileType := c.Param("type")
	subPath := strings.TrimPrefix(c.Param("filepath"), "/")
	h.serveProtectedFile(c, fileType, subPath)
}

func (h NativeHandlers) serveProtectedFile(c *gin.Context, fileType string, subPath string) {
	if _, ok := allowedFileTypes[fileType]; !ok {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgFileNoResource, nil)
		return
	}
	if invalidSubPath(subPath) {
		status := http.StatusBadRequest
		message := msgFileInvalidPath
		if !strings.Contains(subPath, "/") {
			message = msgFileInvalidName
		}
		response.JSON(c, status, response.CodeValidationError, message, nil)
		return
	}
	if !h.authenticateFileAccess(c, fileType, subPath) {
		return
	}
	h.serveLocalFile(c, fileType, subPath, fileType == "attachments")
}

func (h NativeHandlers) authenticateFileAccess(c *gin.Context, fileType string, subPath string) bool {
	category := allowedFileTypes[fileType]
	if category == "open" {
		return true
	}
	if category == "note" && (h.Settings == nil || !h.Settings.IsNoteGuestRestricted()) {
		return true
	}
	if category == "video" && (h.Settings == nil || !h.Settings.IsVideoGuestRestricted()) {
		return true
	}

	canonicalPath, hasCanonicalPath := h.canonicalFileURL(fileType, subPath)
	if hasCanonicalPath {
		now := time.Now()
		if h.verifyFileURLSignature(canonicalPath, c.Query(fileSignatureExpiryParam), c.Query(fileSignatureParam), now) {
			return true
		}
		if h.verifyDASHSegmentURLSignature(canonicalPath, c.Query(fileSignatureExpiryParam), c.Query(fileSignatureParam), now) {
			return true
		}
	}

	if h.Auth != nil {
		if cookieToken, err := c.Cookie("file_token"); err == nil && cookieToken != "" {
			if _, ok := h.Auth.VerifyFileToken(cookieToken); ok {
				return true
			}
		}
		if bearer := services.ExtractBearerToken(c.GetHeader("Authorization")); bearer != "" {
			if _, ok := h.Auth.VerifyTokenClaims(bearer); ok {
				return true
			}
		}
	}

	if hasCanonicalPath {
		if h.fileBelongsToPublicAccessExemptPost(c.Request.Context(), canonicalPath) {
			return true
		}
	}

	apiKey := c.GetHeader("X-API-Key")
	if apiKey != "" {
		if h.DB == nil {
			c.Header("Cache-Control", "no-store")
			response.JSON(c, http.StatusInternalServerError, response.CodeError, "\u8ba4\u8bc1\u5904\u7406\u5931\u8d25", nil)
			return false
		}
		if h.apiKeyRequestBlocked(c, openAPIKeyScope) {
			return false
		}
		openAPI, err := h.lookupOpenAPIKey(c.Request.Context(), apiKey)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.Header("Cache-Control", "no-store")
			h.rejectInvalidAPIKey(c, openAPIKeyScope, "API\u5bc6\u94a5\u65e0\u6548\u6216\u5df2\u7981\u7528")
			return false
		}
		if err != nil {
			c.Header("Cache-Control", "no-store")
			response.JSON(c, http.StatusInternalServerError, response.CodeError, "\u8ba4\u8bc1\u5904\u7406\u5931\u8d25", nil)
			return false
		}
		h.acceptAPIKey(c, openAPIKeyScope)
		h.touchOpenAPIKey(openAPI.ID)
		return true
	}

	c.Header("Cache-Control", "no-store")
	response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "\u8bbf\u95ee\u4ee4\u724c\u7f3a\u5931", nil)
	return false
}

func (h NativeHandlers) serveLocalFile(c *gin.Context, fileType, subPath string, public bool) {
	file, stat, filePath, ok := h.openUploadFile(fileType, subPath)
	if !ok {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgFileNotFound, nil)
		return
	}
	defer file.Close()

	if !public && fileType == "videos" && strings.EqualFold(filepath.Ext(filePath), ".mpd") {
		if canonicalPath, ok := h.canonicalFileURL(fileType, subPath); ok {
			h.serveDASHManifest(c, file, stat, canonicalPath)
			return
		}
	}

	etag := `"` + strconv.FormatInt(stat.Size(), 16) + "-" + strconv.FormatInt(stat.ModTime().UnixMilli(), 16) + `"`
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}

	contentType := contentTypeForPath(filePath)
	c.Header("Content-Type", contentType)
	c.Header("ETag", etag)
	c.Header("Accept-Ranges", "bytes")
	if public {
		c.Header("Cache-Control", "public, max-age=31536000")
	} else {
		c.Header("Cache-Control", "private, no-cache")
	}

	http.ServeContent(c.Writer, c.Request, filepath.Base(filePath), stat.ModTime(), file)
}

func (h NativeHandlers) serveDASHManifest(c *gin.Context, file *os.File, stat os.FileInfo, canonicalPath string) {
	data, err := io.ReadAll(file)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgFileReadFailed, nil)
		return
	}
	body, rewritten := h.signDASHManifestReferences(data, canonicalPath, time.Now())
	if !rewritten {
		body = data
	}
	c.Header("Content-Type", fileMIMETypes[".mpd"])
	c.Header("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
	c.Header("Cache-Control", "private, no-cache")
	c.Header("Content-Length", strconv.Itoa(len(body)))
	c.Status(http.StatusOK)
	_, _ = c.Writer.Write(body)
}

func (h NativeHandlers) verifyDASHSegmentURLSignature(canonicalPath, expiryText, signature string, now time.Time) bool {
	if !strings.HasPrefix(canonicalPath, "/api/file/videos/") || !strings.EqualFold(pathpkg.Ext(canonicalPath), ".m4s") {
		return false
	}
	manifestPath := pathpkg.Join(pathpkg.Dir(canonicalPath), "manifest.mpd")
	return h.verifyFileURLSignature(manifestPath, expiryText, signature, now)
}

func (h NativeHandlers) findUploadFile(fileType, subPath string) (string, bool) {
	diskSubPath := profileImageDiskSubPath(fileType, subPath)
	if cachedPath, ok := h.cachedUploadFilePath(fileType, subPath); ok {
		if stat, err := os.Stat(cachedPath); err == nil && stat.Mode().IsRegular() {
			return cachedPath, true
		}
		h.deleteCachedUploadFilePath(fileType, subPath)
	}
	for _, dir := range h.uploadCandidateDirs(fileType) {
		candidate, ok := safeJoinRelative(dir, diskSubPath)
		if !ok {
			continue
		}
		if stat, err := os.Stat(candidate); err == nil && stat.Mode().IsRegular() {
			h.cacheUploadFilePath(fileType, subPath, candidate)
			return candidate, true
		}
	}
	return "", false
}

func (h NativeHandlers) openUploadFile(fileType, subPath string) (*os.File, os.FileInfo, string, bool) {
	diskSubPath := profileImageDiskSubPath(fileType, subPath)
	if cachedPath, ok := h.cachedUploadFilePath(fileType, subPath); ok {
		file, stat, ok := openResolvedUploadFile(cachedPath)
		if ok {
			return file, stat, cachedPath, true
		}
		h.deleteCachedUploadFilePath(fileType, subPath)
	}
	for _, dir := range h.uploadCandidateDirs(fileType) {
		candidate, ok := safeJoinRelative(dir, diskSubPath)
		if !ok {
			continue
		}
		file, stat, ok := openResolvedUploadFile(candidate)
		if ok {
			h.cacheUploadFilePath(fileType, subPath, candidate)
			return file, stat, candidate, true
		}
	}
	return nil, nil, "", false
}

func openResolvedUploadFile(path string) (*os.File, os.FileInfo, bool) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, false
	}
	stat, err := file.Stat()
	if err == nil && stat.Mode().IsRegular() {
		return file, stat, true
	}
	_ = file.Close()
	return nil, nil, false
}

func (h NativeHandlers) cachedUploadFilePath(fileType, subPath string) (string, bool) {
	if h.Cache == nil {
		return "", false
	}
	value, ok := h.Cache.Get(uploadPathCacheKey(fileType, subPath))
	if !ok {
		return "", false
	}
	path, ok := value.(string)
	return path, ok && path != ""
}

func (h NativeHandlers) cacheUploadFilePath(fileType, subPath, filePath string) {
	if h.Cache != nil && filePath != "" {
		h.Cache.Set(uploadPathCacheKey(fileType, subPath), filePath, uploadPathCacheTTL)
	}
}

func (h NativeHandlers) deleteCachedUploadFilePath(fileType, subPath string) {
	if h.Cache != nil {
		h.Cache.Delete(uploadPathCacheKey(fileType, subPath))
	}
}

func uploadPathCacheKey(fileType, subPath string) string {
	return uploadPathCachePrefix + fileType + ":" + subPath
}

func (h NativeHandlers) uploadCandidateDirs(fileType string) []string {
	if h.UploadPaths != nil {
		return h.UploadPaths.candidateDirs[fileType]
	}
	dirs := append([]string{}, h.uploadTypeDirs(fileType)...)
	for _, root := range h.uploadRoots() {
		dirs = append(dirs, filepath.Join(root, fileType))
	}
	return uniqueStrings(dirs)
}

func (h NativeHandlers) uploadTypeDirs(fileType string) []string {
	if h.UploadPaths != nil {
		return h.UploadPaths.typeDirs[fileType]
	}
	return configuredUploadTypeDirs(h.Config, fileType)
}

func configuredUploadTypeDirs(cfg config.Config, fileType string) []string {
	var dirs []string
	switch fileType {
	case "images":
		dirs = append(dirs, cfg.Upload.Image.LocalUploadDir)
	case "avatar":
		dirs = append(dirs, cfg.Upload.AvatarDir)
	case "banner":
		dirs = append(dirs, cfg.Upload.BannerDir)
	case "videos":
		dirs = append(dirs, cfg.Upload.Video.LocalUploadDir, cfg.Upload.Video.LegacyUploadDir)
	case "covers":
		dirs = append(dirs, cfg.Upload.Video.CoverDir)
	case "attachments":
		dirs = append(dirs, cfg.Upload.Attachment.LocalUploadDir)
	case "media":
		dirs = append(dirs, "uploads/media")
	case "apk":
		dirs = append(dirs, "uploads/apk")
	case "plsc":
		dirs = append(dirs, "uploads/plsc")
	case "thumbnails":
		dirs = append(dirs, "uploads/thumbnails")
	case "bfiu":
		dirs = append(dirs, "uploads/bfiu")
	case "static":
		dirs = append(dirs, "static")
	}
	return absPaths(dirs)
}

func (h NativeHandlers) uploadRoots() []string {
	if h.UploadPaths != nil {
		return h.UploadPaths.roots
	}
	return configuredUploadRoots(h.Config)
}

func configuredUploadRoots(cfg config.Config) []string {
	configured := strings.TrimSpace(cfg.Upload.RootDir)
	if configured == "" {
		configured = "uploads"
	}
	out := []string{absPath(configured)}
	if configured == "uploads" || configured == "."+string(filepath.Separator)+"uploads" {
		if cwd, err := os.Getwd(); err == nil {
			out = append(out,
				filepath.Clean(filepath.Join(cwd, "..", "uploads")),
				filepath.Clean(filepath.Join(cwd, "..", "backend", "uploads")),
				filepath.Clean(filepath.Join(cwd, "backend", "uploads")),
			)
		}
	}
	return uniqueStrings(out)
}

func safeJoinRelative(root, rest string) (string, bool) {
	root = filepath.Clean(root)
	if rest == "" {
		return "", false
	}
	candidate := filepath.Clean(filepath.Join(root, filepath.FromSlash(rest)))
	rel, err := filepath.Rel(root, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return candidate, true
}

func invalidSingleFilename(filename string) bool {
	return filename == "" || strings.Contains(filename, "/") || strings.Contains(filename, "\\") || strings.Contains(filename, "..")
}

func invalidSubPath(path string) bool {
	if path == "" || strings.Contains(path, "\\") {
		return true
	}
	segments := strings.SplitSeq(path, "/")
	for segment := range segments {
		if segment == "" || segment == ".." {
			return true
		}
	}
	return false
}

func contentTypeForPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if value := fileMIMETypes[ext]; value != "" {
		return value
	}
	if value := mime.TypeByExtension(ext); value != "" {
		return value
	}
	return "application/octet-stream"
}

func absPath(path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}

func absPaths(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		out = append(out, absPath(value))
	}
	return uniqueStrings(out)
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
