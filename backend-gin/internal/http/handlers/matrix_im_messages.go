package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/http/response"
)

func (h NativeHandlers) imConversations(c *gin.Context, uid int64) {
	var members []domain.IMConversationMember
	if err := h.DB.WithContext(c.Request.Context()).Where("user_id = ?", uid).Find(&members).Error; writeDBError(c, err, "") {
		return
	}
	convIDs := make([]int64, 0, len(members))
	for _, m := range members {
		convIDs = append(convIDs, m.ConversationID)
	}
	if len(convIDs) == 0 {
		writeSuccess(c, "ok", []gin.H{})
		return
	}
	var convs []domain.IMConversation
	if err := h.DB.WithContext(c.Request.Context()).Where("id IN ?", convIDs).Order("updated_at DESC").Find(&convs).Error; writeDBError(c, err, "") {
		return
	}
	allMembers, err := h.imMembers(c, convIDs)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	userMap, err := h.imUsersByMembers(c, allMembers)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	out := make([]gin.H, 0, len(convs))
	lastMessages, err := h.imLastMessages(c, convs)
	if err != nil {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return
	}
	unreadCounts := h.imUnreadCounts(c, convIDs, uid)
	for _, conv := range convs {
		out = append(out, h.imConversationMap(conv, allMembers[conv.ID], userMap, uid, lastMessages[conv.ID], unreadCounts[conv.ID]))
	}
	writeSuccess(c, "ok", out)
}

func (h NativeHandlers) imCreateConversation(c *gin.Context, uid int64) {
	body := readBodyMap(c)
	rawIDs := int64SliceFromAny(body["member_ids"])
	memberSet := map[int64]bool{uid: true}
	for _, id := range rawIDs {
		if id > 0 {
			memberSet[id] = true
		}
	}
	memberIDs := mapKeysInt64(memberSet)
	if len(memberIDs) < 2 {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "至少需要 2 名成员", nil)
		return
	}
	var valid int64
	if err := h.DB.WithContext(c.Request.Context()).Model(&domain.User{}).Where("id IN ? AND is_active = ?", memberIDs, true).Count(&valid).Error; writeDBError(c, err, "") {
		return
	}
	if valid != int64(len(memberIDs)) {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "存在无效成员", nil)
		return
	}
	convType := "group"
	if len(memberIDs) == 2 {
		convType = "direct"
		if existing, found := h.findDirectConversation(c, memberIDs[0], memberIDs[1]); found {
			members, _ := h.imMembers(c, []int64{existing.ID})
			writeSuccess(c, "ok", gin.H{"id": existing.ID, "external_id": existing.ExternalID, "type": existing.Type, "name": existing.Name, "member_ids": memberIDsFromMembers(members[existing.ID]), "created_at": existing.CreatedAt, "updated_at": existing.UpdatedAt})
			return
		}
	}
	name := sanitizePlainSubmittedText(toString(body["name"]))
	var created domain.IMConversation
	err := h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var namePtr *string
		if convType == "group" && name != "" {
			namePtr = &name
		}
		created = domain.IMConversation{Type: convType, Name: namePtr, CreatorID: uid}
		if err := tx.Create(&created).Error; err != nil {
			return err
		}
		rows := make([]domain.IMConversationMember, 0, len(memberIDs))
		now := time.Now()
		for _, id := range memberIDs {
			rows = append(rows, domain.IMConversationMember{ConversationID: created.ID, UserID: id, JoinedAt: now})
		}
		return tx.Create(&rows).Error
	})
	if writeDBError(c, err, "") {
		return
	}
	writeSuccess(c, "ok", gin.H{"id": created.ID, "external_id": created.ExternalID, "type": created.Type, "name": created.Name, "member_ids": memberIDs, "created_at": created.CreatedAt, "updated_at": created.UpdatedAt})
}

func (h NativeHandlers) imMessages(c *gin.Context, uid int64) {
	convID, ok := int64FromAny(matrixParam(c, "id"))
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效会话 ID", nil)
		return
	}
	if !h.imEnsureMember(c, convID, uid) {
		return
	}
	limit := min(positiveIntQuery(c, "limit", 50), imMaxPageSize)
	query := h.DB.WithContext(c.Request.Context()).Where("conversation_id = ?", convID)
	if before, ok := int64FromAny(c.Query("before")); ok && before > 0 {
		query = query.Where("id < ?", before)
	}
	var messages []domain.IMMessage
	if err := query.Order("id DESC").Limit(limit).Find(&messages).Error; writeDBError(c, err, "") {
		return
	}
	reverseMessages(messages)
	receipts, _ := h.imReceipts(c, messageIDs(messages))
	out := make([]gin.H, 0, len(messages))
	for _, msg := range messages {
		out = append(out, shapeIMMessage(msg, receipts[msg.ID], uid))
	}
	nextBefore := any(nil)
	if len(messages) > 0 {
		nextBefore = messages[0].ID
	}
	c.JSON(http.StatusOK, gin.H{"code": response.CodeSuccess, "message": "ok", "data": out, "pagination": gin.H{"limit": limit, "has_more": len(messages) == limit, "next_before": nextBefore}})
}

func (h NativeHandlers) imSendMessage(c *gin.Context, uid int64) {
	convID, ok := int64FromAny(matrixParam(c, "id"))
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效会话 ID", nil)
		return
	}
	members, ok := h.imConversationMembersForUser(c, convID, uid)
	if !ok {
		return
	}
	body := readBodyMap(c)
	content := sanitizeMarkdownSubmittedText(toString(body["content"]))
	if content == "" {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "消息不能为空", nil)
		return
	}
	if len([]rune(content)) > imMaxMessageLen {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "消息长度不能超过 "+strconv.Itoa(imMaxMessageLen)+" 字符", nil)
		return
	}
	clientMsgID := toString(body["client_msg_id"])
	if clientMsgID != "" {
		var existing domain.IMMessage
		err := h.DB.WithContext(c.Request.Context()).Where("conversation_id = ? AND client_msg_id = ?", convID, clientMsgID).First(&existing).Error
		if err == nil {
			receipts, _ := h.imReceipts(c, []int64{existing.ID})
			writeSuccess(c, "ok", shapeIMMessage(existing, receipts[existing.ID], uid))
			return
		}
	}
	var msg domain.IMMessage
	err := h.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		var clientPtr *string
		if clientMsgID != "" {
			clientPtr = &clientMsgID
		}
		msg = domain.IMMessage{ConversationID: convID, SenderID: uid, Content: content, ClientMsgID: clientPtr}
		if err := tx.Create(&msg).Error; err != nil {
			return err
		}
		receipts := make([]domain.IMMessageReceipt, 0, len(members))
		for _, member := range members {
			if member.UserID != uid {
				receipts = append(receipts, domain.IMMessageReceipt{MessageID: msg.ID, UserID: member.UserID})
			}
		}
		if len(receipts) > 0 {
			if err := tx.Create(&receipts).Error; err != nil {
				return err
			}
		}
		now := time.Now()
		return tx.Model(&domain.IMConversation{}).Where("id = ?", convID).Updates(map[string]any{"last_message_id": msg.ID, "updated_at": now}).Error
	})
	if writeDBError(c, err, "") {
		return
	}
	receipts, _ := h.imReceipts(c, []int64{msg.ID})
	h.bumpCacheVersions(cacheScopeIM, cacheScopeNotifications)
	conversationID := strconv.FormatInt(convID, 10)
	for _, member := range members {
		if member.UserID != uid {
			imHubScheduleNewMessage(member.UserID, conversationID)
		}
	}
	writeSuccess(c, "ok", shapeIMMessage(msg, receipts[msg.ID], uid))
}

func (h NativeHandlers) imMarkDelivered(c *gin.Context, uid int64) {
	h.imMarkReceipt(c, uid, false)
}

func (h NativeHandlers) imMarkRead(c *gin.Context, uid int64) {
	h.imMarkReceipt(c, uid, true)
}

func (h NativeHandlers) imMarkReceipt(c *gin.Context, uid int64, read bool) {
	msgID, ok := int64FromAny(matrixParam(c, "id"))
	if !ok {
		response.JSON(c, http.StatusBadRequest, response.CodeValidationError, "无效消息 ID", nil)
		return
	}
	h.imMarkReceiptFast(c, uid, msgID, read)
}

func (h NativeHandlers) imMarkReceiptFast(c *gin.Context, uid int64, msgID int64, read bool) {
	now := time.Now()
	if read {
		convID, ok := h.imConversationIDForReadableMessage(c, uid, msgID)
		if !ok {
			return
		}
		messagesThroughTarget := h.DB.WithContext(c.Request.Context()).Model(&domain.IMMessage{}).
			Select("id").
			Where("conversation_id = ? AND id <= ?", convID, msgID)
		result := h.DB.WithContext(c.Request.Context()).Model(&domain.IMMessageReceipt{}).
			Where("user_id = ? AND read_at IS NULL", uid).
			Where("message_id IN (?)", messagesThroughTarget).
			Updates(map[string]any{
				"delivered_at": gorm.Expr("COALESCE(delivered_at, ?)", now),
				"read_at":      now,
				"updated_at":   now,
			})
		if writeDBError(c, result.Error, "") {
			return
		}
		h.imAdvanceReadCursorForConversation(c, uid, convID, msgID)
		h.bumpCacheVersions(cacheScopeIM)
		writeSimpleSuccess(c, "ok")
		return
	}

	result := h.DB.WithContext(c.Request.Context()).Model(&domain.IMMessageReceipt{}).
		Where("message_id = ? AND user_id = ?", msgID, uid).
		Where("delivered_at IS NULL").
		Where("EXISTS (SELECT 1 FROM im_messages AS m JOIN im_conversation_members AS cm ON cm.conversation_id = m.conversation_id AND cm.user_id = ? WHERE m.id = im_message_receipts.message_id AND m.id = ?)", uid, msgID).
		Updates(map[string]any{"delivered_at": now, "updated_at": now})
	if writeDBError(c, result.Error, "") {
		return
	}

	if result.RowsAffected > 0 {
		writeSimpleSuccess(c, "ok")
		return
	}

	if _, ok := h.imConversationIDForReadableMessage(c, uid, msgID); !ok {
		return
	}
	writeSimpleSuccess(c, "ok")
}

func (h NativeHandlers) imConversationIDForReadableMessage(c *gin.Context, uid int64, msgID int64) (int64, bool) {
	var row struct {
		ConversationID int64
	}
	err := h.DB.WithContext(c.Request.Context()).Table("im_messages AS m").
		Select("m.conversation_id").
		Joins("JOIN im_conversation_members AS cm ON cm.conversation_id = m.conversation_id AND cm.user_id = ?", uid).
		Where("m.id = ?", msgID).
		Take(&row).Error
	if err == nil {
		return row.ConversationID, true
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, matrixMsgInternal, nil)
		return 0, false
	}

	var count int64
	err = h.DB.WithContext(c.Request.Context()).Model(&domain.IMMessage{}).Where("id = ?", msgID).Count(&count).Error
	if writeDBError(c, err, "") {
		return 0, false
	}
	if count == 0 {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "消息不存在", nil)
	} else {
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "会话不存在或无权访问", nil)
	}
	return 0, false
}

func (h NativeHandlers) imAdvanceReadCursorForConversation(c *gin.Context, uid int64, convID int64, msgID int64) {
	_ = h.DB.WithContext(c.Request.Context()).Model(&domain.IMConversationMember{}).
		Where("conversation_id = ? AND user_id = ?", convID, uid).
		Where("last_read_message_id IS NULL OR last_read_message_id < ?", msgID).
		Update("last_read_message_id", msgID).Error
}
