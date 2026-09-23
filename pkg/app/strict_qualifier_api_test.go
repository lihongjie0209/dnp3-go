package app_test

import (
	"reflect"
	"testing"

	"github.com/lihongjie0209/dnp3-go/pkg/app"
)

func TestStrictQualifierDescriptor(t *testing.T) {
	tests := []struct {
		name      string
		qualifier byte
		want      app.StrictQualifier
	}{
		{name: "all objects", qualifier: 0x06, want: app.StrictQualifier{Kind: app.StrictRangeAll}},
		{name: "start stop", qualifier: 0x01, want: app.StrictQualifier{Kind: app.StrictRangeStartStop, RangeWidth: 2}},
		{name: "indexed count", qualifier: 0x28, want: app.StrictQualifier{Kind: app.StrictRangeCount, RangeWidth: 2, PrefixWidth: 2}},
		{name: "sized count", qualifier: 0x57, want: app.StrictQualifier{Kind: app.StrictRangeCount, RangeWidth: 1, PrefixWidth: 2, SizePrefix: true}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := app.DecodeStrictQualifier(test.qualifier)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("descriptor = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestStrictUnsignedWidthCodec(t *testing.T) {
	raw, err := app.AppendStrictUnsigned([]byte{0xaa}, 0x1234, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(raw, []byte{0xaa, 0x34, 0x12}) {
		t.Fatalf("wire = %x", raw)
	}
	value, err := app.DecodeStrictUnsigned(raw[1:], 2)
	if err != nil {
		t.Fatal(err)
	}
	if value != 0x1234 {
		t.Fatalf("value = %#x", value)
	}
}

func TestStrictUnsignedWidthCodecRejectsInvalidInput(t *testing.T) {
	if _, err := app.AppendStrictUnsigned(nil, 256, 1); err == nil {
		t.Fatal("want overflow error")
	}
	if _, err := app.DecodeStrictUnsigned([]byte{1}, 2); err == nil {
		t.Fatal("want truncation error")
	}
	if _, err := app.DecodeStrictQualifier(0x70); err == nil {
		t.Fatal("want invalid prefix error")
	}
}
