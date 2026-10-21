package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type emptyDoorTestUnit struct{ SimpleTestMapUnit }

func (*emptyDoorTestUnit) IsEmptyMapUnit() bool { return true }

func TestOccupiedDoorsUseAuthoritativeFinalActors(t *testing.T) {
	for _, closed := range []indexes.SpriteIndex{
		indexes.RegularDoor, indexes.RegularDoorView, indexes.LockedDoor,
		indexes.LockedDoorView, indexes.MagicLockDoor, indexes.MagicLockDoorWithView,
	} {
		t.Run(fmt.Sprintf("tile%d", closed), func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).
				WithPlayerAt(10, 10).WithSystemCallbacks().Build()
			door := references.Position{X: 11, Y: 10}
			layer := state.GetCurrentLayeredMap()
			layer.SetTileByLayer(map_state.MapLayer, &door, closed)
			npc := &SimpleTestMapUnit{position: door, floor: 0, visible: true}
			controller := &SimpleTestController{mapUnits: map_units.MapUnits{npc}}
			state.CurrentNPCAIController = controller
			date, calls := state.DateTime, state.GetRNG().Calls()
			assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetEffectiveTile(&door))
			assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), layer.GetLiveMapOnlyTile(&door).Index)
			assert.True(t, state.MapState.IsPositionOpenDoor(&door))
			assert.False(t, state.IsPassable(&door), "NPC blocking is independent of terrain")
			assert.Equal(t, closed, layer.GetTileTopMapOnlyTile(&door).Index)
			assert.False(t, state.MapState.HasTransientDoor())
			assert.Equal(t, date, state.DateTime)
			assert.Equal(t, calls, state.GetRNG().Calls())

			for _, excluded := range []string{"hidden", "other floor", "empty", "removed"} {
				t.Run(excluded, func(t *testing.T) {
					npc.visible, npc.floor = true, 0
					controller.mapUnits = map_units.MapUnits{npc}
					// Deliberately leave a stale sprite: removal paths need not refresh it.
					layer.SetTileByLayer(map_state.MapUnitLayer, &door, indexes.TownsPersonKeyIndex)
					switch excluded {
					case "hidden":
						npc.visible = false
					case "other floor":
						npc.floor = 1
					case "empty":
						controller.mapUnits = map_units.MapUnits{&emptyDoorTestUnit{*npc}}
					case "removed":
						controller.mapUnits = nil
					}
					assert.Equal(t, closed, state.GetEffectiveTile(&door))
					assert.False(t, state.MapState.IsPositionOpenDoor(&door))
				})
			}
			controller.mapUnits = nil
			state.SetPlayerPosition(door)
			assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetEffectiveTile(&door), "Avatar holds the doorway")
			state.SetPlayerPosition(door.Left())
			assert.Equal(t, closed, state.GetEffectiveTile(&door))
		})
	}
}

func TestOccupiedDoorsMultipleHoldsSurviveTimerReplacementAndRebuild(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(10, 14).WithSystemCallbacks().Build()
	first, second := references.Position{X: 11, Y: 14}, references.Position{X: 10, Y: 13}
	layer := state.GetCurrentLayeredMap()
	layer.SetTileByLayer(map_state.MapOverrideLayer, &second, indexes.RegularDoorView)
	one := &SimpleTestMapUnit{position: first, floor: 0, visible: true}
	two := &SimpleTestMapUnit{position: second, floor: 0, visible: true}
	state.CurrentNPCAIController = &SimpleTestController{mapUnits: map_units.MapUnits{one, two}}
	require.Equal(t, map_state.OpenDoorOpened, state.MapState.OpenDoor(references.Right))
	require.Equal(t, map_state.OpenDoorOpened, state.MapState.OpenDoor(references.Up))
	assert.True(t, state.MapState.IsPositionOpenDoor(&first), "replacing the timer cannot defeat another actor's hold")
	assert.True(t, state.MapState.IsPositionOpenDoor(&second))
	state.MapState.CloseOpenDoor()
	require.Equal(t, MagicLockSuccess, state.MagicLockDoor(&first), "lock rules read persistent doors")
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetEffectiveTile(&first))
	state.reloadCurrentSmallMapTerrain()
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetEffectiveTile(&first))
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetEffectiveTile(&second))
	one.SetPos(second)
	assert.Equal(t, indexes.SpriteIndex(indexes.MagicLockDoor), state.GetEffectiveTile(&first), "latest lock is restored immediately")
	two.SetVisible(false)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetEffectiveTile(&second), "final visible occupant still holds")
	one.SetVisible(false)
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoorView), state.GetEffectiveTile(&second))
	state.TransitionToLocation(references.BritanniaUnderworld, 0, references.Position{X: 100, Y: 100}, TransitionFullLocation, false)
	state.TransitionToLocation(references.SerpentsHold, 0, references.Position{X: 10, Y: 14}, TransitionFullLocation, false)
	assert.False(t, state.MapState.IsPositionOpenDoor(&first), "departed actors cannot hold doors in a new visit")
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), state.GetEffectiveTile(&first))
}

func TestOccupiedDoorsLiveLOSAndDaylight(t *testing.T) {
	state, _ := newDaylightTintingTestState(t)
	inside, door := references.Position{X: 10, Y: 14}, references.Position{X: 11, Y: 14}
	layer := state.GetCurrentLayeredMap()
	npc := &SimpleTestMapUnit{position: door, floor: 0, visible: true}
	state.CurrentNPCAIController = &SimpleTestController{mapUnits: map_units.MapUnits{npc}}
	defaults := map_state.DefaultDaylightOptions()
	for _, present := range []bool{true, false, true} {
		npc.SetVisible(present)
		state.UpdateLightSourceTinting(layer)
		want := defaults.IndoorBrightness
		if present {
			want = defaults.OutdoorBrightness
		}
		assertGameDaylightTarget(t, state, inside, want, true)
		assert.Equal(t, present, state.MapState.IsPositionOpenDoor(&door))
	}
	// A sealed corridor makes the far cell reachable only through this doorway.
	for y := references.Coordinate(0); y < references.YSmallMapTiles; y++ {
		for x := references.Coordinate(0); x < references.XSmallMapTiles; x++ {
			position := references.Position{X: x, Y: y}
			layer.SetTileByLayer(map_state.MapLayer, &position, indexes.StoneBrickWall)
		}
	}
	for x := references.Coordinate(9); x <= 14; x++ {
		position := references.Position{X: x, Y: 14}
		layer.SetTileByLayer(map_state.MapLayer, &position, indexes.BrickFloor)
	}
	layer.SetTileByLayer(map_state.MapLayer, &door, indexes.RegularDoor)
	far := door.Right().Right()
	for _, present := range []bool{false, true, false} {
		npc.SetVisible(present)
		layer.RecalculateVisibleTilesWithMoongates(inside, &state.MapState.Lighting, nil, map_state.PlayerLightInLor)
		assert.Equal(t, present, layer.IsPositionVisible(&far, state.DateTime, false), "LOS must follow live occupancy")
	}
}

func TestOccupiedDoorOpenRetainsLockRules(t *testing.T) {
	for _, closed := range []indexes.SpriteIndex{indexes.LockedDoor, indexes.LockedDoorView, indexes.MagicLockDoor, indexes.MagicLockDoorWithView} {
		t.Run(fmt.Sprintf("tile%d", closed), func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).WithLocation(references.Britain).
				WithPlayerAt(10, 10).WithSystemCallbacks().Build()
			door := references.Position{X: 11, Y: 10}
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &door, closed)
			npc := &SimpleTestMapUnit{position: door, floor: 0, visible: true}
			state.CurrentNPCAIController = &SimpleTestController{mapUnits: map_units.MapUnits{npc}}
			date, calls, keys := state.DateTime, state.GetRNG().Calls(), state.PartyState.Inventory.Provisions.Keys.Get()
			require.False(t, state.ActionOpenSmallMap(references.Right))
			assert.False(t, state.MapState.HasTransientDoor())
			assert.Equal(t, date, state.DateTime)
			assert.Equal(t, calls, state.GetRNG().Calls())
			assert.Equal(t, keys, state.PartyState.Inventory.Provisions.Keys.Get())
			assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetEffectiveTile(&door))
			npc.SetVisible(false)
			assert.Equal(t, closed, state.GetEffectiveTile(&door))
		})
	}
}

func TestOccupiedDoorsRebindOnBothFloorRoutes(t *testing.T) {
	for _, direct := range []bool{true, false} {
		t.Run(fmt.Sprintf("direct%t", direct), func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
				WithPlayerAt(10, 14).WithSystemCallbacks().Build()
			door := references.Position{X: 11, Y: 14}
			ref := references.NPCReference{Schedule: references.CreateNPCScheduledFixedOneLocation(door, 0)}
			ref.SetKeyIndex(indexes.TownsPersonKeyIndex)
			npc := map_units.NewNPCFriendly(ref, 1)
			npc.PlaceAtScheduleSlot(0)
			state.CurrentNPCAIController.GetNpcs().ReplaceWith(map_units.MapUnits{npc})
			state.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 1).
				SetTileByLayer(map_state.MapOverrideLayer, &door, indexes.MagicLockDoorWithView)
			require.Equal(t, map_state.OpenDoorOpened, state.MapState.OpenDoor(references.Right))
			changeSmallMapTestFloor(t, state, 1, direct)
			assert.False(t, state.MapState.HasTransientDoor(), "floor travel normalizes only the player timer")
			assert.Equal(t, indexes.SpriteIndex(indexes.MagicLockDoorWithView), state.GetEffectiveTile(&door))
			assert.False(t, state.MapState.IsPositionOpenDoor(&door), "ground-floor actor cannot hold the upper door")
			changeSmallMapTestFloor(t, state, 0, direct)
			assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetEffectiveTile(&door), "newly rebuilt floor observes retained schedule placement")
			state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
			assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), state.GetEffectiveTile(&door), "removal without layer refresh releases the final hold")
		})
	}
}

func TestOccupiedDoorDoesNotGrantFurniturePullPermission(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithLocation(references.SerpentsHold).
		WithPlayerAt(11, 14).WithSystemCallbacks().Build()
	state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
	door := state.MapState.PlayerLocation.Position
	chair, wall := door.Right(), door.Right().Right()
	layer := state.GetCurrentLayeredMap()
	require.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), layer.GetTileTopMapOnlyTile(&door).Index)
	require.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), state.GetEffectiveTile(&door))
	layer.SetTileByLayer(map_state.MapLayer, &chair, indexes.ChairBackForward)
	layer.SetTileByLayer(map_state.MapLayer, &wall, indexes.StoneBrickWall)
	date, calls := state.DateTime, state.GetRNG().Calls()
	assert.False(t, state.ActionPushSmallMap(references.Right), "an occupancy-only floor cannot authorize pulling furniture onto a stored door")
	assert.Equal(t, door, state.MapState.PlayerLocation.Position)
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), layer.GetTileTopMapOnlyTile(&door).Index,
		"pulling must not relocate the closed door identity through SwapTiles")
	assert.Equal(t, indexes.SpriteIndex(indexes.ChairBackForward), layer.GetTileTopMapOnlyTile(&chair).Index)
	assert.Equal(t, date, state.DateTime)
	assert.Equal(t, calls, state.GetRNG().Calls())
}
