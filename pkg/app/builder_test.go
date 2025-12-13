package app

import (
	"testing"
)

func TestObjectBuilder(t *testing.T) {
	builder := NewObjectBuilder()

	// Add a class 0 read header
	err := builder.AddHeader(GroupClass0Data, 1, QualifierNoRange, NoRange{})
	if err != nil {
		t.Fatalf("AddHeader failed: %v", err)
	}

	data := builder.Build()

	// Should be 3 bytes: group, variation, qualifier
	if len(data) != 3 {
		t.Errorf("Expected 3 bytes, got %d", len(data))
	}

	if data[0] != GroupClass0Data {
		t.Errorf("Group: got %d, want %d", data[0], GroupClass0Data)
	}
	if data[1] != 1 {
		t.Errorf("Variation: got %d, want 1", data[1])
	}
	if data[2] != uint8(QualifierNoRange) {
		t.Errorf("Qualifier: got 0x%02X, want 0x%02X", data[2], QualifierNoRange)
	}
}

func TestObjectBuilderStartStop8(t *testing.T) {
	builder := NewObjectBuilder()

	err := builder.AddHeader(GroupBinaryInput, 2, Qualifier8BitStartStop,
		StartStopRange{Start: 0, Stop: 15})
	if err != nil {
		t.Fatalf("AddHeader failed: %v", err)
	}

	data := builder.Build()

	// 3 bytes header + 2 bytes range
	if len(data) != 5 {
		t.Errorf("Expected 5 bytes, got %d", len(data))
	}

	if data[3] != 0 {
		t.Errorf("Start: got %d, want 0", data[3])
	}
	if data[4] != 15 {
		t.Errorf("Stop: got %d, want 15", data[4])
	}
}

func TestObjectBuilderStartStop16(t *testing.T) {
	builder := NewObjectBuilder()

	err := builder.AddHeader(GroupAnalogInput, 1, Qualifier16BitStartStop,
		StartStopRange{Start: 100, Stop: 200})
	if err != nil {
		t.Fatalf("AddHeader failed: %v", err)
	}

	data := builder.Build()

	// 3 bytes header + 4 bytes range (2 uint16s)
	if len(data) != 7 {
		t.Errorf("Expected 7 bytes, got %d", len(data))
	}
}

func TestBuildIntegrityPoll(t *testing.T) {
	data := BuildIntegrityPoll()

	// Should contain Group 60, Var 1, Qualifier 06 (no range)
	if len(data) < 3 {
		t.Fatalf("Data too short: %d bytes", len(data))
	}

	if data[0] != GroupClass0Data {
		t.Errorf("Group: got %d, want %d", data[0], GroupClass0Data)
	}
	if data[1] != 1 {
		t.Errorf("Variation: got %d, want 1", data[1])
	}
	if data[2] != uint8(QualifierNoRange) {
		t.Errorf("Qualifier: got 0x%02X, want 0x%02X", data[2], QualifierNoRange)
	}
}

func TestBuildEventPoll(t *testing.T) {
	data := BuildEventPoll()

	// Should contain 3 object headers for Class 1, 2, 3
	// Each header is 3 bytes (group, var, qualifier)
	if len(data) != 9 {
		t.Errorf("Expected 9 bytes (3 headers), got %d", len(data))
	}

	// Check first header (Class 1)
	if data[0] != GroupClass0Data || data[1] != 2 {
		t.Errorf("First header should be Group 60, Var 2")
	}

	// Check second header (Class 2)
	if data[3] != GroupClass0Data || data[4] != 3 {
		t.Errorf("Second header should be Group 60, Var 3")
	}

	// Check third header (Class 3)
	if data[6] != GroupClass0Data || data[7] != 4 {
		t.Errorf("Third header should be Group 60, Var 4")
	}
}

func TestBuildEnableUnsolicited(t *testing.T) {
	data := BuildEnableUnsolicited(Class1, Class2, Class3)

	// Should contain 3 object headers
	if len(data) != 9 {
		t.Errorf("Expected 9 bytes, got %d", len(data))
	}

	// All should be Group 60 with different variations
	if data[0] != GroupClass0Data {
		t.Error("Should be Group 60")
	}
}

func TestBuildRangeRead(t *testing.T) {
	// Test 8-bit range
	data := BuildRangeRead(GroupBinaryInput, 2, 0, 15)
	if len(data) < 5 {
		t.Errorf("Expected at least 5 bytes, got %d", len(data))
	}

	// Verify it uses 8-bit qualifier
	if data[2] != uint8(Qualifier8BitStartStop) {
		t.Errorf("Expected 8-bit qualifier, got 0x%02X", data[2])
	}

	// Test 16-bit range
	data = BuildRangeRead(GroupAnalogInput, 1, 0, 300)
	if data[2] != uint8(Qualifier16BitStartStop) {
		t.Errorf("Expected 16-bit qualifier, got 0x%02X", data[2])
	}
}

func TestBuilderReset(t *testing.T) {
	builder := NewObjectBuilder()

	builder.AddHeader(GroupClass0Data, 1, QualifierNoRange, NoRange{})
	data1 := builder.Build()

	// Verify first data
	if len(data1) != 3 {
		t.Fatalf("First build: expected 3 bytes, got %d", len(data1))
	}
	if data1[1] != 1 {
		t.Fatalf("First build should have variation 1, got %d", data1[1])
	}

	builder.Reset()

	builder.AddHeader(GroupClass0Data, 2, QualifierNoRange, NoRange{})
	data2 := builder.Build()

	// Verify second data
	if len(data2) != 3 {
		t.Errorf("After reset, expected 3 bytes, got %d", len(data2))
	}

	if data2[1] != 2 {
		t.Errorf("After reset, variation should be 2, got %d", data2[1])
	}

	// Ensure data is different
	if data1[1] == data2[1] {
		t.Error("Reset did not clear previous data - variations should differ")
	}
}
