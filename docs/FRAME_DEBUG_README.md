# DNP3 Frame Debugging

This document explains how to enable and use the frame debugging feature in the dnp3-go library.

## Overview

The frame debugging feature allows you to log all DNP3 frames sent and received at the link layer. This is useful for:
- Debugging communication issues
- Understanding the DNP3 protocol flow
- Analyzing frame structure and content
- Troubleshooting master-outstation interactions

## How to Enable Frame Debugging

There are two ways to enable frame debugging:

### Option 1: Environment Variable (Recommended)

Set the `DNP3_FRAME_DEBUG` environment variable to `true` before running your application:

**Linux/macOS:**
```bash
export DNP3_FRAME_DEBUG=true
./your-dnp3-application
```

**Windows (Command Prompt):**
```cmd
set DNP3_FRAME_DEBUG=true
your-dnp3-application.exe
```

**Windows (PowerShell):**
```powershell
$env:DNP3_FRAME_DEBUG="true"
.\your-dnp3-application.exe
```

### Option 2: Programmatic Control

You can enable or disable frame debugging programmatically in your Go code:

```go
import "avaneesh/dnp3-go/pkg/internal/logger"

func main() {
    // Enable frame debugging
    logger.SetFrameDebug(true)

    // Your application code here...

    // Disable frame debugging
    logger.SetFrameDebug(false)
}
```

## Output Format

When frame debugging is enabled, you'll see detailed output for each frame:

### Received Frame Example
```
[INFO] <<< FRAME RECEIVED [Channel: channel1] (18 bytes)
[INFO]     0000 | 05 64 0A 44 0A 00 01 00 8E 7E C0 C1 01 3C 02 06 | .d.D.....~...<..
[INFO]     0010 | 00 00                                           | ..
```

### Sent Frame Example
```
[INFO] >>> FRAME SENT [Channel: channel1] (22 bytes)
[INFO]     0000 | 05 64 0E 44 01 00 0A 00 23 6A C0 C1 81 80 00 3C | .d.D....#j.....<
[INFO]     0010 | 02 06 00 00 E4 0B                               | ......
```

### Output Fields Explained

- **Direction indicators:**
  - `<<<` = Frame received from remote device
  - `>>>` = Frame sent to remote device

- **Hex dump format:**
  - `0000 |` = Byte offset within the frame
  - `05 64 ...` = Hexadecimal representation of frame bytes
  - `|` = Separator
  - `.d.D....` = ASCII representation (non-printable characters shown as `.`)

## Frame Structure

DNP3 link layer frames have the following structure:

```
Byte 0-1:   Start bytes (0x05 0x64)
Byte 2:     Length (number of bytes following length byte, excluding CRCs)
Byte 3:     Control byte (direction, PRM/SEC, function code)
Byte 4-5:   Destination address (little-endian)
Byte 6-7:   Source address (little-endian)
Byte 8-9:   Header CRC
Byte 10+:   User data (with CRC every 16 bytes)
```

## Performance Considerations

Frame debugging adds minimal overhead when disabled (simple boolean check). However, when enabled:

- Each frame is formatted as a hex dump
- Log output is written to stdout
- For high-throughput applications, this may impact performance

**Recommendation:** Only enable frame debugging during development, testing, or troubleshooting.

## Example Usage

Here's a complete example showing how to use frame debugging:

```go
package main

import (
    "fmt"
    "time"

    "avaneesh/dnp3-go/pkg/dnp3"
    "avaneesh/dnp3-go/pkg/channel"
    "avaneesh/dnp3-go/pkg/internal/logger"
)

func main() {
    // Enable frame debugging
    logger.SetFrameDebug(true)

    // Create manager
    manager := dnp3.NewManager()
    defer manager.Shutdown()

    // Create TCP channel
    tcpConfig := channel.TCPChannelConfig{
        Address:      "127.0.0.1:20000",
        IsServer:     false,
        ReadTimeout:  30 * time.Second,
        WriteTimeout: 10 * time.Second,
    }

    tcpChannel, err := channel.NewTCPChannel(tcpConfig)
    if err != nil {
        panic(err)
    }

    // Add channel - now all frames will be logged
    ch, err := manager.AddChannel("channel1", tcpChannel)
    if err != nil {
        panic(err)
    }

    // Your application logic here...
    // All DNP3 frames will be logged with detailed hex dumps

    fmt.Println("Frame debugging is active. Check the logs for frame details.")
}
```

## Checking if Frame Debugging is Enabled

You can check the current state of frame debugging:

```go
import "avaneesh/dnp3-go/pkg/internal/logger"

if logger.IsFrameDebugEnabled() {
    fmt.Println("Frame debugging is enabled")
} else {
    fmt.Println("Frame debugging is disabled")
}
```

## Integration with Logging System

Frame debugging uses the existing logger infrastructure:

- Frames are logged at `INFO` level
- The global default logger is used
- You can customize the logger using `logger.SetDefault()`

## Troubleshooting

**Q: I set DNP3_FRAME_DEBUG=true but don't see any frames**

A: Make sure:
1. The environment variable is set before starting the application
2. Your logger level is set to INFO or lower
3. Communication is actually happening (check that the channel is open)

**Q: Frame debugging output is too verbose**

A: You can:
1. Disable frame debugging when not needed
2. Redirect output to a file: `./app > frames.log 2>&1`
3. Filter the output: `./app | grep FRAME`

## See Also

- DNP3 Protocol Specification
- [Link Layer Frame Format](https://en.wikipedia.org/wiki/DNP3#Link_layer)
- Logger package documentation: `pkg/internal/logger/logger.go`
