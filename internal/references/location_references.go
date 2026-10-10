package references

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// LocationReferences a collection of SingleSmallMapReferences. Provides an easier way to keep
// the raw map data organized and accessible
type LocationReferences struct {
	maps           map[Location]*SmallLocationReference
	mapsByStr      map[string]*SmallLocationReference
	config         *config.UltimaVConfiguration
	dataOvl        *DataOvl
	WorldLocations *WorldLocations `json:"world_locations" yaml:"world_locations"`

	npcRefs *NPCReferences
}

// Dungeon map constants - dungeons are 8x8 tiles vs 32x32 for small maps
const (
	XDungeonMapTiles      = Coordinate(8)
	YDungeonMapTiles      = Coordinate(8)
	dungeonMapSizeInBytes = int(XDungeonMapTiles * YDungeonMapTiles)
)

// isDungeonLocation returns true if the location is a dungeon (8x8 tiles)
func isDungeonLocation(location Location) bool {
	return location >= Deceit && location <= Doom
}

func (s *LocationReferences) GetLocationReference(location Location) *SmallLocationReference {
	return s.maps[location]
}

func (s *LocationReferences) GetSmallLocationReference(name string) *SmallLocationReference {
	return s.mapsByStr[strings.ToLower(name)]
}

func newSingleMapReferences(config *config.UltimaVConfiguration, dataOvl *DataOvl) (*LocationReferences, error) {
	smr := &LocationReferences{}
	smr.config = config
	smr.dataOvl = dataOvl
	var err error
	smr.WorldLocations, err = NewWorldLocations(smr.config)
	if err != nil {
		return nil, err
	}

	smr.maps = make(map[Location]*SmallLocationReference)
	smr.mapsByStr = make(map[string]*SmallLocationReference)

	smr.npcRefs, err = NewNPCReferences(smr.config)
	if err != nil {
		return nil, err
	}

	return smr, nil
}

func (s *LocationReferences) addLocation(location Location, bHasBasement bool, nFloors, nOffset int) (int, error) {
	maps := make(map[int]*SmallLocationReference)
	// get the file
	mapFileAndPath := path.Join(s.config.SavedConfigData.DataFilePath, GetSmallMapFile(GetMapMasterFromLocation(location)))

	// a bit wasteful since it's open a few times, but... fast computers...
	theChunksSerial, err := os.ReadFile(mapFileAndPath)
	if err != nil {
		return 0, fmt.Errorf("reading location %s map file %s: %w", location, mapFileAndPath, err)
	}

	floorModifier := 0
	if bHasBasement {
		floorModifier = -1
	}

	smr := NewSingleSmallMapReference(
		location,
		s.npcRefs.GetNPCReferencesByLocation(location),
		s.dataOvl)

	for floor := range nFloors {
		actualFloor := floor + floorModifier
		smr.addBlankFloor(actualFloor)

		if isDungeonLocation(location) {
			// Dungeons are 8x8 tiles - load data and fill remaining space with default tiles
			for x := range int(XDungeonMapTiles) {
				for y := range int(YDungeonMapTiles) {
					byteIndex := nOffset + (floor * dungeonMapSizeInBytes) + x + (y * int(YDungeonMapTiles))
					if byteIndex >= len(theChunksSerial) {
						return 0, fmt.Errorf("reading location %s tile %d from %s: %w",
							location, byteIndex, mapFileAndPath, io.ErrUnexpectedEOF)
					}
					smr.rawData[floor+floorModifier][x][y] = theChunksSerial[byteIndex]
				}
			}
			// Fill remaining space in the 32x32 array with default tiles to prevent crashes
			// when accessing positions outside the 8x8 dungeon area
			for x := int(XDungeonMapTiles); x < int(XSmallMapTiles); x++ {
				for y := 0; y < int(YSmallMapTiles); y++ {
					smr.rawData[floor+floorModifier][x][y] = byte(indexes.BrickFloor) // Default dungeon floor
				}
			}
			for x := 0; x < int(XDungeonMapTiles); x++ {
				for y := int(YDungeonMapTiles); y < int(YSmallMapTiles); y++ {
					smr.rawData[floor+floorModifier][x][y] = byte(indexes.BrickFloor) // Default dungeon floor
				}
			}
		} else {
			// Small maps are 32x32 tiles - load normally
			for x := range int(XSmallMapTiles) {
				for y := range int(YSmallMapTiles) {
					byteIndex := nOffset + (floor * smallMapSizeInBytes) + x + (y * int(YSmallMapTiles))
					if byteIndex >= len(theChunksSerial) {
						return 0, fmt.Errorf("reading location %s tile %d from %s: %w",
							location, byteIndex, mapFileAndPath, io.ErrUnexpectedEOF)
					}
					smr.rawData[floor+floorModifier][x][y] = theChunksSerial[byteIndex]
				}
			}
		}

		maps[actualFloor] = smr
	}

	// returns the next offset - a handy way of keeping count
	s.maps[location] = smr
	s.mapsByStr[strings.ToLower(location.String())] = smr

	// Return correct offset based on location type
	if isDungeonLocation(location) {
		return nFloors*dungeonMapSizeInBytes + nOffset, nil
	}

	return nFloors*smallMapSizeInBytes + nOffset, nil
}
