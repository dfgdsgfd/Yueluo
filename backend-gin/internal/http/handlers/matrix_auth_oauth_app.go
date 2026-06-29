package handlers

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
)

const (
	oauthAppClient      = "android"
	oauthAppHandoffTTL  = 10 * time.Minute
	oauthAppCallbackURL = "xsewebfast://oauth/callback"
)

var errOAuthAppTicketInvalid = errors.New("oauth app ticket is invalid")

func validOAuthAppState(value string) bool {
	return validBase64URLValue(value, 32, 128)
}

func validPKCEChallenge(value string) bool {
	return validBase64URLValue(value, 43, 128)
}

func validPKCEVerifier(value string) bool {
	return validBase64URLValue(value, 43, 128)
}

func validBase64URLValue(value string, minimum int, maximum int) bool {
	if len(value) < minimum || len(value) > maximum {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_' {
			continue
		}
		return false
	}
	return true
}

func (h NativeHandlers) completeOAuthAppCallback(c *gin.Context, entry oauthStateEntry, user domain.User, isNew bool) {
	if h.DB == nil {
		h.recordSecurityAudit(c, "oauth_login", "app_handoff", "failure", "database_unavailable", http.StatusFound, &user.ID, "user", user.UserID, nil)
		c.Redirect(http.StatusFound, h.oauthAppErrorRedirectLocation(entry.AppState, "handoff_unavailable"))
		return
	}

	now := time.Now().UTC()
	code := base64URLRandom(32)
	handoff := domain.OAuthAppHandoff{
		CodeHash:      sha256Hex(code),
		AppStateHash:  sha256Hex(entry.AppState),
		CodeChallenge: entry.AppCodeChallenge,
		UserID:        user.ID,
		IsNewUser:     isNew,
		ExpiresAt:     now.Add(oauthAppHandoffTTL),
		CreatedAt:     now,
	}
	if err := h.DB.WithContext(c.Request.Context()).Create(&handoff).Error; err != nil {
		h.recordSecurityAudit(c, "oauth_login", "app_handoff", "failure", "handoff_create_failed", http.StatusFound, &user.ID, "user", user.UserID, nil)
		c.Redirect(http.StatusFound, h.oauthAppErrorRedirectLocation(entry.AppState, "handoff_failed"))
		return
	}
	// Cleanup is deliberately best effort and only removes already-expired rows.
	_ = h.DB.WithContext(c.Request.Context()).Where("expires_at < ?", now.Add(-time.Hour)).Delete(&domain.OAuthAppHandoff{}).Error

	h.recordSecurityAudit(c, "oauth_login", "app_handoff", "success", "", http.StatusFound, &user.ID, "user", user.UserID, gin.H{"is_new_user": isNew})
	c.Redirect(http.StatusFound, h.oauthAppSuccessRedirectLocation(code, entry.AppState, isNew))
}

func (h NativeHandlers) oauthAppSuccessRedirectLocation(code string, appState string, isNew bool) string {
	return h.oauthAppRedirectLocation(url.Values{
		"code":        {code},
		"app_state":   {appState},
		"is_new_user": {strconvBool(isNew)},
	})
}

func (h NativeHandlers) oauthAppErrorRedirectLocation(appState string, errorCode string) string {
	return h.oauthAppRedirectLocation(url.Values{
		"app_state": {appState},
		"error":     {errorCode},
	})
}

func (h NativeHandlers) oauthAppRedirectLocation(values url.Values) string {
	rawCallback := strings.TrimSpace(h.Config.OAuth2.AppCallbackURL)
	if callback, err := url.Parse(rawCallback); rawCallback == "" || err != nil || callback.Host == oauthMobileCallbackHost {
		rawCallback = oauthAppCallbackURL
	}
	callback, err := url.Parse(rawCallback)
	if err != nil || callback.Scheme == "" || callback.Host == "" {
		callback, _ = url.Parse(oauthAppCallbackURL)
	}
	query := callback.Query()
	for key, entries := range values {
		for _, value := range entries {
			query.Add(key, value)
		}
	}
	callback.RawQuery = query.Encode()
	return callback.String()
}

func (h NativeHandlers) authOAuthAppToken(c *gin.Context) {
	if !h.Config.OAuth2.Enabled {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.oauth2_disabled", nil)
		return
	}
	if !h.requireDB(c) {
		return
	}
	body := readBodyMap(c)
	code := strings.TrimSpace(toString(body["code"]))
	appState := strings.TrimSpace(toString(body["app_state"]))
	verifier := strings.TrimSpace(toString(body["code_verifier"]))
	if !validBase64URLValue(code, 32, 128) || !validOAuthAppState(appState) || !validPKCEVerifier(verifier) {
		h.recordSecurityAudit(c, "oauth_login", "app_token", "failure", "invalid_request", http.StatusBadRequest, nil, "user", "", nil)
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "error.oauth_app_invalid_request", nil)
		return
	}

	handoff, user, err := h.consumeOAuthAppHandoff(c, code, appState, verifier)
	if err != nil {
		h.recordSecurityAudit(c, "oauth_login", "app_token", "failure", "invalid_or_expired_ticket", http.StatusUnauthorized, nil, "user", "", nil)
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, "error.oauth_app_ticket_invalid", nil)
		return
	}
	access, refresh, ok := h.issueUserTokens(c, user.ID, user.UserID)
	if !ok {
		return
	}
	h.recordSecurityAudit(c, "oauth_login", "app_token", "success", "", http.StatusOK, &user.ID, "user", user.UserID, gin.H{"is_new_user": handoff.IsNewUser})
	writeSuccess(c, matrixMsgOK, gin.H{
		"access_token":  access,
		"refresh_token": refresh,
		"is_new_user":   handoff.IsNewUser,
		"user":          h.userPublicMap(user),
	})
}

func (h NativeHandlers) consumeOAuthAppHandoff(c *gin.Context, code string, appState string, verifier string) (domain.OAuthAppHandoff, domain.User, error) {
	var handoff domain.OAuthAppHandoff
	var user domain.User
	err := h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("code_hash = ? AND app_state_hash = ?", sha256Hex(code), sha256Hex(appState)).
			First(&handoff).Error
		if err != nil {
			return errOAuthAppTicketInvalid
		}
		now := time.Now().UTC()
		if handoff.ConsumedAt != nil || !handoff.ExpiresAt.After(now) {
			return errOAuthAppTicketInvalid
		}
		challenge := base64URLSHA256(verifier)
		if subtle.ConstantTimeCompare([]byte(challenge), []byte(handoff.CodeChallenge)) != 1 {
			return errOAuthAppTicketInvalid
		}
		result := tx.Model(&domain.OAuthAppHandoff{}).
			Where("id = ? AND consumed_at IS NULL AND expires_at > ?", handoff.ID, now).
			Update("consumed_at", now)
		if result.Error != nil || result.RowsAffected != 1 {
			return errOAuthAppTicketInvalid
		}
		handoff.ConsumedAt = &now
		if err := tx.Where("id = ? AND is_active = ?", handoff.UserID, true).First(&user).Error; err != nil {
			return errOAuthAppTicketInvalid
		}
		return nil
	})
	if err != nil {
		return domain.OAuthAppHandoff{}, domain.User{}, err
	}
	return handoff, user, nil
}
