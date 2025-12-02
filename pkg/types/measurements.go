package types

// Binary represents a binary input (on/off) measurement
type Binary struct {
	Value bool
	Flags Flags
	Time  DNP3Time
}

// GetFlags returns the quality flags for this measurement
func (b Binary) GetFlags() Flags {
	return b.Flags
}

// GetTime returns the timestamp for this measurement
func (b Binary) GetTime() DNP3Time {
	return b.Time
}

// DoubleBitValue represents the state of a double-bit binary input
type DoubleBitValue uint8

const (
	DoubleBitIntermediate DoubleBitValue = 0
	DoubleBitOff          DoubleBitValue = 1
	DoubleBitOn           DoubleBitValue = 2
	DoubleBitIndeterminate DoubleBitValue = 3
)

// DoubleBitBinary represents a double-bit binary input measurement
type DoubleBitBinary struct {
	Value DoubleBitValue
	Flags Flags
	Time  DNP3Time
}

// GetFlags returns the quality flags for this measurement
func (d DoubleBitBinary) GetFlags() Flags {
	return d.Flags
}

// GetTime returns the timestamp for this measurement
func (d DoubleBitBinary) GetTime() DNP3Time {
	return d.Time
}

// Analog represents an analog input measurement
type Analog struct {
	Value float64 // Always stored as float64 internally
	Flags Flags
	Time  DNP3Time
}

// GetFlags returns the quality flags for this measurement
func (a Analog) GetFlags() Flags {
	return a.Flags
}

// GetTime returns the timestamp for this measurement
func (a Analog) GetTime() DNP3Time {
	return a.Time
}

// Counter represents a counter value
type Counter struct {
	Value uint32
	Flags Flags
	Time  DNP3Time
}

// GetFlags returns the quality flags for this measurement
func (c Counter) GetFlags() Flags {
	return c.Flags
}

// GetTime returns the timestamp for this measurement
func (c Counter) GetTime() DNP3Time {
	return c.Time
}

// FrozenCounter represents a frozen counter value
type FrozenCounter struct {
	Value uint32
	Flags Flags
	Time  DNP3Time
}

// GetFlags returns the quality flags for this measurement
func (f FrozenCounter) GetFlags() Flags {
	return f.Flags
}

// GetTime returns the timestamp for this measurement
func (f FrozenCounter) GetTime() DNP3Time {
	return f.Time
}

// BinaryOutputStatus represents the status of a binary output
type BinaryOutputStatus struct {
	Value bool
	Flags Flags
	Time  DNP3Time
}

// GetFlags returns the quality flags for this measurement
func (b BinaryOutputStatus) GetFlags() Flags {
	return b.Flags
}

// GetTime returns the timestamp for this measurement
func (b BinaryOutputStatus) GetTime() DNP3Time {
	return b.Time
}

// AnalogOutputStatus represents the status of an analog output
type AnalogOutputStatus struct {
	Value float64
	Flags Flags
	Time  DNP3Time
}

// GetFlags returns the quality flags for this measurement
func (a AnalogOutputStatus) GetFlags() Flags {
	return a.Flags
}

// GetTime returns the timestamp for this measurement
func (a AnalogOutputStatus) GetTime() DNP3Time {
	return a.Time
}

// OctetString represents an arbitrary byte string
type OctetString struct {
	Value []byte
}

// IntervalUnits represents the units for TimeAndInterval
type IntervalUnits uint8

const (
	IntervalUnitsNoRepeat    IntervalUnits = 0
	IntervalUnitsMilliseconds IntervalUnits = 1
	IntervalUnitsSeconds     IntervalUnits = 2
	IntervalUnitsMinutes     IntervalUnits = 3
	IntervalUnitsHours       IntervalUnits = 4
	IntervalUnitsDays        IntervalUnits = 5
	IntervalUnitsWeeks       IntervalUnits = 6
	IntervalUnitsMonths      IntervalUnits = 7
	IntervalUnitsSeasons     IntervalUnits = 8
)

// TimeAndInterval represents time and interval
type TimeAndInterval struct {
	Time     DNP3Time
	Interval uint32
	Units    IntervalUnits
}

// Measurement is a generic interface for all measurement types
type Measurement interface {
	GetFlags() Flags
	GetTime() DNP3Time
}

// Indexed measurement types for collections

// IndexedBinary is a binary measurement with its index
type IndexedBinary struct {
	Index uint16
	Value Binary
}

// IndexedDoubleBitBinary is a double-bit binary measurement with its index
type IndexedDoubleBitBinary struct {
	Index uint16
	Value DoubleBitBinary
}

// IndexedAnalog is an analog measurement with its index
type IndexedAnalog struct {
	Index uint16
	Value Analog
}

// IndexedCounter is a counter measurement with its index
type IndexedCounter struct {
	Index uint16
	Value Counter
}

// IndexedFrozenCounter is a frozen counter measurement with its index
type IndexedFrozenCounter struct {
	Index uint16
	Value FrozenCounter
}

// IndexedBinaryOutputStatus is a binary output status with its index
type IndexedBinaryOutputStatus struct {
	Index uint16
	Value BinaryOutputStatus
}

// IndexedAnalogOutputStatus is an analog output status with its index
type IndexedAnalogOutputStatus struct {
	Index uint16
	Value AnalogOutputStatus
}
