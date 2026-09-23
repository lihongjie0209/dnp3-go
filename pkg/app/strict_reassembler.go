package app

import (
	"errors"
	"fmt"
)

// StrictReassembler joins bounded application fragments.
type StrictReassembler struct {
	maximumObjects int
	active         bool
	firstSequence  byte
	nextSequence   byte
	function       byte
	unsolicited    bool
	confirm        bool
	iin            uint16
	objects        []byte
}

// NewStrictReassembler creates an application reassembler with an object-byte
// limit applied across the whole fragment sequence.
func NewStrictReassembler(maximumObjects int) *StrictReassembler {
	return &StrictReassembler{maximumObjects: maximumObjects}
}

// Reset discards any partial sequence.
func (r *StrictReassembler) Reset() {
	r.active = false
	r.firstSequence = 0
	r.nextSequence = 0
	r.function = 0
	r.unsolicited = false
	r.confirm = false
	r.iin = 0
	r.objects = r.objects[:0]
}

// Push consumes one fragment and returns an owned complete fragment when FIN
// closes a valid sequence.
func (r *StrictReassembler) Push(fragment StrictFragment) (bool, StrictFragment, error) {
	if r.maximumObjects < 1 {
		return false, StrictFragment{}, errors.New("DNP3 maximum application object bytes must be positive")
	}
	if fragment.Sequence > 15 {
		r.Reset()
		return false, StrictFragment{}, errors.New("DNP3 application sequence exceeds 15")
	}
	if fragment.FIR {
		if r.active {
			r.Reset()
			return false, StrictFragment{}, errors.New("DNP3 application FIR repeated before FIN")
		}
		r.active = true
		r.firstSequence = fragment.Sequence
		r.nextSequence = fragment.Sequence
		r.function = fragment.Function
		r.unsolicited = fragment.UNS
	} else if !r.active {
		return false, StrictFragment{}, errors.New("DNP3 application continuation without FIR")
	}
	if fragment.Sequence != r.nextSequence {
		expected := r.nextSequence
		r.Reset()
		return false, StrictFragment{}, fmt.Errorf("DNP3 application sequence %d, want %d", fragment.Sequence, expected)
	}
	if fragment.Function != r.function {
		r.Reset()
		return false, StrictFragment{}, errors.New("DNP3 application function changed between fragments")
	}
	if fragment.UNS != r.unsolicited {
		r.Reset()
		return false, StrictFragment{}, errors.New("DNP3 unsolicited flag changed between fragments")
	}
	if len(fragment.Objects) > r.maximumObjects-len(r.objects) {
		r.Reset()
		return false, StrictFragment{}, errors.New("DNP3 application objects exceed configured limit")
	}
	r.objects = append(r.objects, fragment.Objects...)
	r.confirm = r.confirm || fragment.CON
	r.iin |= fragment.IIN
	r.nextSequence = (fragment.Sequence + 1) & 0x0f
	if !fragment.FIN {
		return false, StrictFragment{}, nil
	}
	result := StrictFragment{
		FIR: true, FIN: true, CON: r.confirm, UNS: r.unsolicited,
		Sequence: r.firstSequence, Function: r.function, IIN: r.iin,
		Objects: append([]byte(nil), r.objects...),
	}
	r.Reset()
	return true, result, nil
}
