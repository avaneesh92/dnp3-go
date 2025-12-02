package outstation

import (
	"avaneesh/dnp3-go/pkg/dnp3"
	"avaneesh/dnp3-go/pkg/types"
)

// UpdateBuilder builds atomic measurement updates
type UpdateBuilder struct {
	updates map[updateKey]updateValue
}

type updateKey struct {
	pointType dnp3.MeasurementType
	index     uint16
}

type updateValue struct {
	measurement interface{}
	mode        dnp3.EventMode
}

// NewUpdateBuilder creates a new update builder
func NewUpdateBuilder() *UpdateBuilder {
	return &UpdateBuilder{
		updates: make(map[updateKey]updateValue),
	}
}

// UpdateBinary updates a binary point
func (b *UpdateBuilder) UpdateBinary(value types.Binary, index uint16, mode dnp3.EventMode) *UpdateBuilder {
	key := updateKey{pointType: dnp3.MeasurementTypeBinary, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// UpdateDoubleBitBinary updates a double-bit binary point
func (b *UpdateBuilder) UpdateDoubleBitBinary(value types.DoubleBitBinary, index uint16, mode dnp3.EventMode) *UpdateBuilder {
	key := updateKey{pointType: dnp3.MeasurementTypeDoubleBitBinary, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// UpdateAnalog updates an analog point
func (b *UpdateBuilder) UpdateAnalog(value types.Analog, index uint16, mode dnp3.EventMode) *UpdateBuilder {
	key := updateKey{pointType: dnp3.MeasurementTypeAnalog, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// UpdateCounter updates a counter point
func (b *UpdateBuilder) UpdateCounter(value types.Counter, index uint16, mode dnp3.EventMode) *UpdateBuilder {
	key := updateKey{pointType: dnp3.MeasurementTypeCounter, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// UpdateFrozenCounter updates a frozen counter point
func (b *UpdateBuilder) UpdateFrozenCounter(value types.FrozenCounter, index uint16, mode dnp3.EventMode) *UpdateBuilder {
	key := updateKey{pointType: dnp3.MeasurementTypeFrozenCounter, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// UpdateBinaryOutputStatus updates a binary output status point
func (b *UpdateBuilder) UpdateBinaryOutputStatus(value types.BinaryOutputStatus, index uint16, mode dnp3.EventMode) *UpdateBuilder {
	key := updateKey{pointType: dnp3.MeasurementTypeBinaryOutputStatus, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// UpdateAnalogOutputStatus updates an analog output status point
func (b *UpdateBuilder) UpdateAnalogOutputStatus(value types.AnalogOutputStatus, index uint16, mode dnp3.EventMode) *UpdateBuilder {
	key := updateKey{pointType: dnp3.MeasurementTypeAnalogOutputStatus, index: index}
	b.updates[key] = updateValue{measurement: value, mode: mode}
	return b
}

// Build builds the updates object
func (b *UpdateBuilder) Build() *dnp3.Updates {
	// Create updates with internal data
	return &dnp3.Updates{} // Simplified - would need to expose internals properly
}

// GetUpdates returns the updates map (for internal use)
func (b *UpdateBuilder) GetUpdates() map[updateKey]updateValue {
	return b.updates
}
