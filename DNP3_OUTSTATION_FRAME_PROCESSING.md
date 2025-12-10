# DNP3 Outstation Frame Processing Flow

This document details how a DNP3 frame is processed in the outstation implementation, from raw data reception at the channel layer through to application layer handling and response generation.

## Overview

The DNP3 outstation processes incoming frames through four main layers:
1. **Physical/Channel Layer** - Raw data reception and channel management
2. **Link Layer** - Frame parsing, CRC verification, and routing
3. **Transport Layer** - Segment reassembly and sequencing
4. **Application Layer** - APDU parsing and request handling

---

## Layer 1: Physical/Channel Layer

### Entry Point: Channel Read Loop
**File**: [pkg/channel/channel.go:134-176](pkg/channel/channel.go#L134-L176)

The channel's `readLoop()` function continuously reads data from the physical channel (TCP/UDP):

```
Channel.readLoop() → PhysicalChannel.Read() → Returns raw bytes
```

**Key Functions:**
- **`readLoop()`** (channel.go:134): Main read loop that runs as a goroutine
  - Continuously reads from physical channel in an infinite loop
  - Handles context cancellation for graceful shutdown
  - Logs errors but continues operation (fault tolerance)
  - Updates statistics for bad frames

**Processing Steps:**
1. Read raw bytes from physical channel (TCP/UDP socket)
2. Check for context cancellation (shutdown signal)
3. Handle read errors (log and continue)
4. Pass data to link layer for parsing

---

## Layer 2: Link Layer

### Frame Parsing
**File**: [pkg/link/frame.go:120-176](pkg/link/frame.go#L120-L176)

The `link.Parse()` function validates and parses the raw bytes into a structured frame:

```
link.Parse(data) → Validates frame → Verifies CRCs → Returns *Frame
```

**Key Functions:**
- **`Parse(data []byte)`** (frame.go:121): Main parsing function
  - Validates minimum frame size (≥10 bytes)
  - Checks start bytes (0x05, 0x64)
  - Extracts length field and validates it (must be ≥5)
  - Calculates expected frame size with CRC blocks
  - Verifies header CRC (10 bytes with 2-byte CRC)
  - Extracts header fields (control, destination, source addresses)
  - Verifies user data CRCs (every 16 bytes + 2-byte CRC)
  - Returns parsed frame and bytes consumed

- **`parseControl()`** (frame.go:65): Parses control byte into frame fields
  - Extracts function code (lower 4 bits)
  - Extracts direction bit (DIR) - master→outstation or outstation→master
  - Extracts primary/secondary bit (PRM)
  - Extracts Frame Count Valid (FCV) and Frame Count Bit (FCB)
  - Populates Frame structure with parsed values

**Frame Structure:**
```
Frame {
    Control:      uint8        // Control byte with DIR, PRM, FCB, FCV, Function
    Destination:  uint16       // Destination link address
    Source:       uint16       // Source link address
    Dir:          Direction    // Master→Outstation or Outstation→Master
    IsPrimary:    IsPrimary    // Primary or Secondary frame
    FCB:          bool         // Frame Count Bit
    FCV:          bool         // Frame Count Valid
    FunctionCode: FunctionCode // Link function (0x04=confirmed, 0x05=unconfirmed)
    UserData:     []byte       // Transport layer data (CRCs removed)
}
```

**CRC Verification:**
- Header CRC: Covers bytes 0-7 (start bytes through source address)
- Data CRCs: Every 16 bytes of user data followed by 2-byte CRC
- Function `VerifyCRC()` validates checksums
- Function `RemoveCRCs()` strips CRC bytes and validates each block

### Frame Routing
**File**: [pkg/channel/router.go:92-116](pkg/channel/router.go#L92-L116)

After parsing, the channel routes the frame to the appropriate session:

```
Channel.readLoop() → Router.Route(frame) → Session.OnReceive(frame)
```

**Key Functions:**
- **`Router.Route(frame)`** (router.go:94): Routes frame to correct session
  - Determines target address from frame direction and destination field
  - For master→outstation frames: routes to outstation at frame.Destination address
  - For outstation→master frames: routes to master at frame.Destination address
  - Looks up session in sessions map by address
  - Returns error if no session found
  - Calls session's `OnReceive()` method

**Routing Logic:**
```
if frame.Dir == DirectionMasterToOutstation:
    targetAddr = frame.Destination  // Route to outstation
else:
    targetAddr = frame.Destination  // Route to master
```

---

## Layer 3: Transport Layer

### Entry Point: Session Receive Handler
**File**: [pkg/outstation/outstation.go:285-305](pkg/outstation/outstation.go#L285-L305)

The outstation session's `OnReceive()` method handles incoming frames:

```
Session.OnReceive(frame) → Transport.Receive(UserData) → Returns APDU or nil
```

**Key Functions:**
- **`session.OnReceive(frame)`** (outstation.go:286): Main frame handler
  - Logs received frame details (source address, frame info)
  - Extracts user data from link frame
  - Passes to transport layer for reassembly
  - Handles transport errors gracefully (auto-recovery)
  - Returns nil for partial segments (waiting for more)
  - Passes complete APDU to application layer

### Segment Reassembly
**File**: [pkg/transport/layer.go:18-36](pkg/transport/layer.go#L18-L36) and [pkg/transport/reassembly.go:35-78](pkg/transport/reassembly.go#L35-L78)

The transport layer handles segment reassembly:

```
Transport.Receive(data) → Parses header → Reassembler.Process() → Complete APDU
```

**Key Functions:**
- **`Layer.Receive(data)`** (layer.go:18): Transport receive handler
  - Validates minimum size (≥1 byte header)
  - Parses transport header byte
  - Extracts FIR (First), FIN (Final), and sequence number
  - Creates Segment structure
  - Passes to reassembler for processing

- **`Reassembler.Process(segment)`** (reassembly.go:35): Segment reassembly
  - **FIR segment**: Resets buffer, starts new reassembly, sets expected sequence
  - **Non-FIR without FIR seen**: Silently discards (out of sync), waits for FIR
  - **Sequence check**: Validates segment.Seq matches expected sequence
  - **Sequence error**: Resets reassembly, returns nil (auto-recovery)
  - **Buffer overflow check**: Validates total size ≤ 2048 bytes
  - **Append data**: Adds segment data to buffer
  - **Update sequence**: Increments expected sequence (mod 64)
  - **FIN segment**: Returns complete APDU, resets state
  - **Partial**: Returns nil (waiting for more segments)

**Transport Header:**
```
Header Byte (1 byte):
  Bit 7:     FIN (Final segment)
  Bit 6:     FIR (First segment)
  Bits 5-0:  Sequence number (0-63)
```

**Reassembly States:**
- `inProgress=false`: Waiting for FIR segment to start
- `inProgress=true`: Actively reassembling multi-segment APDU
- Automatic recovery on sequence errors (discards partial, waits for next FIR)

---

## Layer 4: Application Layer

### APDU Parsing
**File**: [pkg/outstation/outstation.go:358-382](pkg/outstation/outstation.go#L358-L382)

Once a complete APDU is received, it's parsed and routed to handlers:

```
onReceiveAPDU(data) → app.Parse(data) → Routes to handler → Generates response
```

**Key Functions:**
- **`onReceiveAPDU(data)`** (outstation.go:359): APDU processing entry point
  - Calls `app.Parse()` to parse raw APDU bytes
  - Logs parsing errors
  - Logs received APDU details
  - Routes to appropriate handler based on function code
  - Returns error on unsupported functions

- **`app.Parse(data)`** (app/apdu.go:123): Parses APDU from bytes
  - Validates minimum size (≥2 bytes)
  - Extracts control byte (FIR, FIN, CON, UNS, sequence)
  - Extracts function code
  - For responses: extracts IIN (Internal Indication) bytes
  - Extracts object data (remaining bytes)
  - Returns parsed APDU structure

**APDU Structure:**
```
APDU {
    Control:      uint8        // Control byte
    FIR:          bool         // First fragment
    FIN:          bool         // Final fragment
    CON:          bool         // Confirmation required
    UNS:          bool         // Unsolicited response
    Sequence:     uint8        // Application sequence (0-15)
    FunctionCode: FunctionCode // Function code (READ, SELECT, OPERATE, etc.)
    IIN:          IIN          // Internal Indications (responses only)
    Objects:      []byte       // Raw object data
}
```

**Application Control Byte:**
```
Bit 7:     FIR (First fragment)
Bit 6:     FIN (Final fragment)
Bit 5:     CON (Confirmation required)
Bit 4:     UNS (Unsolicited)
Bits 3-0:  Sequence number (0-15)
```

### Request Handling
**File**: [pkg/outstation/outstation.go:368-436](pkg/outstation/outstation.go#L368-L436)

The outstation routes APDUs to specific handlers based on function code:

**Supported Function Codes:**
1. **READ (0x01)** → `handleRead()`
   - Reads data from outstation database
   - Returns current values and events
   - Builds response with requested objects

2. **SELECT (0x03)** → `handleSelect()`
   - Pre-validates control operation
   - Does not execute the operation
   - Returns success/failure status

3. **OPERATE (0x04)** → `handleOperate()`
   - Executes control after SELECT
   - Must be preceded by SELECT
   - Returns operation result

4. **DIRECT OPERATE (0x05)** → `handleDirectOperate()`
   - Executes control without SELECT
   - Single-step operation
   - Returns operation result

**Handler Functions:**

- **`handleRead(apdu)`** (outstation.go:385): Handles READ requests
  - Retrieves application IIN from callbacks
  - Builds response data from database (TODO: full implementation)
  - Creates response APDU with same sequence number
  - Sends response via session

- **`handleSelect(apdu)`** (outstation.go:399): Handles SELECT requests
  - Processes SELECT through command handler (TODO: full implementation)
  - Validates control command
  - Returns success/failure IIN
  - Sends response APDU

- **`handleOperate(apdu)`** (outstation.go:409): Handles OPERATE requests
  - Processes OPERATE through command handler (TODO: full implementation)
  - Executes control operation
  - Returns operation result IIN
  - Sends response APDU

- **`handleDirectOperate(apdu)`** (outstation.go:419): Handles DIRECT OPERATE
  - Processes direct operation through command handler (TODO: full implementation)
  - Validates and executes in one step
  - Returns operation result IIN
  - Sends response APDU

- **`sendErrorResponse(seq)`** (outstation.go:429): Handles unsupported functions
  - Sets IIN2 flag: NoFuncCodeSupport
  - Creates response with error indication
  - Maintains sequence number
  - Sends error response APDU

---

## Layer 5: Response Generation

### APDU Response Building
**File**: [pkg/outstation/outstation.go:385-436](pkg/outstation/outstation.go#L385-L436)

Response generation follows this flow:

```
Handler → Build Response APDU → Transport Segmentation → Link Framing → Channel Write
```

**Key Functions:**
- **`app.NewResponseAPDU(seq, iin, data)`** (app/apdu.go:49): Creates response
  - Sets function code to RESPONSE (0x81)
  - Copies request sequence number
  - Sets FIR=true, FIN=true (single fragment)
  - Sets CON=false (no confirmation needed)
  - Sets UNS=false (solicited response)
  - Includes IIN (Internal Indications)
  - Attaches object data

- **`APDU.Serialize()`** (app/apdu.go:99): Converts APDU to bytes
  - Builds control byte from flags and sequence
  - Writes function code
  - Writes IIN bytes (for responses)
  - Appends object data
  - Returns complete APDU bytes

### Response Transmission
**File**: [pkg/outstation/outstation.go:329-356](pkg/outstation/outstation.go#L329-L356)

The session handles sending the response:

```
session.sendAPDU(apdu) → Transport.Send() → Create Link Frames → Channel.Write()
```

**Key Functions:**
- **`session.sendAPDU(apdu)`** (outstation.go:330): Sends APDU response
  - Calls transport layer to segment APDU
  - Creates link frame for each segment
  - Sets link layer parameters:
    - Direction: OutstationToMaster
    - IsPrimary: SecondaryFrame
    - Function: UserDataUnconfirmed (0x05)
    - Destination: Remote master address
    - Source: Local outstation address
  - Serializes each frame (adds CRCs)
  - Writes each frame to channel

- **`Transport.Send(apdu)`** (transport/layer.go:40): Segments APDU
  - Segments data into ≤249 byte chunks
  - Sets FIR=true for first segment
  - Sets FIN=true for last segment
  - Assigns sequence numbers (0-63, wrapping)
  - Returns list of transport segments

- **`link.Frame.Serialize()`** (link/frame.go:84): Serializes link frame
  - Builds header (start bytes, length, control, addresses)
  - Calculates and appends header CRC
  - Adds user data with CRCs every 16 bytes
  - Returns complete frame ready for transmission

- **`Channel.Write(data)`** (channel/channel.go:210): Writes to channel
  - Validates channel is open
  - Creates write request
  - Queues request to write loop
  - Waits for completion
  - Returns write result

---

## Complete Flow Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│ 1. PHYSICAL/CHANNEL LAYER                                       │
│    [pkg/channel/channel.go]                                     │
└─────────────────────────────────────────────────────────────────┘
              │
              │ PhysicalChannel.Read(ctx) → raw bytes
              ▼
    Channel.readLoop() (line 134)
              │
              │ Raw data received
              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 2. LINK LAYER                                                   │
│    [pkg/link/frame.go]                                          │
└─────────────────────────────────────────────────────────────────┘
              │
              │ link.Parse(data)
              ▼
    Parse start bytes (0x05 0x64) (line 128)
              │
              ▼
    Validate length field (line 133)
              │
              ▼
    Verify header CRC (line 152)
              │
              ▼
    Extract: Control, Destination, Source (line 157-160)
              │
              ▼
    parseControl() - Extract DIR, PRM, FCB, FCV, FunctionCode (line 163)
              │
              ▼
    Remove and verify data CRCs (line 168)
              │
              │ Returns *Frame
              ▼
    Router.Route(frame) (router.go:94)
              │
              │ Determine target address
              ▼
    session.OnReceive(frame) (outstation.go:286)
              │
              │ Extract frame.UserData
              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 3. TRANSPORT LAYER                                              │
│    [pkg/transport/layer.go, reassembly.go]                      │
└─────────────────────────────────────────────────────────────────┘
              │
              │ Transport.Receive(UserData)
              ▼
    Parse transport header (layer.go:24)
    Extract: FIR, FIN, Sequence (bits 7,6,5-0)
              │
              │ Create Segment
              ▼
    Reassembler.Process(segment) (reassembly.go:35)
              │
              ├─── FIR? → Reset buffer, start reassembly
              │
              ├─── Sequence OK? → Add to buffer
              │            └─ NO → Discard, wait for FIR
              │
              └─── FIN? → Return complete APDU
                         └─ NO → Return nil (wait for more)
              │
              │ Returns []byte APDU (or nil if incomplete)
              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 4. APPLICATION LAYER                                            │
│    [pkg/outstation/outstation.go, pkg/app/apdu.go]             │
└─────────────────────────────────────────────────────────────────┘
              │
              │ outstation.onReceiveAPDU(data)
              ▼
    app.Parse(data) (apdu.go:123)
              │
              ▼
    Extract: Control byte (FIR,FIN,CON,UNS,Seq)
              │
              ▼
    Extract: FunctionCode
              │
              ▼
    If Response: Extract IIN (IIN1, IIN2)
              │
              ▼
    Extract: Objects (remaining bytes)
              │
              │ Returns *APDU
              ▼
    Route by FunctionCode (outstation.go:369)
              │
              ├─── READ (0x01) → handleRead()
              │                      │
              │                      ├─ Get IIN from callbacks
              │                      ├─ Build response from database
              │                      └─ Send response APDU
              │
              ├─── SELECT (0x03) → handleSelect()
              │                      │
              │                      ├─ Validate control command
              │                      └─ Send status response
              │
              ├─── OPERATE (0x04) → handleOperate()
              │                      │
              │                      ├─ Execute control operation
              │                      └─ Send result response
              │
              ├─── DIRECT OPERATE (0x05) → handleDirectOperate()
              │                              │
              │                              ├─ Validate and execute
              │                              └─ Send result response
              │
              └─── Other → sendErrorResponse()
                           └─ Send IIN with NoFuncCodeSupport
              │
              ▼
┌─────────────────────────────────────────────────────────────────┐
│ 5. RESPONSE TRANSMISSION                                        │
│    [pkg/outstation/outstation.go, pkg/transport/layer.go,       │
│     pkg/link/frame.go, pkg/channel/channel.go]                  │
└─────────────────────────────────────────────────────────────────┘
              │
              │ session.sendAPDU(responseAPDU)
              ▼
    APDU.Serialize() (apdu.go:99)
    → Build control byte
    → Append function code
    → Append IIN (if response)
    → Append objects
              │
              │ Returns []byte
              ▼
    Transport.Send(apdu) (layer.go:40)
    → Segment into ≤249 byte chunks
    → Set FIR on first, FIN on last
    → Assign sequences
              │
              │ Returns [][]byte (segments)
              ▼
    For each segment:
        link.NewFrame() (frame.go:27)
        → Direction: OutstationToMaster
        → IsPrimary: SecondaryFrame
        → Function: UserDataUnconfirmed
        → Dest: remoteAddr, Src: linkAddress
              │
              ▼
        Frame.Serialize() (frame.go:84)
        → Build header with addresses
        → Calculate header CRC
        → Add user data with CRCs
              │
              │ Returns []byte (frame)
              ▼
        Channel.Write(frameData) (channel.go:210)
        → Queue write request
        → writeLoop processes
        → PhysicalChannel.Write()
              │
              ▼
        [Transmitted on TCP/UDP socket]
```

---

## Key Data Structures Summary

### Link Frame
- **Size**: 10-292 bytes
- **Header**: 10 bytes (2 start + 1 len + 1 ctrl + 2 dst + 2 src + 2 CRC)
- **Data**: 0-250 bytes with CRC every 16 bytes
- **Control Fields**: DIR, PRM, FCB, FCV, FunctionCode

### Transport Segment
- **Header**: 1 byte (FIR, FIN, Seq)
- **Data**: Up to 249 bytes per segment
- **Sequence**: 0-63 (6-bit counter)

### Application APDU
- **Control**: 1 byte (FIR, FIN, CON, UNS, Seq)
- **Function**: 1 byte
- **IIN**: 2 bytes (responses only)
- **Objects**: Variable length
- **Sequence**: 0-15 (4-bit counter)

---

## Error Handling

### Link Layer Errors
- **Invalid start bytes**: Frame discarded, statistics updated
- **Invalid CRC**: Frame discarded, logged as bad frame
- **Invalid length**: Frame discarded, error logged

### Transport Layer Errors
- **Sequence error**: Reassembly reset, waits for next FIR (auto-recovery)
- **Missing FIR**: Segment discarded silently (out-of-sync recovery)
- **Buffer overflow**: Reassembly reset, error returned

### Application Layer Errors
- **Parse error**: Logged, no response sent
- **Unsupported function**: Error response with IIN2.NoFuncCodeSupport
- **Handler error**: Logged, error response sent

---

## Connection State Management

The outstation handles connection events:

**Connection Established** (outstation.go:318):
- Reset transport layer state
- Clear reassembly buffer
- Reset sequence counters

**Connection Lost** (outstation.go:324):
- Reset transport layer state
- Log connection loss
- Wait for reconnection

---

## Concurrency Model

### Goroutines
1. **Channel read loop**: Continuously reads from physical channel
2. **Channel write loop**: Serializes writes to physical channel
3. **Update processor**: Processes database updates
4. **Unsolicited processor**: Sends periodic unsolicited responses

### Thread Safety
- Router uses RWMutex for session map access
- Channel state protected by RWMutex
- Outstation state protected by RWMutex
- Write queue for serialized channel writes

---

## Function Reference Quick Index

### Channel Layer
- `Channel.readLoop()` - [channel.go:134](pkg/channel/channel.go#L134)
- `Channel.Write()` - [channel.go:210](pkg/channel/channel.go#L210)
- `Router.Route()` - [router.go:94](pkg/channel/router.go#L94)

### Link Layer
- `link.Parse()` - [frame.go:121](pkg/link/frame.go#L121)
- `Frame.parseControl()` - [frame.go:65](pkg/link/frame.go#L65)
- `Frame.Serialize()` - [frame.go:84](pkg/link/frame.go#L84)

### Transport Layer
- `Layer.Receive()` - [layer.go:18](pkg/transport/layer.go#L18)
- `Reassembler.Process()` - [reassembly.go:35](pkg/transport/reassembly.go#L35)
- `Layer.Send()` - [layer.go:40](pkg/transport/layer.go#L40)

### Application Layer
- `session.OnReceive()` - [outstation.go:286](pkg/outstation/outstation.go#L286)
- `outstation.onReceiveAPDU()` - [outstation.go:359](pkg/outstation/outstation.go#L359)
- `app.Parse()` - [apdu.go:123](pkg/app/apdu.go#L123)
- `handleRead()` - [outstation.go:385](pkg/outstation/outstation.go#L385)
- `session.sendAPDU()` - [outstation.go:330](pkg/outstation/outstation.go#L330)

---

## Summary

The DNP3 outstation processes frames through a well-defined layered architecture:

1. **Physical Layer** receives raw bytes from TCP/UDP
2. **Link Layer** validates frame structure, CRCs, and routes to sessions
3. **Transport Layer** reassembles segments into complete APDUs with auto-recovery
4. **Application Layer** parses requests, executes handlers, generates responses
5. **Response path** reverses the flow: APDU → segments → frames → wire

The implementation includes robust error handling, automatic recovery from transport errors, and concurrent processing with thread-safe data structures.
