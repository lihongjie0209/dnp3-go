package link

import (
	"bytes"
	"testing"
)

// TestCalculateCRC_KnownVectors tests CRC calculation with known DNP3 test vectors
func TestCalculateCRC_KnownVectors(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected uint16
	}{
		{
			name:     "Empty data",
			data:     []byte{},
			expected: 0xFFFF, // Inverted 0x0000
		},
		{
			name:     "Single byte 0x05",
			data:     []byte{0x05},
			expected: 0x10D9,
		},
		{
			name:     "DNP3 header start bytes",
			data:     []byte{0x05, 0x64},
			expected: 0xC0F2,
		},
		{
			name:     "CRC standard check",
			data:     []byte("123456789"),
			expected: 0xEA82,
		},
		{
			name: "Full DNP3 link header (without CRC)",
			// 0x05 0x64 (start) + 0x05 (len) + 0xC0 (ctrl) + 0x01 0x00 (dest) + 0x00 0x04 (src)
			data:     []byte{0x05, 0x64, 0x05, 0xC0, 0x01, 0x00, 0x00, 0x04},
			expected: 0x21E9,
		},
		{
			name:     "All zeros (16 bytes)",
			data:     make([]byte, 16),
			expected: 0xFFFF, // Inverted 0x0000
		},
		{
			name:     "All 0xFF (16 bytes)",
			data:     bytes.Repeat([]byte{0xFF}, 16),
			expected: 0x0053,
		},
		{
			name:     "Sequential bytes 0x00-0x0F",
			data:     []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F},
			expected: 0x10EC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateCRC(tt.data)
			if result != tt.expected {
				t.Errorf("CalculateCRC() = 0x%04X, expected 0x%04X\nData: % X", result, tt.expected, tt.data)
			}
		})
	}
}

// TestCalculateCRC_EdgeCases tests edge cases for CRC calculation
func TestCalculateCRC_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"Nil slice", nil},
		{"Empty slice", []byte{}},
		{"Single byte", []byte{0x42}},
		{"Two bytes", []byte{0x12, 0x34}},
		{"Large data (1000 bytes)", make([]byte, 1000)},
		{"Max frame data (250 bytes)", make([]byte, MaxDataSize)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Should not panic
			result := CalculateCRC(tt.data)

			// Result should be 16-bit value
			if result > 0xFFFF {
				t.Errorf("CRC result exceeds 16 bits: 0x%X", result)
			}

			// CRC should be deterministic
			result2 := CalculateCRC(tt.data)
			if result != result2 {
				t.Errorf("CRC not deterministic: first=0x%04X, second=0x%04X", result, result2)
			}
		})
	}
}

// TestVerifyCRC_ValidCRCs tests CRC verification with valid CRCs
func TestVerifyCRC_ValidCRCs(t *testing.T) {
	tests := []struct {
		name string
		data []byte // Data with CRC appended (little-endian)
	}{
		{
			name: "Empty data with CRC",
			data: []byte{0xFF, 0xFF}, // CRC of empty data
		},
		{
			name: "Single byte with CRC",
			data: []byte{0x05, 0xD9, 0x10},
		},
		{
			name: "DNP3 start bytes with CRC",
			data: []byte{0x05, 0x64, 0xF2, 0xC0},
		},
		{
			name: "Full header with CRC",
			data: []byte{0x05, 0x64, 0x05, 0xC0, 0x01, 0x00, 0x00, 0x04, 0xE9, 0x21},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !VerifyCRC(tt.data) {
				t.Errorf("VerifyCRC() = false, expected true for valid CRC\nData: % X", tt.data)
			}
		})
	}
}

// TestVerifyCRC_InvalidCRCs tests CRC verification with invalid CRCs
func TestVerifyCRC_InvalidCRCs(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Too short (0 bytes)",
			data: []byte{},
		},
		{
			name: "Too short (1 byte)",
			data: []byte{0x05},
		},
		{
			name: "Wrong CRC value",
			data: []byte{0x05, 0x64, 0x00, 0x00}, // Wrong CRC
		},
		{
			name: "Corrupted data",
			data: []byte{0x05, 0xFF, 0x65, 0x7A}, // Second byte corrupted
		},
		{
			name: "Swapped CRC bytes (wrong endianness)",
			data: []byte{0x05, 0x64, 0x7A, 0x65}, // CRC bytes swapped
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if VerifyCRC(tt.data) {
				t.Errorf("VerifyCRC() = true, expected false for invalid CRC\nData: % X", tt.data)
			}
		})
	}
}

// TestAppendCRC tests CRC appending
func TestAppendCRC(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"Empty", []byte{}},
		{"Single byte", []byte{0x05}},
		{"DNP3 start bytes", []byte{0x05, 0x64}},
		{"Full header", []byte{0x05, 0x64, 0x05, 0xC0, 0x01, 0x00, 0x00, 0x04}},
		{"16 bytes", make([]byte, 16)},
		{"Large data", make([]byte, 100)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AppendCRC(tt.data)

			// Check length is correct (original + 2 bytes for CRC)
			expectedLen := len(tt.data) + 2
			if len(result) != expectedLen {
				t.Errorf("AppendCRC() length = %d, expected %d", len(result), expectedLen)
			}

			// Original data should be preserved
			if !bytes.Equal(result[:len(tt.data)], tt.data) {
				t.Errorf("AppendCRC() corrupted original data")
			}

			// Result should verify successfully
			if !VerifyCRC(result) {
				t.Errorf("AppendCRC() result failed CRC verification\nData: % X\nResult: % X", tt.data, result)
			}

			// Original data should not be modified
			original := make([]byte, len(tt.data))
			copy(original, tt.data)
			AppendCRC(tt.data)
			if !bytes.Equal(tt.data, original) {
				t.Errorf("AppendCRC() modified original slice")
			}
		})
	}
}

// TestAppendCRC_LittleEndian verifies CRC is appended in little-endian format
func TestAppendCRC_LittleEndian(t *testing.T) {
	data := []byte{0x05}
	expected := uint16(0x10D9)

	result := AppendCRC(data)

	// Extract CRC from result (little-endian)
	crcLow := result[1]
	crcHigh := result[2]
	crc := uint16(crcLow) | (uint16(crcHigh) << 8)

	if crc != expected {
		t.Errorf("CRC byte order incorrect: got 0x%04X [%02X %02X], expected 0x%04X [%02X %02X]",
			crc, crcLow, crcHigh, expected, byte(expected), byte(expected>>8))
	}
}

// TestAddCRCs tests adding CRCs to 16-byte blocks
func TestAddCRCs(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		expectedLen    int
		expectedBlocks int
	}{
		{
			name:           "Empty data",
			data:           []byte{},
			expectedLen:    0,
			expectedBlocks: 0,
		},
		{
			name:           "Exact 16 bytes (1 block)",
			data:           make([]byte, 16),
			expectedLen:    18, // 16 + 2 (CRC)
			expectedBlocks: 1,
		},
		{
			name:           "17 bytes (2 blocks)",
			data:           make([]byte, 17),
			expectedLen:    21, // 16 + 2 + 1 + 2
			expectedBlocks: 2,
		},
		{
			name:           "32 bytes (2 full blocks)",
			data:           make([]byte, 32),
			expectedLen:    36, // 16 + 2 + 16 + 2
			expectedBlocks: 2,
		},
		{
			name:           "50 bytes (4 blocks)",
			data:           make([]byte, 50),
			expectedLen:    58, // 16 + 2 + 16 + 2 + 16 + 2 + 2 + 2
			expectedBlocks: 4,
		},
		{
			name:           "1 byte (1 block)",
			data:           []byte{0x42},
			expectedLen:    3, // 1 + 2
			expectedBlocks: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddCRCs(tt.data)

			if len(result) != tt.expectedLen {
				t.Errorf("AddCRCs() length = %d, expected %d", len(result), tt.expectedLen)
			}

			// Verify we can remove CRCs successfully
			if len(tt.data) > 0 {
				removed, err := RemoveCRCs(result)
				if err != nil {
					t.Errorf("RemoveCRCs() error = %v", err)
				}
				if !bytes.Equal(removed, tt.data) {
					t.Errorf("RemoveCRCs() data mismatch\nOriginal: % X\nRemoved:  % X", tt.data, removed)
				}
			}
		})
	}
}

// TestAddCRCs_BlockBoundaries tests CRC insertion at block boundaries
func TestAddCRCs_BlockBoundaries(t *testing.T) {
	// Create data with known pattern
	data := make([]byte, 48) // Exactly 3 blocks of 16 bytes
	for i := range data {
		data[i] = byte(i)
	}

	result := AddCRCs(data)

	// Expected: 16 bytes + 2 CRC + 16 bytes + 2 CRC + 16 bytes + 2 CRC = 54 bytes
	expectedLen := 54
	if len(result) != expectedLen {
		t.Errorf("AddCRCs() length = %d, expected %d", len(result), expectedLen)
	}

	// Verify block 1
	block1 := result[0:16]
	crc1 := uint16(result[16]) | (uint16(result[17]) << 8)
	expectedCRC1 := CalculateCRC(block1)
	if crc1 != expectedCRC1 {
		t.Errorf("Block 1 CRC = 0x%04X, expected 0x%04X", crc1, expectedCRC1)
	}

	// Verify block 2
	block2 := result[18:34]
	crc2 := uint16(result[34]) | (uint16(result[35]) << 8)
	expectedCRC2 := CalculateCRC(block2)
	if crc2 != expectedCRC2 {
		t.Errorf("Block 2 CRC = 0x%04X, expected 0x%04X", crc2, expectedCRC2)
	}

	// Verify block 3
	block3 := result[36:52]
	crc3 := uint16(result[52]) | (uint16(result[53]) << 8)
	expectedCRC3 := CalculateCRC(block3)
	if crc3 != expectedCRC3 {
		t.Errorf("Block 3 CRC = 0x%04X, expected 0x%04X", crc3, expectedCRC3)
	}
}

// TestRemoveCRCs tests removing and verifying CRCs
func TestRemoveCRCs(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "Empty data",
			data:    []byte{},
			wantErr: false,
		},
		{
			name:    "Valid 16-byte block",
			data:    AddCRCs(make([]byte, 16)),
			wantErr: false,
		},
		{
			name:    "Valid 32-byte blocks",
			data:    AddCRCs(make([]byte, 32)),
			wantErr: false,
		},
		{
			name:    "Valid partial block",
			data:    AddCRCs([]byte{0x01, 0x02, 0x03}),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := RemoveCRCs(tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("RemoveCRCs() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && len(tt.data) > 0 {
				// Verify length is correct
				// For n blocks: original has (16*n + 2*n) bytes, result should have 16*n bytes
				if result == nil {
					t.Errorf("RemoveCRCs() returned nil for non-empty input")
				}
			}
		})
	}
}

// TestRemoveCRCs_InvalidData tests error cases for RemoveCRCs
func TestRemoveCRCs_InvalidData(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "Too short for CRC",
			data: []byte{0x05},
		},
		{
			name: "Invalid CRC",
			data: []byte{0x05, 0x64, 0x00, 0x00}, // Wrong CRC
		},
		{
			name: "Corrupted first block",
			data: append([]byte{0xFF, 0xFF}, []byte{0x00, 0x00}...), // Wrong data with wrong CRC
		},
		{
			name: "Truncated data (missing CRC bytes)",
			data: make([]byte, 17), // 16 bytes + 1 (should be 16 + 2)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := RemoveCRCs(tt.data)
			if err == nil {
				t.Errorf("RemoveCRCs() expected error for invalid data, got nil\nData: % X", tt.data)
			}
		})
	}
}

// TestAddRemoveCRCs_RoundTrip tests round-trip adding and removing CRCs
func TestAddRemoveCRCs_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"1 byte", []byte{0x42}},
		{"15 bytes", make([]byte, 15)},
		{"16 bytes", make([]byte, 16)},
		{"17 bytes", make([]byte, 17)},
		{"31 bytes", make([]byte, 31)},
		{"32 bytes", make([]byte, 32)},
		{"50 bytes", make([]byte, 50)},
		{"100 bytes", make([]byte, 100)},
		{"Max frame data (250 bytes)", make([]byte, MaxDataSize)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill with pattern
			for i := range tt.data {
				tt.data[i] = byte(i & 0xFF)
			}

			// Add CRCs
			withCRCs := AddCRCs(tt.data)

			// Remove CRCs
			result, err := RemoveCRCs(withCRCs)
			if err != nil {
				t.Errorf("RemoveCRCs() error = %v", err)
			}

			// Verify data is unchanged
			if !bytes.Equal(result, tt.data) {
				t.Errorf("Round-trip failed\nOriginal: % X\nResult:   % X", tt.data, result)
			}
		})
	}
}

// TestCRC_Deterministic verifies CRC calculation is deterministic
func TestCRC_Deterministic(t *testing.T) {
	data := []byte{0x05, 0x64, 0x05, 0xC0, 0x01, 0x00, 0x00, 0x04}

	// Calculate CRC multiple times
	results := make([]uint16, 1000)
	for i := range results {
		results[i] = CalculateCRC(data)
	}

	// All results should be identical
	first := results[0]
	for i, crc := range results {
		if crc != first {
			t.Errorf("CRC not deterministic at iteration %d: got 0x%04X, expected 0x%04X", i, crc, first)
		}
	}
}

// BenchmarkCalculateCRC benchmarks CRC calculation performance
func BenchmarkCalculateCRC(b *testing.B) {
	data := make([]byte, 250) // Max frame data size
	for i := range data {
		data[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CalculateCRC(data)
	}
}

// BenchmarkAddCRCs benchmarks adding CRCs to blocks
func BenchmarkAddCRCs(b *testing.B) {
	data := make([]byte, 250) // Max frame data size
	for i := range data {
		data[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = AddCRCs(data)
	}
}

// BenchmarkRemoveCRCs benchmarks removing CRCs from blocks
func BenchmarkRemoveCRCs(b *testing.B) {
	data := make([]byte, 250)
	for i := range data {
		data[i] = byte(i)
	}
	withCRCs := AddCRCs(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = RemoveCRCs(withCRCs)
	}
}
