package legacy

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	lzwClearCode       = 256
	lzwEndCode         = 257
	lzwFirstCode       = 258
	lzwInitialWidth    = 9
	lzwMaximumWidth    = 12
	lzwDictionarySize  = 1 << lzwMaximumWidth
	maximumExpandedLen = 64 << 20
)

// Decompress expands the LZW wrapper used by shipped Ultima V graphics files.
func Decompress(data []byte) ([]byte, error) {
	if len(data) < 4 {
		return nil, errors.New("compressed graphics header is truncated")
	}
	expected := int(binary.LittleEndian.Uint32(data[:4]))
	if expected < 0 || expected > maximumExpandedLen {
		return nil, fmt.Errorf("expanded graphics length %d is invalid", expected)
	}

	reader := bitReader{data: data[4:], byteIndex: 0, buffer: 0, bits: 0}
	dictionary := make([][]byte, lzwDictionarySize)
	for value := range 256 {
		dictionary[value] = []byte{byte(value)}
	}

	output := make([]byte, 0, expected)
	width, nextCode := lzwInitialWidth, lzwFirstCode
	var previous []byte
	first := true
	for {
		code, err := reader.read(width)
		if err != nil {
			return nil, fmt.Errorf("reading %d-bit LZW code: %w", width, err)
		}
		if first {
			if code != lzwClearCode {
				return nil, fmt.Errorf("first LZW code is %d, want clear code %d", code, lzwClearCode)
			}
			first = false
		}

		switch code {
		case lzwClearCode:
			width, nextCode, previous = lzwInitialWidth, lzwFirstCode, nil

			continue
		case lzwEndCode:
			if len(output) != expected {
				return nil, fmt.Errorf("expanded %d bytes, want %d", len(output), expected)
			}
			if err := reader.requireZeroPadding(); err != nil {
				return nil, err
			}

			return output, nil
		}

		current, err := resolveLZWCode(dictionary, code, nextCode, previous)
		if err != nil {
			return nil, err
		}
		if len(output)+len(current) > expected {
			return nil, fmt.Errorf("LZW output exceeds declared length %d", expected)
		}
		output = append(output, current...)

		if previous != nil && nextCode < lzwDictionarySize {
			entry := make([]byte, len(previous)+1)
			copy(entry, previous)
			entry[len(previous)] = current[0]
			dictionary[nextCode] = entry
			nextCode++
			if width < lzwMaximumWidth && nextCode == 1<<width {
				width++
			}
		}
		previous = current
	}
}

func resolveLZWCode(dictionary [][]byte, code, nextCode int, previous []byte) ([]byte, error) {
	if previous == nil {
		if code > 255 {
			return nil, fmt.Errorf("first code after clear is nonliteral %d", code)
		}

		return dictionary[code], nil
	}
	if code < nextCode {
		if code >= len(dictionary) || dictionary[code] == nil {
			return nil, fmt.Errorf("LZW code %d has no dictionary entry", code)
		}

		return dictionary[code], nil
	}
	if code == nextCode && nextCode < len(dictionary) {
		current := make([]byte, len(previous)+1)
		copy(current, previous)
		current[len(previous)] = previous[0]

		return current, nil
	}

	return nil, fmt.Errorf("invalid forward LZW code %d with next code %d", code, nextCode)
}

type bitReader struct {
	data      []byte
	byteIndex int
	buffer    uint64
	bits      int
}

func (r *bitReader) read(width int) (int, error) {
	for r.bits < width {
		if r.byteIndex >= len(r.data) {
			return 0, errors.New("compressed graphics stream ended before end code")
		}
		r.buffer |= uint64(r.data[r.byteIndex]) << r.bits
		r.bits += 8
		r.byteIndex++
	}
	mask := uint64(1<<width) - 1
	result := int(r.buffer & mask) //nolint:gosec // The mask limits codes to 12 bits.
	r.buffer >>= width
	r.bits -= width

	return result, nil
}

func (r *bitReader) requireZeroPadding() error {
	if r.byteIndex != len(r.data) {
		return fmt.Errorf("compressed graphics stream has %d trailing bytes", len(r.data)-r.byteIndex)
	}
	if r.buffer != 0 {
		return errors.New("compressed graphics stream has nonzero padding bits")
	}

	return nil
}
