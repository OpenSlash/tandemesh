// Package hub tracks daemon WebSocket connections and pushes messages to them.
//
// The control plane never dials a daemon: machines connect outbound, which keeps
// them usable behind NAT and firewalls.
package hub

import (
	"encoding/json"
	"sync"
)

// Message is the wire envelope between the control plane and a daemon.
type Message struct {
	Type    string          `json:"type"`
	RunID   string          `json:"runId,omitempty"`
	Fence   int64           `json:"fence,omitempty"`
	Payload json.RawMessage `json:"payload,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// Conn is one daemon connection.
type Conn struct {
	machineID string
	send      chan Message
	done      chan struct{}
	once      sync.Once
}

// MachineID returns the machine this connection belongs to.
func (c *Conn) MachineID() string { return c.machineID }

// Messages is the outbound queue.
func (c *Conn) Messages() <-chan Message { return c.send }

// Done is closed when the connection is torn down.
func (c *Conn) Done() <-chan struct{} { return c.done }

// Send enqueues a message without blocking. It reports false when the queue is
// full or the connection is closed.
func (c *Conn) Send(msg Message) bool {
	select {
	case <-c.done:
		return false
	default:
	}
	select {
	case c.send <- msg:
		return true
	default:
		return false
	}
}

func (c *Conn) close() { c.once.Do(func() { close(c.done) }) }

// Hub is a registry of live daemon connections.
type Hub struct {
	mu    sync.RWMutex
	conns map[string]*Conn
}

// New creates an empty hub.
func New() *Hub {
	return &Hub{conns: make(map[string]*Conn)}
}

// Register adds a connection, replacing and closing any previous one for the
// same machine. The returned function unregisters it.
func (h *Hub) Register(machineID string) (*Conn, func()) {
	c := &Conn{
		machineID: machineID,
		send:      make(chan Message, 64),
		done:      make(chan struct{}),
	}
	h.mu.Lock()
	if old, ok := h.conns[machineID]; ok {
		old.close()
	}
	h.conns[machineID] = c
	h.mu.Unlock()

	return c, func() {
		h.mu.Lock()
		if h.conns[machineID] == c {
			delete(h.conns, machineID)
		}
		h.mu.Unlock()
		c.close()
	}
}

// Send pushes a message to a machine. It reports false when the machine is not
// connected.
func (h *Hub) Send(machineID string, msg Message) bool {
	h.mu.RLock()
	c, ok := h.conns[machineID]
	h.mu.RUnlock()
	if !ok {
		return false
	}
	return c.Send(msg)
}

// Connected reports whether a machine has a live connection.
func (h *Hub) Connected(machineID string) bool {
	h.mu.RLock()
	_, ok := h.conns[machineID]
	h.mu.RUnlock()
	return ok
}

// Count returns the number of live connections.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.conns)
}
