package types

// IIN (Internal Indication) represents DNP3 internal indication bits
// These indicate the internal state of a device
type IIN struct {
	IIN1 uint8 // First IIN byte
	IIN2 uint8 // Second IIN byte
}

// IIN1 bit masks
const (
	IIN1AllStations       uint8 = 0x01 // Broadcast message received
	IIN1Class1Events      uint8 = 0x02 // Class 1 events available
	IIN1Class2Events      uint8 = 0x04 // Class 2 events available
	IIN1Class3Events      uint8 = 0x08 // Class 3 events available
	IIN1NeedTime          uint8 = 0x10 // Device needs time synchronization
	IIN1LocalControl      uint8 = 0x20 // Device in local control mode
	IIN1DeviceTrouble     uint8 = 0x40 // Device trouble or malfunction
	IIN1DeviceRestart     uint8 = 0x80 // Device restart detected
)

// IIN2 bit masks
const (
	IIN2NoFuncCodeSupport uint8 = 0x01 // Function code not supported
	IIN2ObjectUnknown     uint8 = 0x02 // Object unknown
	IIN2ParameterError    uint8 = 0x04 // Parameter error
	IIN2EventBufferOverflow uint8 = 0x08 // Event buffer overflow
	IIN2AlreadyExecuting  uint8 = 0x10 // Operation already executing
	IIN2ConfigCorrupt     uint8 = 0x20 // Configuration corrupt
	IIN2Reserved1         uint8 = 0x40 // Reserved
	IIN2Reserved2         uint8 = 0x80 // Reserved
)

// HasClass1Events returns true if Class 1 events are available
func (iin IIN) HasClass1Events() bool {
	return iin.IIN1&IIN1Class1Events != 0
}

// HasClass2Events returns true if Class 2 events are available
func (iin IIN) HasClass2Events() bool {
	return iin.IIN1&IIN1Class2Events != 0
}

// HasClass3Events returns true if Class 3 events are available
func (iin IIN) HasClass3Events() bool {
	return iin.IIN1&IIN1Class3Events != 0
}

// HasAnyClassEvents returns true if any class events are available
func (iin IIN) HasAnyClassEvents() bool {
	return iin.HasClass1Events() || iin.HasClass2Events() || iin.HasClass3Events()
}

// NeedTime returns true if the device needs time synchronization
func (iin IIN) NeedTime() bool {
	return iin.IIN1&IIN1NeedTime != 0
}

// IsInLocalControl returns true if the device is in local control mode
func (iin IIN) IsInLocalControl() bool {
	return iin.IIN1&IIN1LocalControl != 0
}

// HasDeviceTrouble returns true if device trouble is indicated
func (iin IIN) HasDeviceTrouble() bool {
	return iin.IIN1&IIN1DeviceTrouble != 0
}

// HasDeviceRestart returns true if device restart was detected
func (iin IIN) HasDeviceRestart() bool {
	return iin.IIN1&IIN1DeviceRestart != 0
}

// HasParameterError returns true if there was a parameter error
func (iin IIN) HasParameterError() bool {
	return iin.IIN2&IIN2ParameterError != 0
}

// HasEventBufferOverflow returns true if event buffer overflow occurred
func (iin IIN) HasEventBufferOverflow() bool {
	return iin.IIN2&IIN2EventBufferOverflow != 0
}

// Clear returns an IIN with all bits cleared
func (iin IIN) Clear() IIN {
	return IIN{IIN1: 0, IIN2: 0}
}

// SetClass1Events sets or clears the Class 1 events bit
func (iin IIN) SetClass1Events(value bool) IIN {
	if value {
		iin.IIN1 |= IIN1Class1Events
	} else {
		iin.IIN1 &^= IIN1Class1Events
	}
	return iin
}

// SetClass2Events sets or clears the Class 2 events bit
func (iin IIN) SetClass2Events(value bool) IIN {
	if value {
		iin.IIN1 |= IIN1Class2Events
	} else {
		iin.IIN1 &^= IIN1Class2Events
	}
	return iin
}

// SetClass3Events sets or clears the Class 3 events bit
func (iin IIN) SetClass3Events(value bool) IIN {
	if value {
		iin.IIN1 |= IIN1Class3Events
	} else {
		iin.IIN1 &^= IIN1Class3Events
	}
	return iin
}
