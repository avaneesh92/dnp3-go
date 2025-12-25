# DNP3-Go

A pure Go implementation of the DNP3 (Distributed Network Protocol 3) protocol for SCADA and industrial control systems. Built with Go idioms, this library provides both master (client) and outstation (server) implementations with support for TCP, UDP, and custom transports.

## Features

- **Master (Client)** - Integrity scans, class scans, command execution (SELECT/OPERATE, DIRECT OPERATE), unsolicited response handling
- **Outstation (Server)** - Measurement database, event generation with deadbands, command processing, unsolicited responses
- **Complete Protocol Stack** - Link, Transport, and Application layers
- **Pluggable Transports** - TCP/UDP channels included, custom transports via simple interface
- **Concurrent & Thread-Safe** - Built with goroutines and channels for safe concurrent operations
- **Production Ready** - Working implementation with examples

## Installation

```bash
go get github.com/yourusername/dnp3-go
```

## Usage

### Master (Client) Example

```go
package main

import (
    "time"
    "github.com/yourusername/dnp3-go/pkg/dnp3"
    "github.com/yourusername/dnp3-go/pkg/channel"
)

func main() {
    // Create TCP channel (client connects to outstation)
    tcpConfig := channel.TCPChannelConfig{
        Address:        "127.0.0.1:20000",
        IsServer:       false,
        ReconnectDelay: 5 * time.Second,
    }
    tcpChannel, _ := channel.NewTCPChannel(tcpConfig)

    // Create manager and add channel
    manager := dnp3.NewManager()
    dnp3Channel, _ := manager.AddChannel("channel1", tcpChannel)

    // Configure and create master
    config := dnp3.DefaultMasterConfig()
    config.LocalAddress = 1
    config.RemoteAddress = 10

    master, _ := dnp3Channel.AddMaster(config, &MyCallbacks{})
    master.Enable()

    // Perform integrity scan
    master.ScanIntegrity()

    // Add periodic scanning
    master.AddIntegrityScan(60 * time.Second)
}
```

### Outstation (Server) Example

```go
package main

import (
    "time"
    "github.com/yourusername/dnp3-go/pkg/dnp3"
    "github.com/yourusername/dnp3-go/pkg/types"
    "github.com/yourusername/dnp3-go/pkg/channel"
)

func main() {
    // Create TCP channel (server listens for connections)
    tcpConfig := channel.TCPChannelConfig{
        Address:  "127.0.0.1:20000",
        IsServer: true,
    }
    tcpChannel, _ := channel.NewTCPChannel(tcpConfig)

    // Create manager and add channel
    manager := dnp3.NewManager()
    dnp3Channel, _ := manager.AddChannel("channel1", tcpChannel)

    // Configure database
    dbConfig := dnp3.DatabaseConfig{
        Binary:  make([]dnp3.BinaryPointConfig, 10),
        Analog:  make([]dnp3.AnalogPointConfig, 10),
        Counter: make([]dnp3.CounterPointConfig, 10),
    }

    // Configure and create outstation
    config := dnp3.DefaultOutstationConfig()
    config.LocalAddress = 10
    config.RemoteAddress = 1
    config.Database = dbConfig

    outstation, _ := dnp3Channel.AddOutstation(config, &MyCallbacks{})
    outstation.Enable()

    // Update measurements
    builder := dnp3.NewUpdateBuilder()
    builder.UpdateBinary(types.Binary{
        Value: true,
        Flags: types.FlagOnline,
        Time:  types.Now(),
    }, 0, dnp3.EventModeDetect)

    outstation.Apply(builder.Build())
}
```

See [examples/](examples/) for complete working examples.

## Supported Data Types & Operations

### Data Types
- Binary Input/Output (Groups 1, 2, 10)
- Analog Input/Output (Groups 30, 32, 40)
- Counter (Groups 20, 22)
- Frozen Counter (Groups 21, 23)
- Double-bit Binary (Groups 3, 4)

### Master Operations
- Integrity scans (Class 0)
- Class scans (Class 1, 2, 3)
- SELECT/OPERATE and DIRECT OPERATE commands
- Unsolicited response handling
- Automatic retry and timeout handling

### Outstation Operations
- Static data responses
- Event generation with deadbands
- Unsolicited responses
- Command processing (CROB, Analog Output)
- Time synchronization

## Transports

Built-in support for TCP and UDP. Custom transports can be added by implementing the `PhysicalChannel` interface.

**TCP Example:**
```go
tcpConfig := channel.TCPChannelConfig{
    Address:        "127.0.0.1:20000",
    IsServer:       false,  // true for server, false for client
    ReconnectDelay: 5 * time.Second,
}
tcpChannel, _ := channel.NewTCPChannel(tcpConfig)
```

**UDP Example:**
```go
udpConfig := channel.UDPChannelConfig{
    LocalAddress:  "0.0.0.0:20000",
    RemoteAddress: "127.0.0.1:20001",
}
udpChannel, _ := channel.NewUDPChannel(udpConfig)
```

## Testing

```bash
# Run all tests
go test ./pkg/...

# Run with coverage
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. ./pkg/...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License

Copyright (c) 2025

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Acknowledgments

This implementation is inspired by [OpenDNP3](https://github.com/dnp3/opendnp3) and follows the DNP3 IEEE-1815 standard.
