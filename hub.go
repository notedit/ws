package ws

import (
	"sync"
)

type Hub struct {
	// Registered Conn.
	clients map[string]*Conn

	mu sync.Mutex
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[string]*Conn),
		mu:      sync.Mutex{},
	}
}

func (h *Hub) Register(conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[conn.id] = conn
}

func (h *Hub) UnRegister(conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.clients, conn.id)
}

func (h *Hub) Conn(id string) *Conn {
	h.mu.Lock()
	defer h.mu.Unlock()

	conn, _ := h.clients[id]

	return conn
}
