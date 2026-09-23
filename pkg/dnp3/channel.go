package dnp3

import (
	"github.com/lihongjie0209/dnp3-go/pkg/channel"
)

// Channel is the public interface for a DNP3 channel
type Channel interface {
	// AddMaster adds a master session to this channel
	AddMaster(config MasterConfig, callbacks MasterCallbacks) (Master, error)

	// AddOutstation adds an outstation session to this channel
	AddOutstation(config OutstationConfig, callbacks OutstationCallbacks) (Outstation, error)

	// Shutdown closes the channel and all sessions
	Shutdown() error

	// Statistics returns channel statistics
	Statistics() ChannelStatistics
}

// ChannelStatistics provides channel-level statistics
type ChannelStatistics struct {
	LinkFramesTx    uint64 // Link frames transmitted
	LinkFramesRx    uint64 // Link frames received
	BadLinkFrames   uint64 // Bad link frames
	CRCErrors       uint64 // CRC errors
	TransportTx     uint64 // Transport segments transmitted
	TransportRx     uint64 // Transport segments received
	TransportErrors uint64 // Transport errors
	ActiveSessions  uint64 // Number of active sessions
	PhysicalBytesTx uint64 // Physical bytes transmitted
	PhysicalBytesRx uint64 // Physical bytes received
}

// channelImpl implements the Channel interface
type channelImpl struct {
	channel *channel.Channel
	manager *Manager
}

// AddMaster adds a master session to this channel
func (c *channelImpl) AddMaster(config MasterConfig, callbacks MasterCallbacks) (Master, error) {
	// Import master package to avoid circular dependency
	// Create master through internal factory
	return c.manager.createMaster(config, callbacks, c.channel)
}

// AddOutstation adds an outstation session to this channel
func (c *channelImpl) AddOutstation(config OutstationConfig, callbacks OutstationCallbacks) (Outstation, error) {
	// Create outstation through internal factory
	return c.manager.createOutstation(config, callbacks, c.channel)
}

// Shutdown closes the channel
func (c *channelImpl) Shutdown() error {
	return c.manager.RemoveChannel(c.channel.ID())
}

// Statistics returns channel statistics
func (c *channelImpl) Statistics() ChannelStatistics {
	stats := c.channel.GetStatistics()
	physStats := c.channel.GetPhysicalStatistics()

	return ChannelStatistics{
		LinkFramesTx:    stats.GetLinkFramesTx(),
		LinkFramesRx:    stats.GetLinkFramesRx(),
		BadLinkFrames:   stats.GetBadLinkFrames(),
		CRCErrors:       stats.GetCRCErrors(),
		TransportTx:     stats.GetTransportTx(),
		TransportRx:     stats.GetTransportRx(),
		TransportErrors: stats.GetTransportErrors(),
		ActiveSessions:  stats.GetActiveSessions(),
		PhysicalBytesTx: physStats.BytesSent,
		PhysicalBytesRx: physStats.BytesReceived,
	}
}
