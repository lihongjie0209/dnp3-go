package transport

import (
	"bytes"
	"testing"
)

func TestStrictTransportSegmentsAndReassembles(t *testing.T) {
	fragment := make([]byte, 700)
	for index := range fragment {
		fragment[index] = byte(index)
	}
	segments, next, err := SegmentStrict(fragment, 62, 250)
	if err != nil || len(segments) != 3 || next != 1 {
		t.Fatalf("segments=%d next=%d err=%v", len(segments), next, err)
	}
	if segments[0][0] != 0xbe || segments[1][0] != 0x3f || segments[2][0] != 0x40 {
		t.Fatalf("headers=%02x %02x %02x", segments[0][0], segments[1][0], segments[2][0])
	}
	reassembler, err := NewStrictReassembler(700)
	if err != nil {
		t.Fatal(err)
	}
	for index, segment := range segments {
		complete, value, pushErr := reassembler.Push(segment)
		if pushErr != nil || complete != (index == len(segments)-1) {
			t.Fatalf("segment=%d complete=%v err=%v", index, complete, pushErr)
		}
		if complete {
			if !bytes.Equal(value, fragment) {
				t.Fatal("reassembled fragment mismatch")
			}
			segment[1] ^= 0xff
			if !bytes.Equal(value, fragment) {
				t.Fatal("completed fragment aliases segment")
			}
		}
	}
}

func TestStrictTransportRejectsMalformedSequences(t *testing.T) {
	if _, _, err := SegmentStrict(nil, 0, 250); err == nil {
		t.Fatal("accepted empty fragment")
	}
	if _, _, err := SegmentStrict([]byte{1}, 0, 1); err == nil {
		t.Fatal("accepted one-byte link payload")
	}
	if _, err := NewStrictReassembler(0); err == nil {
		t.Fatal("accepted zero fragment bound")
	}
	r, _ := NewStrictReassembler(3)
	for _, segment := range [][]byte{nil, {0x40, 1}, {0xc0, 1, 2, 3, 4}} {
		if _, _, err := r.Push(segment); err == nil {
			t.Fatalf("accepted malformed segment %x", segment)
		}
		r.Reset()
	}
	complete, _, err := r.Push([]byte{0x80, 1})
	if err != nil || complete {
		t.Fatalf("first complete=%v err=%v", complete, err)
	}
	if _, _, err = r.Push([]byte{0x02, 2}); err == nil {
		t.Fatal("accepted sequence gap")
	}
}
