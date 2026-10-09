package map_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestSmallMapProcessTurnDoors_NoOpenDoor(t *testing.T) {
	mapState := &MapState{}

	assert.False(t, mapState.SmallMapProcessTurnDoors(func(references.Position, references.FloorNumber) bool { return false }))
	assert.False(t, mapState.HasTransientDoor())
}

func TestSmallMapProcessTurnDoors_ClosesExpiredDoor(t *testing.T) {
	tileRefs := references.NewTileReferences()
	position := &references.Position{X: 1, Y: 1}
	layeredMap := newLayeredMap(4, 4, tileRefs, 4, 4, false)
	layeredMap.SetTileByLayer(MapLayer, position, indexes.RegularDoor)
	layeredMap.SetTileByLayer(MapOverrideLayer, position, indexes.BrickFloor)

	mapState := &MapState{
		PlayerLocation: references.PlayerLocation{Floor: 0},
		LayeredMaps: LayeredMaps{layeredMaps: map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap{
			references.SmallMapType: {0: layeredMap},
		}},
		openDoorPos:  position,
		openDoorType: indexes.RegularDoor,
	}

	assert.True(t, mapState.SmallMapProcessTurnDoors(func(references.Position, references.FloorNumber) bool { return false }))
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), layeredMap.GetTileTopMapOnlyTile(position).Index)
	assert.False(t, mapState.HasTransientDoor())
}

func TestSmallMapProcessTurnDoors_DefersExpiredOccupiedDoor(t *testing.T) {
	tileRefs := references.NewTileReferences()
	position := &references.Position{X: 1, Y: 1}
	layeredMap := newLayeredMap(4, 4, tileRefs, 4, 4, false)
	layeredMap.SetTileByLayer(MapLayer, position, indexes.RegularDoorView)
	layeredMap.SetTileByLayer(MapOverrideLayer, position, indexes.BrickFloor)
	mapState := &MapState{
		LayeredMaps: LayeredMaps{layeredMaps: map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap{
			references.SmallMapType: {1: layeredMap},
		}},
		openDoorPos:   position,
		openDoorFloor: 1,
		openDoorType:  indexes.RegularDoorView,
	}
	occupied := true
	check := func(gotPosition references.Position, gotFloor references.FloorNumber) bool {
		assert.Equal(t, *position, gotPosition)
		assert.Equal(t, references.FloorNumber(1), gotFloor)
		return occupied
	}

	assert.False(t, mapState.SmallMapProcessTurnDoors(check))
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layeredMap.GetTileTopMapOnlyTile(position).Index)
	assert.True(t, mapState.HasTransientDoor())

	occupied = false
	assert.True(t, mapState.SmallMapProcessTurnDoors(check))
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoorView), layeredMap.GetTileTopMapOnlyTile(position).Index)
	assert.False(t, mapState.HasTransientDoor())
}

func TestTrackedOpenDoorDetachedRestoreAndOrigin(t *testing.T) {
	tiles := references.NewTileReferences()
	lower, upper := newLayeredMap(4, 4, tiles, 4, 4, false), newLayeredMap(4, 4, tiles, 4, 4, false)
	state := &MapState{
		PlayerLocation: references.PlayerLocation{Location: references.Britain, Position: references.Position{X: 1, Y: 1}},
		LayeredMaps: LayeredMaps{
			smallMapLocation: references.Britain,
			layeredMaps: map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap{
				references.SmallMapType: {-1: lower, 0: upper},
			},
		},
	}
	position := references.Position{X: 2, Y: 1}
	lower.SetTileByLayer(MapLayer, &position, indexes.LockedDoorView)
	lower.SetTileByLayer(MapOverrideLayer, &position, indexes.RegularDoorView)
	upper.SetTileByLayer(MapLayer, &position, indexes.LockedDoor)
	saved := TrackedOpenDoor{
		Position:       references.PlayerLocation{Location: references.Britain, Floor: -1, Position: position},
		RemainingTurns: 0, ClosedTile: indexes.RegularDoorView,
	}
	require.NoError(t, state.RestoreOpenDoor(saved))
	assert.Equal(t, saved, *state.TrackedOpenDoor())
	detached := state.TrackedOpenDoor()
	detached.Position.Position.X++
	detached.RemainingTurns = 2
	detached.ClosedTile = indexes.RegularDoor
	assert.Equal(t, saved, *state.TrackedOpenDoor(), "callers cannot mutate runtime continuation")
	assert.False(t, state.IsPositionOpenDoor(&position), "matching coordinates upstairs have independent terrain")
	require.True(t, state.SmallMapProcessTurnDoors(func(references.Position, references.FloorNumber) bool { return false }))
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoorView), lower.GetTileTopMapOnlyTile(&position).Index)
	assert.Equal(t, indexes.SpriteIndex(indexes.LockedDoor), upper.GetTileTopMapOnlyTile(&position).Index)
	assert.Nil(t, state.TrackedOpenDoor())

	saved.ClosedTile = indexes.RegularDoor
	require.ErrorIs(t, state.RestoreOpenDoor(saved), ErrInvalidOpenDoor)
	assert.False(t, state.HasTransientDoor(), "invalid restoration is atomic")
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoorView), lower.GetTileTopMapOnlyTile(&position).Index)
}
