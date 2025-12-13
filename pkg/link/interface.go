package link

import "time"

// LinkLayer defines the interface for DNP3 link layer operations
type LinkLayer interface {
	// Control operations
	ResetLink() error
	ResetUserProcess() error
	TestLink() error
	RequestLinkStatus() error

	// Data transmission
	SendConfirmedUserData(data []byte) error
	SendUnconfirmedUserData(data []byte) error

	// Reception handling
	OnFrameReceived(frame *Frame) error

	// State management
	GetState() LinkState
	IsOnline() bool

	// Configuration
	SetTimeout(duration time.Duration)
	SetRetries(count int)

	// Lifecycle
	Start() error
	Stop() error
}

// DataCallback is called when user data is received from the link layer
type DataCallback func(data []byte) error

// StatusCallback is called when link layer state changes
type StatusCallback func(state LinkState, err error)

// LinkLayerConfig contains configuration for link layer
type LinkLayerConfig struct {
	LocalAddress    uint16        // Local station address
	RemoteAddress   uint16        // Remote station address
	IsMaster        bool          // true for master, false for outstation
	Timeout         time.Duration // Response timeout
	MaxRetries      int           // Maximum number of retries
	DataCallback    DataCallback  // Callback for received user data
	StatusCallback  StatusCallback // Callback for status changes
}

// DefaultLinkLayerConfig returns default configuration
func DefaultLinkLayerConfig() LinkLayerConfig {
	return LinkLayerConfig{
		LocalAddress:  1,
		RemoteAddress: 1024,
		IsMaster:      true,
		Timeout:       2 * time.Second,
		MaxRetries:    3,
	}
}
