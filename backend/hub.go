package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub keeps track of connected clients and broadcasts messages to all of
// them. Multiple independent upstream sources (flights, ships, satellites,
// weather) each broadcast on their own timer.
//
// Critical constraint: gorilla/websocket allows at most ONE concurrent
// writer per connection. With several independent goroutines broadcasting
// on different schedules, two can call WriteMessage on the same connection
// at the same instant — gorilla panics instead of queuing it. Every
// connection gets its own mutex here, and every write goes through it.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]*sync.Mutex // per-connection write lock

	latestByType map[string][]byte
}

func NewHub() *Hub {
	return &Hub{
		clients:      make(map[*websocket.Conn]*sync.Mutex),
		latestByType: make(map[string][]byte),
	}
}

func (h *Hub) Register(conn *websocket.Conn) {
	writeMu := &sync.Mutex{}

	h.mu.Lock()
	h.clients[conn] = writeMu
	cached := make([][]byte, 0, len(h.latestByType))
	for _, v := range h.latestByType {
		cached = append(cached, v)
	}
	h.mu.Unlock()

	for _, data := range cached {
		if err := writeToClient(conn, writeMu, data); err != nil {
			log.Println("failed to send initial snapshot:", err)
		}
	}
}

func (h *Hub) Unregister(conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, conn)
	conn.Close()
}

// writeToClient serializes every write to a given connection through that
// connection's own mutex — the actual fix for the panic.
func writeToClient(conn *websocket.Conn, writeMu *sync.Mutex, data []byte) error {
	writeMu.Lock()
	defer writeMu.Unlock()
	return conn.WriteMessage(websocket.TextMessage, data)
}

// Broadcast is safe to call concurrently from multiple goroutines.
func (h *Hub) Broadcast(msgType string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Println("marshal error:", err)
		return
	}

	h.mu.Lock()
	h.latestByType[msgType] = data
	type client struct {
		conn    *websocket.Conn
		writeMu *sync.Mutex
	}
	clients := make([]client, 0, len(h.clients))
	for c, m := range h.clients {
		clients = append(clients, client{c, m})
	}
	h.mu.Unlock()

	for _, c := range clients {
		if err := writeToClient(c.conn, c.writeMu, data); err != nil {
			log.Println("write error, dropping client:", err)
			h.Unregister(c.conn)
		}
	}
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
