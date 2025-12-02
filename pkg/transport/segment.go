package transport

// Transport layer constants
const (
	MaxSegmentSize = 249 // Maximum transport segment size (250 - 1 for header)
	HeaderSize     = 1   // Transport header is 1 byte
)

// Transport header bits
const (
	TransportFIN uint8 = 0x80 // Final segment bit
	TransportFIR uint8 = 0x40 // First segment bit
	TransportSeqMask uint8 = 0x3F // Sequence number mask (6 bits)
)

// Segment represents a transport layer segment
type Segment struct {
	FIN  bool   // Final segment
	FIR  bool   // First segment
	Seq  uint8  // Sequence number (0-63)
	Data []byte // Segment data
}

// NewSegment creates a new transport segment
func NewSegment(fir, fin bool, seq uint8, data []byte) *Segment {
	return &Segment{
		FIR:  fir,
		FIN:  fin,
		Seq:  seq & TransportSeqMask,
		Data: data,
	}
}

// BuildHeader builds the transport header byte
func (s *Segment) BuildHeader() uint8 {
	header := s.Seq & TransportSeqMask
	if s.FIR {
		header |= TransportFIR
	}
	if s.FIN {
		header |= TransportFIN
	}
	return header
}

// ParseHeader parses transport header byte
func ParseHeader(header uint8) (fir, fin bool, seq uint8) {
	fir = (header & TransportFIR) != 0
	fin = (header & TransportFIN) != 0
	seq = header & TransportSeqMask
	return
}

// Serialize converts segment to wire format
func (s *Segment) Serialize() []byte {
	result := make([]byte, 1+len(s.Data))
	result[0] = s.BuildHeader()
	copy(result[1:], s.Data)
	return result
}

// SegmentData breaks APDU data into transport segments
func SegmentData(data []byte, startSeq uint8) []*Segment {
	if len(data) == 0 {
		return nil
	}

	var segments []*Segment
	seq := startSeq & TransportSeqMask

	for offset := 0; offset < len(data); {
		// Determine segment size
		remaining := len(data) - offset
		segSize := MaxSegmentSize
		if remaining < segSize {
			segSize = remaining
		}

		// Create segment
		segData := data[offset : offset+segSize]
		fir := (offset == 0)
		fin := (offset+segSize >= len(data))

		segment := NewSegment(fir, fin, seq, segData)
		segments = append(segments, segment)

		offset += segSize
		seq = (seq + 1) & TransportSeqMask
	}

	return segments
}
