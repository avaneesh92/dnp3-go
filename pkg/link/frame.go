package link

import (
	"bytes"
	"fmt"
)

// Frame represents a DNP3 link layer frame
type Frame struct {
	// Header fields
	Control      uint8        // Control byte
	Destination  uint16       // Destination address
	Source       uint16       // Source address

	// Derived fields from control byte
	Dir          Direction    // Direction (master->outstation or outstation->master)
	IsPrimary    IsPrimary    // Primary or secondary frame
	FCB          bool         // Frame Count Bit
	FCV          bool         // Frame Count Valid
	FunctionCode FunctionCode // Function code

	// User data
	UserData     []byte       // User data (without CRCs)
}

// NewFrame creates a new link frame
func NewFrame(dir Direction, isPrimary IsPrimary, fc FunctionCode, dst, src uint16, data []byte) *Frame {
	frame := &Frame{
		Dir:          dir,
		IsPrimary:    isPrimary,
		FunctionCode: fc,
		Destination:  dst,
		Source:       src,
		UserData:     data,
		FCB:          false,
		FCV:          false,
	}
	frame.buildControl()
	return frame
}

// buildControl builds the control byte from frame fields
func (f *Frame) buildControl() {
	f.Control = uint8(f.FunctionCode) & CtrlFuncMask

	if f.Dir == DirectionMasterToOutstation {
		f.Control |= CtrlDIR
	}

	if f.IsPrimary == PrimaryFrame {
		f.Control |= CtrlPRM
		if f.FCV {
			f.Control |= CtrlFCV
			if f.FCB {
				f.Control |= CtrlFCB
			}
		}
	} else {
		// Secondary frame - DFC bit uses same position as FCV
		// Not setting it here as it's typically set by layer logic
	}
}

// parseControl parses the control byte into frame fields
func (f *Frame) parseControl() {
	f.FunctionCode = FunctionCode(f.Control & CtrlFuncMask)
	f.Dir = Direction((f.Control & CtrlDIR) != 0)
	f.IsPrimary = IsPrimary((f.Control & CtrlPRM) != 0)

	if f.IsPrimary == PrimaryFrame {
		f.FCV = (f.Control & CtrlFCV) != 0
		f.FCB = (f.Control & CtrlFCB) != 0
	}
}

// SetFCB sets the Frame Count Bit and Frame Count Valid
func (f *Frame) SetFCB(fcb bool) {
	f.FCB = fcb
	f.FCV = true
	f.buildControl()
}

// Serialize converts frame to wire format with CRCs
func (f *Frame) Serialize() ([]byte, error) {
	dataLen := len(f.UserData)
	if dataLen > MaxDataSize {
		return nil, ErrFrameTooLong
	}

	// Build header (without CRCs yet)
	header := make([]byte, 8)
	header[0] = StartByte1
	header[1] = StartByte2
	header[2] = byte(dataLen + 5) // Length includes control + addresses
	header[3] = f.Control
	header[4] = byte(f.Destination)
	header[5] = byte(f.Destination >> 8)
	header[6] = byte(f.Source)
	header[7] = byte(f.Source >> 8)

	// Add header CRC
	headerCRC := CalculateCRC(header)
	header = append(header, byte(headerCRC), byte(headerCRC>>8))

	// If no user data, we're done
	if dataLen == 0 {
		return header, nil
	}

	// Add user data with CRCs every 16 bytes
	dataWithCRCs := AddCRCs(f.UserData)

	result := make([]byte, len(header)+len(dataWithCRCs))
	copy(result, header)
	copy(result[len(header):], dataWithCRCs)

	return result, nil
}

// Parse parses wire format data into a Frame
func Parse(data []byte) (*Frame, int, error) {
	if len(data) < MinFrameSize {
		// Debug: log the actual length received
		return nil, len(data), ErrFrameTooShort
	}

	// Check start bytes
	if data[0] != StartByte1 || data[1] != StartByte2 {
		return nil, 0, ErrInvalidStartBytes
	}

	// Get length field (includes control + addresses = 5 bytes)
	length := int(data[2])
	if length < 5 {
		return nil, 0, ErrInvalidLength
	}

	dataLen := length - 5 // User data length

	// Calculate total frame size
	// Header: 10 bytes (2 start + 1 len + 5 fields + 2 CRC)
	// Data: blocks of 16 bytes + 2-byte CRC each
	numBlocks := (dataLen + 15) / 16
	expectedSize := 10 + dataLen + (numBlocks * 2)

	if len(data) < expectedSize {
		return nil, 0, ErrFrameTooShort
	}

	// Verify header CRC
	headerWithoutCRC := data[0:8]
	if !VerifyCRC(data[0:10]) {
		return nil, 0, ErrInvalidCRC
	}

	// Parse header fields
	frame := &Frame{
		Control:     headerWithoutCRC[3],
		Destination: uint16(headerWithoutCRC[4]) | (uint16(headerWithoutCRC[5]) << 8),
		Source:      uint16(headerWithoutCRC[6]) | (uint16(headerWithoutCRC[7]) << 8),
	}

	frame.parseControl()

	// Extract and verify user data if present
	if dataLen > 0 {
		dataWithCRCs := data[10:expectedSize]
		userData, err := RemoveCRCs(dataWithCRCs)
		if err != nil {
			return nil, 0, err
		}
		frame.UserData = userData
	}

	return frame, expectedSize, nil
}

// String returns a string representation of the frame
func (f *Frame) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("Frame{Dir=%s, ", f.Dir))
	buf.WriteString(fmt.Sprintf("Func=%d, ", f.FunctionCode))
	buf.WriteString(fmt.Sprintf("Dst=%d, Src=%d, ", f.Destination, f.Source))
	if f.IsPrimary == PrimaryFrame && f.FCV {
		buf.WriteString(fmt.Sprintf("FCB=%t, ", f.FCB))
	}
	buf.WriteString(fmt.Sprintf("DataLen=%d}", len(f.UserData)))
	return buf.String()
}

// Clone creates a deep copy of the frame
func (f *Frame) Clone() *Frame {
	userData := make([]byte, len(f.UserData))
	copy(userData, f.UserData)

	return &Frame{
		Control:      f.Control,
		Destination:  f.Destination,
		Source:       f.Source,
		Dir:          f.Dir,
		IsPrimary:    f.IsPrimary,
		FCB:          f.FCB,
		FCV:          f.FCV,
		FunctionCode: f.FunctionCode,
		UserData:     userData,
	}
}
