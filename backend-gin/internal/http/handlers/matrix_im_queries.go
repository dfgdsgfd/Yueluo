package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
)

func (h NativeHandlers) imSync(c *gin.Context, uid int64) {
	since, _ := int64FromAny(c.Query("since"))
	limit := min(positiveIntQuery(c, "limit", 200), 500)
	var memberships []domain.IMConversationMember
	if err := h.DB.WithContext(c.Request.Context()).Where("user_id = ?", uid).Select("conversation_id").Find(&memberships).Error; writeDBError(c, err, "") {
		return
	}
	convIDs := make([]int64, 0, len(memberships))
	for _, m := range memberships {
		convIDs = append(convIDs, m.ConversationID)
	}
	if len(convIDs) == 0 {
		writeSuccess(c, "ok", gin.H{"messages": []gin.H{}, "cursor": since})
		return
	}
	var messages []domain.IMMessage
	if err := h.DB.WithContext(c.Request.Context()).Where("conversation_id IN ? AND id > ?", convIDs, since).Order("id ASC").Limit(limit).Find(&messages).Error; writeDBError(c, err, "") {
		return
	}
	receipts, _ := h.imReceipts(c, messageIDs(messages))
	out := make([]gin.H, 0, len(messages))
	cursor := since
	for _, msg := range messages {
		out = append(out, shapeIMMessage(msg, receipts[msg.ID], uid))
		cursor = msg.ID
	}
	writeSuccess(c, "ok", gin.H{"messages": out, "cursor": cursor, "has_more": len(messages) == limit})
}

func (h NativeHandlers) imUsers(c *gin.Context, uid int64) {
	var memberships []domain.IMConversationMember
	if err := h.DB.WithContext(c.Request.Context()).Where("user_id = ?", uid).Select("conversation_id").Find(&memberships).Error; writeDBError(c, err, "") {
		return
	}
	convIDs := make([]int64, 0, len(memberships))
	for _, m := range memberships {
		convIDs = append(convIDs, m.ConversationID)
	}
	if len(convIDs) == 0 {
		writeSuccess(c, "ok", []gin.H{})
		return
	}
	var others []domain.IMConversationMember
	if err := h.DB.WithContext(c.Request.Context()).Where("conversation_id IN ? AND user_id <> ?", convIDs, uid).Find(&others).Error; writeDBError(c, err, "") {
		return
	}
	ids := uniqueInt64Local(memberIDsFromMembers(others))
	var users []domain.User
	if len(ids) > 0 {
		if err := h.DB.WithContext(c.Request.Context()).Where("id IN ? AND is_active = ?", ids, true).Find(&users).Error; writeDBError(c, err, "") {
			return
		}
	}
	out := make([]gin.H, 0, len(users))
	for _, user := range users {
		out = append(out, gin.H{"id": user.ID, "username": user.Nickname, "nickname": user.Nickname, "avatar": h.signFileURLPtr(user.Avatar)})
	}
	writeSuccess(c, "ok", out)
}

func (h NativeHandlers) imTokenForUser(c *gin.Context, uid int64) (sig string, ts string, err error) {
	ts = strconv.FormatInt(time.Now().Unix(), 10)
	sigInput := "uid=" + strconv.FormatInt(uid, 10) + "&ts=" + ts
	mac := hmac.New(sha256.New, []byte(h.Config.IM.HMACSecret))
	mac.Write([]byte(sigInput))
	sig = hex.EncodeToString(mac.Sum(nil))
	return sig, ts, nil
}

func (h NativeHandlers) imAPIBase() string {
	base := strings.TrimRight(h.Config.IM.BaseURL, "/")
	if base == "" {
		return "https://ws-im.yuelk.com:29443"
	}
	return base
}

func imWebSocketURL(apiBase string) string {
	base := strings.TrimRight(apiBase, "/")
	switch {
	case strings.HasPrefix(base, "https://"):
		return "wss://" + strings.TrimPrefix(base, "https://") + "/ws"
	case strings.HasPrefix(base, "http://"):
		return "ws://" + strings.TrimPrefix(base, "http://") + "/ws"
	case strings.HasPrefix(base, "wss://") || strings.HasPrefix(base, "ws://"):
		return base + "/ws"
	default:
		return "wss://" + base + "/ws"
	}
}

func (h NativeHandlers) findDirectConversation(c *gin.Context, a int64, b int64) (domain.IMConversation, bool) {
	var rows []struct {
		ConversationID int64 `gorm:"column:conversation_id"`
	}
	err := h.DB.WithContext(c.Request.Context()).Table("im_conversation_members").
		Select("conversation_id").
		Where("user_id IN ?", []int64{a, b}).
		Group("conversation_id").
		Having("COUNT(DISTINCT user_id) = 2").
		Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return domain.IMConversation{}, false
	}
	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ConversationID)
	}
	var conv domain.IMConversation
	err = h.DB.WithContext(c.Request.Context()).Where("id IN ? AND type = ?", ids, "direct").First(&conv).Error
	return conv, err == nil
}

func (h NativeHandlers) imEnsureMember(c *gin.Context, convID int64, uid int64) bool {
	var count int64
	err := h.DB.WithContext(c.Request.Context()).Model(&domain.IMConversationMember{}).Where("conversation_id = ? AND user_id = ?", convID, uid).Count(&count).Error
	if writeDBError(c, err, "") {
		return false
	}
	if count == 0 {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "会话不存在或无权访问", nil)
		return false
	}
	return true
}

func (h NativeHandlers) imConversationMembersForUser(c *gin.Context, convID int64, uid int64) ([]domain.IMConversationMember, bool) {
	var members []domain.IMConversationMember
	if err := h.DB.WithContext(c.Request.Context()).Where("conversation_id = ?", convID).Find(&members).Error; writeDBError(c, err, "") {
		return nil, false
	}
	for _, member := range members {
		if member.UserID == uid {
			return members, true
		}
	}
	response.JSON(c, http.StatusNotFound, response.CodeNotFound, "会话不存在或无权访问", nil)
	return nil, false
}

func (h NativeHandlers) imMembers(c *gin.Context, convIDs []int64) (map[int64][]domain.IMConversationMember, error) {
	out := map[int64][]domain.IMConversationMember{}
	if len(convIDs) == 0 {
		return out, nil
	}
	var rows []domain.IMConversationMember
	err := h.DB.WithContext(c.Request.Context()).Where("conversation_id IN ?", uniqueInt64Local(convIDs)).Find(&rows).Error
	for _, row := range rows {
		out[row.ConversationID] = append(out[row.ConversationID], row)
	}
	return out, err
}

func (h NativeHandlers) imUsersByMembers(c *gin.Context, members map[int64][]domain.IMConversationMember) (map[int64]domain.User, error) {
	ids := []int64{}
	for _, rows := range members {
		ids = append(ids, memberIDsFromMembers(rows)...)
	}
	ids = uniqueInt64Local(ids)
	out := map[int64]domain.User{}
	if len(ids) == 0 {
		return out, nil
	}
	var users []domain.User
	err := h.DB.WithContext(c.Request.Context()).Where("id IN ?", ids).Find(&users).Error
	for _, user := range users {
		out[user.ID] = user
	}
	return out, err
}

func (h NativeHandlers) imLastMessage(c *gin.Context, conv domain.IMConversation) (*domain.IMMessage, error) {
	var msg domain.IMMessage
	query := h.DB.WithContext(c.Request.Context())
	var err error
	if conv.LastMessageID != nil {
		err = query.Where("id = ?", *conv.LastMessageID).First(&msg).Error
	} else {
		err = query.Where("conversation_id = ?", conv.ID).Order("created_at DESC").First(&msg).Error
	}
	if err != nil {
		return nil, err
	}
	return &msg, nil
}

func (h NativeHandlers) imLastMessages(c *gin.Context, convs []domain.IMConversation) (map[int64]*domain.IMMessage, error) {
	out := map[int64]*domain.IMMessage{}
	if len(convs) == 0 {
		return out, nil
	}
	lastIDs := []int64{}
	needsLookup := []int64{}
	for _, conv := range convs {
		if conv.LastMessageID != nil {
			lastIDs = append(lastIDs, *conv.LastMessageID)
		} else {
			needsLookup = append(needsLookup, conv.ID)
		}
	}
	if len(needsLookup) > 0 {
		var rows []struct {
			ConversationID int64 `gorm:"column:conversation_id"`
			MessageID      int64 `gorm:"column:message_id"`
		}
		if err := h.DB.WithContext(c.Request.Context()).Model(&domain.IMMessage{}).
			Select("conversation_id, MAX(id) AS message_id").
			Where("conversation_id IN ?", uniqueInt64Local(needsLookup)).
			Group("conversation_id").
			Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, row := range rows {
			if row.MessageID > 0 {
				lastIDs = append(lastIDs, row.MessageID)
			}
		}
	}
	if len(lastIDs) == 0 {
		return out, nil
	}
	var messages []domain.IMMessage
	if err := h.DB.WithContext(c.Request.Context()).Where("id IN ?", uniqueInt64Local(lastIDs)).Find(&messages).Error; err != nil {
		return nil, err
	}
	for i := range messages {
		msg := messages[i]
		out[msg.ConversationID] = &msg
	}
	return out, nil
}

func (h NativeHandlers) imUnreadCount(c *gin.Context, convID int64, uid int64, lastRead *int64) int64 {
	query := h.DB.WithContext(c.Request.Context()).Model(&domain.IMMessage{}).Where("conversation_id = ? AND sender_id <> ?", convID, uid)
	if lastRead != nil {
		query = query.Where("id > ?", *lastRead)
	}
	var count int64
	_ = query.Count(&count).Error
	return count
}

func (h NativeHandlers) imUnreadCounts(c *gin.Context, convIDs []int64, uid int64) map[int64]int64 {
	out := map[int64]int64{}
	if len(convIDs) == 0 {
		return out
	}
	var rows []struct {
		ConversationID int64 `gorm:"column:conversation_id"`
		Count          int64 `gorm:"column:count"`
	}
	err := h.DB.WithContext(c.Request.Context()).Table("im_messages AS m").
		Select("m.conversation_id, COUNT(*) AS count").
		Joins("JOIN im_conversation_members AS cm ON cm.conversation_id = m.conversation_id AND cm.user_id = ?", uid).
		Where("m.conversation_id IN ? AND m.sender_id <> ?", uniqueInt64Local(convIDs), uid).
		Where("(cm.last_read_message_id IS NULL OR m.id > cm.last_read_message_id)").
		Group("m.conversation_id").
		Scan(&rows).Error
	if err != nil {
		return out
	}
	for _, row := range rows {
		out[row.ConversationID] = row.Count
	}
	return out
}

func (h NativeHandlers) imReceipts(c *gin.Context, msgIDs []int64) (map[int64][]domain.IMMessageReceipt, error) {
	out := map[int64][]domain.IMMessageReceipt{}
	if len(msgIDs) == 0 {
		return out, nil
	}
	var rows []domain.IMMessageReceipt
	err := h.DB.WithContext(c.Request.Context()).Where("message_id IN ?", uniqueInt64Local(msgIDs)).Find(&rows).Error
	for _, row := range rows {
		out[row.MessageID] = append(out[row.MessageID], row)
	}
	return out, err
}
