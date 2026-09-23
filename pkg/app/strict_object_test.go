package app

import (
	"bytes"
	"testing"
)

func TestStrictObjectHeaderRoundTrip(t *testing.T) {
	for _, test := range []struct {
		name string
		in   StrictObjectHeader
		want []byte
	}{
		{name: "all", in: StrictObjectHeader{Group: 60, Variation: 1, Qualifier: 6}, want: []byte{60, 1, 6}},
		{name: "range8", in: StrictObjectHeader{Group: 1, Variation: 2, Qualifier: 0, Start: 3, Stop: 9}, want: []byte{1, 2, 0, 3, 9}},
		{name: "range16", in: StrictObjectHeader{Group: 30, Variation: 5, Qualifier: 1, Start: 256, Stop: 500}, want: []byte{30, 5, 1, 0, 1, 0xf4, 1}},
		{name: "range32", in: StrictObjectHeader{Group: 110, Qualifier: 2, Start: 65536, Stop: 65538}, want: []byte{110, 0, 2, 0, 0, 1, 0, 2, 0, 1, 0}},
		{name: "count8", in: StrictObjectHeader{Group: 2, Variation: 1, Qualifier: 7, Count: 4}, want: []byte{2, 1, 7, 4}},
		{name: "indexed count", in: StrictObjectHeader{Group: 32, Variation: 7, Qualifier: 0x28, Count: 2}, want: []byte{32, 7, 0x28, 2, 0}},
		{name: "sized count", in: StrictObjectHeader{Group: 70, Variation: 5, Qualifier: 0x57, Count: 1}, want: []byte{70, 5, 0x57, 1}},
	} {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := EncodeStrictObjectHeader(test.in)
			if err != nil || !bytes.Equal(encoded, test.want) {
				t.Fatalf("encoded=%x err=%v want=%x", encoded, err, test.want)
			}
			withBody := append(append([]byte(nil), encoded...), 0xaa)
			decoded, consumed, err := DecodeStrictObjectHeader(withBody)
			if err != nil || consumed != len(encoded) || decoded != test.in {
				t.Fatalf("decoded=%+v consumed=%d err=%v", decoded, consumed, err)
			}
		})
	}
}

func TestStrictObjectHeaderRejectsMalformedQualifiers(t *testing.T) {
	for _, header := range []StrictObjectHeader{
		{Group: 1, Variation: 1, Qualifier: 3},
		{Group: 1, Variation: 1, Qualifier: 0x16},
		{Group: 1, Variation: 1, Qualifier: 0, Start: 10, Stop: 9},
		{Group: 1, Variation: 1, Qualifier: 0, Stop: 256},
		{Group: 1, Variation: 1, Qualifier: 7, Count: 256},
	} {
		if _, err := EncodeStrictObjectHeader(header); err == nil {
			t.Fatalf("accepted %+v", header)
		}
	}
	for _, data := range [][]byte{nil, {1, 2}, {1, 2, 3}, {1, 2, 0, 10}, {1, 2, 0, 10, 9}, {1, 2, 0x16}} {
		if _, _, err := DecodeStrictObjectHeader(data); err == nil {
			t.Fatalf("accepted %x", data)
		}
	}
}
