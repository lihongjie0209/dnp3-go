package link

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestStrictFramePublishedVectorAndOwnership(t *testing.T) {
	want := []byte{0x05, 0x64, 0x05, 0xc0, 0x0a, 0x00, 0x01, 0x00, 0xb1, 0xac}
	encoded, err := EncodeStrictFrame(StrictFrame{Control: 0xc0, Destination: 10, Source: 1})
	if err != nil || !bytes.Equal(encoded, want) {
		t.Fatalf("encoded=%x err=%v", encoded, err)
	}
	decoded, err := DecodeStrictFrame(want)
	if err != nil || decoded.Control != 0xc0 || decoded.Destination != 10 || decoded.Source != 1 {
		t.Fatalf("decoded=%+v err=%v", decoded, err)
	}
	payload := bytes.Repeat([]byte{0x5a}, 250)
	wire, err := EncodeStrictFrame(StrictFrame{Control: 0xd3, Destination: 65519, UserData: payload})
	if err != nil {
		t.Fatal(err)
	}
	payload[0] = 0
	decoded, err = DecodeStrictFrame(wire)
	if err != nil || decoded.UserData[0] != 0x5a {
		t.Fatalf("decoded payload aliases input: %+v err=%v", decoded, err)
	}
	wire[len(wire)-3] = 0
	if decoded.UserData[len(decoded.UserData)-1] != 0x5a {
		t.Fatal("decoded payload aliases wire")
	}
}

func TestStrictFrameRejectsControlBoundsAndTrailingData(t *testing.T) {
	for _, control := range []byte{0xc1, 0xd0, 0xc2, 0xe4, 0xc5, 0x20, 0x02, 0x0a} {
		if err := ValidateStrictControl(control); err == nil {
			t.Fatalf("accepted control %02x", control)
		}
	}
	if _, err := EncodeStrictFrame(StrictFrame{Control: 0xc4, UserData: make([]byte, 251)}); !errors.Is(err, ErrFrameTooLong) {
		t.Fatalf("oversize error=%v", err)
	}
	valid, _ := EncodeStrictFrame(StrictFrame{Control: 0xc4, UserData: bytes.Repeat([]byte{1}, 17)})
	for _, malformed := range [][]byte{
		nil,
		valid[:9],
		valid[:len(valid)-1],
		append(append([]byte(nil), valid...), 0),
	} {
		if _, err := DecodeStrictFrame(malformed); err == nil {
			t.Fatalf("accepted malformed frame %x", malformed)
		}
	}
	badControl := append([]byte(nil), valid...)
	badControl[3] = 0xc1
	crc := CalculateCRC(badControl[:8])
	badControl[8], badControl[9] = byte(crc), byte(crc>>8)
	if _, err := DecodeStrictFrame(badControl); err == nil {
		t.Fatal("accepted invalid control with valid CRC")
	}
}

func TestReadStrictFrameConsumesOneFragmentedFrame(t *testing.T) {
	first, _ := EncodeStrictFrame(StrictFrame{Control: 0xc4, Destination: 10, Source: 1, UserData: []byte{1, 2}})
	second, _ := EncodeStrictFrame(StrictFrame{Control: 0xc0, Destination: 10, Source: 1})
	reader := &strictOneByteReader{data: append(append([]byte(nil), first...), second...)}
	frame, err := ReadStrictFrame(reader)
	if err != nil || !bytes.Equal(frame.UserData, []byte{1, 2}) {
		t.Fatalf("frame=%+v err=%v", frame, err)
	}
	frame, err = ReadStrictFrame(reader)
	if err != nil || frame.Control != 0xc0 {
		t.Fatalf("second=%+v err=%v", frame, err)
	}
	if _, err = ReadStrictFrame(reader); !errors.Is(err, io.EOF) {
		t.Fatalf("end error=%v", err)
	}
}

type strictOneByteReader struct{ data []byte }

func (r *strictOneByteReader) Read(buffer []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	buffer[0] = r.data[0]
	r.data = r.data[1:]
	return 1, nil
}
