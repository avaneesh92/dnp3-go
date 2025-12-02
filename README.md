# DNP3-Go

A pure Go implementation of the DNP3 (Distributed Network Protocol 3) protocol, translated from the OpenDNP3 C++ library with idiomatic Go patterns.

## Features

✅ **Master (Client) Implementation** - Full scanning operations, command execution, and measurement callbacks
✅ **Outstation (Server) Implementation** - Measurement database, event generation, and command handling
✅ **Complete Protocol Stack** - Link, Transport, and Application layers
✅ **Pluggable Transports** - Simple interface for TCP, Serial, UDP, or custom transports
✅ **Go Idioms** - Goroutines, channels, and clean interfaces (not a direct C++ port)
✅ **Thread-Safe** - Built for concurrent operations

## Project Status

🚧 **Phase 3 Complete** - Channel infrastructure with pluggable transports
⏳ **Phase 4-6 In Progress** - Master, Outstation, and examples coming soon

### Completed

- ✅ Core data types (measurements, commands, quality flags, timestamps)
- ✅ Link layer (framing, CRC-16, addressing)
- ✅ Transport layer (segmentation, reassembly)
- ✅ Application layer (APDU, object groups/variations, parsing)
- ✅ Channel abstraction with pluggable `PhysicalChannel` interface
- ✅ DNP3Manager and public API structure

### In Progress

- ⏳ Master implementation (scanning, commands, SOE handler)
- ⏳ Outstation implementation (database, events, command processing)
- ⏳ Examples and integration tests

## Installation

```bash
go get avaneesh/dnp3-go
```

## Quick Start

### Pluggable Channel Interface

The key innovation of this library is the `PhysicalChannel` interface - implement just 4 methods to plug in any transport:

```go
type PhysicalChannel interface {
    Read(ctx context.Context) ([]byte, error)
    Write(ctx context.Context, data []byte) error
    Close() error
    Statistics() TransportStats
}
```

### Example: Master (Coming in Phase 4)

```go
package main

import (
    "time"
    "avaneesh/dnp3-go/pkg/dnp3"
)

func main() {
    // Create manager
    manager := dnp3.NewManager()
    defer manager.Shutdown()

    // User provides custom transport (TCP, Serial, etc.)
    physicalChannel := NewMyTCPChannel("127.0.0.1:20000")

    // Create channel
    channel, _ := manager.AddChannel("channel1", physicalChannel)

    // Add master with callbacks
    config := dnp3.MasterConfig{
        LocalAddress:    1,
        RemoteAddress:   10,
        ResponseTimeout: 5 * time.Second,
    }

    master, _ := channel.AddMaster(config, &MyCallbacks{})
    master.Enable()

    // Perform operations
    master.AddIntegrityScan(60 * time.Second)
    master.DirectOperate(commands)
}
```

### Example: Outstation (Coming in Phase 5)

```go
// Create outstation
config := dnp3.OutstationConfig{
    LocalAddress:  10,
    RemoteAddress: 1,
    Database:      dbConfig,
}

outstation, _ := channel.AddOutstation(config, &MyCallbacks{})
outstation.Enable()

// Update measurements atomically
builder := dnp3.NewUpdateBuilder()
builder.UpdateBinary(types.Binary{
    Value: true,
    Flags: types.FlagOnline,
    Time:  types.Now(),
}, 0, dnp3.EventModeDetect)

outstation.Apply(builder.Build())
```

## Architecture

```
DNP3Manager
  └─> Channels (pluggable transports)
      ├─> Read Loop (goroutine)
      ├─> Write Loop (goroutine)
      └─> Sessions
          ├─> Master (client)
          │   ├─> Task Queue
          │   ├─> Command Processor
          │   └─> SOE Handler
          └─> Outstation (server)
              ├─> Database
              ├─> Event Buffer
              └─> Command Handler
```

## Pluggable Transports

See [examples/custom_channel/mock_channel.go](examples/custom_channel/mock_channel.go) for implementation examples.

### TCP Transport Example

```go
type TCPChannel struct {
    conn net.Conn
}

func (t *TCPChannel) Read(ctx context.Context) ([]byte, error) {
    // Read from TCP connection
}

func (t *TCPChannel) Write(ctx context.Context, data []byte) error {
    // Write to TCP connection
}
```

### Serial Transport Example

```go
type SerialChannel struct {
    port *serial.Port
}

func (s *SerialChannel) Read(ctx context.Context) ([]byte, error) {
    // Read from serial port
}

func (s *SerialChannel) Write(ctx context.Context, data []byte) error {
    // Write to serial port
}
```

## DNP3 Protocol Support

### Data Types

- Binary Input/Output (Groups 1, 2, 10)
- Analog Input/Output (Groups 30, 32, 40)
- Counter (Groups 20, 22)
- Frozen Counter (Groups 21, 23)
- Double-bit Binary (Groups 3, 4)

### Operations

**Master:**
- Integrity scans (Class 0)
- Class scans (Class 1, 2, 3)
- Range scans
- SELECT/OPERATE commands
- DIRECT OPERATE commands
- Unsolicited response handling

**Outstation:**
- Static data responses
- Event generation with deadbands
- Unsolicited responses
- Command processing (CROB, Analog Output)
- Time synchronization

## Concurrency Model

- Each channel runs in dedicated goroutines (read + write loops)
- Sessions are serialized within their channel
- User callbacks run in separate goroutines (non-blocking)
- Thread-safe operations throughout

## Development Roadmap

### Phase 1: Foundation ✅
- Core types, link layer CRC, basic parsing

### Phase 2: Protocol Stack ✅
- Complete 3-layer DNP3 protocol stack

### Phase 3: Channel Infrastructure ✅
- Pluggable transport abstraction

### Phase 4: Master Implementation ⏳
- Full master with scanning and commands

### Phase 5: Outstation Implementation ⏳
- Full outstation with database

### Phase 6: Testing & Examples ⏳
- Comprehensive tests and documentation

## References

This implementation is based on:
- [OpenDNP3](https://github.com/dnp3/opendnp3) - Original C++ implementation
- [DNP3 IEEE-1815 Standard](https://en.wikipedia.org/wiki/DNP3)
- [OpenDNP3 Documentation](https://dnp3.github.io/)

## License

[Your License Here]

## Contributing

Contributions welcome! This project is actively under development.

---

**Note:** This library is in active development. Phases 4-6 (Master, Outstation, Examples) are in progress. The pluggable channel interface and protocol stack are complete and ready to use.
