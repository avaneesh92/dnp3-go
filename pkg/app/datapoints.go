package app

import (
	"encoding/binary"
	"math"
)

// Flag bits for data points
const (
	FlagOnline       uint8 = 0x01 // Point is online
	FlagRestart      uint8 = 0x02 // Device restart detected
	FlagCommLost     uint8 = 0x04 // Communication lost
	FlagRemoteForced uint8 = 0x08 // Value remotely forced
	FlagLocalForced  uint8 = 0x10 // Value locally forced
	FlagOverRange    uint8 = 0x20 // Value over range (analog)
	FlagReferenceErr uint8 = 0x40 // Reference error
	FlagState        uint8 = 0x80 // Binary state (1=ON, 0=OFF)
)

// BinaryInput represents a binary input data point (Group 1)
type BinaryInput struct {
	Value bool  // Binary state
	Flags uint8 // Status flags
}

// NewBinaryInput creates a binary input with given value
func NewBinaryInput(value bool) BinaryInput {
	flags := FlagOnline
	if value {
		flags |= FlagState
	}
	return BinaryInput{Value: value, Flags: flags}
}

// Serialize serializes binary input with flags (Group 1, Var 2)
func (b BinaryInput) Serialize() []byte {
	return []byte{b.Flags}
}

// ParseBinaryInput parses binary input with flags
func ParseBinaryInput(data []byte) BinaryInput {
	if len(data) < 1 {
		return BinaryInput{}
	}
	flags := data[0]
	return BinaryInput{
		Value: (flags & FlagState) != 0,
		Flags: flags,
	}
}

// BinaryInputEvent represents a binary input change event (Group 2)
type BinaryInputEvent struct {
	Value     bool   // Binary state
	Flags     uint8  // Status flags
	Timestamp uint64 // DNP3 time (ms since epoch), optional
}

// NewBinaryInputEvent creates a binary input event
func NewBinaryInputEvent(value bool, timestamp uint64) BinaryInputEvent {
	flags := FlagOnline
	if value {
		flags |= FlagState
	}
	return BinaryInputEvent{
		Value:     value,
		Flags:     flags,
		Timestamp: timestamp,
	}
}

// SerializeWithoutTime serializes event without timestamp (Group 2, Var 1)
func (e BinaryInputEvent) SerializeWithoutTime() []byte {
	return []byte{e.Flags}
}

// SerializeWithTime serializes event with absolute timestamp (Group 2, Var 2)
func (e BinaryInputEvent) SerializeWithTime() []byte {
	buf := make([]byte, 7)
	buf[0] = e.Flags
	// DNP3 time is 48-bit (6 bytes)
	timeBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(timeBuf, e.Timestamp)
	copy(buf[1:], timeBuf[:6])
	return buf
}

// AnalogInput represents an analog input data point (Group 30)
type AnalogInput struct {
	Value interface{} // int16, int32, float32, or float64
	Flags uint8       // Status flags
}

// NewAnalogInputInt32 creates an analog input with 32-bit integer value
func NewAnalogInputInt32(value int32) AnalogInput {
	return AnalogInput{
		Value: value,
		Flags: FlagOnline,
	}
}

// NewAnalogInputInt16 creates an analog input with 16-bit integer value
func NewAnalogInputInt16(value int16) AnalogInput {
	return AnalogInput{
		Value: value,
		Flags: FlagOnline,
	}
}

// NewAnalogInputFloat creates an analog input with float32 value
func NewAnalogInputFloat(value float32) AnalogInput {
	return AnalogInput{
		Value: value,
		Flags: FlagOnline,
	}
}

// NewAnalogInputDouble creates an analog input with float64 value
func NewAnalogInputDouble(value float64) AnalogInput {
	return AnalogInput{
		Value: value,
		Flags: FlagOnline,
	}
}

// Serialize32Bit serializes analog input as 32-bit integer with flag (Group 30, Var 1)
func (a AnalogInput) Serialize32Bit() []byte {
	buf := make([]byte, 5)
	buf[0] = a.Flags

	var val int32
	switch v := a.Value.(type) {
	case int32:
		val = v
	case int16:
		val = int32(v)
	case float32:
		val = int32(v)
	case float64:
		val = int32(v)
	}

	binary.LittleEndian.PutUint32(buf[1:], uint32(val))
	return buf
}

// Serialize16Bit serializes analog input as 16-bit integer with flag (Group 30, Var 2)
func (a AnalogInput) Serialize16Bit() []byte {
	buf := make([]byte, 3)
	buf[0] = a.Flags

	var val int16
	switch v := a.Value.(type) {
	case int16:
		val = v
	case int32:
		val = int16(v)
	case float32:
		val = int16(v)
	case float64:
		val = int16(v)
	}

	binary.LittleEndian.PutUint16(buf[1:], uint16(val))
	return buf
}

// SerializeFloat serializes analog input as float32 with flag (Group 30, Var 5)
func (a AnalogInput) SerializeFloat() []byte {
	buf := make([]byte, 5)
	buf[0] = a.Flags

	var val float32
	switch v := a.Value.(type) {
	case float32:
		val = v
	case float64:
		val = float32(v)
	case int32:
		val = float32(v)
	case int16:
		val = float32(v)
	}

	binary.LittleEndian.PutUint32(buf[1:], math.Float32bits(val))
	return buf
}

// SerializeDouble serializes analog input as float64 with flag (Group 30, Var 6)
func (a AnalogInput) SerializeDouble() []byte {
	buf := make([]byte, 9)
	buf[0] = a.Flags

	var val float64
	switch v := a.Value.(type) {
	case float64:
		val = v
	case float32:
		val = float64(v)
	case int32:
		val = float64(v)
	case int16:
		val = float64(v)
	}

	binary.LittleEndian.PutUint64(buf[1:], math.Float64bits(val))
	return buf
}

// ParseAnalogInput32Bit parses 32-bit analog input with flag
func ParseAnalogInput32Bit(data []byte) AnalogInput {
	if len(data) < 5 {
		return AnalogInput{}
	}
	return AnalogInput{
		Flags: data[0],
		Value: int32(binary.LittleEndian.Uint32(data[1:])),
	}
}

// ParseAnalogInput16Bit parses 16-bit analog input with flag
func ParseAnalogInput16Bit(data []byte) AnalogInput {
	if len(data) < 3 {
		return AnalogInput{}
	}
	return AnalogInput{
		Flags: data[0],
		Value: int16(binary.LittleEndian.Uint16(data[1:])),
	}
}

// ParseAnalogInputFloat parses float32 analog input with flag
func ParseAnalogInputFloat(data []byte) AnalogInput {
	if len(data) < 5 {
		return AnalogInput{}
	}
	return AnalogInput{
		Flags: data[0],
		Value: math.Float32frombits(binary.LittleEndian.Uint32(data[1:])),
	}
}

// ParseAnalogInputDouble parses float64 analog input with flag
func ParseAnalogInputDouble(data []byte) AnalogInput {
	if len(data) < 9 {
		return AnalogInput{}
	}
	return AnalogInput{
		Flags: data[0],
		Value: math.Float64frombits(binary.LittleEndian.Uint64(data[1:])),
	}
}

// AnalogInputEvent represents an analog input change event (Group 32)
type AnalogInputEvent struct {
	Value     interface{} // int16, int32, float32, or float64
	Flags     uint8       // Status flags
	Timestamp uint64      // DNP3 time (ms since epoch), optional
}

// NewAnalogInputEventInt32 creates an analog input event with 32-bit value
func NewAnalogInputEventInt32(value int32, timestamp uint64) AnalogInputEvent {
	return AnalogInputEvent{
		Value:     value,
		Flags:     FlagOnline,
		Timestamp: timestamp,
	}
}

// SerializeInt32WithTime serializes as 32-bit integer with time (Group 32, Var 3)
func (e AnalogInputEvent) SerializeInt32WithTime() []byte {
	buf := make([]byte, 11)
	buf[0] = e.Flags

	var val int32
	if v, ok := e.Value.(int32); ok {
		val = v
	}
	binary.LittleEndian.PutUint32(buf[1:], uint32(val))

	// DNP3 timestamp is 48-bit (6 bytes)
	binary.LittleEndian.PutUint64(buf[5:], e.Timestamp)
	return buf[:11]
}

// Counter represents a counter data point (Group 20)
type Counter struct {
	Value uint32 // Counter value
	Flags uint8  // Status flags
}

// NewCounter creates a counter with given value
func NewCounter(value uint32) Counter {
	return Counter{
		Value: value,
		Flags: FlagOnline,
	}
}

// Serialize32Bit serializes counter as 32-bit with flag (Group 20, Var 1)
func (c Counter) Serialize32Bit() []byte {
	buf := make([]byte, 5)
	buf[0] = c.Flags
	binary.LittleEndian.PutUint32(buf[1:], c.Value)
	return buf
}

// Serialize16Bit serializes counter as 16-bit with flag (Group 20, Var 2)
func (c Counter) Serialize16Bit() []byte {
	buf := make([]byte, 3)
	buf[0] = c.Flags
	binary.LittleEndian.PutUint16(buf[1:], uint16(c.Value))
	return buf
}

// ParseCounter32Bit parses 32-bit counter with flag
func ParseCounter32Bit(data []byte) Counter {
	if len(data) < 5 {
		return Counter{}
	}
	return Counter{
		Flags: data[0],
		Value: binary.LittleEndian.Uint32(data[1:]),
	}
}

// ParseCounter16Bit parses 16-bit counter with flag
func ParseCounter16Bit(data []byte) Counter {
	if len(data) < 3 {
		return Counter{}
	}
	return Counter{
		Flags: data[0],
		Value: uint32(binary.LittleEndian.Uint16(data[1:])),
	}
}

// BinaryOutput represents a binary output status (Group 10)
type BinaryOutput struct {
	Value bool  // Output state
	Flags uint8 // Status flags
}

// NewBinaryOutput creates a binary output with given value
func NewBinaryOutput(value bool) BinaryOutput {
	flags := FlagOnline
	if value {
		flags |= FlagState
	}
	return BinaryOutput{Value: value, Flags: flags}
}

// Serialize serializes binary output with flags (Group 10, Var 2)
func (b BinaryOutput) Serialize() []byte {
	return []byte{b.Flags}
}

// ParseBinaryOutput parses binary output with flags
func ParseBinaryOutput(data []byte) BinaryOutput {
	if len(data) < 1 {
		return BinaryOutput{}
	}
	flags := data[0]
	return BinaryOutput{
		Value: (flags & FlagState) != 0,
		Flags: flags,
	}
}

// AnalogOutputStatus represents analog output status (Group 40)
type AnalogOutputStatus struct {
	Value interface{} // int16, int32, float32, or float64
	Flags uint8       // Status flags
}

// NewAnalogOutputStatusInt32 creates an analog output status with 32-bit value
func NewAnalogOutputStatusInt32(value int32) AnalogOutputStatus {
	return AnalogOutputStatus{
		Value: value,
		Flags: FlagOnline,
	}
}

// Serialize32Bit serializes as 32-bit integer with flag (Group 40, Var 1)
func (a AnalogOutputStatus) Serialize32Bit() []byte {
	buf := make([]byte, 5)
	buf[0] = a.Flags

	var val int32
	if v, ok := a.Value.(int32); ok {
		val = v
	}
	binary.LittleEndian.PutUint32(buf[1:], uint32(val))
	return buf
}
