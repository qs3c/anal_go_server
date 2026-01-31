package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func TestNewHub(t *testing.T) {
	hub := NewHub()

	assert.NotNil(t, hub)
	assert.NotNil(t, hub.connections)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.broadcast)
	assert.Equal(t, 0, len(hub.connections))
}

func TestHub_ConnectionCount_Empty(t *testing.T) {
	hub := NewHub()

	count := hub.ConnectionCount()
	assert.Equal(t, 0, count)
}

func TestHub_IsOnline_NoConnections(t *testing.T) {
	hub := NewHub()

	online := hub.IsOnline(123)
	assert.False(t, online)
}

func TestHub_SendToUser_UserNotOnline(t *testing.T) {
	hub := NewHub()

	msg := &Message{
		Type: "test",
		Data: map[string]string{"key": "value"},
	}

	// Should return nil (not error) for offline user
	err := hub.SendToUser(123, msg)
	assert.NoError(t, err)
}

func TestMessage_Structure(t *testing.T) {
	msg := &Message{
		Type: "progress",
		Data: map[string]interface{}{
			"job_id":   123,
			"progress": 50,
		},
	}

	assert.Equal(t, "progress", msg.Type)
	data := msg.Data.(map[string]interface{})
	assert.Equal(t, 123, data["job_id"])
	assert.Equal(t, 50, data["progress"])
}

func TestClient_Structure(t *testing.T) {
	client := &Client{
		UserID: 456,
		Conn:   nil,
	}

	assert.Equal(t, int64(456), client.UserID)
	assert.Nil(t, client.Conn)
}

func TestHub_WithRealWebSocket(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}

		client := &Client{
			UserID: 100,
			Conn:   conn,
		}
		hub.Register(client)

		// Keep connection open for a bit
		time.Sleep(100 * time.Millisecond)

		hub.Unregister(client)
	}))
	defer server.Close()

	// Connect via websocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	// Verify user is online
	assert.True(t, hub.IsOnline(100))
	assert.Equal(t, 1, hub.ConnectionCount())

	// Wait for unregistration
	time.Sleep(100 * time.Millisecond)
}

func TestHub_SendToUser_WithConnection(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := &Client{
			UserID: 200,
			Conn:   conn,
		}
		hub.Register(client)

		// Keep connection open
		time.Sleep(500 * time.Millisecond)
	}))
	defer server.Close()

	// Connect via websocket
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn.Close()

	// Wait for registration
	time.Sleep(50 * time.Millisecond)

	// Send message to user
	msg := &Message{
		Type: "notification",
		Data: map[string]string{"content": "Hello"},
	}
	err = hub.SendToUser(200, msg)
	assert.NoError(t, err)

	// Read message on client side
	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, received, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Contains(t, string(received), "notification")
	assert.Contains(t, string(received), "Hello")
}

func TestHub_ReplaceExistingConnection(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	connectionCount := 0

	// Create test server that tracks connections
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		connectionCount++

		client := &Client{
			UserID: 300, // Same user ID
			Conn:   conn,
		}
		hub.Register(client)

		// Keep connection open
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	// Connect first time
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn1, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn1.Close()

	time.Sleep(50 * time.Millisecond)

	// Connect second time with same user
	conn2, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer conn2.Close()

	time.Sleep(50 * time.Millisecond)

	// Should still have only 1 connection for user 300
	assert.Equal(t, 1, hub.ConnectionCount())
	assert.True(t, hub.IsOnline(300))
}

func TestHub_MultipleUsers(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	var userID int64 = 0

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		userID++
		client := &Client{
			UserID: userID,
			Conn:   conn,
		}
		hub.Register(client)

		// Keep connection open
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	// Connect multiple users
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	var conns []*websocket.Conn
	for i := 0; i < 3; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		conns = append(conns, conn)
	}

	// Clean up connections
	defer func() {
		for _, conn := range conns {
			conn.Close()
		}
	}()

	time.Sleep(100 * time.Millisecond)

	// Should have 3 connections
	assert.Equal(t, 3, hub.ConnectionCount())
	assert.True(t, hub.IsOnline(1))
	assert.True(t, hub.IsOnline(2))
	assert.True(t, hub.IsOnline(3))
	assert.False(t, hub.IsOnline(4))
}
