package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"yuem-go/backend-gin/internal/config"
	"yuem-go/backend-gin/internal/domain"
	"yuem-go/backend-gin/internal/services"
)

type imQueryCounterLogger struct {
	mu    sync.Mutex
	count int
}

func (l *imQueryCounterLogger) LogMode(logger.LogLevel) logger.Interface { return l }
func (l *imQueryCounterLogger) Info(context.Context, string, ...any)     {}
func (l *imQueryCounterLogger) Warn(context.Context, string, ...any)     {}
func (l *imQueryCounterLogger) Error(context.Context, string, ...any)    {}

func (l *imQueryCounterLogger) Trace(context.Context, time.Time, func() (string, int64), error) {
	l.mu.Lock()
	l.count++
	l.mu.Unlock()
}

func (l *imQueryCounterLogger) Reset() {
	l.mu.Lock()
	l.count = 0
	l.mu.Unlock()
}

func (l *imQueryCounterLogger) Count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.count
}

func TestIMHubScheduleNewMessageCoalescesByUserAndConversation(t *testing.T) {
	imPushScheduler.Lock()
	for key, timer := range imPushScheduler.timers {
		timer.Stop()
		delete(imPushScheduler.timers, key)
	}
	imPushScheduler.pending = map[imPushKey]bool{}
	imPushScheduler.Unlock()
	t.Cleanup(func() {
		imPushScheduler.Lock()
		for key, timer := range imPushScheduler.timers {
			timer.Stop()
			delete(imPushScheduler.timers, key)
		}
		imPushScheduler.pending = map[imPushKey]bool{}
		imPushScheduler.Unlock()
	})

	imHubScheduleNewMessage(7, "99")
	imHubScheduleNewMessage(7, "99")

	imPushScheduler.Lock()
	pending := len(imPushScheduler.pending)
	timers := len(imPushScheduler.timers)
	imPushScheduler.Unlock()

	if pending != 1 || timers != 1 {
		t.Fatalf("scheduled pushes pending=%d timers=%d, want 1/1", pending, timers)
	}
}

func TestIMHubAddRemoveCleansSocketMap(t *testing.T) {
	imWSHub.Lock()
	imWSHub.sockets = map[int64]map[*websocket.Conn]bool{}
	imWSHub.Unlock()

	var conn websocket.Conn
	imHubAdd(7, &conn)
	imWSHub.RLock()
	if len(imWSHub.sockets[7]) != 1 {
		t.Fatalf("hub socket count = %d, want 1", len(imWSHub.sockets[7]))
	}
	imWSHub.RUnlock()

	imHubRemove(7, &conn)
	imWSHub.RLock()
	_, exists := imWSHub.sockets[7]
	imWSHub.RUnlock()
	if exists {
		t.Fatal("hub should remove empty user socket bucket")
	}
}

func TestIMWebSocketRemovesHubEntryWhenClientCloses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	imWSHub.Lock()
	imWSHub.sockets = map[int64]map[*websocket.Conn]bool{}
	imWSHub.Unlock()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&domain.User{}); err != nil {
		t.Fatalf("migrate users: %v", err)
	}
	user := domain.User{ID: 7, UserID: "u7", Nickname: "User 7", IsActive: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}

	redisServer := miniredis.RunT(t)
	redisStore := services.NewRedisStore(config.RedisConfig{Addr: redisServer.Addr()})
	auth := services.NewAuthService(db, redisStore, config.AuthConfig{JWTSecret: "test-secret", JWTExpiresIn: "3600"})
	token, err := auth.GenerateAccessToken(map[string]any{"userId": user.ID, "user_id": user.UserID})
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}
	if !auth.CreateSession(context.Background(), services.Session{UserID: "7", Token: token}, time.Hour) {
		t.Fatal("create session failed")
	}

	router := gin.New()
	router.GET("/api/im/ws", NativeHandlers{Auth: auth}.IMWebSocket)
	server := httptest.NewServer(router)
	defer server.Close()

	wsURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	wsURL.Scheme = "ws"
	wsURL.Path = "/api/im/ws"
	wsURL.RawQuery = "token=" + url.QueryEscape(token)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(2 * time.Second)
	for {
		imWSHub.RLock()
		socketCount := len(imWSHub.sockets[user.ID])
		imWSHub.RUnlock()
		if socketCount == 1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("websocket was not added to hub")
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
		t.Fatalf("write close message: %v", err)
	}

	deadline = time.Now().Add(2 * time.Second)
	for {
		imWSHub.RLock()
		_, exists := imWSHub.sockets[user.ID]
		imWSHub.RUnlock()
		if !exists {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("websocket hub entry was not cleaned after client close")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestIMWebSocketURLFromAPIBase(t *testing.T) {
	tests := []struct {
		name string
		base string
		want string
	}{
		{name: "https", base: "https://ws-im.yuelk.com:29443", want: "wss://ws-im.yuelk.com:29443/ws"},
		{name: "http", base: "http://im.example.com", want: "ws://im.example.com/ws"},
		{name: "bare host", base: "ws-im.yuelk.com:29443", want: "wss://ws-im.yuelk.com:29443/ws"},
		{name: "wss", base: "wss://im.example.com", want: "wss://im.example.com/ws"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := imWebSocketURL(tt.base); got != tt.want {
				t.Fatalf("imWebSocketURL(%q) = %q, want %q", tt.base, got, tt.want)
			}
		})
	}
}

func TestIMProxyDoesNotForwardInDirectMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NativeHandlers{Config: config.Config{IM: config.IMConfig{BaseURL: "https://ws-im.yuelk.com:29443"}}}
	router := gin.New()
	router.Any("/api/im/proxy/*path", handler.imProxy)

	req := httptest.NewRequest(http.MethodPost, "/api/im/proxy/api/v1/chats", nil)
	req.Header.Set("X-IM-Token", "remote-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusGone {
		t.Fatalf("status = %d, want 410, body=%s", rec.Code, rec.Body.String())
	}
	var body struct {
		Code int `json:"code"`
		Data struct {
			APIBase  string `json:"api_base"`
			WSURL    string `json:"ws_url"`
			Mode     string `json:"mode"`
			UseProxy bool   `json:"use_proxy"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.Code != http.StatusGone || body.Data.APIBase != "https://ws-im.yuelk.com:29443" || body.Data.WSURL != "wss://ws-im.yuelk.com:29443/ws" {
		t.Fatalf("unexpected body: %#v", body)
	}
	if body.Data.Mode != "direct" || body.Data.UseProxy {
		t.Fatalf("proxy should be off in response: %#v", body.Data)
	}
}

func TestIMConversationsKeepsBatchQueryCount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	counter := &imQueryCounterLogger{}
	db, err := gorm.Open(sqlite.Open("file:im-conversations-performance?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   counter,
	})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(
		&domain.User{},
		&domain.IMConversation{},
		&domain.IMConversationMember{},
		&domain.IMMessage{},
		&domain.IMMessageReceipt{},
	); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	now := time.Now()
	viewerID := int64(1)
	if err := db.Create(&domain.User{ID: viewerID, UserID: "viewer", Nickname: "Viewer", IsActive: true}).Error; err != nil {
		t.Fatalf("create viewer: %v", err)
	}
	for i := 2; i <= 9; i++ {
		otherID := int64(i)
		if err := db.Create(&domain.User{ID: otherID, UserID: fmt.Sprintf("u%d", i), Nickname: fmt.Sprintf("User %d", i), IsActive: true}).Error; err != nil {
			t.Fatalf("create user %d: %v", i, err)
		}
		convID := int64(i - 1)
		lastID := int64(100 + i)
		conv := domain.IMConversation{ID: convID, Type: "direct", CreatorID: viewerID, LastMessageID: &lastID, CreatedAt: now, UpdatedAt: &now}
		if err := db.Create(&conv).Error; err != nil {
			t.Fatalf("create conversation %d: %v", i, err)
		}
		members := []domain.IMConversationMember{
			{ConversationID: convID, UserID: viewerID, JoinedAt: now},
			{ConversationID: convID, UserID: otherID, JoinedAt: now},
		}
		if err := db.Create(&members).Error; err != nil {
			t.Fatalf("create members %d: %v", i, err)
		}
		olderID := int64(i*100 + 1)
		messages := []domain.IMMessage{
			{ID: olderID, ConversationID: convID, SenderID: otherID, Content: "older", CreatedAt: now},
			{ID: lastID, ConversationID: convID, SenderID: otherID, Content: "latest", CreatedAt: now.Add(time.Minute)},
		}
		if err := db.Create(&messages).Error; err != nil {
			t.Fatalf("create messages %d: %v", i, err)
		}
		if err := db.Create(&domain.IMMessageReceipt{MessageID: lastID, UserID: viewerID}).Error; err != nil {
			t.Fatalf("create receipt %d: %v", i, err)
		}
	}

	recorder := httptest.NewRecorder()
	requestContext, _ := gin.CreateTestContext(recorder)
	requestContext.Request = httptest.NewRequest(http.MethodGet, "/api/im/conversations", nil)
	counter.Reset()
	NativeHandlers{DB: db}.imConversations(requestContext, viewerID)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	if got := counter.Count(); got > 6 {
		t.Fatalf("imConversations query count = %d, want <= 6 to avoid per-conversation lookups", got)
	}
}

func TestIMMarkReadThroughLatestOwnMessageClearsEarlierReceipts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(
		&domain.IMConversation{},
		&domain.IMConversationMember{},
		&domain.IMMessage{},
		&domain.IMMessageReceipt{},
	); err != nil {
		t.Fatalf("migrate database: %v", err)
	}

	now := time.Now()
	conversation := domain.IMConversation{ID: 1, Type: "direct", CreatorID: 1, CreatedAt: now}
	membership := domain.IMConversationMember{ID: 1, ConversationID: 1, UserID: 2, JoinedAt: now}
	messages := []domain.IMMessage{
		{ID: 10, ConversationID: 1, SenderID: 1, Content: "first", CreatedAt: now},
		{ID: 11, ConversationID: 1, SenderID: 1, Content: "second", CreatedAt: now},
		{ID: 12, ConversationID: 1, SenderID: 2, Content: "reply", CreatedAt: now},
	}
	receipts := []domain.IMMessageReceipt{
		{ID: 1, MessageID: 10, UserID: 2},
		{ID: 2, MessageID: 11, UserID: 2},
	}
	if err := db.Create(&conversation).Error; err != nil {
		t.Fatalf("create conversation: %v", err)
	}
	if err := db.Create(&membership).Error; err != nil {
		t.Fatalf("create membership: %v", err)
	}
	if err := db.Create(&messages).Error; err != nil {
		t.Fatalf("create messages: %v", err)
	}
	if err := db.Create(&receipts).Error; err != nil {
		t.Fatalf("create receipts: %v", err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/im/messages/12/read", nil)
	handler := NativeHandlers{DB: db}
	handler.imMarkReceiptFast(context, 2, 12, true)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", recorder.Code, recorder.Body.String())
	}
	var updatedReceipts []domain.IMMessageReceipt
	if err := db.Order("message_id ASC").Find(&updatedReceipts).Error; err != nil {
		t.Fatalf("load receipts: %v", err)
	}
	for _, receipt := range updatedReceipts {
		if receipt.ReadAt == nil || receipt.DeliveredAt == nil {
			t.Fatalf("receipt for message %d was not marked read: %#v", receipt.MessageID, receipt)
		}
	}
	if err := db.First(&membership, membership.ID).Error; err != nil {
		t.Fatalf("reload membership: %v", err)
	}
	if membership.LastReadMessageID == nil || *membership.LastReadMessageID != 12 {
		t.Fatalf("last read message = %v, want 12", membership.LastReadMessageID)
	}
}
