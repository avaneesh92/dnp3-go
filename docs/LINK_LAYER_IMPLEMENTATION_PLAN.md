# DNP3 Link Layer Implementation Plan

## Overview
This document outlines the implementation plan for the DNP3 Link Layer functionality in the dnp3-go library. The link layer is responsible for reliable frame transmission, duplicate detection, and link state management between Master and Outstation devices.

## Current State Analysis

### Existing Components
1. **Frame Structure** (`pkg/link/frame.go`)
   - Frame parsing and serialization
   - Control byte handling (DIR, PRM, FCB, FCV)
   - CRC validation
   - Basic frame construction

2. **Constants** (`pkg/link/constants.go`)
   - Function code definitions (partial)
   - Control byte masks
   - Frame size constants
   - Error definitions

3. **CRC Handling** (`pkg/link/crc.go`)
   - CRC calculation and verification
   - Block-based CRC for data segments

### Missing Components
1. **Link Layer State Machine** - No implementation found
2. **FCB Tracking Logic** - No duplicate detection mechanism
3. **Timeout and Retry Management** - No retry logic
4. **Link Layer Interface** - No clear API for upper layers
5. **Master/Outstation-specific Logic** - No role-based behavior

## Implementation Plan

### Phase 1: Complete Constants and Function Codes

#### Task 1.1: Add Missing Function Codes
**File:** `pkg/link/constants.go`

**Action Items:**
- Add RESET USER PROCESS (0x01)
- Add TEST LINK STATES (0x02)
- Add CONFIRMED USER DATA (0x03)
- Add UNCONFIRMED USER DATA (0x04)
- Add LINK NOT FUNCTIONING (0x0E)
- Add LINK NOT USED (0x0F)
- Fix function code mapping to match DNP3 specification

**Current Issues:**
```go
// Current (INCORRECT)
FuncUserDataConfirmed   FunctionCode = 0x04
FuncUserDataUnconfirmed FunctionCode = 0x05

// Should be (CORRECT)
FuncUserDataConfirmed   FunctionCode = 0x03
FuncUserDataUnconfirmed FunctionCode = 0x04
```

#### Task 1.2: Add Control Byte Constants
**File:** `pkg/link/constants.go`

**Action Items:**
- Add predefined control bytes for common operations:
  - RESET_LINK = 0xC0
  - RESET_USER_PROCESS = 0xC1
  - TEST_LINK_STATES = 0xC2
  - USER_DATA_CONF_FCB0 = 0x53
  - USER_DATA_CONF_FCB1 = 0x73
  - USER_DATA_UNCONF = 0x44
  - REQUEST_LINK_STATUS = 0xC9
  - ACK = 0x80
  - NACK = 0x81
  - LINK_STATUS = 0x8B
  - LINK_NOT_FUNCTIONING = 0x8E
  - LINK_NOT_USED = 0x8F

### Phase 2: Design Link Layer Interface

#### Task 2.1: Define Link Layer Interface
**File:** `pkg/link/interface.go` (NEW)

**Interface Design:**
```go
type LinkLayer interface {
    // Transmission
    SendFrame(frame *Frame) error
    SendUserData(data []byte, confirmed bool) error

    // Reception
    OnFrameReceived(frame *Frame) error

    // Control operations
    ResetLink() error
    TestLink() error

    // State management
    GetState() LinkState
    IsOnline() bool

    // Configuration
    SetTimeout(duration time.Duration)
    SetRetries(count int)

    // Callbacks for upper layers
    SetDataCallback(callback DataCallback)
    SetStatusCallback(callback StatusCallback)
}

type DataCallback func(data []byte)
type StatusCallback func(state LinkState)
```

### Phase 3: Implement Master Link Layer

#### Task 3.1: Create Master Link Layer Structure
**File:** `pkg/link/master_link.go` (NEW)

**Structure Design:**
```go
type MasterLink struct {
    // Configuration
    localAddress  uint16
    remoteAddress uint16
    timeout       time.Duration
    maxRetries    int

    // State
    state         LinkState
    fcb           bool          // Current FCB value
    lastSentFrame *Frame       // For retransmission

    // Timing
    responseTimer *time.Timer
    retryCount    int

    // Communication
    sendChan      chan []byte   // To physical layer
    recvChan      chan *Frame   // From physical layer
    dataChan      chan []byte   // To transport layer

    // Callbacks
    dataCallback   DataCallback
    statusCallback StatusCallback

    // Synchronization
    mu            sync.Mutex
    ctx           context.Context
    cancel        context.CancelFunc
}
```

#### Task 3.2: Implement Master State Machine
**File:** `pkg/link/master_link.go`

**States:**
- IDLE: Waiting for data to send
- WAIT_ACK: Waiting for acknowledgment
- RESET_PENDING: Reset link in progress
- TEST_PENDING: Test link in progress
- ERROR: Link error state

**State Transitions:**
```
IDLE → WAIT_ACK (on SendUserData)
WAIT_ACK → IDLE (on ACK received)
WAIT_ACK → WAIT_ACK (on timeout, retry)
WAIT_ACK → ERROR (on max retries exceeded)
IDLE → RESET_PENDING (on ResetLink)
RESET_PENDING → IDLE (on ACK)
IDLE → TEST_PENDING (on TestLink)
TEST_PENDING → IDLE (on ACK)
```

#### Task 3.3: Implement Master Functions

**3.3.1: ResetLink()**
- Send RESET LINK frame (0xC0)
- Wait for ACK (0x80)
- Reset FCB to 0 on success
- Handle timeout and retry

**3.3.2: ResetUserProcess()**
- Send RESET USER PROCESS frame (0xC1)
- Wait for ACK (0x80)
- Does NOT reset FCB

**3.3.3: TestLink()**
- Send TEST LINK STATES frame (0xC2)
- Wait for ACK (0x80)
- Return success/failure

**3.3.4: SendUserData(confirmed bool)**
- If confirmed:
  - Toggle FCB for NEW messages
  - Set FCV = 1
  - Send frame with function code 0x03
  - Wait for ACK/NACK
  - Retry on timeout/NACK (with SAME FCB)
- If unconfirmed:
  - Send frame with function code 0x04
  - No ACK expected

**3.3.5: RequestLinkStatus()**
- Send REQUEST LINK STATUS frame (0xC9)
- Wait for LINK STATUS response (0x8B)

#### Task 3.4: Implement FCB Management
**File:** `pkg/link/master_link.go`

**FCB Logic:**
```go
func (m *MasterLink) toggleFCB() {
    m.fcb = !m.fcb
}

func (m *MasterLink) getCurrentFCB() bool {
    return m.fcb
}

// Only toggle on NEW confirmed messages
// Keep same FCB for retransmissions
```

#### Task 3.5: Implement Timeout and Retry Logic
**File:** `pkg/link/master_link.go`

**Retry Logic:**
```go
func (m *MasterLink) sendWithRetry(frame *Frame) error {
    m.retryCount = 0
    m.lastSentFrame = frame

    for m.retryCount <= m.maxRetries {
        // Send frame
        m.transmit(frame)

        // Wait for response
        select {
        case response := <-m.responseChan:
            return m.handleResponse(response)
        case <-time.After(m.timeout):
            m.retryCount++
            // Retry with SAME frame (same FCB)
        }
    }

    return ErrMaxRetriesExceeded
}
```

#### Task 3.6: Implement Response Handling
**File:** `pkg/link/master_link.go`

**Handle Response Types:**
- ACK (0x80): Success, proceed
- NACK (0x81): Retry after delay
- LINK STATUS (0x8B): Extract status info
- UNCONFIRMED USER DATA (0x84): Pass to transport layer
- LINK NOT FUNCTIONING (0x8E): Alert, attempt reset
- LINK NOT USED (0x8F): Configuration error

### Phase 4: Implement Outstation Link Layer

#### Task 4.1: Create Outstation Link Layer Structure
**File:** `pkg/link/outstation_link.go` (NEW)

**Structure Design:**
```go
type OutstationLink struct {
    // Configuration
    localAddress  uint16
    remoteAddress uint16
    responseTimeout time.Duration

    // State
    state         LinkState
    lastFCB       bool          // Last valid FCB received
    fcbInitialized bool         // FCB has been set

    // Communication
    sendChan      chan []byte   // To physical layer
    recvChan      chan *Frame   // From physical layer
    dataChan      chan []byte   // To transport layer

    // Callbacks
    dataCallback   DataCallback
    statusCallback StatusCallback

    // Unsolicited support
    unsolicitedEnabled bool

    // Synchronization
    mu            sync.Mutex
    ctx           context.Context
    cancel        context.CancelFunc
}
```

#### Task 4.2: Implement Outstation State Machine
**File:** `pkg/link/outstation_link.go`

**States:**
- IDLE: Waiting for master requests
- PROCESSING: Processing received frame
- SEND_ACK: Sending acknowledgment
- SEND_NACK: Sending negative acknowledgment
- ERROR: Link error state

#### Task 4.3: Implement Outstation Functions

**4.3.1: HandleResetLink()**
- Receive RESET LINK frame (0xC0)
- Reset FCB state (lastFCB = false, fcbInitialized = false)
- Clear buffers
- Send ACK (0x80)

**4.3.2: HandleResetUserProcess()**
- Receive RESET USER PROCESS frame (0xC1)
- Clear application layer buffers
- Do NOT reset FCB
- Send ACK (0x80)

**4.3.3: HandleTestLink()**
- Receive TEST LINK STATES frame (0xC2)
- Send ACK (0x80)
- No other action required

**4.3.4: HandleConfirmedUserData()**
- Receive USER DATA CONFIRMED frame (0x03)
- Check FCB duplicate detection:
  ```go
  if FCV == 1 {
      if !fcbInitialized {
          // First confirmed message
          fcbInitialized = true
          lastFCB = FCB
          processData()
          sendACK()
      } else {
          if FCB == lastFCB {
              // Duplicate frame
              sendACK()  // ACK but don't reprocess
          } else {
              // New frame
              lastFCB = FCB
              processData()
              sendACK()
          }
      }
  }
  ```
- If busy or error: Send NACK (0x81)

**4.3.5: HandleUnconfirmedUserData()**
- Receive USER DATA UNCONFIRMED frame (0x04)
- Process data
- No ACK sent

**4.3.6: HandleRequestLinkStatus()**
- Receive REQUEST LINK STATUS frame (0xC9)
- Send LINK STATUS response (0x8B)
- Include status information

**4.3.7: SendUnsolicitedData()**
- Send UNCONFIRMED USER DATA frame (0x84)
- DIR = 1 (to master)
- PRM = 0 (secondary initiating)
- No ACK expected

#### Task 4.4: Implement FCB Duplicate Detection
**File:** `pkg/link/outstation_link.go`

**FCB Validation:**
```go
type FCBValidator struct {
    lastFCB       bool
    initialized   bool
}

func (v *FCBValidator) ValidateAndUpdate(fcb bool, fcv bool) (isDuplicate bool) {
    if !fcv {
        return false // FCB not used
    }

    if !v.initialized {
        v.initialized = true
        v.lastFCB = fcb
        return false // First message
    }

    if fcb == v.lastFCB {
        return true // Duplicate
    }

    v.lastFCB = fcb
    return false // New message
}

func (v *FCBValidator) Reset() {
    v.initialized = false
    v.lastFCB = false
}
```

#### Task 4.5: Implement Response Generation
**File:** `pkg/link/outstation_link.go`

**Create Response Frames:**
```go
func (o *OutstationLink) sendACK() error {
    frame := NewFrame(
        DirectionOutstationToMaster,
        SecondaryFrame,
        FuncAck,
        o.remoteAddress,
        o.localAddress,
        nil, // No data
    )
    return o.transmit(frame)
}

func (o *OutstationLink) sendNACK() error {
    frame := NewFrame(
        DirectionOutstationToMaster,
        SecondaryFrame,
        FuncNack,
        o.remoteAddress,
        o.localAddress,
        nil,
    )
    return o.transmit(frame)
}

func (o *OutstationLink) sendLinkStatus() error {
    frame := NewFrame(
        DirectionOutstationToMaster,
        SecondaryFrame,
        FuncLinkStatusResponse,
        o.remoteAddress,
        o.localAddress,
        nil,
    )
    return o.transmit(frame)
}
```

### Phase 5: Integration with Existing Code

#### Task 5.1: Update Frame Builder
**File:** `pkg/link/frame.go`

**Add Convenience Methods:**
```go
// Helper functions for common frame types
func NewResetLinkFrame(dst, src uint16) *Frame
func NewTestLinkFrame(dst, src uint16) *Frame
func NewUserDataFrame(dst, src uint16, data []byte, confirmed bool) *Frame
func NewACKFrame(dst, src uint16) *Frame
func NewNACKFrame(dst, src uint16) *Frame
```

#### Task 5.2: Create Link Layer Factory
**File:** `pkg/link/factory.go` (NEW)

```go
type LinkLayerConfig struct {
    LocalAddress  uint16
    RemoteAddress uint16
    IsMaster      bool
    Timeout       time.Duration
    MaxRetries    int
}

func NewLinkLayer(config LinkLayerConfig) LinkLayer {
    if config.IsMaster {
        return NewMasterLink(config)
    }
    return NewOutstationLink(config)
}
```

### Phase 6: Testing

#### Task 6.1: Unit Tests
**File:** `pkg/link/master_link_test.go` (NEW)

**Test Cases:**
- Reset link success
- Reset link timeout retry
- FCB toggle on new messages
- FCB unchanged on retransmission
- Confirmed data with ACK
- Confirmed data with NACK retry
- Unconfirmed data transmission
- Max retries exceeded

#### Task 6.2: Unit Tests
**File:** `pkg/link/outstation_link_test.go` (NEW)

**Test Cases:**
- Handle reset link
- Handle test link
- FCB duplicate detection (accept new, reject duplicate)
- Send ACK for valid frame
- Send NACK when busy
- Process unconfirmed data
- Send unsolicited data

#### Task 6.3: Integration Tests
**File:** `pkg/link/link_integration_test.go` (NEW)

**Test Scenarios:**
- Master-Outstation complete handshake
- Link reset sequence
- Confirmed data exchange with retries
- Unsolicited message from outstation
- Timeout and recovery
- Multiple sequential confirmed messages (FCB toggling)

#### Task 6.4: Message Sequence Tests

**Sequence 1: Initial Link Establishment**
```
Master → Outstation: RESET LINK (0xC0)
Outstation → Master: ACK (0x80)
Verify: Outstation FCB reset
```

**Sequence 2: Confirmed Data Transfer**
```
Master → Outstation: USER DATA CONF, FCB=0 (0x53)
Outstation → Master: ACK (0x80)
Master → Outstation: USER DATA CONF, FCB=1 (0x73)
Outstation → Master: ACK (0x80)
Verify: FCB toggles correctly
```

**Sequence 3: Duplicate Detection**
```
Master → Outstation: USER DATA CONF, FCB=0 (0x53)
Outstation → Master: ACK (0x80)
[ACK lost, master retries]
Master → Outstation: USER DATA CONF, FCB=0 (0x53)
Outstation → Master: ACK (0x80)
Verify: Data processed only once
```

**Sequence 4: Retry After NACK**
```
Master → Outstation: USER DATA CONF, FCB=0 (0x53)
Outstation → Master: NACK (0x81)
[Delay]
Master → Outstation: USER DATA CONF, FCB=0 (0x53)
Outstation → Master: ACK (0x80)
Verify: Same FCB on retry
```

**Sequence 5: Unsolicited Response**
```
[Event at outstation]
Outstation → Master: USER DATA UNCONF (0x84)
Verify: No ACK sent
```

## Implementation Timeline

### Week 1: Foundation
- Phase 1: Complete constants and function codes
- Phase 2: Design link layer interface
- Initial code structure

### Week 2: Master Implementation
- Phase 3: Master link layer
  - State machine
  - FCB management
  - Retry logic
  - Response handling

### Week 3: Outstation Implementation
- Phase 4: Outstation link layer
  - State machine
  - FCB duplicate detection
  - Response generation
  - Unsolicited support

### Week 4: Integration and Testing
- Phase 5: Integration
- Phase 6: Testing
  - Unit tests
  - Integration tests
  - Message sequence validation

## Key Considerations

### 1. Thread Safety
- All link layer operations must be thread-safe
- Use mutexes for state access
- Channel-based communication between layers

### 2. Timing Requirements
- Typical link timeout: 1-5 seconds
- Retry intervals: 100-500ms
- Number of retries: 1-3 (configurable)
- Response time: < link timeout

### 3. Error Handling
- CRC errors: Discard silently
- Timeout: Retry with same FCB
- NACK: Retry after delay with same FCB
- Max retries: Alert upper layer, consider reset

### 4. Memory Management
- Reuse frame buffers where possible
- Limit outstanding frames
- Clear buffers on reset

### 5. Logging and Diagnostics
- Log all frame transmissions
- Log retries and failures
- Log FCB toggles
- Log state transitions

## Dependencies

### Internal
- `pkg/link/frame.go` - Frame structure (EXISTS)
- `pkg/link/crc.go` - CRC functions (EXISTS)
- `pkg/link/constants.go` - Constants (NEEDS UPDATE)

### External
- `time` - Timeouts and timers
- `sync` - Mutexes and synchronization
- `context` - Cancellation and lifecycle
- `errors` - Error definitions

## Success Criteria

1. **Functional Requirements:**
   - Master can initiate all link layer functions
   - Outstation responds correctly to all master requests
   - FCB duplicate detection works correctly
   - Timeout and retry mechanism functions properly
   - Unsolicited messages transmitted successfully

2. **Performance Requirements:**
   - Response time within timeout limits
   - No memory leaks
   - Efficient frame processing

3. **Quality Requirements:**
   - 100% test coverage for link layer functions
   - All DNP3 specification sequences work correctly
   - Thread-safe operation
   - Comprehensive error handling

4. **Documentation Requirements:**
   - API documentation
   - Usage examples
   - Sequence diagrams
   - Troubleshooting guide

## References

1. DNP3 Specification - Data Link Layer
2. IEEE 1815-2012 Standard
3. Architecture document: `dnp3_link_layer_functions.txt`
4. Existing implementation: `pkg/link/frame.go`

## Appendix: Control Byte Reference

### Master to Outstation (PRM=1, DIR=1)
- `0xC0` - RESET LINK
- `0xC1` - RESET USER PROCESS
- `0xC2` - TEST LINK STATES
- `0x53` - USER DATA CONF (FCB=0)
- `0x73` - USER DATA CONF (FCB=1)
- `0x44` - USER DATA UNCONF
- `0xC9` - REQUEST LINK STATUS

### Outstation to Master (PRM=0, DIR=0)
- `0x80` - ACK
- `0x81` - NACK
- `0x84` - USER DATA UNCONF (unsolicited)
- `0x8B` - LINK STATUS
- `0x8E` - LINK NOT FUNCTIONING
- `0x8F` - LINK NOT USED
