package app

// DNP3 Object Groups and Variations

// Object Group numbers
const (
	GroupBinaryInput           uint8 = 1
	GroupBinaryInputEvent      uint8 = 2
	GroupDoubleBitBinaryInput  uint8 = 3
	GroupDoubleBitBinaryEvent  uint8 = 4
	GroupBinaryOutput          uint8 = 10
	GroupBinaryOutputEvent     uint8 = 11
	GroupBinaryOutputCommand   uint8 = 12
	GroupCounter               uint8 = 20
	GroupFrozenCounter         uint8 = 21
	GroupCounterEvent          uint8 = 22
	GroupFrozenCounterEvent    uint8 = 23
	GroupAnalogInput           uint8 = 30
	GroupFrozenAnalogInput     uint8 = 31
	GroupAnalogInputEvent      uint8 = 32
	GroupFrozenAnalogEvent     uint8 = 33
	GroupAnalogOutputStatus    uint8 = 40
	GroupAnalogOutputEvent     uint8 = 42
	GroupAnalogOutputCommand   uint8 = 41
	GroupTimeDate              uint8 = 50
	GroupClass0Data            uint8 = 60
	GroupClass1Data            uint8 = 61
	GroupClass2Data            uint8 = 62
	GroupClass3Data            uint8 = 63
	GroupInternalIndications   uint8 = 80
)

// Common variations
const (
	VariationAny uint8 = 0 // Request any variation
)

// Binary Input variations (Group 1)
const (
	BinaryInputAny                  uint8 = 0
	BinaryInputPacked               uint8 = 1 // Packed format
	BinaryInputWithFlags            uint8 = 2 // With flags
)

// Binary Input Event variations (Group 2)
const (
	BinaryInputEventAny             uint8 = 0
	BinaryInputEventWithoutTime     uint8 = 1
	BinaryInputEventWithTime        uint8 = 2
	BinaryInputEventWithRelativeTime uint8 = 3
)

// Counter variations (Group 20)
const (
	CounterAny                      uint8 = 0
	Counter32Bit                    uint8 = 1
	Counter16Bit                    uint8 = 2
	Counter32BitWithFlag            uint8 = 5
	Counter16BitWithFlag            uint8 = 6
)

// Analog Input variations (Group 30)
const (
	AnalogInputAny                  uint8 = 0
	AnalogInput32Bit                uint8 = 1 // 32-bit integer
	AnalogInput16Bit                uint8 = 2 // 16-bit integer
	AnalogInput32BitNoFlag          uint8 = 3 // 32-bit without flag
	AnalogInput16BitNoFlag          uint8 = 4 // 16-bit without flag
	AnalogInputFloat                uint8 = 5 // Single-precision float
	AnalogInputDouble               uint8 = 6 // Double-precision float
)

// Analog Input Event variations (Group 32)
const (
	AnalogInputEventAny             uint8 = 0
	AnalogInputEvent32BitNoTime     uint8 = 1
	AnalogInputEvent16BitNoTime     uint8 = 2
	AnalogInputEvent32BitWithTime   uint8 = 3
	AnalogInputEvent16BitWithTime   uint8 = 4
	AnalogInputEventFloatNoTime     uint8 = 5
	AnalogInputEventDoubleNoTime    uint8 = 6
	AnalogInputEventFloatWithTime   uint8 = 7
	AnalogInputEventDoubleWithTime  uint8 = 8
)

// Qualifier codes
type QualifierCode uint8

const (
	Qualifier8BitStartStop        QualifierCode = 0x00 // 8-bit start-stop indices
	Qualifier16BitStartStop       QualifierCode = 0x01 // 16-bit start-stop indices
	Qualifier32BitStartStop       QualifierCode = 0x02 // 32-bit start-stop indices
	Qualifier8BitAbsoluteAddress  QualifierCode = 0x03 // 8-bit absolute addressing
	Qualifier16BitAbsoluteAddress QualifierCode = 0x04 // 16-bit absolute addressing
	Qualifier32BitAbsoluteAddress QualifierCode = 0x05 // 32-bit absolute addressing
	QualifierNoRange              QualifierCode = 0x06 // No range field
	Qualifier8BitCount            QualifierCode = 0x07 // 8-bit quantity
	Qualifier16BitCount           QualifierCode = 0x08 // 16-bit quantity
	Qualifier32BitCount           QualifierCode = 0x09 // 32-bit quantity
	QualifierFreeFormat           QualifierCode = 0x5B // Free format
)

// ObjectHeader represents a DNP3 object header
type ObjectHeader struct {
	Group      uint8         // Object group
	Variation  uint8         // Object variation
	Qualifier  QualifierCode // Qualifier code
	Range      Range         // Range specification
}

// Range represents the range/addressing in an object header
type Range interface {
	isRange()
}

// StartStopRange represents start-stop index range
type StartStopRange struct {
	Start uint32
	Stop  uint32
}

func (StartStopRange) isRange() {}

// CountRange represents count-based range
type CountRange struct {
	Count uint32
}

func (CountRange) isRange() {}

// NoRange represents headers with no range
type NoRange struct{}

func (NoRange) isRange() {}

// ClassField represents DNP3 class assignments
type ClassField uint8

const (
	ClassNone ClassField = 0
	Class0    ClassField = 1 << 0 // Static data
	Class1    ClassField = 1 << 1 // High priority events
	Class2    ClassField = 1 << 2 // Medium priority events
	Class3    ClassField = 1 << 3 // Low priority events
	ClassAll  ClassField = Class1 | Class2 | Class3
)

// HasClass checks if a specific class is set
func (c ClassField) HasClass(class ClassField) bool {
	return c&class != 0
}

// String returns string representation of ClassField
func (c ClassField) String() string {
	if c == ClassNone {
		return "None"
	}
	if c == Class0 {
		return "Class0"
	}
	if c == ClassAll {
		return "Class1,2,3"
	}

	result := ""
	if c&Class1 != 0 {
		result += "Class1"
	}
	if c&Class2 != 0 {
		if result != "" {
			result += ","
		}
		result += "Class2"
	}
	if c&Class3 != 0 {
		if result != "" {
			result += ","
		}
		result += "Class3"
	}
	return result
}
