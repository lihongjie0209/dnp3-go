package master

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/app"
	"github.com/lihongjie0209/dnp3-go/pkg/channel"
	"github.com/lihongjie0209/dnp3-go/pkg/internal/logger"
	"github.com/lihongjie0209/dnp3-go/pkg/internal/queue"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
)

var (
	ErrMasterDisabled = errors.New("master is disabled")
	ErrTimeout        = errors.New("operation timeout")
)

// MasterConfig and callback interfaces moved here to avoid circular import
// These will be type-aliased or wrapped in dnp3 package

// master implements the Master interface
type master struct {
	config    MasterConfig
	callbacks MasterCallbacks
	logger    logger.Logger

	// Session
	session *session

	// Task management
	taskQueue  *queue.PriorityQueue
	scans      map[int]*PeriodicScan
	nextScanID int
	scansMu    sync.RWMutex

	// State
	enabled    bool
	seqCounter *app.SequenceCounter
	lastIIN    types.IIN
	stateMu    sync.RWMutex

	// Concurrency
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Response handling
	pendingResp chan *app.APDU
	pendingMu   sync.Mutex
}

// New creates a new master
func New(config MasterConfig, callbacks MasterCallbacks, ch *channel.Channel, log logger.Logger) (*master, error) {
	if log == nil {
		log = logger.NewNoOpLogger()
	}

	ctx, cancel := context.WithCancel(context.Background())

	m := &master{
		config:      config,
		callbacks:   callbacks,
		logger:      log,
		taskQueue:   queue.NewPriorityQueue(),
		scans:       make(map[int]*PeriodicScan),
		nextScanID:  1,
		enabled:     false,
		seqCounter:  app.NewSequenceCounter(),
		ctx:         ctx,
		cancel:      cancel,
		pendingResp: make(chan *app.APDU, 1),
	}

	// Create session
	m.session = newSession(config.LocalAddress, config.RemoteAddress, ch, m)

	// Add session to channel
	if err := ch.AddSession(m.session); err != nil {
		cancel()
		return nil, err
	}

	m.logger.Info("Master %s created: local=%d, remote=%d", config.ID, config.LocalAddress, config.RemoteAddress)
	return m, nil
}

// Enable enables the master
func (m *master) Enable() error {
	m.stateMu.Lock()
	if m.enabled {
		m.stateMu.Unlock()
		return nil
	}
	m.enabled = true
	m.stateMu.Unlock()

	m.logger.Info("Master %s enabled", m.config.ID)

	// Start task processor
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.taskProcessor()
	}()

	// Perform startup sequence
	if m.config.DisableUnsolOnStartup {
		// TODO: Send disable unsolicited
	}

	if m.config.StartupIntegrityScan {
		go func() {
			time.Sleep(100 * time.Millisecond)
			m.ScanIntegrity()
		}()
	}

	// Start automatic integrity scan if configured
	if m.config.IntegrityPeriod > 0 {
		m.AddIntegrityScan(m.config.IntegrityPeriod)
	}

	return nil
}

// Disable disables the master
func (m *master) Disable() error {
	m.stateMu.Lock()
	m.enabled = false
	m.stateMu.Unlock()

	m.logger.Info("Master %s disabled", m.config.ID)
	return nil
}

// Shutdown shuts down the master
func (m *master) Shutdown() error {
	m.logger.Info("Master %s shutting down", m.config.ID)

	m.Disable()
	m.cancel()
	m.wg.Wait()

	m.logger.Info("Master %s shutdown complete", m.config.ID)
	return nil
}

// taskProcessor processes tasks from the queue
func (m *master) taskProcessor() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.processTasks()
		}
	}
}

// processTasks processes ready tasks
func (m *master) processTasks() {
	if !m.isEnabled() {
		return
	}

	// Check for ready task
	task := m.taskQueue.NextReady(time.Now())
	if task == nil {
		return
	}

	// Execute task
	t := task.(Task)
	m.callbacks.OnTaskStart(t.Type(), 0)

	err := t.Execute(m)

	result := TaskResultSuccess
	if err != nil {
		m.logger.Error("Master %s: Task failed: %v", m.config.ID, err)
		result = TaskResultFailure
	}

	m.callbacks.OnTaskComplete(t.Type(), 0, result)

	// Reschedule periodic scans
	m.reschedulePeriodicScans()
}

// reschedulePer iodicScans reschedules periodic scans
func (m *master) reschedulePeriodicScans() {
	m.scansMu.Lock()
	defer m.scansMu.Unlock()

	for _, scan := range m.scans {
		if scan.enabled {
			if scan.demanded {
				// Demand request - run immediately
				scan.demanded = false
				m.taskQueue.Push(scan.task, scan.task.Priority(), time.Now())
			} else {
				// Regular periodic - schedule for next period
				nextRun := time.Now().Add(scan.period)
				m.taskQueue.Push(scan.task, scan.task.Priority(), nextRun)
			}
		}
	}
}

// isEnabled returns true if master is enabled
func (m *master) isEnabled() bool {
	m.stateMu.RLock()
	defer m.stateMu.RUnlock()
	return m.enabled
}

// getNextSequence returns the next sequence number using app layer helper
func (m *master) getNextSequence() uint8 {
	return m.seqCounter.Next()
}

// onReceiveAPDU handles received APDU
func (m *master) onReceiveAPDU(data []byte) error {
	apdu, err := app.Parse(data)
	if err != nil {
		m.logger.Error("Master %s: APDU parse error: %v", m.config.ID, err)
		return err
	}

	m.logger.Debug("Master %s: Received APDU: %s", m.config.ID, apdu)

	// Update IIN
	if apdu.IsResponse() {
		m.stateMu.Lock()
		m.lastIIN = apdu.IIN
		m.stateMu.Unlock()
		m.callbacks.OnReceiveIIN(apdu.IIN)
	}

	// Send to pending response channel
	m.pendingMu.Lock()
	select {
	case m.pendingResp <- apdu:
	default:
		m.logger.Warn("Master %s: Dropped response (no pending request)", m.config.ID)
	}
	m.pendingMu.Unlock()

	// Process measurements
	if apdu.IsResponse() && len(apdu.Objects) > 0 {
		m.processMeasurements(apdu)
	}

	return nil
}

// sendAndWait sends an APDU and waits for response
func (m *master) sendAndWait(apdu *app.APDU, timeout time.Duration) (*app.APDU, error) {
	// Serialize and send
	data := apdu.Serialize()
	if err := m.session.sendAPDU(data); err != nil {
		return nil, err
	}

	m.logger.Debug("Master %s: Sent APDU: %s", m.config.ID, apdu)

	// Wait for response
	select {
	case resp := <-m.pendingResp:
		return resp, nil
	case <-time.After(timeout):
		return nil, ErrTimeout
	case <-m.ctx.Done():
		return nil, m.ctx.Err()
	}
}

// Session returns the session (for channel registration)
func (m *master) Session() channel.Session {
	return m.session
}

// String returns string representation
func (m *master) String() string {
	return fmt.Sprintf("Master{ID=%s, Local=%d, Remote=%d}",
		m.config.ID, m.config.LocalAddress, m.config.RemoteAddress)
}
