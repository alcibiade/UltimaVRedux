package references

import (
	"fmt"
	"io"

	"github.com/bradhannah/UltimaVRedux/internal/config"
)

const (
	xCoordsOffset  = 0x1e9a
	yCoordsOffset  = 0x1ec2
	TotalLocations = 40
)

type WorldLocations struct {
	LargeMapLocationPositions map[Location]WorldLocation `json:"large_map_location_positions" yaml:"large_map_location_positions"`
	// shrinePositions
}

type WorldLocation struct {
	Position Position    `json:"position" yaml:"position"`
	Location Location    `json:"location" yaml:"location"`
	Floor    FloorNumber `json:"floor"    yaml:"floor"`
}

// NewWorldLocations loads location coordinates from the configured DATA.OVL bytes.
func NewWorldLocations(gameConfig *config.UltimaVConfiguration) (*WorldLocations, error) {
	requiredLength := yCoordsOffset + TotalLocations
	if len(gameConfig.RawDataOvl) < requiredLength {
		return nil, fmt.Errorf("reading world locations: expected at least %d DATA.OVL bytes, got %d: %w",
			requiredLength, len(gameConfig.RawDataOvl), io.ErrUnexpectedEOF)
	}

	wl := &WorldLocations{}
	wl.LargeMapLocationPositions = make(map[Location]WorldLocation)

	for i := 0; i < TotalLocations; i++ {
		x := Coordinate(gameConfig.RawDataOvl[xCoordsOffset+i])
		y := Coordinate(gameConfig.RawDataOvl[yCoordsOffset+i])
		location := Location(i + 1)

		// Determine floor: Ararat is the only location in the underworld (floor -1)
		// All other towns, castles, keeps, and dungeons exit to overworld (floor 0)
		floor := FloorNumber(OVERWORLD)
		if location == Ararat {
			floor = FloorNumber(UNDERWORLD)
		}

		wl.LargeMapLocationPositions[location] = WorldLocation{
			Location: location,
			Position: Position{
				X: x,
				Y: y,
			},
			Floor: floor,
		}
	}

	return wl, nil
}

func (wl *WorldLocations) GetLocationByPosition(position Position) Location {
	for _, worldLocation := range wl.LargeMapLocationPositions {
		if worldLocation.Position.Equals(&position) {
			return worldLocation.Location
		}
	}

	return EmptyLocation
}

// GetWorldLocation returns the world map position and floor for a location.
// Used when exiting towns/dungeons to determine where player appears on large map.
// Panics if location not found (programming error - all valid locations 1-40 should exist).
func (wl *WorldLocations) GetWorldLocation(location Location) (Position, FloorNumber) {
	worldLoc, exists := wl.LargeMapLocationPositions[location]
	if !exists {
		panic("GetWorldLocation: location not found in world locations map - programming error")
	}

	return worldLoc.Position, worldLoc.Floor
}
