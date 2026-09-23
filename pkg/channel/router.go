package channel

import (
	"fmt"
	"sync"

	"github.com/lihongjie0209/dnp3-go/pkg/link"
)

// Session represents a master or outstation session on a channel
type Session interface {
	// OnReceive is called when a link frame is received for this session
	OnReceive(frame *link.Frame) error

	// LinkAddress returns the link address for this session
	LinkAddress() uint16

	// Type returns the type of session
	Type() SessionType
}

// SessionWithConnectionState is an optional interface that sessions can implement
// to receive connection state notifications
type SessionWithConnectionState interface {
	Session

	// OnConnectionEstablished is called when the underlying connection is established
	OnConnectionEstablished()

	// OnConnectionLost is called when the underlying connection is lost
	OnConnectionLost()
}

// SessionType identifies the type of session
type SessionType int

const (
	SessionTypeMaster SessionType = iota
	SessionTypeOutstation
)

// String returns string representation of SessionType
func (t SessionType) String() string {
	switch t {
	case SessionTypeMaster:
		return "Master"
	case SessionTypeOutstation:
		return "Outstation"
	default:
		return "Unknown"
	}
}

// Router routes link frames to appropriate sessions based on address
// Supports multi-drop configurations
type Router struct {
	sessions map[uint16]Session // Key: link address
	mu       sync.RWMutex
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{
		sessions: make(map[uint16]Session),
	}
}

// AddSession adds a session to the router
func (r *Router) AddSession(session Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	addr := session.LinkAddress()

	// Check if address is already in use
	if _, exists := r.sessions[addr]; exists {
		return fmt.Errorf("session with address %d already exists", addr)
	}

	r.sessions[addr] = session
	return nil
}

// RemoveSession removes a session from the router
func (r *Router) RemoveSession(address uint16) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.sessions, address)
}

// Route routes a frame to the appropriate session
// Returns error if no session found for address
func (r *Router) Route(frame *link.Frame) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Determine target address based on frame direction
	var targetAddr uint16
	if frame.Dir == link.DirectionMasterToOutstation {
		// Frame from master, route to outstation
		targetAddr = frame.Destination
	} else {
		// Frame from outstation, route to master
		targetAddr = frame.Destination
	}

	// Find session
	session, exists := r.sessions[targetAddr]
	if !exists {
		return fmt.Errorf("no session found for address %d", targetAddr)
	}

	// Deliver to session
	return session.OnReceive(frame)
}

// GetSession returns a session by address
func (r *Router) GetSession(address uint16) (Session, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[address]
	return session, exists
}

// GetSessionCount returns the number of active sessions
func (r *Router) GetSessionCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.sessions)
}

// Clear removes all sessions
func (r *Router) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions = make(map[uint16]Session)
}

// NotifyConnectionEstablished notifies all sessions that support connection state
func (r *Router) NotifyConnectionEstablished() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, session := range r.sessions {
		if cs, ok := session.(SessionWithConnectionState); ok {
			cs.OnConnectionEstablished()
		}
	}
}

// NotifyConnectionLost notifies all sessions that support connection state
func (r *Router) NotifyConnectionLost() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, session := range r.sessions {
		if cs, ok := session.(SessionWithConnectionState); ok {
			cs.OnConnectionLost()
		}
	}
}
