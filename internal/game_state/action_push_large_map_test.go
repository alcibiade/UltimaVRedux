package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestActionPushLargeMapExactSourceTileMatrix(t *testing.T) {
	objects := []indexes.SpriteIndex{
		indexes.Plant,
		indexes.ChairBackForward, indexes.ChairBackLeft, indexes.ChairBackBack, indexes.ChairBackRight,
		indexes.EndTable, indexes.Barrel, indexes.Vanity, indexes.WaterJugTable,
		indexes.Dresser, indexes.CoffeeTable, indexes.Box,
		indexes.CannonUp, indexes.CannonDown, indexes.CannonRight, indexes.CannonLeft,
	}
	require.Len(t, objects, 16)

	for _, object := range objects {
		t.Run(fmt.Sprintf("0x%02x", object), func(t *testing.T) {
			state, callbacks, positions := newLargeMapPushFixture(t, object, references.Right)
			legalFloor := indexes.SpriteIndex(indexes.BrickFloor)
			if object >= indexes.CannonUp && object <= indexes.CannonLeft {
				legalFloor = indexes.SpriteIndex(indexes.MetalFloor)
			}
			setLargeMapPushTile(state, positions.actor, legalFloor)
			setLargeMapPushTile(state, positions.forward, legalFloor)
			callbacks.Reset()

			require.True(t, state.ActionPushLargeMap(references.Right))

			assert.Equal(t, positions.target, state.MapState.PlayerLocation.Position)
			assert.Equal(t, legalFloor, largeMapPushTile(state, positions.target))
			assert.Equal(t, originalPushFacing(object, references.Right), largeMapPushTile(state, positions.forward))
			assert.Equal(t, []string{"Pushed!"}, callbacks.Messages)
			assert.Equal(t, []int{DefaultLargeMapMinutesPerTurn}, callbacks.TimeAdvanced())
		})
	}
}

func TestActionPushLargeMapRejectsExtendedReduxPushablesAndWrongFloors(t *testing.T) {
	tests := []struct {
		name         string
		object       indexes.SpriteIndex
		actorFloor   indexes.SpriteIndex
		forwardFloor indexes.SpriteIndex
		wantMessage  string
	}{
		{name: "extended table is not source movable", object: indexes.TableMiddle, actorFloor: indexes.BrickFloor, forwardFloor: indexes.BrickFloor, wantMessage: "Won't budge!"},
		{name: "ordinary object rejects metal floor", object: indexes.Barrel, actorFloor: indexes.Grass, forwardFloor: indexes.MetalFloor, wantMessage: "Won't budge"},
		{name: "cannon rejects brick floor", object: indexes.CannonUp, actorFloor: indexes.Grass, forwardFloor: indexes.BrickFloor, wantMessage: "Won't budge"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks, positions := newLargeMapPushFixture(t, test.object, references.Right)
			setLargeMapPushTile(state, positions.actor, test.actorFloor)
			setLargeMapPushTile(state, positions.forward, test.forwardFloor)
			before := state.MapState.PlayerLocation.Position
			callbacks.Reset()

			assert.False(t, state.ActionPushLargeMap(references.Right))

			assert.Equal(t, before, state.MapState.PlayerLocation.Position)
			assert.Equal(t, test.object, largeMapPushTile(state, positions.target))
			assert.Equal(t, []string{test.wantMessage}, callbacks.Messages)
			assert.Equal(t, []int{DefaultLargeMapMinutesPerTurn}, callbacks.TimeAdvanced())
		})
	}
}

func TestActionPushLargeMapOccupancyBlocksTargetAndSelectsPullFallback(t *testing.T) {
	t.Run("item stack blocks adjacent target first", func(t *testing.T) {
		state, callbacks, positions := newLargeMapPushFixture(t, indexes.Barrel, references.Right)
		require.NoError(t, state.ItemStacksMap.Push(&positions.target, &references.ItemStack{
			Items: references.ItemStacks{{Item: references.Gold, Quantity: 1}},
		}))
		callbacks.Reset()

		assert.False(t, state.ActionPushLargeMap(references.Right))
		assert.Equal(t, []string{"Won't budge!"}, callbacks.Messages)
		assert.Equal(t, indexes.SpriteIndex(indexes.Barrel), largeMapPushTile(state, positions.target))
	})

	for _, unitType := range []string{"enemy", "vehicle"} {
		t.Run(unitType+" blocks adjacent target", func(t *testing.T) {
			state, callbacks, positions := newLargeMapPushFixture(t, indexes.Barrel, references.Right)
			var unit map_units.MapUnit
			if unitType == "enemy" {
				enemy := map_units.NewEnemyNPC(*state.enemyReference(references.MonsterOrc), 1)
				enemy.SetPos(positions.target)
				enemy.SetFloor(state.MapState.PlayerLocation.Floor)
				enemy.SetVisible(true)
				unit = &enemy
			} else {
				vehicle := map_units.NewNPCFriendlyVehiceNewRef(
					references.SkiffVehicle,
					positions.target,
					state.MapState.PlayerLocation.Floor,
				)
				vehicle.SetPos(positions.target)
				unit = vehicle
			}
			state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{unit})
			callbacks.Reset()

			assert.False(t, state.ActionPushLargeMap(references.Right))
			assert.Equal(t, []string{"Won't budge!"}, callbacks.Messages)
		})
	}

	t.Run("occupied destination automatically pulls", func(t *testing.T) {
		state, callbacks, positions := newLargeMapPushFixture(t, indexes.ChairBackForward, references.Down)
		setLargeMapPushTile(state, positions.actor, indexes.BrickFloor)
		setLargeMapPushTile(state, positions.forward, indexes.BrickFloor)
		require.NoError(t, state.ItemStacksMap.Push(&positions.forward, &references.ItemStack{
			Items: references.ItemStacks{{Item: references.Gold, Quantity: 1}},
		}))
		callbacks.Reset()

		require.True(t, state.ActionPushLargeMap(references.Down))

		assert.Equal(t, positions.target, state.MapState.PlayerLocation.Position)
		assert.Equal(t, originalPushFacing(indexes.ChairBackForward, references.Up), largeMapPushTile(state, positions.actor))
		assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), largeMapPushTile(state, positions.target))
		assert.Equal(t, []string{"Pulled!"}, callbacks.Messages)
	})
}

func TestActionPushLargeMapRotatesChairsAndCannonsByOriginalNumericDirection(t *testing.T) {
	directions := []struct {
		direction references.Direction
		offset    indexes.SpriteIndex
	}{
		{direction: references.Up, offset: 0},
		{direction: references.Right, offset: 1},
		{direction: references.Down, offset: 2},
		{direction: references.Left, offset: 3},
	}

	for _, base := range []indexes.SpriteIndex{indexes.ChairBackForward, indexes.CannonUp} {
		for _, test := range directions {
			t.Run(fmt.Sprintf("0x%02x_%s", base, test.direction.GetDirectionCompassName()), func(t *testing.T) {
				assert.Equal(t, base+test.offset, originalPushFacing(base+3, test.direction))
				assert.Equal(t, base+(test.offset^2), originalPushFacing(base, test.direction.GetOppositeDirection()))
			})
		}
	}
	assert.Equal(t, indexes.SpriteIndex(0xb5), originalPushFacing(indexes.CannonLeft, references.Right))
	assert.Equal(t, indexes.SpriteIndex(0xb6), originalPushFacing(indexes.CannonUp, references.Down))
}

func TestActionPushLargeMapWrapsActorWithoutMovementSideEffects(t *testing.T) {
	state, callbacks, positions := newLargeMapPushFixtureAt(
		t,
		indexes.Plant,
		references.Right,
		references.Position{X: references.XLargeMapTiles - 1, Y: 40},
	)
	setLargeMapPushTile(state, positions.actor, indexes.BrickFloor)
	setLargeMapPushTile(state, positions.forward, indexes.BrickFloor)
	callbacks.Reset()

	require.True(t, state.ActionPushLargeMap(references.Right))

	assert.Equal(t, references.Position{X: 0, Y: 40}, state.MapState.PlayerLocation.Position)
	assert.Equal(t, indexes.SpriteIndex(indexes.Plant), largeMapPushTile(state, references.Position{X: 1, Y: 40}))
	assert.Equal(t, []int{DefaultLargeMapMinutesPerTurn}, callbacks.TimeAdvanced())
}

func TestActionPushLargeMapTransientTilesDoNotPersistButActorAndUnrelatedMutationDo(t *testing.T) {
	state, _, positions := newLargeMapPushFixture(t, indexes.Barrel, references.Right)
	setLargeMapPushTile(state, positions.actor, indexes.BrickFloor)
	setLargeMapPushTile(state, positions.forward, indexes.BrickFloor)
	unrelated := references.Position{X: 80, Y: 81}
	setLargeMapPushTile(state, unrelated, indexes.RegularDoor)
	require.True(t, state.ActionPushLargeMap(references.Right))

	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	assert.Equal(t, int(indexes.Barrel), mapMutationValueAt(t, snapshot.MapMutations, positions.target))
	assert.Equal(t, int(indexes.BrickFloor), mapMutationValueAt(t, snapshot.MapMutations, positions.forward))
	assert.Equal(t, int(indexes.RegularDoor), mapMutationValueAt(t, snapshot.MapMutations, unrelated))

	restored, err := NewGameStateFromSaveSnapshot(
		snapshot,
		state.GameConfig,
		state.GameReferences,
		xTilesVisibleOnGameScreen,
		yTilesVisibleOnGameScreen,
	)
	require.NoError(t, err)
	assert.Equal(t, positions.target, restored.MapState.PlayerLocation.Position)
	assert.Equal(t, indexes.SpriteIndex(indexes.Barrel), largeMapPushTile(restored, positions.target))
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), largeMapPushTile(restored, positions.forward))
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), largeMapPushTile(restored, unrelated))

	state.MapState.LayeredMaps.SetInitialLargeMap(
		state.MapState.PlayerLocation.Floor,
		state.GameReferences.OverworldLargeMapReference,
	)
	assert.Equal(t, state.baseTile(references.BritanniaUnderworld, state.MapState.PlayerLocation.Floor, positions.target), largeMapPushTile(state, positions.target))
	assert.Equal(t, state.baseTile(references.BritanniaUnderworld, state.MapState.PlayerLocation.Floor, positions.forward), largeMapPushTile(state, positions.forward))
	reloadedSnapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	assert.False(t, hasMapTileMutationAt(reloadedSnapshot.MapMutations, positions.target))
	assert.False(t, hasMapTileMutationAt(reloadedSnapshot.MapMutations, positions.forward))
}

type largeMapPushPositions struct {
	actor   references.Position
	target  references.Position
	forward references.Position
}

func newLargeMapPushFixture(
	t *testing.T,
	object indexes.SpriteIndex,
	direction references.Direction,
) (*GameState, *MockSystemCallbacks, largeMapPushPositions) {
	t.Helper()

	return newLargeMapPushFixtureAt(t, object, direction, references.Position{X: 50, Y: 50})
}

func newLargeMapPushFixtureAt(
	t *testing.T,
	object indexes.SpriteIndex,
	direction references.Direction,
	actor references.Position,
) (*GameState, *MockSystemCallbacks, largeMapPushPositions) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(int(actor.X), int(actor.Y)).
		WithSystemCallbacks().
		WithRandomSeed(390).
		Build()
	state.DebugOptions.MonsterGen = false
	state.CurrentNPCAIController.GetNpcs().ClearAllMapUnits()
	state.ItemStacksMap = *references.NewItemStacksMap()
	target, ok := largeMapPushPosition(actor, direction)
	require.True(t, ok)
	forward, ok := largeMapPushPosition(target, direction)
	require.True(t, ok)
	positions := largeMapPushPositions{actor: actor, target: target, forward: forward}
	setLargeMapPushTile(state, target, object)
	setLargeMapPushTile(state, actor, indexes.BrickFloor)
	setLargeMapPushTile(state, forward, indexes.BrickFloor)

	return state, callbacks, positions
}

func setLargeMapPushTile(state *GameState, position references.Position, tile indexes.SpriteIndex) {
	state.GetCurrentLayeredMap().UnSetTileByLayer(map_state.MapOverrideLayer, &position)
	state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, tile)
}

func largeMapPushTile(state *GameState, position references.Position) indexes.SpriteIndex {
	return state.GetCurrentLayeredMap().SpriteIndexByLayer(map_state.MapLayer, position)
}

func mapMutationValueAt(
	t *testing.T,
	mutations []save_snapshot.MapMutationSnapshot,
	position references.Position,
) int {
	t.Helper()
	for _, mutation := range mutations {
		if mutation.Kind == save_snapshot.MapMutationSetTile &&
			mutation.Position.Floor == int(references.OVERWORLD) &&
			mutation.Position.X == int(position.X) && mutation.Position.Y == int(position.Y) {
			return mutation.Value
		}
	}
	t.Fatalf("no tile mutation at (%d,%d)", position.X, position.Y)

	return 0
}

func hasMapTileMutationAt(
	mutations []save_snapshot.MapMutationSnapshot,
	position references.Position,
) bool {
	for _, mutation := range mutations {
		if mutation.Kind == save_snapshot.MapMutationSetTile &&
			mutation.Position.Floor == int(references.OVERWORLD) &&
			mutation.Position.X == int(position.X) && mutation.Position.Y == int(position.Y) {
			return true
		}

	}

	return false
}
