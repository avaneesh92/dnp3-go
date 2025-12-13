package transport

import (
	"bytes"
	"testing"
	"time"
)

func TestOutstationTransport_SendSingleFragment(t *testing.T) {
	config := DefaultTransportConfig()
	outstation := NewOutstationTransport(config)

	// Small APDU that fits in one fragment
	apdu := []byte{0x01, 0x02, 0x03, 0x04, 0x05}

	segments := outstation.Send(apdu)

	if len(segments) != 1 {
		t.Fatalf("Expected 1 segment, got %d", len(segments))
	}

	// Verify header: FIR=1, FIN=1, SEQ=0
	header := segments[0][0]
	fir, fin, seq := ParseHeader(header)

	if !fir || !fin {
		t.Errorf("Expected FIR=1 FIN=1, got FIR=%v FIN=%v", fir, fin)
	}

	if seq != 0 {
		t.Errorf("Expected SEQ=0, got %d", seq)
	}

	// Verify data
	if !bytes.Equal(segments[0][1:], apdu) {
		t.Errorf("Data mismatch")
	}

	// Verify statistics
	stats := outstation.GetStats()
	if stats.GetTxFragments() != 1 {
		t.Errorf("Expected 1 TX fragment, got %d", stats.GetTxFragments())
	}
	if stats.GetTxMessages() != 1 {
		t.Errorf("Expected 1 TX message, got %d", stats.GetTxMessages())
	}
}

func TestOutstationTransport_SendMultipleFragments(t *testing.T) {
	config := DefaultTransportConfig()
	outstation := NewOutstationTransport(config)

	// Create APDU that requires 3 fragments (600 bytes)
	apdu := make([]byte, 600)
	for i := range apdu {
		apdu[i] = byte(i % 256)
	}

	segments := outstation.Send(apdu)

	expectedFragments := 3 // 248 + 248 + 104 = 600
	if len(segments) != expectedFragments {
		t.Fatalf("Expected %d segments, got %d", expectedFragments, len(segments))
	}

	// Verify first fragment: FIR=1, FIN=0, SEQ=0
	fir, fin, seq := ParseHeader(segments[0][0])
	if !fir || fin {
		t.Errorf("Fragment 0: Expected FIR=1 FIN=0, got FIR=%v FIN=%v", fir, fin)
	}
	if seq != 0 {
		t.Errorf("Fragment 0: Expected SEQ=0, got %d", seq)
	}

	// Verify middle fragment: FIR=0, FIN=0, SEQ=1
	fir, fin, seq = ParseHeader(segments[1][0])
	if fir || fin {
		t.Errorf("Fragment 1: Expected FIR=0 FIN=0, got FIR=%v FIN=%v", fir, fin)
	}
	if seq != 1 {
		t.Errorf("Fragment 1: Expected SEQ=1, got %d", seq)
	}

	// Verify final fragment: FIR=0, FIN=1, SEQ=2
	fir, fin, seq = ParseHeader(segments[2][0])
	if fir || !fin {
		t.Errorf("Fragment 2: Expected FIR=0 FIN=1, got FIR=%v FIN=%v", fir, fin)
	}
	if seq != 2 {
		t.Errorf("Fragment 2: Expected SEQ=2, got %d", seq)
	}

	// Verify statistics
	stats := outstation.GetStats()
	if stats.GetTxFragments() != 3 {
		t.Errorf("Expected 3 TX fragments, got %d", stats.GetTxFragments())
	}
	if stats.GetTxMessages() != 1 {
		t.Errorf("Expected 1 TX message, got %d", stats.GetTxMessages())
	}
}

func TestOutstationTransport_SequenceIncrement(t *testing.T) {
	config := DefaultTransportConfig()
	outstation := NewOutstationTransport(config)

	apdu := []byte{0x01, 0x02, 0x03}

	// Send first message
	segments1 := outstation.Send(apdu)
	_, _, seq1 := ParseHeader(segments1[0][0])

	// Send second message (could be solicited or unsolicited)
	segments2 := outstation.Send(apdu)
	_, _, seq2 := ParseHeader(segments2[0][0])

	if seq2 != (seq1+1)&TransportSeqMask {
		t.Errorf("Sequence not incrementing: seq1=%d, seq2=%d", seq1, seq2)
	}
}

func TestOutstationTransport_SequenceWrapAround(t *testing.T) {
	config := DefaultTransportConfig()
	outstation := NewOutstationTransport(config)

	// Manually set sequence to 62
	outstation.SetTxSequence(62)

	// Create APDU requiring 4 fragments
	apdu := make([]byte, 900) // Will create 4 segments
	segments := outstation.Send(apdu)

	if len(segments) != 4 {
		t.Fatalf("Expected 4 segments, got %d", len(segments))
	}

	// Verify sequences: 62, 63, 0, 1
	expectedSeqs := []uint8{62, 63, 0, 1}
	for i, seg := range segments {
		_, _, seq := ParseHeader(seg[0])
		if seq != expectedSeqs[i] {
			t.Errorf("Fragment %d: Expected SEQ=%d, got %d", i, expectedSeqs[i], seq)
		}
	}
}

func TestOutstationTransport_ReceiveSingleFragment(t *testing.T) {
	config := DefaultTransportConfig()
	outstation := NewOutstationTransport(config)

	expectedAPDU := []byte{0x10, 0x20, 0x30}

	// Create transport segment: FIR=1, FIN=1, SEQ=0
	tpdu := make([]byte, len(expectedAPDU)+1)
	tpdu[0] = 0xC0 // FIR=1, FIN=1, SEQ=0
	copy(tpdu[1:], expectedAPDU)

	apdu, err := outstation.Receive(tpdu)
	if err != nil {
		t.Fatalf("Receive error: %v", err)
	}

	if !bytes.Equal(apdu, expectedAPDU) {
		t.Errorf("APDU mismatch: expected %v, got %v", expectedAPDU, apdu)
	}

	// Verify statistics
	stats := outstation.GetStats()
	if stats.GetRxFragments() != 1 {
		t.Errorf("Expected 1 RX fragment, got %d", stats.GetRxFragments())
	}
	if stats.GetRxMessages() != 1 {
		t.Errorf("Expected 1 RX message, got %d", stats.GetRxMessages())
	}
}

func TestOutstationTransport_ReceiveMultipleFragments(t *testing.T) {
	config := DefaultTransportConfig()
	outstation := NewOutstationTransport(config)

	// Create 3 fragments
	fragment1 := make([]byte, 249) // 1 header + 248 data
	fragment1[0] = 0x40            // FIR=1, FIN=0, SEQ=0
	for i := 1; i < len(fragment1); i++ {
		fragment1[i] = byte(i)
	}

	fragment2 := make([]byte, 249)
	fragment2[0] = 0x01 // FIR=0, FIN=0, SEQ=1
	for i := 1; i < len(fragment2); i++ {
		fragment2[i] = byte(i + 248)
	}

	fragment3 := make([]byte, 105) // 1 header + 104 data
	fragment3[0] = 0x82            // FIR=0, FIN=1, SEQ=2
	for i := 1; i < len(fragment3); i++ {
		fragment3[i] = byte(i + 496)
	}

	// Receive first fragment - should return nil (incomplete)
	apdu, err := outstation.Receive(fragment1)
	if err != nil {
		t.Fatalf("Fragment 1 error: %v", err)
	}
	if apdu != nil {
		t.Error("Fragment 1 should not return complete APDU")
	}

	// Receive second fragment - should return nil (incomplete)
	apdu, err = outstation.Receive(fragment2)
	if err != nil {
		t.Fatalf("Fragment 2 error: %v", err)
	}
	if apdu != nil {
		t.Error("Fragment 2 should not return complete APDU")
	}

	// Receive third fragment - should return complete APDU
	apdu, err = outstation.Receive(fragment3)
	if err != nil {
		t.Fatalf("Fragment 3 error: %v", err)
	}
	if apdu == nil {
		t.Fatal("Fragment 3 should return complete APDU")
	}

	// Verify total size: 248 + 248 + 104 = 600
	if len(apdu) != 600 {
		t.Errorf("Expected 600 bytes, got %d", len(apdu))
	}

	// Verify statistics
	stats := outstation.GetStats()
	if stats.GetRxFragments() != 3 {
		t.Errorf("Expected 3 RX fragments, got %d", stats.GetRxFragments())
	}
	if stats.GetRxMessages() != 1 {
		t.Errorf("Expected 1 RX message, got %d", stats.GetRxMessages())
	}
}

func TestOutstationTransport_SequenceError(t *testing.T) {
	config := DefaultTransportConfig()
	outstation := NewOutstationTransport(config)

	// Receive first fragment: SEQ=0
	fragment1 := []byte{0x40, 0x01, 0x02} // FIR=1, FIN=0, SEQ=0
	apdu, err := outstation.Receive(fragment1)
	if err != nil {
		t.Fatalf("Fragment 1 error: %v", err)
	}
	if apdu != nil {
		t.Error("Should not complete on first fragment")
	}

	// Receive fragment with wrong sequence: SEQ=5 (expected 1)
	fragment2 := []byte{0x05, 0x03, 0x04} // FIR=0, FIN=0, SEQ=5
	apdu, err = outstation.Receive(fragment2)
	if err != nil {
		t.Fatalf("Fragment 2 error: %v", err)
	}
	if apdu != nil {
		t.Error("Should not complete on sequence error")
	}

	// Verify sequence error was counted
	stats := outstation.GetStats()
	if stats.GetSequenceErrors() != 1 {
		t.Errorf("Expected 1 sequence error, got %d", stats.GetSequenceErrors())
	}

	// Verify reassembly was reset
	if outstation.IsReassembling() {
		t.Error("Should not be reassembling after sequence error")
	}
}

func TestOutstationTransport_ReassemblyTimeout(t *testing.T) {
	config := DefaultTransportConfig()
	config.ReassemblyTimeout = 100 * time.Millisecond // Short timeout for testing
	outstation := NewOutstationTransport(config)

	// Receive first fragment
	fragment1 := []byte{0x40, 0x01, 0x02} // FIR=1, FIN=0, SEQ=0
	_, err := outstation.Receive(fragment1)
	if err != nil {
		t.Fatalf("Fragment 1 error: %v", err)
	}

	// Verify reassembly in progress
	if !outstation.IsReassembling() {
		t.Error("Should be reassembling")
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Verify reassembly was reset
	if outstation.IsReassembling() {
		t.Error("Should not be reassembling after timeout")
	}

	// Verify timeout error was counted
	stats := outstation.GetStats()
	if stats.GetTimeoutErrors() != 1 {
		t.Errorf("Expected 1 timeout error, got %d", stats.GetTimeoutErrors())
	}
}

func TestOutstationTransport_Reset(t *testing.T) {
	config := DefaultTransportConfig()
	outstation := NewOutstationTransport(config)

	// Send a message
	apdu := []byte{0x01, 0x02, 0x03}
	outstation.Send(apdu)

	// Verify statistics
	stats := outstation.GetStats()
	if stats.GetTxMessages() != 1 {
		t.Error("Expected statistics before reset")
	}

	// Verify sequence incremented
	if outstation.GetTxSequence() != 1 {
		t.Error("Sequence should be 1")
	}

	// Reset
	outstation.Reset()

	// Verify statistics cleared
	stats = outstation.GetStats()
	if stats.GetTxMessages() != 0 {
		t.Error("Statistics should be cleared after reset")
	}

	// Verify sequence reset
	if outstation.GetTxSequence() != 0 {
		t.Error("Sequence should be reset to 0")
	}

	// Next message should start at SEQ=0
	segments := outstation.Send(apdu)
	_, _, seq := ParseHeader(segments[0][0])
	if seq != 0 {
		t.Errorf("Expected SEQ=0 after reset, got %d", seq)
	}
}

func TestOutstationTransport_NewFIRInterruptsReassembly(t *testing.T) {
	config := DefaultTransportConfig()
	outstation := NewOutstationTransport(config)

	// Start reassembly
	fragment1 := []byte{0x40, 0x01, 0x02} // FIR=1, FIN=0, SEQ=0
	_, err := outstation.Receive(fragment1)
	if err != nil {
		t.Fatalf("Fragment 1 error: %v", err)
	}

	// Verify reassembling
	if !outstation.IsReassembling() {
		t.Error("Should be reassembling")
	}

	// New message with FIR interrupts
	fragment2 := []byte{0xC0, 0x03, 0x04} // FIR=1, FIN=1, SEQ=0
	apdu, err := outstation.Receive(fragment2)
	if err != nil {
		t.Fatalf("Fragment 2 error: %v", err)
	}

	// Should get the new complete message
	if !bytes.Equal(apdu, []byte{0x03, 0x04}) {
		t.Errorf("Should receive new message: %v", apdu)
	}

	// Should not be reassembling anymore
	if outstation.IsReassembling() {
		t.Error("Should not be reassembling after complete message")
	}
}

func TestOutstationTransport_SetTxSequence(t *testing.T) {
	config := DefaultTransportConfig()
	outstation := NewOutstationTransport(config)

	// Set sequence to specific value
	outstation.SetTxSequence(42)

	if outstation.GetTxSequence() != 42 {
		t.Errorf("Expected sequence 42, got %d", outstation.GetTxSequence())
	}

	// Send message should use that sequence
	apdu := []byte{0x01, 0x02, 0x03}
	segments := outstation.Send(apdu)
	_, _, seq := ParseHeader(segments[0][0])

	if seq != 42 {
		t.Errorf("Expected SEQ=42, got %d", seq)
	}

	// Next send should be 43
	segments = outstation.Send(apdu)
	_, _, seq = ParseHeader(segments[0][0])

	if seq != 43 {
		t.Errorf("Expected SEQ=43, got %d", seq)
	}
}

func TestOutstationTransport_ContinuationWithoutFIR(t *testing.T) {
	config := DefaultTransportConfig()
	outstation := NewOutstationTransport(config)

	// Receive continuation fragment without starting with FIR
	fragment := []byte{0x01, 0x01, 0x02} // FIR=0, FIN=0, SEQ=1
	apdu, err := outstation.Receive(fragment)
	if err != nil {
		t.Fatalf("Should not error: %v", err)
	}

	// Should silently discard
	if apdu != nil {
		t.Error("Should not return APDU for continuation without FIR")
	}

	// Should not be reassembling
	if outstation.IsReassembling() {
		t.Error("Should not be reassembling")
	}
}

func TestOutstationTransport_EmptyAPDU(t *testing.T) {
	config := DefaultTransportConfig()
	outstation := NewOutstationTransport(config)

	segments := outstation.Send([]byte{})

	if segments != nil {
		t.Error("Empty APDU should return nil")
	}

	segments = outstation.Send(nil)

	if segments != nil {
		t.Error("Nil APDU should return nil")
	}
}

func TestOutstationTransport_StatisticsDisabled(t *testing.T) {
	config := DefaultTransportConfig()
	config.EnableStatistics = false
	outstation := NewOutstationTransport(config)

	apdu := []byte{0x01, 0x02, 0x03}
	outstation.Send(apdu)

	stats := outstation.GetStats()
	// Statistics should not be incremented when disabled
	if stats.GetTxMessages() != 0 {
		t.Error("Statistics should not increment when disabled")
	}
}
