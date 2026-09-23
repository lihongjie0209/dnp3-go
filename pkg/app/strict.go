package app

import (
	"encoding/binary"
	"errors"
)

type StrictFragment struct {
	FIR      bool
	FIN      bool
	CON      bool
	UNS      bool
	Sequence byte
	Function byte
	IIN      uint16
	Objects  []byte
}

func EncodeStrictFragment(fragment StrictFragment, maximum int) ([]byte, error) {
	if maximum < 2 {
		return nil, errors.New("DNP3 maximum application fragment must be at least 2 bytes")
	}
	if fragment.Sequence > AppCtrlSeqMask {
		return nil, errors.New("DNP3 application sequence exceeds 15")
	}
	headerLength := 2
	if isStrictResponse(fragment.Function) {
		headerLength = 4
	}
	if len(fragment.Objects) > maximum-headerLength {
		return nil, errors.New("DNP3 application fragment exceeds configured limit")
	}
	control := fragment.Sequence
	if fragment.FIR {
		control |= AppCtrlFIR
	}
	if fragment.FIN {
		control |= AppCtrlFIN
	}
	if fragment.CON {
		control |= AppCtrlCON
	}
	if fragment.UNS {
		control |= AppCtrlUNS
	}
	result := make([]byte, headerLength, headerLength+len(fragment.Objects))
	result[0], result[1] = control, fragment.Function
	if headerLength == 4 {
		binary.LittleEndian.PutUint16(result[2:4], fragment.IIN)
	}
	return append(result, fragment.Objects...), nil
}

func DecodeStrictFragment(data []byte, maximum int) (StrictFragment, error) {
	if maximum < 2 {
		return StrictFragment{}, errors.New("DNP3 maximum application fragment must be at least 2 bytes")
	}
	if len(data) < 2 {
		return StrictFragment{}, errors.New("truncated DNP3 application header")
	}
	if len(data) > maximum {
		return StrictFragment{}, errors.New("DNP3 application fragment exceeds configured limit")
	}
	control, function := data[0], data[1]
	fragment := StrictFragment{
		FIR: control&AppCtrlFIR != 0, FIN: control&AppCtrlFIN != 0,
		CON: control&AppCtrlCON != 0, UNS: control&AppCtrlUNS != 0,
		Sequence: control & AppCtrlSeqMask, Function: function,
	}
	offset := 2
	if isStrictResponse(function) {
		if len(data) < 4 {
			return StrictFragment{}, errors.New("truncated DNP3 application response header")
		}
		fragment.IIN = binary.LittleEndian.Uint16(data[2:4])
		offset = 4
	}
	fragment.Objects = append([]byte(nil), data[offset:]...)
	return fragment, nil
}

func isStrictResponse(function byte) bool { return function&0x80 != 0 }
