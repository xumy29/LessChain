package messageHub

import (
	"net"
	"sync"
)

// ConnectionsMap is a safe map for concurrent use.
type ConnectionsMap struct {
	sync.RWMutex
	connections map[uint32]net.Conn
}

// NewConnectionsMap creates a new ConnectionsMap.
func NewConnectionsMap() *ConnectionsMap {
	return &ConnectionsMap{
		connections: make(map[uint32]net.Conn),
	}
}

// Add adds a connection to the map.
func (cm *ConnectionsMap) Add(key uint32, conn net.Conn) {
	cm.Lock()
	cm.connections[key] = conn
	cm.Unlock()
}

// Get retrieves a connection by key.
func (cm *ConnectionsMap) Get(key uint32) (net.Conn, bool) {
	cm.RLock()
	conn, ok := cm.connections[key]
	cm.RUnlock()
	return conn, ok
}

// Remove removes a connection by key.
func (cm *ConnectionsMap) Remove(key uint32) {
	cm.Lock()
	delete(cm.connections, key)
	cm.Unlock()
}
