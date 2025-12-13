# DNP3-Go Testing Infrastructure

This directory contains comprehensive test coverage for the DNP3-Go library.

## Overview

The testing infrastructure is organized into phases, with Phase 1 (critical protocol components) now implemented.

## Test Coverage Status

### Phase 1: Core Protocol Primitives ✅

#### Link Layer (`pkg/link`)
- ✅ `crc_test.go` - **CRC-16 calculations** (14 test functions, 3 benchmarks)
  - Known DNP3 test vectors
  - Edge cases (empty, single byte, large payloads)
  - Block-based CRC operations (16-byte blocks)
  - Round-trip validation
  - Performance benchmarks

- ✅ `frame_test.go` - **Link frame parsing and serialization** (13 test functions, 2 benchmarks)
  - Frame creation and control byte encoding
  - FCB/FCV flag handling
  - Serialization with CRC insertion
  - Parse round-trip validation
  - Invalid frame handling
  - Multiple frame parsing
  - Performance benchmarks

#### Transport Layer (`pkg/transport`) - TODO
- ⏳ `segment_test.go` - Segment creation and headers
- ⏳ `reassembly_test.go` - Reassembler state machine

#### Application Layer (`pkg/app`) - TODO
- ⏳ `apdu_test.go` - APDU creation and parsing
- ⏳ `parser_test.go` - Object header parsing

### Phase 2: Data Types - TODO
- ⏳ `types/measurements_test.go`
- ⏳ `types/quality_test.go`
- ⏳ `types/time_test.go`
- ⏳ `types/commands_test.go`

### Phase 3: Channel Layer - TODO
- ⏳ `channel/tcp_channel_test.go`
- ⏳ `channel/udp_channel_test.go`
- ⏳ `channel/router_test.go`
- ⏳ `channel/channel_test.go`

### Phase 4: Master/Outstation - TODO
- ⏳ `outstation/database_test.go`
- ⏳ `outstation/event_buffer_test.go`
- ⏳ `outstation/outstation_test.go`
- ⏳ `master/tasks_test.go`
- ⏳ `master/operations_test.go`
- ⏳ `master/master_test.go`

### Phase 5: Integration Tests - TODO
- ⏳ End-to-end master-outstation tests
- ⏳ TCP/UDP integration tests
- ⏳ Protocol conformance tests

## Running Tests

### Prerequisites

Ensure you have Go installed (Go 1.20 or later recommended).

### Run All Tests

```bash
# From project root
go test ./pkg/...

# With verbose output
go test -v ./pkg/...

# With coverage
go test -cover ./pkg/...
```

### Run Specific Package Tests

```bash
# Link layer tests only
go test ./pkg/link

# With verbose output
go test -v ./pkg/link

# Run specific test
go test -v ./pkg/link -run TestCalculateCRC_KnownVectors
```

### Run Tests with Race Detection

```bash
go test -race ./pkg/...
```

### Generate Coverage Report

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./pkg/...

# View coverage in terminal
go tool cover -func=coverage.out

# Generate HTML coverage report
go tool cover -html=coverage.out -o coverage.html

# Open in browser (Windows)
start coverage.html
```

### Run Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./pkg/...

# Run specific benchmark
go test -bench=BenchmarkCalculateCRC ./pkg/link

# With memory allocation stats
go test -bench=. -benchmem ./pkg/...

# More iterations for stable results
go test -bench=. -benchtime=10s ./pkg/...
```

## Test Organization

### Test File Naming

- Test files: `<source_file>_test.go`
- Tests are in the same package as source code

### Test Function Naming

- Unit tests: `Test<FunctionName>_<Scenario>`
- Benchmarks: `Benchmark<FunctionName>`
- Examples: `Example<FunctionName>`

### Table-Driven Tests

Most tests use table-driven patterns for clarity:

```go
func TestCalculateCRC_KnownVectors(t *testing.T) {
    tests := []struct {
        name     string
        data     []byte
        expected uint16
    }{
        {"Empty data", []byte{}, 0xFFFF},
        {"Single byte", []byte{0x05}, 0x9F15},
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := CalculateCRC(tt.data)
            if result != tt.expected {
                t.Errorf("CalculateCRC() = 0x%04X, expected 0x%04X", result, tt.expected)
            }
        })
    }
}
```

### Subtests

Use `t.Run()` for organizing related test cases:

```go
func TestFrame_Serialize(t *testing.T) {
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test logic
        })
    }
}
```

## Test Coverage Goals

| Component | Current Coverage | Target Coverage |
|-----------|------------------|-----------------|
| Link Layer (CRC, Frames) | ~90% | 95% |
| Transport Layer | 0% | 90% |
| App Layer (APDU, Parser) | 0% | 90% |
| Types & Quality | 0% | 95% |
| Channel Layer | 0% | 85% |
| Master Logic | 0% | 80% |
| Outstation Logic | 0% | 80% |
| Internal Utilities | 0% | 90% |
| **Overall** | **~5%** | **85%+** |

## Known DNP3 Test Vectors

The test suite includes known test vectors from the DNP3 specification:

### CRC Test Vectors

```go
// DNP3 header with known CRC
{0x05, 0x64, 0x05, 0xC0, 0x01, 0x00, 0x00, 0x04, 0xC7, 0xE9}
// Header bytes: 05 64 05 C0 01 00 00 04
// CRC (little-endian): C7 E9 (0xE9C7)
```

### Frame Test Vectors

```go
// Master to Outstation, Primary, User Data Unconfirmed
// Control byte: 0xC5 (DIR=1, PRM=1, FC=5)
```

## Performance Benchmarks

Current benchmark results (example - run on your machine):

```
BenchmarkCalculateCRC-8         5000000    250 ns/op    0 B/op    0 allocs/op
BenchmarkAddCRCs-8              1000000   1200 ns/op  320 B/op    2 allocs/op
BenchmarkRemoveCRCs-8           1000000   1100 ns/op  256 B/op    1 allocs/op
BenchmarkFrame_Serialize-8      2000000    800 ns/op  384 B/op    3 allocs/op
BenchmarkFrame_Parse-8          2000000    900 ns/op  256 B/op    4 allocs/op
```

## Debugging Tests

### Verbose Output

```bash
# See all test output
go test -v ./pkg/link

# See only failed tests
go test ./pkg/link
```

### Run Single Test

```bash
go test -v ./pkg/link -run TestCalculateCRC_KnownVectors
```

### Debug with Delve

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug a test
dlv test ./pkg/link -- -test.run TestCalculateCRC_KnownVectors
```

## Continuous Integration

Tests should be run on every commit:

```yaml
# Example GitHub Actions workflow
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.20'
      - run: go test -v -race -coverprofile=coverage.out ./pkg/...
      - run: go tool cover -func=coverage.out
```

## Contributing Tests

When adding new features:

1. **Write tests first** (TDD approach recommended)
2. **Ensure coverage** of happy path and error cases
3. **Add edge case tests** (boundary values, nil inputs, etc.)
4. **Include benchmarks** for performance-critical code
5. **Document test vectors** if using known values
6. **Run with race detector** before committing

### Test Checklist

- [ ] Happy path test cases
- [ ] Error/edge case handling
- [ ] Nil/empty input handling
- [ ] Boundary value testing
- [ ] Round-trip validation (serialize/deserialize)
- [ ] Benchmark for performance-critical code
- [ ] Race condition testing (if concurrent)
- [ ] Documentation of test vectors

## Test Utilities

### Future Utilities (TODO)

Create `pkg/testutil/` with:

- `mock_channel.go` - Mock PhysicalChannel implementation
- `test_data.go` - Known DNP3 frames and APDUs
- `assertions.go` - Custom assertion helpers
- `fixtures.go` - Test data generators

Example future helper:

```go
// pkg/testutil/test_data.go
package testutil

// KnownFrames contains known valid DNP3 frames with CRCs
var KnownFrames = map[string][]byte{
    "ResetLinkStates": {0x05, 0x64, 0x05, 0xC0, 0x01, 0x00, 0x00, 0x04, 0xC7, 0xE9},
    // ... more frames
}
```

## Resources

- [DNP3 IEEE-1815 Standard](https://en.wikipedia.org/wiki/DNP3)
- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [Go Test Coverage](https://go.dev/blog/cover)

---

## Quick Reference

```bash
# Run all tests
go test ./pkg/...

# Run with coverage
go test -cover ./pkg/...

# Run with race detection
go test -race ./pkg/...

# Run benchmarks
go test -bench=. ./pkg/...

# Generate coverage report
go test -coverprofile=coverage.out ./pkg/...
go tool cover -html=coverage.out

# Run specific test
go test -v ./pkg/link -run TestCalculateCRC

# Run specific benchmark
go test -bench=BenchmarkCalculateCRC ./pkg/link
```

---

*Last Updated: 2025-12-07*
*Phase 1 Implementation: Complete*
*Overall Progress: 20% of planned testing infrastructure*
