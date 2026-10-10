package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type scriptedDungeonRandom struct {
	t      *testing.T
	values []int
	index  int
}

func newScriptedDungeonRandom(t *testing.T, values ...int) *scriptedDungeonRandom {
	t.Helper()

	return &scriptedDungeonRandom{t: t, values: values}
}

func (random *scriptedDungeonRandom) next(minimum, maximum int) int {
	random.t.Helper()
	require.Less(random.t, random.index, len(random.values), "unexpected RNG call for range [%d,%d]", minimum, maximum)
	value := random.values[random.index]
	random.index++
	require.GreaterOrEqual(random.t, value, minimum)
	require.LessOrEqual(random.t, value, maximum)

	return value
}

func (random *scriptedDungeonRandom) requireExhausted() {
	random.t.Helper()
	assert.Equal(random.t, len(random.values), random.index)
}

func TestDungeonWandererGenerationSelectsEveryDefinitionUniformly(t *testing.T) {
	const placement = 3 + 3*int(references.XDungeonMapTiles)
	for definitionIndex := range references.DungeonWanderingMonsterCount {
		definition, ok := references.DungeonWanderingMonsterDefinitionAt(definitionIndex)
		require.True(t, ok)
		t.Run(definition.Monster.String(), func(t *testing.T) {
			state := newDungeonStateTestGame(t, references.DungeonCells{})
			rolls := []int{definitionIndex, placement}
			if definition.Monster == references.MonsterSpider || definition.Monster == references.MonsterSlime {
				rolls = append(rolls, 0)
			}
			random := newScriptedDungeonRandom(t, rolls...)

			require.True(t, state.generateDungeonWandererWithRandom(random.next))

			random.requireExhausted()
			wanderer := state.activeDungeon.wanderer
			require.NotNil(t, wanderer)
			assert.Equal(t, definition.Monster, wanderer.Monster)
			assert.Equal(t, references.FloorNumber(0), wanderer.Floor)
			assert.Equal(t, references.Position{X: 3, Y: 3}, wanderer.Position)
			assert.Equal(t, wanderer.Position, wanderer.PreviousPosition)
			assert.False(t, wanderer.OnCeiling)
		})
	}
}

func TestDungeonWandererGenerationPlacementClassMatrix(t *testing.T) {
	const placement = 3 + 3*int(references.XDungeonMapTiles)
	position := references.Position{X: 3, Y: 3}
	for kind := references.DungeonPassage; kind <= references.DungeonLiveRoom; kind++ {
		t.Run(fmt.Sprintf("kind_%X", kind), func(t *testing.T) {
			var cells references.DungeonCells
			require.True(t, cells.SetCell(0, position, references.DungeonCell(byte(kind)<<4|0x0f)))
			state := newDungeonStateTestGame(t, cells)
			eligible := kind < references.DungeonPitOrTrap || kind == references.DungeonOpenChest
			rolls := []int{0, placement}
			if !eligible {
				for range dungeonWandererPlacementAttempts - 1 {
					rolls = append(rolls, placement)
				}
			}
			random := newScriptedDungeonRandom(t, rolls...)

			generated := state.generateDungeonWandererWithRandom(random.next)

			random.requireExhausted()
			assert.Equal(t, eligible, generated)
			if eligible {
				require.NotNil(t, state.activeDungeon.wanderer)
				assert.Equal(t, position, state.activeDungeon.wanderer.Position)
			} else {
				assert.Nil(t, state.activeDungeon.wanderer)
			}
		})
	}
}

func TestDungeonWandererGenerationRejectsMatchingRowOrColumn(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	random := newScriptedDungeonRandom(t,
		0,
		3+1*int(references.XDungeonMapTiles),
		1+3*int(references.XDungeonMapTiles),
		3+3*int(references.XDungeonMapTiles),
	)

	require.True(t, state.generateDungeonWandererWithRandom(random.next))

	random.requireExhausted()
	require.NotNil(t, state.activeDungeon.wanderer)
	assert.Equal(t, references.Position{X: 3, Y: 3}, state.activeDungeon.wanderer.Position)
}

func TestDungeonWandererGenerationCeilingRollHasExactFiftyOnePercentBoundary(t *testing.T) {
	const placement = 3 + 3*int(references.XDungeonMapTiles)
	for _, definitionIndex := range []int{2, 4} {
		definition, ok := references.DungeonWanderingMonsterDefinitionAt(definitionIndex)
		require.True(t, ok)
		t.Run(definition.Monster.String(), func(t *testing.T) {
			ceilingCount := 0
			for ceilingRoll := range 100 {
				state := newDungeonStateTestGame(t, references.DungeonCells{})
				random := newScriptedDungeonRandom(t, definitionIndex, placement, ceilingRoll)

				require.True(t, state.generateDungeonWandererWithRandom(random.next))
				random.requireExhausted()
				if state.activeDungeon.wanderer.OnCeiling {
					ceilingCount++
				}
				assert.Equal(t, ceilingRoll > 48, state.activeDungeon.wanderer.OnCeiling)
			}
			assert.Equal(t, 51, ceilingCount)
		})
	}
}

func TestDungeonWandererGenerationCeilingRollWaitsForSuccessfulPlacement(t *testing.T) {
	const validPlacement = 3 + 3*int(references.XDungeonMapTiles)
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	random := newScriptedDungeonRandom(t,
		2,
		3+1*int(references.XDungeonMapTiles),
		validPlacement,
		49,
	)

	require.True(t, state.generateDungeonWandererWithRandom(random.next))

	random.requireExhausted()
	assert.True(t, state.activeDungeon.wanderer.OnCeiling)
}

func TestDungeonWandererGenerationExhaustsEightSamplesWithoutRetry(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	state.activeDungeon.wanderer = &DungeonWanderer{
		Monster: references.MonsterRat, Floor: 0,
		Position: references.Position{X: 3, Y: 3}, PreviousPosition: references.Position{X: 3, Y: 3},
	}
	rolls := []int{2}
	for range dungeonWandererPlacementAttempts {
		rolls = append(rolls, 3+1*int(references.XDungeonMapTiles))
	}
	random := newScriptedDungeonRandom(t, rolls...)

	assert.False(t, state.generateDungeonWandererWithRandom(random.next))

	random.requireExhausted()
	assert.Nil(t, state.activeDungeon.wanderer)
	beforeCalls := state.GetRNG().Calls()
	result := state.moveDungeonWanderer()
	assert.Equal(t, references.MonsterNone, result.Monster)
	assert.Equal(t, beforeCalls, state.GetRNG().Calls(), "nil state must not retry generation")
}

func TestDungeonWandererMovementClassMatrix(t *testing.T) {
	position := references.Position{X: 3, Y: 3}
	target := references.Position{X: 3, Y: 2}
	for kind := references.DungeonPassage; kind <= references.DungeonLiveRoom; kind++ {
		t.Run(fmt.Sprintf("kind_%X", kind), func(t *testing.T) {
			var cells references.DungeonCells
			require.True(t, cells.SetCell(0, target, references.DungeonCell(byte(kind)<<4|0x0f)))
			state := newDungeonWandererMovementTestState(t, cells, references.MonsterRat, position, position)
			eligible := kind < references.DungeonClearedRoom &&
				kind != references.DungeonPitOrTrap && kind != references.DungeonEnergyField
			rolls := []int{0}
			if !eligible {
				for range dungeonWandererPlacementAttempts - 1 {
					rolls = append(rolls, 0)
				}
			}
			random := newScriptedDungeonRandom(t, rolls...)

			result := state.moveDungeonWandererWithRandom(random.next)

			random.requireExhausted()
			assert.Equal(t, eligible, result.Moved)
			assert.Equal(t, map[bool]int{true: 1, false: 8}[eligible], result.Attempts)
			if eligible {
				assert.Equal(t, target, state.activeDungeon.wanderer.Position)
				assert.Equal(t, position, state.activeDungeon.wanderer.PreviousPosition)
			} else {
				assert.Equal(t, position, state.activeDungeon.wanderer.Position)
				assert.Equal(t, position, state.activeDungeon.wanderer.PreviousPosition)
			}
		})
	}
}

func TestDungeonWandererMovementWraps(t *testing.T) {
	position := references.Position{X: 0, Y: 0}
	state := newDungeonWandererMovementTestState(
		t, references.DungeonCells{}, references.MonsterRat, position, position,
	)
	random := newScriptedDungeonRandom(t, 0)

	result := state.moveDungeonWandererWithRandom(random.next)

	random.requireExhausted()
	assert.True(t, result.Moved)
	assert.Equal(t, references.Position{X: 0, Y: 7}, result.FinalPosition)
}

func TestDungeonWandererMovementPreviousSquareRequiresExtraOneInEight(t *testing.T) {
	position := references.Position{X: 3, Y: 3}
	previous := references.Position{X: 3, Y: 2}
	state := newDungeonWandererMovementTestState(
		t, references.DungeonCells{}, references.MonsterRat, position, previous,
	)
	random := newScriptedDungeonRandom(t, 0, 0, 1)

	result := state.moveDungeonWandererWithRandom(random.next)

	random.requireExhausted()
	assert.True(t, result.Moved)
	assert.Equal(t, references.Position{X: 4, Y: 3}, result.FinalPosition)
	assert.Equal(t, position, state.activeDungeon.wanderer.PreviousPosition)

	state = newDungeonWandererMovementTestState(
		t, references.DungeonCells{}, references.MonsterRat, position, previous,
	)
	random = newScriptedDungeonRandom(t, 0, 1)

	result = state.moveDungeonWandererWithRandom(random.next)

	random.requireExhausted()
	assert.True(t, result.Moved)
	assert.Equal(t, previous, result.FinalPosition)
}

func TestDungeonWandererMovementDoesNotRollForBlockedPreviousSquare(t *testing.T) {
	position := references.Position{X: 3, Y: 3}
	previous := references.Position{X: 3, Y: 2}
	var cells references.DungeonCells
	require.True(t, cells.SetCell(0, previous, 0x60))
	state := newDungeonWandererMovementTestState(t, cells, references.MonsterRat, position, previous)
	random := newScriptedDungeonRandom(t, 0, 1)

	result := state.moveDungeonWandererWithRandom(random.next)

	random.requireExhausted()
	assert.True(t, result.Moved)
	assert.Equal(t, references.Position{X: 4, Y: 3}, result.FinalPosition)
}

func TestDungeonWandererReaperAndOffFloorOccupantConsumeNoMovementRNG(t *testing.T) {
	position := references.Position{X: 3, Y: 3}
	for _, test := range []struct {
		name    string
		monster references.MonsterType
		floor   references.FloorNumber
	}{
		{name: "Reaper", monster: references.MonsterReaper, floor: 0},
		{name: "other floor", monster: references.MonsterRat, floor: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			state := newDungeonWandererMovementTestState(
				t, references.DungeonCells{}, test.monster, position, position,
			)
			state.activeDungeon.wanderer.Floor = test.floor
			beforeCalls := state.GetRNG().Calls()

			result := state.moveDungeonWanderer()

			assert.False(t, result.Moved)
			assert.False(t, result.Contact)
			assert.Equal(t, beforeCalls, state.GetRNG().Calls())
			assert.Equal(t, position, state.activeDungeon.wanderer.Position)
		})
	}
}

func TestDungeonWandererContactRewindsAndReportsAttackOrigin(t *testing.T) {
	position := references.Position{X: 3, Y: 3}
	party := references.Position{X: 3, Y: 2}
	state := newDungeonWandererMovementTestState(
		t, references.DungeonCells{}, references.MonsterGhost, position, position,
	)
	state.MapState.PlayerLocation.Position = party
	random := newScriptedDungeonRandom(t, 0)

	result := state.moveDungeonWandererWithRandom(random.next)

	random.requireExhausted()
	assert.True(t, result.Moved)
	assert.True(t, result.Contact)
	assert.Equal(t, position, result.From)
	assert.Equal(t, party, result.Target)
	assert.Equal(t, position, result.AttackOrigin)
	assert.Equal(t, position, result.FinalPosition)
	assert.Equal(t, position, state.activeDungeon.wanderer.Position)
	assert.Equal(t, position, state.activeDungeon.wanderer.PreviousPosition)
}

func TestDungeonCommandActorCadenceMovesWandererOnlyOnActivePhases(t *testing.T) {
	position := references.Position{X: 3, Y: 3}
	state := newDungeonWandererMovementTestState(
		t, references.DungeonCells{}, references.MonsterGhost, position, position,
	)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectQuickness, TurnsRemaining: 10}
	state.SetRandomSeed(265)

	firstToken, firstStart := state.BeginDungeonCommandCycle()
	beforeFirst := state.GetRNG().Calls()
	firstResult := state.CompleteDungeonCommandCycle(firstToken, DungeonFullEndPlayer)
	afterFirst := state.GetRNG().Calls()

	secondToken, secondStart := state.BeginDungeonCommandCycle()
	beforeSecond := state.GetRNG().Calls()
	secondResult := state.CompleteDungeonCommandCycle(secondToken, DungeonFullEndPlayer)

	assert.True(t, firstStart.DungeonActorsMove)
	assert.True(t, firstResult.WandererMovement.Moved)
	assert.Greater(t, afterFirst, beforeFirst)
	assert.False(t, secondStart.DungeonActorsMove)
	assert.Equal(t, references.MonsterNone, secondResult.WandererMovement.Monster)
	assert.Equal(t, beforeSecond, state.GetRNG().Calls())
}

func TestDungeonCommandCycleReturnsWandererContactForEncounterOrchestration(t *testing.T) {
	const seed = 3265
	position := references.Position{X: 3, Y: 3}
	state := newDungeonWandererMovementTestState(
		t, references.DungeonCells{}, references.MonsterGhost, position, position,
	)
	control := deterministic_random.New(seed)
	direction := dungeonWandererDirection(control.Intn(4))
	party := direction.GetNewPositionInDirection(&position).GetWrapped(
		references.XDungeonMapTiles,
		references.YDungeonMapTiles,
	)
	state.MapState.PlayerLocation.Position = *party
	state.SetRandomSeed(seed)

	result := state.processDungeonEndPlayerBeforeCell(true)

	assert.True(t, result.Contact)
	assert.Equal(t, position, result.AttackOrigin)
	assert.Equal(t, *party, result.Target)
	assert.Equal(t, position, state.activeDungeon.wanderer.Position)
}

func TestDungeonCommandActorCadenceTimeStopSuppressesMovement(t *testing.T) {
	position := references.Position{X: 3, Y: 3}
	state := newDungeonWandererMovementTestState(
		t, references.DungeonCells{}, references.MonsterGhost, position, position,
	)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectTimeStop, TurnsRemaining: 2}
	beforeCalls := state.GetRNG().Calls()

	token, start := state.BeginDungeonCommandCycle()
	result := state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

	assert.True(t, start.TimeStopped)
	assert.False(t, start.DungeonActorsMove)
	assert.Equal(t, references.MonsterNone, result.WandererMovement.Monster)
	assert.Equal(t, beforeCalls, state.GetRNG().Calls())
	assert.Equal(t, position, state.activeDungeon.wanderer.Position)
}

func TestDungeonCommandActorCadenceTimeStopPrecedesOverlappingQuickness(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	state.activeDungeon.quicknessSkipNext = true

	actorsMove, quicknessSkip := state.activeDungeon.beginActorPhase(true, true)

	assert.False(t, actorsMove)
	assert.False(t, quicknessSkip)
	assert.True(t, state.activeDungeon.quicknessSkipNext, "suppressed overlap must preserve the Quickness phase")
}

func newDungeonWandererMovementTestState(
	t *testing.T,
	cells references.DungeonCells,
	monster references.MonsterType,
	position references.Position,
	previous references.Position,
) *GameState {
	t.Helper()
	state := newDungeonStateTestGame(t, cells)
	state.MapState.PlayerLocation.Position = references.Position{X: 7, Y: 7}
	state.activeDungeon.wanderer = &DungeonWanderer{
		Monster: monster, Floor: 0, Position: position, PreviousPosition: previous,
	}

	return state
}
