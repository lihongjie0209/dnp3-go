package link

import (
	"errors"
	"fmt"
)

// ErrStrictRequestPending reports an attempt to start a second primary
// transaction before the first has completed.
var ErrStrictRequestPending = errors.New("DNP3 link request is already pending")

// StrictAction describes the state transition caused by one received frame.
type StrictAction struct {
	Response                *StrictFrame
	UserData                []byte
	Reset                   bool
	Duplicate               bool
	Acknowledged            bool
	NegativeAcknowledgement bool
	LinkStatus              bool
	NotSupported            bool
	DataFlowControl         bool
}

// StrictState implements the DNP3 link-layer primary/secondary state machine.
type StrictState struct {
	localAddress  uint16
	remoteAddress uint16
	localIsMaster bool
	ready         bool
	nextSendFCB   bool
	expectedFCB   bool
	pending       *StrictFrame
	pendingFn     byte
}

// NewStrictState creates a link state for one peer pair.
func NewStrictState(localAddress, remoteAddress uint16, localIsMaster bool) *StrictState {
	return &StrictState{
		localAddress: localAddress, remoteAddress: remoteAddress,
		localIsMaster: localIsMaster, nextSendFCB: true, expectedFCB: true,
	}
}

// Ready reports whether reset-link-state has completed.
func (s *StrictState) Ready() bool { return s.ready }

// MarkReset marks link state ready and discards a pending request.
func (s *StrictState) MarkReset() {
	s.ready, s.nextSendFCB, s.expectedFCB = true, true, true
	s.pending, s.pendingFn = nil, 0
}

// StartReset begins a reset-link-state exchange.
func (s *StrictState) StartReset() (StrictFrame, error) {
	if s.pending != nil {
		return StrictFrame{}, ErrStrictRequestPending
	}
	s.ready, s.nextSendFCB = false, true
	return s.startPending(0, false, nil)
}

// StartConfirmed begins a confirmed user-data exchange.
func (s *StrictState) StartConfirmed(userData []byte) (StrictFrame, error) {
	if !s.ready {
		return StrictFrame{}, errors.New("DNP3 link state has not been reset")
	}
	if len(userData) == 0 {
		return StrictFrame{}, errors.New("DNP3 confirmed user data is empty")
	}
	return s.startPending(3, s.nextSendFCB, userData)
}

// StartLinkStatus begins a request-link-status exchange.
func (s *StrictState) StartLinkStatus() (StrictFrame, error) {
	return s.startPending(9, false, nil)
}

// Unconfirmed constructs an unconfirmed user-data frame without state change.
func (s *StrictState) Unconfirmed(userData []byte) (StrictFrame, error) {
	if len(userData) == 0 {
		return StrictFrame{}, errors.New("DNP3 unconfirmed user data is empty")
	}
	if len(userData) > MaxDataSize {
		return StrictFrame{}, ErrFrameTooLong
	}
	return StrictFrame{
		Control: s.primaryControl(4, false, false), Destination: s.remoteAddress,
		Source: s.localAddress, UserData: append([]byte(nil), userData...),
	}, nil
}

// Retransmission returns an owned copy of the current pending request.
func (s *StrictState) Retransmission() (StrictFrame, error) {
	if s.pending == nil {
		return StrictFrame{}, errors.New("DNP3 link has no pending request")
	}
	return cloneStrictFrame(*s.pending), nil
}

func (s *StrictState) startPending(function byte, fcb bool, userData []byte) (StrictFrame, error) {
	if s.pending != nil {
		return StrictFrame{}, ErrStrictRequestPending
	}
	if len(userData) > MaxDataSize {
		return StrictFrame{}, ErrFrameTooLong
	}
	frame := StrictFrame{
		Control:     s.primaryControl(function, fcb, function == 2 || function == 3),
		Destination: s.remoteAddress, Source: s.localAddress,
		UserData: append([]byte(nil), userData...),
	}
	pending := cloneStrictFrame(frame)
	s.pending, s.pendingFn = &pending, function
	return cloneStrictFrame(frame), nil
}

func (s *StrictState) primaryControl(function byte, fcb, fcv bool) byte {
	control := byte(CtrlPRM) | function
	if s.localIsMaster {
		control |= CtrlDIR
	}
	if fcb {
		control |= CtrlFCB
	}
	if fcv {
		control |= CtrlFCV
	}
	return control
}

func (s *StrictState) secondaryFrame(function byte, dfc bool) StrictFrame {
	control := function
	if s.localIsMaster {
		control |= CtrlDIR
	}
	if dfc {
		control |= CtrlFCV
	}
	return StrictFrame{Control: control, Destination: s.remoteAddress, Source: s.localAddress}
}

// Handle validates and applies one inbound frame.
func (s *StrictState) Handle(frame StrictFrame) (StrictAction, error) {
	if frame.Destination != s.localAddress || frame.Source != s.remoteAddress {
		return StrictAction{}, errors.New("DNP3 link frame address does not match session")
	}
	if err := validateStrictControl(frame.Control); err != nil {
		return StrictAction{}, err
	}
	remoteIsMaster := frame.Control&CtrlDIR != 0
	if remoteIsMaster == s.localIsMaster {
		return StrictAction{}, errors.New("DNP3 link direction does not match remote station role")
	}
	if frame.Control&CtrlPRM != 0 {
		return s.handlePrimary(frame)
	}
	return s.handleSecondary(frame)
}

func (s *StrictState) handlePrimary(frame StrictFrame) (StrictAction, error) {
	function := frame.Control & CtrlFuncMask
	switch function {
	case 0:
		if len(frame.UserData) != 0 {
			return StrictAction{}, errors.New("DNP3 reset-link-state contains user data")
		}
		s.MarkReset()
		response := s.secondaryFrame(0, false)
		return StrictAction{Response: &response, Reset: true}, nil
	case 2:
		if len(frame.UserData) != 0 {
			return StrictAction{}, errors.New("DNP3 test-link-state contains user data")
		}
		return s.handleConfirmedPrimary(frame.Control, nil, false)
	case 3:
		if len(frame.UserData) == 0 {
			return StrictAction{}, errors.New("DNP3 confirmed frame has no user data")
		}
		return s.handleConfirmedPrimary(frame.Control, frame.UserData, true)
	case 4:
		if len(frame.UserData) == 0 {
			return StrictAction{}, errors.New("DNP3 unconfirmed frame has no user data")
		}
		return StrictAction{UserData: append([]byte(nil), frame.UserData...)}, nil
	case 9:
		if len(frame.UserData) != 0 {
			return StrictAction{}, errors.New("DNP3 link-status request contains user data")
		}
		response := s.secondaryFrame(11, false)
		return StrictAction{Response: &response, LinkStatus: true}, nil
	default:
		response := s.secondaryFrame(15, false)
		return StrictAction{Response: &response, NotSupported: true}, nil
	}
}

func (s *StrictState) handleConfirmedPrimary(control byte, userData []byte, deliver bool) (StrictAction, error) {
	if !s.ready {
		response := s.secondaryFrame(1, false)
		return StrictAction{Response: &response, NegativeAcknowledgement: true}, nil
	}
	response := s.secondaryFrame(0, false)
	fcb := control&CtrlFCB != 0
	if fcb != s.expectedFCB {
		return StrictAction{Response: &response, Duplicate: true}, nil
	}
	s.expectedFCB = !s.expectedFCB
	action := StrictAction{Response: &response}
	if deliver {
		action.UserData = append([]byte(nil), userData...)
	}
	return action, nil
}

func (s *StrictState) handleSecondary(frame StrictFrame) (StrictAction, error) {
	if len(frame.UserData) != 0 {
		return StrictAction{}, errors.New("DNP3 secondary frame contains user data")
	}
	if s.pending == nil {
		return StrictAction{}, errors.New("unexpected DNP3 secondary response")
	}
	function := frame.Control & CtrlFuncMask
	action := StrictAction{DataFlowControl: frame.Control&CtrlFCV != 0}
	switch function {
	case 0:
		if s.pendingFn != 0 && s.pendingFn != 2 && s.pendingFn != 3 {
			return StrictAction{}, fmt.Errorf("DNP3 ACK does not match pending function %d", s.pendingFn)
		}
		action.Acknowledged = true
		switch s.pendingFn {
		case 0:
			s.ready, s.nextSendFCB = true, true
		case 3:
			s.nextSendFCB = !s.nextSendFCB
		}
		s.pending, s.pendingFn = nil, 0
	case 1:
		action.NegativeAcknowledgement = true
	case 11:
		if s.pendingFn != 9 {
			return StrictAction{}, fmt.Errorf("DNP3 link status does not match pending function %d", s.pendingFn)
		}
		action.LinkStatus = true
		s.pending, s.pendingFn = nil, 0
	case 15:
		action.NotSupported = true
		s.pending, s.pendingFn = nil, 0
	default:
		return StrictAction{}, fmt.Errorf("unsupported DNP3 secondary function %d", function)
	}
	return action, nil
}

func cloneStrictFrame(frame StrictFrame) StrictFrame {
	frame.UserData = append([]byte(nil), frame.UserData...)
	return frame
}
