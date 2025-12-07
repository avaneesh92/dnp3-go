package channel

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

// UDPChannel implements PhysicalChannel for UDP connections
type UDPChannel struct {
	// Connection
	conn     *net.UDPConn
	connLock sync.RWMutex

	// Configuration
	address      string
	isServer     bool
	remoteAddr   *net.UDPAddr // Used for client mode to know where to send
	lastPeerAddr *net.UDPAddr // Used for server mode to remember last peer
	peerLock     sync.RWMutex
	readTimeout  time.Duration
	writeTimeout time.Duration

	// Statistics
	stats struct {
		bytesSent     atomic.Uint64
		bytesReceived atomic.Uint64
		writeErrors   atomic.Uint64
		readErrors    atomic.Uint64
		connects      atomic.Uint64
		disconnects   atomic.Uint64
	}

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	closed atomic.Bool
}

// UDPChannelConfig configures a UDP channel
type UDPChannelConfig struct {
	Address      string        // "host:port" format
	IsServer     bool          // true = bind and listen, false = bind and send to remote
	ReadTimeout  time.Duration // Read timeout (0 = no timeout)
	WriteTimeout time.Duration // Write timeout (0 = no timeout)
}

// NewUDPChannel creates a new UDP channel
func NewUDPChannel(config UDPChannelConfig) (*UDPChannel, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("address is required")
	}

	// Set defaults
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 30 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 10 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	uc := &UDPChannel{
		address:      config.Address,
		isServer:     config.IsServer,
		readTimeout:  config.ReadTimeout,
		writeTimeout: config.WriteTimeout,
		ctx:          ctx,
		cancel:       cancel,
	}

	// Initialize connection
	if err := uc.initialize(); err != nil {
		cancel()
		return nil, err
	}

	return uc, nil
}

// initialize sets up the UDP connection
func (uc *UDPChannel) initialize() error {
	addr, err := net.ResolveUDPAddr("udp", uc.address)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address %s: %w", uc.address, err)
	}

	if uc.isServer {
		// Server mode: bind to local address to receive from any client
		conn, err := net.ListenUDP("udp", addr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", uc.address, err)
		}
		uc.conn = conn
	} else {
		// Client mode: bind to local address and remember remote address
		uc.remoteAddr = addr

		// Bind to any local address
		localAddr, err := net.ResolveUDPAddr("udp", ":0")
		if err != nil {
			return fmt.Errorf("failed to resolve local UDP address: %w", err)
		}

		conn, err := net.ListenUDP("udp", localAddr)
		if err != nil {
			return fmt.Errorf("failed to create UDP connection: %w", err)
		}
		uc.conn = conn
	}

	uc.stats.connects.Add(1)
	return nil
}

// Read implements PhysicalChannel.Read
func (uc *UDPChannel) Read(ctx context.Context) ([]byte, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-uc.ctx.Done():
			return nil, fmt.Errorf("channel closed")
		default:
		}

		uc.connLock.RLock()
		conn := uc.conn
		uc.connLock.RUnlock()

		if conn == nil {
			return nil, fmt.Errorf("no connection")
		}

		// Set read deadline
		if uc.readTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(uc.readTimeout))
		}

		// UDP datagrams can be up to 64KB, but DNP3 frames are typically much smaller
		// We'll use a reasonable buffer size for DNP3 frames
		buffer := make([]byte, 2048)
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout, continue to check context
				continue
			}
			if uc.closed.Load() {
				return nil, fmt.Errorf("channel closed")
			}
			uc.stats.readErrors.Add(1)
			return nil, err
		}

		// Store the remote address for server mode (to reply to the same peer)
		if uc.isServer && remoteAddr != nil {
			uc.peerLock.Lock()
			uc.lastPeerAddr = remoteAddr
			uc.peerLock.Unlock()
		}

		// Verify minimum DNP3 frame size (10 bytes)
		if n < 10 {
			uc.stats.readErrors.Add(1)
			continue
		}

		frame := buffer[:n]

		// Verify sync bytes
		if frame[0] != 0x05 || frame[1] != 0x64 {
			uc.stats.readErrors.Add(1)
			continue
		}

		uc.stats.bytesReceived.Add(uint64(n))
		return frame, nil
	}
}

// Write implements PhysicalChannel.Write
func (uc *UDPChannel) Write(ctx context.Context, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-uc.ctx.Done():
		return fmt.Errorf("channel closed")
	default:
	}

	uc.connLock.RLock()
	conn := uc.conn
	uc.connLock.RUnlock()

	if conn == nil {
		uc.stats.writeErrors.Add(1)
		return fmt.Errorf("no connection")
	}

	// Determine the destination address
	var destAddr *net.UDPAddr
	if uc.isServer {
		// Server mode: send to the last peer we received from
		uc.peerLock.RLock()
		destAddr = uc.lastPeerAddr
		uc.peerLock.RUnlock()

		if destAddr == nil {
			uc.stats.writeErrors.Add(1)
			return fmt.Errorf("no peer address available (no data received yet)")
		}
	} else {
		// Client mode: send to the configured remote address
		destAddr = uc.remoteAddr
	}

	// Set write deadline
	if uc.writeTimeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(uc.writeTimeout))
	}

	_, err := conn.WriteToUDP(data, destAddr)
	if err != nil {
		uc.stats.writeErrors.Add(1)
		return err
	}

	uc.stats.bytesSent.Add(uint64(len(data)))
	return nil
}

// Close implements PhysicalChannel.Close
func (uc *UDPChannel) Close() error {
	if !uc.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	// Cancel context
	uc.cancel()

	// Close connection
	uc.connLock.Lock()
	if uc.conn != nil {
		uc.conn.Close()
		uc.stats.disconnects.Add(1)
		uc.conn = nil
	}
	uc.connLock.Unlock()

	return nil
}

// Statistics implements PhysicalChannel.Statistics
func (uc *UDPChannel) Statistics() TransportStats {
	return TransportStats{
		BytesSent:     uc.stats.bytesSent.Load(),
		BytesReceived: uc.stats.bytesReceived.Load(),
		WriteErrors:   uc.stats.writeErrors.Load(),
		ReadErrors:    uc.stats.readErrors.Load(),
		Connects:      uc.stats.connects.Load(),
		Disconnects:   uc.stats.disconnects.Load(),
	}
}

// IsConnected returns true if the connection is open
// Note: For UDP, this just means the socket is bound
func (uc *UDPChannel) IsConnected() bool {
	uc.connLock.RLock()
	defer uc.connLock.RUnlock()
	return uc.conn != nil
}

// LocalAddr returns the local address of the connection
func (uc *UDPChannel) LocalAddr() net.Addr {
	uc.connLock.RLock()
	defer uc.connLock.RUnlock()
	if uc.conn != nil {
		return uc.conn.LocalAddr()
	}
	return nil
}

// RemoteAddr returns the remote address
// For server mode, this returns the last peer address
// For client mode, this returns the configured remote address
func (uc *UDPChannel) RemoteAddr() net.Addr {
	if uc.isServer {
		uc.peerLock.RLock()
		defer uc.peerLock.RUnlock()
		return uc.lastPeerAddr
	}
	return uc.remoteAddr
}
