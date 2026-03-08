package httpx

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type eventHub struct {
	mu      sync.Mutex
	clients map[*websocket.Conn]struct{}
}

var hub = &eventHub{
	clients: make(map[*websocket.Conn]struct{}),
}

func (h *eventHub) register(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()
}

func (h *eventHub) unregister(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
}

func (h *eventHub) broadcast(eventType string, payload interface{}) {
	msg, err := json.Marshal(map[string]interface{}{
		"type":    eventType,
		"payload": payload,
	})
	if err != nil {
		log.Printf("ws broadcast marshal error: %v", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			delete(h.clients, conn)
			conn.Close()
		}
	}
}

// BroadcastEvent sends a typed event to all connected WebSocket clients.
func BroadcastEvent(eventType string, payload interface{}) {
	hub.broadcast(eventType, payload)
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}
	hub.register(conn)
	defer func() {
		hub.unregister(conn)
		conn.Close()
	}()

	// Read loop — keeps connection alive and detects client disconnect.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
