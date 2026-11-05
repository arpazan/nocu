// Package ws pushes broker events to connected WebSocket clients.
package ws

import (
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

// Client is one connected WebSocket session.
type Client struct {
	UserID uuid.UUID
	send   chan []byte
	once   sync.Once
}

func newClient(userID uuid.UUID, buffer int) *Client {
	return &Client{UserID: userID, send: make(chan []byte, buffer)}
}

// close is idempotent so the hub and the pumps can both call it.
func (c *Client) close() {
	c.once.Do(func() { close(c.send) })
}

// Hub tracks clients and fans messages out to them.
type Hub struct {
	mu      sync.RWMutex
	clients map[*Client]struct{}
	log     *slog.Logger
	buffer  int
}

// NewHub builds an empty hub. buffer is the per-client outbound queue length.
func NewHub(log *slog.Logger, buffer int) *Hub {
	if buffer <= 0 {
		buffer = 64
	}
	return &Hub{clients: map[*Client]struct{}{}, log: log.With("component", "ws"), buffer: buffer}
}

// Register adds a client and returns it.
func (h *Hub) Register(userID uuid.UUID) *Client {
	c := newClient(userID, h.buffer)
	h.mu.Lock()
	h.clients[c] = struct{}{}
	n := len(h.clients)
	h.mu.Unlock()
	h.log.Debug("client connected", "user_id", userID, "clients", n)
	return c
}

// Unregister removes a client and closes its queue.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	_, ok := h.clients[c]
	delete(h.clients, c)
	n := len(h.clients)
	h.mu.Unlock()
	if ok {
		c.close()
		h.log.Debug("client disconnected", "user_id", c.UserID, "clients", n)
	}
}

// Broadcast queues msg for every client. Clients whose queue is full are dropped
// so one slow consumer cannot stall the rest.
func (h *Hub) Broadcast(msg []byte) {
	h.mu.RLock()
	targets := make([]*Client, 0, len(h.clients))
	for c := range h.clients {
		targets = append(targets, c)
	}
	h.mu.RUnlock()

	var slow []*Client
	for _, c := range targets {
		select {
		case c.send <- msg:
		default:
			slow = append(slow, c)
		}
	}
	for _, c := range slow {
		h.log.Warn("dropping slow client", "user_id", c.UserID)
		h.Unregister(c)
	}
}

// Count returns the number of connected clients.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// CloseAll disconnects every client; used on shutdown.
func (h *Hub) CloseAll() {
	h.mu.Lock()
	clients := h.clients
	h.clients = map[*Client]struct{}{}
	h.mu.Unlock()
	for c := range clients {
		c.close()
	}
}
