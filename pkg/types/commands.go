package types

// ControlCode defines DNP3 control operations for binary outputs
type ControlCode uint8

// DNP3 Control Code values
const (
	ControlCodeNUL         ControlCode = 0x00 // No operation
	ControlCodePulseOn     ControlCode = 0x01 // Pulse output on
	ControlCodePulseOff    ControlCode = 0x02 // Pulse output off
	ControlCodeLatchOn     ControlCode = 0x03 // Latch output on
	ControlCodeLatchOff    ControlCode = 0x04 // Latch output off
	ControlCodeCloseOn     ControlCode = 0x41 // Close on with pulse
	ControlCodeTripOff     ControlCode = 0x81 // Trip off with pulse
)

// CROB (Control Relay Output Block) represents a binary control command
type CROB struct {
	OpType    ControlCode    // Type of control operation
	Count     uint8          // Number of times to repeat operation
	OnTimeMs  uint32         // Time the output is on (milliseconds)
	OffTimeMs uint32         // Time between operations (milliseconds)
	Status    CommandStatus  // Status of the command
}

// AnalogOutputInt32 represents a 32-bit integer analog output command (G41V1)
type AnalogOutputInt32 struct {
	Value  int32
	Status CommandStatus
}

// AnalogOutputInt16 represents a 16-bit integer analog output command (G41V2)
type AnalogOutputInt16 struct {
	Value  int16
	Status CommandStatus
}

// AnalogOutputFloat32 represents a 32-bit float analog output command (G41V3)
type AnalogOutputFloat32 struct {
	Value  float32
	Status CommandStatus
}

// AnalogOutputDouble64 represents a 64-bit float analog output command (G41V4)
type AnalogOutputDouble64 struct {
	Value  float64
	Status CommandStatus
}

// CommandType identifies the type of command
type CommandType uint8

const (
	CommandTypeCROB              CommandType = 0
	CommandTypeAnalogOutputInt32 CommandType = 1
	CommandTypeAnalogOutputInt16 CommandType = 2
	CommandTypeAnalogOutputFloat32 CommandType = 3
	CommandTypeAnalogOutputDouble64 CommandType = 4
)

// Command is a tagged union for all command types
type Command struct {
	Index uint16      // Index of the point to control
	Type  CommandType // Type of command
	Data  interface{} // Command data (CROB or AnalogOutput variant)
}

// CommandStatus indicates the result of a command operation
type CommandStatus uint8

// DNP3 Command Status values
const (
	CommandStatusSuccess           CommandStatus = 0  // Command accepted and executed
	CommandStatusTimeout           CommandStatus = 1  // Command timed out
	CommandStatusNoSelect          CommandStatus = 2  // No previous SELECT for this OPERATE
	CommandStatusFormatError       CommandStatus = 3  // Command format error
	CommandStatusNotSupported      CommandStatus = 4  // Command not supported
	CommandStatusAlreadyActive     CommandStatus = 5  // Command already in progress
	CommandStatusHardwareError     CommandStatus = 6  // Hardware error
	CommandStatusLocal             CommandStatus = 7  // In local mode, command rejected
	CommandStatusTooManyOps        CommandStatus = 8  // Too many operations requested
	CommandStatusNotAuthorized     CommandStatus = 9  // Not authorized
	CommandStatusAutomationInhibit CommandStatus = 10 // Automation inhibit prevents operation
	CommandStatusProcessingLimited CommandStatus = 11 // Processing limited
	CommandStatusOutOfRange        CommandStatus = 12 // Value out of range
	CommandStatusDownstreamLocal   CommandStatus = 13 // Downstream device in local mode
	CommandStatusAlreadyComplete   CommandStatus = 14 // Operation already complete
	CommandStatusBlocked           CommandStatus = 15 // Operation blocked
	CommandStatusCancelled         CommandStatus = 16 // Operation cancelled
	CommandStatusBlockedOther      CommandStatus = 17 // Operation blocked by other mask
	CommandStatusDownstreamFail    CommandStatus = 18 // Downstream device failure
	CommandStatusNonParticipating  CommandStatus = 126 // Device is non-participating
	CommandStatusUndefined         CommandStatus = 127 // Undefined error
)

// String returns a string representation of CommandStatus
func (s CommandStatus) String() string {
	switch s {
	case CommandStatusSuccess:
		return "Success"
	case CommandStatusTimeout:
		return "Timeout"
	case CommandStatusNoSelect:
		return "NoSelect"
	case CommandStatusFormatError:
		return "FormatError"
	case CommandStatusNotSupported:
		return "NotSupported"
	case CommandStatusAlreadyActive:
		return "AlreadyActive"
	case CommandStatusHardwareError:
		return "HardwareError"
	case CommandStatusLocal:
		return "Local"
	case CommandStatusTooManyOps:
		return "TooManyOps"
	case CommandStatusNotAuthorized:
		return "NotAuthorized"
	case CommandStatusAutomationInhibit:
		return "AutomationInhibit"
	case CommandStatusProcessingLimited:
		return "ProcessingLimited"
	case CommandStatusOutOfRange:
		return "OutOfRange"
	case CommandStatusDownstreamLocal:
		return "DownstreamLocal"
	case CommandStatusAlreadyComplete:
		return "AlreadyComplete"
	case CommandStatusBlocked:
		return "Blocked"
	case CommandStatusCancelled:
		return "Cancelled"
	case CommandStatusBlockedOther:
		return "BlockedOther"
	case CommandStatusDownstreamFail:
		return "DownstreamFail"
	case CommandStatusNonParticipating:
		return "NonParticipating"
	case CommandStatusUndefined:
		return "Undefined"
	default:
		return "Unknown"
	}
}

// IsSuccess returns true if the command was successful
func (s CommandStatus) IsSuccess() bool {
	return s == CommandStatusSuccess
}
