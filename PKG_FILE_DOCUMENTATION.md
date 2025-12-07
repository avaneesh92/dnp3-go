# DNP3-Go Package File Documentation

This document provides a brief description of each Go source file implemented in the `pkg` folder.

## pkg/types

### [time.go](pkg/types/time.go)
DNP3 timestamp type implementation. Defines `DNP3Time` type representing milliseconds since Unix epoch, with conversion functions to/from Go's `time.Time`.

### [quality.go](pkg/types/quality.go)
DNP3 quality flags implementation. Defines `Flags` type with helper methods to check and manipulate quality bits (online, restart, comm lost, forced, over range, reference error).

### [measurements.go](pkg/types/measurements.go)
DNP3 measurement types. Implements Binary, DoubleBitBinary, Analog, Counter, FrozenCounter, BinaryOutputStatus, AnalogOutputStatus, OctetString, TimeAndInterval, and their indexed variants with quality flags and timestamps.

### [commands.go](pkg/types/commands.go)
DNP3 command types and control codes. Defines CROB (Control Relay Output Block), analog output commands (Int32, Int16, Float32, Double64), command types, and command status enumeration with helper methods.

### [status.go](pkg/types/status.go)
DNP3 Internal Indication (IIN) bits. Re-exports IIN type with constants and helper methods for checking device status (class events, time sync needed, local control, device trouble, restart, errors).

## pkg/link

### [crc.go](pkg/link/crc.go)
DNP3 CRC-16 implementation. Provides functions to calculate, verify, append, add, and remove CRCs using DNP3 polynomial (0xA6BC). Handles 16-byte block CRC framing.

### [constants.go](pkg/link/constants.go)
DNP3 link layer constants. Defines start bytes, frame sizes, function codes, control field bits, errors, direction, and primary/secondary frame types.

### [frame.go](pkg/link/frame.go)
DNP3 link layer frame structure. Implements `Frame` type with header fields, control byte parsing/building, serialization to wire format with CRCs, parsing from wire format, and frame validation.

## pkg/internal/logger

### [logger.go](pkg/internal/logger/logger.go)
Logging interface and implementations. Provides `Logger` interface with Debug/Info/Warn/Error methods, `DefaultLogger` using standard log package, `NoOpLogger` for silent operation, and global default logger.

## pkg/transport

### [segment.go](pkg/transport/segment.go)
Transport layer segment structure. Defines `Segment` type with FIR/FIN/Sequence fields, header parsing/building, serialization, and data segmentation into 249-byte chunks.

### [reassembly.go](pkg/transport/reassembly.go)
Transport segment reassembly. Implements `Reassembler` to reconstruct APDUs from transport segments, handling sequence validation, FIR/FIN detection, and buffer overflow protection.

### [layer.go](pkg/transport/layer.go)
Transport layer implementation. Provides `Layer` type that combines reassembly for RX and segmentation for TX, managing transport sequence numbers.

## pkg/app

### [iin.go](pkg/app/iin.go)
Application layer IIN re-exports. Re-exports IIN type and constants from types package for convenience in application layer code.

### [objects.go](pkg/app/objects.go)
DNP3 object groups and variations. Defines object group constants (binary, analog, counter, etc.), variation constants, qualifier codes, object headers, range specifications, and class field helpers.

### [apdu.go](pkg/app/apdu.go)
Application Protocol Data Unit structure. Implements `APDU` type with control field (FIR/FIN/CON/UNS/Sequence), function code, IIN, object data, serialization/parsing, and helper constructors.

### [parser.go](pkg/app/parser.go)
Object header parser. Provides `Parser` for reading object headers from APDU data, parsing qualifiers and ranges (start-stop, count), reading bytes, and counting items in ranges.

### [functions.go](pkg/app/functions.go)
DNP3 function codes. Defines `FunctionCode` type with all DNP3 application functions (Read, Write, Select, Operate, DirectOperate, etc.) and helper methods to identify requests vs responses.

## pkg/channel

### [interface.go](pkg/channel/interface.go)
Physical channel interface. Defines `PhysicalChannel` interface for pluggable transports (Read/Write/Close/Statistics), `TransportStats` structure, and `ChannelState` enum.

### [statistics.go](pkg/channel/statistics.go)
Channel statistics tracking. Implements `Statistics` type with atomic counters for link frames, transport segments, CRC errors, and active sessions.

### [router.go](pkg/channel/router.go)
Session routing by address. Implements `Router` that routes link frames to appropriate sessions (master/outstation) based on link address, supporting multi-drop configurations.

### [tcp_channel.go](pkg/channel/tcp_channel.go)
TCP transport implementation. Implements `TCPChannel` as `PhysicalChannel` with client/server modes, automatic reconnection, DNP3 frame reading with length parsing, timeouts, and connection management.

### [udp_channel.go](pkg/channel/udp_channel.go)
UDP transport implementation. Implements `UDPChannel` as `PhysicalChannel` with client/server modes, datagram-based DNP3 frame transmission, peer address tracking for server responses.

### [channel.go](pkg/channel/channel.go)
Channel manager. Implements `Channel` type that manages physical channel, router, statistics, read/write loops, session management, and concurrent frame processing.

## pkg/dnp3

### [master.go](pkg/dnp3/master.go)
Master interface and types. Defines public `Master` interface with scanning operations, command operations, callbacks (`MasterCallbacks`, `SOEHandler`), configuration structures, and task types.

### [channel.go](pkg/dnp3/channel.go)
Channel interface wrapper. Implements public `Channel` interface wrapping internal channel implementation, providing AddMaster/AddOutstation methods and statistics.

### [manager.go](pkg/dnp3/manager.go)
DNP3 manager (root object). Implements `Manager` for creating/managing channels, factory methods for masters and outstations, and shutdown coordination.

### [interfaces.go](pkg/dnp3/interfaces.go)
Shared interfaces. Re-exports ClassField and class constants to avoid circular imports between dnp3 and master/outstation packages.

### [master_factory.go](pkg/dnp3/master_factory.go)
Master factory and wrapper. Implements `newMaster` factory function, `masterWrapper` for public interface, `masterCallbacksWrapper` for callback adaptation between public and internal types.

### [outstation.go](pkg/dnp3/outstation.go)
Outstation interface and types. Defines public `Outstation` interface, callbacks (`OutstationCallbacks`, `CommandHandler`), database configuration, point configurations, update types, and operate types.

### [outstation_factory.go](pkg/dnp3/outstation_factory.go)
Outstation factory and wrapper. Implements `newOutstation` factory function, configuration converters, `outstationWrapper`, `outstationCallbacksWrapper`, `updateHandlerWrapper`, and `UpdateBuilder` wrapper.

## pkg/master

### [session.go](pkg/master/session.go)
Master session link layer. Implements `session` type connecting master to channel, handling link frame reception, transport layer processing, and APDU transmission.

### [config.go](pkg/master/config.go)
Master configuration. Defines `MasterConfig` structure with identity, link addresses, timeouts, behavior flags, and callback interfaces (`MasterCallbacks`, `SOEHandler`).

### [master.go](pkg/master/master.go)
Master implementation core. Implements `master` type with task queue, scan management, enable/disable, task processor loop, APDU reception, send-and-wait mechanism, and sequence management.

### [measurements.go](pkg/master/measurements.go)
Measurement processing. Implements APDU measurement processing, object header parsing, binary/analog/counter object handling, event detection, and object size calculation.

### [operations.go](pkg/master/operations.go)
Master operations. Implements integrity scans, class scans, range scans, SELECT/OPERATE, DIRECT OPERATE commands, scan handle management, and READ request building.

### [tasks.go](pkg/master/tasks.go)
Task definitions. Defines `Task` interface, task types (`IntegrityScanTask`, `ClassScanTask`, `RangeScanTask`, `CommandTask`), `PeriodicScan` structure, and `ScanHandleImpl`.

## pkg/outstation

### [config.go](pkg/outstation/config.go)
Outstation configuration. Defines `OutstationConfig`, `DatabaseConfig`, point configuration types for all measurement types, callback interfaces, and operation types.

### [database.go](pkg/outstation/database.go)
Measurement database. Implements `Database` storing all point types with current values and configuration, update methods with event generation based on deadband/change detection.

### [outstation.go](pkg/outstation/outstation.go)
Outstation implementation. Implements `outstation` type with database, event buffer, session management, APDU handling (Read, Select, Operate, DirectOperate), update processor, and unsolicited response generator.

### [updates.go](pkg/outstation/updates.go)
Update data structure. Defines `Updates` type holding measurement update batch data for atomic application to database.

### [update_builder.go](pkg/outstation/update_builder.go)
Update builder. Implements `UpdateBuilder` with fluent API for building atomic measurement updates for all point types with event mode control.

### [event_buffer.go](pkg/outstation/event_buffer.go)
Event buffering. Implements `EventBuffer` managing class-based event storage (Class 1/2/3) with FIFO overflow handling, capacity limits, and event counting.

## pkg/internal/queue

### [priority_queue.go](pkg/internal/queue/priority_queue.go)
Priority queue implementation. Implements `PriorityQueue` using heap for time-based task scheduling with priority, supporting push/pop, next ready item lookup, and rescheduling.

---

**Total Files Documented:** 47 Go source files in the pkg folder
