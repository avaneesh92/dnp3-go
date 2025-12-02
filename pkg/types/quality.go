package types

// Flags represents DNP3 quality flags
// These flags indicate the quality and status of measurement data
type Flags uint8

// DNP3 quality flag bits
const (
	FlagOnline       Flags = 0x01 // Point is online
	FlagRestart      Flags = 0x02 // Device restart detected
	FlagCommLost     Flags = 0x04 // Communication lost
	FlagRemoteForced Flags = 0x08 // Value forced by remote entity
	FlagLocalForced  Flags = 0x10 // Value forced locally
	FlagOverRange    Flags = 0x20 // Value exceeds measurement range
	FlagReferenceErr Flags = 0x40 // Reference error (e.g., ADC error)
	FlagReserved     Flags = 0x80 // Reserved bit
)

// Quality flag helper methods

// IsOnline returns true if the point is marked as online
func (f Flags) IsOnline() bool {
	return f&FlagOnline != 0
}

// HasRestart returns true if a device restart was detected
func (f Flags) HasRestart() bool {
	return f&FlagRestart != 0
}

// HasCommLost returns true if communication was lost
func (f Flags) HasCommLost() bool {
	return f&FlagCommLost != 0
}

// IsRemoteForced returns true if the value was forced by a remote entity
func (f Flags) IsRemoteForced() bool {
	return f&FlagRemoteForced != 0
}

// IsLocalForced returns true if the value was forced locally
func (f Flags) IsLocalForced() bool {
	return f&FlagLocalForced != 0
}

// IsForced returns true if the value was forced (remote or local)
func (f Flags) IsForced() bool {
	return f.IsRemoteForced() || f.IsLocalForced()
}

// IsOverRange returns true if the value exceeds the measurement range
func (f Flags) IsOverRange() bool {
	return f&FlagOverRange != 0
}

// HasReferenceErr returns true if there's a reference error
func (f Flags) HasReferenceErr() bool {
	return f&FlagReferenceErr != 0
}

// IsGood returns true if the quality is good (online and no error flags)
func (f Flags) IsGood() bool {
	return f.IsOnline() && !f.HasCommLost() && !f.HasReferenceErr()
}

// WithOnline returns a copy of flags with online bit set
func (f Flags) WithOnline(online bool) Flags {
	if online {
		return f | FlagOnline
	}
	return f &^ FlagOnline
}

// WithRestart returns a copy of flags with restart bit set
func (f Flags) WithRestart(restart bool) Flags {
	if restart {
		return f | FlagRestart
	}
	return f &^ FlagRestart
}
