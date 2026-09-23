package app

import (
	"slices"
	"testing"
)

func TestStrictReassemblerSequenceWrap(t *testing.T) {
	t.Parallel()
	reassembler := NewStrictReassembler(6)
	fragments := []StrictFragment{
		{FIR: true, Sequence: 15, Function: 0x81, IIN: 1, Objects: []byte{1, 2}},
		{Sequence: 0, Function: 0x81, IIN: 2, Objects: []byte{3, 4}},
		{FIN: true, CON: true, Sequence: 1, Function: 0x81, IIN: 4, Objects: []byte{5, 6}},
	}
	for index, fragment := range fragments {
		complete, result, err := reassembler.Push(fragment)
		if err != nil {
			t.Fatal(err)
		}
		if complete != (index == len(fragments)-1) {
			t.Fatalf("fragment %d complete=%v", index, complete)
		}
		if complete && (!result.FIR || !result.FIN || !result.CON || result.Sequence != 15 || result.IIN != 7 || !slices.Equal(result.Objects, []byte{1, 2, 3, 4, 5, 6})) {
			t.Fatalf("result=%#v", result)
		}
	}
}

func TestStrictReassemblerRejectsAndResets(t *testing.T) {
	t.Parallel()
	reassembler := NewStrictReassembler(2)
	if _, _, err := reassembler.Push(StrictFragment{FIN: true, Sequence: 1, Function: 0x81}); err == nil {
		t.Fatal("accepted continuation without FIR")
	}
	if complete, _, err := reassembler.Push(StrictFragment{FIR: true, FIN: true, Sequence: 0, Function: 0x81}); err != nil || !complete {
		t.Fatalf("not reusable: complete=%v err=%v", complete, err)
	}
	if _, _, err := reassembler.Push(StrictFragment{FIR: true, Sequence: 1, Function: 0x81, Objects: []byte{1, 2}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := reassembler.Push(StrictFragment{FIN: true, Sequence: 3, Function: 0x81, Objects: []byte{3}}); err == nil {
		t.Fatal("accepted gap and overflow")
	}
}
