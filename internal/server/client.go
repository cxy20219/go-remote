package server

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a connected agent
type Client struct {
	ClientID string           `json:"client_id"`
	Hostname string           `json:"hostname"`
	OS       string           `json:"os"`
	Conn     *websocket.Conn  `json:"-"`
	RegTime  time.Time        `json:"reg_time"`
	LastPing time.Time        `json:"last_ping"`
	Status   string           `json:"status"` // online/offline
}

// ClientManager manages connected clients
type ClientManager struct {
	mu      sync.RWMutex
	clients map[string]*Client // clientID -> Client
}

// NewClientManager creates a new ClientManager
func NewClientManager() *ClientManager {
	return &ClientManager{
		clients: make(map[string]*Client),
	}
}

// Add adds a client to the manager
func (m *ClientManager) Add(client *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	client.Status = "online"
	client.RegTime = time.Now()
	client.LastPing = time.Now()
	m.clients[client.ClientID] = client
}

// Remove removes a client by ID
func (m *ClientManager) Remove(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, clientID)
}

// Get retrieves a client by ID
func (m *ClientManager) Get(clientID string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[clientID]
	return client, ok
}

// List returns all clients
func (m *ClientManager) List() []*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	clients := make([]*Client, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}
	return clients
}

// UpdatePing updates the last ping time for a client
func (m *ClientManager) UpdatePing(clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if client, ok := m.clients[clientID]; ok {
		client.LastPing = time.Now()
	}
}

// Broadcast sends a message to all connected clients
func (m *ClientManager) Broadcast(data []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, client := range m.clients {
		if err := client.Conn.WriteMessage(websocket.TextMessage, data); err != nil {
			slog.Warn("failed to broadcast to client", "client_id", client.ClientID, "error", err)
		}
	}
	return nil
}

// SendTo sends a message to a specific client
func (m *ClientManager) SendTo(clientID string, data []byte) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.clients[clientID]
	if !ok {
		return nil
	}
	return client.Conn.WriteMessage(websocket.TextMessage, data)
}
