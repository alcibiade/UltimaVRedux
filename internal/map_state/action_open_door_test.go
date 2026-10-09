package map_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestOpenDoorAt(t *testing.T) {
	tileRefs := references.NewTileReferences()
	position := &references.Position{X: 1, Y: 1}

	tests := []struct {
		name        string
		tile        indexes.SpriteIndex
		wantState   DoorOpenState
		wantTopTile indexes.SpriteIndex
	}{
		{name: "regular door opens", tile: indexes.RegularDoor, wantState: OpenDoorOpened, wantTopTile: indexes.BrickFloor},
		{name: "windowed door opens", tile: indexes.RegularDoorView, wantState: OpenDoorOpened, wantTopTile: indexes.BrickFloor},
		{name: "locked door stays closed", tile: indexes.LockedDoor, wantState: OpenDoorLocked, wantTopTile: indexes.LockedDoor},
		{name: "magically locked door stays closed", tile: indexes.MagicLockDoor, wantState: OpenDoorLockedMagical, wantTopTile: indexes.MagicLockDoor},
		{name: "non-door is rejected", tile: indexes.StoneBrickWall, wantState: OpenDoorNotADoor, wantTopTile: indexes.StoneBrickWall},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			layeredMap := newLayeredMap(4, 4, tileRefs, 4, 4, false)
			layeredMap.SetTileByLayer(MapLayer, position, test.tile)

			assert.Equal(t, test.wantState, OpenDoorAt(layeredMap, position))
			assert.Equal(t, test.wantTopTile, layeredMap.GetTileTopMapOnlyTile(position).Index)
		})
	}
}

func TestOpenDoorVisitExactClosureAndSourceFloor(t *testing.T) {
	for _, closed := range []indexes.SpriteIndex{indexes.RegularDoor, indexes.RegularDoorView} {
		for _, source := range []indexes.SpriteIndex{indexes.LockedDoor, indexes.LockedDoorView, indexes.StoneBrickWallSecret} {
			tiles := references.NewTileReferences()
			ground, upper := newLayeredMap(4, 4, tiles, 4, 4, false), newLayeredMap(4, 4, tiles, 4, 4, false)
			state := &MapState{
				PlayerLocation: references.PlayerLocation{Location: references.Britain, Floor: -1, Position: references.Position{X: 1, Y: 1}},
				LayeredMaps: LayeredMaps{layeredMaps: map[references.GeneralMapType]map[references.FloorNumber]*LayeredMap{
					references.SmallMapType: {-1: ground, 0: upper},
				}},
			}
			a, b := references.Position{X: 1, Y: 0}, references.Position{X: 2, Y: 1}
			ground.SetTileByLayer(MapLayer, &a, source)
			ground.SetTileByLayer(MapOverrideLayer, &a, closed)
			ground.SetTileByLayer(MapLayer, &b, indexes.RegularDoorView)
			upper.SetTileByLayer(MapLayer, &b, indexes.LockedDoor)
			require.Equal(t, OpenDoorOpened, state.OpenDoor(references.Up))
			require.True(t, state.IsPositionOpenDoor(&a))
			assert.Equal(t, OpenDoorNotADoor, state.OpenDoor(references.Up), "failed Open must not close/restart A")
			assert.Equal(t, 2, state.openDoorTurns)
			require.Equal(t, OpenDoorOpened, state.OpenDoor(references.Right))
			assert.Equal(t, closed, ground.GetTileTopMapOnlyTile(&a).Index, "replacement never exposes the original lock/wall")
			state.PlayerLocation.Floor = 0
			assert.False(t, state.IsPositionOpenDoor(&b), "same coordinates upstairs are not open")
			notOccupied := func(references.Position, references.FloorNumber) bool { return false }
			assert.False(t, state.SmallMapProcessTurnDoors(notOccupied))
			assert.False(t, state.SmallMapProcessTurnDoors(notOccupied))
			assert.True(t, state.SmallMapProcessTurnDoors(notOccupied))
			assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoorView), ground.GetTileTopMapOnlyTile(&b).Index)
			assert.Equal(t, indexes.SpriteIndex(indexes.LockedDoor), upper.GetTileTopMapOnlyTile(&b).Index)
			assert.False(t, state.HasTransientDoor())
			assert.Nil(t, state.openDoorPos)
			assert.Zero(t, state.openDoorTurns)
			assert.Zero(t, state.openDoorFloor)
			assert.Equal(t, indexes.SpriteIndex(indexes.NoSprites), state.openDoorType)
		}
	}
}
