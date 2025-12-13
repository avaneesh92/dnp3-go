package link

import (
	"bytes"
	"testing"
)

// TestNewFrame tests frame creation
func TestNewFrame(t *testing.T) {
	tests := []struct {
		name      string
		dir       Direction
		isPrimary IsPrimary
		fc        FunctionCode
		dst       uint16
		src       uint16
		data      []byte
	}{
		{
			name:      "Master to outstation primary",
			dir:       DirectionMasterToOutstation,
			isPrimary: PrimaryFrame,
			fc:        FuncUserDataUnconfirmed,
			dst:       10,
			src:       1,
			data:      []byte{0x01, 0x02, 0x03},
		},
		{
			name:      "Outstation to master secondary",
			dir:       DirectionOutstationToMaster,
			isPrimary: SecondaryFrame,
			fc:        FuncAck,
			dst:       1,
			src:       10,
			data:      nil,
		},
		{
			name:      "Empty data",
			dir:       DirectionMasterToOutstation,
			isPrimary: PrimaryFrame,
			fc:        FuncResetLink,
			dst:       100,
			src:       1,
			data:      []byte{},
		},
		{
			name:      "Max data size",
			dir:       DirectionMasterToOutstation,
			isPrimary: PrimaryFrame,
			fc:        FuncUserDataConfirmed,
			dst:       65535,
			src:       65535,
			data:      make([]byte, MaxDataSize),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := NewFrame(tt.dir, tt.isPrimary, tt.fc, tt.dst, tt.src, tt.data)

			if frame.Dir != tt.dir {
				t.Errorf("Dir = %v, want %v", frame.Dir, tt.dir)
			}
			if frame.IsPrimary != tt.isPrimary {
				t.Errorf("IsPrimary = %v, want %v", frame.IsPrimary, tt.isPrimary)
			}
			if frame.FunctionCode != tt.fc {
				t.Errorf("FunctionCode = %v, want %v", frame.FunctionCode, tt.fc)
			}
			if frame.Destination != tt.dst {
				t.Errorf("Destination = %d, want %d", frame.Destination, tt.dst)
			}
			if frame.Source != tt.src {
				t.Errorf("Source = %d, want %d", frame.Source, tt.src)
			}
			if !bytes.Equal(frame.UserData, tt.data) {
				t.Errorf("UserData = %v, want %v", frame.UserData, tt.data)
			}
		})
	}
}

// TestFrame_ControlByte tests control byte encoding
func TestFrame_ControlByte(t *testing.T) {
	tests := []struct {
		name            string
		dir             Direction
		isPrimary       IsPrimary
		fc              FunctionCode
		expectedControl uint8
	}{
		{
			name:            "Master to outstation, primary, reset",
			dir:             DirectionMasterToOutstation,
			isPrimary:       PrimaryFrame,
			fc:              FuncResetLink,
			expectedControl: 0xC0, // DIR(1) | PRM(1) | FC(0000) = 11000000
		},
		{
			name:            "Master to outstation, primary, user data confirmed",
			dir:             DirectionMasterToOutstation,
			isPrimary:       PrimaryFrame,
			fc:              FuncUserDataConfirmed,
			expectedControl: 0xC3, // DIR(1) | PRM(1) | FC(0011) = 11000011
		},
		{
			name:            "Outstation to master, secondary, ack",
			dir:             DirectionOutstationToMaster,
			isPrimary:       SecondaryFrame,
			fc:              FuncAck,
			expectedControl: 0x00, // DIR(0) | PRM(0) | FC(0000) = 00000000
		},
		{
			name:            "Outstation to master, secondary, nack",
			dir:             DirectionOutstationToMaster,
			isPrimary:       SecondaryFrame,
			fc:              FuncNack,
			expectedControl: 0x01, // DIR(0) | PRM(0) | FC(0001) = 00000001
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := NewFrame(tt.dir, tt.isPrimary, tt.fc, 1, 2, nil)

			if frame.Control != tt.expectedControl {
				t.Errorf("Control = 0x%02X, want 0x%02X", frame.Control, tt.expectedControl)
			}
		})
	}
}

// TestFrame_SetFCB tests FCB and FCV flag setting
func TestFrame_SetFCB(t *testing.T) {
	frame := NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataConfirmed, 10, 1, nil)

	// Initially FCB and FCV should be false
	if frame.FCB {
		t.Errorf("Initial FCB = true, want false")
	}
	if frame.FCV {
		t.Errorf("Initial FCV = true, want false")
	}

	// Set FCB to true
	frame.SetFCB(true)
	if !frame.FCB {
		t.Errorf("After SetFCB(true), FCB = false, want true")
	}
	if !frame.FCV {
		t.Errorf("After SetFCB(true), FCV = false, want true")
	}
	if (frame.Control & CtrlFCB) == 0 {
		t.Errorf("FCB bit not set in control byte: 0x%02X", frame.Control)
	}
	if (frame.Control & CtrlFCV) == 0 {
		t.Errorf("FCV bit not set in control byte: 0x%02X", frame.Control)
	}

	// Set FCB to false (but FCV should remain true)
	frame.SetFCB(false)
	if frame.FCB {
		t.Errorf("After SetFCB(false), FCB = true, want false")
	}
	if !frame.FCV {
		t.Errorf("After SetFCB(false), FCV = false, want true")
	}
	if (frame.Control & CtrlFCB) != 0 {
		t.Errorf("FCB bit set in control byte: 0x%02X", frame.Control)
	}
	if (frame.Control & CtrlFCV) == 0 {
		t.Errorf("FCV bit not set in control byte: 0x%02X", frame.Control)
	}
}

// TestFrame_Serialize tests frame serialization
func TestFrame_Serialize(t *testing.T) {
	tests := []struct {
		name    string
		frame   *Frame
		wantErr bool
	}{
		{
			name:    "Empty data frame",
			frame:   NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncResetLink, 10, 1, nil),
			wantErr: false,
		},
		{
			name:    "Small data frame",
			frame:   NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataUnconfirmed, 10, 1, []byte{0x01, 0x02, 0x03}),
			wantErr: false,
		},
		{
			name:    "16-byte data (exact block)",
			frame:   NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataConfirmed, 10, 1, make([]byte, 16)),
			wantErr: false,
		},
		{
			name:    "17-byte data (2 blocks)",
			frame:   NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataConfirmed, 10, 1, make([]byte, 17)),
			wantErr: false,
		},
		{
			name:    "Max data size",
			frame:   NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataConfirmed, 10, 1, make([]byte, MaxDataSize)),
			wantErr: false,
		},
		{
			name:    "Too much data",
			frame:   NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataConfirmed, 10, 1, make([]byte, MaxDataSize+1)),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.frame.Serialize()

			if (err != nil) != tt.wantErr {
				t.Errorf("Serialize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				// Verify start bytes
				if data[0] != StartByte1 {
					t.Errorf("Start byte 1 = 0x%02X, want 0x%02X", data[0], StartByte1)
				}
				if data[1] != StartByte2 {
					t.Errorf("Start byte 2 = 0x%02X, want 0x%02X", data[1], StartByte2)
				}

				// Verify length field
				expectedLen := len(tt.frame.UserData) + 5
				if data[2] != byte(expectedLen) {
					t.Errorf("Length field = %d, want %d", data[2], expectedLen)
				}

				// Verify header CRC
				if !VerifyCRC(data[0:10]) {
					t.Errorf("Header CRC verification failed")
				}
			}
		})
	}
}

// TestFrame_SerializeParse_RoundTrip tests serialization and parsing round-trip
func TestFrame_SerializeParse_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		frame *Frame
	}{
		{
			name:  "Empty data",
			frame: NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncResetLink, 10, 1, nil),
		},
		{
			name:  "Small data",
			frame: NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataUnconfirmed, 100, 5, []byte{0x01, 0x02, 0x03}),
		},
		{
			name:  "16 bytes",
			frame: NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataConfirmed, 255, 10, make([]byte, 16)),
		},
		{
			name:  "50 bytes",
			frame: NewFrame(DirectionOutstationToMaster, SecondaryFrame, FuncAck, 1, 100, make([]byte, 50)),
		},
		{
			name:  "Max size",
			frame: NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataConfirmed, 65535, 65535, make([]byte, MaxDataSize)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Fill data with pattern
			for i := range tt.frame.UserData {
				tt.frame.UserData[i] = byte(i & 0xFF)
			}

			// Serialize
			data, err := tt.frame.Serialize()
			if err != nil {
				t.Fatalf("Serialize() error = %v", err)
			}

			// Parse
			parsed, consumed, err := Parse(data)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if consumed != len(data) {
				t.Errorf("Parse() consumed %d bytes, want %d", consumed, len(data))
			}

			// Compare fields
			if parsed.Dir != tt.frame.Dir {
				t.Errorf("Dir = %v, want %v", parsed.Dir, tt.frame.Dir)
			}
			if parsed.IsPrimary != tt.frame.IsPrimary {
				t.Errorf("IsPrimary = %v, want %v", parsed.IsPrimary, tt.frame.IsPrimary)
			}
			if parsed.FunctionCode != tt.frame.FunctionCode {
				t.Errorf("FunctionCode = %v, want %v", parsed.FunctionCode, tt.frame.FunctionCode)
			}
			if parsed.Destination != tt.frame.Destination {
				t.Errorf("Destination = %d, want %d", parsed.Destination, tt.frame.Destination)
			}
			if parsed.Source != tt.frame.Source {
				t.Errorf("Source = %d, want %d", parsed.Source, tt.frame.Source)
			}
			if !bytes.Equal(parsed.UserData, tt.frame.UserData) {
				t.Errorf("UserData mismatch\nGot:  % X\nWant: % X", parsed.UserData, tt.frame.UserData)
			}
		})
	}
}

// TestParse_InvalidFrames tests parsing of invalid frames
func TestParse_InvalidFrames(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr error
	}{
		{
			name:    "Too short (< 10 bytes)",
			data:    []byte{0x05, 0x64, 0x05},
			wantErr: ErrFrameTooShort,
		},
		{
			name:    "Invalid start byte 1",
			data:    []byte{0xFF, 0x64, 0x05, 0xC0, 0x01, 0x00, 0x00, 0x04, 0x00, 0x00},
			wantErr: ErrInvalidStartBytes,
		},
		{
			name:    "Invalid start byte 2",
			data:    []byte{0x05, 0xFF, 0x05, 0xC0, 0x01, 0x00, 0x00, 0x04, 0x00, 0x00},
			wantErr: ErrInvalidStartBytes,
		},
		{
			name:    "Invalid length (< 5)",
			data:    []byte{0x05, 0x64, 0x04, 0xC0, 0x01, 0x00, 0x00, 0x04, 0x00, 0x00},
			wantErr: ErrInvalidLength,
		},
		{
			name:    "Invalid header CRC",
			data:    []byte{0x05, 0x64, 0x05, 0xC0, 0x01, 0x00, 0x00, 0x04, 0x00, 0x00},
			wantErr: ErrInvalidCRC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Parse(tt.data)

			if err == nil {
				t.Errorf("Parse() error = nil, want %v", tt.wantErr)
				return
			}

			if err != tt.wantErr {
				t.Errorf("Parse() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

// TestParse_ValidFrame tests parsing of a known valid frame
func TestParse_ValidFrame(t *testing.T) {
	// Create a valid frame with known values
	frame := NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataUnconfirmed, 10, 1, []byte{0xC0, 0x01})

	// Serialize it
	data, err := frame.Serialize()
	if err != nil {
		t.Fatalf("Serialize() error = %v", err)
	}

	// Parse it
	parsed, consumed, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	// Verify consumed bytes
	if consumed != len(data) {
		t.Errorf("consumed = %d, want %d", consumed, len(data))
	}

	// Verify fields
	if parsed.Destination != 10 {
		t.Errorf("Destination = %d, want 10", parsed.Destination)
	}
	if parsed.Source != 1 {
		t.Errorf("Source = %d, want 1", parsed.Source)
	}
	if parsed.FunctionCode != FuncUserDataUnconfirmed {
		t.Errorf("FunctionCode = %v, want %v", parsed.FunctionCode, FuncUserDataUnconfirmed)
	}
	if !bytes.Equal(parsed.UserData, []byte{0xC0, 0x01}) {
		t.Errorf("UserData = % X, want C0 01", parsed.UserData)
	}
}

// TestFrame_Clone tests frame cloning
func TestFrame_Clone(t *testing.T) {
	original := NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataConfirmed, 10, 1, []byte{0x01, 0x02, 0x03})
	original.SetFCB(true)

	clone := original.Clone()

	// Verify all fields match
	if clone.Dir != original.Dir {
		t.Errorf("Dir mismatch")
	}
	if clone.IsPrimary != original.IsPrimary {
		t.Errorf("IsPrimary mismatch")
	}
	if clone.FunctionCode != original.FunctionCode {
		t.Errorf("FunctionCode mismatch")
	}
	if clone.Destination != original.Destination {
		t.Errorf("Destination mismatch")
	}
	if clone.Source != original.Source {
		t.Errorf("Source mismatch")
	}
	if clone.FCB != original.FCB {
		t.Errorf("FCB mismatch")
	}
	if clone.FCV != original.FCV {
		t.Errorf("FCV mismatch")
	}
	if !bytes.Equal(clone.UserData, original.UserData) {
		t.Errorf("UserData mismatch")
	}

	// Verify deep copy (modifying clone doesn't affect original)
	clone.UserData[0] = 0xFF
	if original.UserData[0] == 0xFF {
		t.Errorf("Clone modified original UserData")
	}
}

// TestFrame_String tests string representation
func TestFrame_String(t *testing.T) {
	frame := NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataUnconfirmed, 10, 1, []byte{0x01, 0x02})

	str := frame.String()
	if str == "" {
		t.Errorf("String() returned empty string")
	}

	// String should contain key information
	if !bytes.Contains([]byte(str), []byte("Dst=10")) {
		t.Errorf("String() missing destination: %s", str)
	}
	if !bytes.Contains([]byte(str), []byte("Src=1")) {
		t.Errorf("String() missing source: %s", str)
	}
	if !bytes.Contains([]byte(str), []byte("DataLen=2")) {
		t.Errorf("String() missing data length: %s", str)
	}
}

// TestFrame_ParseMultipleFrames tests parsing multiple frames from buffer
func TestFrame_ParseMultipleFrames(t *testing.T) {
	frame1 := NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncResetLink, 10, 1, nil)
	frame2 := NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataUnconfirmed, 10, 1, []byte{0x01, 0x02})
	frame3 := NewFrame(DirectionOutstationToMaster, SecondaryFrame, FuncAck, 1, 10, nil)

	data1, _ := frame1.Serialize()
	data2, _ := frame2.Serialize()
	data3, _ := frame3.Serialize()

	// Combine all frames into single buffer
	buffer := append(append(data1, data2...), data3...)

	// Parse frame 1
	parsed1, consumed1, err := Parse(buffer)
	if err != nil {
		t.Fatalf("Parse frame 1 error = %v", err)
	}
	if parsed1.FunctionCode != FuncResetLink {
		t.Errorf("Frame 1 FunctionCode = %v, want %v", parsed1.FunctionCode, FuncResetLink)
	}

	// Parse frame 2
	parsed2, consumed2, err := Parse(buffer[consumed1:])
	if err != nil {
		t.Fatalf("Parse frame 2 error = %v", err)
	}
	if parsed2.FunctionCode != FuncUserDataUnconfirmed {
		t.Errorf("Frame 2 FunctionCode = %v, want %v", parsed2.FunctionCode, FuncUserDataUnconfirmed)
	}

	// Parse frame 3
	parsed3, consumed3, err := Parse(buffer[consumed1+consumed2:])
	if err != nil {
		t.Fatalf("Parse frame 3 error = %v", err)
	}
	if parsed3.FunctionCode != FuncAck {
		t.Errorf("Frame 3 FunctionCode = %v, want %v", parsed3.FunctionCode, FuncAck)
	}

	// Verify total consumed equals buffer length
	totalConsumed := consumed1 + consumed2 + consumed3
	if totalConsumed != len(buffer) {
		t.Errorf("Total consumed = %d, want %d", totalConsumed, len(buffer))
	}
}

// BenchmarkFrame_Serialize benchmarks frame serialization
func BenchmarkFrame_Serialize(b *testing.B) {
	frame := NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataConfirmed, 10, 1, make([]byte, 100))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = frame.Serialize()
	}
}

// BenchmarkFrame_Parse benchmarks frame parsing
func BenchmarkFrame_Parse(b *testing.B) {
	frame := NewFrame(DirectionMasterToOutstation, PrimaryFrame, FuncUserDataConfirmed, 10, 1, make([]byte, 100))
	data, _ := frame.Serialize()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = Parse(data)
	}
}
