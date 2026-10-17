package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/require"
)

func TestStonegateFloodRetiresDoorOverridesAndTimerOnlyOnFloodedFloor(t *testing.T) {
	for _, closeFirst := range []bool{false, true} {
		state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Stonegate).
			WithPlayerAt(15, 16).WithSystemCallbacks().Build()
		layer := state.GetCurrentLayeredMap()
		door := state.MapState.PlayerLocation.Position.Left()
		layer.SetTileByLayer(map_state.MapLayer, &door, indexes.RegularDoorView)
		require.Equal(t, map_state.OpenDoorOpened, state.MapState.OpenDoor(references.Left))
		if closeFirst {
			state.MapState.CloseOpenDoor()
		}
		unrelated := references.Position{X: 2, Y: 2}
		layer.SetTileByLayer(map_state.MapLayer, &unrelated, indexes.Grass)
		layer.SetTileByLayer(map_state.MapOverrideLayer, &unrelated, indexes.BrickFloor)
		var changedDoors []references.Position
		for index, variant := range []indexes.SpriteIndex{indexes.RegularDoor, indexes.RegularDoorView,
			indexes.LockedDoor, indexes.LockedDoorView, indexes.MagicLockDoor, indexes.MagicLockDoorWithView} {
			position := references.Position{X: references.Coordinate(index + 3), Y: 3}
			layer.SetTileByLayer(map_state.MapLayer, &position, variant)
			if index > 0 {
				layer.SetTileByLayer(map_state.MapOverrideLayer, &position, indexes.BrickFloor)
			}
			changedDoors = append(changedDoors, position)
		}
		createdDoor := references.Position{X: 9, Y: 3}
		layer.SetTileByLayer(map_state.MapLayer, &createdDoor, indexes.Grass)
		layer.SetTileByLayer(map_state.MapOverrideLayer, &createdDoor, indexes.RegularDoorView)
		changedDoors = append(changedDoors, createdDoor)
		review, err := state.ReviewStonegateTrap()
		require.NoError(t, err)
		_, err = state.CommitStonegateTrap(review.Fingerprint)
		require.NoError(t, err)
		require.Equal(t, indexes.SpriteIndex(indexes.Lava), layer.GetTileTopMapOnlyTile(&door).Index)
		require.Nil(t, state.MapState.TrackedOpenDoor())
		for range 5 {
			state.MapState.SmallMapProcessTurnDoors(func(references.Position, references.FloorNumber) bool { return false })
		}
		require.Equal(t, indexes.SpriteIndex(indexes.Lava), layer.GetTileTopMapOnlyTile(&door).Index)
		require.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layer.GetTileTopMapOnlyTile(&unrelated).Index)
		for _, position := range changedDoors {
			require.Equal(t, indexes.SpriteIndex(indexes.Lava), layer.GetTileTopMapOnlyTile(&position).Index)
		}
	}
}

func TestFloodPreservesTrackedDoorOnAnotherFloor(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Lycaeum).
		WithFloor(2).WithPlayerAt(15, 9).WithSystemCallbacks().Build()
	otherFloor := state.GetCurrentLayeredMap()
	door := state.MapState.PlayerLocation.Position.Left()
	otherFloor.SetTileByLayer(map_state.MapLayer, &door, indexes.RegularDoor)
	require.Equal(t, map_state.OpenDoorOpened, state.MapState.OpenDoor(references.Left))
	tracked := state.MapState.TrackedOpenDoor()
	state.MapState.PlayerLocation.Floor = 0
	state.FloodFloorWithLava()
	require.Equal(t, tracked, state.MapState.TrackedOpenDoor())
	require.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), otherFloor.GetTileTopMapOnlyTile(&door).Index)
	state.MapState.CloseOpenDoor()
	require.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), otherFloor.GetTileTopMapOnlyTile(&door).Index)
}
