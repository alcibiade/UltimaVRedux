package map_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// DoorOpenState is the outcome of an attempt to open a door.
type DoorOpenState int

const (
	// OpenDoorLocked means the door is locked and needs a key.
	OpenDoorLocked DoorOpenState = iota
	// OpenDoorLockedMagical means the door is held shut by a magic lock.
	OpenDoorLockedMagical
	// OpenDoorNotADoor means the target tile is not a door.
	OpenDoorNotADoor
	// OpenDoorOpened means the door was opened.
	OpenDoorOpened
)

// OpenDoor opens the door next to the player in direction and starts its auto-close timer.
func (m *MapState) OpenDoor(direction references.Direction) DoorOpenState {
	newPosition := direction.GetNewPositionInDirection(&m.PlayerLocation.Position)
	theMap := m.LayeredMaps.GetLayeredMap(references.SmallMapType, m.PlayerLocation.Floor)
	closedTile := theMap.GetTileTopMapOnlyTile(newPosition)
	result := OpenDoorAt(theMap, newPosition)
	if result != OpenDoorOpened {
		return result
	}

	m.CloseOpenDoor()
	m.openDoorPos = newPosition
	m.openDoorTurns = DefaultOpenDoorTurns
	m.openDoorFloor = m.PlayerLocation.Floor
	m.openDoorType = closedTile.Index

	return OpenDoorOpened
}

// OpenDoorAt opens one surface-style door without applying map-specific lifecycle rules.
func OpenDoorAt(theMap *LayeredMap, position *references.Position) DoorOpenState {
	targetTile := theMap.GetTileTopMapOnlyTile(position)

	doorIndex := targetTile.Index
	if doorIndex == indexes.MagicLockDoor || doorIndex == indexes.MagicLockDoorWithView {
		return OpenDoorLockedMagical
	}
	if doorIndex == indexes.LockedDoor || doorIndex == indexes.LockedDoorView {
		return OpenDoorLocked
	}
	if doorIndex != indexes.RegularDoor && doorIndex != indexes.RegularDoorView {
		return OpenDoorNotADoor
	}

	theMap.SetTileByLayer(MapOverrideLayer, position, indexes.BrickFloor)

	return OpenDoorOpened
}

// IsPositionOpenDoor returns true if the position currently has an open door
func (m *MapState) IsPositionOpenDoor(pos *references.Position) bool {
	if m.openDoorPos != nil && m.openDoorFloor == m.PlayerLocation.Floor && m.openDoorPos.Equals(pos) {
		return true
	}
	if m.PlayerLocation.Location.GetMapType() != references.SmallMapType {
		return false
	}
	layer := m.GetLayeredMapByCurrentLocation()
	terrain := layer.GetTileTopMapOnlyTile(pos)

	return layer.isOccupiedDoor(pos, terrain)
}
