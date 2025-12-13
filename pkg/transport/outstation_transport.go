package transport

import (
	"sync"
	"time"
)

// OutstationTransport manages transport layer for DNP3 outstation
// Simpler than master - only communicates with one master
type OutstationTransport struct {
	// TX direction (Outstation → Master)
	txSequence uint8

	// RX direction (Master → Outstation)
	rxReassembler   *Reassembler
	reassemblyTimer *time.Timer

	// Configuration
	config TransportConfig

	// Statistics
	stats *TransportStatistics

	// Synchronization
	mu sync.Mutex
}

// NewOutstationTransport creates a new outstation transport layer
func NewOutstationTransport(config TransportConfig) *OutstationTransport {
	return &OutstationTransport{
		txSequence:    0,
		rxReassembler: NewReassembler(),
		config:        config,
		stats:         NewTransportStatistics(),
	}
}

// Send segments an APDU for transmission to the master
// Returns array of transport segments ready for link layer
// Used for both solicited responses and unsolicited messages
func (o *OutstationTransport) Send(apdu []byte) [][]byte {
	if len(apdu) == 0 {
		return nil
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	// Segment the APDU
	segments := SegmentData(apdu, o.txSequence)

	// Update sequence for next transmission
	o.txSequence = (o.txSequence + uint8(len(segments))) & TransportSeqMask

	// Update statistics
	if o.config.EnableStatistics {
		for range segments {
			o.stats.IncrementTxFragments()
		}
		o.stats.IncrementTxMessages()
	}

	// Serialize segments
	result := make([][]byte, len(segments))
	for i, seg := range segments {
		result[i] = seg.Serialize()
	}

	return result
}

// Receive processes a received transport segment from the master
// Returns complete APDU if reassembly is complete, nil if more fragments needed
func (o *OutstationTransport) Receive(tpdu []byte) ([]byte, error) {
	if len(tpdu) < HeaderSize {
		return nil, ErrMissingFIR
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	// Update statistics
	if o.config.EnableStatistics {
		o.stats.IncrementRxFragments()
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
		o.startReassemblyTimer()
	}

	// Process through reassembler
	apdu, err := o.rxReassembler.Process(segment)

	// Handle errors
	if err != nil {
		o.stopReassemblyTimer()
		if err == ErrBufferOverflow {
			if o.config.EnableStatistics {
				o.stats.IncrementBufferOverflows()
			}
		}
		return nil, err
	}

	// Check for sequence errors (reassembler returns nil on sequence mismatch)
	if apdu == nil && !o.rxReassembler.InProgress() && !segment.FIR {
		// Sequence error occurred
		if o.config.EnableStatistics {
			o.stats.IncrementSequenceErrors()
		}
	}

	// If reassembly complete, stop timer and update stats
	if apdu != nil {
		o.stopReassemblyTimer()
		if o.config.EnableStatistics {
			o.stats.IncrementRxMessages()
		}
	}

	return apdu, nil
}

// startReassemblyTimer starts or restarts the reassembly timeout timer
func (o *OutstationTransport) startReassemblyTimer() {
	// Stop existing timer if any
	o.stopReassemblyTimer()

	// Start new timer
	o.reassemblyTimer = time.AfterFunc(o.config.ReassemblyTimeout, func() {
		o.mu.Lock()
		defer o.mu.Unlock()

		// Timeout - discard incomplete message
		if o.rxReassembler.InProgress() {
			o.rxReassembler.Reset()
			if o.config.EnableStatistics {
				o.stats.IncrementTimeoutErrors()
			}
		}
	})
}

// stopReassemblyTimer stops the reassembly timer
func (o *OutstationTransport) stopReassemblyTimer() {
	if o.reassemblyTimer != nil {
		o.reassemblyTimer.Stop()
		o.reassemblyTimer = nil
	}
}

// GetStats returns transport layer statistics
func (o *OutstationTransport) GetStats() *TransportStatistics {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.stats
}

// Reset resets transport layer state
func (o *OutstationTransport) Reset() {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Stop timer
	o.stopReassemblyTimer()

	// Reset state
	o.rxReassembler.Reset()
	o.txSequence = 0
	if o.config.EnableStatistics {
		o.stats.Reset()
	}
}

// IsReassembling returns true if reassembly is in progress
func (o *OutstationTransport) IsReassembling() bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.rxReassembler.InProgress()
}

// GetTxSequence returns the current TX sequence number (for diagnostics)
func (o *OutstationTransport) GetTxSequence() uint8 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.txSequence
}

// SetTxSequence sets the TX sequence number (use with caution)
// This can be used to initialize the sequence to a specific value
func (o *OutstationTransport) SetTxSequence(seq uint8) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.txSequence = seq & TransportSeqMask
}
