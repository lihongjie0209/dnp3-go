package app

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

const (
	CROBPulseOn  byte = 1
	CROBPulseOff byte = 2
	CROBLatchOn  byte = 3
	CROBLatchOff byte = 4

	CROBNul   byte = 0
	CROBClose byte = 1
	CROBTrip  byte = 2
)

type StrictCommand struct {
	Group     byte
	Variation byte
	Index     uint32
	Operation byte
	TripClose byte
	Count     byte
	OnTime    uint32
	OffTime   uint32
	Value     any
	Status    byte
}

func EncodeStrictCommandValue(command StrictCommand) ([]byte, error) {
	if command.Status > 0x7f {
		return nil, errors.New("DNP3 command status exceeds seven bits")
	}
	if command.Group == 12 && command.Variation == 1 {
		if command.Operation < CROBPulseOn || command.Operation > CROBLatchOff {
			return nil, errors.New("DNP3 CROB operation is invalid")
		}
		if command.TripClose > CROBTrip {
			return nil, errors.New("DNP3 CROB trip/close code is invalid")
		}
		if command.Count == 0 {
			return nil, errors.New("DNP3 CROB count is zero")
		}
		result := []byte{command.Operation | command.TripClose<<6, command.Count}
		result = binary.LittleEndian.AppendUint32(result, command.OnTime)
		result = binary.LittleEndian.AppendUint32(result, command.OffTime)
		return append(result, command.Status), nil
	}
	if command.Group != 41 || command.Variation < 1 || command.Variation > 4 {
		return nil, fmt.Errorf("unsupported DNP3 command group %d variation %d", command.Group, command.Variation)
	}
	result := make([]byte, 0, 9)
	switch command.Variation {
	case 1:
		value, ok := command.Value.(int32)
		if !ok {
			return nil, errors.New("DNP3 analog command value must be int32")
		}
		result = binary.LittleEndian.AppendUint32(result, uint32(value))
	case 2:
		value, ok := command.Value.(int16)
		if !ok {
			return nil, errors.New("DNP3 analog command value must be int16")
		}
		result = binary.LittleEndian.AppendUint16(result, uint16(value))
	case 3:
		value, ok := command.Value.(float32)
		if !ok || math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return nil, errors.New("DNP3 analog command value must be a finite float32")
		}
		result = binary.LittleEndian.AppendUint32(result, math.Float32bits(value))
	case 4:
		value, ok := command.Value.(float64)
		if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil, errors.New("DNP3 analog command value must be a finite float64")
		}
		result = binary.LittleEndian.AppendUint64(result, math.Float64bits(value))
	}
	return append(result, command.Status), nil
}

func DecodeStrictCommandValue(group, variation byte, data []byte) (StrictCommand, int, error) {
	command := StrictCommand{Group: group, Variation: variation}
	if group == 12 && variation == 1 {
		if len(data) < 11 {
			return StrictCommand{}, 0, errors.New("truncated DNP3 CROB")
		}
		control := data[0]
		command.Operation = control & 0x0f
		command.TripClose = control >> 6
		if control&0x30 != 0 || command.Operation < CROBPulseOn || command.Operation > CROBLatchOff || command.TripClose > CROBTrip {
			return StrictCommand{}, 0, errors.New("invalid DNP3 CROB control code")
		}
		command.Count = data[1]
		if command.Count == 0 {
			return StrictCommand{}, 0, errors.New("DNP3 CROB count is zero")
		}
		command.OnTime = binary.LittleEndian.Uint32(data[2:6])
		command.OffTime = binary.LittleEndian.Uint32(data[6:10])
		command.Status = data[10]
		if command.Status > 0x7f {
			return StrictCommand{}, 0, errors.New("DNP3 command status exceeds seven bits")
		}
		return command, 11, nil
	}
	if group != 41 || variation < 1 || variation > 4 {
		return StrictCommand{}, 0, fmt.Errorf("unsupported DNP3 command group %d variation %d", group, variation)
	}
	width := map[byte]int{1: 4, 2: 2, 3: 4, 4: 8}[variation]
	if len(data) < width+1 {
		return StrictCommand{}, 0, errors.New("truncated DNP3 analog command")
	}
	switch variation {
	case 1:
		command.Value = int32(binary.LittleEndian.Uint32(data))
	case 2:
		command.Value = int16(binary.LittleEndian.Uint16(data))
	case 3:
		value := math.Float32frombits(binary.LittleEndian.Uint32(data))
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return StrictCommand{}, 0, errors.New("DNP3 analog command float32 is not finite")
		}
		command.Value = value
	case 4:
		value := math.Float64frombits(binary.LittleEndian.Uint64(data))
		if math.IsNaN(value) || math.IsInf(value, 0) {
			return StrictCommand{}, 0, errors.New("DNP3 analog command float64 is not finite")
		}
		command.Value = value
	}
	command.Status = data[width]
	if command.Status > 0x7f {
		return StrictCommand{}, 0, errors.New("DNP3 command status exceeds seven bits")
	}
	return command, width + 1, nil
}

func EncodeStrictCommandBlock(header StrictObjectHeader, commands []StrictCommand) ([]byte, error) {
	descriptor, err := DecodeStrictQualifier(header.Qualifier)
	if err != nil {
		return nil, err
	}
	if descriptor.Kind != StrictRangeCount || descriptor.PrefixWidth == 0 || descriptor.SizePrefix {
		return nil, errors.New("DNP3 command block requires count and index prefixes")
	}
	if uint64(len(commands)) != uint64(header.Count) {
		return nil, errors.New("DNP3 command block count does not match its header")
	}
	result, err := EncodeStrictObjectHeader(header)
	if err != nil {
		return nil, err
	}
	for index, command := range commands {
		if command.Group != header.Group || command.Variation != header.Variation {
			return nil, fmt.Errorf("DNP3 command %d group or variation differs from its header", index)
		}
		result, err = AppendStrictUnsigned(result, command.Index, descriptor.PrefixWidth)
		if err != nil {
			return nil, fmt.Errorf("encoding DNP3 command %d index: %w", index, err)
		}
		encoded, encodeErr := EncodeStrictCommandValue(command)
		if encodeErr != nil {
			return nil, fmt.Errorf("encoding DNP3 command %d: %w", index, encodeErr)
		}
		result = append(result, encoded...)
	}
	return result, nil
}

func DecodeStrictCommandBlock(data []byte, maximumCommands int) (StrictObjectHeader, []StrictCommand, int, error) {
	if maximumCommands < 1 {
		return StrictObjectHeader{}, nil, 0, errors.New("DNP3 maximum commands must be positive")
	}
	header, offset, err := DecodeStrictObjectHeader(data)
	if err != nil {
		return StrictObjectHeader{}, nil, 0, err
	}
	descriptor, err := DecodeStrictQualifier(header.Qualifier)
	if err != nil {
		return StrictObjectHeader{}, nil, 0, err
	}
	if descriptor.Kind != StrictRangeCount || descriptor.PrefixWidth == 0 || descriptor.SizePrefix {
		return StrictObjectHeader{}, nil, 0, errors.New("DNP3 command block requires count and index prefixes")
	}
	if header.Count > uint32(maximumCommands) {
		return StrictObjectHeader{}, nil, 0, errors.New("DNP3 command block exceeds command limit")
	}
	commands := make([]StrictCommand, 0, int(header.Count))
	for index := uint32(0); index < header.Count; index++ {
		commandIndex, decodeIndexErr := DecodeStrictUnsigned(data[offset:], descriptor.PrefixWidth)
		if decodeIndexErr != nil {
			return StrictObjectHeader{}, nil, 0, fmt.Errorf("decoding DNP3 command %d index: %w", index, decodeIndexErr)
		}
		offset += descriptor.PrefixWidth
		command, consumed, decodeErr := DecodeStrictCommandValue(header.Group, header.Variation, data[offset:])
		if decodeErr != nil {
			return StrictObjectHeader{}, nil, 0, fmt.Errorf("decoding DNP3 command %d: %w", index, decodeErr)
		}
		command.Index = commandIndex
		commands = append(commands, command)
		offset += consumed
	}
	return header, commands, offset, nil
}
