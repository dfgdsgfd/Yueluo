package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

const (
	matrixMsgOK               = "success"
	matrixMsgInternal         = "服务器内部错误"
	matrixMsgMissingParams    = "缺少必要参数"
	matrixMsgInvalidToken     = "无效的访问令牌"
	matrixMsgPermissionDenied = "权限不足"
)

func matrixPath(c *gin.Context) string {
	value, _ := c.Get("express_path")
	if text, ok := value.(string); ok {
		return text
	}
	return c.Request.URL.Path
}

func matrixMethod(c *gin.Context) string {
	value, _ := c.Get("express_method")
	if text, ok := value.(string); ok && text != "" {
		return strings.ToUpper(text)
	}
	return strings.ToUpper(c.Request.Method)
}

func matrixParam(c *gin.Context, name string) string {
	pattern := matrixPath(c)
	if pattern == "" {
		return c.Param(name)
	}
	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	requestParts := strings.Split(strings.Trim(c.Request.URL.Path, "/"), "/")
	for i, part := range patternParts {
		if i >= len(requestParts) {
			return ""
		}
		if strings.HasPrefix(part, ":") && strings.TrimPrefix(part, ":") == name {
			return requestParts[i]
		}
		if strings.HasSuffix(part, "/*") || part == "*" || strings.HasPrefix(part, "*") {
			return strings.Join(requestParts[i:], "/")
		}
	}
	return c.Param(name)
}

func matrixTail(c *gin.Context, prefix string) string {
	path := strings.TrimPrefix(c.Request.URL.Path, prefix)
	return strings.TrimPrefix(path, "/")
}

func (h NativeHandlers) requireMatrixAuth(c *gin.Context) (*services.RequestUser, bool) {
	if user, ok := currentUser(c); ok {
		return user, true
	}
	if h.Auth == nil {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, matrixMsgInvalidToken, nil)
		return nil, false
	}
	user, failure := h.Auth.Authenticate(c.Request.Context(), h.authAccessTokenFromRequest(c))
	if failure != nil {
		response.JSON(c, failure.Status, failure.Code, failure.Message, nil)
		return nil, false
	}
	c.Set("user", user)
	return user, true
}

func (h NativeHandlers) requireMatrixAdmin(c *gin.Context) (*services.RequestUser, bool) {
	user, ok := h.requireMatrixAuth(c)
	if !ok {
		return nil, false
	}
	if user.Type != "admin" {
		response.JSON(c, http.StatusForbidden, response.CodeForbidden, matrixMsgPermissionDenied, nil)
		return nil, false
	}
	return user, true
}

func (h NativeHandlers) optionalMatrixAuth(c *gin.Context) *services.RequestUser {
	if user, ok := currentUser(c); ok {
		return user
	}
	if h.Auth == nil {
		return nil
	}
	user := h.Auth.Optional(c.Request.Context(), h.authAccessTokenFromRequest(c))
	if user != nil {
		c.Set("user", user)
	}
	return user
}

func (h NativeHandlers) enforceMatrixAuth(c *gin.Context, authClass string) bool {
	switch authClass {
	case "", "public", "optional", "optional-note-guest-restricted", "optional-video-guest-restricted", "file-access", "proxy-api-key":
		return true
	case "user":
		_, ok := h.requireMatrixAuth(c)
		return ok
	case "admin":
		_, ok := h.requireMatrixAdmin(c)
		return ok
	case "open-api-key":
		return h.requireMatrixOpenAPIKey(c)
	default:
		response.JSON(c, http.StatusForbidden, response.CodeForbidden, matrixMsgPermissionDenied, nil)
		return false
	}
}

func (h NativeHandlers) requireMatrixOpenAPIKey(c *gin.Context) bool {
	handler := h.RequireOpenAPIKey()
	handler(c)
	return !c.IsAborted()
}

func (h NativeHandlers) requireDB(c *gin.Context) bool {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return false
	}
	return true
}

func readBodyMap(c *gin.Context) map[string]any {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil || body == nil {
		return map[string]any{}
	}
	return body
}

func writeSuccess(c *gin.Context, message string, data any) {
	if message == "" {
		message = matrixMsgOK
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": message, "data": data})
}

func writeCreated(c *gin.Context, message string, data any) {
	if message == "" {
		message = matrixMsgOK
	}
	c.JSON(http.StatusCreated, gin.H{"code": response.CodeSuccess, "message": message, "data": data})
}

func writeSimpleSuccess(c *gin.Context, message string) {
	if message == "" {
		message = matrixMsgOK
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": message})
}

func writeDBError(c *gin.Context, err error, notFoundMessage string) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if notFoundMessage == "" {
			notFoundMessage = "资源不存在"
		}
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, notFoundMessage, nil)
		return true
	}
	response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
	return true
}

func pageLimit(c *gin.Context, fallbackLimit int) (int, int, int) {
	if fallbackLimit <= 0 {
		fallbackLimit = 20
	}
	page := positiveIntQuery(c, "page", 1)
	limit := positiveIntQuery(c, "limit", fallbackLimit)
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = fallbackLimit
	}
	if limit > 200 {
		limit = 200
	}
	return page, limit, (page - 1) * limit
}

func matrixPagination(page, limit int, total int64) gin.H {
	pages := int64(0)
	if limit > 0 {
		pages = int64(math.Ceil(float64(total) / float64(limit)))
	}
	return gin.H{"page": page, "limit": limit, "total": total, "pages": pages}
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func randomHex(n int) string {
	if n <= 0 {
		n = 16
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(buf)
}

var defaultClientIPHeaders = []string{"X-Forwarded-For", "X-Real-IP", "CF-Connecting-IP"}

func (h NativeHandlers) clientIP(c *gin.Context) string {
	return clientIPFromHeaders(c, h.Config.Server.ClientIPHeaders)
}

func clientIP(c *gin.Context) string {
	return clientIPFromHeaders(c, nil)
}

func clientIPFromHeaders(c *gin.Context, headers []string) string {
	if c == nil || c.Request == nil {
		return ""
	}
	for _, header := range normalizedClientIPHeaders(headers) {
		if ip := clientIPFromHeaderValue(header, c.GetHeader(header)); ip != "" {
			return ip
		}
	}
	if host, _, err := net.SplitHostPort(c.Request.RemoteAddr); err == nil {
		if ip := normalizeIPCandidate(host); ip != "" {
			return ip
		}
	}
	return c.ClientIP()
}

func normalizedClientIPHeaders(headers []string) []string {
	if len(headers) == 0 {
		return defaultClientIPHeaders
	}
	out := make([]string, 0, len(headers))
	for _, header := range headers {
		header = strings.TrimSpace(header)
		if header != "" {
			out = append(out, header)
		}
	}
	if len(out) == 0 {
		return defaultClientIPHeaders
	}
	return out
}

func clientIPFromHeaderValue(header string, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.EqualFold(header, "Forwarded") {
		return clientIPFromForwardedHeader(value)
	}
	for part := range strings.SplitSeq(value, ",") {
		if ip := normalizeIPCandidate(part); ip != "" {
			return ip
		}
	}
	return ""
}

func clientIPFromForwardedHeader(value string) string {
	for group := range strings.SplitSeq(value, ",") {
		for part := range strings.SplitSeq(group, ";") {
			key, raw, ok := strings.Cut(strings.TrimSpace(part), "=")
			if !ok || !strings.EqualFold(strings.TrimSpace(key), "for") {
				continue
			}
			if ip := normalizeIPCandidate(raw); ip != "" {
				return ip
			}
		}
	}
	return ""
}

func normalizeIPCandidate(value string) string {
	value = strings.TrimSpace(strings.Trim(value, `"`))
	if value == "" || strings.EqualFold(value, "unknown") {
		return ""
	}
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		host = strings.Trim(host, "[]")
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
	}
	value = strings.Trim(value, "[]")
	if ip := net.ParseIP(value); ip != nil {
		return ip.String()
	}
	return ""
}

func stringPtr(v string) *string {
	return &v
}

func jsonBytes(value any) datatypes.JSON {
	if value == nil {
		return nil
	}
	if raw, ok := value.(json.RawMessage); ok {
		return datatypes.JSON(raw)
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	return datatypes.JSON(data)
}

func jsonValue(data []byte) any {
	if len(data) == 0 {
		return nil
	}
	var out any
	if json.Unmarshal(data, &out) != nil {
		return nil
	}
	return out
}

func jsonValueAny(value any) any {
	switch typed := value.(type) {
	case nil:
		return nil
	case datatypes.JSON:
		if parsed := jsonValue([]byte(typed)); parsed != nil {
			return parsed
		}
	case []byte:
		if parsed := jsonValue(typed); parsed != nil {
			return parsed
		}
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		if parsed := jsonValue([]byte(text)); parsed != nil {
			return parsed
		}
	}
	return value
}

func (h NativeHandlers) userPublicMap(user domain.User) gin.H {
	return gin.H{
		"id":                int(user.ID),
		"user_id":           user.UserID,
		"xise_id":           user.XiseID,
		"nickname":          user.Nickname,
		"email":             user.Email,
		"avatar":            h.signFileURLPtr(user.Avatar),
		"background":        h.signFileURLPtr(user.Background),
		"bio":               user.Bio,
		"bio_audit_status":  user.BioAuditStatus,
		"location":          user.Location,
		"follow_count":      user.FollowCount,
		"fans_count":        user.FansCount,
		"like_count":        user.LikeCount,
		"is_active":         user.IsActive,
		"created_at":        user.CreatedAt,
		"gender":            user.Gender,
		"zodiac_sign":       user.ZodiacSign,
		"mbti":              user.MBTI,
		"education":         user.Education,
		"major":             user.Major,
		"interests":         jsonValue(user.Interests),
		"birthday":          user.Birthday,
		"custom_fields":     jsonValue(user.CustomFields),
		"profile_completed": user.ProfileCompleted,
		"privacy_birthday":  user.PrivacyBirthday,
		"privacy_age":       user.PrivacyAge,
		"privacy_zodiac":    user.PrivacyZodiac,
		"privacy_mbti":      user.PrivacyMBTI,
		"verified":          user.Verified,
		"verified_name":     user.VerifiedName,
	}
}

func parseTimeAny(value any) *time.Time {
	text, _ := value.(string)
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02", "2006-01-02 15:04:05"} {
		if parsed, err := time.Parse(layout, text); err == nil {
			return &parsed
		}
	}
	return nil
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case fmt.Stringer:
		return typed.String()
	case float64:
		if math.Trunc(typed) == typed {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case nil:
		return ""
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
}

func parseStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := toString(item); text != "" {
				out = append(out, text)
			}
		}
		return out
	case datatypes.JSON:
		var out []string
		if json.Unmarshal([]byte(typed), &out) == nil {
			return out
		}
	case []byte:
		var out []string
		if json.Unmarshal(typed, &out) == nil {
			return out
		}
	case string:
		if strings.TrimSpace(typed) == "" {
			return []string{}
		}
		var out []string
		if json.Unmarshal([]byte(typed), &out) == nil {
			return out
		}
		return []string{typed}
	default:
		return []string{}
	}
	return []string{}
}
