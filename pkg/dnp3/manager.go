package dnp3

import (
	"fmt"
	"sync"

	"github.com/lihongjie0209/dnp3-go/pkg/channel"
	"github.com/lihongjie0209/dnp3-go/pkg/internal/logger"
)

// Manager is the root object for DNP3 operations
// It manages channels and provides the main API entry point
type Manager struct {
	channels map[string]*channel.Channel
	mu       sync.RWMutex
	logger   logger.Logger
}

// NewManager creates a new DNP3 manager
func NewManager() *Manager {
	return NewManagerWithLogger(logger.GetDefault())
}

// NewManagerWithLogger creates a new DNP3 manager with custom logger
func NewManagerWithLogger(log logger.Logger) *Manager {
	if log == nil {
		log = logger.NewNoOpLogger()
	}

	return &Manager{
		channels: make(map[string]*channel.Channel),
		logger:   log,
	}
}

// AddChannel creates a new channel with the given physical channel
// The physical channel must implement the PhysicalChannel interface
func (m *Manager) AddChannel(id string, physical channel.PhysicalChannel) (Channel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if channel already exists
	if _, exists := m.channels[id]; exists {
		return nil, fmt.Errorf("channel %s already exists", id)
	}

	// Create channel
	ch := channel.New(id, physical, m.logger)

	// Open channel
	if err := ch.Open(); err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	m.channels[id] = ch
	m.logger.Info("Manager: Added channel %s", id)

	return &channelImpl{channel: ch, manager: m}, nil
}

// RemoveChannel removes a channel
func (m *Manager) RemoveChannel(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ch, exists := m.channels[id]
	if !exists {
		return fmt.Errorf("channel %s not found", id)
	}

	// Close channel
	if err := ch.Close(); err != nil {
		m.logger.Error("Error closing channel %s: %v", id, err)
	}

	delete(m.channels, id)
	m.logger.Info("Manager: Removed channel %s", id)
	return nil
}

// GetChannel returns a channel by ID
func (m *Manager) GetChannel(id string) (Channel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ch, exists := m.channels[id]
	if !exists {
		return nil, false
	}

	return &channelImpl{channel: ch, manager: m}, true
}

// Shutdown shuts down the manager and all channels
func (m *Manager) Shutdown() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Info("Manager: Shutting down")

	// Close all channels
	for id, ch := range m.channels {
		if err := ch.Close(); err != nil {
			m.logger.Error("Error closing channel %s: %v", id, err)
		}
	}

	m.channels = make(map[string]*channel.Channel)
	m.logger.Info("Manager: Shutdown complete")
	return nil
}

// ChannelCount returns the number of channels
func (m *Manager) ChannelCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.channels)
}

// SetLogger sets the logger for the manager
func (m *Manager) SetLogger(log logger.Logger) {
	m.logger = log
}

// createMaster creates a master (internal method to avoid circular dependency)
func (m *Manager) createMaster(config MasterConfig, callbacks MasterCallbacks, ch *channel.Channel) (Master, error) {
	// Lazy import to avoid circular dependency
	// This will be resolved by the master package's factory
	return newMaster(config, callbacks, ch, m.logger)
}

// createOutstation creates an outstation (internal method to avoid circular dependency)
func (m *Manager) createOutstation(config OutstationConfig, callbacks OutstationCallbacks, ch *channel.Channel) (Outstation, error) {
	return newOutstation(config, callbacks, ch, m.logger)
}
