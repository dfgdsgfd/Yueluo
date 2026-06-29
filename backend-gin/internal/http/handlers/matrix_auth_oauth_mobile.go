package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

const (
	oauthMobileSessionPrefix = "oauth2:mobile-session:"
	oauthMobileSessionTTL    = 2 * time.Minute
	oauthMobileCallbackURL   = "xsewebfast://auth-return"
	oauthMobileCallbackHost  = "auth-return"
)

type oauthMobileSessionTicket struct {
	UserID    int64     `json:"user_id"`
	IsNewUser bool      `json:"is_new_user"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (h NativeHandlers) safeOAuthMobileCallbackURL(raw string) (string, bool) {
	candidate, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !validOAuthMobileCallbackURL(candidate) || !h.oauthMobileCallbackURLAllowed(candidate) {
		return "", false
	}
	candidate.Fragment = ""
	query := candidate.Query()
	removeOAuthMobileCallbackSensitiveParams(query)
	candidate.RawQuery = query.Encode()
	return candidate.String(), true
}

func (h NativeHandlers) oauthMobileCallbackURLAllowed(candidate *url.URL) bool {
	for _, raw := range h.oauthMobileAllowedCallbackURLs() {
		allowed, err := url.Parse(raw)
		if err != nil || !validOAuthMobileCallbackURL(allowed) {
			continue
		}
		if oauthMobileCallbackBaseMatches(candidate, allowed) {
			return true
		}
	}
	return false
}

func (h NativeHandlers) oauthMobileAllowedCallbackURLs() []string {
	var configured any
	if h.Settings != nil {
		configured = h.Settings.Get(services.OAuth2AppCallbackURLsSetting)
	}
	callbacks := normalizeOAuth2AppCallbackURLs(configured)
	if len(callbacks) == 0 {
		callbacks = normalizeOAuth2AppCallbackURLs(h.Config.OAuth2.AppCallbackURL)
	}
	if len(callbacks) == 0 {
		callbacks = []string{oauthMobileCallbackURL}
	}
	return callbacks
}

func normalizeOAuth2AppCallbackURLSetting(value any) (string, bool) {
	entries := oauth2AppCallbackURLStrings(value)
	if len(entries) == 0 {
		return "", false
	}
	normalized := make([]string, 0, len(entries))
	seen := map[string]bool{}
	for _, entry := range entries {
		text := strings.TrimSpace(entry)
		if text == "" {
			continue
		}
		callback, ok := normalizeOAuth2AppCallbackURL(text)
		if !ok {
			return "", false
		}
		key := oauthMobileCallbackBaseKey(callback)
		if seen[key] {
			continue
		}
		seen[key] = true
		normalized = append(normalized, callback)
	}
	if len(normalized) == 0 {
		return "", false
	}
	return strings.Join(normalized, "\n"), true
}

func normalizeOAuth2AppCallbackURLs(value any) []string {
	entries := oauth2AppCallbackURLStrings(value)
	normalized := make([]string, 0, len(entries))
	seen := map[string]bool{}
	for _, entry := range entries {
		callback, ok := normalizeOAuth2AppCallbackURL(entry)
		if !ok {
			continue
		}
		key := oauthMobileCallbackBaseKey(callback)
		if seen[key] {
			continue
		}
		seen[key] = true
		normalized = append(normalized, callback)
	}
	return normalized
}

func oauth2AppCallbackURLStrings(value any) []string {
	switch typed := value.(type) {
	case string:
		text := strings.TrimSpace(typed)
		if text == "" {
			return nil
		}
		if strings.HasPrefix(text, "[") {
			var list []string
			if json.Unmarshal([]byte(text), &list) == nil {
				return list
			}
			var anyList []any
			if json.Unmarshal([]byte(text), &anyList) == nil {
				return oauth2AppCallbackURLStrings(anyList)
			}
		}
		return strings.FieldsFunc(text, func(r rune) bool {
			return r == '\n' || r == '\r' || r == ','
		})
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if text := strings.TrimSpace(toString(item)); text != "" {
				out = append(out, text)
			}
		}
		return out
	default:
		text := strings.TrimSpace(toString(value))
		if text == "" {
			return nil
		}
		return oauth2AppCallbackURLStrings(text)
	}
}

func normalizeOAuth2AppCallbackURL(raw string) (string, bool) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !validOAuthMobileCallbackURL(parsed) {
		return "", false
	}
	parsed.Fragment = ""
	query := parsed.Query()
	removeOAuthMobileCallbackSensitiveParams(query)
	parsed.RawQuery = query.Encode()
	return parsed.String(), true
}

func validOAuthMobileCallbackURL(parsed *url.URL) bool {
	return parsed != nil && parsed.Scheme != "" && parsed.Host != "" && parsed.User == nil
}

func removeOAuthMobileCallbackSensitiveParams(query url.Values) {
	for _, name := range []string{"access_token", "refresh_token", "token", "code", "app_state", "ticket", "error"} {
		query.Del(name)
	}
}

func oauthMobileCallbackBaseMatches(candidate *url.URL, allowed *url.URL) bool {
	return strings.EqualFold(candidate.Scheme, allowed.Scheme) &&
		strings.EqualFold(candidate.Host, allowed.Host) &&
		oauthMobileCallbackPath(candidate) == oauthMobileCallbackPath(allowed)
}

func oauthMobileCallbackPath(parsed *url.URL) string {
	if parsed == nil || parsed.Path == "/" {
		return ""
	}
	return parsed.Path
}

func oauthMobileCallbackBaseKey(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return strings.TrimSpace(raw)
	}
	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host) + oauthMobileCallbackPath(parsed)
}

func (h NativeHandlers) safeOAuthMobileReturnURL(c *gin.Context, raw string) string {
	base, err := url.Parse(strings.TrimSpace(h.Config.Frontend.BaseURL))
	if err != nil || base.Scheme == "" || base.Host == "" {
		base, _ = url.Parse("https://xse.yuelk.com")
	}
	fallback := *base
	fallback.Path = "/explore"
	fallback.RawPath = ""
	fallback.RawQuery = ""
	fallback.Fragment = ""

	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback.String()
	}
	candidate, err := url.Parse(raw)
	if err != nil {
		return fallback.String()
	}
	if !candidate.IsAbs() {
		candidate = base.ResolveReference(candidate)
	}
	if !strings.EqualFold(candidate.Scheme, base.Scheme) || !strings.EqualFold(candidate.Host, base.Host) || candidate.User != nil {
		return fallback.String()
	}
	candidate.Fragment = ""
	return candidate.String()
}

func (h NativeHandlers) completeOAuthMobileCallback(c *gin.Context, entry oauthStateEntry, user domain.User, isNew bool) {
	ticket := base64URLRandom(32)
	payload := oauthMobileSessionTicket{
		UserID:    user.ID,
		IsNewUser: isNew,
		ExpiresAt: time.Now().UTC().Add(oauthMobileSessionTTL),
	}
	if !h.storeOAuthMobileSessionTicket(c.Request.Context(), ticket, payload) {
		h.recordSecurityAudit(c, "oauth_login", "mobile_handoff", "failure", "handoff_unavailable", http.StatusFound, &user.ID, "user", user.UserID, nil)
		c.Redirect(http.StatusFound, h.oauthMobileRedirectLocation(entry, "", "handoff_unavailable"))
		return
	}
	h.recordSecurityAudit(c, "oauth_login", "mobile_handoff", "success", "", http.StatusFound, &user.ID, "user", user.UserID, gin.H{"is_new_user": isNew})
	c.Redirect(http.StatusFound, h.oauthMobileRedirectLocation(entry, ticket, ""))
}

func (h NativeHandlers) oauthMobileRedirectLocation(entry oauthStateEntry, ticket string, errorCode string) string {
	callback, err := url.Parse(entry.AppCallbackURL)
	if err != nil || callback.Scheme == "" || callback.Host == "" {
		callback, _ = url.Parse(oauthMobileCallbackURL)
	}
	query := callback.Query()
	query.Set("url", entry.AppReturnURL)
	if ticket != "" {
		query.Set("ticket", ticket)
	}
	if errorCode != "" {
		query.Set("error", errorCode)
	}
	callback.RawQuery = query.Encode()
	return callback.String()
}

func (h NativeHandlers) storeOAuthMobileSessionTicket(ctx context.Context, ticket string, payload oauthMobileSessionTicket) bool {
	key := oauthMobileSessionPrefix + sha256Hex(ticket)
	if h.Redis != nil && h.Redis.Set(ctx, key, payload, oauthMobileSessionTTL) {
		return true
	}
	if h.Cache == nil {
		return false
	}
	h.Cache.Set(key, payload, oauthMobileSessionTTL)
	return true
}

func (h NativeHandlers) consumeOAuthMobileSessionTicket(ctx context.Context, ticket string) (oauthMobileSessionTicket, bool) {
	key := oauthMobileSessionPrefix + sha256Hex(ticket)
	var payload oauthMobileSessionTicket
	if h.Redis != nil && h.Redis.GetDelJSON(ctx, key, &payload) {
		return payload, payload.UserID > 0 && payload.ExpiresAt.After(time.Now().UTC())
	}
	if h.Cache == nil {
		return oauthMobileSessionTicket{}, false
	}
	value, ok := h.Cache.Take(key)
	if !ok {
		return oauthMobileSessionTicket{}, false
	}
	payload, ok = value.(oauthMobileSessionTicket)
	return payload, ok && payload.UserID > 0 && payload.ExpiresAt.After(time.Now().UTC())
}

func (h NativeHandlers) authOAuthMobileSession(c *gin.Context) {
	if !h.Config.OAuth2.Enabled {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.oauth2_disabled", nil)
		return
	}
	if !h.requireDB(c) {
		return
	}
	ticket := strings.TrimSpace(toString(readBodyMap(c)["ticket"]))
	if !validBase64URLValue(ticket, 32, 128) {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.oauth_mobile_ticket_invalid", nil)
		return
	}
	payload, ok := h.consumeOAuthMobileSessionTicket(c.Request.Context(), ticket)
	if !ok {
		h.recordSecurityAudit(c, "oauth_login", "mobile_session", "failure", "invalid_or_expired_ticket", http.StatusUnauthorized, nil, "user", "", nil)
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.oauth_mobile_ticket_invalid", nil)
		return
	}
	var user domain.User
	if err := h.DB.WithContext(c.Request.Context()).Where("id = ? AND is_active = ?", payload.UserID, true).First(&user).Error; writeDBError(c, err, "error.oauth_mobile_ticket_invalid") {
		return
	}
	access, refresh, ok := h.issueUserTokens(c, user.ID, user.UserID)
	if !ok {
		return
	}
	h.recordSecurityAudit(c, "oauth_login", "mobile_session", "success", "", http.StatusOK, &user.ID, "user", user.UserID, gin.H{"is_new_user": payload.IsNewUser})
	writeSuccess(c, matrixMsgOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"is_new_user":   payload.IsNewUser,
		"user":          h.userPublicMap(user),
	})
}
