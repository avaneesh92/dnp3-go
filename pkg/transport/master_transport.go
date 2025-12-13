package transport

import (
	"sync"
	"time"
)

// MasterTransport manages transport layer for DNP3 master station
// Handles multiple concurrent outstations with independent sequencing
type MasterTransport struct {
	// Per-outstation state tracking
	outstations map[uint16]*outstationState

	// Configuration
	config TransportConfig

	// Synchronization
	mu sync.RWMutex
}

// outstationState tracks transport layer state for one outstation
type outstationState struct {
	// TX direction (Master → Outstation)
	txSequence uint8

	// RX direction (Outstation → Master)
	rxReassembler   *Reassembler
	reassemblyTimer *time.Timer

	// Statistics
	stats *TransportStatistics

	// Synchronization
	mu sync.Mutex
}

// NewMasterTransport creates a new master transport layer
func NewMasterTransport(config TransportConfig) *MasterTransport {
	return &MasterTransport{
		outstations: make(map[uint16]*outstationState),
		config:      config,
	}
}

// getOrCreateOutstation gets or creates state for an outstation
func (m *MasterTransport) getOrCreateOutstation(addr uint16) *outstationState {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.outstations[addr]
	if !exists {
		state = &outstationState{
			txSequence:    0,
			rxReassembler: NewReassembler(),
			stats:         NewTransportStatistics(),
		}
		m.outstations[addr] = state
	}
	return state
}

// Send segments an APDU for transmission to a specific outstation
// Returns array of transport segments ready for link layer
func (m *MasterTransport) Send(outstationAddr uint16, apdu []byte) [][]byte {
	if len(apdu) == 0 {
		return nil
	}

	state := m.getOrCreateOutstation(outstationAddr)
	state.mu.Lock()
	defer state.mu.Unlock()

	// Segment the APDU
	segments := SegmentData(apdu, state.txSequence)

	// Update sequence for next transmission
	state.txSequence = (state.txSequence + uint8(len(segments))) & TransportSeqMask

	// Update statistics
	if m.config.EnableStatistics {
		for range segments {
			state.stats.IncrementTxFragments()
		}
		state.stats.IncrementTxMessages()
	}

	// Serialize segments
	result := make([][]byte, len(segments))
	for i, seg := range segments {
		result[i] = seg.Serialize()
	}

	return result
}

// Receive processes a received transport segment from an outstation
// Returns complete APDU if reassembly is complete, nil if more fragments needed
func (m *MasterTransport) Receive(outstationAddr uint16, tpdu []byte) ([]byte, error) {
	if len(tpdu) < HeaderSize {
		return nil, ErrMissingFIR
	}

	state := m.getOrCreateOutstation(outstationAddr)
	state.mu.Lock()
	defer state.mu.Unlock()

	// Update statistics
	if m.config.EnableStatistics {
		state.stats.IncrementRxFragments()
	}

	// Parse header
	fir, fin, seq := ParseHeader(tpdu[0])

	// Create segment
	segment := &Segment{
		FIR:  fir,
		FIN:  fin,
		Seq:  seq,
		Data: tpdu[1:],
	}

	// Handle reassembly timer
	if segment.FIR {
		// Start of new message - start/restart timer
		m.startReassemblyTimer(state, outstationAddr)
	}

	// Process through reassembler
	apdu, err := state.rxReassembler.Process(segment)

	// Handle errors
	if err != nil {
		m.stopReassemblyTimer(state)
		if err == ErrBufferOverflow {
			if m.config.EnableStatistics {
				state.stats.IncrementBufferOverflows()
			}
		}
		return nil, err
	}

	// Check for sequence errors (reassembler returns nil on sequence mismatch)
	if apdu == nil && !state.rxReassembler.InProgress() && !segment.FIR {
		// Sequence error occurred
		if m.config.EnableStatistics {
			state.stats.IncrementSequenceErrors()
		}
	}

	// If reassembly complete, stop timer and update stats
	if apdu != nil {
		m.stopReassemblyTimer(state)
		if m.config.EnableStatistics {
			state.stats.IncrementRxMessages()
		}
	}

	return apdu, nil
}

// startReassemblyTimer starts or restarts the reassembly timeout timer
func (m *MasterTransport) startReassemblyTimer(state *outstationState, addr uint16) {
	// Stop existing timer if any
	m.stopReassemblyTimer(state)

	// Start new timer
	state.reassemblyTimer = time.AfterFunc(m.config.ReassemblyTimeout, func() {
		state.mu.Lock()
		defer state.mu.Unlock()

		// Timeout - discard incomplete message
		if state.rxReassembler.InProgress() {
			state.rxReassembler.Reset()
			if m.config.EnableStatistics {
				state.stats.IncrementTimeoutErrors()
			}
		}
	})
}

// stopReassemblyTimer stops the reassembly timer
func (m *MasterTransport) stopReassemblyTimer(state *outstationState) {
	if state.reassemblyTimer != nil {
		state.reassemblyTimer.Stop()
		state.reassemblyTimer = nil
	}
}

// GetStats returns statistics for a specific outstation
func (m *MasterTransport) GetStats(outstationAddr uint16) *TransportStatistics {
	m.mu.RLock()
	state, exists := m.outstations[outstationAddr]
	m.mu.RUnlock()

	if !exists {
		return NewTransportStatistics()
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	return state.stats
}

// Reset resets transport state for a specific outstation
func (m *MasterTransport) Reset(outstationAddr uint16) {
	m.mu.RLock()
	state, exists := m.outstations[outstationAddr]
	m.mu.RUnlock()

	if !exists {
		return
	}

	state.mu.Lock()
	defer state.mu.Unlock()

	// Stop timer
	m.stopReassemblyTimer(state)

	// Reset state
	state.rxReassembler.Reset()
	state.txSequence = 0
	if m.config.EnableStatistics {
		state.stats.Reset()
	}
}

// ResetAll resets transport state for all outstations
func (m *MasterTransport) ResetAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, state := range m.outstations {
		state.mu.Lock()
		m.stopReassemblyTimer(state)
		state.rxReassembler.Reset()
		state.txSequence = 0
		if m.config.EnableStatistics {
			state.stats.Reset()
		}
		state.mu.Unlock()
	}
}

// RemoveOutstation removes state for a specific outstation
// Use this when an outstation is removed from the system
func (m *MasterTransport) RemoveOutstation(outstationAddr uint16) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if state, exists := m.outstations[outstationAddr]; exists {
		state.mu.Lock()
		m.stopReassemblyTimer(state)
		state.mu.Unlock()
		delete(m.outstations, outstationAddr)
	}
}

// GetOutstationAddresses returns list of all tracked outstation addresses
func (m *MasterTransport) GetOutstationAddresses() []uint16 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	addrs := make([]uint16, 0, len(m.outstations))
	for addr := range m.outstations {
		addrs = append(addrs, addr)
	}
	return addrs
}

// IsReassembling returns true if reassembly is in progress for an outstation
func (m *MasterTransport) IsReassembling(outstationAddr uint16) bool {
	m.mu.RLock()
	state, exists := m.outstations[outstationAddr]
	m.mu.RUnlock()

	if !exists {
		return false
	}

	state.mu.Lock()
	defer state.mu.Unlock()
	return state.rxReassembler.InProgress()
}
