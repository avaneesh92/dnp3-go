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
	// Primary to Secondary
	FuncResetLinkStates     FunctionCode = 0x00 // Reset link states
	FuncUserDataConfirmed   FunctionCode = 0x04 // User data with confirmation
	FuncUserDataUnconfirmed FunctionCode = 0x05 // User data without confirmation
	FuncRequestLinkStatus   FunctionCode = 0x09 // Request link status

	// Secondary to Primary
	FuncAck                 FunctionCode = 0x00 // ACK
	FuncNack                FunctionCode = 0x01 // NACK
	FuncLinkStatusResponse  FunctionCode = 0x0B // Link status response
	FuncNotSupported        FunctionCode = 0x0F // Function not supported
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

// Errors
var (
	ErrInvalidStartBytes  = errors.New("invalid start bytes")
	ErrInvalidLength      = errors.New("invalid frame length")
	ErrInvalidCRC         = errors.New("invalid CRC")
	ErrFrameTooShort      = errors.New("frame too short")
	ErrFrameTooLong       = errors.New("frame too long")
	ErrInvalidDirection   = errors.New("invalid direction bit")
	ErrInvalidAddress     = errors.New("invalid address")
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
