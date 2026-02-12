package ws

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Hub struct {
	// 每个用户可以有多个连接（多标签页、重连等场景）
	clients map[int64]map[*Client]struct{}
	mu      sync.RWMutex
}

type Client struct {
	UserID int64
	Conn   *websocket.Conn
	mu     sync.Mutex // 写锁，防止并发写入
}

type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[int64]map[*Client]struct{}),
	}
}

// Run 不再需要，Register/Unregister 直接用锁处理
func (h *Hub) Run() {
	// 保留空方法，兼容现有调用
}

func (h *Hub) Register(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[client.UserID] == nil {
		h.clients[client.UserID] = make(map[*Client]struct{})
	}
	h.clients[client.UserID][client] = struct{}{}

	total := 0
	for _, conns := range h.clients {
		total += len(conns)
	}
	log.Printf("User %d connected, user_conns: %d, total: %d", client.UserID, len(h.clients[client.UserID]), total)
}

func (h *Hub) Unregister(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if conns, ok := h.clients[client.UserID]; ok {
		delete(conns, client)
		if len(conns) == 0 {
			delete(h.clients, client.UserID)
		}
	}
	log.Printf("User %d disconnected", client.UserID)
}

// SendToUser 向指定用户的所有连接发送消息
func (h *Hub) SendToUser(userID int64, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	h.mu.RLock()
	conns, ok := h.clients[userID]
	if !ok {
		h.mu.RUnlock()
		return nil
	}
	// 复制一份引用，避免长时间持锁
	clients := make([]*Client, 0, len(conns))
	for c := range conns {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	for _, c := range clients {
		c.mu.Lock()
		err := c.Conn.WriteMessage(websocket.TextMessage, data)
		c.mu.Unlock()
		if err != nil {
			log.Printf("SendToUser write error for user %d: %v", userID, err)
		}
	}
	return nil
}

// IsOnline 检查用户是否在线
func (h *Hub) IsOnline(userID int64) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conns, ok := h.clients[userID]
	return ok && len(conns) > 0
}

// ConnectionCount 获取在线连接数
func (h *Hub) ConnectionCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, conns := range h.clients {
		total += len(conns)
	}
	return total
}
