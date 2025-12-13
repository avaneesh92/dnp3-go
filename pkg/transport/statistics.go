package transport

import (
	"sync/atomic"
	"time"
)

// TransportStatistics tracks transport layer metrics
type TransportStatistics struct {
	// Fragment counts
	TxFragments uint64
	RxFragments uint64

	// Message counts
	TxMessages uint64
	RxMessages uint64

	// Error counts
	SequenceErrors   uint64
	TimeoutErrors    uint64
	BufferOverflows  uint64

	// Timing (stored as Unix nano for atomic operations)
	lastTxTimeNano int64
	lastRxTimeNano int64
}

// NewTransportStatistics creates a new statistics tracker
func NewTransportStatistics() *TransportStatistics {
	return &TransportStatistics{}
}

// IncrementTxFragments increments transmitted fragment count
func (s *TransportStatistics) IncrementTxFragments() {
	atomic.AddUint64(&s.TxFragments, 1)
}

// IncrementRxFragments increments received fragment count
func (s *TransportStatistics) IncrementRxFragments() {
	atomic.AddUint64(&s.RxFragments, 1)
}

// IncrementTxMessages increments transmitted message count
func (s *TransportStatistics) IncrementTxMessages() {
	atomic.AddUint64(&s.TxMessages, 1)
	atomic.StoreInt64(&s.lastTxTimeNano, time.Now().UnixNano())
}

// IncrementRxMessages increments received message count
func (s *TransportStatistics) IncrementRxMessages() {
	atomic.AddUint64(&s.RxMessages, 1)
	atomic.StoreInt64(&s.lastRxTimeNano, time.Now().UnixNano())
}

// IncrementSequenceErrors increments sequence error count
func (s *TransportStatistics) IncrementSequenceErrors() {
	atomic.AddUint64(&s.SequenceErrors, 1)
}

// IncrementTimeoutErrors increments timeout error count
func (s *TransportStatistics) IncrementTimeoutErrors() {
	atomic.AddUint64(&s.TimeoutErrors, 1)
}

// IncrementBufferOverflows increments buffer overflow count
func (s *TransportStatistics) IncrementBufferOverflows() {
	atomic.AddUint64(&s.BufferOverflows, 1)
}

// GetTxFragments returns transmitted fragment count
func (s *TransportStatistics) GetTxFragments() uint64 {
	return atomic.LoadUint64(&s.TxFragments)
}

// GetRxFragments returns received fragment count
func (s *TransportStatistics) GetRxFragments() uint64 {
	return atomic.LoadUint64(&s.RxFragments)
}

// GetTxMessages returns transmitted message count
func (s *TransportStatistics) GetTxMessages() uint64 {
	return atomic.LoadUint64(&s.TxMessages)
}

// GetRxMessages returns received message count
func (s *TransportStatistics) GetRxMessages() uint64 {
	return atomic.LoadUint64(&s.RxMessages)
}

// GetSequenceErrors returns sequence error count
func (s *TransportStatistics) GetSequenceErrors() uint64 {
	return atomic.LoadUint64(&s.SequenceErrors)
}

// GetTimeoutErrors returns timeout error count
func (s *TransportStatistics) GetTimeoutErrors() uint64 {
	return atomic.LoadUint64(&s.TimeoutErrors)
}

// GetBufferOverflows returns buffer overflow count
func (s *TransportStatistics) GetBufferOverflows() uint64 {
	return atomic.LoadUint64(&s.BufferOverflows)
}

// GetLastTxTime returns the last transmission time
func (s *TransportStatistics) GetLastTxTime() time.Time {
	nano := atomic.LoadInt64(&s.lastTxTimeNano)
	if nano == 0 {
		return time.Time{}
	}
	return time.Unix(0, nano)
}

// GetLastRxTime returns the last reception time
func (s *TransportStatistics) GetLastRxTime() time.Time {
	nano := atomic.LoadInt64(&s.lastRxTimeNano)
	if nano == 0 {
		return time.Time{}
	}
	return time.Unix(0, nano)
}

// Reset resets all statistics to zero
func (s *TransportStatistics) Reset() {
	atomic.StoreUint64(&s.TxFragments, 0)
	atomic.StoreUint64(&s.RxFragments, 0)
	atomic.StoreUint64(&s.TxMessages, 0)
	atomic.StoreUint64(&s.RxMessages, 0)
	atomic.StoreUint64(&s.SequenceErrors, 0)
	atomic.StoreUint64(&s.TimeoutErrors, 0)
	atomic.StoreUint64(&s.BufferOverflows, 0)
	atomic.StoreInt64(&s.lastTxTimeNano, 0)
	atomic.StoreInt64(&s.lastRxTimeNano, 0)
}
