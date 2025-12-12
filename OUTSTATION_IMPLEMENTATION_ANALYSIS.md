# DNP3 Outstation Implementation Analysis

## Based on Correct Frame Exchange Logs (ex.txt)

This document analyzes the actual DNP3 protocol exchange captured from a working master-outstation communication and identifies what needs to be implemented in the Go outstation.

---

## Summary of Frame Exchange Pattern

The logs in `ex.txt` show the following sequence from a master's perspective:

### 1. Initial Handshake (Frames 1-2)
```
T -> 05 64 05 c0 0a 00 01 00 b1 ac         # RESET_LINK_STATES
R <- 05 64 05 00 01 00 0a 00 2e dd         # ACK
```

**Analysis:**
- Master (addr 0x0a) sends RESET_LINK_STATES to outstation (addr 0x01)
- Link layer function code: 0xc0 (PRI=1, FuncCode=0x00 = RESET_LINK_STATES)
- Outstation responds with ACK (FC=0x00)
- **Current Implementation:** ✓ Correctly handles this in `outstation.go:294`

---

### 2. Initial Class Scan (Frames 3-4)

**Master Request (Frame 3):**
```
05 64 14 c4 0a 00 01 00 8f ed    # Link header
c0 c0                             # App: FIR+FIN, Seq=0
01                                # Function: READ (0x01)
3c 01 06                          # Class 1 (Group 60, Var 1, Qual 06)
3c 02 06                          # Class 2 (Group 60, Var 2, Qual 06)
3c 03 06                          # Class 3 (Group 60, Var 3, Qual 06)
3c 04 06                          # Class 4 (Group 60, Var 4, Qual 06)
9c 09                             # CRC
```

**Outstation Response (Frame 4):**
```
05 64 1a 44 01 00 0a 00 af 5d    # Link header
c1 c0                             # App: FIR+FIN, Seq=0
81                                # Function: RESPONSE (0x81)
90 00                             # IIN: 0x90 0x00 (IIN1.7=Restart, IIN1.4=TimeSync needed)
01 02 00 00 00 01                 # Group 1, Var 2, Start-Stop 8-bit: 0-0 (1 object)
1e 05                             # Flags: 0x1e, Quality: 0x05
00 00 00                          # Timestamp (3 bytes shown, actual format depends)
af 6c                             # CRC
01 00 00 00 00                    # Group 1 continuation or error
f9 dc                             # CRC
```

**Analysis:**
- Master reads Class 1/2/3/4 data (static + events)
- Outstation returns:
  - IIN1 = 0x90: Bit 7 (Device Restart), Bit 4 (Need Time)
  - IIN2 = 0x00
  - Object: Group 1 Variation 2 (Binary Input with Status)
- **Current Implementation:** ✗ MISSING - `handleRead()` returns empty response

---

### 3. Time Synchronization (Frames 5-6)

**Master Request (Frame 5):**
```
05 64 12 c4 0a 00 01 00 56 86    # Link header
c1 c1                             # App: FIR+FIN, Seq=1
02                                # Function: WRITE (0x02)
32 01 07 01                       # Group 50, Var 1, Count=1 (Time and Date)
ca c9 65 15 9b 01                 # 6-byte timestamp (DNP3 time format)
aa c9                             # CRC
```

**Outstation Response (Frame 6):**
```
05 64 0a 44 01 00 0a 00 6e 25    # Link header
c2 c1                             # App: FIR+FIN, Seq=1
81                                # Function: RESPONSE
80 00                             # IIN: 0x80 0x00 (Device Restart bit still set)
57 77                             # CRC
```

**Analysis:**
- Master writes current time (Group 50, Variation 1)
- DNP3 time format: 48-bit milliseconds since midnight Jan 1, 1970 UTC
- Outstation acknowledges with empty response
- **Current Implementation:** ✗ MISSING - No WRITE handler implemented

---

### 4. Enable/Disable Unsolicited or Control Operation (Frames 7-8)

**Master Request (Frame 7):**
```
05 64 0e c4 0a 00 01 00 25 29    # Link header
c2 c2                             # App: FIR+FIN, Seq=2
02                                # Function: WRITE
50 01 00 07 07 00                 # Group 80, Var 1, Count (IIN manipulation)
f3 95                             # CRC
```

**Outstation Response (Frame 8):**
```
05 64 0a 44 01 00 0a 00 6e 25    # Link header
c3 c2                             # App: FIR+FIN, Seq=2
81                                # Function: RESPONSE
00 00                             # IIN: 0x00 0x00 (Restart cleared, time synced)
3f 45                             # CRC
```

**Analysis:**
- This appears to be a WRITE to Group 80 (Internal Indications)
- Purpose: Clear restart flag, confirm time sync
- After this, IIN becomes 0x00 0x00 (normal operation)
- **Current Implementation:** ✗ MISSING - No WRITE handler for Group 80

---

### 5. Periodic Event Scans (Frames 9+)

**Master Request (Frame 9, 11, 13, ...):**
```
c4 c4                             # App: Seq=4
01                                # Function: READ
3c 02 06                          # Class 2
3c 03 06                          # Class 3
3c 04 06                          # Class 4
```

**Outstation Response (Frame 10, 12, 14, ...):**
```
c5 c4                             # App: Response to Seq=4
81                                # Function: RESPONSE
00 00                             # IIN: 0x00 0x00 (no events, normal)
(no objects - empty response)
```

**Analysis:**
- Master polls Class 2/3/4 every second (event data)
- Outstation returns NULL response (no events pending)
- **Current Implementation:** ✗ MISSING - `handleRead()` doesn't build proper responses

---

### 6. Periodic Full Scan (Frame 47-48)

Every ~30 requests, master does a full Class 1/2/3/4 scan again (integrity poll).

---

## Critical Missing Implementations

### 1. **READ Request Handler** (HIGHEST PRIORITY)

**File:** `pkg/outstation/outstation.go:466`

**Current Code:**
```go
func (o *outstation) handleRead(apdu *app.APDU) error {
    o.logger.Debug("Outstation %s: Handling READ request", o.config.ID)

    // Build response with IIN
    iin := o.callbacks.GetApplicationIIN()

    // TODO: Build response data from database
    responseData := []byte{}

    response := app.NewResponseAPDU(apdu.Sequence, iin, responseData)
    return o.session.sendAPDU(response.Serialize())
}
```

**What's Missing:**
1. Parse requested objects from `apdu.Objects`
2. Identify which classes/groups/variations are being requested
3. Query the database for matching data
4. Serialize the response objects
5. Build proper object headers (Group, Variation, Qualifier, Range)
6. Include measurement values with flags and timestamps

**Required Implementation:**
```go
func (o *outstation) handleRead(apdu *app.APDU) error {
    // Parse request objects
    parser := app.NewParser(apdu.Objects)
    var responseObjects []byte

    for parser.HasMore() {
        header, err := parser.ReadObjectHeader()
        if err != nil {
            break
        }

        // Handle different object types
        switch header.Group {
        case app.GroupClass0Data:  // Class 0 - All static data
            responseObjects = append(responseObjects, o.buildStaticData()...)
        case app.GroupClass1Data:  // Class 1 events
            responseObjects = append(responseObjects, o.buildEventData(1)...)
        case app.GroupClass2Data:  // Class 2 events
            responseObjects = append(responseObjects, o.buildEventData(2)...)
        case app.GroupClass3Data:  // Class 3 events
            responseObjects = append(responseObjects, o.buildEventData(3)...)
        case app.GroupBinaryInput:  // Specific binary input request
            responseObjects = append(responseObjects, o.buildBinaryData(header)...)
        case app.GroupAnalogInput:  // Specific analog input request
            responseObjects = append(responseObjects, o.buildAnalogData(header)...)
        // ... more cases
        }
    }

    iin := o.callbacks.GetApplicationIIN()
    response := app.NewResponseAPDU(apdu.Sequence, iin, responseObjects)
    return o.session.sendAPDU(response.Serialize())
}
```

---

### 2. **WRITE Request Handler** (HIGH PRIORITY)

**What's Missing:**
- No WRITE function code handler exists
- Needed for time synchronization (Group 50 Var 1)
- Needed for IIN clearing (Group 80)

**Required Implementation:**
```go
func (o *outstation) handleWrite(apdu *app.APDU) error {
    parser := app.NewParser(apdu.Objects)

    for parser.HasMore() {
        header, err := parser.ReadObjectHeader()
        if err != nil {
            break
        }

        switch header.Group {
        case app.GroupTimeAndDate:  // Group 50 - Time sync
            // Parse timestamp and update internal clock
            o.handleTimeSync(header, parser)

        case app.GroupInternalIndications:  // Group 80 - IIN control
            // Handle IIN bit manipulation
            o.handleIINWrite(header, parser)
        }
    }

    iin := o.callbacks.GetApplicationIIN()
    response := app.NewResponseAPDU(apdu.Sequence, iin, nil)
    return o.session.sendAPDU(response.Serialize())
}
```

**Add to onReceiveAPDU() switch:**
```go
case app.FuncWrite:
    return o.handleWrite(apdu)
```

---

### 3. **Object Response Building**

**Missing Functions:**
- `buildStaticData()` - Build all static measurements (Class 0)
- `buildEventData(class)` - Build event data for specified class
- `buildBinaryData(header)` - Build binary input responses
- `buildAnalogData(header)` - Build analog input responses

**Example - Binary Input Response:**
```go
func (o *outstation) buildBinaryData(header app.ObjectHeader) []byte {
    var buf bytes.Buffer

    // Get binary values from database
    values := o.database.GetBinaryRange(header.Range.Start, header.Range.Stop)

    // Write object header
    buf.WriteByte(app.GroupBinaryInput)        // Group 1
    buf.WriteByte(o.config.Database.Binary[0].StaticVariation)  // Variation
    buf.WriteByte(header.Qualifier)             // Qualifier code
    buf.WriteByte(uint8(header.Range.Start))    // Start index
    buf.WriteByte(uint8(header.Range.Stop))     // Stop index

    // Write each binary value
    for _, binary := range values {
        buf.WriteByte(binary.Flags)  // Quality flags
        // Add timestamp if variation includes it
    }

    return buf.Bytes()
}
```

---

### 4. **IIN (Internal Indication) Management**

**Current Behavior:**
- IIN comes from `callbacks.GetApplicationIIN()` which returns `{0, 0}`
- Should include status bits:
  - IIN1.7: Device Restart (set on startup, cleared after time sync)
  - IIN1.4: Need Time (set if time not synchronized)
  - IIN1.1-3: Class 1/2/3 events available
  - IIN2.3: Event buffer overflow

**Required:**
- Add IIN state management to outstation
- Set restart flag on startup
- Set time-needed flag until time sync received
- Set class event bits when events are buffered

---

### 5. **Time Synchronization**

**What's Needed:**
- Parse Group 50 Variation 1 (6-byte timestamp)
- DNP3 time format: milliseconds since Jan 1, 1970 00:00 UTC
- Update internal time reference
- Clear "Need Time" IIN bit
- Optionally invoke callback for time updates

---

## Implementation Priority

1. **CRITICAL: READ Response with Objects**
   - Implement object response building
   - Parse requested groups/classes
   - Return proper data from database

2. **CRITICAL: WRITE Handler**
   - Handle Group 50 (time sync)
   - Handle Group 80 (IIN control)

3. **HIGH: IIN Management**
   - Track restart state
   - Track time sync state
   - Track event availability

4. **MEDIUM: Event Data Building**
   - Build event responses from event buffer
   - Clear events after confirm

5. **LOW: Unsolicited Response**
   - Already stubbed, can be implemented later

---

## Testing Checklist

After implementation, verify:

- [ ] Master can read static data (Class 0)
- [ ] Master receives proper object headers
- [ ] Binary/Analog values are correctly formatted
- [ ] Time synchronization works
- [ ] IIN bits correctly reflect outstation state
- [ ] Event polling returns buffered events
- [ ] NULL responses when no events pending

---

## Code Files to Modify

1. `pkg/outstation/outstation.go`
   - Add `handleWrite()` method
   - Enhance `handleRead()` method
   - Add object building methods
   - Add IIN state management

2. `pkg/outstation/database.go`
   - Add `GetBinaryRange()`, `GetAnalogRange()`, etc.
   - Query methods for static data

3. `pkg/app/groups.go` (if doesn't exist, create)
   - Define Group 50 (Time and Date)
   - Define Group 80 (Internal Indications)

4. `pkg/app/objects.go` (enhance)
   - Add object serialization helpers
   - Build object headers

---

## Frame Structure Reference

### DNP3 Link Frame
```
0x05 0x64          - Start bytes
<len>              - Length (data bytes after this, excluding CRCs)
<ctrl>             - Control byte (DIR, PRM/SEC, FCV, FCB, Function)
<dest> <dest>      - Destination address (little-endian)
<src> <src>        - Source address (little-endian)
<crc> <crc>        - Header CRC
<data...>          - User data (16 bytes max per block)
<crc> <crc>        - Data CRC (every 16 bytes)
```

### DNP3 Application Layer (APDU)
```
<ctrl>             - Control byte (FIR, FIN, CON, UNS, SEQ[4])
<func>             - Function code
[<iin1> <iin2>]    - IIN bytes (responses only)
<objects...>       - Object headers and data
```

### Object Header Format
```
<group>            - Object group (1 byte)
<variation>        - Variation (1 byte)
<qualifier>        - Qualifier code (1 byte)
[<range>...]       - Range (size depends on qualifier)
<data...>          - Object data
```

---

## Next Steps

1. Create helper functions for object serialization
2. Implement `handleWrite()` for time sync
3. Enhance `handleRead()` to build real responses
4. Add IIN state tracking
5. Test with actual DNP3 master

