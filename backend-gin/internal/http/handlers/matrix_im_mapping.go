package handlers

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"yuem-go/backend-gin/internal/domain"
)

const (
	imPushCoalesceWindow = 250 * time.Millisecond
	imPushWriteTimeout   = 150 * time.Millisecond
)

type imPushKey struct {
	uid            int64
	conversationID string
}

var imPushScheduler = struct {
	sync.Mutex
	pending map[imPushKey]bool
	timers  map[imPushKey]*time.Timer
}{
	pending: map[imPushKey]bool{},
	timers:  map[imPushKey]*time.Timer{},
}

func (h NativeHandlers) imConversationMap(conv domain.IMConversation, members []domain.IMConversationMember, users map[int64]domain.User, uid int64, lastMessage *domain.IMMessage, unread int64) gin.H {
	memberIDs := memberIDsFromMembers(members)
	visibleMembers := make([]gin.H, 0, len(members))
	for _, member := range members {
		if member.UserID == uid {
			continue
		}
		if user, ok := users[member.UserID]; ok {
			visibleMembers = append(visibleMembers, gin.H{"id": user.ID, "username": user.Nickname, "nickname": user.Nickname, "avatar": h.signFileURLPtr(user.Avatar)})
		}
	}
	var last any
	if lastMessage != nil {
		last = shapeIMMessage(*lastMessage, nil, uid)
	}
	return gin.H{"id": conv.ID, "external_id": conv.ExternalID, "type": conv.Type, "name": conv.Name, "creator_id": conv.CreatorID, "member_ids": memberIDs, "members": visibleMembers, "last_message": last, "unread_count": unread, "created_at": conv.CreatedAt, "updated_at": conv.UpdatedAt}
}

func shapeIMMessage(msg domain.IMMessage, receipts []domain.IMMessageReceipt, currentUID int64) gin.H {
	status := "sent"
	if msg.SenderID == currentUID {
		others := 0
		read := 0
		delivered := 0
		for _, receipt := range receipts {
			if receipt.UserID == currentUID {
				continue
			}
			others++
			if receipt.ReadAt != nil {
				read++
			}
			if receipt.DeliveredAt != nil || receipt.ReadAt != nil {
				delivered++
			}
		}
		if others > 0 && read == others {
			status = "read"
		} else if delivered > 0 {
			status = "delivered"
		}
	} else {
		for _, receipt := range receipts {
			if receipt.UserID == currentUID {
				if receipt.ReadAt != nil {
					status = "read"
				} else if receipt.DeliveredAt != nil {
					status = "delivered"
				}
			}
		}
	}
	return gin.H{"id": msg.ID, "conversation_id": msg.ConversationID, "sender_id": msg.SenderID, "content": msg.Content, "client_msg_id": msg.ClientMsgID, "created_at": msg.CreatedAt, "status": status}
}

func imHubAdd(uid int64, conn *websocket.Conn) {
	imWSHub.Lock()
	defer imWSHub.Unlock()
	if imWSHub.sockets[uid] == nil {
		imWSHub.sockets[uid] = map[*websocket.Conn]bool{}
	}
	imWSHub.sockets[uid][conn] = true
}

func imHubRemove(uid int64, conn *websocket.Conn) {
	imWSHub.Lock()
	defer imWSHub.Unlock()
	delete(imWSHub.sockets[uid], conn)
	if len(imWSHub.sockets[uid]) == 0 {
		delete(imWSHub.sockets, uid)
	}
}

func imHubPush(uid int64, payload any) {
	data, _ := json.Marshal(payload)
	imWSHub.RLock()
	conns := make([]*websocket.Conn, 0, len(imWSHub.sockets[uid]))
	for conn := range imWSHub.sockets[uid] {
		conns = append(conns, conn)
	}
	imWSHub.RUnlock()
	for _, conn := range conns {
		_ = conn.SetWriteDeadline(time.Now().Add(imPushWriteTimeout))
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			imHubRemove(uid, conn)
			_ = conn.Close()
		}
	}
}

func imHubScheduleNewMessage(uid int64, conversationID string) {
	if uid <= 0 || conversationID == "" {
		return
	}
	key := imPushKey{uid: uid, conversationID: conversationID}
	imPushScheduler.Lock()
	if imPushScheduler.pending[key] {
		imPushScheduler.Unlock()
		return
	}
	imPushScheduler.pending[key] = true
	imPushScheduler.timers[key] = time.AfterFunc(imPushCoalesceWindow, func() {
		imPushScheduler.Lock()
		delete(imPushScheduler.pending, key)
		delete(imPushScheduler.timers, key)
		imPushScheduler.Unlock()
		imHubPush(uid, gin.H{"type": "new_message", "conversation_id": conversationID})
	})
	imPushScheduler.Unlock()
}

func mapKeysInt64(values map[int64]bool) []int64 {
	out := make([]int64, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	return out
}

func memberIDsFromMembers(rows []domain.IMConversationMember) []int64 {
	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.UserID)
	}
	return out
}

func messageIDs(rows []domain.IMMessage) []int64 {
	out := make([]int64, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.ID)
	}
	return out
}

func reverseMessages(rows []domain.IMMessage) {
	for i, j := 0, len(rows)-1; i < j; i, j = i+1, j-1 {
		rows[i], rows[j] = rows[j], rows[i]
	}
}

func upsertReceipt(tx *gorm.DB, msgID int64, uid int64, updates map[string]any) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "message_id"}, {Name: "user_id"}},
		DoUpdates: clause.Assignments(updates),
	}).Create(&domain.IMMessageReceipt{MessageID: msgID, UserID: uid}).Error
}
