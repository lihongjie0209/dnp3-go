package link

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MasterLink implements link layer for DNP3 master station
type MasterLink struct {
	// Configuration
	localAddress  uint16
	remoteAddress uint16
	timeout       time.Duration
	maxRetries    int

	// State
	state         LinkState
	fcb           bool   // Current FCB value
	lastSentFrame *Frame // For retransmission

	// Timing
	retryCount int

	// Callbacks
	dataCallback   DataCallback
	statusCallback StatusCallback

	// Channels for communication
	sendChan     chan []byte // To physical layer
	responseChan chan *Frame // From frame parser

	// Synchronization
	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewMasterLink creates a new master link layer
func NewMasterLink(config LinkLayerConfig) *MasterLink {
	ctx, cancel := context.WithCancel(context.Background())

	return &MasterLink{
		localAddress:   config.LocalAddress,
		remoteAddress:  config.RemoteAddress,
		timeout:        config.Timeout,
		maxRetries:     config.MaxRetries,
		state:          LinkStateIdle,
		fcb:            false,
		dataCallback:   config.DataCallback,
		statusCallback: config.StatusCallback,
		sendChan:       make(chan []byte, 10),
		responseChan:   make(chan *Frame, 10),
		ctx:            ctx,
		cancel:         cancel,
	}
}

// Start starts the link layer
func (m *MasterLink) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != LinkStateIdle {
		return ErrInvalidState
	}

	m.wg.Add(1)
	go m.receiveLoop()

	return nil
}

// Stop stops the link layer
func (m *MasterLink) Stop() error {
	m.cancel()
	m.wg.Wait()
	return nil
}

// GetState returns current link state
func (m *MasterLink) GetState() LinkState {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// IsOnline returns true if link is operational
func (m *MasterLink) IsOnline() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state == LinkStateIdle || m.state == LinkStateWaitACK
}

// GetFCB returns the current frame-count bit under the link state lock.
func (m *MasterLink) GetFCB() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fcb
}

// SetTimeout sets the response timeout
func (m *MasterLink) SetTimeout(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.timeout = duration
}

// SetRetries sets the maximum number of retries
func (m *MasterLink) SetRetries(count int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxRetries = count
}

// ResetLink sends a RESET LINK command
func (m *MasterLink) ResetLink() error {
	m.mu.Lock()
	if m.state != LinkStateIdle {
		m.mu.Unlock()
		return ErrInvalidState
	}
	m.state = LinkStateResetPending
	m.mu.Unlock()

	frame := NewFrame(
		DirectionMasterToOutstation,
		PrimaryFrame,
		FuncResetLink,
		m.remoteAddress,
		m.localAddress,
		nil,
	)

	err := m.sendWithRetry(frame, true)

	m.mu.Lock()
	if err == nil {
		// Reset successful - initialize FCB to 0
		m.fcb = false
		m.state = LinkStateIdle
		m.notifyStatus(LinkStateIdle, nil)
	} else {
		m.state = LinkStateError
		m.notifyStatus(LinkStateError, err)
	}
	m.mu.Unlock()

	return err
}

// ResetUserProcess sends a RESET USER PROCESS command
func (m *MasterLink) ResetUserProcess() error {
	m.mu.Lock()
	if m.state != LinkStateIdle {
		m.mu.Unlock()
		return ErrInvalidState
	}
	m.mu.Unlock()

	frame := NewFrame(
		DirectionMasterToOutstation,
		PrimaryFrame,
		FuncResetUserProcess,
		m.remoteAddress,
		m.localAddress,
		nil,
	)

	return m.sendWithRetry(frame, true)
}

// TestLink sends a TEST LINK STATES command
func (m *MasterLink) TestLink() error {
	m.mu.Lock()
	if m.state != LinkStateIdle {
		m.mu.Unlock()
		return ErrInvalidState
	}
	m.state = LinkStateTestPending
	m.mu.Unlock()

	frame := NewFrame(
		DirectionMasterToOutstation,
		PrimaryFrame,
		FuncTestLinkStates,
		m.remoteAddress,
		m.localAddress,
		nil,
	)

	err := m.sendWithRetry(frame, true)

	m.mu.Lock()
	if err == nil {
		m.state = LinkStateIdle
	} else {
		m.state = LinkStateError
	}
	m.mu.Unlock()

	return err
}

// RequestLinkStatus sends a REQUEST LINK STATUS command
func (m *MasterLink) RequestLinkStatus() error {
	m.mu.Lock()
	if m.state != LinkStateIdle {
		m.mu.Unlock()
		return ErrInvalidState
	}
	m.mu.Unlock()

	frame := NewFrame(
		DirectionMasterToOutstation,
		PrimaryFrame,
		FuncRequestLinkStatus,
		m.remoteAddress,
		m.localAddress,
		nil,
	)

	return m.sendWithRetry(frame, true)
}

// SendConfirmedUserData sends confirmed user data
func (m *MasterLink) SendConfirmedUserData(data []byte) error {
	m.mu.Lock()
	if m.state != LinkStateIdle {
		m.mu.Unlock()
		return ErrInvalidState
	}
	m.state = LinkStateWaitACK

	// Toggle FCB for new message
	m.fcb = !m.fcb
	m.mu.Unlock()

	frame := NewFrame(
		DirectionMasterToOutstation,
		PrimaryFrame,
		FuncUserDataConfirmed,
		m.remoteAddress,
		m.localAddress,
		data,
	)

	// Set FCB and FCV
	frame.SetFCB(m.fcb)

	err := m.sendWithRetry(frame, true)

	m.mu.Lock()
	if err == nil {
		m.state = LinkStateIdle
	} else {
		m.state = LinkStateError
		// On error, toggle FCB back since message wasn't successfully sent
		m.fcb = !m.fcb
	}
	m.mu.Unlock()

	return err
}

// SendUnconfirmedUserData sends unconfirmed user data
func (m *MasterLink) SendUnconfirmedUserData(data []byte) error {
	m.mu.Lock()
	if m.state != LinkStateIdle {
		m.mu.Unlock()
		return ErrInvalidState
	}
	m.mu.Unlock()

	frame := NewFrame(
		DirectionMasterToOutstation,
		PrimaryFrame,
		FuncUserDataUnconfirmed,
		m.remoteAddress,
		m.localAddress,
		data,
	)

	// Send without waiting for response
	return m.transmit(frame)
}

// OnFrameReceived handles received frames
func (m *MasterLink) OnFrameReceived(frame *Frame) error {
	// Validate frame is from correct source
	if frame.Source != m.remoteAddress || frame.Destination != m.localAddress {
		return ErrInvalidAddress
	}

	// Handle unsolicited messages
	if frame.FunctionCode == FuncUserDataUnconfirmed &&
		frame.Dir == DirectionOutstationToMaster {
		// Unsolicited response - pass to upper layer
		if m.dataCallback != nil {
			return m.dataCallback(frame.UserData)
		}
		return nil
	}

	// For other responses, send to response channel
	select {
	case m.responseChan <- frame:
		return nil
	case <-m.ctx.Done():
		return m.ctx.Err()
	default:
		// Channel full - should not happen in normal operation
		return fmt.Errorf("response channel full")
	}
}

// sendWithRetry sends a frame with retry logic
func (m *MasterLink) sendWithRetry(frame *Frame, expectResponse bool) error {
	m.lastSentFrame = frame
	m.retryCount = 0

	for m.retryCount <= m.maxRetries {
		// Send frame
		if err := m.transmit(frame); err != nil {
			return err
		}

		if !expectResponse {
			return nil
		}

		// Wait for response with timeout
		select {
		case response := <-m.responseChan:
			return m.handleResponse(response)

		case <-time.After(m.timeout):
			m.retryCount++
			if m.retryCount > m.maxRetries {
				return ErrMaxRetriesExceeded
			}
			// Retry with SAME frame (same FCB)
			continue

		case <-m.ctx.Done():
			return m.ctx.Err()
		}
	}

	return ErrMaxRetriesExceeded
}

// transmit sends a frame to the physical layer
func (m *MasterLink) transmit(frame *Frame) error {
	data, err := frame.Serialize()
	if err != nil {
		return err
	}

	select {
	case m.sendChan <- data:
		return nil
	case <-m.ctx.Done():
		return m.ctx.Err()
	}
}

// handleResponse processes received response frames
func (m *MasterLink) handleResponse(frame *Frame) error {
	switch frame.FunctionCode {
	case FuncAck:
		// Positive acknowledgment
		return nil

	case FuncNack:
		// Negative acknowledgment - retry after delay
		time.Sleep(100 * time.Millisecond)
		m.retryCount++
		if m.retryCount > m.maxRetries {
			return ErrMaxRetriesExceeded
		}
		// Will retry with same frame
		return m.transmit(m.lastSentFrame)

	case FuncLinkStatusResponse:
		// Link status response
		// TODO: Extract and process status information
		return nil

	case FuncLinkNotFunctioning:
		// Link not functioning
		return fmt.Errorf("link not functioning")

	case FuncLinkNotUsed:
		// Link/address not configured
		return fmt.Errorf("link not used/configured")

	case FuncUserDataUnconfirmed:
		// Response with data - pass to upper layer
		if m.dataCallback != nil {
			return m.dataCallback(frame.UserData)
		}
		return nil

	default:
		return fmt.Errorf("unexpected function code: %d", frame.FunctionCode)
	}
}

// receiveLoop processes incoming frames
func (m *MasterLink) receiveLoop() {
	defer m.wg.Done()

	for {
		select {
		case <-m.ctx.Done():
			return
			// Additional frame processing can be added here
		}
	}
}

// notifyStatus calls the status callback if set
func (m *MasterLink) notifyStatus(state LinkState, err error) {
	if m.statusCallback != nil {
		go m.statusCallback(state, err)
	}
}

// GetSendChannel returns the channel for sending data to physical layer
func (m *MasterLink) GetSendChannel() <-chan []byte {
	return m.sendChan
}
