package main

import (
	"context"
	"errors"
	"sync"

	"github.com/lihongjie0209/dnp3-go/pkg/channel"
)

// MockChannel is an example implementation of PhysicalChannel
// This shows how users can implement their own transport layers
type MockChannel struct {
	readChan  chan []byte
	writeChan chan []byte
	closeChan chan struct{}
	closed    bool
	mu        sync.RWMutex
	stats     channel.TransportStats
}

// NewMockChannel creates a new mock channel
func NewMockChannel() *MockChannel {
	return &MockChannel{
		readChan:  make(chan []byte, 10),
		writeChan: make(chan []byte, 10),
		closeChan: make(chan struct{}),
	}
}

// Read implements PhysicalChannel.Read
func (m *MockChannel) Read(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-m.closeChan:
		return nil, errors.New("channel closed")
	case data := <-m.readChan:
		m.mu.Lock()
		m.stats.BytesReceived += uint64(len(data))
		m.mu.Unlock()
		return data, nil
	}
}

// Write implements PhysicalChannel.Write
func (m *MockChannel) Write(ctx context.Context, data []byte) error {
	m.mu.RLock()
	if m.closed {
		m.mu.RUnlock()
		return errors.New("channel closed")
	}
	m.mu.RUnlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case m.writeChan <- data:
		m.mu.Lock()
		m.stats.BytesSent += uint64(len(data))
		m.mu.Unlock()
		return nil
	}
}

// Close implements PhysicalChannel.Close
func (m *MockChannel) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}

	m.closed = true
	close(m.closeChan)
	return nil
}

// Statistics implements PhysicalChannel.Statistics
func (m *MockChannel) Statistics() channel.TransportStats {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.stats
}

// InjectRead simulates receiving data (for testing)
func (m *MockChannel) InjectRead(data []byte) {
	m.readChan <- data
}

// GetWritten retrieves written data (for testing)
func (m *MockChannel) GetWritten() []byte {
	select {
	case data := <-m.writeChan:
		return data
	default:
		return nil
	}
}

// Example: TCP Channel implementation
// This shows how you might implement a real TCP transport

/*
import "net"

type TCPChannel struct {
	conn  net.Conn
	stats channel.TransportStats
	mu    sync.Mutex
}

func NewTCPChannel(address string) (*TCPChannel, error) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, err
	}
	return &TCPChannel{conn: conn}, nil
}

func (t *TCPChannel) Read(ctx context.Context) ([]byte, error) {
	// Set deadline from context
	if deadline, ok := ctx.Deadline(); ok {
		t.conn.SetReadDeadline(deadline)
	}

	// Read length prefix (2 bytes)
	lenBuf := make([]byte, 2)
	if _, err := t.conn.Read(lenBuf); err != nil {
		t.mu.Lock()
		t.stats.ReadErrors++
		t.mu.Unlock()
		return nil, err
	}

	length := uint16(lenBuf[0])<<8 | uint16(lenBuf[1])

	// Read frame data
	data := make([]byte, length)
	if _, err := t.conn.Read(data); err != nil {
		t.mu.Lock()
		t.stats.ReadErrors++
		t.mu.Unlock()
		return nil, err
	}

	t.mu.Lock()
	t.stats.BytesReceived += uint64(length)
	t.mu.Unlock()

	return data, nil
}

func (t *TCPChannel) Write(ctx context.Context, data []byte) error {
	if deadline, ok := ctx.Deadline(); ok {
		t.conn.SetWriteDeadline(deadline)
	}

	// Write length prefix
	length := uint16(len(data))
	lenBuf := []byte{byte(length >> 8), byte(length)}

	if _, err := t.conn.Write(lenBuf); err != nil {
		t.mu.Lock()
		t.stats.WriteErrors++
		t.mu.Unlock()
		return err
	}

	// Write data
	if _, err := t.conn.Write(data); err != nil {
		t.mu.Lock()
		t.stats.WriteErrors++
		t.mu.Unlock()
		return err
	}

	t.mu.Lock()
	t.stats.BytesSent += uint64(length)
	t.mu.Unlock()

	return nil
}

func (t *TCPChannel) Close() error {
	return t.conn.Close()
}

func (t *TCPChannel) Statistics() channel.TransportStats {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.stats
}
*/
