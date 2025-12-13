package link

import "errors"

// DNP3 Link Layer Constants

// Start bytes
const (
	StartByte1 uint8 = 0x05 // First start byte
	StartByte2 uint8 = 0x64 // Second start byte
)

// Frame sizes
const (
	MinFrameSize    = 10  // Minimum frame size (header + CRCs)
	MaxFrameSize    = 292 // Maximum frame size
	HeaderSize      = 10  // Size of link header (including start bytes and header CRC)
	MaxDataSize     = 250 // Maximum user data in a frame
	BlockSize       = 16  // CRC block size
)

// Function codes
type FunctionCode uint8

const (
	// Primary to Secondary (Master to Outstation)
	FuncResetLink           FunctionCode = 0x00 // Reset link
	FuncResetUserProcess    FunctionCode = 0x01 // Reset user process
	FuncTestLinkStates      FunctionCode = 0x02 // Test link states
	FuncUserDataConfirmed   FunctionCode = 0x03 // User data with confirmation
	FuncUserDataUnconfirmed FunctionCode = 0x04 // User data without confirmation
	FuncRequestLinkStatus   FunctionCode = 0x09 // Request link status

	// Secondary to Primary (Outstation to Master)
	FuncAck                 FunctionCode = 0x00 // ACK
	FuncNack                FunctionCode = 0x01 // NACK
	FuncLinkStatusResponse  FunctionCode = 0x0B // Link status response
	FuncLinkNotFunctioning  FunctionCode = 0x0E // Link not functioning
	FuncLinkNotUsed         FunctionCode = 0x0F // Link not used/supported
)

// Control field bits
const (
	CtrlDIR      uint8 = 0x80 // Direction bit (1=master to outstation, 0=outstation to master)
	CtrlPRM      uint8 = 0x40 // Primary bit (1=from primary station)
	CtrlFCB      uint8 = 0x20 // Frame Count Bit (toggles with each new transmission)
	CtrlFCV      uint8 = 0x10 // Frame Count Valid (FCB is valid)
	CtrlDFC      uint8 = 0x10 // Data Flow Control (used in secondary frames)
	CtrlFuncMask uint8 = 0x0F // Function code mask (lower 4 bits)
)

// Predefined control bytes for common operations
const (
	// Master to Outstation (PRM=1, DIR=1)
	CtrlResetLink         uint8 = 0xC0 // RESET LINK (11000000)
	CtrlResetUserProcess  uint8 = 0xC1 // RESET USER PROCESS (11000001)
	CtrlTestLinkStates    uint8 = 0xC2 // TEST LINK STATES (11000010)
	CtrlUserDataConfFCB0  uint8 = 0x53 // USER DATA CONF, FCB=0 (01010011)
	CtrlUserDataConfFCB1  uint8 = 0x73 // USER DATA CONF, FCB=1 (01110011)
	CtrlUserDataUnconf    uint8 = 0x44 // USER DATA UNCONF (01000100)
	CtrlRequestLinkStatus uint8 = 0xC9 // REQUEST LINK STATUS (11001001)

	// Outstation to Master (PRM=0, DIR=0)
	CtrlAck               uint8 = 0x80 // ACK (10000000)
	CtrlNack              uint8 = 0x81 // NACK (10000001)
	CtrlUserDataUnconfOut uint8 = 0x84 // USER DATA UNCONF - Unsolicited (10000100)
	CtrlLinkStatus        uint8 = 0x8B // LINK STATUS (10001011)
	CtrlLinkNotFunc       uint8 = 0x8E // LINK NOT FUNCTIONING (10001110)
	CtrlLinkNotUsed       uint8 = 0x8F // LINK NOT USED (10001111)
)

// Link layer states
type LinkState int

const (
	LinkStateIdle         LinkState = iota // Idle, ready for operations
	LinkStateWaitACK                       // Waiting for ACK
	LinkStateResetPending                  // Reset in progress
	LinkStateTestPending                   // Test link in progress
	LinkStateProcessing                    // Processing received frame
	LinkStateError                         // Error state
)

// String returns string representation of LinkState
func (s LinkState) String() string {
	switch s {
	case LinkStateIdle:
		return "Idle"
	case LinkStateWaitACK:
		return "WaitACK"
	case LinkStateResetPending:
		return "ResetPending"
	case LinkStateTestPending:
		return "TestPending"
	case LinkStateProcessing:
		return "Processing"
	case LinkStateError:
		return "Error"
	default:
		return "Unknown"
	}
}

// Errors
var (
	ErrInvalidStartBytes  = errors.New("invalid start bytes")
	ErrInvalidLength      = errors.New("invalid frame length")
	ErrInvalidCRC         = errors.New("invalid CRC")
	ErrFrameTooShort      = errors.New("frame too short")
	ErrFrameTooLong       = errors.New("frame too long")
	ErrInvalidDirection   = errors.New("invalid direction bit")
	ErrInvalidAddress     = errors.New("invalid address")
	ErrTimeout            = errors.New("link layer timeout")
	ErrMaxRetriesExceeded = errors.New("max retries exceeded")
	ErrLinkNotReady       = errors.New("link layer not ready")
	ErrInvalidState       = errors.New("invalid link state")
	ErrFCBMismatch        = errors.New("FCB mismatch")
)

// Direction indicates frame direction
type Direction bool

const (
	DirectionMasterToOutstation Direction = true
	DirectionOutstationToMaster Direction = false
)

// String returns string representation of Direction
func (d Direction) String() string {
	if d {
		return "Master->Outstation"
	}
	return "Outstation->Master"
}

// IsPrimary indicates if frame is from primary station
type IsPrimary bool

const (
	PrimaryFrame   IsPrimary = true
	SecondaryFrame IsPrimary = false
)

// String returns string representation of IsPrimary
func (p IsPrimary) String() string {
	if p {
		return "Primary"
	}
	return "Secondary"
}
