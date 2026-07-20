package main

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

// Hub keeps track of connected clients and broadcasts messages to all of them.
// This is the classic "fan-out" pattern: one upstream data source (the OpenSky
// poller) feeds many downstream WebSocket clients without each client hitting
// the upstream API directly.
type Hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool

	// latestByType holds the most recent broadcast payload per message
	// type, so a brand-new client gets caught up on everything (flights
	// AND satellites) immediately instead of waiting for whichever type
	// happens to broadcast next.
	latestByType map[string][]byte
}

func NewHub() *Hub {
	return &Hub{
		clients:      make(map[*websocket.Conn]bool),
		latestByType: make(map[string][]byte),
	}
}

func (h *Hub) Register(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = true
	cached := make([][]byte, 0, len(h.latestByType))
	for _, v := range h.latestByType {
		cached = append(cached, v)
	}
	h.mu.Unlock()

	for _, data := range cached {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
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

// Broadcast marshals payload once, caches it under msgType for future new
// clients, and pushes it to every currently connected client.
func (h *Hub) Broadcast(msgType string, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Println("marshal error:", err)
		return
	}

	h.mu.Lock()
	h.latestByType[msgType] = data
	clients := make([]*websocket.Conn, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.mu.Unlock()

	for _, c := range clients {
		if err := c.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Println("write error, dropping client:", err)
			h.Unregister(c)
		}
	}
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}
