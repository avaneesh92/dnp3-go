# DNP3-Go Testing Infrastructure Plan

## Overview

This document outlines the comprehensive testing strategy for the DNP3-Go library. The testing infrastructure is organized into phases, prioritizing critical protocol components first.

## Current Status

- **Total Packages**: 11 main packages with 45 Go files (~6,610 lines of code)
- **Existing Tests**: 0 test files (CRITICAL GAP)
- **Test Coverage**: 0%

## Testing Phases

### Phase 1: Core Protocol Primitives ⏳ (CRITICAL - Week 1)

**Priority**: HIGHEST - These are the foundation of the protocol implementation

#### 1.1 Link Layer Tests (`pkg/link`)
- ✅ `crc_test.go` - CRC-16 calculations (DNP3 polynomial 0xA6BC)
  - CalculateCRC() for various data sizes
  - VerifyCRC() validation
  - AddCRCs() / RemoveCRCs() for 16-byte blocks
  - Edge cases: empty data, single byte, large payloads
  - Known test vectors from DNP3 spec

- ✅ `frame_test.go` - Link frame parsing and serialization
  - Frame header parsing (start bytes, length, control, dest, src)
  - Control byte combinations (DIR, PRM, FCB, FCV, DFC, FC)
  - Round-trip serialization/deserialization
  - Invalid frames and error handling
  - CRC validation in frames

#### 1.2 Transport Layer Tests (`pkg/transport`)
- [ ] `segment_test.go` - Segment creation
  - Header byte encoding (FIR, FIN, SEQ)
  - 249-byte segment boundaries
  - Sequence number wrapping (0-63)

- [ ] `reassembly_test.go` - Reassembler state machine
  - Normal sequential reassembly
  - Out-of-sequence detection
  - Buffer overflow handling
  - State transitions (idle → receiving → complete)
  - Reset on errors

#### 1.3 Application Layer Tests (`pkg/app`)
- [ ] `apdu_test.go` - APDU creation and parsing
  - Control byte manipulation (CON, FIN, FIR, UNS, SEQ)
  - Request/Response/Unsolicited differentiation
  - APDU constructors (NewRequest, NewResponse, NewUnsolicited)
  - Round-trip encoding/decoding

- [ ] `parser_test.go` - Object header parsing
  - All 6 qualifier types (8-bit, 16-bit, 32-bit start/stop)
  - Range parsing (start-stop, count, all objects)
  - Indexed vs ranged vs all objects
  - Insufficient data error handling
  - Malformed headers

- [ ] `functions_test.go` - Function codes
  - Valid function code constants
  - Function code validation

**Success Criteria**: All protocol primitives have >90% code coverage

---

### Phase 2: Data Types and Utilities ⏳ (HIGH - Week 2)

#### 2.1 Types Tests (`pkg/types`)
- [ ] `measurements_test.go` - Measurement types
  - Binary, DoubleBitBinary, Analog, Counter creation
  - Flag combinations
  - Timestamp handling
  - IndexedXXX wrapper types

- ✅ `quality_test.go` - Quality flag operations
  - Flag bit operations (Online, Restart, CommLost, RemoteForced, LocalForced, Overrange)
  - IsOnline(), HasCommLost(), IsForced(), IsGood()
  - Flag combinations and precedence
  - Complex real-world scenarios

- [ ] `time_test.go` - DNP3Time operations
  - Epoch calculations (milliseconds since 1970)
  - Now() function
  - Time conversions to/from Go time.Time

- [ ] `commands_test.go` - Command types
  - CROB (Control Relay Output Block)
  - Analog outputs (Int16, Int32, Float32, Double64)
  - Command status codes

#### 2.2 Internal Utilities Tests (`pkg/internal`)
- [ ] `queue/priority_queue_test.go` - Priority queue
  - Push/Pop operations
  - Priority ordering
  - Time-based scheduling
  - NextReady() time filtering
  - Thread safety (concurrent operations)
  - Empty queue handling

**Success Criteria**: All data types validated with edge cases

---

### Phase 3: Channel Layer Tests ⏳ (HIGH - Week 3)

#### 3.1 Channel Tests (`pkg/channel`)
- [ ] `tcp_channel_test.go` - TCP channel
  - Connection establishment
  - Read/Write operations
  - Timeout handling
  - Reconnection logic
  - Statistics tracking

- [ ] `udp_channel_test.go` - UDP channel
  - Datagram send/receive
  - Address binding
  - Statistics tracking

- [ ] `router_test.go` - Session routing
  - Address-based routing
  - Session registration/removal
  - Unknown address handling

- [ ] `channel_test.go` - Channel manager
  - Open/Close state transitions
  - Concurrent read/write loops
  - Session management
  - Error recovery

**Success Criteria**: Channel state machines thoroughly tested

---

### Phase 4: Master/Outstation Logic ⏳ (COMPLEX - Week 4-5)

#### 4.1 Outstation Tests (`pkg/outstation`)
- [ ] `database_test.go` - Database operations
  - Point storage for all 7 types
  - Static vs event variations
  - Deadband calculations
  - Index bounds checking

- [ ] `event_buffer_test.go` - Event buffering
  - Event queue management
  - Class assignment (1, 2, 3)
  - Buffer overflow handling
  - Event clearing

- [ ] `update_builder_test.go` - Update builder DSL
  - Builder pattern usage
  - Atomic updates
  - EventMode (Detect, Force, Suppress)

- [ ] `outstation_test.go` - Outstation state machine
  - Enable/Disable/Shutdown lifecycle
  - Request handling
  - Unsolicited response generation
  - Command processing

#### 4.2 Master Tests (`pkg/master`)
- [ ] `tasks_test.go` - Task definitions
  - Integrity scan task
  - Class scan tasks
  - Command tasks
  - Task priorities

- [ ] `operations_test.go` - Read/Write operations
  - Integrity polls
  - Class polls
  - Direct operate
  - Select/operate sequences

- [ ] `master_test.go` - Master state machine
  - Enable/Disable/Shutdown lifecycle
  - Task queue management
  - Periodic scans
  - Response correlation
  - Timeout handling
  - Sequence number tracking

**Success Criteria**: State machines validated with all transitions

---

### Phase 5: Integration Tests ⏳ (Week 6)

#### 5.1 End-to-End Tests
- [ ] `integration/master_outstation_test.go` - Full protocol stack
  - Master-Outstation communication over mock channel
  - Integrity scan flow
  - Event reporting
  - Command execution (SELECT/OPERATE, DIRECT OPERATE)
  - Unsolicited responses

- [ ] `integration/tcp_integration_test.go` - TCP transport
  - Real TCP connections (localhost)
  - Connection recovery
  - Multi-session handling

- [ ] `integration/protocol_conformance_test.go` - DNP3 spec compliance
  - Known test vectors from IEEE-1815 standard
  - Interoperability scenarios

**Success Criteria**: Full protocol flows work end-to-end

---

### Phase 6: Test Utilities and Helpers ⏳ (Ongoing)

#### 6.1 Test Helpers (`pkg/testutil/`)
- [ ] `testutil/mock_channel.go` - Mock PhysicalChannel
  - Controllable Read/Write
  - Inject errors
  - Capture sent data

- [ ] `testutil/test_data.go` - Test fixtures
  - Known DNP3 frames with CRCs
  - Sample APDUs
  - Test measurements

- [ ] `testutil/assertions.go` - Custom assertions
  - AssertBytesEqual with hex dump
  - AssertAPDUEqual
  - AssertMeasurementEqual

#### 6.2 Benchmarks
- [ ] `link/crc_bench_test.go` - CRC performance
- [ ] `app/parser_bench_test.go` - Parser performance
- [ ] `channel/channel_bench_test.go` - Throughput benchmarks

**Success Criteria**: Reusable test infrastructure for future development

---

## Test Coverage Goals

| Component | Target Coverage | Priority |
|-----------|----------------|----------|
| Link Layer (CRC, Frames) | 95% | CRITICAL |
| Transport Layer | 90% | CRITICAL |
| App Layer (APDU, Parser) | 90% | CRITICAL |
| Types & Quality | 95% | HIGH |
| Channel Layer | 85% | HIGH |
| Master Logic | 80% | MEDIUM |
| Outstation Logic | 80% | MEDIUM |
| Internal Utilities | 90% | MEDIUM |
| **Overall Target** | **85%+** | |

---

## Testing Tools and Conventions

### Tools
- Standard Go testing framework (`testing` package)
- Table-driven tests for multiple scenarios
- Subtests (`t.Run()`) for organization
- `go test -race` for race condition detection
- `go test -cover` for coverage reporting
- `go test -bench` for performance testing

### Conventions
- Test file naming: `<file>_test.go`
- Test function naming: `Test<FunctionName>_<Scenario>`
- Benchmark naming: `Benchmark<FunctionName>`
- Use `testdata/` directories for fixtures
- Clear test case descriptions in table tests

### Example Test Structure
```go
func TestCalculateCRC_ValidData(t *testing.T) {
    tests := []struct {
        name     string
        data     []byte
        expected uint16
    }{
        {"Empty", []byte{}, 0x0000},
        {"SingleByte", []byte{0x05}, 0x9F15},
        {"KnownVector", []byte{0x05, 0x64, 0x05, 0xC0, 0x01, 0x00, 0x00, 0x04}, 0xE9C7},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := CalculateCRC(tt.data)
            if result != tt.expected {
                t.Errorf("CalculateCRC(%x) = %04X, expected %04X", tt.data, result, tt.expected)
            }
        })
    }
}
```

---

## Running Tests

```bash
# Run all tests
go test ./pkg/...

# Run with coverage
go test -cover ./pkg/...

# Run with race detection
go test -race ./pkg/...

# Run specific package
go test ./pkg/link/

# Run with verbose output
go test -v ./pkg/...

# Generate coverage report
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out

# Run benchmarks
go test -bench=. ./pkg/...
```

---

## Success Metrics

1. **Coverage**: Achieve 85%+ overall test coverage
2. **CI Integration**: All tests pass on every commit
3. **Performance**: No performance regressions in benchmarks
4. **Race Conditions**: Zero race conditions detected
5. **Documentation**: All public APIs have usage examples in tests
6. **Regression**: All bug fixes include regression tests

---

## Timeline Summary

| Phase | Duration | Deliverable |
|-------|----------|------------|
| Phase 1 | Week 1 | Core protocol tests (Link, Transport, App) |
| Phase 2 | Week 2 | Data types and utilities tests |
| Phase 3 | Week 3 | Channel layer tests |
| Phase 4 | Week 4-5 | Master/Outstation tests |
| Phase 5 | Week 6 | Integration tests |
| Phase 6 | Ongoing | Test utilities and benchmarks |

**Total Estimated Effort**: 6 weeks for comprehensive coverage

---

## Next Steps

1. ✅ Create this testing plan
2. ⏳ Implement Phase 1 critical tests (CRC, frames, segments, APDUs)
3. Set up CI pipeline to run tests automatically
4. Establish coverage tracking and reporting
5. Begin Phase 2-6 implementation

---

*Last Updated: 2025-12-07*
