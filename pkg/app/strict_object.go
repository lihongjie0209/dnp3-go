package app

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

type StrictObjectHeader struct {
	Group     byte
	Variation byte
	Qualifier byte
	Start     uint32
	Stop      uint32
	Count     uint32
}

type StrictRangeKind uint8

const (
	StrictRangeStartStop StrictRangeKind = iota
	StrictRangeAll
	StrictRangeCount
)

type StrictQualifier struct {
	Kind        StrictRangeKind
	RangeWidth  int
	PrefixWidth int
	SizePrefix  bool
}

func EncodeStrictObjectHeader(header StrictObjectHeader) ([]byte, error) {
	descriptor, err := decodeStrictQualifier(header.Qualifier)
	if err != nil {
		return nil, err
	}
	result := []byte{header.Group, header.Variation, header.Qualifier}
	switch descriptor.Kind {
	case StrictRangeAll:
		return result, nil
	case StrictRangeStartStop:
		if header.Stop < header.Start {
			return nil, errors.New("DNP3 object stop index is below start index")
		}
		result, err = appendStrictUnsigned(result, header.Start, descriptor.RangeWidth)
		if err != nil {
			return nil, fmt.Errorf("encoding DNP3 object start: %w", err)
		}
		result, err = appendStrictUnsigned(result, header.Stop, descriptor.RangeWidth)
		if err != nil {
			return nil, fmt.Errorf("encoding DNP3 object stop: %w", err)
		}
		return result, nil
	case StrictRangeCount:
		if header.Count == 0 {
			return nil, errors.New("DNP3 object count is zero")
		}
		result, err = appendStrictUnsigned(result, header.Count, descriptor.RangeWidth)
		if err != nil {
			return nil, fmt.Errorf("encoding DNP3 object count: %w", err)
		}
		return result, nil
	default:
		return nil, errors.New("unsupported DNP3 qualifier range")
	}
}

func DecodeStrictObjectHeader(data []byte) (StrictObjectHeader, int, error) {
	if len(data) < 3 {
		return StrictObjectHeader{}, 0, errors.New("truncated DNP3 object header")
	}
	header := StrictObjectHeader{Group: data[0], Variation: data[1], Qualifier: data[2]}
	descriptor, err := decodeStrictQualifier(header.Qualifier)
	if err != nil {
		return StrictObjectHeader{}, 0, err
	}
	offset := 3
	switch descriptor.Kind {
	case StrictRangeAll:
		return header, offset, nil
	case StrictRangeStartStop:
		if len(data)-offset < 2*descriptor.RangeWidth {
			return StrictObjectHeader{}, 0, errors.New("truncated DNP3 object range")
		}
		header.Start = readStrictUnsigned(data[offset:], descriptor.RangeWidth)
		offset += descriptor.RangeWidth
		header.Stop = readStrictUnsigned(data[offset:], descriptor.RangeWidth)
		offset += descriptor.RangeWidth
		if header.Stop < header.Start {
			return StrictObjectHeader{}, 0, errors.New("DNP3 object stop index is below start index")
		}
		return header, offset, nil
	case StrictRangeCount:
		if len(data)-offset < descriptor.RangeWidth {
			return StrictObjectHeader{}, 0, errors.New("truncated DNP3 object count")
		}
		header.Count = readStrictUnsigned(data[offset:], descriptor.RangeWidth)
		if header.Count == 0 {
			return StrictObjectHeader{}, 0, errors.New("DNP3 object count is zero")
		}
		offset += descriptor.RangeWidth
		return header, offset, nil
	default:
		return StrictObjectHeader{}, 0, errors.New("unsupported DNP3 qualifier range")
	}
}

func decodeStrictQualifier(qualifier byte) (StrictQualifier, error) {
	prefixCode, rangeCode := qualifier>>4, qualifier&0x0f
	descriptor := StrictQualifier{}
	switch prefixCode {
	case 0:
	case 1, 2, 3:
		descriptor.PrefixWidth = 1 << (prefixCode - 1)
	case 4, 5, 6:
		descriptor.PrefixWidth = 1 << (prefixCode - 4)
		descriptor.SizePrefix = true
	default:
		return descriptor, fmt.Errorf("unsupported DNP3 qualifier prefix code %d", prefixCode)
	}
	switch rangeCode {
	case 0, 1, 2:
		if prefixCode != 0 {
			return descriptor, errors.New("DNP3 start-stop qualifier may not use a prefix")
		}
		descriptor.Kind = StrictRangeStartStop
		descriptor.RangeWidth = 1 << rangeCode
	case 6:
		if prefixCode != 0 {
			return descriptor, errors.New("DNP3 all-objects qualifier may not use a prefix")
		}
		descriptor.Kind = StrictRangeAll
	case 7, 8, 9:
		descriptor.Kind = StrictRangeCount
		descriptor.RangeWidth = 1 << (rangeCode - 7)
	default:
		return descriptor, fmt.Errorf("unsupported DNP3 qualifier range code %d", rangeCode)
	}
	return descriptor, nil
}

// DecodeStrictQualifier validates a DNP3 qualifier and returns its range and
// prefix representation.
func DecodeStrictQualifier(qualifier byte) (StrictQualifier, error) {
	return decodeStrictQualifier(qualifier)
}

// AppendStrictUnsigned appends a bounded little-endian qualifier integer.
func AppendStrictUnsigned(destination []byte, value uint32, width int) ([]byte, error) {
	return appendStrictUnsigned(destination, value, width)
}

// DecodeStrictUnsigned decodes a complete-width little-endian qualifier
// integer.
func DecodeStrictUnsigned(data []byte, width int) (uint32, error) {
	if width != 1 && width != 2 && width != 4 {
		return 0, errors.New("unsupported DNP3 integer width")
	}
	if len(data) < width {
		return 0, errors.New("truncated DNP3 integer")
	}
	return readStrictUnsigned(data, width), nil
}

func appendStrictUnsigned(destination []byte, value uint32, width int) ([]byte, error) {
	switch width {
	case 1:
		if value > math.MaxUint8 {
			return nil, errors.New("value exceeds one octet")
		}
		return append(destination, byte(value)), nil
	case 2:
		if value > math.MaxUint16 {
			return nil, errors.New("value exceeds two octets")
		}
		return binary.LittleEndian.AppendUint16(destination, uint16(value)), nil
	case 4:
		return binary.LittleEndian.AppendUint32(destination, value), nil
	default:
		return nil, errors.New("unsupported DNP3 integer width")
	}
}

func readStrictUnsigned(data []byte, width int) uint32 {
	switch width {
	case 1:
		return uint32(data[0])
	case 2:
		return uint32(binary.LittleEndian.Uint16(data))
	case 4:
		return binary.LittleEndian.Uint32(data)
	default:
		return 0
	}
}
