# QUIC Channel Implementation

This package provides a QUIC-based transport channel for DNP3 communication using the [quic-go](https://github.com/quic-go/quic-go) library.

## Overview

QUIC (Quick UDP Internet Connections) is a modern transport protocol that provides several advantages over traditional TCP:

- **Lower latency**: QUIC combines the connection establishment and encryption handshake
- **Better performance**: Built-in congestion control and loss recovery
- **Connection migration**: Supports IP address changes without connection loss
- **Multiplexing**: Multiple streams without head-of-line blocking
- **Security**: TLS 1.3 encryption built-in

## Features

- ✅ Server and client modes
- ✅ Automatic reconnection (client mode)
- ✅ Connection state notifications
- ✅ TLS encryption (self-signed certs by default)
- ✅ DNP3 frame parsing and handling
- ✅ Statistics tracking
- ✅ Configurable timeouts

## Usage

### Server (Outstation) Mode

```go
import (
    "time"
    "avaneesh/dnp3-go/pkg/channel"
)

// Create QUIC channel in server mode
quicChannel, err := channel.NewQUICChannel(channel.QUICChannelConfig{
    Address:        ":20000",           // Listen on port 20000
    IsServer:       true,               // Server mode
    ReconnectDelay: 5 * time.Second,    // N/A for server
    ReadTimeout:    30 * time.Second,   // Read timeout
    WriteTimeout:   10 * time.Second,   // Write timeout
    TLSConfig:      nil,                // Auto-generate self-signed cert
})
if err != nil {
    panic(err)
}
defer quicChannel.Close()

// Use with DNP3 channel
dnp3Channel := channel.New("my-channel", quicChannel, logger)
dnp3Channel.Open()
defer dnp3Channel.Close()
```

### Client (Master) Mode

```go
import (
    "time"
    "avaneesh/dnp3-go/pkg/channel"
)

// Create QUIC channel in client mode
quicChannel, err := channel.NewQUICChannel(channel.QUICChannelConfig{
    Address:        "localhost:20000",  // Connect to server
    IsServer:       false,              // Client mode
    ReconnectDelay: 5 * time.Second,    // Auto-reconnect delay
    ReadTimeout:    30 * time.Second,   // Read timeout
    WriteTimeout:   10 * time.Second,   // Write timeout
    TLSConfig:      nil,                // Auto-generate self-signed cert
})
if err != nil {
    panic(err)
}
defer quicChannel.Close()

// Use with DNP3 channel
dnp3Channel := channel.New("my-channel", quicChannel, logger)
dnp3Channel.Open()
defer dnp3Channel.Close()
```

### Custom TLS Configuration

You can provide your own TLS configuration:

```go
import (
    "crypto/tls"
    "avaneesh/dnp3-go/pkg/channel"
)

tlsConfig := &tls.Config{
    Certificates: []tls.Certificate{cert},
    NextProtos:   []string{"dnp3-quic"},
    // ... other TLS settings
}

quicChannel, err := channel.NewQUICChannel(channel.QUICChannelConfig{
    Address:   ":20000",
    IsServer:  true,
    TLSConfig: tlsConfig,
})
```

## Configuration

### QUICChannelConfig

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `Address` | `string` | Host:port to listen on (server) or connect to (client) | Required |
| `IsServer` | `bool` | true = server mode, false = client mode | Required |
| `ReconnectDelay` | `time.Duration` | Delay between reconnection attempts (client only) | 5s |
| `ReadTimeout` | `time.Duration` | Read timeout (0 = no timeout) | 30s |
| `WriteTimeout` | `time.Duration` | Write timeout (0 = no timeout) | 10s |
| `TLSConfig` | `*tls.Config` | Custom TLS configuration (nil = auto-generate) | nil |

## Architecture

### Connection Flow

**Server Mode:**
1. Listen for incoming QUIC connections on specified port
2. Accept new connections (closes existing connection if any)
3. Accept the first stream from the connection
4. Notify DNP3 layer of connection state changes
5. Read/write DNP3 frames on the stream

**Client Mode:**
1. Dial QUIC connection to remote server
2. Open a stream for communication
3. Start reconnection loop to handle disconnects
4. Read/write DNP3 frames on the stream
5. Auto-reconnect if connection is lost

### Thread Safety

- All public methods are thread-safe
- Uses separate locks for connection and stream access
- Supports concurrent reads and writes
- Statistics are updated atomically

### DNP3 Frame Handling

The QUIC channel handles DNP3 frames according to the protocol specification:

1. **Header (10 bytes)**:
   - Sync bytes: `0x05 0x64`
   - Length field (1 byte)
   - Control byte (1 byte)
   - Destination address (2 bytes)
   - Source address (2 bytes)
   - CRC (2 bytes)

2. **Data blocks**: 16-byte blocks with 2-byte CRC after each block

The Read() method automatically:
- Validates sync bytes
- Calculates frame length
- Reads complete frames including all CRC blocks
- Returns errors for malformed frames

## Statistics

The QUIC channel tracks the following statistics:

```go
stats := quicChannel.Statistics()

fmt.Printf("Bytes Sent: %d\n", stats.BytesSent)
fmt.Printf("Bytes Received: %d\n", stats.BytesReceived)
fmt.Printf("Write Errors: %d\n", stats.WriteErrors)
fmt.Printf("Read Errors: %d\n", stats.ReadErrors)
fmt.Printf("Connects: %d\n", stats.Connects)
fmt.Printf("Disconnects: %d\n", stats.Disconnects)
```

## Connection State Notifications

The QUIC channel implements the `ConnectionStateListener` interface:

```go
type ConnectionStateListener interface {
    OnConnectionEstablished()
    OnConnectionLost()
}
```

These callbacks are automatically triggered when:
- A new connection is established
- An existing connection is lost
- The DNP3 channel layer receives these notifications

## Error Handling

### Read Errors
- Closes the stream and connection
- Triggers `OnConnectionLost()` notification
- Client mode: Auto-reconnects after delay
- Server mode: Waits for new connection

### Write Errors
- Same behavior as read errors
- Connection is torn down and reconnection attempted

## Examples

See [examples/quic_example/main.go](../../examples/quic_example/main.go) for a complete working example.

## Requirements

- Go 1.19 or later
- github.com/quic-go/quic-go v0.58.0 or later

## Performance Considerations

- QUIC uses UDP, which may have better performance than TCP in some network conditions
- Built-in encryption adds minimal overhead compared to TLS over TCP
- Stream multiplexing allows future expansion to multiple DNP3 sessions over one connection
- Consider tuning `ReadTimeout` and `WriteTimeout` based on your network conditions

## Security

- TLS 1.3 encryption is mandatory in QUIC
- Default implementation uses self-signed certificates (for testing)
- For production, provide your own TLS certificates via `TLSConfig`
- Certificate validation is disabled by default (`InsecureSkipVerify: true`)

**⚠️ Warning**: The default self-signed certificate configuration is suitable for development and testing only. For production deployments, use proper certificates from a trusted CA.

## Limitations

- Currently supports single stream per connection
- Server mode accepts only one client at a time (new connections replace old ones)
- No built-in authentication beyond TLS certificates

## Future Enhancements

Possible improvements:
- [ ] Multi-client support in server mode
- [ ] Stream pooling for multiple concurrent DNP3 sessions
- [ ] Connection pooling
- [ ] Better certificate management
- [ ] QUIC configuration tuning (congestion control, etc.)
- [ ] Connection migration support

## License

Same as the parent dnp3-go project.
