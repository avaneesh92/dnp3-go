package channel

import "sync/atomic"

// Statistics tracks channel-level statistics
type Statistics struct {
	// Link layer statistics
	numLinkFramesTx   uint64
	numLinkFramesRx   uint64
	numBadLinkFrames  uint64
	numCRCErrors      uint64

	// Transport layer statistics
	numTransportTx    uint64
	numTransportRx    uint64
	numTransportErrors uint64

	// Session statistics
	numActiveSessions uint64
}

// NewStatistics creates a new statistics tracker
func NewStatistics() *Statistics {
	return &Statistics{}
}

// LinkFrameTx increments transmitted link frames
func (s *Statistics) LinkFrameTx() {
	atomic.AddUint64(&s.numLinkFramesTx, 1)
}

// LinkFrameRx increments received link frames
func (s *Statistics) LinkFrameRx() {
	atomic.AddUint64(&s.numLinkFramesRx, 1)
}

// BadLinkFrame increments bad link frames
func (s *Statistics) BadLinkFrame() {
	atomic.AddUint64(&s.numBadLinkFrames, 1)
}

// CRCError increments CRC errors
func (s *Statistics) CRCError() {
	atomic.AddUint64(&s.numCRCErrors, 1)
}

// TransportTx increments transmitted transport segments
func (s *Statistics) TransportTx() {
	atomic.AddUint64(&s.numTransportTx, 1)
}

// TransportRx increments received transport segments
func (s *Statistics) TransportRx() {
	atomic.AddUint64(&s.numTransportRx, 1)
}

// TransportError increments transport errors
func (s *Statistics) TransportError() {
	atomic.AddUint64(&s.numTransportErrors, 1)
}

// SetActiveSessions sets the number of active sessions
func (s *Statistics) SetActiveSessions(count uint64) {
	atomic.StoreUint64(&s.numActiveSessions, count)
}

// GetLinkFramesTx returns transmitted link frames
func (s *Statistics) GetLinkFramesTx() uint64 {
	return atomic.LoadUint64(&s.numLinkFramesTx)
}

// GetLinkFramesRx returns received link frames
func (s *Statistics) GetLinkFramesRx() uint64 {
	return atomic.LoadUint64(&s.numLinkFramesRx)
}

// GetBadLinkFrames returns bad link frames
func (s *Statistics) GetBadLinkFrames() uint64 {
	return atomic.LoadUint64(&s.numBadLinkFrames)
}

// GetCRCErrors returns CRC errors
func (s *Statistics) GetCRCErrors() uint64 {
	return atomic.LoadUint64(&s.numCRCErrors)
}

// GetTransportTx returns transmitted transport segments
func (s *Statistics) GetTransportTx() uint64 {
	return atomic.LoadUint64(&s.numTransportTx)
}

// GetTransportRx returns received transport segments
func (s *Statistics) GetTransportRx() uint64 {
	return atomic.LoadUint64(&s.numTransportRx)
}

// GetTransportErrors returns transport errors
func (s *Statistics) GetTransportErrors() uint64 {
	return atomic.LoadUint64(&s.numTransportErrors)
}

// GetActiveSessions returns number of active sessions
func (s *Statistics) GetActiveSessions() uint64 {
	return atomic.LoadUint64(&s.numActiveSessions)
}

// Reset resets all statistics
func (s *Statistics) Reset() {
	atomic.StoreUint64(&s.numLinkFramesTx, 0)
	atomic.StoreUint64(&s.numLinkFramesRx, 0)
	atomic.StoreUint64(&s.numBadLinkFrames, 0)
	atomic.StoreUint64(&s.numCRCErrors, 0)
	atomic.StoreUint64(&s.numTransportTx, 0)
	atomic.StoreUint64(&s.numTransportRx, 0)
	atomic.StoreUint64(&s.numTransportErrors, 0)
}
