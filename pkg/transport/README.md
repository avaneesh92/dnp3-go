# DNP3 Transport Layer

This package implements the DNP3 transport layer for both Master and Outstation stations.

## Overview

The transport layer sits between the Data Link Layer and Application Layer, handling:
- **Segmentation**: Breaking large APDUs into 248-byte fragments
- **Reassembly**: Reconstructing complete APDUs from fragments
- **Sequence Management**: Tracking fragment order with 6-bit sequence numbers
- **Error Detection**: Identifying sequence gaps and reassembly timeouts

## Components

### Master Transport ([master_transport.go](master_transport.go))
Manages transport layer for DNP3 master stations communicating with multiple outstations.

**Key Features**:
- Independent state tracking per outstation
- Concurrent communication with multiple devices
- Per-outstation statistics and error tracking
- Configurable reassembly timeouts

```go
config := transport.DefaultTransportConfig()
master := transport.NewMasterTransport(config)

// Send to outstation 10
segments := master.Send(10, apdu)

// Receive from outstation 10
completeAPDU, err := master.Receive(10, tpdu)
```

### Outstation Transport ([outstation_transport.go](outstation_transport.go))
Manages transport layer for DNP3 outstations communicating with a master.

**Key Features**:
- Simplified state for single master
- Supports both solicited and unsolicited responses
- Same reassembly and timeout mechanisms as master

```go
config := transport.DefaultTransportConfig()
outstation := transport.NewOutstationTransport(config)

// Send response or unsolicited message
segments := outstation.Send(apdu)

// Receive request from master
completeAPDU, err := outstation.Receive(tpdu)
```

### Segmentation ([segment.go](segment.go))
Handles breaking APDUs into transport fragments.

- Maximum fragment size: 248 bytes
- Automatic FIR/FIN/SEQ header generation
- Sequence wrap-around at 63→0

### Reassembly ([reassembly.go](reassembly.go))
Handles reconstructing APDUs from fragments.

- Sequence validation
- Buffer management (up to 2048 bytes)
- Error detection and recovery

### Configuration ([config.go](config.go))
Transport layer configuration options.

```go
config := transport.TransportConfig{
    ReassemblyTimeout: 120 * time.Second,
    MaxReassemblySize: 2048,
    EnableStatistics:  true,
}
```

### Statistics ([statistics.go](statistics.go))
Thread-safe statistics tracking.

```go
stats := master.GetStats(outstationAddr)
fmt.Printf("TX: %d messages, %d fragments\n",
    stats.GetTxMessages(), stats.GetTxFragments())
fmt.Printf("Errors: %d sequence, %d timeouts\n",
    stats.GetSequenceErrors(), stats.GetTimeoutErrors())
```

## Transport Header Format

The transport header is a single byte with the following structure:

```
Bit 7      Bit 6      Bits 5-0
┌──────┬──────────┬──────────────┐
│ FIN  │   FIR    │   SEQUENCE   │
└──────┴──────────┴──────────────┘
```

- **FIN** (Bit 7): Final fragment indicator
- **FIR** (Bit 6): First fragment indicator
- **SEQ** (Bits 5-0): Sequence number (0-63)

### Fragment Types

| FIR | FIN | Type | Hex Range | Description |
|-----|-----|------|-----------|-------------|
| 1 | 1 | Single | 0xC0-0xFF | Complete message in one fragment |
| 1 | 0 | First | 0x40-0x7F | First fragment of multi-fragment message |
| 0 | 0 | Middle | 0x00-0x3F | Middle fragment |
| 0 | 1 | Final | 0x80-0xBF | Final fragment of multi-fragment message |

## Usage Examples

### Sending a Large Message (Master)

```go
// Application layer provides large APDU (e.g., 600 bytes)
apdu := make([]byte, 600)
// ... populate APDU ...

// Transport layer segments it automatically
segments := master.Send(outstationAddr, apdu)

// Results in 3 segments:
// Segment 1: [0x40][248 bytes data]  (FIR=1, FIN=0, SEQ=0)
// Segment 2: [0x01][248 bytes data]  (FIR=0, FIN=0, SEQ=1)
// Segment 3: [0x82][104 bytes data]  (FIR=0, FIN=1, SEQ=2)

// Send each segment via link layer
for _, tpdu := range segments {
    linkLayer.SendUserData(outstationAddr, tpdu)
}
```

### Receiving a Multi-Fragment Message (Outstation)

```go
// Receive fragment 1 from link layer
tpdu1 := []byte{0x40, /* 248 bytes data */}
apdu, err := outstation.Receive(tpdu1)
// Returns: apdu=nil (incomplete), err=nil

// Receive fragment 2
tpdu2 := []byte{0x01, /* 248 bytes data */}
apdu, err = outstation.Receive(tpdu2)
// Returns: apdu=nil (incomplete), err=nil

// Receive fragment 3 (final)
tpdu3 := []byte{0x82, /* 104 bytes data */}
apdu, err = outstation.Receive(tpdu3)
// Returns: apdu=[600 bytes complete APDU], err=nil

// Process complete APDU
if apdu != nil {
    applicationLayer.ProcessRequest(apdu)
}
```

### Handling Sequence Errors

```go
// Receive fragment with SEQ=0
tpdu1 := []byte{0x40, 0x01, 0x02}
apdu, _ := master.Receive(10, tpdu1)
// apdu=nil, waiting for more fragments

// Receive fragment with wrong sequence (SEQ=5, expected 1)
tpdu2 := []byte{0x05, 0x03, 0x04}
apdu, _ = master.Receive(10, tpdu2)
// apdu=nil, reassembly discarded due to sequence error

// Check statistics
stats := master.GetStats(10)
if stats.GetSequenceErrors() > 0 {
    log.Printf("Sequence errors detected")
}
```

### Monitoring Timeouts

```go
// Fragment 1 received, starts 120-second timer
tpdu1 := []byte{0x40, 0x01, 0x02}
outstation.Receive(tpdu1)

// Check if reassembly in progress
if outstation.IsReassembling() {
    log.Println("Waiting for more fragments...")
}

// If no more fragments arrive within 120 seconds,
// reassembly is automatically discarded

// Later, check timeout statistics
stats := outstation.GetStats()
if stats.GetTimeoutErrors() > 0 {
    log.Printf("Reassembly timeouts: %d", stats.GetTimeoutErrors())
}
```

## Error Handling

The transport layer handles errors according to DNP3 specification:

### Sequence Errors
- **Detection**: Received SEQ ≠ expected SEQ
- **Action**: Discard buffered fragments, wait for new FIR
- **Recovery**: Next FIR starts fresh reassembly
- **Tracking**: Increments SequenceErrors counter

### Reassembly Timeouts
- **Detection**: No fragments for 120 seconds (default)
- **Action**: Discard incomplete message, free buffer
- **Recovery**: Automatic cleanup via timer
- **Tracking**: Increments TimeoutErrors counter

### Buffer Overflows
- **Detection**: Message exceeds 2048 bytes
- **Action**: Discard message, return error
- **Recovery**: Wait for new FIR
- **Tracking**: Increments BufferOverflows counter

### Continuation Without FIR
- **Detection**: FIR=0 fragment when not reassembling
- **Action**: Silently discard fragment
- **Recovery**: Wait for FIR=1 fragment
- **Tracking**: No counter (expected during sync)

## Thread Safety

All components are thread-safe:

- **MasterTransport**: RWMutex for outstation map, Mutex per outstation state
- **OutstationTransport**: Mutex for all state
- **TransportStatistics**: Atomic operations for all counters
- **Reassembler**: Called only under parent lock

Concurrent calls are safe:
```go
// Safe to call from multiple goroutines
go master.Send(10, apdu1)
go master.Send(20, apdu2)
go master.Receive(10, tpdu)
```

## Performance

### Memory Usage
- **Per Outstation State**: ~100 bytes + reassembly buffer (0-2048 bytes)
- **Statistics**: 80 bytes per entity
- **Segments**: Temporary allocation during Send()

### CPU Overhead
- **Segmentation**: O(n/248) where n = APDU size
- **Reassembly**: O(1) append per fragment
- **Validation**: O(1) per fragment

### Optimization Tips
1. Reuse transport instances (don't recreate)
2. Disable statistics if not needed (`EnableStatistics: false`)
3. Use appropriate reassembly timeout for your network

## Testing

Run all transport layer tests:
```bash
go test ./pkg/transport/... -v
```

Run specific test:
```bash
go test ./pkg/transport -run TestMasterTransport_SendMultipleFragments
```

Check test coverage:
```bash
go test ./pkg/transport -cover
```

## References

- **DNP3 Specification**: See [../../architecture_docs/dnp3_transport_layer.txt](../../architecture_docs/dnp3_transport_layer.txt)
- **Implementation Plan**: See [../../docs/transport_layer_implementation_plan.md](../../docs/transport_layer_implementation_plan.md)
- **Implementation Summary**: See [../../docs/transport_layer_implementation_summary.md](../../docs/transport_layer_implementation_summary.md)

## License

Part of the dnp3-go project.
