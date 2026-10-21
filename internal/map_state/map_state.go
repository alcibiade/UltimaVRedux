package map_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// MapState is the authoritative map side of game state: player location, loaded maps, lighting and the timed open door.
type MapState struct {
	PlayerLocation references.PlayerLocation
	LayeredMaps    LayeredMaps

	XTilesVisibleOnGameScreen int
	YTilesVisibleOnGameScreen int

	Lighting Lighting

	// open door
	openDoorPos   *references.Position
	openDoorTurns int
	openDoorFloor references.FloorNumber
	openDoorType  indexes.SpriteIndex
}

// NewMapStateInput carries the view dimensions NewMapState needs.
type NewMapStateInput struct {
	GameDimensions            GameDimensions
	XTilesVisibleOnGameScreen int
	YTilesVisibleOnGameScreen int
}

// NewMapState creates a MapState with no maps loaded.
func NewMapState(input NewMapStateInput) *MapState {
	return &MapState{
		PlayerLocation: references.PlayerLocation{
			Location: 0,
			Position: references.Position{X: 0, Y: 0},
			Floor:    0,
		},
		LayeredMaps:               LayeredMaps{layeredMaps: nil, smallMapLocation: 0, doorOccupancy: nil},
		XTilesVisibleOnGameScreen: input.XTilesVisibleOnGameScreen,
		YTilesVisibleOnGameScreen: input.YTilesVisibleOnGameScreen,
		Lighting:                  NewLighting(input.GameDimensions),
		openDoorPos:               nil,
		openDoorTurns:             0,
		openDoorFloor:             0,
		openDoorType:              indexes.NoSprites,
	}
}

// SavePlayerLocation returns a copy of the current player location (Location, Position, Floor)
func (m *MapState) SavePlayerLocation() references.PlayerLocation {
	return m.PlayerLocation // Go structs are copied by value
}

// RestorePlayerLocation sets all player location fields at once from a saved location
func (m *MapState) RestorePlayerLocation(location references.PlayerLocation) {
	m.PlayerLocation = location
}

// HasTransientDoor reports whether a timed open-door mutation is active.
func (m *MapState) HasTransientDoor() bool {
	return m.openDoorPos != nil || m.openDoorTurns != 0
}

// ResetTransientDoor clears timed door state after source terrain is reloaded.
func (m *MapState) ResetTransientDoor() {
	m.openDoorPos = nil
	m.openDoorTurns = 0
	m.openDoorFloor = 0
	m.openDoorType = indexes.NoSprites
}

// GetLayeredMapByCurrentLocation returns the layered map the player is currently on.
func (m *MapState) GetLayeredMapByCurrentLocation() *LayeredMap {
	return m.LayeredMaps.GetLayeredMap(m.PlayerLocation.Location.GetMapType(), m.PlayerLocation.Floor)
}
