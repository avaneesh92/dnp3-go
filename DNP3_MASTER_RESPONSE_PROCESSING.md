# DNP3 Master Response and Unsolicited Frame Processing

This document details how a DNP3 master processes both solicited responses (replies to requests) and unsolicited responses (event notifications from outstations) from reception at the channel layer through to measurement data extraction and callback invocation.

## Overview

The DNP3 master handles two types of incoming frames:
1. **Solicited Responses** - Responses to master-initiated requests (READ, OPERATE, etc.)
2. **Unsolicited Responses** - Event notifications initiated by outstations

Both follow the same initial processing path but differ in correlation and handling logic.

---

## Architecture Overview

The master processes responses through the same four-layer stack as the outstation:

1. **Physical/Channel Layer** - Raw data reception from TCP/UDP
2. **Link Layer** - Frame parsing, CRC verification, and routing
3. **Transport Layer** - Segment reassembly
4. **Application Layer** - APDU parsing, correlation, and measurement extraction

**Key Difference from Outstation:**
- Master uses **request-response correlation** to match responses with pending requests
- Master uses **task queue** to manage scan operations and commands
- Master processes **measurement data** and invokes callbacks

---

## Layer 1-3: Common Processing Path

The initial processing (Layers 1-3) is identical to the outstation processing documented in [DNP3_OUTSTATION_FRAME_PROCESSING.md](DNP3_OUTSTATION_FRAME_PROCESSING.md):

1. **Channel Layer**: `readLoop()` receives raw bytes
2. **Link Layer**: `link.Parse()` validates and parses frame
3. **Transport Layer**: `Reassembler.Process()` reassembles segments
4. **Router**: Routes to master session based on destination address

The master session receives the complete APDU through the same `OnReceive()` interface.

---

## Layer 4: Application Layer - Master Processing

### Entry Point: Master Session Receive Handler
**File**: [pkg/master/session.go:30-49](pkg/master/session.go#L30-L49)

```
Session.OnReceive(frame) → Transport.Receive() → master.onReceiveAPDU()
```

**Key Functions:**
- **`session.OnReceive(frame)`** (session.go:30): Frame reception entry point
  - Logs received frame details
  - Passes frame.UserData to transport layer
  - Handles transport errors gracefully (auto-recovery)
  - Returns nil for incomplete segments
  - Passes complete APDU to master's handler

### APDU Processing Entry Point
**File**: [pkg/master/master.go:233-266](pkg/master/master.go#L233-L266)

The master's `onReceiveAPDU()` method is the central processing hub:

```
onReceiveAPDU(data) → Parse APDU → Update IIN → Send to pending channel → Process measurements
```

**Key Functions:**
- **`onReceiveAPDU(data)`** (master.go:234): Main APDU processing
  - Parses APDU using `app.Parse()`
  - Logs APDU details for debugging
  - Updates last IIN (Internal Indication) state
  - Sends IIN to callbacks for monitoring
  - Routes APDU to pending response channel (for correlation)
  - Processes measurement data if present
  - Handles both solicited and unsolicited responses

---

## Solicited Response Processing

Solicited responses are replies to master-initiated requests. The master correlates responses with pending requests using a response channel.

### Request-Response Correlation
**File**: [pkg/master/master.go:268-287](pkg/master/master.go#L268-L287)

The master uses a **blocking response channel** for correlation:

```
sendAndWait(apdu, timeout) → Send APDU → Block on pendingResp channel → Return response
```

**Key Functions:**
- **`sendAndWait(apdu, timeout)`** (master.go:269): Send request and wait for response
  - Serializes APDU to bytes
  - Sends via `session.sendAPDU()`
  - Logs sent APDU
  - Blocks on `pendingResp` channel (buffered, size=1)
  - Implements timeout mechanism
  - Returns response or timeout error

**Response Channel Flow:**
```
Request Thread                    Receive Thread
     |                                  |
     v                                  v
sendAndWait()                    onReceiveAPDU()
     |                                  |
     v                                  v
Send APDU                         Parse response
     |                                  |
     v                                  v
Block on                          Send to
pendingResp <--------------------- pendingResp
     |
     v
Return response
```

**Correlation Mechanism:**
- Master maintains single `pendingResp` channel (capacity: 1)
- Only one request active at a time (synchronous model)
- Response matched by arriving on channel (no sequence matching needed)
- If channel full (no pending request), response is dropped with warning

### Task-Based Request Management
**File**: [pkg/master/master.go:152-195](pkg/master/master.go#L152-L195)

The master uses a **priority task queue** to manage operations:

```
Task Queue → processTasks() → Execute task → Send request → Wait for response → Callback
```

**Key Functions:**
- **`taskProcessor()`** (master.go:153): Main task processing loop
  - Runs as goroutine
  - Ticks every 100ms to check for ready tasks
  - Calls `processTasks()` each tick
  - Stops on context cancellation

- **`processTasks()`** (master.go:168): Execute ready tasks
  - Checks if master is enabled
  - Gets next ready task from priority queue
  - Invokes `OnTaskStart` callback
  - Executes task (which sends request and waits for response)
  - Invokes `OnTaskComplete` callback with result
  - Reschedules periodic scans

**Task Types:**
1. **IntegrityScanTask** - Class 0 (all static data)
2. **ClassScanTask** - Class 1/2/3 event scans
3. **RangeScanTask** - Specific object ranges
4. **CommandTask** - Control operations (SELECT, OPERATE, DIRECT OPERATE)

### Integrity Scan Example
**File**: [pkg/master/operations.go:168-177](pkg/master/operations.go#L168-L177)

Flow of an integrity scan (READ Class 0):

```
IntegrityScanTask.Execute() → performIntegrityScan() → Build READ request → sendAndWait() → Response
```

**Processing Steps:**
1. Build READ request for Class 0 (all static data)
   - Group 60, Variation 0 (any), No Range qualifier
2. Get next application sequence number (0-15)
3. Create request APDU with READ function code
4. Call `sendAndWait()` with configured timeout
5. Transport segments, link frames, channel write
6. Block waiting for response
7. Response received, parsed, measurements extracted
8. Callback invoked with data
9. Task completes

### Command Operations
**File**: [pkg/master/operations.go:291-339](pkg/master/operations.go#L291-L339)

Commands (SELECT/OPERATE or DIRECT OPERATE) follow two-phase processing:

**SELECT-OPERATE Pattern:**
```
performSelectAndOperate() → Send SELECT → Wait → Send OPERATE → Wait → Extract statuses
```

**Key Functions:**
- **`performSelectAndOperate(commands)`** (operations.go:292): Two-phase control
  - Phase 1: Send SELECT request
    - Validates control without executing
    - Waits for response with timeout
    - Checks response status
  - Phase 2: Send OPERATE request
    - Executes the validated control
    - Waits for response with timeout
    - Extracts command statuses
  - Returns array of command statuses (success/failure per point)

- **`performDirectOperate(commands)`** (operations.go:323): Single-phase control
  - Sends DIRECT OPERATE request
  - Waits for response with timeout
  - Extracts command statuses
  - Returns statuses array

---

## Unsolicited Response Processing

Unsolicited responses are event notifications sent by outstations without a master request. They bypass the request-response correlation mechanism.

### Detection and Routing
**File**: [pkg/master/master.go:233-266](pkg/master/master.go#L233-L266)

Unsolicited responses are identified by function code:

```
onReceiveAPDU() → Check function code → Route to pendingResp or handle directly
```

**Key Identification:**
- **Function Code**: `0x82` (FuncUnsolicitedResponse)
- **UNS Flag**: Set in application control byte
- **CON Flag**: Typically set (requires confirmation)

**Routing Logic:**
```go
// In onReceiveAPDU()
if apdu.FunctionCode == app.FuncUnsolicitedResponse {
    // Unsolicited - not correlated with request
    // Still sent to pendingResp channel if available
    // Measurement processing proceeds regardless
} else if apdu.FunctionCode == app.FuncResponse {
    // Solicited - correlated with pending request
    // Must be consumed by sendAndWait()
}
```

**Current Behavior:**
- Both solicited and unsolicited are sent to `pendingResp` channel
- If no pending request, warning logged: "Dropped response (no pending request)"
- Measurement processing occurs for both types
- Callbacks distinguish via `ResponseInfo.Unsolicited` flag

### Unsolicited Confirmation
**File**: [pkg/app/functions.go:8](pkg/app/functions.go#L8)

Unsolicited responses typically require confirmation:

**Confirmation Requirement:**
- CON flag set in APDU control byte
- Master should send CONFIRM (0x00) with matching sequence number
- **TODO**: Current implementation does not send CONFIRM (not yet implemented)

**Expected Flow (not yet implemented):**
```
Receive unsolicited → Process measurements → Send CONFIRM → Outstation clears event buffer
```

### Disabling/Enabling Unsolicited
**File**: [pkg/app/functions.go:28-29](pkg/app/functions.go#L28-L29)

Masters can control unsolicited responses:

**Function Codes:**
- `FuncEnableUnsolicited (0x14)` - Enable unsolicited for specified classes
- `FuncDisableUnsolicited (0x15)` - Disable unsolicited for specified classes

**Configuration:**
- `DisableUnsolOnStartup` - Master config option
- **TODO**: Currently not implemented (master.go:112)

---

## Measurement Data Processing

Both solicited and unsolicited responses containing measurements are processed identically.

### Measurement Extraction Entry Point
**File**: [pkg/master/master.go:260-263](pkg/master/master.go#L260-L263) and [pkg/master/measurements.go:8-54](pkg/master/measurements.go#L8-L54)

```
onReceiveAPDU() → Check for objects → processMeasurements() → Parse headers → Invoke callbacks
```

**Key Functions:**
- **`processMeasurements(apdu)`** (measurements.go:9): Main measurement processor
  - Creates ResponseInfo (unsolicited flag, FIR, FIN)
  - Invokes `OnBeginFragment` callback
  - Creates parser for object data
  - Loops through object headers
  - Routes to type-specific processors
  - Invokes `OnEndFragment` callback

**ResponseInfo Structure:**
```go
ResponseInfo {
    Unsolicited: bool  // True if FuncUnsolicitedResponse
    FIR:         bool  // First fragment
    FIN:         bool  // Final fragment
}
```

### Object Header Parsing
**File**: [pkg/app/parser.go:40-78](pkg/app/parser.go#L40-L78)

The parser extracts object headers from APDU data:

```
Parser.ReadObjectHeader() → Read group, variation, qualifier → Read range → Return header
```

**Object Header Structure:**
```
ObjectHeader {
    Group:      uint8         // Object group (1=Binary, 30=Analog, etc.)
    Variation:  uint8         // Variation (0=any, 1-255=specific)
    Qualifier:  QualifierCode // Range specification method
    Range:      Range         // Start-stop or count
}
```

**Qualifier Types:**
- **Start-Stop** (0x00-0x02): Index range (start to stop)
  - 8-bit: Qualifier8BitStartStop
  - 16-bit: Qualifier16BitStartStop
  - 32-bit: Qualifier32BitStartStop
- **Count** (0x07-0x09): Number of objects
  - 8-bit: Qualifier8BitCount
  - 16-bit: Qualifier16BitCount
  - 32-bit: Qualifier32BitCount
- **No Range** (0x06): Class assignments (Class 0/1/2/3)

### Type-Specific Processing
**File**: [pkg/master/measurements.go:36-50](pkg/master/measurements.go#L36-L50)

Based on object group, measurements are routed to specific processors:

```
switch header.Group {
    case GroupBinaryInput, GroupBinaryInputEvent:
        processBinaryObjects()
    case GroupAnalogInput, GroupAnalogInputEvent:
        processAnalogObjects()
    case GroupCounter, GroupCounterEvent:
        processCounterObjects()
}
```

**Object Group Categories:**

| Group | Name | Type | Description |
|-------|------|------|-------------|
| 1 | Binary Input | Static | Current binary states |
| 2 | Binary Input Event | Event | Binary state changes |
| 20 | Counter | Static | Counter values |
| 22 | Counter Event | Event | Counter changes |
| 30 | Analog Input | Static | Current analog values |
| 32 | Analog Input Event | Event | Analog value changes |

**Event vs Static:**
- **Static Data**: Current values (Class 0)
- **Event Data**: Changes with timestamps (Class 1/2/3)
- Events cleared from outstation buffer after confirmation

### Binary Object Processing
**File**: [pkg/master/measurements.go:56-69](pkg/master/measurements.go#L56-L69)

```
processBinaryObjects() → Parse count → Calculate size → Skip data → Invoke callback
```

**Processing Steps:**
1. Extract count from header range
2. Calculate object size based on variation
3. Skip object data (TODO: actual parsing not implemented)
4. Create empty values array (placeholder)
5. Invoke `ProcessBinary` callback with HeaderInfo and values

**HeaderInfo Structure:**
```go
HeaderInfo {
    Group:     uint8  // Object group number
    Variation: uint8  // Object variation number
    Qualifier: uint8  // Qualifier code
    IsEvent:   bool   // True if event group (2, 22, 32, etc.)
}
```

### Analog Object Processing
**File**: [pkg/master/measurements.go:71-84](pkg/master/measurements.go#L71-L84)

Same pattern as binary processing:
1. Extract count and size
2. Skip data (TODO: parsing not implemented)
3. Invoke `ProcessAnalog` callback

**Analog Variations:**
- **32-bit integer**: Variation 1 (5 bytes: 4 value + 1 flags)
- **16-bit integer**: Variation 2 (3 bytes: 2 value + 1 flags)
- **Float**: Variation 5 (5 bytes: 4 float + 1 flags)
- **Double**: Variation 6 (9 bytes: 8 double + 1 flags)

### Counter Object Processing
**File**: [pkg/master/measurements.go:86-99](pkg/master/measurements.go#L86-L99)

Same pattern:
1. Extract count and size
2. Skip data (TODO: parsing not implemented)
3. Invoke `ProcessCounter` callback

**Counter Variations:**
- **32-bit**: Variation 1 (4 bytes)
- **16-bit**: Variation 2 (2 bytes)
- **32-bit with flags**: Variation 5 (5 bytes)
- **16-bit with flags**: Variation 6 (3 bytes)

### Event Detection
**File**: [pkg/master/measurements.go:101-116](pkg/master/measurements.go#L101-L116)

Event groups are identified by group number:

```go
func isEventGroup(group uint8) bool {
    switch group {
    case GroupBinaryInputEvent:      // 2
    case GroupCounterEvent:          // 22
    case GroupAnalogInputEvent:      // 32
    case GroupDoubleBitBinaryEvent:  // 4
    case GroupFrozenCounterEvent:    // 23
    case GroupFrozenAnalogEvent:     // 33
    case GroupBinaryOutputEvent:     // 11
    case GroupAnalogOutputEvent:     // 42
        return true
    }
    return false
}
```

---

## IIN (Internal Indication) Processing

IIN bytes provide outstation status information included in all responses.

### IIN Structure
**File**: [pkg/types/iin.go](pkg/types/iin.go)

IIN consists of two bytes with bit flags:

```
IIN {
    IIN1: uint8  // First byte (device status)
    IIN2: uint8  // Second byte (operational status)
}
```

**IIN1 Flags (Device Status):**
- Bit 0: All stations broadcast received
- Bit 1: Class 1 events available
- Bit 2: Class 2 events available
- Bit 3: Class 3 events available
- Bit 4: Need time synchronization
- Bit 5: Local control mode
- Bit 6: Device trouble
- Bit 7: Device restart

**IIN2 Flags (Operational Status):**
- Bit 0: Function code not supported
- Bit 1: Object unknown
- Bit 2: Parameter error
- Bit 3: Event buffer overflow
- Bit 4: Already executing
- Bit 5: Configuration corrupt
- Bit 6: Reserved
- Bit 7: Reserved

### IIN Update and Callback
**File**: [pkg/master/master.go:243-249](pkg/master/master.go#L243-L249)

The master updates and reports IIN on every response:

```go
// Update IIN
if apdu.IsResponse() {
    m.stateMu.Lock()
    m.lastIIN = apdu.IIN
    m.stateMu.Unlock()
    m.callbacks.OnReceiveIIN(apdu.IIN)
}
```

**Usage:**
- Application monitors IIN via `OnReceiveIIN` callback
- Class 1/2/3 bits indicate event availability → trigger event scans
- Device trouble bit → alert monitoring system
- Need time flag → trigger time synchronization

---

## Complete Flow Diagrams

### Solicited Response Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. REQUEST INITIATION (Task Queue)                             │
│    [pkg/master/master.go, operations.go, tasks.go]             │
└─────────────────────────────────────────────────────────────────┘
              |
              v
    User calls ScanIntegrity() (operations.go:109)
              |
              v
    Create IntegrityScanTask
              |
              v
    Push to taskQueue with priority and time
              |
              v
    taskProcessor() wakes up (master.go:153)
              |
              v
    processTasks() - Get next ready task (master.go:168)
              |
              v
    Invoke OnTaskStart callback
              |
              v
    task.Execute() → performIntegrityScan() (operations.go:169)
              |
              ├─ Build READ request (Class 0)
              ├─ Get next sequence number
              └─ Create APDU
              |
              v
    sendAndWait(apdu, timeout) (master.go:269)
              |
              ├─ Serialize APDU
              ├─ session.sendAPDU()
              │      ├─ Transport.Send() → segments
              │      ├─ link.NewFrame() for each segment
              │      ├─ Frame.Serialize() → add CRCs
              │      └─ Channel.Write()
              |
              v
    Block on pendingResp channel
              |
              |
┌─────────────────────────────────────────────────────────────────┐
│ 2. RESPONSE RECEPTION (Layers 1-3)                             │
│    [Same as outstation: channel → link → transport → session]  │
└─────────────────────────────────────────────────────────────────┘
              |
              v
    session.OnReceive(frame) (session.go:30)
              |
              v
    Transport.Receive() → reassemble segments
              |
              v
    master.onReceiveAPDU(data) (master.go:234)
              |
              v
┌─────────────────────────────────────────────────────────────────┐
│ 3. RESPONSE PROCESSING                                          │
│    [pkg/master/master.go, measurements.go]                      │
└─────────────────────────────────────────────────────────────────┘
              |
              v
    app.Parse(data) (apdu.go:123)
    → Extract: Control, Function, IIN, Objects
              |
              v
    Update IIN state
              |
              v
    callbacks.OnReceiveIIN(apdu.IIN)
              |
              v
    Send apdu to pendingResp channel
              |
              v
    (sendAndWait unblocks with response)
              |
              v
    processMeasurements(apdu) (measurements.go:9)
              |
              ├─ ResponseInfo{Unsolicited: false}
              ├─ OnBeginFragment callback
              |
              v
    Parser.ReadObjectHeader() loop (parser.go:40)
              |
              ├─ Group 1/2 → processBinaryObjects()
              │                  └─ ProcessBinary callback
              │
              ├─ Group 30/32 → processAnalogObjects()
              │                   └─ ProcessAnalog callback
              │
              └─ Group 20/22 → processCounterObjects()
                                  └─ ProcessCounter callback
              |
              v
    OnEndFragment callback
              |
              v
    Return from sendAndWait() with response
              |
              v
    OnTaskComplete callback
              |
              v
    Reschedule periodic scans if needed
```

### Unsolicited Response Flow

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. OUTSTATION EVENT OCCURS                                      │
│    Outstation detects change → adds to event buffer             │
└─────────────────────────────────────────────────────────────────┘
              |
              v
    Outstation periodic check (if unsolicited enabled)
              |
              v
    Build unsolicited response APDU
    → FunctionCode = 0x82 (FuncUnsolicitedResponse)
    → UNS flag set
    → CON flag set (requires confirmation)
    → Include event objects (Group 2, 22, 32)
              |
              v
    Send to master
              |
              |
┌─────────────────────────────────────────────────────────────────┐
│ 2. MASTER RECEPTION (Layers 1-3)                               │
│    [Same path: channel → link → transport → session]           │
└─────────────────────────────────────────────────────────────────┘
              |
              v
    session.OnReceive(frame) (session.go:30)
              |
              v
    Transport.Receive() → reassemble
              |
              v
    master.onReceiveAPDU(data) (master.go:234)
              |
              v
┌─────────────────────────────────────────────────────────────────┐
│ 3. UNSOLICITED PROCESSING                                       │
│    [pkg/master/master.go, measurements.go]                      │
└─────────────────────────────────────────────────────────────────┘
              |
              v
    app.Parse(data) (apdu.go:123)
    → FunctionCode = 0x82 (UnsolicitedResponse)
    → UNS = true
    → CON = true
              |
              v
    Update IIN state
              |
              v
    callbacks.OnReceiveIIN(apdu.IIN)
              |
              v
    Attempt to send to pendingResp channel
    → If no pending request: Dropped (warning logged)
    → Does not block processing
              |
              v
    processMeasurements(apdu) (measurements.go:9)
              |
              ├─ ResponseInfo{Unsolicited: true, FIR, FIN}
              ├─ OnBeginFragment callback
              |
              v
    Parser.ReadObjectHeader() loop
              |
              ├─ Group 2 (Binary Event) → processBinaryObjects()
              │                              └─ ProcessBinary callback
              │
              ├─ Group 32 (Analog Event) → processAnalogObjects()
              │                               └─ ProcessAnalog callback
              │
              └─ Group 22 (Counter Event) → processCounterObjects()
                                               └─ ProcessCounter callback
              |
              v
    OnEndFragment callback
              |
              v
    [TODO: Should send CONFIRM with sequence number]
              |
              v
    Processing complete
```

---

## Sequence Number Management

### Application Sequence Numbers

**File**: [pkg/master/master.go:224-231](pkg/master/master.go#L224-L231)

The master maintains application sequence number (0-15):

```go
func (m *master) getNextSequence() uint8 {
    m.stateMu.Lock()
    defer m.stateMu.Unlock()
    seq := m.sequence
    m.sequence = (m.sequence + 1) & 0x0F
    return seq
}
```

**Usage:**
- Incremented for each request (4-bit counter, 0-15)
- Response must have matching sequence number
- Used for confirmation of unsolicited responses
- **Current limitation**: No sequence validation in response handling

---

## Task Queue and Priority Management

### Priority Levels
**File**: [pkg/master/tasks.go:17-22](pkg/master/tasks.go#L17-L22)

```
PriorityHigh   = 100  // Commands, one-time scans
PriorityNormal = 50   // Periodic scans
PriorityLow    = 10   // Background operations
```

### Task Scheduling
**File**: [pkg/master/operations.go:16-42](pkg/master/operations.go#L16-L42)

**Periodic Scans:**
```
AddIntegrityScan(period) → Create task → Push to queue → Schedule next run
```

**One-Time Operations:**
```
ScanIntegrity() → Create high-priority task → Push with immediate time
```

**Task Queue Behavior:**
- Priority-based: Higher priority tasks execute first
- Time-based: Ready time determines when task can execute
- Single-threaded: One task at a time (synchronous requests)
- Periodic reschedule: After completion, schedule next run

---

## Callback Interface

### Master Callbacks
**File**: [pkg/master/config.go](pkg/master/config.go)

The application implements these callbacks to receive data:

**Task Lifecycle:**
- `OnTaskStart(taskType, id)` - Task begins execution
- `OnTaskComplete(taskType, id, result)` - Task completes

**Fragment Processing:**
- `OnBeginFragment(info ResponseInfo)` - Response processing starts
- `OnEndFragment(info ResponseInfo)` - Response processing complete

**Measurement Data:**
- `ProcessBinary(header HeaderInfo, values []IndexedBinary)` - Binary data
- `ProcessAnalog(header HeaderInfo, values []IndexedAnalog)` - Analog data
- `ProcessCounter(header HeaderInfo, values []IndexedCounter)` - Counter data

**Status Monitoring:**
- `OnReceiveIIN(iin IIN)` - Internal indication updates

---

## Key Differences: Solicited vs Unsolicited

| Aspect | Solicited Response | Unsolicited Response |
|--------|-------------------|---------------------|
| **Trigger** | Master request | Outstation event |
| **Function Code** | 0x81 (Response) | 0x82 (UnsolicitedResponse) |
| **UNS Flag** | false | true |
| **CON Flag** | Optional | Typically true (requires CONFIRM) |
| **Correlation** | Matched via pendingResp channel | No correlation needed |
| **Sequence** | Matches request sequence | Independent sequence |
| **Confirmation** | Not required | Master should send CONFIRM |
| **Data Type** | Static or event (depends on request) | Event data only |
| **Classes** | Class 0/1/2/3 (as requested) | Class 1/2/3 events only |
| **Timing** | Response timeout enforced | Arrives asynchronously |
| **Buffer Impact** | No buffer change | Clears events after CONFIRM |

---

## Error Handling and Edge Cases

### Timeout Handling
**File**: [pkg/master/master.go:282-283](pkg/master/master.go#L282-L283)

```go
case <-time.After(timeout):
    return nil, ErrTimeout
```

**Behavior:**
- Request times out if no response received
- Task completes with failure status
- Callback invoked with TaskResultFailure
- Periodic scans reschedule regardless

### Response Channel Full
**File**: [pkg/master/master.go:254-257](pkg/master/master.go#L254-L257)

```go
select {
case m.pendingResp <- apdu:
default:
    m.logger.Warn("Master %s: Dropped response (no pending request)", m.config.ID)
}
```

**Scenarios:**
- Unsolicited arrives when no request pending
- Response arrives after timeout
- Multiple responses for single request (protocol error)

**Handling:**
- Response dropped (not queued)
- Warning logged
- Measurement processing still occurs

### Transport Errors
**File**: [pkg/master/session.go:35-40](pkg/master/session.go#L35-L40)

```go
if err != nil {
    s.master.logger.Debug("Master session %d: Transport error: %v", s.linkAddress, err)
    return nil // Don't propagate error, let transport layer recover
}
```

**Auto-Recovery:**
- Sequence errors: Discard partial, wait for FIR
- Buffer overflow: Reset reassembly, log error
- Missing FIR: Discard silently

### Connection State
**File**: [pkg/master/session.go:62-71](pkg/master/session.go#L62-L71)

**On Connection Established:**
- Reset transport layer
- Clear reassembly buffer
- Reset sequence counters

**On Connection Lost:**
- Reset transport layer
- Tasks may timeout
- Automatic reconnection (at channel level)

---

## TODO Items (Not Yet Implemented)

### Critical Missing Features:

1. **Unsolicited Confirmation** (measurements.go)
   - Master should send CONFIRM (0x00) to acknowledge unsolicited
   - Sequence number must match unsolicited response
   - Required for outstation to clear event buffer

2. **Disable Unsolicited on Startup** (master.go:112)
   - Send DisableUnsolicited (0x15) for Class 1/2/3
   - Recommended for master to control event flow
   - Currently not implemented

3. **Actual Object Parsing** (measurements.go:56-99)
   - Binary, analog, counter values not extracted
   - Currently skips data and returns empty arrays
   - Callbacks invoked but with no data

4. **Command Response Parsing** (operations.go:301-319)
   - Command status extraction not implemented
   - Always returns success
   - Should parse CommandStatus objects from response

5. **Sequence Number Validation**
   - No verification that response sequence matches request
   - Could cause correlation errors
   - Should validate in sendAndWait()

6. **Multi-Fragment Support**
   - Currently assumes single-fragment responses
   - FIR/FIN flags not used for reassembly
   - Large responses may fail

---

## Function Reference Quick Index

### Master Entry Points
- `master.Enable()` - [master.go:92](pkg/master/master.go#L92)
- `master.ScanIntegrity()` - [operations.go:109](pkg/master/operations.go#L109)
- `master.SelectAndOperate()` - [operations.go:248](pkg/master/operations.go#L248)
- `master.DirectOperate()` - [operations.go:270](pkg/master/operations.go#L270)

### Session and Reception
- `session.OnReceive()` - [session.go:30](pkg/master/session.go#L30)
- `master.onReceiveAPDU()` - [master.go:234](pkg/master/master.go#L234)

### Request-Response
- `master.sendAndWait()` - [master.go:269](pkg/master/master.go#L269)
- `session.sendAPDU()` - [session.go:74](pkg/master/session.go#L74)

### Task Processing
- `master.taskProcessor()` - [master.go:153](pkg/master/master.go#L153)
- `master.processTasks()` - [master.go:168](pkg/master/master.go#L168)

### Measurement Processing
- `master.processMeasurements()` - [measurements.go:9](pkg/master/measurements.go#L9)
- `master.processBinaryObjects()` - [measurements.go:57](pkg/master/measurements.go#L57)
- `master.processAnalogObjects()` - [measurements.go:72](pkg/master/measurements.go#L72)
- `master.processCounterObjects()` - [measurements.go:87](pkg/master/measurements.go#L87)

### Object Parsing
- `Parser.ReadObjectHeader()` - [parser.go:40](pkg/app/parser.go#L40)
- `app.Parse()` - [apdu.go:123](pkg/app/apdu.go#L123)

---

## Summary

The DNP3 master processes responses through a sophisticated task-based architecture:

1. **Task Queue** manages all operations with priority and timing
2. **Request-Response Correlation** uses blocking channel for synchronous communication
3. **Solicited Responses** are matched to pending requests and trigger callbacks
4. **Unsolicited Responses** arrive asynchronously, detected by function code 0x82
5. **Measurement Processing** extracts binary/analog/counter data via callbacks
6. **IIN Monitoring** provides outstation status on every response
7. **Error Handling** includes timeouts, transport recovery, and connection state management

Both solicited and unsolicited responses follow the same measurement extraction path, but differ in correlation and confirmation requirements. The master's synchronous request model (one pending request at a time) simplifies correlation but limits throughput for high-performance applications.
