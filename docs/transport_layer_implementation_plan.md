# DNP3 Transport Layer Implementation Plan

## Overview
This document outlines the implementation plan for the DNP3 transport layer for both Master and Outstation stations. The transport layer sits between the Data Link Layer and Application Layer, handling segmentation and reassembly of large messages.

## Current State Analysis

### Existing Implementation
The codebase already contains basic transport layer components:

1. **[pkg/transport/segment.go](../pkg/transport/segment.go)** - Transport segment data structure and segmentation logic
   - `Segment` struct with FIR, FIN, Seq, and Data fields
   - `SegmentData()` function that breaks APDU into 248-byte segments
   - Header parsing and serialization functions
   - **Status**: Complete and spec-compliant

2. **[pkg/transport/reassembly.go](../pkg/transport/reassembly.go)** - Reassembly logic for received segments
   - `Reassembler` struct handling fragment assembly
   - Sequence validation and error detection
   - Buffer management with 2048-byte limit
   - **Status**: Complete and spec-compliant

3. **[pkg/transport/layer.go](../pkg/transport/layer.go)** - Generic transport layer
   - `Layer` struct with receive and send methods
   - Integration of reassembler and segmentation
   - **Status**: Generic implementation exists but needs Master/Outstation specialization

### Gaps Identified

1. **No Master-Specific Transport Layer**
   - Need separate sequence tracking per outstation
   - Need to handle multiple concurrent outstations
   - Need integration with master session management

2. **No Outstation-Specific Transport Layer**
   - Need to track master direction sequences separately
   - Need to handle both solicited and unsolicited responses
   - Need integration with outstation session management

3. **Missing Timer Management**
   - Reassembly timeout (60-120 seconds) not implemented
   - No timeout cleanup for incomplete messages

4. **Missing Statistics/Diagnostics**
   - No tracking of sequence errors
   - No fragment statistics
   - No timeout event logging

5. **No Integration with Existing Link Layer**
   - Master and Outstation link layers exist but transport layer not integrated

## Implementation Plan

### Phase 1: Master Transport Layer

#### 1.1 Create MasterTransport struct

**File**: `pkg/transport/master_transport.go`

**Purpose**: Manage transport layer for DNP3 Master, handling multiple outstations

**Key Components**:
```go
type MasterTransport struct {
    // Per-outstation state tracking
    outstations map[uint16]*OutstationTransportState

    // Configuration
    reassemblyTimeout time.Duration

    // Statistics
    stats *TransportStatistics

    // Synchronization
    mu sync.RWMutex
}

type OutstationTransportState struct {
    // TX direction (Master → Outstation)
    txSequence uint8

    // RX direction (Outstation → Master)
    rxReassembler *Reassembler
    reassemblyTimer *time.Timer

    // Statistics
    lastActivity time.Time
    txFragmentCount uint64
    rxFragmentCount uint64
}
```

**Key Methods**:
- `NewMasterTransport(config)` - Constructor with configuration
- `Send(outstationAddr uint16, apdu []byte) [][]byte` - Segment APDU for specific outstation
- `Receive(outstationAddr uint16, tpdu []byte) ([]byte, error)` - Reassemble from specific outstation
- `GetStats(outstationAddr uint16) *TransportStatistics` - Get statistics
- `Reset(outstationAddr uint16)` - Reset state for specific outstation
- `CleanupStaleReassembly()` - Cleanup timed-out reassembly buffers

**Features**:
1. **Independent Sequence Tracking**
   - Separate TX sequence counter per outstation
   - Separate RX reassembler per outstation
   - Prevents sequence conflicts when communicating with multiple devices

2. **Reassembly Timeout**
   - Start timer when FIR fragment received
   - Reset timer on each valid continuation
   - Discard incomplete message on timeout (60-120 seconds)
   - Clean up resources

3. **Error Handling**
   - Log sequence errors with outstation address
   - Track error statistics per outstation
   - Reset reassembly on errors

4. **Statistics Collection**
   - Count fragments sent/received per outstation
   - Track sequence errors
   - Track timeout events
   - Monitor buffer utilization

#### 1.2 Integration Points

**Integration with Master Link Layer** (`pkg/link/master_link.go`):
- Link layer receives complete frames from channel
- Extracts user data (transport header + app data)
- Passes to `MasterTransport.Receive()`
- Gets reassembled APDU or nil (if incomplete)
- Delivers complete APDU to application layer

**Integration with Master Session** (`pkg/master/session.go`):
- Application layer wants to send request
- Calls `MasterTransport.Send()` with APDU
- Gets array of transport segments
- Wraps each segment in link layer frame
- Transmits via link layer

### Phase 2: Outstation Transport Layer

#### 2.1 Create OutstationTransport struct

**File**: `pkg/transport/outstation_transport.go`

**Purpose**: Manage transport layer for DNP3 Outstation

**Key Components**:
```go
type OutstationTransport struct {
    // TX direction (Outstation → Master)
    txSequence uint8

    // RX direction (Master → Outstation)
    rxReassembler *Reassembler
    reassemblyTimer *time.Timer

    // Configuration
    reassemblyTimeout time.Duration

    // Statistics
    stats *TransportStatistics

    // Synchronization
    mu sync.RWMutex
}
```

**Key Methods**:
- `NewOutstationTransport(config)` - Constructor
- `Send(apdu []byte) [][]byte` - Segment APDU for master
- `Receive(tpdu []byte) ([]byte, error)` - Reassemble from master
- `GetStats() *TransportStatistics` - Get statistics
- `Reset()` - Reset state
- `CleanupStaleReassembly()` - Timeout cleanup

**Features**:
1. **Simplified State Management**
   - Single TX sequence (only one master)
   - Single RX reassembler (only one master)
   - Simpler than master which tracks multiple outstations

2. **Unsolicited Response Support**
   - Same transport mechanism for solicited/unsolicited
   - Use same sequence counter for both
   - No special handling needed

3. **Reassembly Timeout**
   - Same timeout mechanism as master
   - Clean up incomplete requests from master
   - Handle master disconnection gracefully

4. **Statistics Collection**
   - Track fragments sent/received
   - Monitor errors and timeouts
   - Support diagnostics

#### 2.2 Integration Points

**Integration with Outstation Link Layer** (`pkg/link/outstation_link.go`):
- Link layer receives frames from master
- Extracts transport segment
- Passes to `OutstationTransport.Receive()`
- Delivers complete APDU to application layer

**Integration with Outstation** (`pkg/outstation/outstation.go`):
- Application layer generates response
- Calls `OutstationTransport.Send()`
- Gets transport segments
- Wraps in link frames
- Transmits to master

### Phase 3: Shared Components

#### 3.1 Transport Statistics

**File**: `pkg/transport/statistics.go`

**Purpose**: Track transport layer metrics

```go
type TransportStatistics struct {
    // Fragment counts
    TxFragments uint64
    RxFragments uint64

    // Message counts
    TxMessages uint64
    RxMessages uint64

    // Error counts
    SequenceErrors uint64
    TimeoutErrors uint64
    BufferOverflows uint64

    // Timing
    LastTxTime time.Time
    LastRxTime time.Time

    // Current state
    ReassemblyInProgress bool
    BufferUtilization int
}
```

#### 3.2 Transport Configuration

**File**: `pkg/transport/config.go`

**Purpose**: Configuration for transport layer

```go
type TransportConfig struct {
    // Reassembly timeout (default: 120 seconds)
    ReassemblyTimeout time.Duration

    // Maximum reassembly buffer size (default: 2048)
    MaxReassemblySize int

    // Enable statistics collection
    EnableStatistics bool

    // Logger for diagnostics
    Logger Logger
}
```

#### 3.3 Enhanced Error Types

**File**: Update `pkg/transport/reassembly.go`

Add more descriptive errors:
```go
var (
    ErrInvalidSequence = errors.New("invalid transport sequence")
    ErrMissingFIR      = errors.New("missing FIR segment")
    ErrBufferOverflow  = errors.New("reassembly buffer overflow")
    ErrReassemblyTimeout = errors.New("reassembly timeout")
    ErrSequenceGap     = errors.New("sequence gap detected")
    ErrDuplicateSequence = errors.New("duplicate sequence number")
)
```

### Phase 4: Testing Strategy

#### 4.1 Unit Tests

**Master Transport Tests** (`pkg/transport/master_transport_test.go`):
1. Single fragment message
2. Multi-fragment message (2-5 fragments)
3. Sequence wrap-around (62, 63, 0, 1)
4. Multiple concurrent outstations
5. Sequence error detection
6. Reassembly timeout
7. Buffer overflow protection
8. Statistics accuracy

**Outstation Transport Tests** (`pkg/transport/outstation_transport_test.go`):
1. Single fragment message
2. Multi-fragment message
3. Sequence wrap-around
4. Sequence error recovery
5. Timeout handling
6. Statistics tracking

**Existing Component Tests**:
- Verify `segment.go` still works correctly
- Verify `reassembly.go` still works correctly
- Add edge case tests

#### 4.2 Integration Tests

**Master Integration** (`pkg/transport/master_integration_test.go`):
1. End-to-end message flow with mock link layer
2. Multiple outstations simultaneously
3. Error recovery scenarios
4. Timeout scenarios

**Outstation Integration** (`pkg/transport/outstation_integration_test.go`):
1. End-to-end message flow with mock link layer
2. Solicited and unsolicited responses
3. Error scenarios

#### 4.3 Test Scenarios

**Scenario 1: Simple Request/Response**
```
Master → Outstation: 0xC0 (Single fragment request)
Outstation → Master: 0xC0 (Single fragment response)
```

**Scenario 2: Large File Transfer**
```
Master → Outstation: 0x40 (First, 248 bytes)
Master → Outstation: 0x01 (Middle, 248 bytes)
Master → Outstation: 0x02 (Middle, 248 bytes)
Master → Outstation: 0x83 (Final, 150 bytes)
Total: 894 bytes assembled
```

**Scenario 3: Sequence Error**
```
Master → Outstation: 0x40 (SEQ=0)
Master → Outstation: 0x02 (SEQ=2) <- Gap!
Outstation: Discards, waits for FIR
Master → Outstation: 0xC0 (New message)
Outstation: Processes new message
```

**Scenario 4: Timeout**
```
Master → Outstation: 0x40 (SEQ=0)
Master → Outstation: 0x01 (SEQ=1)
... 120 seconds pass ...
Outstation: Timeout, discard buffer
```

**Scenario 5: Multiple Outstations**
```
Master → OS1: 0x40 (SEQ=5, start message to OS1)
Master → OS2: 0x40 (SEQ=10, start message to OS2)
Master → OS1: 0x06 (SEQ=6, continue OS1)
Master → OS2: 0x0B (SEQ=11, continue OS2)
Master → OS1: 0x87 (SEQ=7, finish OS1)
Master → OS2: 0x8C (SEQ=12, finish OS2)
Both messages assembled independently
```

## Implementation Timeline

### Milestone 1: Core Master Transport
- [ ] Create `master_transport.go` with basic structure
- [ ] Implement per-outstation state management
- [ ] Implement Send() method
- [ ] Implement Receive() method
- [ ] Add basic unit tests

### Milestone 2: Core Outstation Transport
- [ ] Create `outstation_transport.go` with basic structure
- [ ] Implement Send() method
- [ ] Implement Receive() method
- [ ] Add basic unit tests

### Milestone 3: Advanced Features
- [ ] Add reassembly timeout to both
- [ ] Implement statistics collection
- [ ] Add configuration support
- [ ] Add comprehensive logging

### Milestone 4: Testing & Integration
- [ ] Complete unit test coverage
- [ ] Integration tests with link layer
- [ ] Performance testing
- [ ] Documentation updates

### Milestone 5: Integration with Existing Code
- [ ] Update master to use MasterTransport
- [ ] Update outstation to use OutstationTransport
- [ ] End-to-end testing
- [ ] Bug fixes and optimization

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Application Layer                        │
│         (Master/Outstation Application Logic)               │
└────────────────────┬───────────────────┬────────────────────┘
                     │                   │
                     │ APDU              │ APDU
                     ↓                   ↓
┌────────────────────────────┐  ┌──────────────────────────┐
│   MasterTransport          │  │  OutstationTransport     │
│                            │  │                          │
│  ┌──────────────────────┐  │  │  TX Sequence: uint8     │
│  │ Outstation 1         │  │  │  RX Reassembler         │
│  │  - TX Seq            │  │  │  Timer                  │
│  │  - RX Reassembler    │  │  │                         │
│  │  - Timer             │  │  └──────────────────────────┘
│  └──────────────────────┘  │
│  ┌──────────────────────┐  │
│  │ Outstation 2         │  │
│  │  - TX Seq            │  │
│  │  - RX Reassembler    │  │
│  │  - Timer             │  │
│  └──────────────────────┘  │
└────────────────────────────┘
         │                              │
         │ TPDU                         │ TPDU
         ↓                              ↓
┌────────────────────────────┐  ┌──────────────────────────┐
│   MasterLink               │  │   OutstationLink         │
│   (Link Layer)             │  │   (Link Layer)           │
└────────────────────────────┘  └──────────────────────────┘
```

## Key Design Decisions

### 1. Separate Master/Outstation Implementations
**Decision**: Create distinct `MasterTransport` and `OutstationTransport` rather than one generic layer

**Rationale**:
- Master manages multiple outstations, outstation has one master
- Different state management requirements
- Clearer code with specific purpose
- Easier to optimize for each role

### 2. Per-Outstation State in Master
**Decision**: Map of outstation address → transport state

**Rationale**:
- Prevents sequence conflicts between outstations
- Independent reassembly buffers prevent data mixing
- Matches DNP3 spec requirement for independent sequencing

### 3. Reassembly Timeout
**Decision**: Implement using Go timers, default 120 seconds

**Rationale**:
- Prevents memory leaks from incomplete messages
- Matches DNP3 recommended timeout
- Allows cleanup of stale connections

### 4. Statistics Collection
**Decision**: Optional statistics with minimal overhead

**Rationale**:
- Essential for diagnostics and monitoring
- Helps debug communication issues
- Minimal performance impact

### 5. Error Handling Philosophy
**Decision**: Silent discard on errors, log for diagnostics

**Rationale**:
- Transport layer doesn't retry (matches DNP3 spec)
- Application layer handles retries
- Logging enables troubleshooting

## File Structure

```
pkg/transport/
├── segment.go              (existing - segmentation logic)
├── reassembly.go          (existing - reassembly logic)
├── layer.go               (existing - generic layer, may deprecate)
├── master_transport.go    (NEW - master transport layer)
├── outstation_transport.go (NEW - outstation transport layer)
├── config.go              (NEW - configuration)
├── statistics.go          (NEW - statistics tracking)
├── errors.go              (NEW - error definitions)
├── master_transport_test.go (NEW - master tests)
└── outstation_transport_test.go (NEW - outstation tests)
```

## Dependencies

### Internal Dependencies
- `pkg/link` - Link layer integration
- `pkg/master` - Master session integration
- `pkg/outstation` - Outstation integration
- Logging framework (to be determined)

### External Dependencies
- Standard library only:
  - `sync` - Synchronization primitives
  - `time` - Timers and timeouts
  - `bytes` - Buffer management (already used)
  - `errors` - Error handling

## Success Criteria

1. **Correctness**
   - All unit tests pass
   - Integration tests pass
   - Handles all DNP3 spec scenarios

2. **Reliability**
   - No memory leaks
   - Proper timeout cleanup
   - Correct error handling

3. **Performance**
   - Minimal overhead per fragment
   - Efficient buffer management
   - Low CPU usage

4. **Maintainability**
   - Clear code structure
   - Comprehensive documentation
   - Good test coverage (>80%)

## References

1. DNP3 Transport Layer Specification - [architecture_docs/dnp3_transport_layer.txt](../architecture_docs/dnp3_transport_layer.txt)
2. Existing Implementation:
   - [pkg/transport/segment.go](../pkg/transport/segment.go)
   - [pkg/transport/reassembly.go](../pkg/transport/reassembly.go)
   - [pkg/link/master_link.go](../pkg/link/master_link.go)
   - [pkg/link/outstation_link.go](../pkg/link/outstation_link.go)
