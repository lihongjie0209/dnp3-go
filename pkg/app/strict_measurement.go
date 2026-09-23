package app

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"
)

// StrictPoint is one validated DNP3 measurement value.
type StrictPoint struct {
	Group     byte
	Variation byte
	Index     uint32
	Value     any
	Quality   byte
	Timestamp *time.Time
}

// StrictObjectBlock is a header and its complete measurement point set.
type StrictObjectBlock struct {
	Header StrictObjectHeader
	Points []StrictPoint
}

type strictValueKind uint8

const (
	strictBinary strictValueKind = iota
	strictDoubleBinary
	strictUnsigned
	strictInt16
	strictInt32
	strictFloat32
	strictFloat64
	strictOctets
)

type strictValueFormat struct {
	kind      strictValueKind
	width     int
	quality   bool
	timestamp bool
}

// StrictValueKind identifies the Go representation required by a point.
type StrictValueKind uint8

const (
	StrictValueBinary StrictValueKind = iota
	StrictValueDoubleBinary
	StrictValueUnsigned
	StrictValueInt16
	StrictValueInt32
	StrictValueFloat32
	StrictValueFloat64
	StrictValueOctets
)

// StrictPointFormat describes a supported measurement payload without
// exposing the codec's internal representation.
type StrictPointFormat struct {
	Kind       StrictValueKind
	Width      int
	HasQuality bool
	HasTime    bool
}

// DescribeStrictPoint returns the required representation for a supported
// group and variation.
func DescribeStrictPoint(group, variation byte) (StrictPointFormat, error) {
	format, err := strictPointFormat(group, variation)
	if err != nil {
		return StrictPointFormat{}, err
	}
	return StrictPointFormat{
		Kind: StrictValueKind(format.kind), Width: format.width,
		HasQuality: format.quality, HasTime: format.timestamp,
	}, nil
}

// EncodeStrictPoint encodes an exact supported measurement payload.
func EncodeStrictPoint(point StrictPoint) ([]byte, error) {
	format, err := strictPointFormat(point.Group, point.Variation)
	if err != nil {
		return nil, err
	}
	result := make([]byte, 0, format.width+7)
	if format.quality && format.kind != strictBinary && format.kind != strictDoubleBinary {
		result = append(result, point.Quality)
	}
	switch format.kind {
	case strictBinary:
		value, ok := point.Value.(bool)
		if !ok {
			return nil, errors.New("DNP3 binary value must be boolean")
		}
		if point.Quality&0x80 != 0 {
			return nil, errors.New("DNP3 binary quality contains the state bit")
		}
		flags := point.Quality
		if value {
			flags |= 0x80
		}
		result = append(result, flags)
	case strictDoubleBinary:
		value, ok := point.Value.(uint8)
		if !ok || value > 3 {
			return nil, errors.New("DNP3 double-bit value must be between 0 and 3")
		}
		if point.Quality&0xc0 != 0 {
			return nil, errors.New("DNP3 double-bit quality contains state bits")
		}
		result = append(result, point.Quality|value<<6)
	case strictUnsigned:
		value, ok := point.Value.(uint32)
		if !ok {
			return nil, errors.New("DNP3 counter value must be uint32")
		}
		result, err = appendStrictUnsigned(result, value, format.width)
		if err != nil {
			return nil, fmt.Errorf("encoding DNP3 counter: %w", err)
		}
	case strictInt16:
		value, ok := point.Value.(int16)
		if !ok {
			return nil, errors.New("DNP3 analog value must be int16")
		}
		result = binary.LittleEndian.AppendUint16(result, uint16(value))
	case strictInt32:
		value, ok := point.Value.(int32)
		if !ok {
			return nil, errors.New("DNP3 analog value must be int32")
		}
		result = binary.LittleEndian.AppendUint32(result, uint32(value))
	case strictFloat32:
		value, ok := point.Value.(float32)
		if !ok || math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return nil, errors.New("DNP3 analog value must be a finite float32")
		}
		result = binary.LittleEndian.AppendUint32(result, math.Float32bits(value))
	case strictFloat64:
		value, ok := point.Value.(float64)
		if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, errors.New("DNP3 analog value must be a finite float64")
		}
		result = binary.LittleEndian.AppendUint64(result, math.Float64bits(value))
	case strictOctets:
		value, ok := point.Value.([]byte)
		if !ok || len(value) != format.width {
			return nil, fmt.Errorf("DNP3 octet-string value must contain %d bytes", format.width)
		}
		result = append(result, value...)
	default:
		return nil, errors.New("unsupported DNP3 value kind")
	}
	if format.timestamp {
		if point.Timestamp == nil {
			return nil, errors.New("DNP3 timed object requires a timestamp")
		}
		return AppendStrictTime48(result, *point.Timestamp)
	}
	if point.Timestamp != nil {
		return nil, errors.New("DNP3 untimed object contains a timestamp")
	}
	return result, nil
}

// DecodeStrictPoint decodes one exact supported measurement payload.
func DecodeStrictPoint(group, variation byte, data []byte) (StrictPoint, int, error) {
	format, err := strictPointFormat(group, variation)
	if err != nil {
		return StrictPoint{}, 0, err
	}
	required := format.width
	if format.quality {
		required++
	}
	if format.timestamp {
		required += 6
	}
	if len(data) < required {
		return StrictPoint{}, 0, errors.New("truncated DNP3 point value")
	}
	point := StrictPoint{Group: group, Variation: variation}
	offset := 0
	if format.quality && format.kind != strictBinary && format.kind != strictDoubleBinary {
		point.Quality = data[offset]
		offset++
	}
	switch format.kind {
	case strictBinary:
		point.Value, point.Quality = data[offset]&0x80 != 0, data[offset]&0x7f
		offset++
	case strictDoubleBinary:
		point.Value, point.Quality = data[offset]>>6, data[offset]&0x3f
		offset++
	case strictUnsigned:
		point.Value = readStrictUnsigned(data[offset:], format.width)
		offset += format.width
	case strictInt16:
		point.Value = int16(binary.LittleEndian.Uint16(data[offset:]))
		offset += 2
	case strictInt32:
		point.Value = int32(binary.LittleEndian.Uint32(data[offset:]))
		offset += 4
	case strictFloat32:
		value := math.Float32frombits(binary.LittleEndian.Uint32(data[offset:]))
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return StrictPoint{}, 0, errors.New("DNP3 float32 point is not finite")
		}
		point.Value, offset = value, offset+4
	case strictFloat64:
		value := math.Float64frombits(binary.LittleEndian.Uint64(data[offset:]))
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return StrictPoint{}, 0, errors.New("DNP3 float64 point is not finite")
		}
		point.Value, offset = value, offset+8
	case strictOctets:
		point.Value = append([]byte(nil), data[offset:offset+format.width]...)
		offset += format.width
	}
	if format.timestamp {
		value, timeErr := DecodeStrictTime48(data[offset : offset+6])
		if timeErr != nil {
			return StrictPoint{}, 0, timeErr
		}
		point.Timestamp, offset = &value, offset+6
	}
	return point, offset, nil
}

// StrictVariationHasTime reports whether the supported variation carries a
// 48-bit DNP3 absolute timestamp.
func StrictVariationHasTime(group, variation byte) bool {
	format, err := strictPointFormat(group, variation)
	return err == nil && format.timestamp
}

// IsStrictPackedObject reports whether the variation uses packed point bits.
func IsStrictPackedObject(group, variation byte) bool {
	return strictPackedObject(group, variation)
}

// ValidateStrictPointVariation validates a supported measurement group/variation.
func ValidateStrictPointVariation(group, variation byte) error {
	_, err := strictPointFormat(group, variation)
	return err
}

// AppendStrictTime48 appends milliseconds since Unix epoch as a 48-bit value.
func AppendStrictTime48(destination []byte, value time.Time) ([]byte, error) {
	milliseconds := value.UTC().UnixMilli()
	if milliseconds < 0 || uint64(milliseconds) > 1<<48-1 {
		return nil, errors.New("DNP3 absolute time is outside the 48-bit range")
	}
	encoded := uint64(milliseconds)
	return append(destination, byte(encoded), byte(encoded>>8), byte(encoded>>16), byte(encoded>>24), byte(encoded>>32), byte(encoded>>40)), nil
}

// DecodeStrictTime48 decodes milliseconds since Unix epoch from six bytes.
func DecodeStrictTime48(data []byte) (time.Time, error) {
	if len(data) < 6 {
		return time.Time{}, errors.New("truncated DNP3 absolute time")
	}
	milliseconds := uint64(data[0]) | uint64(data[1])<<8 | uint64(data[2])<<16 | uint64(data[3])<<24 | uint64(data[4])<<32 | uint64(data[5])<<40
	return time.UnixMilli(int64(milliseconds)).UTC(), nil
}

func strictPointFormat(group, variation byte) (strictValueFormat, error) {
	switch group {
	case 1, 10:
		if variation == 2 {
			return strictValueFormat{kind: strictBinary, quality: true}, nil
		}
	case 2, 11:
		if variation == 1 || variation == 2 {
			return strictValueFormat{kind: strictBinary, quality: true, timestamp: variation == 2}, nil
		}
	case 3:
		if variation == 2 {
			return strictValueFormat{kind: strictDoubleBinary, quality: true}, nil
		}
	case 4:
		if variation == 1 || variation == 2 {
			return strictValueFormat{kind: strictDoubleBinary, quality: true, timestamp: variation == 2}, nil
		}
	case 20:
		return strictCounterFormat(variation, false)
	case 21:
		return strictFrozenCounterFormat(variation)
	case 22, 23:
		return strictCounterFormat(variation, true)
	case 30:
		return strictAnalogFormat(variation, false)
	case 32, 42:
		return strictAnalogFormat(variation, true)
	case 40:
		return strictAnalogOutputFormat(variation)
	case 110, 111:
		if variation != 0 {
			return strictValueFormat{kind: strictOctets, width: int(variation)}, nil
		}
	}
	return strictValueFormat{}, fmt.Errorf("unsupported DNP3 object group %d variation %d", group, variation)
}

func strictCounterFormat(variation byte, timed bool) (strictValueFormat, error) {
	switch variation {
	case 1:
		return strictValueFormat{kind: strictUnsigned, width: 4, quality: true}, nil
	case 2:
		return strictValueFormat{kind: strictUnsigned, width: 2, quality: true}, nil
	case 5:
		if timed {
			return strictValueFormat{kind: strictUnsigned, width: 4, quality: true, timestamp: true}, nil
		}
		return strictValueFormat{kind: strictUnsigned, width: 4}, nil
	case 6:
		if timed {
			return strictValueFormat{kind: strictUnsigned, width: 2, quality: true, timestamp: true}, nil
		}
		return strictValueFormat{kind: strictUnsigned, width: 2}, nil
	}
	return strictValueFormat{}, fmt.Errorf("unsupported DNP3 counter variation %d", variation)
}

func strictFrozenCounterFormat(variation byte) (strictValueFormat, error) {
	switch variation {
	case 1:
		return strictValueFormat{kind: strictUnsigned, width: 4, quality: true}, nil
	case 2:
		return strictValueFormat{kind: strictUnsigned, width: 2, quality: true}, nil
	case 5:
		return strictValueFormat{kind: strictUnsigned, width: 4, quality: true, timestamp: true}, nil
	case 6:
		return strictValueFormat{kind: strictUnsigned, width: 2, quality: true, timestamp: true}, nil
	case 9:
		return strictValueFormat{kind: strictUnsigned, width: 4}, nil
	case 10:
		return strictValueFormat{kind: strictUnsigned, width: 2}, nil
	}
	return strictValueFormat{}, fmt.Errorf("unsupported DNP3 frozen-counter variation %d", variation)
}

func strictAnalogFormat(variation byte, event bool) (strictValueFormat, error) {
	timed := event && (variation == 3 || variation == 4 || variation == 7 || variation == 8)
	switch variation {
	case 1, 3:
		return strictValueFormat{kind: strictInt32, width: 4, quality: variation != 3 || event, timestamp: timed}, nil
	case 2, 4:
		return strictValueFormat{kind: strictInt16, width: 2, quality: variation != 4 || event, timestamp: timed}, nil
	case 5, 7:
		if !event && variation == 7 {
			break
		}
		return strictValueFormat{kind: strictFloat32, width: 4, quality: true, timestamp: timed}, nil
	case 6, 8:
		if !event && variation == 8 {
			break
		}
		return strictValueFormat{kind: strictFloat64, width: 8, quality: true, timestamp: timed}, nil
	}
	return strictValueFormat{}, fmt.Errorf("unsupported DNP3 analog variation %d", variation)
}

func strictAnalogOutputFormat(variation byte) (strictValueFormat, error) {
	switch variation {
	case 1:
		return strictValueFormat{kind: strictInt32, width: 4, quality: true}, nil
	case 2:
		return strictValueFormat{kind: strictInt16, width: 2, quality: true}, nil
	case 3:
		return strictValueFormat{kind: strictFloat32, width: 4, quality: true}, nil
	case 4:
		return strictValueFormat{kind: strictFloat64, width: 8, quality: true}, nil
	}
	return strictValueFormat{}, fmt.Errorf("unsupported DNP3 analog-output variation %d", variation)
}

// EncodeStrictObjectBlock encodes a complete measurement block.
func EncodeStrictObjectBlock(block StrictObjectBlock) ([]byte, error) {
	header, points := block.Header, block.Points
	descriptor, err := decodeStrictQualifier(header.Qualifier)
	if err != nil {
		return nil, err
	}
	count, err := strictHeaderCount(header, descriptor)
	if err != nil {
		return nil, err
	}
	if uint64(len(points)) != count {
		return nil, fmt.Errorf("DNP3 object block has %d points, want %d", len(points), count)
	}
	if descriptor.sizePrefix {
		return nil, errors.New("DNP3 measurement blocks do not support object-size prefixes")
	}
	result, err := EncodeStrictObjectHeader(header)
	if err != nil {
		return nil, err
	}
	if count == 0 {
		if !strictSelectorObject(header.Group, header.Variation) {
			return nil, fmt.Errorf("unsupported all-objects selector group %d variation %d", header.Group, header.Variation)
		}
		return result, nil
	}
	if strictPackedObject(header.Group, header.Variation) {
		if descriptor.prefixWidth != 0 {
			return nil, errors.New("DNP3 packed objects may not use index prefixes")
		}
		packed, packErr := encodeStrictPackedPoints(header, points)
		return append(result, packed...), packErr
	}
	for index, point := range points {
		if point.Group != header.Group || point.Variation != header.Variation {
			return nil, fmt.Errorf("DNP3 point %d group or variation differs from its header", index)
		}
		expected := uint32(index)
		if descriptor.kind == strictRangeStartStop {
			expected = header.Start + uint32(index)
		}
		if descriptor.prefixWidth == 0 {
			if point.Index != expected {
				return nil, fmt.Errorf("DNP3 point %d index is %d, want %d", index, point.Index, expected)
			}
		} else {
			result, err = appendStrictUnsigned(result, point.Index, descriptor.prefixWidth)
			if err != nil {
				return nil, fmt.Errorf("encoding DNP3 point %d index: %w", index, err)
			}
		}
		encoded, encodeErr := EncodeStrictPoint(point)
		if encodeErr != nil {
			return nil, fmt.Errorf("encoding DNP3 point %d: %w", index, encodeErr)
		}
		result = append(result, encoded...)
	}
	return result, nil
}

// DecodeStrictObjectBlock decodes one complete measurement block.
func DecodeStrictObjectBlock(data []byte, maximumPoints int) (StrictObjectBlock, int, error) {
	if maximumPoints < 1 {
		return StrictObjectBlock{}, 0, errors.New("DNP3 maximum points must be positive")
	}
	header, offset, err := DecodeStrictObjectHeader(data)
	if err != nil {
		return StrictObjectBlock{}, 0, err
	}
	descriptor, err := decodeStrictQualifier(header.Qualifier)
	if err != nil {
		return StrictObjectBlock{}, 0, err
	}
	if descriptor.sizePrefix {
		return StrictObjectBlock{}, 0, errors.New("DNP3 measurement blocks do not support object-size prefixes")
	}
	count, err := strictHeaderCount(header, descriptor)
	if err != nil {
		return StrictObjectBlock{}, 0, err
	}
	if count == 0 {
		if !strictSelectorObject(header.Group, header.Variation) {
			return StrictObjectBlock{}, 0, fmt.Errorf("unsupported all-objects selector group %d variation %d", header.Group, header.Variation)
		}
		return StrictObjectBlock{Header: header}, offset, nil
	}
	if count > uint64(maximumPoints) || count > uint64(math.MaxInt) {
		return StrictObjectBlock{}, 0, errors.New("DNP3 object block exceeds point limit")
	}
	if strictPackedObject(header.Group, header.Variation) {
		if descriptor.prefixWidth != 0 {
			return StrictObjectBlock{}, 0, errors.New("DNP3 packed objects may not use index prefixes")
		}
		points, consumed, decodeErr := decodeStrictPackedPoints(header, data[offset:], int(count))
		return StrictObjectBlock{Header: header, Points: points}, offset + consumed, decodeErr
	}
	points := make([]StrictPoint, 0, int(count))
	for index := 0; index < int(count); index++ {
		pointIndex := uint32(index)
		if descriptor.kind == strictRangeStartStop {
			pointIndex = header.Start + uint32(index)
		} else if descriptor.prefixWidth != 0 {
			if len(data)-offset < descriptor.prefixWidth {
				return StrictObjectBlock{}, 0, errors.New("truncated DNP3 point index prefix")
			}
			pointIndex = readStrictUnsigned(data[offset:], descriptor.prefixWidth)
			offset += descriptor.prefixWidth
		}
		point, consumed, decodeErr := DecodeStrictPoint(header.Group, header.Variation, data[offset:])
		if decodeErr != nil {
			return StrictObjectBlock{}, 0, fmt.Errorf("decoding DNP3 point %d: %w", index, decodeErr)
		}
		point.Index = pointIndex
		points = append(points, point)
		offset += consumed
	}
	return StrictObjectBlock{Header: header, Points: points}, offset, nil
}

// DecodeStrictObjectBlocks decodes all blocks under aggregate bounds.
func DecodeStrictObjectBlocks(data []byte, maximumPoints, maximumBlocks int) ([]StrictObjectBlock, error) {
	if maximumPoints < 1 || maximumBlocks < 1 {
		return nil, errors.New("DNP3 object block and point limits must be positive")
	}
	blocks := make([]StrictObjectBlock, 0, min(maximumBlocks, 8))
	offset, pointCount := 0, 0
	for offset < len(data) {
		if len(blocks) == maximumBlocks {
			return nil, errors.New("DNP3 application fragment exceeds object block limit")
		}
		block, consumed, err := DecodeStrictObjectBlock(data[offset:], maximumPoints-pointCount)
		if err != nil {
			return nil, fmt.Errorf("decoding DNP3 object block %d: %w", len(blocks), err)
		}
		if consumed <= 0 {
			return nil, errors.New("DNP3 object decoder made no progress")
		}
		blocks = append(blocks, block)
		pointCount += len(block.Points)
		offset += consumed
	}
	return blocks, nil
}

func strictHeaderCount(header StrictObjectHeader, descriptor strictQualifier) (uint64, error) {
	switch descriptor.kind {
	case strictRangeAll:
		return 0, nil
	case strictRangeStartStop:
		if header.Stop < header.Start {
			return 0, errors.New("DNP3 object stop index is below start index")
		}
		return uint64(header.Stop) - uint64(header.Start) + 1, nil
	case strictRangeCount:
		if header.Count == 0 {
			return 0, errors.New("DNP3 object count is zero")
		}
		return uint64(header.Count), nil
	}
	return 0, errors.New("unsupported DNP3 qualifier range")
}

func strictSelectorObject(group, variation byte) bool {
	return group == 60 && variation >= 1 && variation <= 4
}
func strictPackedObject(group, variation byte) bool {
	return variation == 1 && (group == 1 || group == 3 || group == 10)
}

func encodeStrictPackedPoints(header StrictObjectHeader, points []StrictPoint) ([]byte, error) {
	bits := 1
	if header.Group == 3 {
		bits = 2
	}
	result := make([]byte, (len(points)*bits+7)/8)
	for index, point := range points {
		if point.Group != header.Group || point.Variation != header.Variation || point.Index != header.Start+uint32(index) {
			return nil, fmt.Errorf("DNP3 packed point %d does not match its header", index)
		}
		if point.Quality != 0 || point.Timestamp != nil {
			return nil, fmt.Errorf("DNP3 packed point %d contains quality or time", index)
		}
		bitOffset := index * bits
		if bits == 1 {
			value, ok := point.Value.(bool)
			if !ok {
				return nil, fmt.Errorf("DNP3 packed point %d value must be boolean", index)
			}
			if value {
				result[bitOffset/8] |= 1 << (bitOffset % 8)
			}
		} else {
			value, ok := point.Value.(uint8)
			if !ok || value > 3 {
				return nil, fmt.Errorf("DNP3 packed point %d double-bit value is invalid", index)
			}
			result[bitOffset/8] |= value << (bitOffset % 8)
		}
	}
	return result, nil
}

func decodeStrictPackedPoints(header StrictObjectHeader, data []byte, count int) ([]StrictPoint, int, error) {
	bits := 1
	if header.Group == 3 {
		bits = 2
	}
	byteCount := (count*bits + 7) / 8
	if len(data) < byteCount {
		return nil, 0, errors.New("truncated DNP3 packed object data")
	}
	points := make([]StrictPoint, count)
	for index := range points {
		bitOffset := index * bits
		point := StrictPoint{Group: header.Group, Variation: header.Variation, Index: header.Start + uint32(index)}
		if bits == 1 {
			point.Value = data[bitOffset/8]&(1<<(bitOffset%8)) != 0
		} else {
			point.Value = data[bitOffset/8] >> (bitOffset % 8) & 3
		}
		points[index] = point
	}
	unused := byteCount*8 - count*bits
	if unused > 0 && data[byteCount-1]>>uint(8-unused) != 0 {
		return nil, 0, errors.New("DNP3 packed object has nonzero padding bits")
	}
	return points, byteCount, nil
}
