package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestOutdoorWindowInitializationAndMovementHistory(t *testing.T) {
	for _, floor := range []references.FloorNumber{0, -1} {
		state := &GameState{}
		state.MapState.PlayerLocation = references.PlayerLocation{
			Location: references.BritanniaUnderworld, Floor: floor,
			Position: references.Position{X: 119, Y: 120},
		}
		state.InitializeOutdoorWindow()
		assert.Equal(t, references.Position{X: 96, Y: 112}, state.outdoorWindow.origin)
		assert.Equal(t, floor, state.outdoorWindow.floor)
		for range 3 { // relative 24, 25, 26 retain the window
			state.MovePlayerOneStep(references.Right)
			assert.Equal(t, references.Position{X: 96, Y: 112}, state.outdoorWindow.origin)
		}
		state.MovePlayerOneStep(references.Right) // relative 27 shifts once
		assert.Equal(t, references.Position{X: 123, Y: 120}, state.MapState.PlayerLocation.Position)
		assert.Equal(t, references.Position{X: 112, Y: 112}, state.outdoorWindow.origin)
		for range 6 { // reverse to relative 5; do not reinitialize from position
			state.MovePlayerOneStep(references.Left)
			assert.Equal(t, references.Position{X: 112, Y: 112}, state.outdoorWindow.origin)
		}
		state.MovePlayerOneStep(references.Left)
		assert.Equal(t, references.Position{X: 116, Y: 120}, state.MapState.PlayerLocation.Position)
		assert.Equal(t, references.Position{X: 96, Y: 112}, state.outdoorWindow.origin)
		for range 3 {
			state.MovePlayerOneStep(references.Up)
		}
		assert.Equal(t, references.Position{X: 96, Y: 112}, state.outdoorWindow.origin)
		state.MovePlayerOneStep(references.Up)
		assert.Equal(t, references.Position{X: 96, Y: 96}, state.outdoorWindow.origin)
		for range 6 {
			state.MovePlayerOneStep(references.Down)
		}
		assert.Equal(t, references.Position{X: 96, Y: 96}, state.outdoorWindow.origin)
		state.MovePlayerOneStep(references.Down)
		assert.Equal(t, references.Position{X: 96, Y: 112}, state.outdoorWindow.origin)
	}
}

func TestOutdoorWindowWrappedStepAndSingleConditionalShift(t *testing.T) {
	state := &GameState{}
	state.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.BritanniaUnderworld, Position: references.Position{X: 255, Y: 120},
	}
	state.InitializeOutdoorWindow()
	state.MovePlayerOneStep(references.Right)
	assert.Equal(t, references.Position{X: 0, Y: 120}, state.MapState.PlayerLocation.Position)
	assert.Equal(t, references.Position{X: 240, Y: 112}, state.outdoorWindow.origin)
	for range 11 {
		state.MovePlayerOneStep(references.Right)
	}
	assert.Equal(t, references.Position{X: 0, Y: 112}, state.outdoorWindow.origin)

	// Direct displacement can leave the perpendicular axis out of the safe band.
	// A horizontal step must shift X once, without normalizing/recentering Y.
	state.SetPlayerPosition(references.Position{X: 10, Y: 116})
	state.MovePlayerOneStep(references.Right)
	assert.Equal(t, references.Position{X: 16, Y: 112}, state.outdoorWindow.origin)
	assert.Equal(t, references.Position{X: 11, Y: 116}, state.MapState.PlayerLocation.Position)
}

func newBlinkHistoryFixture(t *testing.T, floor references.FloorNumber) *GameState {
	t.Helper()
	state, _ := NewIntegrationTestBuilder(t).WithFloor(floor).
		WithPlayerAt(119, 120).WithRandomSeed(35).Build()
	state.CurrentNPCAIController = state.GetCurrentLargeMapNPCAIController()
	state.CurrentNPCAIController.GetNpcs().ReplaceWith(nil)
	state.DebugOptions.MonsterGen = false
	setBlinkRayTiles(t, state.GetCurrentLayeredMap(), references.Position{X: 120, Y: 120}, references.Right, indexes.Forest1)
	for _, x := range []references.Coordinate{100, 124, 140} {
		state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapOverrideLayer, &references.Position{X: x, Y: 120}, indexes.Grass)
	}
	state.MovePlayerOneStep(references.Right)

	return state
}

func TestBlinkHistoryCounterexamplesAndPostSuccessInitialization(t *testing.T) {
	for _, floor := range []references.FloorNumber{0, -1} {
		for _, test := range []struct {
			direction references.Direction
			fresh     bool
			result    BlinkResult
			x         references.Coordinate
		}{
			{references.Left, false, BlinkSuccess, 100},
			{references.Right, false, BlinkSuccess, 124},
			{references.Left, true, BlinkBlocked, 120},
			{references.Right, true, BlinkSuccess, 140},
		} {
			state := newBlinkHistoryFixture(t, floor)
			if test.fresh {
				state.InitializeOutdoorWindow()
			}
			before, window, calls := state.DateTime, state.outdoorWindow, state.GetRNG().Calls()
			assert.Equal(t, test.result, state.BlinkTeleport(test.direction))
			assert.Equal(t, references.Position{X: test.x, Y: 120}, state.MapState.PlayerLocation.Position)
			assert.Equal(t, before, state.DateTime)
			assert.Equal(t, calls, state.GetRNG().Calls())
			if test.result == BlinkBlocked {
				assert.Equal(t, window, state.outdoorWindow)
			}
			if !test.fresh && test.direction == references.Right {
				assert.Equal(t, references.Position{X: 112, Y: 112}, state.outdoorWindow.origin)
				assert.Equal(t, BlinkSuccess, state.BlinkTeleport(references.Right))
				assert.Equal(t, references.Position{X: 140, Y: 120}, state.MapState.PlayerLocation.Position)
			}
		}
	}
}

func TestBlinkNumericSeamsRejectBothAxesOnBothPlanes(t *testing.T) {
	for _, floor := range []references.FloorNumber{0, -1} {
		for _, start := range []references.Position{{X: 1, Y: 120}, {X: 120, Y: 1}, {X: 0, Y: 0}} {
			state, _ := NewIntegrationTestBuilder(t).WithFloor(floor).WithPlayerAt(int(start.X), int(start.Y)).Build()
			for _, direction := range []references.Direction{references.Up, references.Down, references.Left, references.Right} {
				setBlinkRayTiles(t, state.GetCurrentLayeredMap(), start, direction, indexes.Grass)
				window := state.outdoorWindow
				assert.Equal(t, BlinkBlocked, state.BlinkTeleport(direction))
				assert.Equal(t, start, state.MapState.PlayerLocation.Position)
				assert.Equal(t, window, state.outdoorWindow)
			}
		}
		state, _ := NewIntegrationTestBuilder(t).WithFloor(floor).WithPlayerAt(255, 120).Build()
		setBlinkRayTiles(t, state.GetCurrentLayeredMap(), state.MapState.PlayerLocation.Position, references.Right, indexes.Grass)
		assert.Equal(t, BlinkBlocked, state.BlinkTeleport(references.Right))
	}
}

func TestBlinkIgnoresOutdoorOccupantsAndInterveningObstacles(t *testing.T) {
	state := newBlinkHistoryFixture(t, -1)
	layer := state.GetCurrentLayeredMap()
	layer.SetTileByLayer(map_state.MapOverrideLayer, &references.Position{X: 121, Y: 120}, indexes.TallMountains)
	layer.SetTileByLayer(map_state.MapOverrideLayer, &references.Position{X: 122, Y: 120}, indexes.Water1)
	endpoint := references.Position{X: 124, Y: 120}
	occupant := map_units.NewEnemyNPC(*state.enemyReference(references.MonsterOrc), 1)
	occupant.SetPos(endpoint)
	occupant.SetFloor(-1)
	occupant.SetVisible(true)
	state.CurrentNPCAIController.GetNpcs().ReplaceWith([]map_units.MapUnit{&occupant})
	require.True(t, state.ObjectPresentAt(&endpoint))
	require.False(t, state.IsPassable(&endpoint), "ordinary movement still rejects an occupant")

	// Identical surface coordinates deliberately contain different terrain.
	surface := state.MapState.LayeredMaps.GetLayeredMap(references.LargeMapType, 0)
	setBlinkRayTiles(t, surface, state.MapState.PlayerLocation.Position, references.Right, indexes.Forest1)
	surface.SetTileByLayer(map_state.MapOverrideLayer, &references.Position{X: 126, Y: 120}, indexes.Grass)
	require.Equal(t, BlinkSuccess, state.BlinkTeleport(references.Right))
	assert.Equal(t, endpoint, state.MapState.PlayerLocation.Position)
	state.TransitionToLocation(references.BritanniaUnderworld, 0, references.Position{X: 120, Y: 120}, TransitionFullLocation, false)
	require.Equal(t, BlinkSuccess, state.BlinkTeleport(references.Right))
	assert.Equal(t, references.Position{X: 126, Y: 120}, state.MapState.PlayerLocation.Position)
}

func TestOutdoorWindowMissingPlaneInitializationIsNotRepairedByCast(t *testing.T) {
	state := newBlinkHistoryFixture(t, 0)
	state.MapState.PlayerLocation.Floor = -1
	assert.Panics(t, func() { state.BlinkTeleport(references.Right) })
	assert.Panics(t, func() { state.MovePlayerOneStep(references.Right) })
}
