package references

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/config"
)

var errInvalidDataOvlStringTable = errors.New("invalid DATA.OVL string table")

// DATA.OVL contains the compressed conversation-word list at offset 0x104c,
// spanning 0x24e bytes. Token expansion is described in ref:TALK-0002.

const (
	nLocationNameOffset  = 0xa4d
	nLocationNameOffset2 = 0xac1
)
const (
	nTalkCompressedWordsOffset = 0x104c
	nTalkCompressedWordsLength = 0x24e
)

type DataOvl struct {
	LocationNames   []string `json:"location_names"   yaml:"location_names"`
	CompressedWords []string `json:"compressed_words" yaml:"compressed_words"`
	Filenames       []string `json:"filenames"        yaml:"filenames"`
}

func readNullTerminatedStrings(data *[]byte, offset, n int) ([]string, error) {
	if offset < 0 || offset >= len(*data) {
		return nil, fmt.Errorf("%w: offset %#x outside %d-byte data", errInvalidDataOvlStringTable, offset, len(*data))
	}

	var result []string
	start := offset
	for i := 0; i < n; i++ {
		// Find the null terminator (0 byte) to end the current string
		end := start
		for end < len(*data) && (*data)[end] != 0 {
			end++
		}

		// Check if we have reached the end of data without finding enough null terminators
		if end == len(*data) {
			return nil, fmt.Errorf("unexpected end of data before reading %d strings", n)
		}

		// Extract the string
		result = append(result, string((*data)[start:end]))

		// Move the start position to the byte after the null terminator
		start = end + 1
	}

	return result, nil
}

// Reads all null-terminated strings within a byte range (not a fixed count)
func readNullTerminatedStringsByLength(data *[]byte, offset, length int) ([]string, error) {
	if offset < 0 || length < 0 || offset > len(*data)-length {
		return nil, fmt.Errorf("%w: range %#x:%#x outside %d-byte data",
			errInvalidDataOvlStringTable, offset, offset+length, len(*data))
	}

	var result []string
	end := offset + length
	pos := offset
	for pos < end {
		if pos >= len(*data) {
			break
		}
		strEnd := pos
		for strEnd < end && strEnd < len(*data) && (*data)[strEnd] != 0 {
			strEnd++
		}
		result = append(result, string((*data)[pos:strEnd]))
		pos = strEnd + 1
	}

	return result, nil
}

// NewDataOvl parses reference tables from the configured DATA.OVL bytes.
func NewDataOvl(config *config.UltimaVConfiguration) (*DataOvl, error) {
	dataOvl := DataOvl{}

	var err error
	dataOvl.LocationNames, err = readNullTerminatedStrings(&config.RawDataOvl, nLocationNameOffset, int(IolosHut))
	if err != nil {
		return nil, fmt.Errorf("reading primary location names: %w", err)
	}
	dataOvl.LocationNames = append([]string{""}, dataOvl.LocationNames...)
	dataOvl.LocationNames = append(dataOvl.LocationNames,
		[]string{"SUTEK'S HUT", "SIN VRAAL'S HUT", "GRENDAL'S HUT", "LORD BRITISH'S CASTLE", "PALACE OF BLACKTHORN"}...)
	secondHalf, err := readNullTerminatedStrings(&config.RawDataOvl, nLocationNameOffset2, 27-int(IolosHut))
	if err != nil {
		return nil, fmt.Errorf("reading secondary location names: %w", err)
	}
	dataOvl.LocationNames = append(dataOvl.LocationNames, secondHalf...)

	dungeonNames, err := readNullTerminatedStrings(&config.RawDataOvl, 0xb5e, 8)
	if err != nil {
		return nil, fmt.Errorf("reading dungeon names: %w", err)
	}
	dataOvl.LocationNames = append(dataOvl.LocationNames, dungeonNames...)

	// Use new function for compressed words
	dataOvl.CompressedWords, err = readNullTerminatedStringsByLength(
		&config.RawDataOvl, nTalkCompressedWordsOffset, nTalkCompressedWordsLength)
	if err != nil {
		return nil, fmt.Errorf("reading compressed talk words: %w", err)
	}

	// Add filenames parsing
	const nFilenamesOffset = 0x129a
	const nFilenamesLength = 0x11c
	dataOvl.Filenames, err = readNullTerminatedStringsByLength(
		&config.RawDataOvl, nFilenamesOffset, nFilenamesLength)
	if err != nil {
		return nil, fmt.Errorf("reading filenames: %w", err)
	}

	return &dataOvl, nil
}
