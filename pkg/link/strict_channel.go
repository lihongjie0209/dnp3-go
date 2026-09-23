package link

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"time"
)

// ErrStrictChannelBusy reports a concurrent operation on a single-peer link.
var ErrStrictChannelBusy = errors.New("DNP3 link channel already has an active operation")

// StrictConnection is the stream and deadline surface required by StrictChannel.
type StrictConnection interface {
	io.Reader
	io.Writer
	SetDeadline(time.Time) error
}

// StrictUserDataHandler handles owned inbound link user data encountered while
// a transaction is waiting for its secondary response.
type StrictUserDataHandler func(context.Context, []byte) error

// StrictChannel drives one StrictState over one stream connection.
type StrictChannel struct {
	connection      StrictConnection
	state           *StrictState
	responseTimeout time.Duration
	maximumRetries  int
	busy            atomic.Bool
}

// NewStrictChannel creates a bounded single-peer link channel.
func NewStrictChannel(connection StrictConnection, state *StrictState, responseTimeout time.Duration, maximumRetries int) *StrictChannel {
	return &StrictChannel{connection: connection, state: state, responseTimeout: responseTimeout, maximumRetries: maximumRetries}
}

// State returns the channel's owned protocol state.
func (c *StrictChannel) State() *StrictState { return c.state }

// Reset performs reset-link-state and waits for its ACK.
func (c *StrictChannel) Reset(ctx context.Context, handler StrictUserDataHandler) error {
	if err := c.acquire(); err != nil {
		return err
	}
	defer c.release()
	if err := c.validate(); err != nil {
		return err
	}
	frame, err := c.state.StartReset()
	if err != nil {
		return err
	}
	return c.transact(ctx, frame, handler, func(action StrictAction) bool { return action.Acknowledged })
}

// SendConfirmed sends user data and waits for its ACK with bounded retries.
func (c *StrictChannel) SendConfirmed(ctx context.Context, userData []byte, handler StrictUserDataHandler) error {
	if err := c.acquire(); err != nil {
		return err
	}
	defer c.release()
	if err := c.validate(); err != nil {
		return err
	}
	frame, err := c.state.StartConfirmed(userData)
	if err != nil {
		return err
	}
	return c.transact(ctx, frame, handler, func(action StrictAction) bool { return action.Acknowledged })
}

// SendUnconfirmed sends one unconfirmed user-data frame.
func (c *StrictChannel) SendUnconfirmed(ctx context.Context, userData []byte) error {
	if err := c.acquire(); err != nil {
		return err
	}
	defer c.release()
	if err := c.validate(); err != nil {
		return err
	}
	frame, err := c.state.Unconfirmed(userData)
	if err != nil {
		return err
	}
	cleanup, err := c.installCancellation(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	if err = c.setOperationDeadline(ctx); err != nil {
		return err
	}
	if err = WriteStrictFrame(c.connection, frame); err != nil {
		return c.normalizeIOError(ctx, "writing DNP3 link frame", err)
	}
	return nil
}

// KeepAlive requests and waits for link status.
func (c *StrictChannel) KeepAlive(ctx context.Context, handler StrictUserDataHandler) error {
	if err := c.acquire(); err != nil {
		return err
	}
	defer c.release()
	if err := c.validate(); err != nil {
		return err
	}
	frame, err := c.state.StartLinkStatus()
	if err != nil {
		return err
	}
	return c.transact(ctx, frame, handler, func(action StrictAction) bool { return action.LinkStatus })
}

// Receive reads and handles frames until user data is available.
func (c *StrictChannel) Receive(ctx context.Context) ([]byte, error) {
	if err := c.acquire(); err != nil {
		return nil, err
	}
	defer c.release()
	if err := c.validate(); err != nil {
		return nil, err
	}
	cleanup, err := c.installCancellation(ctx)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	if deadline, ok := ctx.Deadline(); ok {
		if err = c.connection.SetDeadline(deadline); err != nil {
			return nil, fmt.Errorf("setting DNP3 receive deadline: %w", err)
		}
	}
	for {
		frame, readErr := ReadStrictFrame(c.connection)
		if readErr != nil {
			return nil, c.normalizeIOError(ctx, "reading DNP3 link frame", readErr)
		}
		action, handleErr := c.state.Handle(frame)
		if handleErr != nil {
			return nil, handleErr
		}
		if action.Response != nil {
			if writeErr := WriteStrictFrame(c.connection, *action.Response); writeErr != nil {
				return nil, c.normalizeIOError(ctx, "writing DNP3 link response", writeErr)
			}
		}
		if len(action.UserData) != 0 {
			return append([]byte(nil), action.UserData...), nil
		}
	}
}

func (c *StrictChannel) transact(ctx context.Context, first StrictFrame, handler StrictUserDataHandler, complete func(StrictAction) bool) error {
	cleanup, err := c.installCancellation(ctx)
	if err != nil {
		return err
	}
	defer cleanup()
	frame := first
	var lastErr error
	for attempt := 0; attempt <= c.maximumRetries; attempt++ {
		if attempt > 0 {
			frame, err = c.state.Retransmission()
			if err != nil {
				return err
			}
		}
		if err = c.setOperationDeadline(ctx); err != nil {
			return err
		}
		if err = WriteStrictFrame(c.connection, frame); err != nil {
			return c.normalizeIOError(ctx, "writing DNP3 link frame", err)
		}
		for {
			incoming, readErr := ReadStrictFrame(c.connection)
			if readErr != nil {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				if isStrictTimeout(readErr) {
					lastErr = readErr
					break
				}
				return fmt.Errorf("reading DNP3 link response: %w", readErr)
			}
			action, handleErr := c.state.Handle(incoming)
			if handleErr != nil {
				return handleErr
			}
			if action.Response != nil {
				if writeErr := WriteStrictFrame(c.connection, *action.Response); writeErr != nil {
					return c.normalizeIOError(ctx, "writing DNP3 link response", writeErr)
				}
			}
			if len(action.UserData) != 0 && handler != nil {
				if handleErr := handler(ctx, append([]byte(nil), action.UserData...)); handleErr != nil {
					return fmt.Errorf("handling DNP3 user data: %w", handleErr)
				}
			}
			if complete(action) {
				return nil
			}
			if action.NegativeAcknowledgement {
				lastErr = errors.New("DNP3 link request was negatively acknowledged")
				break
			}
		}
	}
	return fmt.Errorf("DNP3 link request failed after %d attempts: %w", c.maximumRetries+1, lastErr)
}

func (c *StrictChannel) validate() error {
	if c.connection == nil {
		return errors.New("DNP3 link connection is required")
	}
	if c.state == nil {
		return errors.New("DNP3 link state is required")
	}
	if c.responseTimeout <= 0 {
		return errors.New("DNP3 response timeout must be positive")
	}
	if c.maximumRetries < 0 {
		return errors.New("DNP3 maximum retries cannot be negative")
	}
	return nil
}

func (c *StrictChannel) acquire() error {
	if !c.busy.CompareAndSwap(false, true) {
		return ErrStrictChannelBusy
	}
	return nil
}

func (c *StrictChannel) release() { c.busy.Store(false) }

func (c *StrictChannel) installCancellation(ctx context.Context) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	finished := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { _ = c.connection.SetDeadline(time.Now()); close(finished) })
	return func() {
		if !stop() {
			<-finished
		}
		_ = c.connection.SetDeadline(time.Time{})
	}, nil
}

func (c *StrictChannel) setOperationDeadline(ctx context.Context) error {
	deadline := time.Now().Add(c.responseTimeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := c.connection.SetDeadline(deadline); err != nil {
		return fmt.Errorf("setting DNP3 operation deadline: %w", err)
	}
	return nil
}

func (c *StrictChannel) normalizeIOError(ctx context.Context, operation string, err error) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func isStrictTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

// WriteStrictFrame encodes and completely writes one strict frame.
func WriteStrictFrame(writer io.Writer, frame StrictFrame) error {
	if writer == nil {
		return errors.New("DNP3 link writer is required")
	}
	encoded, err := EncodeStrictFrame(frame)
	if err != nil {
		return err
	}
	for len(encoded) != 0 {
		written, writeErr := writer.Write(encoded)
		if writeErr != nil {
			return fmt.Errorf("writing DNP3 link frame: %w", writeErr)
		}
		if written <= 0 || written > len(encoded) {
			return io.ErrShortWrite
		}
		encoded = encoded[written:]
	}
	return nil
}
