package app

import (
	"bytes"
	"fmt"
)

// Application Control Field bits
const (
	AppCtrlFIR uint8 = 0x80 // First fragment
	AppCtrlFIN uint8 = 0x40 // Final fragment
	AppCtrlCON uint8 = 0x20 // Confirm required
	AppCtrlUNS uint8 = 0x10 // Unsolicited response
	AppCtrlSeqMask uint8 = 0x0F // Sequence number mask (4 bits)
)

// APDU represents an Application Protocol Data Unit
type APDU struct {
	// Application Control field
	Control     uint8        // Control byte
	FIR         bool         // First fragment
	FIN         bool         // Final fragment
	CON         bool         // Confirm required
	UNS         bool         // Unsolicited
	Sequence    uint8        // Sequence number (0-15)

	// Function and IIN
	FunctionCode FunctionCode // Function code
	IIN          IIN          // Internal Indications (only in responses)

	// Object headers and data
	Objects     []byte       // Raw object data
}

// NewRequestAPDU creates a new request APDU
func NewRequestAPDU(fc FunctionCode, seq uint8, objects []byte) *APDU {
	return &APDU{
		FunctionCode: fc,
		Sequence:     seq & AppCtrlSeqMask,
		FIR:          true,
		FIN:          true,
		CON:          false,
		UNS:          false,
		Objects:      objects,
	}
}

// NewResponseAPDU creates a new response APDU
func NewResponseAPDU(seq uint8, iin IIN, objects []byte) *APDU {
	return &APDU{
		FunctionCode: FuncResponse,
		Sequence:     seq & AppCtrlSeqMask,
		FIR:          true,
		FIN:          true,
		CON:          false,
		UNS:          false,
		IIN:          iin,
		Objects:      objects,
	}
}

// NewUnsolicitedResponseAPDU creates a new unsolicited response APDU
func NewUnsolicitedResponseAPDU(seq uint8, iin IIN, objects []byte) *APDU {
	apdu := NewResponseAPDU(seq, iin, objects)
	apdu.FunctionCode = FuncUnsolicitedResponse
	apdu.UNS = true
	apdu.CON = true // Unsolicited typically requires confirm
	return apdu
}

// buildControl builds the control byte from APDU fields
func (a *APDU) buildControl() {
	a.Control = a.Sequence & AppCtrlSeqMask

	if a.FIR {
		a.Control |= AppCtrlFIR
	}
	if a.FIN {
		a.Control |= AppCtrlFIN
	}
	if a.CON {
		a.Control |= AppCtrlCON
	}
	if a.UNS {
		a.Control |= AppCtrlUNS
	}
}

// parseControl parses the control byte into APDU fields
func (a *APDU) parseControl() {
	a.FIR = (a.Control & AppCtrlFIR) != 0
	a.FIN = (a.Control & AppCtrlFIN) != 0
	a.CON = (a.Control & AppCtrlCON) != 0
	a.UNS = (a.Control & AppCtrlUNS) != 0
	a.Sequence = a.Control & AppCtrlSeqMask
}

// Serialize converts APDU to wire format
func (a *APDU) Serialize() []byte {
	a.buildControl()

	var buf bytes.Buffer

	// Write control byte
	buf.WriteByte(a.Control)

	// Write function code
	buf.WriteByte(uint8(a.FunctionCode))

	// Write IIN if this is a response
	if a.FunctionCode.IsResponse() {
		buf.WriteByte(a.IIN.IIN1)
		buf.WriteByte(a.IIN.IIN2)
	}

	// Write objects
	buf.Write(a.Objects)

	return buf.Bytes()
}

// Parse parses wire format data into APDU
func Parse(data []byte) (*APDU, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("APDU too short: %d bytes", len(data))
	}

	apdu := &APDU{
		Control:      data[0],
		FunctionCode: FunctionCode(data[1]),
	}

	apdu.parseControl()

	offset := 2

	// Parse IIN if this is a response
	if apdu.FunctionCode.IsResponse() {
		if len(data) < 4 {
			return nil, fmt.Errorf("response APDU too short for IIN")
		}
		apdu.IIN.IIN1 = data[2]
		apdu.IIN.IIN2 = data[3]
		offset = 4
	}

	// Remaining data is objects
	if offset < len(data) {
		apdu.Objects = data[offset:]
	}

	return apdu, nil
}

// SetConfirm sets the confirm required flag
func (a *APDU) SetConfirm(confirm bool) {
	a.CON = confirm
}

// SetSequence sets the sequence number
func (a *APDU) SetSequence(seq uint8) {
	a.Sequence = seq & AppCtrlSeqMask
}

// IsResponse returns true if this is a response APDU
func (a *APDU) IsResponse() bool {
	return a.FunctionCode.IsResponse()
}

// IsRequest returns true if this is a request APDU
func (a *APDU) IsRequest() bool {
	return a.FunctionCode.IsRequest()
}

// String returns string representation of APDU
func (a *APDU) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("APDU{Func=%s, Seq=%d", a.FunctionCode, a.Sequence))

	if a.FIR {
		buf.WriteString(", FIR")
	}
	if a.FIN {
		buf.WriteString(", FIN")
	}
	if a.CON {
		buf.WriteString(", CON")
	}
	if a.UNS {
		buf.WriteString(", UNS")
	}

	if a.IsResponse() {
		buf.WriteString(fmt.Sprintf(", IIN=[%02X,%02X]", a.IIN.IIN1, a.IIN.IIN2))
	}

	buf.WriteString(fmt.Sprintf(", ObjectsLen=%d}", len(a.Objects)))
	return buf.String()
}
