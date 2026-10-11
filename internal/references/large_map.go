package references

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

const (
	// YLargeMapTiles Total map height in tiles
	YLargeMapTiles = Coordinate(256)
	// XLargeMapTiles Total map width in tiles
	XLargeMapTiles = Coordinate(256)
)

const (
	totalChunks           = 256    // Number of chunks in the map
	tilesPerChunkX        = 16     // Tiles per chunk on the x-axis
	tilesPerChunkY        = 16     // Tiles per chunk on the y-axis
	datOverlayBritMap int = 0x3886 // Offset for overlay data (adjust as needed)
)

// LargeMapReference A reference to a large map. This is a 256x256 map with 16x16 tiles.
// The map is stored in a 256x256 array of bytes, with each byte representing a tile.
type LargeMapReference struct {
	rawData [XLargeMapTiles][YLargeMapTiles]byte
}

// World represents the world of the map. Overworld or Underworld.
type World int

// World constants define the map worlds.
// OVERWORLD (0) is the surface of Britannia.
// UNDERWORLD (-1) is the underground realm, accessed via dungeons or Ararat.
const (
	OVERWORLD  World = 0
	UNDERWORLD World = -1
)

func NewLargeMapReference(gameConfig *config.UltimaVConfiguration, world World) (*LargeMapReference, error) {
	if world == OVERWORLD {
		return loadLargeMapFromFile(OVERWORLD, gameConfig)
	} else {
		return loadLargeMapFromFile(UNDERWORLD, gameConfig)
	}
}

func (m *LargeMapReference) GetSpriteIndex(x, y Coordinate) indexes.SpriteIndex {
	pos := Position{X: x, Y: y}
	wrappedPos := pos.GetWrapped(XLargeMapTiles, YLargeMapTiles)

	return initialLargeMapSprite(indexes.SpriteIndex(m.rawData[wrappedPos.X][wrappedPos.Y]))
}

func initialLargeMapSprite(sprite indexes.SpriteIndex) indexes.SpriteIndex {
	if sprite >= indexes.Waterfall_KeyIndex &&
		sprite < indexes.Waterfall_KeyIndex+indexes.StandardNumberOfAnimationFrames {
		return indexes.Waterfall_KeyIndex
	}
	if sprite == indexes.Clock2 {
		return indexes.Clock1
	}

	return sprite
}

//nolint:nonamedreturns // The deferred overlay close preserves read failures and rejects a failed close.
func loadLargeMapFromFile(world World, gameConfig *config.UltimaVConfiguration) (result *LargeMapReference, err error) {
	mapFileAndPath, dataOvlFileAndPath := path.Join(gameConfig.SavedConfigData.DataFilePath, files.BRIT_DAT),
		path.Join(gameConfig.SavedConfigData.DataFilePath, files.DATA_OVL)
	if world == UNDERWORLD {
		mapFileAndPath = path.Join(gameConfig.SavedConfigData.DataFilePath, files.UNDER_DAT)
		dataOvlFileAndPath = ""
	}

	theChunksSerial, err := os.ReadFile(mapFileAndPath)
	if err != nil {
		return nil, fmt.Errorf("reading large map file %s: %w", mapFileAndPath, err)
	}
	// defer file.Close() // Ensure the file is closed when done

	// Create a buffer for data overlay chunks
	dataOvlChunks := make([]byte, totalChunks)

	var dataOvl *os.File

	ignoreOverlay := dataOvlFileAndPath == ""
	if !ignoreOverlay {
		// Open the overlay file for reading (case-insensitive path)
		dataOvl, err = os.Open(dataOvlFileAndPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open overlay file %s: %w", dataOvlFileAndPath, err)
		}
		defer func() {
			if closeErr := dataOvl.Close(); closeErr != nil {
				result = nil
				err = errors.Join(err, fmt.Errorf("closing overlay file %s: %w", dataOvlFileAndPath, closeErr))
			}
		}()

		// Seek to the overlay data offset
		_, err = dataOvl.Seek(int64(datOverlayBritMap), io.SeekStart)
		if err != nil {
			return nil, fmt.Errorf("seeking overlay map data in %s: %w", dataOvlFileAndPath, err)
		}
	}

	mapRef := LargeMapReference{}

	// Counter for serial chunks from brit.dat
	britDatChunkCount := 0

	// Chunk count
	chunkCount := 0

	// Process each chunk
	for chunk := range totalChunks {
		// Calculate the chunk column and row
		col := chunk % tilesPerChunkX
		row := chunk / tilesPerChunkY

		// Read overlay chunk value (or set to 0 if ignoring overlay)
		if ignoreOverlay {
			dataOvlChunks[chunkCount] = 0x00
		} else {
			if err := binary.Read(dataOvl, binary.LittleEndian, &dataOvlChunks[chunkCount]); err != nil {
				return nil, fmt.Errorf("reading overlay chunk %d from %s: %w", chunkCount, dataOvlFileAndPath, err)
			}
		}

		// Process each row (horizon) in the chunk
		for curRow := row * tilesPerChunkY; curRow < (row*tilesPerChunkY)+tilesPerChunkY; curRow++ {
			for curCol := col * tilesPerChunkX; curCol < (col*tilesPerChunkX)+tilesPerChunkX; curCol++ {
				// Check if it's a water tile (0xFF in overlay means water)
				const waterTile = 0xFF
				if dataOvlChunks[chunkCount] == waterTile {
					mapRef.rawData[curCol][curRow] = 0x01 // Water1 tile
				} else {
					// Land tile, read from brit.dat
					if britDatChunkCount >= len(theChunksSerial) {
						return nil, fmt.Errorf(
							"reading tile %d from %s: %w",
							britDatChunkCount,
							mapFileAndPath,
							io.ErrUnexpectedEOF,
						)
					}
					mapRef.rawData[curCol][curRow] = theChunksSerial[britDatChunkCount]
					britDatChunkCount++
				}
			}
		}

		// Increment the chunk count
		chunkCount++
	}

	return &mapRef, nil
}
