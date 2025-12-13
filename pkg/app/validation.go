package app

import (
	"fmt"
)

// Validation errors
var (
	ErrInvalidQualifier    = fmt.Errorf("invalid qualifier code")
	ErrInvalidGroup        = fmt.Errorf("invalid object group")
	ErrInvalidVariation    = fmt.Errorf("invalid object variation")
	ErrInvalidStartStop    = fmt.Errorf("invalid start-stop range (start > stop)")
	ErrInvalidCount        = fmt.Errorf("invalid count value")
	ErrInvalidFunctionCode = fmt.Errorf("invalid function code")
	ErrInvalidSequence     = fmt.Errorf("invalid sequence number")
	ErrInvalidControlCode  = fmt.Errorf("invalid control code")
)

// ValidateObjectHeader validates an object header
func ValidateObjectHeader(header *ObjectHeader) error {
	// Validate group
	if !IsValidGroup(header.Group) {
		return fmt.Errorf("%w: %d", ErrInvalidGroup, header.Group)
	}

	// Validate variation for group
	if !IsValidVariation(header.Group, header.Variation) {
		return fmt.Errorf("%w: group=%d, variation=%d", ErrInvalidVariation, header.Group, header.Variation)
	}

	// Validate qualifier
	if !IsValidQualifier(header.Qualifier) {
		return fmt.Errorf("%w: 0x%02X", ErrInvalidQualifier, header.Qualifier)
	}

	// Validate range
	if err := ValidateRange(header.Range); err != nil {
		return err
	}

	return nil
}

// ValidateRange validates a range specification
func ValidateRange(r Range) error {
	switch v := r.(type) {
	case StartStopRange:
		if v.Stop < v.Start {
			return fmt.Errorf("%w: start=%d, stop=%d", ErrInvalidStartStop, v.Start, v.Stop)
		}
	case CountRange:
		if v.Count == 0 {
			return fmt.Errorf("%w: count=0", ErrInvalidCount)
		}
	case NoRange:
		// No validation needed
	default:
		return ErrInvalidRange
	}
	return nil
}

// ValidateFunctionCode validates a function code
func ValidateFunctionCode(fc FunctionCode) error {
	if !IsValidFunctionCode(fc) {
		return fmt.Errorf("%w: %d", ErrInvalidFunctionCode, fc)
	}
	return nil
}

// ValidateSequence validates a sequence number (0-15)
func ValidateSequence(seq uint8) error {
	if seq > 15 {
		return fmt.Errorf("%w: %d (must be 0-15)", ErrInvalidSequence, seq)
	}
	return nil
}

// ValidateControlCode validates a CROB control code
func ValidateControlCode(code uint8) error {
	if !IsValidControlCode(code) {
		return fmt.Errorf("%w: 0x%02X", ErrInvalidControlCode, code)
	}
	return nil
}

// ValidateAPDU performs comprehensive APDU validation
func ValidateAPDU(apdu *APDU) error {
	// Validate sequence
	if err := ValidateSequence(apdu.Sequence); err != nil {
		return err
	}

	// Validate function code
	if err := ValidateFunctionCode(apdu.FunctionCode); err != nil {
		return err
	}

	// Validate control bits consistency
	if apdu.UNS && apdu.FunctionCode != FuncUnsolicitedResponse {
		return fmt.Errorf("UNS bit set but function is not unsolicited response")
	}

	if apdu.FunctionCode == FuncUnsolicitedResponse && !apdu.UNS {
		return fmt.Errorf("unsolicited response without UNS bit")
	}

	// Unsolicited responses should have CON set
	if apdu.UNS && !apdu.CON {
		return fmt.Errorf("unsolicited response should have CON bit set")
	}

	return nil
}

// Validation check functions

// IsValidGroup checks if a group number is valid
func IsValidGroup(group uint8) bool {
	// Common valid groups
	validGroups := map[uint8]bool{
		1: true, 2: true, 3: true, 4: true,
		10: true, 11: true, 12: true,
		20: true, 21: true, 22: true, 23: true,
		30: true, 31: true, 32: true, 33: true,
		40: true, 41: true, 42: true, 43: true,
		50: true, 51: true, 52: true,
		60: true, 61: true, 62: true, 63: true,
		70: true, // File transfer
		80: true, // Internal indications
		110: true, 111: true, 112: true, 113: true, // Octet strings
		120: true, // Authentication
	}
	return validGroups[group]
}

// IsValidVariation checks if a variation is valid for a group
func IsValidVariation(group, variation uint8) bool {
	// Variation 0 (any) is valid for all groups
	if variation == 0 {
		return true
	}

	// Define valid variations per group
	switch group {
	case GroupBinaryInput: // Group 1
		return variation <= 2
	case GroupBinaryInputEvent: // Group 2
		return variation >= 1 && variation <= 3
	case GroupDoubleBitBinaryInput: // Group 3
		return variation <= 2
	case GroupBinaryOutput: // Group 10
		return variation <= 2
	case GroupBinaryOutputCommand: // Group 12
		return variation == 1
	case GroupCounter: // Group 20
		return (variation >= 1 && variation <= 2) || (variation >= 5 && variation <= 8)
	case GroupCounterEvent: // Group 22
		return (variation >= 1 && variation <= 2) || (variation >= 5 && variation <= 8)
	case GroupAnalogInput: // Group 30
		return variation >= 1 && variation <= 6
	case GroupAnalogInputEvent: // Group 32
		return variation >= 1 && variation <= 8
	case GroupAnalogOutputStatus: // Group 40
		return variation >= 1 && variation <= 4
	case GroupAnalogOutputCommand: // Group 41
		return variation >= 1 && variation <= 4
	case GroupTimeDate: // Group 50
		return variation >= 1 && variation <= 3
	case GroupClass0Data: // Group 60
		return variation >= 1 && variation <= 4
	default:
		return true // Unknown groups, allow any variation
	}
}

// IsValidQualifier checks if a qualifier code is valid
func IsValidQualifier(q QualifierCode) bool {
	validQualifiers := map[QualifierCode]bool{
		Qualifier8BitStartStop:        true,
		Qualifier16BitStartStop:       true,
		Qualifier32BitStartStop:       true,
		Qualifier8BitAbsoluteAddress:  true,
		Qualifier16BitAbsoluteAddress: true,
		Qualifier32BitAbsoluteAddress: true,
		QualifierNoRange:              true,
		Qualifier8BitCount:            true,
		Qualifier16BitCount:           true,
		Qualifier32BitCount:           true,
		QualifierFreeFormat:           true,
	}
	return validQualifiers[q]
}

// IsValidFunctionCode checks if a function code is valid
func IsValidFunctionCode(fc FunctionCode) bool {
	// Check common function codes
	return fc <= 0x1E || fc == FuncResponse || fc == FuncUnsolicitedResponse || fc == FuncAuthResponse
}

// IsValidControlCode checks if a CROB control code is valid
func IsValidControlCode(code uint8) bool {
	validCodes := map[uint8]bool{
		ControlCodeNUL:      true,
		ControlCodePulseOn:  true,
		ControlCodePulseOff: true,
		ControlCodeLatchOn:  true,
		ControlCodeLatchOff: true,
		ControlCodeCloseOn:  true,
		ControlCodeTripOff:  true,
	}
	return validCodes[code]
}

// Data size validation helpers

// GetObjectSize returns the size in bytes for a single object of given group/variation
// Returns 0 if size is variable or unknown
func GetObjectSize(group, variation uint8) int {
	switch group {
	case GroupBinaryInput: // Group 1
		switch variation {
		case 2: // With flags
			return 1
		}

	case GroupBinaryInputEvent: // Group 2
		switch variation {
		case 1: // Without time
			return 1
		case 2: // With absolute time
			return 7
		case 3: // With relative time
			return 3
		}

	case GroupBinaryOutputCommand: // Group 12
		switch variation {
		case 1: // CROB
			return 11
		}

	case GroupCounter: // Group 20
		switch variation {
		case 1, 5: // 32-bit
			return 5
		case 2, 6: // 16-bit
			return 3
		}

	case GroupAnalogInput: // Group 30
		switch variation {
		case 1: // 32-bit with flag
			return 5
		case 2: // 16-bit with flag
			return 3
		case 5: // Float with flag
			return 5
		case 6: // Double with flag
			return 9
		}

	case GroupAnalogInputEvent: // Group 32
		switch variation {
		case 1: // 32-bit no time
			return 5
		case 2: // 16-bit no time
			return 3
		case 3: // 32-bit with time
			return 11
		case 4: // 16-bit with time
			return 9
		}

	case GroupAnalogOutputStatus: // Group 40
		switch variation {
		case 1: // 32-bit with flag
			return 5
		case 2: // 16-bit with flag
			return 3
		case 3: // Float with flag
			return 5
		case 4: // Double with flag
			return 9
		}

	case GroupAnalogOutputCommand: // Group 41
		switch variation {
		case 1: // 32-bit
			return 5
		case 2: // 16-bit
			return 3
		case 3: // Float
			return 5
		case 4: // Double
			return 9
		}

	case GroupTimeDate: // Group 50
		switch variation {
		case 1: // Absolute time
			return 6
		}
	}

	return 0 // Variable or unknown size
}

// ValidateObjectData validates that object data has correct size
func ValidateObjectData(header *ObjectHeader, data []byte) error {
	expectedSize := GetObjectSize(header.Group, header.Variation)
	if expectedSize == 0 {
		// Variable size, can't validate
		return nil
	}

	count := GetCount(header.Range)
	if count == 0 {
		// No range specified, can't validate count
		return nil
	}

	expectedTotal := int(count) * expectedSize
	if len(data) < expectedTotal {
		return fmt.Errorf("insufficient object data: have %d bytes, need %d bytes for %d objects of size %d",
			len(data), expectedTotal, count, expectedSize)
	}

	return nil
}
