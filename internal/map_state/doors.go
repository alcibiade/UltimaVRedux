package map_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// ErrInvalidOpenDoor identifies incompatible tracked-door restoration data.
var ErrInvalidOpenDoor = errors.New("invalid open-door continuation")

// DefaultOpenDoorTurns is the initial player-open countdown (2, 1, 0, close).
const DefaultOpenDoorTurns = 2

// TrackedOpenDoor is detached continuation state for the single player-open
// timer. Occupancy-only openings have no timer or persistent terrain mutation.
type TrackedOpenDoor struct {
	Position       references.PlayerLocation
	RemainingTurns int
	ClosedTile     indexes.SpriteIndex
}

// TrackedOpenDoor returns a copy, including the originating floor.
func (m *MapState) TrackedOpenDoor() *TrackedOpenDoor {
	if m.openDoorPos == nil {
		return nil
	}

	return &TrackedOpenDoor{
		Position: references.PlayerLocation{
			Location: m.LayeredMaps.smallMapLocation,
			Floor:    m.openDoorFloor,
			Position: *m.openDoorPos,
		},
		RemainingTurns: m.openDoorTurns,
		ClosedTile:     m.openDoorType,
	}
}

// RestoreOpenDoor installs a validated timer after persistent terrain restoration,
// without commands, turns, callbacks or randomness. Zero need not be occupied:
// the final actor can leave after the door phase of the last completed turn.
func (m *MapState) RestoreOpenDoor(door TrackedOpenDoor) error {
	if door.Position.Location.GetMapType() != references.SmallMapType ||
		door.Position.Location != m.LayeredMaps.smallMapLocation ||
		door.Position.Location != m.PlayerLocation.Location ||
		door.RemainingTurns < 0 || door.RemainingTurns > DefaultOpenDoorTurns || !door.ClosedTile.IsUnlockedDoor() {
		return fmt.Errorf("%w: %+v", ErrInvalidOpenDoor, door)
	}
	layer := m.LayeredMaps.GetLayeredMap(references.SmallMapType, door.Position.Floor)
	pos := door.Position.Position
	if layer == nil || pos.X < 0 || pos.Y < 0 || pos.X >= layer.XMaxTilesPerMap || pos.Y >= layer.YMaxTilesPerMap {
		return fmt.Errorf("%w: unavailable position %+v", ErrInvalidOpenDoor, door.Position)
	}
	terrain := layer.GetTileTopMapOnlyTile(&pos)
	if terrain == nil || terrain.Index != door.ClosedTile {
		return fmt.Errorf("%w: persistent terrain does not match closed tile %d", ErrInvalidOpenDoor, door.ClosedTile)
	}
	if m.HasTransientDoor() {
		return fmt.Errorf("%w: another timer is already active", ErrInvalidOpenDoor)
	}
	m.openDoorPos, m.openDoorFloor = &pos, door.Position.Floor
	m.openDoorType, m.openDoorTurns = door.ClosedTile, door.RemainingTurns
	layer.SetTileByLayer(MapOverrideLayer, &pos, indexes.BrickFloor)

	return nil
}

// CloseOpenDoor restores the exact unlocked variant on its originating floor.
// Terrain reconstruction and opening another door also normalize through here.
func (m *MapState) CloseOpenDoor() {
	if m.openDoorPos != nil {
		layeredMap := m.LayeredMaps.GetLayeredMap(references.SmallMapType, m.openDoorFloor)
		layeredMap.SetTileByLayer(MapOverrideLayer, m.openDoorPos, m.openDoorType)
	}
	m.ResetTransientDoor()
}

// SmallMapProcessTurnDoors advances the tracked door timer and closes an unoccupied expired door.
func (m *MapState) SmallMapProcessTurnDoors(isOccupied func(references.Position, references.FloorNumber) bool) bool {
	if isOccupied == nil {
		panic("small-map door occupancy check is required")
	}
	if m.openDoorPos != nil {
		if m.openDoorTurns == 0 {
			// REDUX DEVIATION: Original timed closure (ref:CMD-0006) closes a door without
			// checking Avatar or NPC occupancy. Redux retries after actors vacate
			// so authoritative terrain and collision cannot close beneath them.
			if isOccupied(*m.openDoorPos, m.openDoorFloor) {
				return false
			}
			m.CloseOpenDoor()

			return true // Door was closed
		}
		m.openDoorTurns--
	}

	return false // No door was closed
}
