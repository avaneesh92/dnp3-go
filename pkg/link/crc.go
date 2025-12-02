package link

// DNP3 CRC-16 implementation
// DNP3 uses CRC-16 with polynomial 0x3D65 (reversed 0xA6BC)

var crcTable [256]uint16

func init() {
	// Build CRC table using DNP3 polynomial
	const poly uint16 = 0xA6BC // DNP3 polynomial (reversed)

	for i := 0; i < 256; i++ {
		crc := uint16(i)
		for j := 0; j < 8; j++ {
			if crc&1 != 0 {
				crc = (crc >> 1) ^ poly
			} else {
				crc >>= 1
			}
		}
		crcTable[i] = crc
	}
}

// CalculateCRC calculates DNP3 CRC-16 for the given data
func CalculateCRC(data []byte) uint16 {
	crc := uint16(0)
	for _, b := range data {
		crc = crcTable[(byte(crc)^b)&0xFF] ^ (crc >> 8)
	}
	return ^crc // DNP3 inverts the final CRC
}

// VerifyCRC verifies that data has correct CRC appended
// Data should include the 2-byte CRC at the end
func VerifyCRC(data []byte) bool {
	if len(data) < 2 {
		return false
	}

	// Calculate CRC for all data except last 2 bytes
	calculated := CalculateCRC(data[:len(data)-2])

	// Extract CRC from last 2 bytes (little-endian)
	received := uint16(data[len(data)-2]) | (uint16(data[len(data)-1]) << 8)

	return calculated == received
}

// AppendCRC appends CRC to data and returns new slice
func AppendCRC(data []byte) []byte {
	crc := CalculateCRC(data)
	result := make([]byte, len(data)+2)
	copy(result, data)
	result[len(data)] = byte(crc)
	result[len(data)+1] = byte(crc >> 8)
	return result
}

// AddCRCs adds CRC bytes to data in 16-byte blocks (DNP3 framing)
// Returns new slice with CRCs inserted every 16 bytes
func AddCRCs(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}

	// Calculate how many blocks we need
	numBlocks := (len(data) + 15) / 16 // Round up
	result := make([]byte, 0, len(data)+numBlocks*2)

	for i := 0; i < len(data); i += 16 {
		end := i + 16
		if end > len(data) {
			end = len(data)
		}

		block := data[i:end]
		result = append(result, block...)

		// Add CRC for this block
		crc := CalculateCRC(block)
		result = append(result, byte(crc), byte(crc>>8))
	}

	return result
}

// RemoveCRCs removes and verifies CRC bytes from data
// Expects CRCs every 16 bytes + 2-byte CRC
// Returns data without CRCs and any error
func RemoveCRCs(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, nil
	}

	result := make([]byte, 0, len(data))
	pos := 0

	for pos < len(data) {
		// Determine block size (16 bytes or remaining data)
		blockSize := 16
		if pos+blockSize+2 > len(data) {
			blockSize = len(data) - pos - 2
			if blockSize <= 0 {
				return nil, ErrInvalidCRC
			}
		}

		// Extract block and CRC
		block := data[pos : pos+blockSize]
		if pos+blockSize+2 > len(data) {
			return nil, ErrInvalidCRC
		}

		receivedCRC := uint16(data[pos+blockSize]) | (uint16(data[pos+blockSize+1]) << 8)
		calculatedCRC := CalculateCRC(block)

		if receivedCRC != calculatedCRC {
			return nil, ErrInvalidCRC
		}

		result = append(result, block...)
		pos += blockSize + 2
	}

	return result, nil
}
