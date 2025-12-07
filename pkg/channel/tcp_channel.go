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

// TCPChannel implements PhysicalChannel for TCP connections
type TCPChannel struct {
	// Connection
	conn     net.Conn
	connLock sync.RWMutex

	// Configuration
	address        string
	isServer       bool
	listener       net.Listener
	reconnectDelay time.Duration
	readTimeout    time.Duration
	writeTimeout   time.Duration

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
	wg     sync.WaitGroup
	closed atomic.Bool
}

// TCPChannelConfig configures a TCP channel
type TCPChannelConfig struct {
	Address        string        // "host:port" format
	IsServer       bool          // true = listen, false = connect
	ReconnectDelay time.Duration // Delay between reconnection attempts (client only)
	ReadTimeout    time.Duration // Read timeout (0 = no timeout)
	WriteTimeout   time.Duration // Write timeout (0 = no timeout)
}

// NewTCPChannel creates a new TCP channel
func NewTCPChannel(config TCPChannelConfig) (*TCPChannel, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("address is required")
	}

	// Set defaults
	if config.ReconnectDelay == 0 {
		config.ReconnectDelay = 5 * time.Second
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 30 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 10 * time.Second
	}

	ctx, cancel := context.WithCancel(context.Background())

	tc := &TCPChannel{
		address:        config.Address,
		isServer:       config.IsServer,
		reconnectDelay: config.ReconnectDelay,
		readTimeout:    config.ReadTimeout,
		writeTimeout:   config.WriteTimeout,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Initialize connection
	if config.IsServer {
		if err := tc.startServer(); err != nil {
			cancel()
			return nil, err
		}
	} else {
		if err := tc.connect(); err != nil {
			cancel()
			return nil, err
		}
	}

	return tc, nil
}

// startServer starts listening for incoming connections
func (tc *TCPChannel) startServer() error {
	listener, err := net.Listen("tcp", tc.address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", tc.address, err)
	}

	tc.listener = listener

	// Accept connections in background
	tc.wg.Add(1)
	go tc.acceptLoop()

	return nil
}

// acceptLoop accepts incoming connections
func (tc *TCPChannel) acceptLoop() {
	defer tc.wg.Done()

	for {
		select {
		case <-tc.ctx.Done():
			return
		default:
		}

		// Set accept deadline to allow periodic context checks
		if tcpListener, ok := tc.listener.(*net.TCPListener); ok {
			tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
		}

		conn, err := tc.listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout is expected, continue loop
				continue
			}
			if tc.closed.Load() {
				return
			}
			// Log error but continue accepting
			continue
		}

		// Close existing connection if any
		tc.connLock.Lock()
		if tc.conn != nil {
			tc.conn.Close()
			tc.stats.disconnects.Add(1)
		}
		tc.conn = conn
		tc.stats.connects.Add(1)
		tc.connLock.Unlock()
	}
}

// connect establishes a connection to the remote server
func (tc *TCPChannel) connect() error {
	conn, err := net.DialTimeout("tcp", tc.address, 10*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", tc.address, err)
	}

	tc.connLock.Lock()
	tc.conn = conn
	tc.stats.connects.Add(1)
	tc.connLock.Unlock()

	// Start reconnection handler for clients
	tc.wg.Add(1)
	go tc.reconnectLoop()

	return nil
}

// reconnectLoop handles automatic reconnection for client mode
func (tc *TCPChannel) reconnectLoop() {
	defer tc.wg.Done()

	for {
		select {
		case <-tc.ctx.Done():
			return
		case <-time.After(1 * time.Second):
			// Check if connection is alive
			tc.connLock.RLock()
			conn := tc.conn
			tc.connLock.RUnlock()

			if conn == nil {
				// Try to reconnect
				newConn, err := net.DialTimeout("tcp", tc.address, 10*time.Second)
				if err == nil {
					tc.connLock.Lock()
					tc.conn = newConn
					tc.stats.connects.Add(1)
					tc.connLock.Unlock()
				}
			}
		}
	}
}

// Read implements PhysicalChannel.Read
func (tc *TCPChannel) Read(ctx context.Context) ([]byte, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-tc.ctx.Done():
			return nil, fmt.Errorf("channel closed")
		default:
		}

		// Wait for connection if not available
		var conn net.Conn
		for {
			tc.connLock.RLock()
			conn = tc.conn
			tc.connLock.RUnlock()

			if conn != nil {
				break
			}

			// No connection, wait for one
			select {
			case <-time.After(100 * time.Millisecond):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-tc.ctx.Done():
				return nil, fmt.Errorf("channel closed X")
			}
		}

		// Set read deadline
		if tc.readTimeout > 0 {
			conn.SetReadDeadline(time.Now().Add(tc.readTimeout))
		}

		// Read DNP3 frame header (10 bytes minimum: 0x05 0x64 + length + 5 header fields + 2 CRC)
		header := make([]byte, 10)
		n, err := io.ReadFull(conn, header)
		if err != nil {
			tc.handleReadError(err)
			continue
		}

		// Debug: log what we read
		if n < 10 {
			tc.stats.readErrors.Add(1)
			continue
		}

		// Verify sync bytes
		if header[0] != 0x05 || header[1] != 0x64 {
			tc.stats.readErrors.Add(1)
			continue
		}

		// Get length field (bytes 2) - this is the length of control + addresses + data
		frameLength := int(header[2])
		if frameLength < 5 {
			tc.stats.readErrors.Add(1)
			continue
		}

		// Calculate user data length
		dataLen := frameLength - 5 // Subtract control (1) + dest (2) + source (2)

		// Calculate total additional bytes to read beyond the 10-byte header
		// DNP3 data is in 16-byte blocks with 2-byte CRC after each block
		var additionalBytes int
		if dataLen > 0 {
			numBlocks := (dataLen + 15) / 16
			additionalBytes = dataLen + (numBlocks * 2)
		}

		// Allocate buffer for complete frame
		frame := make([]byte, 10+additionalBytes)
		copy(frame[0:10], header)

		// Read remaining data if any
		if additionalBytes > 0 {
			_, err = io.ReadFull(conn, frame[10:])
			if err != nil {
				tc.handleReadError(err)
				continue
			}
		}

		tc.stats.bytesReceived.Add(uint64(len(frame)))

		// Debug logging
		if len(frame) < 10 {
			// This should never happen, but log it if it does
			tc.stats.readErrors.Add(1)
			continue
		}

		return frame, nil
	}
}

// Write implements PhysicalChannel.Write
func (tc *TCPChannel) Write(ctx context.Context, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-tc.ctx.Done():
		return fmt.Errorf("channel closed")
	default:
	}

	tc.connLock.RLock()
	conn := tc.conn
	tc.connLock.RUnlock()

	if conn == nil {
		tc.stats.writeErrors.Add(1)
		return fmt.Errorf("no connection")
	}

	// Set write deadline
	if tc.writeTimeout > 0 {
		conn.SetWriteDeadline(time.Now().Add(tc.writeTimeout))
	}

	_, err := conn.Write(data)
	if err != nil {
		tc.handleWriteError(err)
		return err
	}

	tc.stats.bytesSent.Add(uint64(len(data)))
	return nil
}

// Close implements PhysicalChannel.Close
func (tc *TCPChannel) Close() error {
	if !tc.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	// Cancel context to stop all goroutines
	tc.cancel()

	// Close listener if server
	if tc.listener != nil {
		tc.listener.Close()
	}

	// Close connection
	tc.connLock.Lock()
	if tc.conn != nil {
		tc.conn.Close()
		tc.stats.disconnects.Add(1)
		tc.conn = nil
	}
	tc.connLock.Unlock()

	// Wait for goroutines to finish
	tc.wg.Wait()

	return nil
}

// Statistics implements PhysicalChannel.Statistics
func (tc *TCPChannel) Statistics() TransportStats {
	return TransportStats{
		BytesSent:     tc.stats.bytesSent.Load(),
		BytesReceived: tc.stats.bytesReceived.Load(),
		WriteErrors:   tc.stats.writeErrors.Load(),
		ReadErrors:    tc.stats.readErrors.Load(),
		Connects:      tc.stats.connects.Load(),
		Disconnects:   tc.stats.disconnects.Load(),
	}
}

// handleReadError handles read errors and manages connection state
func (tc *TCPChannel) handleReadError(err error) {
	tc.stats.readErrors.Add(1)

	tc.connLock.Lock()
	defer tc.connLock.Unlock()

	if tc.conn != nil {
		tc.conn.Close()
		tc.stats.disconnects.Add(1)
		tc.conn = nil
	}
}

// handleWriteError handles write errors and manages connection state
func (tc *TCPChannel) handleWriteError(err error) {
	tc.stats.writeErrors.Add(1)

	tc.connLock.Lock()
	defer tc.connLock.Unlock()

	if tc.conn != nil {
		tc.conn.Close()
		tc.stats.disconnects.Add(1)
		tc.conn = nil
	}
}

// IsConnected returns true if there is an active connection
func (tc *TCPChannel) IsConnected() bool {
	tc.connLock.RLock()
	defer tc.connLock.RUnlock()
	return tc.conn != nil
}

// LocalAddr returns the local address of the connection
func (tc *TCPChannel) LocalAddr() net.Addr {
	tc.connLock.RLock()
	defer tc.connLock.RUnlock()
	if tc.conn != nil {
		return tc.conn.LocalAddr()
	}
	return nil
}

// RemoteAddr returns the remote address of the connection
func (tc *TCPChannel) RemoteAddr() net.Addr {
	tc.connLock.RLock()
	defer tc.connLock.RUnlock()
	if tc.conn != nil {
		return tc.conn.RemoteAddr()
	}
	return nil
}
