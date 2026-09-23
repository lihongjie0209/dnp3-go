package transport

import (
	"errors"
	"fmt"
)

const (
	StrictFIR      byte = 0x80
	StrictFIN      byte = 0x40
	StrictSequence byte = 0x3f
)

func SegmentStrict(fragment []byte, sequence byte, maximumLinkPayload int) ([][]byte, byte, error) {
	if len(fragment) == 0 {
		return nil, sequence & StrictSequence, errors.New("DNP3 application fragment is empty")
	}
	if maximumLinkPayload < 2 || maximumLinkPayload > 250 {
		return nil, sequence & StrictSequence, errors.New("DNP3 transport link payload limit must be between 2 and 250")
	}
	chunkSize := maximumLinkPayload - 1
	segments := make([][]byte, 0, (len(fragment)+chunkSize-1)/chunkSize)
	current := sequence & StrictSequence
	for offset := 0; offset < len(fragment); offset += chunkSize {
		end := min(offset+chunkSize, len(fragment))
		header := current
		if offset == 0 {
			header |= StrictFIR
		}
		if end == len(fragment) {
			header |= StrictFIN
		}
		segment := make([]byte, 1, 1+end-offset)
		segment[0] = header
		segment = append(segment, fragment[offset:end]...)
		segments = append(segments, segment)
		current = (current + 1) & StrictSequence
	}
	return segments, current, nil
}

type StrictReassembler struct {
	maximumFragment int
	active          bool
	nextSequence    byte
	fragment        []byte
}

func NewStrictReassembler(maximumFragment int) (*StrictReassembler, error) {
	if maximumFragment < 1 {
		return nil, errors.New("DNP3 maximum transport fragment must be positive")
	}
	return &StrictReassembler{maximumFragment: maximumFragment}, nil
}

func (r *StrictReassembler) Reset() {
	r.active = false
	r.nextSequence = 0
	r.fragment = r.fragment[:0]
}

func (r *StrictReassembler) Push(segment []byte) (bool, []byte, error) {
	if len(segment) < 2 {
		return false, nil, errors.New("DNP3 transport segment has no payload")
	}
	header := segment[0]
	sequence := header & StrictSequence
	first := header&StrictFIR != 0
	final := header&StrictFIN != 0
	if first {
		r.Reset()
		r.active = true
		r.nextSequence = sequence
	} else if !r.active {
		return false, nil, errors.New("DNP3 transport continuation without first segment")
	}
	if sequence != r.nextSequence {
		expected := r.nextSequence
		r.Reset()
		return false, nil, fmt.Errorf("DNP3 transport sequence %d, want %d", sequence, expected)
	}
	if len(r.fragment)+len(segment)-1 > r.maximumFragment {
		r.Reset()
		return false, nil, errors.New("DNP3 transport fragment exceeds configured limit")
	}
	r.fragment = append(r.fragment, segment[1:]...)
	r.nextSequence = (sequence + 1) & StrictSequence
	if !final {
		return false, nil, nil
	}
	result := append([]byte(nil), r.fragment...)
	r.Reset()
	return true, result, nil
}
