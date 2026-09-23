package link

import (
	"errors"
	"slices"
	"testing"
)

func TestStrictStateResetConfirmedAndDuplicateFlow(t *testing.T) {
	t.Parallel()
	master := NewStrictState(1, 10, true)
	outstation := NewStrictState(10, 1, false)
	reset, err := master.StartReset()
	if err != nil {
		t.Fatal(err)
	}
	resetAction, err := outstation.Handle(reset)
	if err != nil {
		t.Fatal(err)
	}
	if resetAction.Response == nil || resetAction.Response.Control != 0 || !resetAction.Reset {
		t.Fatalf("reset=%#v", resetAction)
	}
	ack, err := master.Handle(*resetAction.Response)
	if err != nil {
		t.Fatal(err)
	}
	if !ack.Acknowledged || !master.Ready() || !outstation.Ready() {
		t.Fatalf("ack=%#v", ack)
	}

	confirmed, err := master.StartConfirmed([]byte{1, 2, 3})
	if err != nil {
		t.Fatal(err)
	}
	if confirmed.Control&0x3f != 0x33 {
		t.Fatalf("control=%02x", confirmed.Control)
	}
	delivered, err := outstation.Handle(confirmed)
	if err != nil {
		t.Fatal(err)
	}
	if delivered.Response == nil || delivered.Duplicate || !slices.Equal(delivered.UserData, []byte{1, 2, 3}) {
		t.Fatalf("delivered=%#v", delivered)
	}
	duplicate, err := outstation.Handle(confirmed)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.Response == nil || !duplicate.Duplicate || len(duplicate.UserData) != 0 {
		t.Fatalf("duplicate=%#v", duplicate)
	}
	if _, err = master.Handle(*delivered.Response); err != nil {
		t.Fatal(err)
	}
	next, err := master.StartConfirmed([]byte{4})
	if err != nil {
		t.Fatal(err)
	}
	if next.Control&CtrlFCB != 0 {
		t.Fatalf("next control=%02x", next.Control)
	}
}

func TestStrictStateRetransmissionAndDefensiveCopies(t *testing.T) {
	t.Parallel()
	master := NewStrictState(1, 10, true)
	master.MarkReset()
	payload := []byte{9, 8}
	frame, err := master.StartConfirmed(payload)
	if err != nil {
		t.Fatal(err)
	}
	payload[0], frame.UserData[1] = 1, 1
	action, err := master.Handle(StrictFrame{Control: 1, Destination: 1, Source: 10})
	if err != nil {
		t.Fatal(err)
	}
	if !action.NegativeAcknowledgement {
		t.Fatalf("action=%#v", action)
	}
	retry, err := master.Retransmission()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(retry.UserData, []byte{9, 8}) {
		t.Fatalf("retry=%#v", retry)
	}
	if _, err = master.StartConfirmed([]byte{7}); !errors.Is(err, ErrStrictRequestPending) {
		t.Fatalf("error=%v", err)
	}
}

func TestStrictStateRejectsMismatchedFrames(t *testing.T) {
	t.Parallel()
	state := NewStrictState(10, 1, false)
	frames := []StrictFrame{
		{Control: 0xc4, Destination: 11, Source: 1, UserData: []byte{1}},
		{Control: 0xc4, Destination: 10, Source: 2, UserData: []byte{1}},
		{Control: 0x44, Destination: 10, Source: 1, UserData: []byte{1}},
		{Control: 0xc0, Destination: 10, Source: 1, UserData: []byte{1}},
		{Control: 0xc4, Destination: 10, Source: 1},
	}
	for _, frame := range frames {
		if _, err := state.Handle(frame); err == nil {
			t.Fatalf("accepted %#v", frame)
		}
	}
}
