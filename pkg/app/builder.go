package app

import (
	"bytes"
	"encoding/binary"
)

// ObjectBuilder helps construct object headers and data
type ObjectBuilder struct {
	buf bytes.Buffer
}

// NewObjectBuilder creates a new object builder
func NewObjectBuilder() *ObjectBuilder {
	return &ObjectBuilder{}
}

// AddHeader adds an object header without data
func (b *ObjectBuilder) AddHeader(group, variation uint8, qualifier QualifierCode, rng Range) error {
	// Write group, variation, qualifier
	b.buf.WriteByte(group)
	b.buf.WriteByte(variation)
	b.buf.WriteByte(uint8(qualifier))

	// Write range based on qualifier
	switch qualifier {
	case Qualifier8BitStartStop:
		if r, ok := rng.(StartStopRange); ok {
			b.buf.WriteByte(uint8(r.Start))
			b.buf.WriteByte(uint8(r.Stop))
		}
	case Qualifier16BitStartStop:
		if r, ok := rng.(StartStopRange); ok {
			binary.Write(&b.buf, binary.LittleEndian, uint16(r.Start))
			binary.Write(&b.buf, binary.LittleEndian, uint16(r.Stop))
		}
	case Qualifier32BitStartStop:
		if r, ok := rng.(StartStopRange); ok {
			binary.Write(&b.buf, binary.LittleEndian, uint32(r.Start))
			binary.Write(&b.buf, binary.LittleEndian, uint32(r.Stop))
		}
	case Qualifier8BitCount:
		if r, ok := rng.(CountRange); ok {
			b.buf.WriteByte(uint8(r.Count))
		}
	case Qualifier16BitCount:
		if r, ok := rng.(CountRange); ok {
			binary.Write(&b.buf, binary.LittleEndian, uint16(r.Count))
		}
	case Qualifier32BitCount:
		if r, ok := rng.(CountRange); ok {
			binary.Write(&b.buf, binary.LittleEndian, uint32(r.Count))
		}
	case QualifierNoRange:
		// No range to write
	}

	return nil
}

// AddHeaderWithData adds an object header followed by raw data
func (b *ObjectBuilder) AddHeaderWithData(group, variation uint8, qualifier QualifierCode, rng Range, data []byte) error {
	if err := b.AddHeader(group, variation, qualifier, rng); err != nil {
		return err
	}
	b.buf.Write(data)
	return nil
}

// AddRawData adds raw data without a header
func (b *ObjectBuilder) AddRawData(data []byte) {
	b.buf.Write(data)
}

// AddByte adds a single byte
func (b *ObjectBuilder) AddByte(val uint8) {
	b.buf.WriteByte(val)
}

// AddUint16 adds a 16-bit value (little endian)
func (b *ObjectBuilder) AddUint16(val uint16) {
	binary.Write(&b.buf, binary.LittleEndian, val)
}

// AddUint32 adds a 32-bit value (little endian)
func (b *ObjectBuilder) AddUint32(val uint32) {
	binary.Write(&b.buf, binary.LittleEndian, val)
}

// AddInt16 adds a signed 16-bit value (little endian)
func (b *ObjectBuilder) AddInt16(val int16) {
	binary.Write(&b.buf, binary.LittleEndian, val)
}

// AddInt32 adds a signed 32-bit value (little endian)
func (b *ObjectBuilder) AddInt32(val int32) {
	binary.Write(&b.buf, binary.LittleEndian, val)
}

// AddFloat32 adds a 32-bit float (little endian)
func (b *ObjectBuilder) AddFloat32(val float32) {
	binary.Write(&b.buf, binary.LittleEndian, val)
}

// AddFloat64 adds a 64-bit float (little endian)
func (b *ObjectBuilder) AddFloat64(val float64) {
	binary.Write(&b.buf, binary.LittleEndian, val)
}

// Build returns a copy of the constructed object data
func (b *ObjectBuilder) Build() []byte {
	// Return a copy to avoid issues with Reset()
	data := b.buf.Bytes()
	result := make([]byte, len(data))
	copy(result, data)
	return result
}

// Reset clears the builder for reuse
func (b *ObjectBuilder) Reset() {
	b.buf.Reset()
}

// Helper functions for common patterns

// BuildClassRead builds a read request for specific classes
func BuildClassRead(classes ...ClassField) []byte {
	builder := NewObjectBuilder()

	for _, class := range classes {
		var variation uint8
		switch class {
		case Class0:
			variation = 1
		case Class1:
			variation = 2
		case Class2:
			variation = 3
		case Class3:
			variation = 4
		default:
			continue
		}

		builder.AddHeader(GroupClass0Data, variation, QualifierNoRange, NoRange{})
	}

	return builder.Build()
}

// BuildIntegrityPoll builds a Class 0 (integrity) read request
func BuildIntegrityPoll() []byte {
	return BuildClassRead(Class0)
}

// BuildEventPoll builds a Class 1,2,3 (event) read request
func BuildEventPoll() []byte {
	return BuildClassRead(Class1, Class2, Class3)
}

// BuildEnableUnsolicited builds objects for enable unsolicited request
func BuildEnableUnsolicited(classes ...ClassField) []byte {
	builder := NewObjectBuilder()

	for _, class := range classes {
		var variation uint8
		switch class {
		case Class1:
			variation = 2
		case Class2:
			variation = 3
		case Class3:
			variation = 4
		default:
			continue
		}

		builder.AddHeader(GroupClass0Data, variation, QualifierNoRange, NoRange{})
	}

	return builder.Build()
}

// BuildDisableUnsolicited builds objects for disable unsolicited request
func BuildDisableUnsolicited(classes ...ClassField) []byte {
	return BuildEnableUnsolicited(classes...) // Same object structure
}

// BuildRangeRead builds a read request for specific object range
func BuildRangeRead(group, variation uint8, start, stop uint32) []byte {
	builder := NewObjectBuilder()

	// Choose appropriate qualifier based on range size
	var qualifier QualifierCode
	if stop <= 255 && start <= 255 {
		qualifier = Qualifier8BitStartStop
	} else if stop <= 65535 && start <= 65535 {
		qualifier = Qualifier16BitStartStop
	} else {
		qualifier = Qualifier32BitStartStop
	}

	builder.AddHeader(group, variation, qualifier, StartStopRange{Start: start, Stop: stop})
	return builder.Build()
}
