# DNP3-Go

A pure Go implementation of the DNP3 (Distributed Network Protocol 3) protocol, translated from the OpenDNP3 C++ library with idiomatic Go patterns.

**🎉 Initial Release: Working master and outstation implementations with TCP/UDP channels!**

## Features

✅ **Master (Client) Implementation** - Full scanning operations, command execution, and measurement callbacks
✅ **Outstation (Server) Implementation** - Measurement database, event generation, and command handling
✅ **Complete Protocol Stack** - Link, Transport, and Application layers
✅ **Pluggable Transports** - Simple interface for TCP, Serial, UDP, or custom transports
✅ **Go Idioms** - Goroutines, channels, and clean interfaces (not a direct C++ port)
✅ **Thread-Safe** - Built for concurrent operations

## Project Status

🎉 **Working Release** - Master and Outstation implementations complete with TCP and UDP channels!

### Completed

- ✅ Core data types (measurements, commands, quality flags, timestamps)
- ✅ Link layer (framing, CRC-16, addressing)
- ✅ Transport layer (segmentation, reassembly)
- ✅ Application layer (APDU, object groups/variations, parsing)
- ✅ Channel abstraction with pluggable `PhysicalChannel` interface
- ✅ DNP3Manager and public API structure
- ✅ **TCP Channel** - Full client/server implementation with auto-reconnect
- ✅ **UDP Channel** - Datagram-based transport
- ✅ **Master implementation** - Scanning, commands, unsolicited responses, SOE handler
- ✅ **Outstation implementation** - Database, event generation, command processing
- ✅ **Working Examples** - Simple master/outstation demo over TCP

### In Progress

- ⏳ Additional protocol features and optimizations
- ⏳ Comprehensive test coverage
- ⏳ Documentation improvements

## Installation

```bash
go get avaneesh/dnp3-go
```

## Quick Start

### Working Example: Master and Outstation over TCP

The library includes complete working examples demonstrating master and outstation communication over TCP.

### Example: Master (TCP Client)

```go
package main

import (
    "fmt"
    "time"
    "avaneesh/dnp3-go/pkg/dnp3"
    "avaneesh/dnp3-go/pkg/channel"
)

func main() {
    // Create TCP channel (client connects to outstation)
    tcpConfig := channel.TCPChannelConfig{
        Address:        "127.0.0.1:20000",
        IsServer:       false,  // Client mode
        ReconnectDelay: 5 * time.Second,
        ReadTimeout:    30 * time.Second,
    }

    tcpChannel, _ := channel.NewTCPChannel(tcpConfig)

    // Create manager and add channel
    manager := dnp3.NewManager()
    dnp3Channel, _ := manager.AddChannel("channel1", tcpChannel)

    // Configure master
    config := dnp3.DefaultMasterConfig()
    config.LocalAddress = 1   // Master address
    config.RemoteAddress = 10  // Outstation address

    // Create master with callbacks
    master, _ := dnp3Channel.AddMaster(config, &MyCallbacks{})
    master.Enable()

    // Perform integrity scan
    master.ScanIntegrity()

    // Add periodic scanning
    master.AddIntegrityScan(60 * time.Second)
    master.AddClassScan(dnp3.Class1, 10 * time.Second)
}
```

See [examples/simple_master.go](examples/simple_master.go) for the complete working example.

### Example: Outstation (TCP Server)

```go
package main

import (
    "time"
    "avaneesh/dnp3-go/pkg/dnp3"
    "avaneesh/dnp3-go/pkg/types"
    "avaneesh/dnp3-go/pkg/channel"
)

func main() {
    // Create TCP channel (server listens for connections)
    tcpConfig := channel.TCPChannelConfig{
        Address:  "127.0.0.1:20000",
        IsServer: true,  // Server mode
    }

    tcpChannel, _ := channel.NewTCPChannel(tcpConfig)

    // Create manager and add channel
    manager := dnp3.NewManager()
    dnp3Channel, _ := manager.AddChannel("channel1", tcpChannel)

    // Configure database with 10 binary, analog, and counter points
    dbConfig := dnp3.DatabaseConfig{
        Binary:  make([]dnp3.BinaryPointConfig, 10),
        Analog:  make([]dnp3.AnalogPointConfig, 10),
        Counter: make([]dnp3.CounterPointConfig, 10),
    }

    // Configure outstation
    config := dnp3.DefaultOutstationConfig()
    config.LocalAddress = 10   // Outstation address
    config.RemoteAddress = 1   // Master address
    config.Database = dbConfig

    // Create outstation
    outstation, _ := dnp3Channel.AddOutstation(config, &MyCallbacks{})
    outstation.Enable()

    // Update measurements atomically
    builder := dnp3.NewUpdateBuilder()
    builder.UpdateBinary(types.Binary{
        Value: true,
        Flags: types.FlagOnline,
        Time:  types.Now(),
    }, 0, dnp3.EventModeDetect)

    builder.UpdateAnalog(types.Analog{
        Value: 123.45,
        Flags: types.FlagOnline,
        Time:  types.Now(),
    }, 0, dnp3.EventModeDetect)

    outstation.Apply(builder.Build())
}
```

See [examples/simple_client.go](examples/simple_client.go) for the complete working outstation example.

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

## Running the Examples

### Start the Outstation (Server)

```bash
cd examples
go run simple_client.go
```

The outstation will listen on `127.0.0.1:20000` and simulate changing sensor values.

### Start the Master (Client)

In a separate terminal:

```bash
cd examples
go run simple_master.go
```

The master will connect to the outstation and display:
- Binary inputs (breaker states)
- Analog inputs (temperature, voltage, current, power sensors)
- Counter values (energy meters)

## Pluggable Transports

The library includes built-in TCP and UDP channels. You can also implement custom transports by implementing the `PhysicalChannel` interface:

```go
type PhysicalChannel interface {
    Read(ctx context.Context) ([]byte, error)
    Write(ctx context.Context, data []byte) error
    Close() error
    Statistics() TransportStats
}
```

### Built-in Transports

**TCP Channel** - Full duplex TCP with automatic reconnection:
```go
tcpConfig := channel.TCPChannelConfig{
    Address:        "127.0.0.1:20000",
    IsServer:       false,  // true for server, false for client
    ReconnectDelay: 5 * time.Second,
}
tcpChannel, _ := channel.NewTCPChannel(tcpConfig)
```

**UDP Channel** - Datagram-based communication:
```go
udpConfig := channel.UDPChannelConfig{
    LocalAddress:  "0.0.0.0:20000",
    RemoteAddress: "127.0.0.1:20001",
}
udpChannel, _ := channel.NewUDPChannel(udpConfig)
```

See [examples/simple_example.go](examples/simple_example.go) for custom channel implementation examples.

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

## Build Instructions

```bash
# Clone the repository
git clone <repository-url>
cd dnp3-go

# Build the library
go build ./...

# Run tests
go test ./pkg/...

# Run tests with coverage
go test -cover ./pkg/...

# Run examples
cd examples

# Terminal 1 - Start outstation (server)
go run simple_client.go

# Terminal 2 - Start master (client)
go run simple_master.go
```

## Testing

The library includes comprehensive test coverage for critical protocol components:

- **Link Layer**: CRC-16 calculations, frame serialization/parsing
- **Types**: Quality flags, measurements, timestamps
- **Test Coverage**: ~90% for tested components
- **Benchmarks**: Performance testing for critical paths

See [TESTING_SUMMARY.md](TESTING_SUMMARY.md) for details.

```bash
# Run all tests
go test ./pkg/...

# Run with coverage report
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. ./pkg/...
```

## Development Roadmap

### Phase 1-5: Complete ✅
- ✅ Core types, link layer, transport, application layers
- ✅ Pluggable transport abstraction
- ✅ TCP and UDP channels with auto-reconnect
- ✅ Full master implementation (scanning, commands, unsolicited)
- ✅ Full outstation implementation (database, events)
- ✅ Working examples demonstrating master/outstation communication

### Phase 6: In Progress ⏳
- ⏳ Comprehensive test coverage
- ⏳ Performance optimizations
- ⏳ Additional transport implementations (Serial)
- ⏳ Advanced protocol features

## Features Implemented

### Master (Client)
- ✅ Integrity scans (Class 0)
- ✅ Class scans (Class 1, 2, 3)
- ✅ Periodic scanning
- ✅ DIRECT OPERATE commands
- ✅ SELECT/OPERATE commands
- ✅ Unsolicited response handling
- ✅ IIN flag processing
- ✅ Response timeout handling
- ✅ Automatic retry logic

### Outstation (Server)
- ✅ Static data responses
- ✅ Event generation with deadbands
- ✅ Unsolicited responses
- ✅ Command processing (CROB, Analog Output)
- ✅ Database management
- ✅ Event buffering
- ✅ Class assignment
- ✅ Time synchronization

### Channels
- ✅ TCP client/server with auto-reconnect
- ✅ UDP datagram support
- ✅ Connection statistics
- ✅ Pluggable architecture for custom transports

## References

This implementation is based on:
- [OpenDNP3](https://github.com/dnp3/opendnp3) - Original C++ implementation
- [DNP3 IEEE-1815 Standard](https://en.wikipedia.org/wiki/DNP3)
- [OpenDNP3 Documentation](https://dnp3.github.io/)

## License

[Your License Here]

## Contributing

Contributions welcome! This project is actively under development.
