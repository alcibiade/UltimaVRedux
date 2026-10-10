package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func performDungeonMove(t *testing.T, state *GameState, movement DungeonMovement) DungeonMoveResult {
	t.Helper()
	token, _ := state.BeginDungeonCommandCycle()
	result := state.tryDungeonMove(movement)
	state.CompleteDungeonCommandCycle(token, result.Completion)

	return result
}

func performDungeonTurn(t *testing.T, state *GameState, turn DungeonTurn) DungeonTurnResult {
	t.Helper()
	token, _ := state.BeginDungeonCommandCycle()
	result := state.TurnDungeon(turn)
	state.CompleteDungeonCommandCycle(token, result.Completion)

	return result
}

func TestDungeonMovementCollisionClasses(t *testing.T) {
	for kind := references.DungeonPassage; kind <= references.DungeonLiveRoom; kind++ {
		for _, movement := range []DungeonMovement{DungeonAdvance, DungeonBackUp} {
			t.Run(fmt.Sprintf("kind_%X_movement_%d", kind, movement), func(t *testing.T) {
				var cells references.DungeonCells
				target := references.Position{X: 2, Y: 1}
				cell := references.DungeonCell(byte(kind) << 4)
				if kind == references.DungeonEnergyField {
					cell = 0x8b // The original treats this electric-looking variant as inert.
				}
				cells.SetCell(0, target, cell)
				state := newDungeonStateTestGame(t, cells)
				if movement == DungeonBackUp {
					state.activeDungeon.facing = references.DungeonFacingWest
				}

				result := performDungeonMove(t, state, movement)
				expected := dungeonCellAllowsMovement(kind, movement)
				assert.Equal(t, expected, result.Outcome == DungeonMoveMoved)
				assert.Equal(t, DungeonFullEndPlayer, result.Completion)
				if movement == DungeonAdvance &&
					(kind == references.DungeonClearedRoom || kind == references.DungeonLiveRoom) {
					assert.True(t, state.activeDungeon.roomContinuation.encounterPending)
				}
			})
		}
	}
}

func TestDungeonMovementWrapsModuloEight(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	state.MapState.PlayerLocation.Position = references.Position{X: 0, Y: 0}
	state.activeDungeon.facing = references.DungeonFacingNorth

	result := performDungeonMove(t, state, DungeonAdvance)

	require.Equal(t, DungeonMoveMoved, result.Outcome)
	assert.Equal(t, references.Position{X: 0, Y: 7}, state.MapState.PlayerLocation.Position)
}

func TestDungeonMovementUsesSemanticAuthorityNotProjection(t *testing.T) {
	var cells references.DungeonCells
	target := references.Position{X: 2, Y: 1}
	cells.SetCell(0, target, references.DungeonCell(0xb0))
	state := newDungeonStateTestGame(t, cells)
	projection := state.MapState.LayeredMaps.GetLayeredMap(references.DungeonMapType, 0)
	projection.SetTileByLayer(map_state.MapLayer, &target, indexes.BrickFloor)

	result := performDungeonMove(t, state, DungeonAdvance)

	assert.Equal(t, DungeonMoveTerrainBlocked, result.Outcome)
	assert.Equal(t, references.Position{X: 1, Y: 1}, state.MapState.PlayerLocation.Position)
}

func TestDungeonWandererBlocksMovement(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	state.activeDungeon.wanderer = &DungeonWanderer{
		Monster:          references.MonsterReaper,
		Floor:            0,
		Position:         references.Position{X: 2, Y: 1},
		PreviousPosition: references.Position{X: 2, Y: 1},
	}

	result := performDungeonMove(t, state, DungeonAdvance)

	assert.Equal(t, DungeonMoveWandererBlocked, result.Outcome)
}

func TestDungeonElectricFieldReturnsTypedOutcome(t *testing.T) {
	target := references.Position{X: 2, Y: 1}
	state, callbacks := newDungeonInteractionTestState(t, 0x00, 21)
	setDungeonInteractionCell(t, state, target, 0x83)
	active := activeDungeonHazardCharacters(state)
	beforeHP := dungeonHazardHP(active)
	callbacks.Reset()

	result := performDungeonMove(t, state, DungeonAdvance)

	assert.Equal(t, DungeonMoveElectricField, result.Outcome)
	assert.Equal(t, references.Position{X: 1, Y: 1}, state.MapState.PlayerLocation.Position)
	assert.Contains(t, callbacks.Messages, "Ouch!")
	assert.Contains(t, callbacks.Messages, "Electric field!")
	for index, character := range active {
		assert.Less(t, character.GetCurrentHp(), beforeHP[index])
	}
}

func TestDungeonElectricOpeningVariantRemainsInert(t *testing.T) {
	state, callbacks := newDungeonInteractionTestState(t, 0x00, 21)
	target := references.Position{X: 2, Y: 1}
	setDungeonInteractionCell(t, state, target, 0x8b)
	active := activeDungeonHazardCharacters(state)
	beforeHP := dungeonHazardHP(active)
	beforeRNG := state.GetRNG().Calls()
	callbacks.Reset()

	result := performDungeonMove(t, state, DungeonAdvance)

	assert.Equal(t, DungeonMoveMoved, result.Outcome)
	assert.Equal(t, target, state.MapState.PlayerLocation.Position)
	assert.Equal(t, beforeHP, dungeonHazardHP(active))
	assert.Equal(t, beforeRNG, state.GetRNG().Calls())
	assert.NotContains(t, callbacks.Messages, "Electric field!")
}

func TestDungeonDoorwayBlocksQuarterTurnsButNotTurnaround(t *testing.T) {
	var cells references.DungeonCells
	cells.SetCell(0, references.Position{X: 1, Y: 1}, references.DungeonCell(0xe8))
	state := newDungeonStateTestGame(t, cells)

	left := performDungeonTurn(t, state, DungeonTurnLeft)
	right := performDungeonTurn(t, state, DungeonTurnRight)
	around := performDungeonTurn(t, state, DungeonTurnAround)

	assert.Equal(t, DungeonTurnBlockedInDoorway, left.Outcome)
	assert.Equal(t, DungeonTurnBlockedInDoorway, right.Outcome)
	assert.Equal(t, DungeonTurned, around.Outcome)
	assert.Equal(t, references.DungeonFacingWest, state.activeDungeon.facing)
	assert.Equal(t, references.DungeonLastWest, state.activeDungeon.lastDirection)
	assert.Equal(t, DungeonCommandOnly, around.Completion)
}

func TestDungeonTurnsUseOriginalClockwiseFacing(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})

	right := performDungeonTurn(t, state, DungeonTurnRight)
	left := performDungeonTurn(t, state, DungeonTurnLeft)

	assert.Equal(t, references.DungeonFacingSouth, right.After)
	assert.Equal(t, references.DungeonFacingEast, left.After)
}

func TestDungeonMovementRecordsLastDirectionEvenWhenBlocked(t *testing.T) {
	var cells references.DungeonCells
	cells.SetCell(0, references.Position{X: 2, Y: 1}, references.DungeonCell(0xb0))
	state := newDungeonStateTestGame(t, cells)

	result := performDungeonMove(t, state, DungeonAdvance)

	assert.Equal(t, DungeonMoveTerrainBlocked, result.Outcome)
	assert.Equal(t, references.DungeonLastEast, state.activeDungeon.lastDirection)
}
