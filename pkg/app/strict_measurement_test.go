package app

import (
	"math"
	"reflect"
	"testing"
	"time"
)

func TestStrictPointValueRoundTrip(t *testing.T) {
	t.Parallel()
	timestamp := time.Date(2026, time.September, 23, 1, 2, 3, 456_000_000, time.UTC)
	tests := []struct {
		name      string
		group     byte
		variation byte
		value     any
		quality   byte
		timestamp *time.Time
	}{
		{name: "binary", group: 1, variation: 2, value: true, quality: 1},
		{name: "timed double bit", group: 4, variation: 2, value: uint8(3), quality: 1, timestamp: &timestamp},
		{name: "counter16", group: 20, variation: 2, value: uint32(math.MaxUint16), quality: 1},
		{name: "frozen timed", group: 21, variation: 5, value: uint32(9), quality: 1, timestamp: &timestamp},
		{name: "analog int32", group: 30, variation: 1, value: int32(-42), quality: 1},
		{name: "analog float64 timed", group: 32, variation: 8, value: float64(42.5), quality: 1, timestamp: &timestamp},
		{name: "output float32", group: 40, variation: 3, value: float32(3.5), quality: 1},
		{name: "octets", group: 110, variation: 3, value: []byte{1, 2, 3}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			input := StrictPoint{Group: test.group, Variation: test.variation, Value: test.value, Quality: test.quality, Timestamp: test.timestamp}
			encoded, err := EncodeStrictPoint(input)
			if err != nil {
				t.Fatal(err)
			}
			decoded, consumed, err := DecodeStrictPoint(test.group, test.variation, encoded)
			if err != nil {
				t.Fatal(err)
			}
			if consumed != len(encoded) || decoded.Group != test.group || decoded.Variation != test.variation || decoded.Quality != test.quality || !reflect.DeepEqual(decoded.Value, test.value) {
				t.Fatalf("decoded=%#v consumed=%d encoded=% X", decoded, consumed, encoded)
			}
			if test.timestamp == nil && decoded.Timestamp != nil || test.timestamp != nil && (decoded.Timestamp == nil || !decoded.Timestamp.Equal(*test.timestamp)) {
				t.Fatalf("timestamp=%v want=%v", decoded.Timestamp, test.timestamp)
			}
		})
	}
}

func TestStrictMeasurementBlockRoundTrip(t *testing.T) {
	t.Parallel()
	block := StrictObjectBlock{
		Header: StrictObjectHeader{Group: 30, Variation: 1, Qualifier: 0x17, Count: 2},
		Points: []StrictPoint{
			{Group: 30, Variation: 1, Index: 5, Value: int32(-1), Quality: 1},
			{Group: 30, Variation: 1, Index: 250, Value: int32(2), Quality: 2},
		},
	}
	encoded, err := EncodeStrictObjectBlock(block)
	if err != nil {
		t.Fatal(err)
	}
	decoded, consumed, err := DecodeStrictObjectBlock(encoded, 10)
	if err != nil {
		t.Fatal(err)
	}
	if consumed != len(encoded) || !reflect.DeepEqual(decoded, block) {
		t.Fatalf("decoded=%#v consumed=%d encoded=% X", decoded, consumed, encoded)
	}
}

func TestStrictPackedBlockAndLimits(t *testing.T) {
	t.Parallel()
	block := StrictObjectBlock{Header: StrictObjectHeader{Group: 1, Variation: 1, Qualifier: 0, Start: 3, Stop: 12}}
	for index, value := range []bool{true, false, true, true, false, false, true, false, true, true} {
		block.Points = append(block.Points, StrictPoint{Group: 1, Variation: 1, Index: 3 + uint32(index), Value: value})
	}
	encoded, err := EncodeStrictObjectBlock(block)
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{1, 1, 0, 3, 12, 0x4d, 0x03}
	if !reflect.DeepEqual(encoded, want) {
		t.Fatalf("encoded=% X want=% X", encoded, want)
	}
	if _, _, err := DecodeStrictObjectBlock(encoded, 9); err == nil {
		t.Fatal("accepted block over point limit")
	}
	malformed := append([]byte(nil), encoded...)
	malformed[len(malformed)-1] |= 0x80
	if _, _, err := DecodeStrictObjectBlock(malformed, 10); err == nil {
		t.Fatal("accepted nonzero packed padding")
	}
}

func TestStrictPointRejectsInvalidValues(t *testing.T) {
	t.Parallel()
	tests := []StrictPoint{
		{Group: 1, Variation: 2, Value: "true"},
		{Group: 3, Variation: 2, Value: uint8(4)},
		{Group: 20, Variation: 2, Value: uint32(65536)},
		{Group: 30, Variation: 5, Value: float32(math.Inf(1))},
		{Group: 110, Variation: 2, Value: []byte{1}},
		{Group: 99, Variation: 1},
	}
	for _, test := range tests {
		if _, err := EncodeStrictPoint(test); err == nil {
			t.Fatalf("accepted %#v", test)
		}
	}
}

func TestDescribeStrictPoint(t *testing.T) {
	t.Parallel()
	format, err := DescribeStrictPoint(32, 7)
	if err != nil {
		t.Fatal(err)
	}
	want := StrictPointFormat{Kind: StrictValueFloat32, Width: 4, HasQuality: true, HasTime: true}
	if format != want {
		t.Fatalf("format=%#v want=%#v", format, want)
	}
}
