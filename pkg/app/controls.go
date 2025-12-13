package app

import (
	"encoding/binary"
	"fmt"
)

// Control codes for CROB
const (
	ControlCodeNUL           uint8 = 0x00 // No operation
	ControlCodePulseOn       uint8 = 0x01 // Pulse on
	ControlCodePulseOff      uint8 = 0x02 // Pulse off
	ControlCodeLatchOn       uint8 = 0x03 // Latch on
	ControlCodeLatchOff      uint8 = 0x04 // Latch off
	ControlCodeCloseOn       uint8 = 0x41 // Close pulse on
	ControlCodeTripOff       uint8 = 0x81 // Trip pulse off
)

// Trip/Close codes
const (
	TripCloseCodeNUL   uint8 = 0x00
	TripCloseCodeClose uint8 = 0x01
	TripCloseCodeTrip  uint8 = 0x02
)

// Queue and Clear bits
const (
	ControlQueueBit uint8 = 0x01 // Queue control
	ControlClearBit uint8 = 0x02 // Clear queue
)

// OpType defines operation type bits
const (
	OpTypeNUL        uint8 = 0x00
	OpTypePulseOn    uint8 = 0x01
	OpTypePulseOff   uint8 = 0x02
	OpTypeLatchOn    uint8 = 0x03
	OpTypeLatchOff   uint8 = 0x04
)

// CROB represents a Control Relay Output Block (Group 12, Var 1)
type CROB struct {
	ControlCode uint8  // Control operation code
	Count       uint8  // Number of times to execute
	OnTime      uint32 // On duration in milliseconds
	OffTime     uint32 // Off duration in milliseconds
	Status      uint8  // Status code (in responses)
}

// NewCROB creates a new CROB with specified parameters
func NewCROB(code uint8, count uint8, onTime, offTime uint32) CROB {
	return CROB{
		ControlCode: code,
		Count:       count,
		OnTime:      onTime,
		OffTime:     offTime,
		Status:      0,
	}
}

// NewLatchOn creates a CROB for latch on operation
func NewLatchOn() CROB {
	return NewCROB(ControlCodeLatchOn, 1, 0, 0)
}

// NewLatchOff creates a CROB for latch off operation
func NewLatchOff() CROB {
	return NewCROB(ControlCodeLatchOff, 1, 0, 0)
}

// NewPulseOn creates a CROB for pulse on operation
func NewPulseOn(onTime uint32) CROB {
	return NewCROB(ControlCodePulseOn, 1, onTime, 0)
}

// NewPulseOff creates a CROB for pulse off operation
func NewPulseOff(offTime uint32) CROB {
	return NewCROB(ControlCodePulseOff, 1, 0, offTime)
}

// Serialize converts CROB to wire format (11 bytes)
func (c CROB) Serialize() []byte {
	buf := make([]byte, 11)
	buf[0] = c.ControlCode
	buf[1] = c.Count
	binary.LittleEndian.PutUint32(buf[2:], c.OnTime)
	binary.LittleEndian.PutUint32(buf[6:], c.OffTime)
	buf[10] = c.Status
	return buf
}

// ParseCROB parses CROB from wire format
func ParseCROB(data []byte) (CROB, error) {
	if len(data) < 11 {
		return CROB{}, fmt.Errorf("CROB data too short: %d bytes", len(data))
	}

	return CROB{
		ControlCode: data[0],
		Count:       data[1],
		OnTime:      binary.LittleEndian.Uint32(data[2:]),
		OffTime:     binary.LittleEndian.Uint32(data[6:]),
		Status:      data[10],
	}, nil
}

// Control status codes (returned in CROB status field)
const (
	ControlStatusSuccess        uint8 = 0  // Operation successful
	ControlStatusTimeout        uint8 = 1  // Control operation timed out
	ControlStatusNoSelect       uint8 = 2  // No previous SELECT
	ControlStatusFormatError    uint8 = 3  // Format error in request
	ControlStatusNotSupported   uint8 = 4  // Control not supported
	ControlStatusAlreadyActive  uint8 = 5  // Control already active
	ControlStatusHardwareError  uint8 = 6  // Hardware error
	ControlStatusLocal          uint8 = 7  // Local mode, remote control disabled
	ControlStatusTooManyOps     uint8 = 8  // Too many operations requested
	ControlStatusNotAuthorized  uint8 = 9  // Not authorized
	ControlStatusAutomationInhibit uint8 = 10 // Automation inhibit
)

// StatusString returns a human-readable status message
func (c CROB) StatusString() string {
	switch c.Status {
	case ControlStatusSuccess:
		return "Success"
	case ControlStatusTimeout:
		return "Timeout"
	case ControlStatusNoSelect:
		return "No SELECT"
	case ControlStatusFormatError:
		return "Format Error"
	case ControlStatusNotSupported:
		return "Not Supported"
	case ControlStatusAlreadyActive:
		return "Already Active"
	case ControlStatusHardwareError:
		return "Hardware Error"
	case ControlStatusLocal:
		return "Local Mode"
	case ControlStatusTooManyOps:
		return "Too Many Operations"
	case ControlStatusNotAuthorized:
		return "Not Authorized"
	case ControlStatusAutomationInhibit:
		return "Automation Inhibit"
	default:
		return fmt.Sprintf("Unknown (%d)", c.Status)
	}
}

// String returns string representation of CROB
func (c CROB) String() string {
	return fmt.Sprintf("CROB{Code=%d, Count=%d, OnTime=%dms, OffTime=%dms, Status=%s}",
		c.ControlCode, c.Count, c.OnTime, c.OffTime, c.StatusString())
}

// AnalogOutputBlock represents an analog output command (Group 41)
type AnalogOutputBlock struct {
	Value  interface{} // int16, int32, float32, or float64
	Status uint8       // Status code (in responses)
}

// NewAnalogOutputBlockInt32 creates an analog output block with 32-bit integer value
func NewAnalogOutputBlockInt32(value int32) AnalogOutputBlock {
	return AnalogOutputBlock{
		Value:  value,
		Status: 0,
	}
}

// NewAnalogOutputBlockInt16 creates an analog output block with 16-bit integer value
func NewAnalogOutputBlockInt16(value int16) AnalogOutputBlock {
	return AnalogOutputBlock{
		Value:  value,
		Status: 0,
	}
}

// NewAnalogOutputBlockFloat creates an analog output block with float32 value
func NewAnalogOutputBlockFloat(value float32) AnalogOutputBlock {
	return AnalogOutputBlock{
		Value:  value,
		Status: 0,
	}
}

// SerializeInt32 serializes as 32-bit integer (Group 41, Var 1)
func (a AnalogOutputBlock) SerializeInt32() []byte {
	buf := make([]byte, 5)

	var val int32
	if v, ok := a.Value.(int32); ok {
		val = v
	}

	binary.LittleEndian.PutUint32(buf[0:], uint32(val))
	buf[4] = a.Status
	return buf
}

// SerializeInt16 serializes as 16-bit integer (Group 41, Var 2)
func (a AnalogOutputBlock) SerializeInt16() []byte {
	buf := make([]byte, 3)

	var val int16
	if v, ok := a.Value.(int16); ok {
		val = v
	}

	binary.LittleEndian.PutUint16(buf[0:], uint16(val))
	buf[2] = a.Status
	return buf
}

// SerializeFloat serializes as float32 (Group 41, Var 3)
func (a AnalogOutputBlock) SerializeFloat() []byte {
	buf := make([]byte, 5)

	var val float32
	if v, ok := a.Value.(float32); ok {
		val = v
	}

	binary.LittleEndian.PutUint32(buf[0:], uint32(val))
	buf[4] = a.Status
	return buf
}

// ParseAnalogOutputInt32 parses 32-bit analog output command
func ParseAnalogOutputInt32(data []byte) (AnalogOutputBlock, error) {
	if len(data) < 5 {
		return AnalogOutputBlock{}, fmt.Errorf("analog output data too short: %d bytes", len(data))
	}

	return AnalogOutputBlock{
		Value:  int32(binary.LittleEndian.Uint32(data[0:])),
		Status: data[4],
	}, nil
}

// ParseAnalogOutputInt16 parses 16-bit analog output command
func ParseAnalogOutputInt16(data []byte) (AnalogOutputBlock, error) {
	if len(data) < 3 {
		return AnalogOutputBlock{}, fmt.Errorf("analog output data too short: %d bytes", len(data))
	}

	return AnalogOutputBlock{
		Value:  int16(binary.LittleEndian.Uint16(data[0:])),
		Status: data[2],
	}, nil
}

// BuildCROBRequest builds a CROB control request for a specific point
func BuildCROBRequest(index uint16, crob CROB) []byte {
	builder := NewObjectBuilder()

	// Add object header for single CROB at specific index
	builder.AddHeader(GroupBinaryOutputCommand, 1, Qualifier8BitStartStop,
		StartStopRange{Start: uint32(index), Stop: uint32(index)})

	// Add CROB data
	builder.AddRawData(crob.Serialize())

	return builder.Build()
}

// BuildAnalogOutputRequest builds an analog output request for a specific point
func BuildAnalogOutputRequest(index uint16, value int32) []byte {
	builder := NewObjectBuilder()

	// Add object header for analog output command
	builder.AddHeader(GroupAnalogOutputCommand, 1, Qualifier8BitStartStop,
		StartStopRange{Start: uint32(index), Stop: uint32(index)})

	// Add analog output value
	ao := NewAnalogOutputBlockInt32(value)
	builder.AddRawData(ao.SerializeInt32())

	return builder.Build()
}
