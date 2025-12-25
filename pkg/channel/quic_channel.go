package channel

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/quic-go/quic-go"
)

// QUICChannel implements PhysicalChannel for QUIC connections
type QUICChannel struct {
	// Connection
	connection *quic.Conn
	stream     *quic.Stream
	connLock   sync.RWMutex
	streamLock sync.RWMutex

	// Configuration
	address        string
	isServer       bool
	listener       *quic.Listener
	reconnectDelay time.Duration
	readTimeout    time.Duration
	writeTimeout   time.Duration
	tlsConfig      *tls.Config

	// Connection state listener
	stateListener     ConnectionStateListener
	stateListenerLock sync.RWMutex

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

// QUICChannelConfig configures a QUIC channel
type QUICChannelConfig struct {
	Address        string        // "host:port" format
	IsServer       bool          // true = listen, false = connect
	ReconnectDelay time.Duration // Delay between reconnection attempts (client only)
	ReadTimeout    time.Duration // Read timeout (0 = no timeout)
	WriteTimeout   time.Duration // Write timeout (0 = no timeout)
	TLSConfig      *tls.Config   // Optional TLS config (if nil, will generate self-signed cert)
}

// NewQUICChannel creates a new QUIC channel
func NewQUICChannel(config QUICChannelConfig) (*QUICChannel, error) {
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

	// Generate TLS config if not provided
	tlsConfig := config.TLSConfig
	if tlsConfig == nil {
		var err error
		tlsConfig, err = generateTLSConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to generate TLS config: %w", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	qc := &QUICChannel{
		address:        config.Address,
		isServer:       config.IsServer,
		reconnectDelay: config.ReconnectDelay,
		readTimeout:    config.ReadTimeout,
		writeTimeout:   config.WriteTimeout,
		tlsConfig:      tlsConfig,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Initialize connection
	if config.IsServer {
		if err := qc.startServer(); err != nil {
			cancel()
			return nil, err
		}
	} else {
		if err := qc.connect(); err != nil {
			cancel()
			return nil, err
		}
	}

	return qc, nil
}

// generateTLSConfig generates a self-signed certificate for QUIC
func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{tlsCert},
		NextProtos:         []string{"dnp3-quic"},
		InsecureSkipVerify: true, // For self-signed certs
	}, nil
}

// startServer starts listening for incoming QUIC connections
func (qc *QUICChannel) startServer() error {
	udpAddr, err := net.ResolveUDPAddr("udp", qc.address)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address %s: %w", qc.address, err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", qc.address, err)
	}

	listener, err := quic.Listen(udpConn, qc.tlsConfig, nil)
	if err != nil {
		udpConn.Close()
		return fmt.Errorf("failed to create QUIC listener: %w", err)
	}

	qc.listener = listener

	// Accept connections in background
	qc.wg.Add(1)
	go qc.acceptLoop()

	return nil
}

// acceptLoop accepts incoming QUIC connections
func (qc *QUICChannel) acceptLoop() {
	defer qc.wg.Done()

	for {
		select {
		case <-qc.ctx.Done():
			return
		default:
		}

		conn, err := qc.listener.Accept(qc.ctx)
		if err != nil {
			if qc.closed.Load() {
				return
			}
			// Log error but continue accepting
			continue
		}

		// Close existing connection if any
		qc.connLock.Lock()
		hadConnection := qc.connection != nil
		if qc.connection != nil {
			qc.connection.CloseWithError(0, "new connection")
			qc.stats.disconnects.Add(1)
		}
		qc.connection = conn
		qc.stats.connects.Add(1)
		qc.connLock.Unlock()

		// Accept the first stream
		qc.wg.Add(1)
		go qc.acceptStream(conn, hadConnection)
	}
}

// acceptStream accepts a stream from the connection
func (qc *QUICChannel) acceptStream(conn *quic.Conn, hadConnection bool) {
	defer qc.wg.Done()

	stream, err := conn.AcceptStream(qc.ctx)
	if err != nil {
		return
	}

	qc.streamLock.Lock()
	if qc.stream != nil {
		qc.stream.Close()
	}
	qc.stream = stream
	qc.streamLock.Unlock()

	// Notify connection state change
	if hadConnection {
		qc.notifyConnectionLost()
	}
	qc.notifyConnectionEstablished()
}

// connect establishes a QUIC connection to the remote server
func (qc *QUICChannel) connect() error {
	udpAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
	if err != nil {
		return fmt.Errorf("failed to resolve local UDP address: %w", err)
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("failed to create UDP socket: %w", err)
	}

	// Resolve the remote address
	remoteAddr, err := net.ResolveUDPAddr("udp", qc.address)
	if err != nil {
		udpConn.Close()
		return fmt.Errorf("failed to resolve remote address %s: %w", qc.address, err)
	}

	conn, err := quic.Dial(qc.ctx, udpConn, remoteAddr, qc.tlsConfig, nil)
	if err != nil {
		udpConn.Close()
		return fmt.Errorf("failed to connect to %s: %w", qc.address, err)
	}

	// Open a stream
	stream, err := conn.OpenStreamSync(qc.ctx)
	if err != nil {
		conn.CloseWithError(0, "failed to open stream")
		return fmt.Errorf("failed to open stream: %w", err)
	}

	qc.connLock.Lock()
	qc.connection = conn
	qc.stats.connects.Add(1)
	qc.connLock.Unlock()

	qc.streamLock.Lock()
	qc.stream = stream
	qc.streamLock.Unlock()

	// Notify connection established
	qc.notifyConnectionEstablished()

	// Start reconnection handler for clients
	qc.wg.Add(1)
	go qc.reconnectLoop()

	return nil
}

// reconnectLoop handles automatic reconnection for client mode
func (qc *QUICChannel) reconnectLoop() {
	defer qc.wg.Done()

	for {
		select {
		case <-qc.ctx.Done():
			return
		case <-time.After(1 * time.Second):
			// Check if connection is alive
			qc.connLock.RLock()
			conn := qc.connection
			qc.connLock.RUnlock()

			if conn == nil || conn.Context().Err() != nil {
				// Connection is dead, wait for reconnect delay
				select {
				case <-qc.ctx.Done():
					return
				case <-time.After(qc.reconnectDelay):
				}

				// Try to reconnect
				udpAddr, err := net.ResolveUDPAddr("udp", "0.0.0.0:0")
				if err != nil {
					continue
				}

				udpConn, err := net.ListenUDP("udp", udpAddr)
				if err != nil {
					continue
				}

				remoteAddr, err := net.ResolveUDPAddr("udp", qc.address)
				if err != nil {
					udpConn.Close()
					continue
				}

				newConn, err := quic.Dial(qc.ctx, udpConn, remoteAddr, qc.tlsConfig, nil)
				if err == nil {
					// Open a stream
					stream, err := newConn.OpenStreamSync(qc.ctx)
					if err == nil {
						qc.connLock.Lock()
						if qc.connection != nil {
							qc.connection.CloseWithError(0, "reconnecting")
						}
						qc.connection = newConn
						qc.stats.connects.Add(1)
						qc.connLock.Unlock()

						qc.streamLock.Lock()
						if qc.stream != nil {
							qc.stream.Close()
						}
						qc.stream = stream
						qc.streamLock.Unlock()

						// Notify connection re-established
						qc.notifyConnectionEstablished()
					} else {
						newConn.CloseWithError(0, "failed to open stream")
					}
				}
			}
		}
	}
}

// Read implements PhysicalChannel.Read
func (qc *QUICChannel) Read(ctx context.Context) ([]byte, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-qc.ctx.Done():
			return nil, fmt.Errorf("channel closed")
		default:
		}

		// Wait for stream if not available
		var stream *quic.Stream
		for {
			qc.streamLock.RLock()
			stream = qc.stream
			qc.streamLock.RUnlock()

			if stream != nil {
				break
			}

			// No stream, wait for one
			select {
			case <-time.After(100 * time.Millisecond):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-qc.ctx.Done():
				return nil, fmt.Errorf("channel closed")
			}
		}

		// Set read deadline
		if qc.readTimeout > 0 {
			stream.SetReadDeadline(time.Now().Add(qc.readTimeout))
		}

		// Read DNP3 frame header (10 bytes minimum: 0x05 0x64 + length + 5 header fields + 2 CRC)
		header := make([]byte, 10)
		n, err := io.ReadFull(stream, header)
		if err != nil {
			qc.handleReadError(err)
			continue
		}

		// Debug: log what we read
		if n < 10 {
			qc.stats.readErrors.Add(1)
			continue
		}

		// Verify sync bytes
		if header[0] != 0x05 || header[1] != 0x64 {
			qc.stats.readErrors.Add(1)
			continue
		}

		// Get length field (bytes 2) - this is the length of control + addresses + data
		frameLength := int(header[2])
		if frameLength < 5 {
			qc.stats.readErrors.Add(1)
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
			_, err = io.ReadFull(stream, frame[10:])
			if err != nil {
				qc.handleReadError(err)
				continue
			}
		}

		qc.stats.bytesReceived.Add(uint64(len(frame)))

		// Debug logging
		if len(frame) < 10 {
			// This should never happen, but log it if it does
			qc.stats.readErrors.Add(1)
			continue
		}

		return frame, nil
	}
}

// Write implements PhysicalChannel.Write
func (qc *QUICChannel) Write(ctx context.Context, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-qc.ctx.Done():
		return fmt.Errorf("channel closed")
	default:
	}

	qc.streamLock.RLock()
	stream := qc.stream
	qc.streamLock.RUnlock()

	if stream == nil {
		qc.stats.writeErrors.Add(1)
		return fmt.Errorf("no stream")
	}

	// Set write deadline
	if qc.writeTimeout > 0 {
		stream.SetWriteDeadline(time.Now().Add(qc.writeTimeout))
	}

	_, err := stream.Write(data)
	if err != nil {
		qc.handleWriteError(err)
		return err
	}

	qc.stats.bytesSent.Add(uint64(len(data)))
	return nil
}

// Close implements PhysicalChannel.Close
func (qc *QUICChannel) Close() error {
	if !qc.closed.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	// Cancel context to stop all goroutines
	qc.cancel()

	// Close listener if server
	if qc.listener != nil {
		qc.listener.Close()
	}

	// Close stream
	qc.streamLock.Lock()
	if qc.stream != nil {
		qc.stream.Close()
		qc.stream = nil
	}
	qc.streamLock.Unlock()

	// Close connection
	qc.connLock.Lock()
	if qc.connection != nil {
		qc.connection.CloseWithError(0, "channel closed")
		qc.stats.disconnects.Add(1)
		qc.connection = nil
	}
	qc.connLock.Unlock()

	// Wait for goroutines to finish
	qc.wg.Wait()

	return nil
}

// Statistics implements PhysicalChannel.Statistics
func (qc *QUICChannel) Statistics() TransportStats {
	return TransportStats{
		BytesSent:     qc.stats.bytesSent.Load(),
		BytesReceived: qc.stats.bytesReceived.Load(),
		WriteErrors:   qc.stats.writeErrors.Load(),
		ReadErrors:    qc.stats.readErrors.Load(),
		Connects:      qc.stats.connects.Load(),
		Disconnects:   qc.stats.disconnects.Load(),
	}
}

// handleReadError handles read errors and manages connection state
func (qc *QUICChannel) handleReadError(err error) {
	qc.stats.readErrors.Add(1)

	qc.streamLock.Lock()
	if qc.stream != nil {
		qc.stream.Close()
		qc.stream = nil
	}
	qc.streamLock.Unlock()

	qc.connLock.Lock()
	hadConnection := qc.connection != nil
	if qc.connection != nil {
		qc.connection.CloseWithError(0, "read error")
		qc.stats.disconnects.Add(1)
		qc.connection = nil
	}
	qc.connLock.Unlock()

	// Notify connection lost
	if hadConnection {
		qc.notifyConnectionLost()
	}
}

// handleWriteError handles write errors and manages connection state
func (qc *QUICChannel) handleWriteError(err error) {
	qc.stats.writeErrors.Add(1)

	qc.streamLock.Lock()
	if qc.stream != nil {
		qc.stream.Close()
		qc.stream = nil
	}
	qc.streamLock.Unlock()

	qc.connLock.Lock()
	hadConnection := qc.connection != nil
	if qc.connection != nil {
		qc.connection.CloseWithError(0, "write error")
		qc.stats.disconnects.Add(1)
		qc.connection = nil
	}
	qc.connLock.Unlock()

	// Notify connection lost
	if hadConnection {
		qc.notifyConnectionLost()
	}
}

// IsConnected returns true if there is an active connection
func (qc *QUICChannel) IsConnected() bool {
	qc.connLock.RLock()
	defer qc.connLock.RUnlock()
	return qc.connection != nil && qc.connection.Context().Err() == nil
}

// LocalAddr returns the local address of the connection
func (qc *QUICChannel) LocalAddr() net.Addr {
	qc.connLock.RLock()
	defer qc.connLock.RUnlock()
	if qc.connection != nil {
		return qc.connection.LocalAddr()
	}
	return nil
}

// RemoteAddr returns the remote address of the connection
func (qc *QUICChannel) RemoteAddr() net.Addr {
	qc.connLock.RLock()
	defer qc.connLock.RUnlock()
	if qc.connection != nil {
		return qc.connection.RemoteAddr()
	}
	return nil
}

// SetConnectionStateListener sets a listener for connection state changes
func (qc *QUICChannel) SetConnectionStateListener(listener ConnectionStateListener) {
	qc.stateListenerLock.Lock()
	defer qc.stateListenerLock.Unlock()
	qc.stateListener = listener
}

// notifyConnectionEstablished notifies the listener that a connection was established
func (qc *QUICChannel) notifyConnectionEstablished() {
	qc.stateListenerLock.RLock()
	listener := qc.stateListener
	qc.stateListenerLock.RUnlock()

	if listener != nil {
		listener.OnConnectionEstablished()
	}
}

// notifyConnectionLost notifies the listener that a connection was lost
func (qc *QUICChannel) notifyConnectionLost() {
	qc.stateListenerLock.RLock()
	listener := qc.stateListener
	qc.stateListenerLock.RUnlock()

	if listener != nil {
		listener.OnConnectionLost()
	}
}
