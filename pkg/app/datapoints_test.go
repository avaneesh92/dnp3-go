package app

import (
	"testing"
)

func TestBinaryInputSerialize(t *testing.T) {
	bi := NewBinaryInput(true)

	data := bi.Serialize()

	if len(data) != 1 {
		t.Fatalf("Expected 1 byte, got %d", len(data))
	}

	// Should have FlagOnline and FlagState set
	expected := FlagOnline | FlagState
	if data[0] != expected {
		t.Errorf("Flags: got 0x%02X, want 0x%02X", data[0], expected)
	}
}

func TestBinaryInputParse(t *testing.T) {
	data := []byte{FlagOnline | FlagState}

	bi := ParseBinaryInput(data)

	if !bi.Value {
		t.Error("Value should be true")
	}
	if bi.Flags != (FlagOnline | FlagState) {
		t.Errorf("Flags: got 0x%02X, want 0x%02X", bi.Flags, FlagOnline|FlagState)
	}
}

func TestAnalogInputInt32Serialize(t *testing.T) {
	ai := NewAnalogInputInt32(12345)

	data := ai.Serialize32Bit()

	if len(data) != 5 {
		t.Fatalf("Expected 5 bytes, got %d", len(data))
	}

	if data[0] != FlagOnline {
		t.Errorf("Flag: got 0x%02X, want 0x%02X", data[0], FlagOnline)
	}

	// Parse back
	parsed := ParseAnalogInput32Bit(data)
	if val, ok := parsed.Value.(int32); ok {
		if val != 12345 {
			t.Errorf("Value: got %d, want 12345", val)
		}
	} else {
		t.Error("Value should be int32")
	}
}

func TestAnalogInputFloatSerialize(t *testing.T) {
	ai := NewAnalogInputFloat(123.45)

	data := ai.SerializeFloat()

	if len(data) != 5 {
		t.Fatalf("Expected 5 bytes, got %d", len(data))
	}

	parsed := ParseAnalogInputFloat(data)
	if val, ok := parsed.Value.(float32); ok {
		if val < 123.44 || val > 123.46 {
			t.Errorf("Value: got %f, want ~123.45", val)
		}
	} else {
		t.Error("Value should be float32")
	}
}

func TestCounterSerialize(t *testing.T) {
	counter := NewCounter(9876)

	data := counter.Serialize32Bit()

	if len(data) != 5 {
		t.Fatalf("Expected 5 bytes, got %d", len(data))
	}

	parsed := ParseCounter32Bit(data)
	if parsed.Value != 9876 {
		t.Errorf("Value: got %d, want 9876", parsed.Value)
	}
}

func TestCounter16BitSerialize(t *testing.T) {
	counter := NewCounter(1234)

	data := counter.Serialize16Bit()

	if len(data) != 3 {
		t.Fatalf("Expected 3 bytes, got %d", len(data))
	}

	parsed := ParseCounter16Bit(data)
	if parsed.Value != 1234 {
		t.Errorf("Value: got %d, want 1234", parsed.Value)
	}
}

func TestBinaryInputEvent(t *testing.T) {
	event := NewBinaryInputEvent(true, 1234567890)

	// Test without time
	data := event.SerializeWithoutTime()
	if len(data) != 1 {
		t.Errorf("Without time: expected 1 byte, got %d", len(data))
	}

	// Test with time
	data = event.SerializeWithTime()
	if len(data) != 7 {
		t.Errorf("With time: expected 7 bytes, got %d", len(data))
	}
}

func TestBinaryOutput(t *testing.T) {
	bo := NewBinaryOutput(false)

	data := bo.Serialize()

	if len(data) != 1 {
		t.Fatalf("Expected 1 byte, got %d", len(data))
	}

	// Should have FlagOnline but not FlagState
	if data[0] != FlagOnline {
		t.Errorf("Flags: got 0x%02X, want 0x%02X", data[0], FlagOnline)
	}

	parsed := ParseBinaryOutput(data)
	if parsed.Value {
		t.Error("Value should be false")
	}
}

func TestAnalogOutputStatus(t *testing.T) {
	ao := NewAnalogOutputStatusInt32(5555)

	data := ao.Serialize32Bit()

	if len(data) != 5 {
		t.Fatalf("Expected 5 bytes, got %d", len(data))
	}

	if data[0] != FlagOnline {
		t.Errorf("Flag: got 0x%02X, want 0x%02X", data[0], FlagOnline)
	}
}
