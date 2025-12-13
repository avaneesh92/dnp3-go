package transport

import "time"

// TransportConfig holds configuration for transport layer
type TransportConfig struct {
	// ReassemblyTimeout is the maximum time to wait for complete message
	// Default: 120 seconds per DNP3 specification
	ReassemblyTimeout time.Duration

	// MaxReassemblySize is the maximum buffer size for reassembly
	// Default: 2048 bytes (typical DNP3 limit)
	MaxReassemblySize int

	// EnableStatistics enables statistics collection
	EnableStatistics bool
}

// DefaultTransportConfig returns default transport configuration
func DefaultTransportConfig() TransportConfig {
	return TransportConfig{
		ReassemblyTimeout: 120 * time.Second,
		MaxReassemblySize: MaxReassemblySize,
		EnableStatistics:  true,
	}
}
