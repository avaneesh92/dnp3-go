package app

import (
	"testing"
)

func TestCROBSerialize(t *testing.T) {
	crob := NewLatchOn()

	data := crob.Serialize()

	if len(data) != 11 {
		t.Fatalf("Expected 11 bytes, got %d", len(data))
	}

	if data[0] != ControlCodeLatchOn {
		t.Errorf("Control code: got 0x%02X, want 0x%02X", data[0], ControlCodeLatchOn)
	}

	if data[1] != 1 {
		t.Errorf("Count: got %d, want 1", data[1])
	}
}

func TestCROBParse(t *testing.T) {
	original := NewPulseOn(1000)
	data := original.Serialize()

	parsed, err := ParseCROB(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.ControlCode != original.ControlCode {
		t.Errorf("Control code: got 0x%02X, want 0x%02X", parsed.ControlCode, original.ControlCode)
	}

	if parsed.Count != original.Count {
		t.Errorf("Count: got %d, want %d", parsed.Count, original.Count)
	}

	if parsed.OnTime != original.OnTime {
		t.Errorf("OnTime: got %d, want %d", parsed.OnTime, original.OnTime)
	}

	if parsed.OffTime != original.OffTime {
		t.Errorf("OffTime: got %d, want %d", parsed.OffTime, original.OffTime)
	}
}

func TestCROBTypes(t *testing.T) {
	tests := []struct {
		name string
		crob CROB
		code uint8
	}{
		{"latch on", NewLatchOn(), ControlCodeLatchOn},
		{"latch off", NewLatchOff(), ControlCodeLatchOff},
		{"pulse on", NewPulseOn(500), ControlCodePulseOn},
		{"pulse off", NewPulseOff(500), ControlCodePulseOff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.crob.ControlCode != tt.code {
				t.Errorf("Control code: got 0x%02X, want 0x%02X", tt.crob.ControlCode, tt.code)
			}
		})
	}
}

func TestCROBStatus(t *testing.T) {
	crob := NewLatchOn()
	crob.Status = ControlStatusSuccess

	status := crob.StatusString()
	if status != "Success" {
		t.Errorf("Status string: got %s, want Success", status)
	}

	crob.Status = ControlStatusTimeout
	status = crob.StatusString()
	if status != "Timeout" {
		t.Errorf("Status string: got %s, want Timeout", status)
	}
}

func TestAnalogOutputBlock(t *testing.T) {
	ao := NewAnalogOutputBlockInt32(7777)

	data := ao.SerializeInt32()

	if len(data) != 5 {
		t.Fatalf("Expected 5 bytes, got %d", len(data))
	}

	parsed, err := ParseAnalogOutputInt32(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if val, ok := parsed.Value.(int32); ok {
		if val != 7777 {
			t.Errorf("Value: got %d, want 7777", val)
		}
	} else {
		t.Error("Value should be int32")
	}
}

func TestAnalogOutputInt16(t *testing.T) {
	ao := NewAnalogOutputBlockInt16(999)

	data := ao.SerializeInt16()

	if len(data) != 3 {
		t.Fatalf("Expected 3 bytes, got %d", len(data))
	}

	parsed, err := ParseAnalogOutputInt16(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if val, ok := parsed.Value.(int16); ok {
		if val != 999 {
			t.Errorf("Value: got %d, want 999", val)
		}
	} else {
		t.Error("Value should be int16")
	}
}

func TestBuildCROBRequest(t *testing.T) {
	crob := NewLatchOn()
	data := BuildCROBRequest(5, crob)

	// Should have object header + CROB data
	// Header: 3 bytes (group, var, qualifier) + 2 bytes (start, stop) = 5 bytes
	// CROB: 11 bytes
	// Total: 16 bytes
	if len(data) != 16 {
		t.Errorf("Expected 16 bytes, got %d", len(data))
	}

	// Verify header
	if data[0] != GroupBinaryOutputCommand {
		t.Errorf("Group: got %d, want %d", data[0], GroupBinaryOutputCommand)
	}
	if data[1] != 1 {
		t.Errorf("Variation: got %d, want 1", data[1])
	}
}

func TestBuildAnalogOutputRequest(t *testing.T) {
	data := BuildAnalogOutputRequest(3, 5000)

	// Header: 5 bytes + Analog output: 5 bytes = 10 bytes
	if len(data) != 10 {
		t.Errorf("Expected 10 bytes, got %d", len(data))
	}

	// Verify header
	if data[0] != GroupAnalogOutputCommand {
		t.Errorf("Group: got %d, want %d", data[0], GroupAnalogOutputCommand)
	}
}
