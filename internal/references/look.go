package references

import (
	"fmt"
	"io"
	"os"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const totalLooks = 512

type LookReferences struct {
	GameConfig       *config.UltimaVConfiguration `json:"game_config"       yaml:"game_config"`
	LookOffsets      [totalLooks]int              `json:"look_offsets"      yaml:"look_offsets"`
	LookData         []byte                       `json:"look_data"         yaml:"look_data"`
	TileDescriptions map[int]string               `json:"tile_descriptions" yaml:"tile_descriptions"`
}

// NewLookReferences loads tile descriptions from the configured LOOK2.DAT file.
func NewLookReferences(gameConfig *config.UltimaVConfiguration) (*LookReferences, error) {
	lookRefs := &LookReferences{}
	lookRefs.GameConfig = gameConfig

	lookDataPath := gameConfig.GetLookDataFilePath()
	logging.Infof(logging.CategorySystem, "Look data file path: %s", lookDataPath)
	var err error
	lookRefs.LookData, err = lookRefs.getLookFileAsBytes()
	if err != nil {
		return nil, err
	}
	if len(lookRefs.LookData) < totalLooks*2 {
		return nil, fmt.Errorf("reading look offsets from %s: expected at least %d bytes, got %d: %w",
			lookDataPath, totalLooks*2, len(lookRefs.LookData), io.ErrUnexpectedEOF)
	}

	count := 0
	for i := 0; i < totalLooks*2; i += 2 {
		lookRefs.LookOffsets[count] = int((lookRefs.LookData)[i]) | int((lookRefs.LookData)[i+1])<<8
		count++
	}

	// Populate tile descriptions
	lookRefs.TileDescriptions = make(map[int]string)
	for i := range totalLooks {
		lookRefs.TileDescriptions[i] = lookRefs.GetTileLookDescription(indexes.SpriteIndex(i))
	}

	return lookRefs, nil
}

func (l *LookReferences) getLookFileAsBytes() ([]byte, error) {
	lookData, err := os.ReadFile(l.GameConfig.GetLookDataFilePath())
	if err != nil {
		return nil, fmt.Errorf("reading look data file %s: %w", l.GameConfig.GetLookDataFilePath(), err)
	}

	return lookData, nil
}

func (l *LookReferences) GetTileLookDescription(tileIndex indexes.SpriteIndex) string {
	// Bounds check to prevent crash with unknown tiles
	if int(tileIndex) < 0 || int(tileIndex) >= totalLooks {
		return "something unusual"
	}

	// Check if LookOffsets is properly initialized
	if len(l.LookOffsets) <= int(tileIndex) {
		return "something unusual"
	}

	// Check if the offset is within bounds of LookData
	offset := l.LookOffsets[tileIndex]
	if offset < 0 || offset >= len(l.LookData) {
		return "something unusual"
	}

	lookBytes := make([]byte, 0)

	for i := offset; i < len(l.LookData)-1; i++ {
		if l.LookData[i] != 0 {
			lookBytes = append(lookBytes, l.LookData[i])

			continue
		}

		break
	}

	// If we didn't get any description, return a fallback
	if len(lookBytes) == 0 {
		return "something"
	}

	return string(lookBytes)
}
