package app

import (
	"bytes"
	"testing"
)

func TestStrictFragmentRoundTripAndOwnership(t *testing.T) {
	for _, test := range []struct {
		name string
		in   StrictFragment
		want []byte
	}{
		{name: "request", in: StrictFragment{FIR: true, FIN: true, Sequence: 3, Function: 1, Objects: []byte{60, 1, 6}}, want: []byte{0xc3, 1, 60, 1, 6}},
		{name: "response", in: StrictFragment{FIR: true, FIN: true, CON: true, Sequence: 15, Function: 0x81, IIN: 0x8201, Objects: []byte{1, 2}}, want: []byte{0xef, 0x81, 1, 0x82, 1, 2}},
		{name: "unsolicited", in: StrictFragment{FIR: true, FIN: true, CON: true, UNS: true, Function: 0x82, IIN: 2}, want: []byte{0xf0, 0x82, 2, 0}},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := EncodeStrictFragment(test.in, 2048)
			if err != nil || !bytes.Equal(encoded, test.want) {
				t.Fatalf("encoded=%x err=%v want=%x", encoded, err, test.want)
			}
			decoded, err := DecodeStrictFragment(encoded, 2048)
			if err != nil || decoded.FIR != test.in.FIR || decoded.FIN != test.in.FIN || decoded.CON != test.in.CON || decoded.UNS != test.in.UNS || decoded.Sequence != test.in.Sequence || decoded.Function != test.in.Function || decoded.IIN != test.in.IIN || !bytes.Equal(decoded.Objects, test.in.Objects) {
				t.Fatalf("decoded=%+v err=%v", decoded, err)
			}
			if len(decoded.Objects) > 0 {
				encoded[len(encoded)-1] ^= 0xff
				if !bytes.Equal(decoded.Objects, test.in.Objects) {
					t.Fatal("decoded objects alias input")
				}
			}
		})
	}
}

func TestStrictFragmentRejectsInvalidBounds(t *testing.T) {
	if _, err := EncodeStrictFragment(StrictFragment{Sequence: 16}, 10); err == nil {
		t.Fatal("accepted sequence above 15")
	}
	if _, err := EncodeStrictFragment(StrictFragment{Objects: make([]byte, 9)}, 10); err == nil {
		t.Fatal("accepted oversized request")
	}
	for _, test := range []struct {
		data []byte
		max  int
	}{
		{data: nil, max: 10},
		{data: []byte{0xc0}, max: 10},
		{data: []byte{0xc0, 0x81, 0}, max: 10},
		{data: []byte{0xc0, 1}, max: 1},
		{data: []byte{0xc0, 1}, max: 0},
	} {
		if _, err := DecodeStrictFragment(test.data, test.max); err == nil {
			t.Fatalf("accepted data=%x max=%d", test.data, test.max)
		}
	}
}
