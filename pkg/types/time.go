package types

import "time"

// DNP3Time represents milliseconds since Unix epoch (Jan 1, 1970 00:00:00 UTC)
// This is the standard DNP3 timestamp format
type DNP3Time uint64

// Now returns the current time as DNP3Time
func Now() DNP3Time {
	return DNP3Time(time.Now().UnixMilli())
}

// FromTime converts a Go time.Time to DNP3Time
func FromTime(t time.Time) DNP3Time {
	return DNP3Time(t.UnixMilli())
}

// ToTime converts DNP3Time to Go time.Time
func (t DNP3Time) ToTime() time.Time {
	return time.UnixMilli(int64(t))
}

// IsValid checks if the timestamp is valid (non-zero)
func (t DNP3Time) IsValid() bool {
	return t != 0
}

// Zero returns a zero DNP3Time value
func ZeroTime() DNP3Time {
	return DNP3Time(0)
}
