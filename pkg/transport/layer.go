package transport

// Layer represents the transport layer
type Layer struct {
	rxReassembler *Reassembler
	txSequence    uint8
}

// NewLayer creates a new transport layer
func NewLayer() *Layer {
	return &Layer{
		rxReassembler: NewReassembler(),
		txSequence:    0,
	}
}

// Receive processes received data and returns complete APDU if available
func (l *Layer) Receive(data []byte) ([]byte, error) {
	if len(data) < HeaderSize {
		return nil, ErrMissingFIR
	}

	// Parse header
	fir, fin, seq := ParseHeader(data[0])

	// Create segment
	segment := &Segment{
		FIR:  fir,
		FIN:  fin,
		Seq:  seq,
		Data: data[1:],
	}

	// Process through reassembler
	return l.rxReassembler.Process(segment)
}

// Send segments APDU data for transmission
// Returns list of transport layer frames ready for link layer
func (l *Layer) Send(apdu []byte) [][]byte {
	if len(apdu) == 0 {
		return nil
	}

	// Segment the APDU
	segments := SegmentData(apdu, l.txSequence)

	// Update sequence for next transmission
	l.txSequence = (l.txSequence + uint8(len(segments))) & TransportSeqMask

	// Serialize segments
	result := make([][]byte, len(segments))
	for i, seg := range segments {
		result[i] = seg.Serialize()
	}

	return result
}

// Reset resets the transport layer state
func (l *Layer) Reset() {
	l.rxReassembler.Reset()
	l.txSequence = 0
}
