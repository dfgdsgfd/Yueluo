package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
)

const (
	sessionKeyPrefix                        = "session:"
	sessionIDCounterKey                     = "session:id_counter"
	defaultAccessTokenTTL                   = time.Hour
	defaultRefreshSessionTTL                = 90 * 24 * time.Hour
	defaultRefreshSessionRenewalIntervalTTL = 24 * time.Hour
	defaultSessionIdleTTL                   = 7 * 24 * time.Hour
	sessionTouchInterval                    = time.Minute
	defaultUserSessionLimit                 = 5
	FileTokenTTL                            = 15 * time.Minute
)

const (
	RefreshTokenModeRedisOpaque = "redis_opaque"
	RefreshTokenModeJWTLegacy   = "jwt_legacy"
)

type AuthService struct {
	db       *gorm.DB
	redis    *RedisStore
	cfg      config.AuthConfig
	settings *SettingsService
}

type RequestUser struct {
	ID        int64
	UserID    string
	XiseID    *string
	Nickname  string
	Avatar    *string
	CreatedAt time.Time
	Username  string
	Type      string
	Token     string
}

type Session struct {
	ID           int64  `json:"id"`
	UserID       string `json:"user_id"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	UserAgent    string `json:"user_agent"`
	ClientIP     string `json:"client_ip"`
	Fingerprint  string `json:"fingerprint"`
	IsActive     bool   `json:"is_active"`
	ExpiresAt    string `json:"expires_at"`
	CreatedAt    string `json:"created_at"`
	LastActiveAt string `json:"last_active_at"`
}

type SessionPolicy struct {
	InactiveTTL            time.Duration
	UserActiveSessionLimit int
}

type TokenPolicy struct {
	AccessTokenTTL               time.Duration
	RefreshTokenActiveTTL        time.Duration
	RefreshTokenRenewalInterval  time.Duration
	RefreshTokenMode             string
	RefreshTokenAutoRenewEnabled bool
}

type AuthFailure struct {
	Status  int
	Code    int
	Message string
}

func NewAuthService(db *gorm.DB, redis *RedisStore, cfg config.AuthConfig, settings ...*SettingsService) *AuthService {
	service := &AuthService{db: db, redis: redis, cfg: cfg}
	if len(settings) > 0 {
		service.settings = settings[0]
	}
	return service
}

func ExtractBearerToken(header string) string {
	if after, ok := strings.CutPrefix(header, "Bearer "); ok {
		return strings.TrimSpace(after)
	}
	return ""
}

func (s *AuthService) Authenticate(ctx context.Context, token string) (*RequestUser, *AuthFailure) {
	if token == "" {
		return nil, &AuthFailure{Status: http.StatusUnauthorized, Code: response.CodeUnauthorized, Message: "访问令牌缺失"}
	}
	claims, err := s.verifyToken(token)
	if err != nil {
		return nil, &AuthFailure{Status: http.StatusUnauthorized, Code: response.CodeUnauthorized, Message: "无效的访问令牌"}
	}

	if value, _ := claims["type"].(string); value == "admin" {
		adminID, ok := numericClaim(claims["adminId"])
		if !ok {
			return nil, &AuthFailure{Status: http.StatusUnauthorized, Code: response.CodeUnauthorized, Message: "无效的访问令牌"}
		}
		if s.db == nil {
			return nil, &AuthFailure{Status: http.StatusUnauthorized, Code: response.CodeUnauthorized, Message: "管理员不存在"}
		}
		var admin domain.Admin
		err := s.db.WithContext(ctx).Where("id = ?", adminID).Select("id", "username").First(&admin).Error
		if err != nil {
			return nil, &AuthFailure{Status: http.StatusUnauthorized, Code: response.CodeUnauthorized, Message: "管理员不存在"}
		}
		return &RequestUser{ID: admin.ID, Username: admin.Username, Type: "admin", Token: token}, nil
	}

	userID, ok := numericClaim(claims["userId"])
	if !ok {
		return nil, &AuthFailure{Status: http.StatusUnauthorized, Code: response.CodeUnauthorized, Message: "无效的访问令牌"}
	}
	if s.db == nil {
		return nil, &AuthFailure{Status: http.StatusUnauthorized, Code: response.CodeUnauthorized, Message: "用户不存在或已被禁用"}
	}
	var user domain.User
	err = s.db.WithContext(ctx).
		Where("id = ? AND is_active = ?", userID, true).
		Select("id", "user_id", "xise_id", "nickname", "avatar", "is_active", "created_at").
		First(&user).Error
	if err != nil {
		return nil, &AuthFailure{Status: http.StatusUnauthorized, Code: response.CodeUnauthorized, Message: "用户不存在或已被禁用"}
	}
	if !s.FindActiveSession(ctx, token, userID) {
		return nil, &AuthFailure{Status: http.StatusUnauthorized, Code: response.CodeUnauthorized, Message: "会话已过期，请重新登录"}
	}
	return &RequestUser{
		ID:        user.ID,
		UserID:    user.UserID,
		XiseID:    user.XiseID,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		CreatedAt: user.CreatedAt,
		Type:      "user",
		Token:     token,
	}, nil
}

func (s *AuthService) Optional(ctx context.Context, token string) *RequestUser {
	if token == "" {
		return nil
	}
	user, failure := s.Authenticate(ctx, token)
	if failure != nil {
		return nil
	}
	return user
}

func (s *AuthService) FindActiveSession(ctx context.Context, token string, userID int64) bool {
	if s == nil || s.redis == nil {
		return false
	}
	var session Session
	if !s.redis.GetJSON(ctx, sessionTokenKey(token), &session) {
		return false
	}
	if !session.IsActive || session.UserID != strconv.FormatInt(userID, 10) {
		return false
	}
	now := time.Now()
	if sessionShouldRemove(session, now, s.SessionPolicy()) {
		s.DeleteSession(ctx, session)
		return false
	}
	s.TouchSession(ctx, session, now)
	return true
}

func (s *AuthService) GenerateAccessToken(payload map[string]any) (string, error) {
	return s.generateToken(payload, s.TokenPolicy().AccessTokenTTL)
}

func (s *AuthService) GenerateAdminAccessToken(payload map[string]any) (string, error) {
	return s.generateToken(payload, s.AdminAccessTokenTTL())
}

func (s *AuthService) AdminAccessTokenTTL() time.Duration {
	if s == nil {
		return defaultAccessTokenTTL
	}
	fallback := ParseAuthDuration(s.cfg.JWTExpiresIn, defaultAccessTokenTTL)
	return clampDuration(ParseAuthDuration(s.cfg.AdminJWTExpiresIn, fallback), time.Minute, 30*24*time.Hour)
}

func (s *AuthService) GenerateRefreshToken(payload map[string]any) (string, error) {
	policy := s.TokenPolicy()
	if policy.RefreshTokenMode == RefreshTokenModeJWTLegacy {
		refreshPayload := map[string]any{}
		maps.Copy(refreshPayload, payload)
		refreshPayload["token_type"] = "refresh"
		return s.generateToken(refreshPayload, policy.RefreshTokenActiveTTL)
	}
	return randomToken(32), nil
}

func (s *AuthService) GenerateFileToken(payload map[string]any) (string, error) {
	payload["purpose"] = "file_access"
	return s.generateToken(payload, FileTokenTTL)
}

func (s *AuthService) VerifyTokenClaims(token string) (jwt.MapClaims, bool) {
	claims, err := s.verifyToken(token)
	return claims, err == nil
}

func (s *AuthService) VerifyFileToken(token string) (jwt.MapClaims, bool) {
	claims, err := s.verifyToken(token)
	if err != nil {
		return nil, false
	}
	purpose, _ := claims["purpose"].(string)
	if purpose != "file_access" {
		return nil, false
	}
	return claims, true
}

func (s *AuthService) SessionPolicy() SessionPolicy {
	if s == nil {
		return ReadSessionPolicy(nil)
	}
	return ReadSessionPolicy(s.settings)
}

func (s *AuthService) TokenPolicy() TokenPolicy {
	if s == nil {
		return ReadTokenPolicy(nil, config.AuthConfig{})
	}
	return ReadTokenPolicy(s.settings, s.cfg)
}

func (s *AuthService) verifyToken(token string) (jwt.MapClaims, error) {
	parsed, err := jwt.ParseWithClaims(token, jwt.MapClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	return claims, nil
}

func (s *AuthService) generateToken(payload map[string]any, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{}
	maps.Copy(claims, payload)
	now := time.Now()
	claims["iat"] = now.Unix()
	claims["exp"] = now.Add(ttl).Unix()
	claims["jti"] = randomID()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func numericClaim(value any) (int64, bool) {
	switch typed := value.(type) {
	case float64:
		if math.Trunc(typed) == typed {
			return int64(typed), true
		}
	case int64:
		return typed, true
	case int:
		return int64(typed), true
	case jsonNumber:
		parsed, err := strconv.ParseInt(string(typed), 10, 64)
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(typed), 10, 64)
		return parsed, err == nil
	}
	return 0, false
}

type jsonNumber string

func ParseAuthDuration(value string, fallback time.Duration) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	unit := value[len(value)-1]
	number := value[:len(value)-1]
	amount, err := strconv.Atoi(number)
	if err != nil {
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds) * time.Second
		}
		return fallback
	}
	switch unit {
	case 'd', 'D':
		return time.Duration(amount) * 24 * time.Hour
	case 'h', 'H':
		return time.Duration(amount) * time.Hour
	case 'm', 'M':
		return time.Duration(amount) * time.Minute
	case 's', 'S':
		return time.Duration(amount) * time.Second
	default:
		return fallback
	}
}

func parseJWTDuration(value string, fallback time.Duration) time.Duration {
	return ParseAuthDuration(value, fallback)
}

func ReadTokenPolicy(settings *SettingsService, cfg config.AuthConfig) TokenPolicy {
	policy := TokenPolicy{
		AccessTokenTTL:               ParseAuthDuration(cfg.JWTExpiresIn, defaultAccessTokenTTL),
		RefreshTokenActiveTTL:        ParseAuthDuration(cfg.RefreshTokenExpiresIn, defaultRefreshSessionTTL),
		RefreshTokenRenewalInterval:  defaultRefreshSessionRenewalIntervalTTL,
		RefreshTokenMode:             RefreshTokenModeRedisOpaque,
		RefreshTokenAutoRenewEnabled: true,
	}
	if settings != nil {
		if value, exists := settings.ExplicitValue("access_token_ttl_seconds"); exists {
			policy.AccessTokenTTL = time.Duration(settingInt(value, int(policy.AccessTokenTTL.Seconds()))) * time.Second
		}
		if value, exists := settings.ExplicitValue("refresh_token_active_ttl_seconds"); exists {
			policy.RefreshTokenActiveTTL = time.Duration(settingInt(value, int(policy.RefreshTokenActiveTTL.Seconds()))) * time.Second
		}
		if value, exists := settings.ExplicitValue("refresh_token_renewal_interval_seconds"); exists {
			policy.RefreshTokenRenewalInterval = time.Duration(settingInt(value, int(policy.RefreshTokenRenewalInterval.Seconds()))) * time.Second
		}
		if value, exists := settings.Value("refresh_token_mode"); exists {
			policy.RefreshTokenMode = settingString(value, policy.RefreshTokenMode)
		}
		if value, exists := settings.Value("refresh_token_auto_renew_enabled"); exists {
			policy.RefreshTokenAutoRenewEnabled = settingBool(value)
		}
	}
	policy.AccessTokenTTL = clampDuration(policy.AccessTokenTTL, time.Minute, 24*time.Hour)
	policy.RefreshTokenActiveTTL = clampDuration(policy.RefreshTokenActiveTTL, time.Hour, 365*24*time.Hour)
	policy.RefreshTokenRenewalInterval = clampDuration(policy.RefreshTokenRenewalInterval, time.Hour, 30*24*time.Hour)
	policy.RefreshTokenMode = normalizeRefreshTokenMode(policy.RefreshTokenMode)
	return policy
}

func ReadSessionPolicy(settings *SettingsService) SessionPolicy {
	inactiveTTL := defaultSessionIdleTTL
	limit := defaultUserSessionLimit
	if settings != nil {
		if _, exists := settings.ExplicitValue("session_inactive_ttl_seconds"); exists {
			inactiveTTL = time.Duration(settings.Int("session_inactive_ttl_seconds", int(inactiveTTL.Seconds()))) * time.Second
		} else {
			inactiveDays := settings.Int("redis_session_inactive_days", int(defaultSessionIdleTTL/(24*time.Hour)))
			inactiveTTL = time.Duration(inactiveDays) * 24 * time.Hour
		}
		limit = settings.Int("redis_user_active_session_limit", limit)
	}
	inactiveTTL = clampDuration(inactiveTTL, time.Hour, 365*24*time.Hour)
	limit = clampSessionInt(limit, 1, 100)
	return SessionPolicy{
		InactiveTTL:            inactiveTTL,
		UserActiveSessionLimit: limit,
	}
}

func clampSessionInt(value int, low int, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func clampDuration(value time.Duration, low time.Duration, high time.Duration) time.Duration {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func normalizeRefreshTokenMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case RefreshTokenModeJWTLegacy:
		return RefreshTokenModeJWTLegacy
	case RefreshTokenModeRedisOpaque:
		return RefreshTokenModeRedisOpaque
	default:
		return RefreshTokenModeRedisOpaque
	}
}

func randomID() string {
	return randomToken(16)
}

func randomToken(size int) string {
	if size <= 0 {
		size = 16
	}
	buf := make([]byte, 16)
	if size != 16 {
		buf = make([]byte, size)
	}
	if _, err := rand.Read(buf); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(buf)
}
