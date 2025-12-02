package app

import (
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrInvalidObjectHeader = errors.New("invalid object header")
	ErrInvalidRange        = errors.New("invalid range specification")
	ErrInsufficientData    = errors.New("insufficient data")
)

// Parser parses DNP3 object headers and data
type Parser struct {
	data   []byte
	offset int
}

// NewParser creates a new parser for object data
func NewParser(data []byte) *Parser {
	return &Parser{
		data:   data,
		offset: 0,
	}
}

// HasMore returns true if there is more data to parse
func (p *Parser) HasMore() bool {
	return p.offset < len(p.data)
}

// Remaining returns the number of bytes remaining
func (p *Parser) Remaining() int {
	return len(p.data) - p.offset
}

// ReadObjectHeader reads an object header
func (p *Parser) ReadObjectHeader() (*ObjectHeader, error) {
	if p.Remaining() < 3 {
		return nil, ErrInsufficientData
	}

	header := &ObjectHeader{
		Group:     p.data[p.offset],
		Variation: p.data[p.offset+1],
		Qualifier: QualifierCode(p.data[p.offset+2]),
	}
	p.offset += 3

	// Parse range based on qualifier
	var err error
	switch header.Qualifier {
	case Qualifier8BitStartStop:
		header.Range, err = p.readStartStop8()
	case Qualifier16BitStartStop:
		header.Range, err = p.readStartStop16()
	case Qualifier32BitStartStop:
		header.Range, err = p.readStartStop32()
	case Qualifier8BitCount:
		header.Range, err = p.readCount8()
	case Qualifier16BitCount:
		header.Range, err = p.readCount16()
	case Qualifier32BitCount:
		header.Range, err = p.readCount32()
	case QualifierNoRange:
		header.Range = NoRange{}
	default:
		return nil, fmt.Errorf("unsupported qualifier: 0x%02X", header.Qualifier)
	}

	if err != nil {
		return nil, err
	}

	return header, nil
}

// readStartStop8 reads 8-bit start-stop range
func (p *Parser) readStartStop8() (Range, error) {
	if p.Remaining() < 2 {
		return nil, ErrInsufficientData
	}
	r := StartStopRange{
		Start: uint32(p.data[p.offset]),
		Stop:  uint32(p.data[p.offset+1]),
	}
	p.offset += 2
	return r, nil
}

// readStartStop16 reads 16-bit start-stop range
func (p *Parser) readStartStop16() (Range, error) {
	if p.Remaining() < 4 {
		return nil, ErrInsufficientData
	}
	r := StartStopRange{
		Start: uint32(binary.LittleEndian.Uint16(p.data[p.offset:])),
		Stop:  uint32(binary.LittleEndian.Uint16(p.data[p.offset+2:])),
	}
	p.offset += 4
	return r, nil
}

// readStartStop32 reads 32-bit start-stop range
func (p *Parser) readStartStop32() (Range, error) {
	if p.Remaining() < 8 {
		return nil, ErrInsufficientData
	}
	r := StartStopRange{
		Start: binary.LittleEndian.Uint32(p.data[p.offset:]),
		Stop:  binary.LittleEndian.Uint32(p.data[p.offset+4:]),
	}
	p.offset += 8
	return r, nil
}

// readCount8 reads 8-bit count
func (p *Parser) readCount8() (Range, error) {
	if p.Remaining() < 1 {
		return nil, ErrInsufficientData
	}
	r := CountRange{
		Count: uint32(p.data[p.offset]),
	}
	p.offset++
	return r, nil
}

// readCount16 reads 16-bit count
func (p *Parser) readCount16() (Range, error) {
	if p.Remaining() < 2 {
		return nil, ErrInsufficientData
	}
	r := CountRange{
		Count: uint32(binary.LittleEndian.Uint16(p.data[p.offset:])),
	}
	p.offset += 2
	return r, nil
}

// readCount32 reads 32-bit count
func (p *Parser) readCount32() (Range, error) {
	if p.Remaining() < 4 {
		return nil, ErrInsufficientData
	}
	r := CountRange{
		Count: binary.LittleEndian.Uint32(p.data[p.offset:]),
	}
	p.offset += 4
	return r, nil
}

// ReadBytes reads n bytes from the parser
func (p *Parser) ReadBytes(n int) ([]byte, error) {
	if p.Remaining() < n {
		return nil, ErrInsufficientData
	}
	data := p.data[p.offset : p.offset+n]
	p.offset += n
	return data, nil
}

// Skip skips n bytes
func (p *Parser) Skip(n int) error {
	if p.Remaining() < n {
		return ErrInsufficientData
	}
	p.offset += n
	return nil
}

// Reset resets the parser to the beginning
func (p *Parser) Reset() {
	p.offset = 0
}

// GetCount returns the number of items in a range
func GetCount(r Range) uint32 {
	switch v := r.(type) {
	case StartStopRange:
		if v.Stop >= v.Start {
			return v.Stop - v.Start + 1
		}
		return 0
	case CountRange:
		return v.Count
	case NoRange:
		return 0
	default:
		return 0
	}
}
