package ws

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	connections map[int64]*websocket.Conn
	mu          sync.RWMutex
	register    chan *Client
	unregister  chan *Client
	broadcast   chan *Message
}

type Client struct {
	UserID int64
	Conn   *websocket.Conn
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewHub() *Hub {
	return &Hub{
		connections: make(map[int64]*websocket.Conn),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan *Message, 256),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			// 关闭旧连接
			if oldConn, ok := h.connections[client.UserID]; ok {
				oldConn.Close()
			}
			h.connections[client.UserID] = client.Conn
			h.mu.Unlock()
			log.Printf("User %d connected, total connections: %d", client.UserID, len(h.connections))

		case client := <-h.unregister:
			h.mu.Lock()
			if conn, ok := h.connections[client.UserID]; ok && conn == client.Conn {
				delete(h.connections, client.UserID)
				conn.Close()
			}
			h.mu.Unlock()
			log.Printf("User %d disconnected", client.UserID)
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// SendToUser 向指定用户发送消息
func (h *Hub) SendToUser(userID int64, msg *Message) error {
	h.mu.RLock()
	conn, ok := h.connections[userID]
	h.mu.RUnlock()

	if !ok {
		return nil // 用户不在线，忽略
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return conn.WriteMessage(websocket.TextMessage, data)
}

// IsOnline 检查用户是否在线
func (h *Hub) IsOnline(userID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.connections[userID]
	return ok
}

// ConnectionCount 获取在线连接数
func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.connections)
}
