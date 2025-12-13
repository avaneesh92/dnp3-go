package transport

import (
	"bytes"
	"testing"
	"time"
)

func TestMasterTransport_SendSingleFragment(t *testing.T) {
	config := DefaultTransportConfig()
	master := NewMasterTransport(config)

	// Small APDU that fits in one fragment
	apdu := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	outstationAddr := uint16(10)

	segments := master.Send(outstationAddr, apdu)

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
	stats := master.GetStats(outstationAddr)
	if stats.GetTxFragments() != 1 {
		t.Errorf("Expected 1 TX fragment, got %d", stats.GetTxFragments())
	}
	if stats.GetTxMessages() != 1 {
		t.Errorf("Expected 1 TX message, got %d", stats.GetTxMessages())
	}
}

func TestMasterTransport_SendMultipleFragments(t *testing.T) {
	config := DefaultTransportConfig()
	master := NewMasterTransport(config)

	// Create APDU that requires 3 fragments (600 bytes)
	apdu := make([]byte, 600)
	for i := range apdu {
		apdu[i] = byte(i % 256)
	}

	outstationAddr := uint16(10)
	segments := master.Send(outstationAddr, apdu)

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
	if len(segments[0])-1 != MaxSegmentSize {
		t.Errorf("Fragment 0: Expected %d bytes, got %d", MaxSegmentSize, len(segments[0])-1)
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
	if len(segments[2])-1 != 104 {
		t.Errorf("Fragment 2: Expected 104 bytes, got %d", len(segments[2])-1)
	}

	// Verify statistics
	stats := master.GetStats(outstationAddr)
	if stats.GetTxFragments() != 3 {
		t.Errorf("Expected 3 TX fragments, got %d", stats.GetTxFragments())
	}
	if stats.GetTxMessages() != 1 {
		t.Errorf("Expected 1 TX message, got %d", stats.GetTxMessages())
	}
}

func TestMasterTransport_SequenceIncrement(t *testing.T) {
	config := DefaultTransportConfig()
	master := NewMasterTransport(config)

	outstationAddr := uint16(10)
	apdu := []byte{0x01, 0x02, 0x03}

	// Send first message
	segments1 := master.Send(outstationAddr, apdu)
	_, _, seq1 := ParseHeader(segments1[0][0])

	// Send second message
	segments2 := master.Send(outstationAddr, apdu)
	_, _, seq2 := ParseHeader(segments2[0][0])

	if seq2 != (seq1+1)&TransportSeqMask {
		t.Errorf("Sequence not incrementing: seq1=%d, seq2=%d", seq1, seq2)
	}
}

func TestMasterTransport_SequenceWrapAround(t *testing.T) {
	config := DefaultTransportConfig()
	master := NewMasterTransport(config)

	outstationAddr := uint16(10)

	// Manually set sequence to 62
	state := master.getOrCreateOutstation(outstationAddr)
	state.txSequence = 62

	// Create APDU requiring 4 fragments
	apdu := make([]byte, 900) // Will create 4 segments
	segments := master.Send(outstationAddr, apdu)

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

func TestMasterTransport_MultipleOutstations(t *testing.T) {
	config := DefaultTransportConfig()
	master := NewMasterTransport(config)

	apdu := []byte{0x01, 0x02, 0x03}

	// Send to outstation 10
	segments10 := master.Send(10, apdu)
	_, _, seq10_1 := ParseHeader(segments10[0][0])

	// Send to outstation 20
	segments20 := master.Send(20, apdu)
	_, _, seq20_1 := ParseHeader(segments20[0][0])

	// Both should start at 0
	if seq10_1 != 0 || seq20_1 != 0 {
		t.Errorf("Expected both to start at SEQ=0, got %d and %d", seq10_1, seq20_1)
	}

	// Send again to outstation 10
	segments10_2 := master.Send(10, apdu)
	_, _, seq10_2 := ParseHeader(segments10_2[0][0])

	// Should increment independently
	if seq10_2 != 1 {
		t.Errorf("Expected outstation 10 SEQ=1, got %d", seq10_2)
	}

	// Verify independent statistics
	stats10 := master.GetStats(10)
	stats20 := master.GetStats(20)

	if stats10.GetTxMessages() != 2 {
		t.Errorf("Outstation 10: Expected 2 messages, got %d", stats10.GetTxMessages())
	}
	if stats20.GetTxMessages() != 1 {
		t.Errorf("Outstation 20: Expected 1 message, got %d", stats20.GetTxMessages())
	}
}

func TestMasterTransport_ReceiveSingleFragment(t *testing.T) {
	config := DefaultTransportConfig()
	master := NewMasterTransport(config)

	outstationAddr := uint16(10)
	expectedAPDU := []byte{0x10, 0x20, 0x30}

	// Create transport segment: FIR=1, FIN=1, SEQ=0
	tpdu := make([]byte, len(expectedAPDU)+1)
	tpdu[0] = 0xC0 // FIR=1, FIN=1, SEQ=0
	copy(tpdu[1:], expectedAPDU)

	apdu, err := master.Receive(outstationAddr, tpdu)
	if err != nil {
		t.Fatalf("Receive error: %v", err)
	}

	if !bytes.Equal(apdu, expectedAPDU) {
		t.Errorf("APDU mismatch: expected %v, got %v", expectedAPDU, apdu)
	}

	// Verify statistics
	stats := master.GetStats(outstationAddr)
	if stats.GetRxFragments() != 1 {
		t.Errorf("Expected 1 RX fragment, got %d", stats.GetRxFragments())
	}
	if stats.GetRxMessages() != 1 {
		t.Errorf("Expected 1 RX message, got %d", stats.GetRxMessages())
	}
}

func TestMasterTransport_ReceiveMultipleFragments(t *testing.T) {
	config := DefaultTransportConfig()
	master := NewMasterTransport(config)

	outstationAddr := uint16(10)

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
	apdu, err := master.Receive(outstationAddr, fragment1)
	if err != nil {
		t.Fatalf("Fragment 1 error: %v", err)
	}
	if apdu != nil {
		t.Error("Fragment 1 should not return complete APDU")
	}

	// Receive second fragment - should return nil (incomplete)
	apdu, err = master.Receive(outstationAddr, fragment2)
	if err != nil {
		t.Fatalf("Fragment 2 error: %v", err)
	}
	if apdu != nil {
		t.Error("Fragment 2 should not return complete APDU")
	}

	// Receive third fragment - should return complete APDU
	apdu, err = master.Receive(outstationAddr, fragment3)
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
	stats := master.GetStats(outstationAddr)
	if stats.GetRxFragments() != 3 {
		t.Errorf("Expected 3 RX fragments, got %d", stats.GetRxFragments())
	}
	if stats.GetRxMessages() != 1 {
		t.Errorf("Expected 1 RX message, got %d", stats.GetRxMessages())
	}
}

func TestMasterTransport_SequenceError(t *testing.T) {
	config := DefaultTransportConfig()
	master := NewMasterTransport(config)

	outstationAddr := uint16(10)

	// Receive first fragment: SEQ=0
	fragment1 := []byte{0x40, 0x01, 0x02} // FIR=1, FIN=0, SEQ=0
	apdu, err := master.Receive(outstationAddr, fragment1)
	if err != nil {
		t.Fatalf("Fragment 1 error: %v", err)
	}
	if apdu != nil {
		t.Error("Should not complete on first fragment")
	}

	// Receive fragment with wrong sequence: SEQ=5 (expected 1)
	fragment2 := []byte{0x05, 0x03, 0x04} // FIR=0, FIN=0, SEQ=5
	apdu, err = master.Receive(outstationAddr, fragment2)
	if err != nil {
		t.Fatalf("Fragment 2 error: %v", err)
	}
	if apdu != nil {
		t.Error("Should not complete on sequence error")
	}

	// Verify sequence error was counted
	stats := master.GetStats(outstationAddr)
	if stats.GetSequenceErrors() != 1 {
		t.Errorf("Expected 1 sequence error, got %d", stats.GetSequenceErrors())
	}

	// Verify reassembly was reset
	if master.IsReassembling(outstationAddr) {
		t.Error("Should not be reassembling after sequence error")
	}
}

func TestMasterTransport_ReassemblyTimeout(t *testing.T) {
	config := DefaultTransportConfig()
	config.ReassemblyTimeout = 100 * time.Millisecond // Short timeout for testing
	master := NewMasterTransport(config)

	outstationAddr := uint16(10)

	// Receive first fragment
	fragment1 := []byte{0x40, 0x01, 0x02} // FIR=1, FIN=0, SEQ=0
	_, err := master.Receive(outstationAddr, fragment1)
	if err != nil {
		t.Fatalf("Fragment 1 error: %v", err)
	}

	// Verify reassembly in progress
	if !master.IsReassembling(outstationAddr) {
		t.Error("Should be reassembling")
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Verify reassembly was reset
	if master.IsReassembling(outstationAddr) {
		t.Error("Should not be reassembling after timeout")
	}

	// Verify timeout error was counted
	stats := master.GetStats(outstationAddr)
	if stats.GetTimeoutErrors() != 1 {
		t.Errorf("Expected 1 timeout error, got %d", stats.GetTimeoutErrors())
	}
}

func TestMasterTransport_Reset(t *testing.T) {
	config := DefaultTransportConfig()
	master := NewMasterTransport(config)

	outstationAddr := uint16(10)

	// Send a message
	apdu := []byte{0x01, 0x02, 0x03}
	master.Send(outstationAddr, apdu)

	// Verify statistics
	stats := master.GetStats(outstationAddr)
	if stats.GetTxMessages() != 1 {
		t.Error("Expected statistics before reset")
	}

	// Reset
	master.Reset(outstationAddr)

	// Verify statistics cleared
	stats = master.GetStats(outstationAddr)
	if stats.GetTxMessages() != 0 {
		t.Error("Statistics should be cleared after reset")
	}

	// Next message should start at SEQ=0
	segments := master.Send(outstationAddr, apdu)
	_, _, seq := ParseHeader(segments[0][0])
	if seq != 0 {
		t.Errorf("Expected SEQ=0 after reset, got %d", seq)
	}
}

func TestMasterTransport_RemoveOutstation(t *testing.T) {
	config := DefaultTransportConfig()
	master := NewMasterTransport(config)

	// Create state for outstation 10
	apdu := []byte{0x01, 0x02, 0x03}
	master.Send(10, apdu)

	// Verify it exists
	addrs := master.GetOutstationAddresses()
	if len(addrs) != 1 || addrs[0] != 10 {
		t.Error("Outstation 10 should exist")
	}

	// Remove it
	master.RemoveOutstation(10)

	// Verify it's gone
	addrs = master.GetOutstationAddresses()
	if len(addrs) != 0 {
		t.Error("Outstation 10 should be removed")
	}
}

func TestMasterTransport_NewFIRInterruptsReassembly(t *testing.T) {
	config := DefaultTransportConfig()
	master := NewMasterTransport(config)

	outstationAddr := uint16(10)

	// Start reassembly
	fragment1 := []byte{0x40, 0x01, 0x02} // FIR=1, FIN=0, SEQ=0
	_, err := master.Receive(outstationAddr, fragment1)
	if err != nil {
		t.Fatalf("Fragment 1 error: %v", err)
	}

	// New message with FIR interrupts
	fragment2 := []byte{0xC0, 0x03, 0x04} // FIR=1, FIN=1, SEQ=0
	apdu, err := master.Receive(outstationAddr, fragment2)
	if err != nil {
		t.Fatalf("Fragment 2 error: %v", err)
	}

	// Should get the new complete message
	if !bytes.Equal(apdu, []byte{0x03, 0x04}) {
		t.Errorf("Should receive new message: %v", apdu)
	}
}
