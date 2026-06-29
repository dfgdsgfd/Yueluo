package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"yuem-go/backend-gin/internal/http/response"
)

const (
	imMaxMessageLen = 4000
	imMaxPageSize   = 100
	imWSCloseWait   = 1 * time.Second
	imWSPingPeriod  = 25 * time.Second
	imWSPongWait    = 60 * time.Second
)

var imWSHub = struct {
	sync.RWMutex
	sockets map[int64]map[*websocket.Conn]bool
}{sockets: map[int64]map[*websocket.Conn]bool{}}

var imWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

func (h NativeHandlers) imDispatch(c *gin.Context) {
	user, ok := h.requireMatrixAuth(c)
	if !ok || !h.requireDB(c) {
		return
	}
	path := c.Request.URL.Path
	method := matrixMethod(c)
	switch {
	case path == "/api/im/session" && method == http.MethodPost:
		h.imSession(c, user.ID)
	case strings.HasPrefix(path, "/api/im/proxy/"):
		h.imProxy(c)
	case path == "/api/im/conversations" && method == http.MethodGet:
		h.imConversations(c, user.ID)
	case path == "/api/im/conversations" && method == http.MethodPost:
		h.imCreateConversation(c, user.ID)
	case strings.HasSuffix(path, "/messages") && strings.Contains(path, "/api/im/conversations/") && method == http.MethodGet:
		h.imMessages(c, user.ID)
	case strings.HasSuffix(path, "/messages") && strings.Contains(path, "/api/im/conversations/") && method == http.MethodPost:
		h.imSendMessage(c, user.ID)
	case strings.HasSuffix(path, "/delivered") && strings.Contains(path, "/api/im/messages/") && method == http.MethodPost:
		h.imMarkDelivered(c, user.ID)
	case strings.HasSuffix(path, "/read") && strings.Contains(path, "/api/im/messages/") && method == http.MethodPost:
		h.imMarkRead(c, user.ID)
	case path == "/api/im/sync" && method == http.MethodGet:
		h.imSync(c, user.ID)
	case path == "/api/im/users" && method == http.MethodGet:
		h.imUsers(c, user.ID)
	default:
		response.JSON(c, http.StatusNotFound, response.CodeNotFound, "im route not found", nil)
	}
}

func (h NativeHandlers) IMWebSocket(c *gin.Context) {
	if h.Auth == nil {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}
	token := strings.TrimSpace(c.Query("token"))
	user, failure := h.Auth.Authenticate(c.Request.Context(), token)
	if failure != nil {
		c.AbortWithStatus(failure.Status)
		return
	}
	conn, err := imWSUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	imHubAdd(user.ID, conn)
	go func() {
		ticker := time.NewTicker(imWSPingPeriod)
		done := make(chan struct{})
		defer func() {
			close(done)
			ticker.Stop()
			imHubRemove(user.ID, conn)
			_ = conn.WriteControl(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
				time.Now().Add(imWSCloseWait),
			)
			_ = conn.Close()
		}()
		_ = conn.SetReadDeadline(time.Now().Add(imWSPongWait))
		conn.SetPongHandler(func(string) error {
			return conn.SetReadDeadline(time.Now().Add(imWSPongWait))
		})
		go func() {
			for {
				select {
				case <-ticker.C:
					if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(imPushWriteTimeout)); err != nil {
						_ = conn.Close()
						return
					}
				case <-done:
					return
				}
			}
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func (h NativeHandlers) imSession(c *gin.Context, uid int64) {
	if h.Config.IM.HMACSecret == "" {
		response.JSON(c, http.StatusInternalServerError, response.CodeError, "IM 服务未配置（IM_HMAC_SECRET 缺失）", nil)
		return
	}
	sig, ts, err := h.imTokenForUser(c, uid)
	if err != nil {
		msg := "IM 服务连接失败: " + err.Error()
		zap.L().Error(msg)
		response.JSON(c, http.StatusInternalServerError, response.CodeError, msg, nil)
		return
	}
	apiBase := h.imAPIBase()
	mode := h.Config.IM.Mode
	if mode == "" {
		mode = "direct"
	}
	u := imWebSocketURL(apiBase) + "?uid=" + strconv.FormatInt(uid, 10) + "&ts=" + ts + "&sig=" + sig
	writeSuccess(c, "ok", gin.H{
		"uid":       uid,
		"ts":        ts,
		"sig":       sig,
		"api_base":  apiBase,
		"ws_url":    u,
		"mode":      mode,
		"use_proxy": mode == "proxy",
	})
}

func (h NativeHandlers) imProxy(c *gin.Context) {
	apiBase := h.imAPIBase()
	response.JSON(c, http.StatusGone, http.StatusGone, "IM 已切换为外部直连，请调用 /api/im/session 获取 token、api_base 和 ws_url 后直接访问外部 IM 服务", gin.H{
		"api_base":  apiBase,
		"ws_url":    imWebSocketURL(apiBase),
		"mode":      "direct",
		"use_proxy": false,
	})
}
