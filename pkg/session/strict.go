// Package session composes the strict DNP3 link, transport, and application
// layers into a bounded single-peer wire session.
package session

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/lihongjie0209/dnp3-go/pkg/app"
	"github.com/lihongjie0209/dnp3-go/pkg/link"
	"github.com/lihongjie0209/dnp3-go/pkg/transport"
)

// ErrStrictSessionBusy reports a concurrent operation on a session.
var ErrStrictSessionBusy = errors.New("DNP3 wire session already has an active operation")

// Strict is a bounded single-peer DNP3 wire session.
type Strict struct {
	channel            *link.StrictChannel
	transportInput     *transport.StrictReassembler
	applicationInput   *app.StrictReassembler
	initializationErr  error
	nextTransport      byte
	maximumApplication int
	busy               atomic.Bool
}

// NewStrict constructs a wire session over an existing stream connection.
func NewStrict(connection link.StrictConnection, localAddress, remoteAddress uint16, localIsMaster bool, responseTimeout time.Duration, maximumRetries, maximumApplication int) *Strict {
	state := link.NewStrictState(localAddress, remoteAddress, localIsMaster)
	transportInput, err := transport.NewStrictReassembler(maximumApplication)
	if maximumApplication < 1 && err == nil {
		err = errors.New("DNP3 maximum application bytes must be positive")
	}
	return &Strict{
		channel:        link.NewStrictChannel(connection, state, responseTimeout, maximumRetries),
		transportInput: transportInput, applicationInput: app.NewStrictReassembler(maximumApplication),
		initializationErr: err, maximumApplication: maximumApplication,
	}
}

// Channel exposes the composed strict link channel for diagnostics and
// explicit link-state initialization.
func (s *Strict) Channel() *link.StrictChannel { return s.channel }

// Reset performs reset-link-state.
func (s *Strict) Reset(ctx context.Context, handler link.StrictUserDataHandler) error {
	if err := s.acquire(); err != nil {
		return err
	}
	defer s.release()
	if s.initializationErr != nil {
		return s.initializationErr
	}
	return s.channel.Reset(ctx, handler)
}

// KeepAlive requests link status.
func (s *Strict) KeepAlive(ctx context.Context, handler link.StrictUserDataHandler) error {
	if err := s.acquire(); err != nil {
		return err
	}
	defer s.release()
	if s.initializationErr != nil {
		return s.initializationErr
	}
	return s.channel.KeepAlive(ctx, handler)
}

// SendApplication sends a complete application fragment using confirmed link
// user data.
func (s *Strict) SendApplication(ctx context.Context, fragment app.StrictFragment, handler link.StrictUserDataHandler) error {
	return s.sendApplication(ctx, fragment, handler, true)
}

// SendApplicationUnconfirmed sends a complete application fragment using
// unconfirmed link user data.
func (s *Strict) SendApplicationUnconfirmed(ctx context.Context, fragment app.StrictFragment) error {
	return s.sendApplication(ctx, fragment, nil, false)
}

func (s *Strict) sendApplication(ctx context.Context, fragment app.StrictFragment, handler link.StrictUserDataHandler, confirmed bool) error {
	if err := s.acquire(); err != nil {
		return err
	}
	defer s.release()
	if s.initializationErr != nil {
		return s.initializationErr
	}
	return s.sendApplicationLocked(ctx, fragment, handler, confirmed)
}

func (s *Strict) sendApplicationLocked(ctx context.Context, fragment app.StrictFragment, handler link.StrictUserDataHandler, confirmed bool) error {
	encoded, err := app.EncodeStrictFragment(fragment, s.maximumApplication)
	if err != nil {
		return err
	}
	segments, next, err := transport.SegmentStrict(encoded, s.nextTransport, link.MaxDataSize)
	if err != nil {
		return err
	}
	for index, segment := range segments {
		if confirmed {
			err = s.channel.SendConfirmed(ctx, segment, handler)
		} else {
			err = s.channel.SendUnconfirmed(ctx, segment)
		}
		if err != nil {
			return fmt.Errorf("sending DNP3 transport segment %d: %w", index, err)
		}
	}
	s.nextTransport = next
	return nil
}

// ReceiveApplication receives, automatically confirms, and reassembles one
// complete application fragment.
func (s *Strict) ReceiveApplication(ctx context.Context) (app.StrictFragment, error) {
	if err := s.acquire(); err != nil {
		return app.StrictFragment{}, err
	}
	defer s.release()
	if s.initializationErr != nil {
		return app.StrictFragment{}, s.initializationErr
	}
	for {
		segment, err := s.channel.Receive(ctx)
		if err != nil {
			return app.StrictFragment{}, err
		}
		transportComplete, encoded, err := s.transportInput.Push(segment)
		if err != nil {
			return app.StrictFragment{}, err
		}
		if !transportComplete {
			continue
		}
		fragment, err := app.DecodeStrictFragment(encoded, s.maximumApplication)
		if err != nil {
			return app.StrictFragment{}, err
		}
		if fragment.CON {
			confirmation := app.StrictFragment{
				FIR: true, FIN: true, UNS: fragment.UNS,
				Sequence: fragment.Sequence, Function: 0,
			}
			if err = s.sendApplicationLocked(ctx, confirmation, nil, false); err != nil {
				return app.StrictFragment{}, fmt.Errorf("confirming DNP3 application fragment: %w", err)
			}
			fragment.CON = false
		}
		applicationComplete, complete, err := s.applicationInput.Push(fragment)
		if err != nil {
			return app.StrictFragment{}, err
		}
		if applicationComplete {
			return complete, nil
		}
	}
}

func (s *Strict) acquire() error {
	if !s.busy.CompareAndSwap(false, true) {
		return ErrStrictSessionBusy
	}
	return nil
}

func (s *Strict) release() { s.busy.Store(false) }
