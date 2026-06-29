package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"

	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/services"
)

const (
	adminSessionKeyPrefix    = "session:"
	adminSessionIDCounterKey = "session:id_counter"
	adminAllSessionsKey      = "all_sessions"
	adminSessionDefaultTTL   = 7 * 24 * time.Hour
	adminCreatedSessionTTL   = 30 * 24 * time.Hour
)

type adminSessionUser struct {
	ID       int64  `gorm:"column:id"`
	UserID   string `gorm:"column:user_id"`
	Nickname string `gorm:"column:nickname"`
}

func (h NativeHandlers) adminSessions(c *gin.Context) {
	id := matrixParam(c, "id")
	switch matrixMethod(c) {
	case http.MethodGet:
		if id != "" {
			h.adminSessionDetail(c, id)
			return
		}
		h.adminSessionList(c)
	case http.MethodPost:
		h.adminSessionCreate(c)
	case http.MethodPut:
		h.adminSessionUpdate(c, id)
	case http.MethodDelete:
		if id != "" {
			h.adminSessionDelete(c, id)
			return
		}
		h.adminSessionBulkDelete(c)
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "admin route not found", nil)
	}
}

func (h NativeHandlers) adminSessionList(c *gin.Context) {
	client, ok := h.adminRedis(c)
	if !ok {
		return
	}
	page, limit, offset := pageLimit(c, 20)
	ctx := c.Request.Context()

	sessionIDs, err := client.ZRange(ctx, adminAllSessionsKey, 0, -1).Result()
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	if len(sessionIDs) == 0 {
		rows := make([]gin.H, 0)
		writeSuccess(c, matrixMsgOK, gin.H{"data": rows, "pagination": matrixPagination(page, limit, 0)})
		return
	}

	keys := make([]string, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		keys = append(keys, adminSessionIDKey(id))
	}
	values, err := client.MGet(ctx, keys...).Result()
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}

	sessions := make([]services.Session, 0, len(values))
	staleIDs := make([]string, 0)
	for i, value := range values {
		session, parsed := adminParseRedisSession(value)
		if parsed {
			sessions = append(sessions, session)
			continue
		}
		if i < len(sessionIDs) {
			staleIDs = append(staleIDs, sessionIDs[i])
		}
	}
	if len(staleIDs) > 0 {
		_ = client.ZRem(ctx, adminAllSessionsKey, stringSliceToAny(staleIDs)...).Err()
	}

	if rawActive := strings.TrimSpace(c.Query("is_active")); rawActive != "" {
		active := rawActive == "true" || rawActive == "1"
		filtered := sessions[:0]
		for _, session := range sessions {
			if session.IsActive == active {
				filtered = append(filtered, session)
			}
		}
		sessions = filtered
	}

	users := h.adminSessionUsers(sessions)
	if userDisplayID := strings.TrimSpace(c.Query("user_display_id")); userDisplayID != "" {
		filtered := sessions[:0]
		for _, session := range sessions {
			if user, exists := users[session.UserID]; exists && strings.Contains(user.UserID, userDisplayID) {
				filtered = append(filtered, session)
			}
		}
		sessions = filtered
	}

	adminSortSessions(sessions, c.DefaultQuery("sortField", "created_at"), c.DefaultQuery("sortOrder", "desc"))
	total := int64(len(sessions))
	end := offset + limit
	if offset > len(sessions) {
		offset = len(sessions)
	}
	if end > len(sessions) {
		end = len(sessions)
	}

	rows := make([]gin.H, 0, end-offset)
	for _, session := range sessions[offset:end] {
		rows = append(rows, adminSessionListRow(session, users[session.UserID]))
	}
	writeSuccess(c, matrixMsgOK, gin.H{"data": rows, "pagination": matrixPagination(page, limit, total)})
}

func (h NativeHandlers) adminSessionDetail(c *gin.Context, id string) {
	session, ok := h.adminFindSession(c, id)
	if !ok {
		return
	}
	users := h.adminSessionUsers([]services.Session{session})
	writeSuccess(c, matrixMsgOK, adminSessionDetailRow(session, users[session.UserID]))
}

func (h NativeHandlers) adminSessionCreate(c *gin.Context) {
	client, ok := h.adminRedis(c)
	if !ok {
		return
	}
	body := readBodyMap(c)
	userID, exists := int64FromAny(body["user_id"])
	if !exists || userID <= 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "缺少必填字段", nil)
		return
	}

	now := time.Now().UTC()
	expiresAt := now.Add(adminCreatedSessionTTL)
	id, err := client.Incr(c.Request.Context(), adminSessionIDCounterKey).Result()
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}

	userAgent := toString(body["user_agent"])
	clientIP := toString(body["client_ip"])
	session := services.Session{
		ID:           id,
		UserID:       strconv.FormatInt(userID, 10),
		Token:        randomHex(32),
		RefreshToken: randomHex(32),
		UserAgent:    userAgent,
		ClientIP:     clientIP,
		Fingerprint:  sha256Hex(userAgent + "|" + clientIP),
		IsActive:     true,
		ExpiresAt:    expiresAt.Format(time.RFC3339Nano),
		CreatedAt:    now.Format(time.RFC3339Nano),
		LastActiveAt: now.Format(time.RFC3339Nano),
	}
	if rawActive, present := body["is_active"]; present {
		session.IsActive = jsTruthy(rawActive)
	}
	if err := h.adminSaveSession(c, session); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	writeSuccess(c, "会话创建成功", gin.H{"id": id})
}

func (h NativeHandlers) adminSessionUpdate(c *gin.Context, id string) {
	if id == "" {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "会话不存在", nil)
		return
	}
	session, ok := h.adminFindSession(c, id)
	if !ok {
		return
	}
	body := readBodyMap(c)
	if raw, present := body["user_agent"]; present {
		session.UserAgent = toString(raw)
		session.Fingerprint = sha256Hex(session.UserAgent + "|" + session.ClientIP)
	}
	if raw, present := body["is_active"]; present {
		session.IsActive = jsTruthy(raw)
	}
	if err := h.adminSaveSession(c, session); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	writeSimpleSuccess(c, "更新成功")
}

func (h NativeHandlers) adminSessionDelete(c *gin.Context, id string) {
	client, ok := h.adminRedis(c)
	if !ok {
		return
	}
	if err := h.adminDeleteSessionByID(c, client, id); err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	writeSimpleSuccess(c, "删除成功")
}

func (h NativeHandlers) adminSessionBulkDelete(c *gin.Context) {
	client, ok := h.adminRedis(c)
	if !ok {
		return
	}
	ids := parseStringSlice(readBodyMap(c)["ids"])
	if len(ids) == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "请提供要删除的ID列表", nil)
		return
	}
	for _, id := range ids {
		if err := h.adminDeleteSessionByID(c, client, id); err != nil {
			response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
			return
		}
	}
	writeSimpleSuccess(c, "成功删除 "+strconv.Itoa(len(ids))+" 条记录")
}

func (h NativeHandlers) adminRedis(c *gin.Context) (*redis.Client, bool) {
	if h.Redis == nil || h.Redis.Client() == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return nil, false
	}
	return h.Redis.Client(), true
}

func (h NativeHandlers) adminFindSession(c *gin.Context, id string) (services.Session, bool) {
	if strings.TrimSpace(id) == "" {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "会话不存在", nil)
		return services.Session{}, false
	}
	var session services.Session
	if h.Redis == nil || !h.Redis.GetJSON(c.Request.Context(), adminSessionIDKey(id), &session) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "会话不存在", nil)
		return services.Session{}, false
	}
	return session, true
}

func (h NativeHandlers) adminSaveSession(c *gin.Context, session services.Session) error {
	client, ok := h.adminRedis(c)
	if !ok {
		return redis.Nil
	}
	if strings.TrimSpace(session.LastActiveAt) == "" {
		session.LastActiveAt = session.CreatedAt
	}
	ttl := adminSessionTTL(session)
	serialized, err := json.Marshal(session)
	if err != nil {
		return err
	}
	pipe := client.Pipeline()
	pipe.Set(c.Request.Context(), adminSessionTokenKey(session.Token), serialized, ttl)
	pipe.Set(c.Request.Context(), adminSessionIDKey(strconv.FormatInt(session.ID, 10)), serialized, ttl)
	if session.RefreshToken != "" {
		pipe.Set(c.Request.Context(), adminSessionRefreshKey(session.RefreshToken), serialized, ttl)
	}
	pipe.SAdd(c.Request.Context(), adminUserSessionsKey(session.UserID), strconv.FormatInt(session.ID, 10))
	pipe.ExpireNX(c.Request.Context(), adminUserSessionsKey(session.UserID), ttl+time.Hour)
	pipe.ExpireGT(c.Request.Context(), adminUserSessionsKey(session.UserID), ttl+time.Hour)
	pipe.ZAdd(c.Request.Context(), adminAllSessionsKey, redis.Z{Score: adminSessionCreatedScore(session), Member: strconv.FormatInt(session.ID, 10)})
	_, err = pipe.Exec(c.Request.Context())
	if err == nil && h.Auth != nil {
		h.Auth.EnforceUserSessionLimit(c.Request.Context(), session.UserID, h.Auth.SessionPolicy())
	}
	return err
}

func (h NativeHandlers) adminDeleteSessionByID(c *gin.Context, client *redis.Client, id string) error {
	var session services.Session
	if h.Redis != nil {
		_ = h.Redis.GetJSON(c.Request.Context(), adminSessionIDKey(id), &session)
	}
	pipe := client.Pipeline()
	if session.Token != "" {
		pipe.Del(c.Request.Context(), adminSessionTokenKey(session.Token))
	}
	if session.RefreshToken != "" {
		pipe.Del(c.Request.Context(), adminSessionRefreshKey(session.RefreshToken))
	}
	if session.UserID != "" {
		pipe.SRem(c.Request.Context(), adminUserSessionsKey(session.UserID), id)
	}
	pipe.Del(c.Request.Context(), adminSessionIDKey(id))
	pipe.ZRem(c.Request.Context(), adminAllSessionsKey, id)
	_, err := pipe.Exec(c.Request.Context())
	return err
}

func (h NativeHandlers) adminSessionUsers(sessions []services.Session) map[string]adminSessionUser {
	ids := make([]int64, 0, len(sessions))
	seen := map[int64]struct{}{}
	for _, session := range sessions {
		userID, ok := int64FromAny(session.UserID)
		if !ok || userID <= 0 {
			continue
		}
		if _, exists := seen[userID]; exists {
			continue
		}
		seen[userID] = struct{}{}
		ids = append(ids, userID)
	}
	out := map[string]adminSessionUser{}
	if len(ids) == 0 || h.DB == nil {
		return out
	}
	var rows []adminSessionUser
	if err := h.DB.Table("users").Select("id, user_id, nickname").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return out
	}
	for _, row := range rows {
		out[strconv.FormatInt(row.ID, 10)] = row
	}
	return out
}

func adminParseRedisSession(value any) (services.Session, bool) {
	raw, ok := value.(string)
	if !ok || raw == "" {
		return services.Session{}, false
	}
	var session services.Session
	if err := json.Unmarshal([]byte(raw), &session); err != nil {
		return services.Session{}, false
	}
	if session.ID == 0 {
		return services.Session{}, false
	}
	return session, true
}

func adminSessionListRow(session services.Session, user adminSessionUser) gin.H {
	userID, _ := int64FromAny(session.UserID)
	return gin.H{
		"id":              session.ID,
		"user_id":         userID,
		"refresh_token":   session.RefreshToken,
		"user_agent":      session.UserAgent,
		"is_active":       session.IsActive,
		"expires_at":      session.ExpiresAt,
		"created_at":      session.CreatedAt,
		"last_active_at":  adminSessionLastActiveText(session),
		"user_display_id": user.UserID,
		"nickname":        user.Nickname,
	}
}

func adminSessionDetailRow(session services.Session, user adminSessionUser) gin.H {
	userID, _ := int64FromAny(session.UserID)
	var userPayload any
	if user.ID != 0 {
		userPayload = gin.H{"id": user.ID, "user_id": user.UserID, "nickname": user.Nickname}
	}
	return gin.H{
		"id":             session.ID,
		"user_id":        userID,
		"token":          session.Token,
		"refresh_token":  session.RefreshToken,
		"user_agent":     session.UserAgent,
		"client_ip":      session.ClientIP,
		"fingerprint":    session.Fingerprint,
		"is_active":      session.IsActive,
		"expires_at":     session.ExpiresAt,
		"created_at":     session.CreatedAt,
		"last_active_at": adminSessionLastActiveText(session),
		"user":           userPayload,
	}
}

func adminSortSessions(sessions []services.Session, sortField string, sortOrder string) {
	sortField = strings.TrimSpace(sortField)
	switch sortField {
	case "id", "user_id", "expires_at", "created_at", "last_active_at", "is_active":
	default:
		sortField = "created_at"
	}
	desc := !strings.EqualFold(sortOrder, "asc")
	sort.SliceStable(sessions, func(i, j int) bool {
		cmp := adminCompareSessionField(sessions[i], sessions[j], sortField)
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
}

func adminCompareSessionField(a services.Session, b services.Session, field string) int {
	switch field {
	case "id":
		return compareInt64(a.ID, b.ID)
	case "user_id":
		av, _ := int64FromAny(a.UserID)
		bv, _ := int64FromAny(b.UserID)
		return compareInt64(av, bv)
	case "expires_at":
		return compareTime(adminParseSessionTime(a.ExpiresAt), adminParseSessionTime(b.ExpiresAt))
	case "last_active_at":
		return compareTime(adminSessionLastActiveTime(a), adminSessionLastActiveTime(b))
	case "is_active":
		return compareBool(a.IsActive, b.IsActive)
	default:
		return compareTime(adminParseSessionTime(a.CreatedAt), adminParseSessionTime(b.CreatedAt))
	}
}

func adminSessionTTL(session services.Session) time.Duration {
	expiresAt := adminParseSessionTime(session.ExpiresAt)
	if expiresAt.IsZero() {
		return adminSessionDefaultTTL
	}
	ttl := time.Until(expiresAt)
	if ttl < time.Second {
		return time.Second
	}
	return ttl
}

func adminSessionCreatedScore(session services.Session) float64 {
	createdAt := adminParseSessionTime(session.CreatedAt)
	if createdAt.IsZero() {
		return float64(time.Now().UnixMilli())
	}
	return float64(createdAt.UnixMilli())
}

func adminSessionLastActiveText(session services.Session) string {
	if strings.TrimSpace(session.LastActiveAt) != "" {
		return session.LastActiveAt
	}
	return session.CreatedAt
}

func adminSessionLastActiveTime(session services.Session) time.Time {
	if lastActive := adminParseSessionTime(session.LastActiveAt); !lastActive.IsZero() {
		return lastActive
	}
	return adminParseSessionTime(session.CreatedAt)
}

func adminParseSessionTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func jsTruthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case float64:
		return typed != 0
	case int:
		return typed != 0
	case int64:
		return typed != 0
	default:
		return true
	}
}

func stringSliceToAny(values []string) []any {
	out := make([]any, 0, len(values))
	for _, value := range values {
		out = append(out, value)
	}
	return out
}

func compareInt64(a int64, b int64) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func compareTime(a time.Time, b time.Time) int {
	switch {
	case a.Before(b):
		return -1
	case a.After(b):
		return 1
	default:
		return 0
	}
}

func compareBool(a bool, b bool) int {
	switch {
	case a == b:
		return 0
	case !a && b:
		return -1
	default:
		return 1
	}
}

func adminSessionTokenKey(token string) string   { return adminSessionKeyPrefix + "token:" + token }
func adminSessionIDKey(id string) string         { return adminSessionKeyPrefix + "id:" + id }
func adminSessionRefreshKey(token string) string { return adminSessionKeyPrefix + "refresh:" + token }
func adminUserSessionsKey(userID string) string  { return "user_sessions:" + userID }
