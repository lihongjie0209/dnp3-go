package app

import (
	"bytes"
	"github.com/lihongjie0209/dnp3-go/pkg/types"
	"testing"
)

func TestAPDUSerializeParse(t *testing.T) {
	tests := []struct {
		name string
		apdu *APDU
	}{
		{
			name: "simple read request",
			apdu: &APDU{
				FunctionCode: FuncRead,
				Sequence:     5,
				FIR:          true,
				FIN:          true,
				CON:          false,
				UNS:          false,
				Objects:      []byte{0x3C, 0x01, 0x06}, // Class 0
			},
		},
		{
			name: "response with IIN",
			apdu: &APDU{
				FunctionCode: FuncResponse,
				Sequence:     3,
				FIR:          true,
				FIN:          true,
				IIN:          types.IIN{IIN1: 0x10, IIN2: 0x00},
				Objects:      []byte{},
			},
		},
		{
			name: "unsolicited response",
			apdu: &APDU{
				FunctionCode: FuncUnsolicitedResponse,
				Sequence:     7,
				FIR:          true,
				FIN:          true,
				CON:          true,
				UNS:          true,
				IIN:          types.IIN{IIN1: 0x02, IIN2: 0x00},
				Objects:      []byte{0x02, 0x02, 0x00},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize
			data := tt.apdu.Serialize()

			// Parse
			parsed, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Compare
			if parsed.FunctionCode != tt.apdu.FunctionCode {
				t.Errorf("FunctionCode mismatch: got %v, want %v", parsed.FunctionCode, tt.apdu.FunctionCode)
			}
			if parsed.Sequence != tt.apdu.Sequence {
				t.Errorf("Sequence mismatch: got %d, want %d", parsed.Sequence, tt.apdu.Sequence)
			}
			if parsed.FIR != tt.apdu.FIR {
				t.Errorf("FIR mismatch: got %v, want %v", parsed.FIR, tt.apdu.FIR)
			}
			if parsed.FIN != tt.apdu.FIN {
				t.Errorf("FIN mismatch: got %v, want %v", parsed.FIN, tt.apdu.FIN)
			}
			if parsed.CON != tt.apdu.CON {
				t.Errorf("CON mismatch: got %v, want %v", parsed.CON, tt.apdu.CON)
			}
			if parsed.UNS != tt.apdu.UNS {
				t.Errorf("UNS mismatch: got %v, want %v", parsed.UNS, tt.apdu.UNS)
			}

			if parsed.IsResponse() {
				if parsed.IIN.IIN1 != tt.apdu.IIN.IIN1 {
					t.Errorf("IIN1 mismatch: got 0x%02X, want 0x%02X", parsed.IIN.IIN1, tt.apdu.IIN.IIN1)
				}
				if parsed.IIN.IIN2 != tt.apdu.IIN.IIN2 {
					t.Errorf("IIN2 mismatch: got 0x%02X, want 0x%02X", parsed.IIN.IIN2, tt.apdu.IIN.IIN2)
				}
			}

			if !bytes.Equal(parsed.Objects, tt.apdu.Objects) {
				t.Errorf("Objects mismatch: got %v, want %v", parsed.Objects, tt.apdu.Objects)
			}
		})
	}
}

func TestNewRequestAPDU(t *testing.T) {
	apdu := NewRequestAPDU(FuncRead, 10, []byte{0x3C, 0x01, 0x06})

	if apdu.FunctionCode != FuncRead {
		t.Errorf("FunctionCode: got %v, want %v", apdu.FunctionCode, FuncRead)
	}
	if apdu.Sequence != 10 {
		t.Errorf("Sequence: got %d, want %d", apdu.Sequence, 10)
	}
	if !apdu.FIR || !apdu.FIN {
		t.Error("FIR and FIN should be true for request")
	}
	if apdu.CON || apdu.UNS {
		t.Error("CON and UNS should be false for request")
	}
}

func TestNewResponseAPDU(t *testing.T) {
	iin := types.IIN{IIN1: 0x10, IIN2: 0x00}
	apdu := NewResponseAPDU(5, iin, nil)

	if apdu.FunctionCode != FuncResponse {
		t.Errorf("FunctionCode: got %v, want %v", apdu.FunctionCode, FuncResponse)
	}
	if apdu.Sequence != 5 {
		t.Errorf("Sequence: got %d, want %d", apdu.Sequence, 5)
	}
	if apdu.IIN.IIN1 != 0x10 {
		t.Errorf("IIN1: got 0x%02X, want 0x10", apdu.IIN.IIN1)
	}
}

func TestNewUnsolicitedResponseAPDU(t *testing.T) {
	iin := types.IIN{IIN1: 0x02, IIN2: 0x00}
	apdu := NewUnsolicitedResponseAPDU(3, iin, nil)

	if apdu.FunctionCode != FuncUnsolicitedResponse {
		t.Errorf("FunctionCode: got %v, want %v", apdu.FunctionCode, FuncUnsolicitedResponse)
	}
	if !apdu.UNS {
		t.Error("UNS should be true for unsolicited")
	}
	if !apdu.CON {
		t.Error("CON should be true for unsolicited")
	}
}

func TestControlByte(t *testing.T) {
	tests := []struct {
		name     string
		fir      bool
		fin      bool
		con      bool
		uns      bool
		seq      uint8
		expected uint8
	}{
		{"normal request seq 0", true, true, false, false, 0, 0xC0},
		{"normal request seq 5", true, true, false, false, 5, 0xC5},
		{"normal request seq 15", true, true, false, false, 15, 0xCF},
		{"unsolicited seq 0", true, true, true, true, 0, 0xF0},
		{"confirm required", true, true, true, false, 7, 0xE7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apdu := &APDU{
				FIR:      tt.fir,
				FIN:      tt.fin,
				CON:      tt.con,
				UNS:      tt.uns,
				Sequence: tt.seq,
			}
			apdu.buildControl()

			if apdu.Control != tt.expected {
				t.Errorf("Control byte: got 0x%02X, want 0x%02X", apdu.Control, tt.expected)
			}
		})
	}
}

func TestSequenceWrapping(t *testing.T) {
	apdu := NewRequestAPDU(FuncRead, 20, nil) // 20 > 15, should wrap

	if apdu.Sequence != 4 { // 20 & 0x0F = 4
		t.Errorf("Sequence should wrap: got %d, want 4", apdu.Sequence)
	}
}
