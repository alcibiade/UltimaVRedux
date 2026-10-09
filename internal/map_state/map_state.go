package map_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

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

type NewMapStateInput struct {
	GameDimensions            GameDimensions
	XTilesVisibleOnGameScreen int
	YTilesVisibleOnGameScreen int
}

func NewMapState(input NewMapStateInput) *MapState {
	mapState := &MapState{}
	mapState.Lighting = NewLighting(input.GameDimensions)
	mapState.XTilesVisibleOnGameScreen = input.XTilesVisibleOnGameScreen
	mapState.YTilesVisibleOnGameScreen = input.YTilesVisibleOnGameScreen

	return mapState
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

func (m *MapState) GetLayeredMapByCurrentLocation() *LayeredMap {
	return m.LayeredMaps.GetLayeredMap(m.PlayerLocation.Location.GetMapType(), m.PlayerLocation.Floor)
}
