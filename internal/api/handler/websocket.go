package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/qs3c/anal_go_server/internal/pkg/jwt"
	"github.com/qs3c/anal_go_server/internal/pkg/ws"
)

const (
	// 读超时：如果超过这个时间没收到任何消息（包括 pong），认为连接已断开
	pongWait = 60 * time.Second
	// 发送 ping 的间隔，必须小于 pongWait
	pingPeriod = 30 * time.Second
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		// TODO: 生产环境需要验证 Origin
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type WebSocketHandler struct {
	hub       *ws.Hub
	jwtSecret string
}

func NewWebSocketHandler(hub *ws.Hub, jwtSecret string) *WebSocketHandler {
	return &WebSocketHandler{
		hub:       hub,
		jwtSecret: jwtSecret,
	}
}

// Handle WebSocket 连接处理
// GET /api/v1/ws?token=xxx
func (h *WebSocketHandler) Handle(c *gin.Context) {
	// 验证 JWT Token
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	claims, err := jwt.ParseToken(token, h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	// 升级连接
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	client := &ws.Client{
		UserID: claims.UserID,
		Conn:   conn,
	}

	// 设置读超时和 pong 处理
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	h.hub.Register(client)

	// 定时发送 ping 保持连接活跃
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()
		for range ticker.C {
			if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second)); err != nil {
				return
			}
		}
	}()

	// 保持连接，读取消息（主要用于检测断开）
	go func() {
		defer h.hub.Unregister(client)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
	}()
}
