package dnp3

import (
	"avaneesh/dnp3-go/pkg/channel"
	"avaneesh/dnp3-go/pkg/internal/logger"
	"avaneesh/dnp3-go/pkg/outstation"
)

// newOutstation creates a new outstation instance
func newOutstation(config OutstationConfig, callbacks OutstationCallbacks, ch *channel.Channel, log logger.Logger) (Outstation, error) {
	return outstation.New(config, callbacks, ch, log)
}

// NewUpdateBuilder creates a new update builder
func NewUpdateBuilder() *UpdateBuilder {
	// Internal builder is in outstation package
	return &UpdateBuilder{
		builder: outstation.NewUpdateBuilder(),
	}
}

// UpdateBuilder wraps the internal update builder
type UpdateBuilder struct {
	builder *outstation.UpdateBuilder
}

// UpdateBinary updates a binary point
func (b *UpdateBuilder) UpdateBinary(value types.Binary, index uint16, mode EventMode) *UpdateBuilder {
	b.builder.UpdateBinary(value, index, mode)
	return b
}

// UpdateAnalog updates an analog point
func (b *UpdateBuilder) UpdateAnalog(value types.Analog, index uint16, mode EventMode) *UpdateBuilder {
	b.builder.UpdateAnalog(value, index, mode)
	return b
}

// UpdateCounter updates a counter point
func (b *UpdateBuilder) UpdateCounter(value types.Counter, index uint16, mode EventMode) *UpdateBuilder {
	b.builder.UpdateCounter(value, index, mode)
	return b
}

// Build builds the updates
func (b *UpdateBuilder) Build() *Updates {
	return b.builder.Build()
}
