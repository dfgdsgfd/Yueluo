package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
	"yuem-go/backend-gin/internal/repositories"
)

const (
	msgInviteInternalError      = "\u670d\u52a1\u5668\u5185\u90e8\u9519\u8bef"
	msgInviteCodeNotFound       = "\u9080\u8bf7\u7801\u4e0d\u5b58\u5728"
	msgInviteCodeInactive       = "\u9080\u8bf7\u7801\u4e0d\u5b58\u5728\u6216\u5df2\u7981\u7528"
	msgInviteRecordNotFound     = "\u8bb0\u5f55\u4e0d\u5b58\u5728"
	msgInviteBadParams          = "\u53c2\u6570\u9519\u8bef"
	msgInviteUserNotFound       = "\u7528\u6237\u4e0d\u5b58\u5728"
	msgInviteRewardOK           = "\u6536\u76ca\u53d1\u653e\u6210\u529f"
	msgInviteManualRewardReason = "\u7ba1\u7406\u5458\u624b\u52a8\u53d1\u653e"
)

func (h NativeHandlers) InviteMyCode(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	record, err := repositories.NewInviteRepository(h.DB).GetOrCreateCode(c.Request.Context(), user.ID)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": inviteCodePayload(h, *record)})
}

func (h NativeHandlers) InviteClick(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	recorded, err := repositories.NewInviteRepository(h.DB).RecordClick(c.Request.Context(), c.Param("code"), hashIP(realIP(c)), c.GetHeader("User-Agent"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgInviteCodeInactive, nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{"recorded": recorded}})
}

func (h NativeHandlers) InviteStats(c *gin.Context) {
	user, ok := currentUser(c)
	if !ok {
		response.JSON(c, http.StatusUnauthorized, response.CodeUnauthorized, msgInvalidToken, nil)
		return
	}
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 20), 100)
	stats, err := repositories.NewInviteRepository(h.DB).Stats(c.Request.Context(), user.ID, page, limit)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	body := inviteCodePayload(h, stats.Code)
	body["invitees"] = h.inviteeResponses(stats.Invitees)
	body["invitees_total"] = stats.InviteesTotal
	body["earnings_logs"] = inviteEarningsLogResponses(stats.EarningsLogs)
	body["pagination"] = pagination(page, limit, stats.InviteesTotal)
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": body})
}

func (h NativeHandlers) InviteInfo(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	user, err := repositories.NewInviteRepository(h.DB).Info(c.Request.Context(), c.Param("code"))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgInviteCodeNotFound, nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{"nickname": user.Nickname, "avatar": h.signFileURLPtr(user.Avatar)}})
}

func (h NativeHandlers) InviteAdminList(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	page := positiveIntQuery(c, "page", 1)
	limit := min(positiveIntQuery(c, "limit", 20), 100)
	total, records, err := repositories.NewInviteRepository(h.DB).AdminList(c.Request.Context(), page, limit, c.Query("keyword"))
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	list := make([]gin.H, 0, len(records))
	for _, record := range records {
		list = append(list, h.inviteAdminResponse(record))
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"message": "success",
		"data": gin.H{
			"list":       list,
			"pagination": pagination(page, limit, total),
		},
	})
}

func (h NativeHandlers) InviteAdminToggle(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgInviteRecordNotFound, nil)
		return
	}
	active, err := repositories.NewInviteRepository(h.DB).Toggle(c.Request.Context(), id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgInviteRecordNotFound, nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "success", "data": gin.H{"is_active": active}})
}

type inviteRewardRequest struct {
	UserID any    `json:"user_id"`
	Amount any    `json:"amount"`
	Type   string `json:"type"`
	Reason string `json:"reason"`
}

func (h NativeHandlers) InviteAdminReward(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	var body inviteRewardRequest
	_ = c.ShouldBindJSON(&body)
	userID, okID := int64FromAny(body.UserID)
	amount, okAmount := float64FromAny(body.Amount)
	if !okID || !okAmount || amount == 0 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, msgInviteBadParams, nil)
		return
	}
	rewardType := body.Type
	if rewardType == "" {
		rewardType = "manual_reward"
	}
	reasonText := body.Reason
	if reasonText == "" {
		reasonText = msgInviteManualRewardReason
	}
	err := repositories.NewInviteRepository(h.DB).Reward(c.Request.Context(), userID, amount, rewardType, &reasonText)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, msgInviteUserNotFound, nil)
		return
	}
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": msgInviteRewardOK})
}

func (h NativeHandlers) InviteAdminOverview(c *gin.Context) {
	if h.DB == nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	overview, err := repositories.NewInviteRepository(h.DB).Overview(c.Request.Context())
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msgInviteInternalError, nil)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    response.CodeSuccess,
		"message": "success",
		"data": gin.H{
			"total_codes":     overview.TotalCodes,
			"total_clicks":    overview.TotalClicks,
			"total_registers": overview.TotalRegisters,
			"total_earnings":  overview.TotalEarnings,
		},
	})
}

func inviteCodePayload(h NativeHandlers, record domain.InviteCode) gin.H {
	return gin.H{
		"invite_code":    record.Code,
		"invite_url":     h.Config.Frontend.BaseURL + "/invite/" + record.Code,
		"click_count":    record.ClickCount,
		"register_count": record.RegisterCount,
		"total_earnings": record.TotalEarnings,
		"is_active":      record.IsActive,
		"created_at":     record.CreatedAt,
	}
}

func (h NativeHandlers) inviteeResponses(invitees []repositories.InviteeBundle) []gin.H {
	out := make([]gin.H, 0, len(invitees))
	for _, invitee := range invitees {
		if invitee.User == nil {
			continue
		}
		out = append(out, gin.H{
			"user_id":   invitee.User.UserID,
			"nickname":  invitee.User.Nickname,
			"avatar":    h.signFileURLPtr(invitee.User.Avatar),
			"joined_at": invitee.Code.CreatedAt,
		})
	}
	return out
}

func inviteEarningsLogResponses(logs []domain.InviteEarningsLog) []gin.H {
	out := make([]gin.H, 0, len(logs))
	for _, log := range logs {
		out = append(out, gin.H{
			"id":         log.ID,
			"amount":     log.Amount,
			"type":       log.Type,
			"reason":     log.Reason,
			"created_at": log.CreatedAt,
		})
	}
	return out
}

func (h NativeHandlers) inviteAdminResponse(bundle repositories.InviteAdminBundle) gin.H {
	var userID string
	var nickname string
	var avatar *string
	if bundle.User != nil {
		userID = bundle.User.UserID
		nickname = bundle.User.Nickname
		avatar = bundle.User.Avatar
	}
	var invitedByNickname any
	if bundle.InvitedBy != nil {
		invitedByNickname = bundle.InvitedBy.Nickname
	}
	return gin.H{
		"id":                  bundle.Code.ID,
		"code":                bundle.Code.Code,
		"user_id":             userID,
		"nickname":            nickname,
		"avatar":              h.signFileURLPtr(avatar),
		"invited_by_nickname": invitedByNickname,
		"click_count":         bundle.Code.ClickCount,
		"register_count":      bundle.Code.RegisterCount,
		"total_earnings":      bundle.Code.TotalEarnings,
		"is_active":           bundle.Code.IsActive,
		"created_at":          bundle.Code.CreatedAt,
	}
}

func hashIP(ip string) string {
	sum := sha256.Sum256([]byte(ip))
	return hex.EncodeToString(sum[:])[:32]
}

func realIP(c *gin.Context) string {
	ip := c.GetHeader("X-Forwarded-For")
	if ip == "" {
		ip = c.GetHeader("X-Real-IP")
	}
	if ip == "" {
		ip = c.ClientIP()
	}
	if after, ok := strings.CutPrefix(ip, "::ffff:"); ok {
		ip = after
	}
	if strings.Contains(ip, ",") {
		ip = strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	return ip
}

func float64FromAny(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case string:
		if strings.TrimSpace(typed) == "" {
			return 0, false
		}
		parsed, err := strconv.ParseFloat(typed, 64)
		return parsed, err == nil && !math.IsNaN(parsed)
	default:
		return 0, false
	}
}
