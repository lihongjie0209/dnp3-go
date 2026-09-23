package app_test

import (
	"math"
	"reflect"
	"slices"
	"testing"

	"github.com/lihongjie0209/dnp3-go/pkg/app"
)

func TestStrictCROBCodec(t *testing.T) {
	t.Parallel()
	input := app.StrictCommand{Group: 12, Variation: 1, Index: 7, Operation: app.CROBLatchOn, TripClose: app.CROBTrip, Count: 3, OnTime: 500, OffTime: 1000}
	encoded, err := app.EncodeStrictCommandValue(input)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x83, 3, 0xf4, 1, 0, 0, 0xe8, 3, 0, 0, 0}
	if !slices.Equal(encoded, want) {
		t.Fatalf("encoded=% X, want % X", encoded, want)
	}
	decoded, consumed, err := app.DecodeStrictCommandValue(12, 1, encoded)
	if err != nil {
		t.Fatal(err)
	}
	decoded.Index = input.Index
	if consumed != len(encoded) || !reflect.DeepEqual(decoded, input) {
		t.Fatalf("decoded=%#v consumed=%d", decoded, consumed)
	}
}

func TestStrictAnalogCommandCodec(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name      string
		variation byte
		value     any
	}{
		{name: "int32", variation: 1, value: int32(math.MinInt32)},
		{name: "int16", variation: 2, value: int16(math.MinInt16)},
		{name: "float32", variation: 3, value: float32(1.25)},
		{name: "float64", variation: 4, value: float64(-2.5)},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			input := app.StrictCommand{Group: 41, Variation: test.variation, Index: 123, Value: test.value, Status: 2}
			encoded, err := app.EncodeStrictCommandValue(input)
			if err != nil {
				t.Fatal(err)
			}
			decoded, consumed, err := app.DecodeStrictCommandValue(41, test.variation, encoded)
			if err != nil {
				t.Fatal(err)
			}
			decoded.Index = input.Index
			if consumed != len(encoded) || !reflect.DeepEqual(decoded, input) {
				t.Fatalf("decoded=%#v consumed=%d", decoded, consumed)
			}
		})
	}
}

func TestStrictCommandBlockRoundTrip(t *testing.T) {
	t.Parallel()
	header := app.StrictObjectHeader{Group: 41, Variation: 1, Qualifier: 0x28, Count: 2}
	commands := []app.StrictCommand{{Group: 41, Variation: 1, Index: 300, Value: int32(1)}, {Group: 41, Variation: 1, Index: 301, Value: int32(2), Status: 1}}
	encoded, err := app.EncodeStrictCommandBlock(header, commands)
	if err != nil {
		t.Fatal(err)
	}
	decodedHeader, decodedCommands, consumed, err := app.DecodeStrictCommandBlock(encoded, 10)
	if err != nil {
		t.Fatal(err)
	}
	if decodedHeader != header || consumed != len(encoded) || !reflect.DeepEqual(decodedCommands, commands) {
		t.Fatalf("header=%#v commands=%#v consumed=%d", decodedHeader, decodedCommands, consumed)
	}
}

func TestStrictCommandCodecRejectsInvalidValues(t *testing.T) {
	t.Parallel()
	for _, command := range []app.StrictCommand{
		{Group: 12, Variation: 1, Operation: 0},
		{Group: 12, Variation: 1, Operation: app.CROBPulseOn, TripClose: 3},
		{Group: 12, Variation: 1, Operation: app.CROBPulseOn, Count: 0},
		{Group: 41, Variation: 1, Value: int16(1)},
		{Group: 41, Variation: 3, Value: float32(math.Inf(1))},
		{Group: 99, Variation: 1},
	} {
		if _, err := app.EncodeStrictCommandValue(command); err == nil {
			t.Fatalf("accepted command %#v", command)
		}
	}
	for _, data := range [][]byte{nil, {0x11}, {0xf1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0}} {
		if _, _, err := app.DecodeStrictCommandValue(12, 1, data); err == nil {
			t.Fatalf("accepted CROB % X", data)
		}
	}
	if _, err := app.EncodeStrictCommandBlock(app.StrictObjectHeader{Group: 41, Variation: 1, Qualifier: 7, Count: 1}, []app.StrictCommand{{Group: 41, Variation: 1, Value: int32(1)}}); err == nil {
		t.Fatal("accepted command block without indexes")
	}
}
