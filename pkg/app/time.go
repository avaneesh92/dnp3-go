package app

import (
	"encoding/binary"
	"time"
)

// DNP3Time represents DNP3 timestamp format
// DNP3 uses 48-bit timestamp: milliseconds since midnight January 1, 1970 UTC
type DNP3Time uint64

// DNP3 epoch is the same as Unix epoch
const dnp3EpochOffset = 0

// Now returns current time as DNP3Time
func Now() DNP3Time {
	return DNP3Time(time.Now().UnixMilli())
}

// FromTime converts Go time.Time to DNP3Time
func FromTime(t time.Time) DNP3Time {
	return DNP3Time(t.UnixMilli())
}

// ToTime converts DNP3Time to Go time.Time
func (t DNP3Time) ToTime() time.Time {
	return time.UnixMilli(int64(t))
}

// SerializeTime48 serializes DNP3Time as 48-bit value (6 bytes)
func (t DNP3Time) SerializeTime48() []byte {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, uint64(t))
	return buf[:6] // DNP3 time is 6 bytes (48 bits)
}

// ParseTime48 parses 48-bit DNP3 time
func ParseTime48(data []byte) DNP3Time {
	if len(data) < 6 {
		return 0
	}

	// Extend to 8 bytes for uint64 parsing
	buf := make([]byte, 8)
	copy(buf, data[:6])

	return DNP3Time(binary.LittleEndian.Uint64(buf))
}

// TimeAndDate represents Group 50 time objects
type TimeAndDate struct {
	Time DNP3Time
}

// NewTimeAndDate creates a new time and date object with current time
func NewTimeAndDate() TimeAndDate {
	return TimeAndDate{Time: Now()}
}

// NewTimeAndDateFrom creates a new time and date object from Go time
func NewTimeAndDateFrom(t time.Time) TimeAndDate {
	return TimeAndDate{Time: FromTime(t)}
}

// Serialize serializes time and date (Group 50, Var 1)
func (t TimeAndDate) Serialize() []byte {
	return t.Time.SerializeTime48()
}

// ParseTimeAndDate parses time and date from wire format
func ParseTimeAndDate(data []byte) TimeAndDate {
	return TimeAndDate{Time: ParseTime48(data)}
}

// BuildTimeSync builds a time synchronization write request (Group 50, Var 1)
func BuildTimeSync(t time.Time) []byte {
	builder := NewObjectBuilder()

	// Add header for time object
	builder.AddHeader(GroupTimeDate, 1, Qualifier8BitCount, CountRange{Count: 1})

	// Add time data (6 bytes)
	dnpTime := FromTime(t)
	builder.AddRawData(dnpTime.SerializeTime48())

	return builder.Build()
}

// BuildTimeSyncNow builds a time synchronization request with current time
func BuildTimeSyncNow() []byte {
	return BuildTimeSync(time.Now())
}

// TimeDelay represents delay measurement (Group 52)
type TimeDelay struct {
	Delay uint16 // Delay in milliseconds or microseconds
}

// NewTimeDelay creates a new time delay object
func NewTimeDelay(delayMs uint16) TimeDelay {
	return TimeDelay{Delay: delayMs}
}

// SerializeCoarse serializes coarse time delay in ms (Group 52, Var 1)
func (d TimeDelay) SerializeCoarse() []byte {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, d.Delay)
	return buf
}

// ParseTimeDelayCoarse parses coarse time delay
func ParseTimeDelayCoarse(data []byte) TimeDelay {
	if len(data) < 2 {
		return TimeDelay{}
	}
	return TimeDelay{
		Delay: binary.LittleEndian.Uint16(data),
	}
}

// RelativeTime represents relative time offset for events
type RelativeTime uint16

// NewRelativeTime creates relative time from milliseconds
func NewRelativeTime(ms uint16) RelativeTime {
	return RelativeTime(ms)
}

// Serialize serializes relative time (2 bytes)
func (r RelativeTime) Serialize() []byte {
	buf := make([]byte, 2)
	binary.LittleEndian.PutUint16(buf, uint16(r))
	return buf
}

// ParseRelativeTime parses relative time
func ParseRelativeTime(data []byte) RelativeTime {
	if len(data) < 2 {
		return 0
	}
	return RelativeTime(binary.LittleEndian.Uint16(data))
}

// CommonTimeOfOccurrence represents CTO for indexed time (Group 51)
type CommonTimeOfOccurrence struct {
	Time         DNP3Time
	Synchronized bool
}

// NewCTO creates a new CTO with current time
func NewCTO(synchronized bool) CommonTimeOfOccurrence {
	return CommonTimeOfOccurrence{
		Time:         Now(),
		Synchronized: synchronized,
	}
}

// Serialize serializes CTO (Group 51, Var 1 or 2)
func (c CommonTimeOfOccurrence) Serialize() []byte {
	return c.Time.SerializeTime48()
}

// ParseCTO parses CTO from wire format
func ParseCTO(data []byte) CommonTimeOfOccurrence {
	return CommonTimeOfOccurrence{
		Time:         ParseTime48(data),
		Synchronized: true, // Variation determines this
	}
}
