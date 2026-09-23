package link

import (
	"testing"
	"time"
)

func TestMasterLink_New(t *testing.T) {
	config := LinkLayerConfig{
		LocalAddress:  1,
		RemoteAddress: 1024,
		IsMaster:      true,
		Timeout:       2 * time.Second,
		MaxRetries:    3,
	}

	master := NewMasterLink(config)

	if master.localAddress != 1 {
		t.Errorf("Expected local address 1, got %d", master.localAddress)
	}

	if master.remoteAddress != 1024 {
		t.Errorf("Expected remote address 1024, got %d", master.remoteAddress)
	}

	if master.state != LinkStateIdle {
		t.Errorf("Expected state Idle, got %s", master.state)
	}

	if master.fcb != false {
		t.Errorf("Expected FCB to be false initially")
	}
}

func TestMasterLink_Start(t *testing.T) {
	config := DefaultLinkLayerConfig()
	master := NewMasterLink(config)

	err := master.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	if !master.IsOnline() {
		t.Errorf("Master should be online after start")
	}

	// Cleanup
	master.Stop()
}

func TestMasterLink_FCBToggle(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.Timeout = 100 * time.Millisecond
	config.MaxRetries = 0 // No retries for this test

	master := NewMasterLink(config)
	master.Start()
	defer master.Stop()

	// Initial FCB should be false
	if master.GetFCB() != false {
		t.Errorf("Initial FCB should be false, got %v", master.GetFCB())
	}

	// Simulate sending confirmed data (will timeout but should toggle FCB)
	// We're testing FCB toggle logic, not actual transmission
	go func() {
		master.SendConfirmedUserData([]byte{0x01, 0x02})
	}()

	// Wait for timeout
	time.Sleep(200 * time.Millisecond)

	// FCB should have toggled back on error (since send failed)
	// Actually, on error it toggles back, so should be false again
	if master.GetFCB() != false {
		t.Errorf("FCB should be false after failed send, got %v", master.GetFCB())
	}
}

func TestMasterLink_ResetLink(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.Timeout = 100 * time.Millisecond
	config.MaxRetries = 1

	master := NewMasterLink(config)
	master.Start()
	defer master.Stop()

	// Set FCB to true to test reset
	master.fcb = true

	// Simulate receiving ACK response
	go func() {
		time.Sleep(10 * time.Millisecond)
		ackFrame := NewACKFrame(master.localAddress, master.remoteAddress)
		master.OnFrameReceived(ackFrame)
	}()

	err := master.ResetLink()
	if err != nil {
		t.Errorf("ResetLink failed: %v", err)
	}

	// FCB should be reset to false
	if master.fcb != false {
		t.Errorf("FCB should be false after reset, got %v", master.fcb)
	}

	if master.state != LinkStateIdle {
		t.Errorf("State should be Idle after reset, got %s", master.state)
	}
}

func TestMasterLink_TestLink(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.Timeout = 100 * time.Millisecond
	config.MaxRetries = 1

	master := NewMasterLink(config)
	master.Start()
	defer master.Stop()

	// Simulate receiving ACK response
	go func() {
		time.Sleep(10 * time.Millisecond)
		ackFrame := NewACKFrame(master.localAddress, master.remoteAddress)
		master.OnFrameReceived(ackFrame)
	}()

	err := master.TestLink()
	if err != nil {
		t.Errorf("TestLink failed: %v", err)
	}

	if master.state != LinkStateIdle {
		t.Errorf("State should be Idle after test, got %s", master.state)
	}
}

func TestMasterLink_OnFrameReceived_InvalidAddress(t *testing.T) {
	config := DefaultLinkLayerConfig()
	master := NewMasterLink(config)

	// Frame from wrong source
	frame := NewACKFrame(master.localAddress, 999)

	err := master.OnFrameReceived(frame)
	if err != ErrInvalidAddress {
		t.Errorf("Expected ErrInvalidAddress, got %v", err)
	}
}

func TestMasterLink_OnFrameReceived_Unsolicited(t *testing.T) {
	config := DefaultLinkLayerConfig()
	dataReceived := false

	config.DataCallback = func(data []byte) error {
		dataReceived = true
		if len(data) != 3 {
			t.Errorf("Expected 3 bytes, got %d", len(data))
		}
		return nil
	}

	master := NewMasterLink(config)
	master.Start()
	defer master.Stop()

	// Unsolicited frame from outstation
	unsolFrame := NewUnsolicitedFrame(
		master.localAddress,
		master.remoteAddress,
		[]byte{0x01, 0x02, 0x03},
	)

	err := master.OnFrameReceived(unsolFrame)
	if err != nil {
		t.Errorf("OnFrameReceived failed: %v", err)
	}

	// Give callback time to execute
	time.Sleep(10 * time.Millisecond)

	if !dataReceived {
		t.Errorf("Data callback was not called for unsolicited frame")
	}
}

func TestMasterLink_HandleResponse_ACK(t *testing.T) {
	config := DefaultLinkLayerConfig()
	master := NewMasterLink(config)

	ackFrame := NewACKFrame(master.localAddress, master.remoteAddress)

	err := master.handleResponse(ackFrame)
	if err != nil {
		t.Errorf("handleResponse(ACK) failed: %v", err)
	}
}

func TestMasterLink_HandleResponse_NACK(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.MaxRetries = 0 // Will fail immediately
	master := NewMasterLink(config)

	master.lastSentFrame = NewTestLinkFrame(master.remoteAddress, master.localAddress)

	nackFrame := NewNACKFrame(master.localAddress, master.remoteAddress)

	err := master.handleResponse(nackFrame)
	if err != ErrMaxRetriesExceeded {
		t.Errorf("Expected ErrMaxRetriesExceeded, got %v", err)
	}
}

func TestMasterLink_SendUnconfirmedUserData(t *testing.T) {
	config := DefaultLinkLayerConfig()
	master := NewMasterLink(config)
	master.Start()
	defer master.Stop()

	// Unconfirmed data doesn't wait for response
	err := master.SendUnconfirmedUserData([]byte{0x01, 0x02, 0x03})

	// Should complete without error (no response expected)
	if err != nil {
		t.Errorf("SendUnconfirmedUserData failed: %v", err)
	}
}

func TestMasterLink_Timeout(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.Timeout = 50 * time.Millisecond
	config.MaxRetries = 1

	master := NewMasterLink(config)
	master.Start()
	defer master.Stop()

	// Don't send any response - should timeout
	start := time.Now()
	err := master.TestLink()
	duration := time.Since(start)

	if err != ErrMaxRetriesExceeded {
		t.Errorf("Expected ErrMaxRetriesExceeded, got %v", err)
	}

	// Should have tried twice (initial + 1 retry) with 50ms timeout each
	expectedMin := 100 * time.Millisecond
	if duration < expectedMin {
		t.Errorf("Expected at least %v duration, got %v", expectedMin, duration)
	}
}

func TestMasterLink_MaxRetriesExceeded(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.Timeout = 20 * time.Millisecond
	config.MaxRetries = 2

	master := NewMasterLink(config)
	master.Start()
	defer master.Stop()

	// No response - should retry maxRetries times
	err := master.TestLink()

	if err != ErrMaxRetriesExceeded {
		t.Errorf("Expected ErrMaxRetriesExceeded, got %v", err)
	}
}

func TestMasterLink_SetTimeout(t *testing.T) {
	config := DefaultLinkLayerConfig()
	master := NewMasterLink(config)

	newTimeout := 5 * time.Second
	master.SetTimeout(newTimeout)

	if master.timeout != newTimeout {
		t.Errorf("Expected timeout %v, got %v", newTimeout, master.timeout)
	}
}

func TestMasterLink_SetRetries(t *testing.T) {
	config := DefaultLinkLayerConfig()
	master := NewMasterLink(config)

	master.SetRetries(5)

	if master.maxRetries != 5 {
		t.Errorf("Expected maxRetries 5, got %d", master.maxRetries)
	}
}

func TestMasterLink_InvalidState(t *testing.T) {
	config := DefaultLinkLayerConfig()
	master := NewMasterLink(config)

	// Set state to non-idle
	master.state = LinkStateWaitACK

	// Should fail because not in idle state
	err := master.ResetLink()
	if err != ErrInvalidState {
		t.Errorf("Expected ErrInvalidState, got %v", err)
	}
}

func TestMasterLink_StatusCallback(t *testing.T) {
	config := DefaultLinkLayerConfig()
	config.Timeout = 50 * time.Millisecond
	config.MaxRetries = 1

	type statusResult struct {
		state LinkState
		err   error
	}
	status := make(chan statusResult, 1)

	config.StatusCallback = func(state LinkState, err error) {
		status <- statusResult{state: state, err: err}
	}

	master := NewMasterLink(config)
	master.Start()
	defer master.Stop()

	// Simulate successful reset
	go func() {
		time.Sleep(10 * time.Millisecond)
		ackFrame := NewACKFrame(master.localAddress, master.remoteAddress)
		master.OnFrameReceived(ackFrame)
	}()

	master.ResetLink()

	select {
	case received := <-status:
		if received.state != LinkStateIdle {
			t.Errorf("Expected state Idle in callback, got %s", received.state)
		}
		if received.err != nil {
			t.Errorf("Expected no error in callback, got %v", received.err)
		}
	case <-time.After(time.Second):
		t.Fatal("Status callback was not called")
	}
}
