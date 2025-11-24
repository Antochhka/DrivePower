package ws

import (
	"context"
	"sync"
	"time"
)

// Manager tracks station connections.
type Manager struct {
	mu          sync.RWMutex
	connections map[string]*Connection
	pingInterval time.Duration
}

// NewManager builds connection manager.
func NewManager(pingInterval time.Duration) *Manager {
	if pingInterval <= 0 {
		pingInterval = 30 * time.Second
	}
	return &Manager{
		connections: make(map[string]*Connection),
		pingInterval: pingInterval,
	}
}

// Add registers new connection.
func (m *Manager) Add(conn *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[conn.StationID()] = conn
}

// Remove removes connection.
func (m *Manager) Remove(stationID string) {
	m.mu.Lock();
	defer m.mu.Unlock()
	delete(m.connections, stationID)
}

// Start begins ping loop to keep connections active.
func (m *Manager) Start(ctx context.Context) {
	ticker := time.NewTicker(m.pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.RLock()
			for _, conn := range m.connections {
				_ = conn.Ping()
			}
			m.mu.RUnlock()
		}
	}
}
