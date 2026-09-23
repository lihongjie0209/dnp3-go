package channel

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/lihongjie0209/dnp3-go/pkg/internal/logger"
	"github.com/lihongjie0209/dnp3-go/pkg/link"
)

var (
	ErrChannelClosed = errors.New("channel is closed")
	ErrChannelOpen   = errors.New("channel is already open")
)

// Channel manages protocol stack and multiple sessions
type Channel struct {
	id              string
	physicalChannel PhysicalChannel
	router          *Router
	stats           *Statistics
	logger          logger.Logger

	// State
	state   ChannelState
	stateMu sync.RWMutex

	// Concurrency
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Write queue for serializing writes
	writeQueue chan *writeRequest
}

// writeRequest represents a write request
type writeRequest struct {
	data []byte
	resp chan error
}

// New creates a new channel
func New(id string, physical PhysicalChannel, log logger.Logger) *Channel {
	if log == nil {
		log = logger.NewNoOpLogger()
	}

	ctx, cancel := context.WithCancel(context.Background())

	c := &Channel{
		id:              id,
		physicalChannel: physical,
		router:          NewRouter(),
		stats:           NewStatistics(),
		logger:          log,
		state:           ChannelStateClosed,
		ctx:             ctx,
		cancel:          cancel,
		writeQueue:      make(chan *writeRequest, 100),
	}

	// Set channel as connection state listener
	physical.SetConnectionStateListener(c)

	return c
}

// ID returns the channel ID
func (c *Channel) ID() string {
	return c.id
}

// Open opens the channel and starts processing
func (c *Channel) Open() error {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	if c.state == ChannelStateOpen {
		return ErrChannelOpen
	}

	c.state = ChannelStateOpen
	c.logger.Info("Channel %s opening", c.id)

	// Start read loop
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.readLoop()
	}()

	// Start write loop
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.writeLoop()
	}()

	c.logger.Info("Channel %s opened", c.id)
	return nil
}

// Close closes the channel
func (c *Channel) Close() error {
	c.stateMu.Lock()
	if c.state == ChannelStateClosed {
		c.stateMu.Unlock()
		return nil
	}
	c.state = ChannelStateClosed
	c.stateMu.Unlock()

	c.logger.Info("Channel %s closing", c.id)

	// Cancel context to stop goroutines
	c.cancel()

	// Close physical channel
	if err := c.physicalChannel.Close(); err != nil {
		c.logger.Error("Error closing physical channel: %v", err)
	}

	// Wait for goroutines to finish
	c.wg.Wait()

	c.logger.Info("Channel %s closed", c.id)
	return nil
}

// readLoop continuously reads from physical channel
func (c *Channel) readLoop() {
	c.logger.Debug("Channel %s read loop started", c.id)
	defer c.logger.Debug("Channel %s read loop stopped", c.id)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		// Read from physical channel
		data, err := c.physicalChannel.Read(c.ctx)
		if err != nil {
			if c.ctx.Err() != nil {
				// Context cancelled, normal shutdown
				return
			}
			c.logger.Error("Channel %s read error: %v", c.id, err)
			c.stats.BadLinkFrame()
			continue
		}

		// Log received frame if debugging enabled
		logger.LogFrameReceived(c.id, data)

		// Parse link frame
		frame, _, err := link.Parse(data)
		if err != nil {
			c.logger.Error("Channel %s parse error: %v (received %d bytes, expected >= 10)", c.id, err, len(data))
			if len(data) < 20 {
				c.logger.Debug("Channel %s received data: %X", c.id, data)
			}
			c.stats.BadLinkFrame()
			continue
		}

		c.stats.LinkFrameRx()
		c.logger.Debug("Channel %s received frame: %s", c.id, frame)

		// Route to appropriate session
		if err := c.router.Route(frame); err != nil {
			c.logger.Warn("Channel %s routing error: %v", c.id, err)
		}
	}
}

// writeLoop processes write requests
func (c *Channel) writeLoop() {
	c.logger.Debug("Channel %s write loop started", c.id)
	defer c.logger.Debug("Channel %s write loop stopped", c.id)

	for {
		select {
		case <-c.ctx.Done():
			// Drain remaining requests with error
			for {
				select {
				case req := <-c.writeQueue:
					req.resp <- ErrChannelClosed
				default:
					return
				}
			}

		case req := <-c.writeQueue:
			// Log sent frame if debugging enabled
			logger.LogFrameSent(c.id, req.data)

			// Write to physical channel
			err := c.physicalChannel.Write(c.ctx, req.data)
			if err != nil {
				c.logger.Error("Channel %s write error: %v", c.id, err)
			} else {
				c.stats.LinkFrameTx()
			}
			req.resp <- err
		}
	}
}

// Write writes data to the channel (used by sessions)
func (c *Channel) Write(data []byte) error {
	c.stateMu.RLock()
	if c.state != ChannelStateOpen {
		c.stateMu.RUnlock()
		return ErrChannelClosed
	}
	c.stateMu.RUnlock()

	req := &writeRequest{
		data: data,
		resp: make(chan error, 1),
	}

	select {
	case c.writeQueue <- req:
		return <-req.resp
	case <-c.ctx.Done():
		return ErrChannelClosed
	}
}

// AddSession adds a session to the channel
func (c *Channel) AddSession(session Session) error {
	if err := c.router.AddSession(session); err != nil {
		return err
	}

	c.stats.SetActiveSessions(uint64(c.router.GetSessionCount()))
	c.logger.Info("Channel %s: Added %s session at address %d", c.id, session.Type(), session.LinkAddress())
	return nil
}

// RemoveSession removes a session from the channel
func (c *Channel) RemoveSession(address uint16) {
	c.router.RemoveSession(address)
	c.stats.SetActiveSessions(uint64(c.router.GetSessionCount()))
	c.logger.Info("Channel %s: Removed session at address %d", c.id, address)
}

// GetStatistics returns channel statistics
func (c *Channel) GetStatistics() *Statistics {
	return c.stats
}

// GetPhysicalStatistics returns physical channel statistics
func (c *Channel) GetPhysicalStatistics() TransportStats {
	return c.physicalChannel.Statistics()
}

// State returns the current channel state
func (c *Channel) State() ChannelState {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return c.state
}

// String returns string representation of channel
func (c *Channel) String() string {
	return fmt.Sprintf("Channel{ID=%s, State=%s, Sessions=%d}",
		c.id, c.State(), c.router.GetSessionCount())
}

// OnConnectionEstablished is called when a connection is established (implements ConnectionStateListener)
func (c *Channel) OnConnectionEstablished() {
	c.logger.Info("Channel %s: Connection established", c.id)
	// Notify all sessions to reset their transport layers
	c.router.NotifyConnectionEstablished()
}

// OnConnectionLost is called when a connection is lost (implements ConnectionStateListener)
func (c *Channel) OnConnectionLost() {
	c.logger.Info("Channel %s: Connection lost", c.id)
	// Notify all sessions about connection loss
	c.router.NotifyConnectionLost()
}
