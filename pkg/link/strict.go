package link

import (
	"errors"
	"fmt"
	"io"
)

// StrictFrame is the exact wire representation used by the strict codec.
// Control is supplied explicitly so callers can preserve all link flags.
type StrictFrame struct {
	Control     byte
	Destination uint16
	Source      uint16
	UserData    []byte
}

// EncodeStrictFrame validates the complete control field and returns an owned
// exact DNP3 link frame.
func EncodeStrictFrame(frame StrictFrame) ([]byte, error) {
	if err := validateStrictControl(frame.Control); err != nil {
		return nil, err
	}
	if len(frame.UserData) > MaxDataSize {
		return nil, ErrFrameTooLong
	}
	value := &Frame{
		Control: frame.Control, Destination: frame.Destination, Source: frame.Source,
		UserData: append([]byte(nil), frame.UserData...),
	}
	return value.Serialize()
}

// DecodeStrictFrame decodes exactly one complete frame. Unlike Parse, it
// rejects trailing bytes and invalid control-field combinations.
func DecodeStrictFrame(data []byte) (StrictFrame, error) {
	frame, consumed, err := Parse(data)
	if err != nil {
		return StrictFrame{}, err
	}
	if consumed != len(data) {
		return StrictFrame{}, errors.New("DNP3 link frame contains trailing bytes")
	}
	if err = validateStrictControl(frame.Control); err != nil {
		return StrictFrame{}, err
	}
	return StrictFrame{
		Control: frame.Control, Destination: frame.Destination, Source: frame.Source,
		UserData: append([]byte(nil), frame.UserData...),
	}, nil
}

// ReadStrictFrame consumes exactly one frame from a fragmented byte stream.
func ReadStrictFrame(reader io.Reader) (StrictFrame, error) {
	if reader == nil {
		return StrictFrame{}, errors.New("DNP3 link reader is required")
	}
	header := make([]byte, HeaderSize)
	if _, err := io.ReadFull(reader, header); err != nil {
		return StrictFrame{}, err
	}
	if header[0] != StartByte1 || header[1] != StartByte2 || header[2] < 5 {
		return StrictFrame{}, errors.New("invalid DNP3 link header")
	}
	payloadLength := int(header[2]) - 5
	wirePayloadLength := payloadLength + 2*((payloadLength+BlockSize-1)/BlockSize)
	wire := make([]byte, HeaderSize+wirePayloadLength)
	copy(wire, header)
	if _, err := io.ReadFull(reader, wire[HeaderSize:]); err != nil {
		return StrictFrame{}, fmt.Errorf("reading DNP3 link data: %w", err)
	}
	return DecodeStrictFrame(wire)
}

// ValidateStrictControl validates primary/secondary flags and function codes.
func ValidateStrictControl(control byte) error { return validateStrictControl(control) }

func validateStrictControl(control byte) error {
	primary := control&CtrlPRM != 0
	function := control & CtrlFuncMask
	if primary {
		fcb := control&CtrlFCB != 0
		fcv := control&CtrlFCV != 0
		requiresFCV := function == byte(FuncTestLinkStates) || function == byte(FuncUserDataConfirmed)
		validFunction := function == byte(FuncResetLink) || requiresFCV ||
			function == byte(FuncUserDataUnconfirmed) || function == byte(FuncRequestLinkStatus)
		if !validFunction || fcv != requiresFCV || fcb && !fcv {
			return fmt.Errorf("invalid DNP3 primary link control 0x%02x", control)
		}
		return nil
	}
	if control&CtrlFCB != 0 {
		return fmt.Errorf("reserved bit set in DNP3 secondary link control 0x%02x", control)
	}
	if function != byte(FuncAck) && function != byte(FuncNack) &&
		function != byte(FuncLinkStatusResponse) && function != byte(FuncLinkNotUsed) {
		return fmt.Errorf("invalid DNP3 secondary link function %d", function)
	}
	return nil
}
