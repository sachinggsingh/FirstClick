package realtime

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type SeatsHub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
}

func NewSeatsHub() *SeatsHub {
	return &SeatsHub{
		clients: make(map[*websocket.Conn]struct{}),
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // allow same-origin & dev usage
}

func (h *SeatsHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "websocket upgrade failed", http.StatusBadRequest)
		return
	}

	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()

	// Read loop: we ignore messages, but we must read to detect client disconnects.
	go func() {
		defer func() {
			h.mu.Lock()
			delete(h.clients, conn)
			h.mu.Unlock()
			_ = conn.Close()
		}()

		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()
}

func (h *SeatsHub) BroadcastSeatsUpdated(movieID string) {
	payload := map[string]any{
		"type":     "SEATS_UPDATED",
		"movie_id": movieID,
		"ts":       time.Now().UnixMilli(),
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {
		_ = conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
		if err := conn.WriteMessage(websocket.TextMessage, b); err != nil {
			// best-effort: a stale client will get cleaned up by its read loop
			continue
		}
	}
}
