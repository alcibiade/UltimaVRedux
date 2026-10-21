package minimap

import (
	"errors"
	"fmt"
	"os"

	"github.com/bradhannah/UltimaVRedux/internal/files"
)

const (
	// MiscMapSize is the width and height of each decoded source mini-map.
	MiscMapSize       = 11
	miscMapStoredCols = 16
	miscMapBytes      = MiscMapSize * miscMapStoredCols
	miscMapCount      = 4
	miscMapsFileName  = "MISCMAPS.DAT"
)

var errMiscMapDataInvalid = errors.New("misc map data is invalid")

// MiscMap is one of the four padded 11x11 attract-scene maps; ref:INTRO-0005.
type MiscMap struct {
	Tiles [MiscMapSize][MiscMapSize]byte
}

// LoadMiscMap loads one of the original prison, shrine, Codex, or endgame maps.
func LoadMiscMap(dataDir string, index int) (MiscMap, error) {
	path := files.Resolve(dataDir, miscMapsFileName)
	data, err := os.ReadFile(path) //nolint:gosec // Configured original-data directory is user-selected.
	if err != nil {
		return MiscMap{}, fmt.Errorf("reading %s: %w", path, err)
	}

	return DecodeMiscMap(data, index)
}

// DecodeMiscMap decodes one padded [11][16] record from MISCMAPS.DAT.
func DecodeMiscMap(data []byte, index int) (MiscMap, error) {
	if index < 0 || index >= miscMapCount {
		return MiscMap{}, fmt.Errorf("%w: map index %d is out of range", errMiscMapDataInvalid, index)
	}
	offset := index * miscMapBytes
	if len(data) < offset+miscMapBytes {
		return MiscMap{}, fmt.Errorf(
			"%w: MISCMAPS.DAT is %d bytes, need at least %d",
			errMiscMapDataInvalid,
			len(data),
			offset+miscMapBytes,
		)
	}

	result := MiscMap{Tiles: [MiscMapSize][MiscMapSize]byte{}}
	for row := range MiscMapSize {
		rowOffset := offset + row*miscMapStoredCols
		for column := range MiscMapSize {
			result.Tiles[column][row] = data[rowOffset+column]
		}
	}

	return result, nil
}
