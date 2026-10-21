package legacy

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecompress(t *testing.T) {
	compressed := packLZWCodes([]int{lzwClearCode, 'A', 'B', lzwFirstCode, lzwEndCode}, lzwInitialWidth)
	data := make([]byte, 4, 4+len(compressed))
	binary.LittleEndian.PutUint32(data, 4)
	data = append(data, compressed...)

	actual, err := Decompress(data)

	require.NoError(t, err)
	assert.Equal(t, []byte("ABAB"), actual)
}

func TestDecompressRejectsMissingInitialClear(t *testing.T) {
	compressed := packLZWCodes([]int{'A', lzwEndCode}, lzwInitialWidth)
	data := make([]byte, 4, 4+len(compressed))
	binary.LittleEndian.PutUint32(data, 1)
	data = append(data, compressed...)

	_, err := Decompress(data)

	require.ErrorContains(t, err, "want clear code")
}

func TestDecompressRejectsLengthMismatch(t *testing.T) {
	compressed := packLZWCodes([]int{lzwClearCode, 'A', lzwEndCode}, lzwInitialWidth)
	data := make([]byte, 4, 4+len(compressed))
	binary.LittleEndian.PutUint32(data, 2)
	data = append(data, compressed...)

	_, err := Decompress(data)

	require.ErrorContains(t, err, "want 2")
}

func packLZWCodes(codes []int, width int) []byte {
	var buffer uint64
	bits := 0
	result := make([]byte, 0, (len(codes)*width+7)/8)
	for _, code := range codes {
		buffer |= uint64(code) << bits
		bits += width
		for bits >= 8 {
			result = append(result, byte(buffer))
			buffer >>= 8
			bits -= 8
		}
	}
	if bits > 0 {
		result = append(result, byte(buffer))
	}

	return result
}
