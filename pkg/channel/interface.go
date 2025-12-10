package channel

import "context"

// ConnectionStateListener receives notifications about connection state changes
type ConnectionStateListener interface {
	// OnConnectionEstablished is called when a new connection is established
	OnConnectionEstablished()

	// OnConnectionLost is called when a connection is lost
	OnConnectionLost()
}

// PhysicalChannel represents a pluggable transport layer
// Users implement this interface to provide TCP, Serial, or any custom transport
// This is THE KEY INTERFACE that enables pluggable transports
type PhysicalChannel interface {
	// Read reads the next frame from the physical medium
	// Should block until data is available or context is cancelled
	// Returns complete frame data (link layer frame) or error
	// Implementations must handle timeouts internally or via context
	Read(ctx context.Context) ([]byte, error)

	// Write writes a frame to the physical medium
	// Must be thread-safe as multiple sessions may write concurrently
	// Should complete the write or return error
	Write(ctx context.Context, data []byte) error

	// Close closes the physical connection
	// Should cleanup all resources and unblock any pending Read/Write
	Close() error

	// Statistics returns transport-level statistics
	// Optional - can return zero values if not tracked
	Statistics() TransportStats

	// SetConnectionStateListener sets a listener for connection state changes
	// Optional - channels that don't support connection state notifications can ignore this
	SetConnectionStateListener(listener ConnectionStateListener)
}

// TransportStats provides transport-level statistics
type TransportStats struct {
	BytesSent     uint64 // Total bytes sent
	BytesReceived uint64 // Total bytes received
	WriteErrors   uint64 // Number of write errors
	ReadErrors    uint64 // Number of read errors
	Connects      uint64 // Number of connections (for connection-oriented transports)
	Disconnects   uint64 // Number of disconnections
}

// ChannelState represents the state of a channel
type ChannelState int

const (
	ChannelStateOpen ChannelState = iota
	ChannelStateClosed
)

// String returns string representation of ChannelState
func (s ChannelState) String() string {
	switch s {
	case ChannelStateOpen:
		return "Open"
	case ChannelStateClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}
