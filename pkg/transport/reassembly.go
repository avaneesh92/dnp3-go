package transport

import (
	"bytes"
	"errors"
)

var (
	ErrInvalidSequence = errors.New("invalid transport sequence")
	ErrMissingFIR      = errors.New("missing FIR segment")
	ErrUnexpectedFIR   = errors.New("unexpected FIR segment")
	ErrBufferOverflow  = errors.New("reassembly buffer overflow")
)

// MaxReassemblySize is the maximum size for reassembly
const MaxReassemblySize = 2048

// Reassembler handles reassembly of transport segments into APDUs
type Reassembler struct {
	buffer      bytes.Buffer
	expectedSeq uint8
	inProgress  bool
}

// NewReassembler creates a new transport reassembler
func NewReassembler() *Reassembler {
	return &Reassembler{
		expectedSeq: 0,
		inProgress:  false,
	}
}

// Process processes a transport segment
// Returns complete APDU if reassembly is complete, nil otherwise
func (r *Reassembler) Process(segment *Segment) ([]byte, error) {
	// Check for FIR (First) segment
	if segment.FIR {
		// If we were already reassembling, this starts over
		r.buffer.Reset()
		r.inProgress = true
		r.expectedSeq = segment.Seq
	} else if !r.inProgress {
		// Received non-FIR segment when not reassembling
		// This can happen after connection reset or transport desync
		// Silently discard and wait for FIR segment to resynchronize
		return nil, nil
	}

	// Verify sequence number
	if segment.Seq != r.expectedSeq {
		// Sequence error - reset reassembly and wait for next FIR
		r.Reset()
		return nil, nil
	}

	// Add data to buffer
	if r.buffer.Len()+len(segment.Data) > MaxReassemblySize {
		r.Reset()
		return nil, ErrBufferOverflow
	}

	r.buffer.Write(segment.Data)

	// Update expected sequence
	r.expectedSeq = (r.expectedSeq + 1) & TransportSeqMask

	// Check for FIN (Final) segment
	if segment.FIN {
		// Reassembly complete
		result := make([]byte, r.buffer.Len())
		copy(result, r.buffer.Bytes())
		r.Reset()
		return result, nil
	}

	// More segments expected
	return nil, nil
}

// Reset resets the reassembler state
func (r *Reassembler) Reset() {
	r.buffer.Reset()
	r.inProgress = false
	r.expectedSeq = 0
}

// InProgress returns true if reassembly is in progress
func (r *Reassembler) InProgress() bool {
	return r.inProgress
}
