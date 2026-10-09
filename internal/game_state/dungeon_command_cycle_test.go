package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

func TestDungeonCommandCycleNormalPrechargeAndCompletionModes(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 5}
	minuteBefore := state.DateTime.Minute

	turnToken, turnStart := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(turnToken, DungeonCommandOnly)

	assert.Equal(t, 1, turnStart.ElapsedMinutes)
	assert.True(t, turnStart.DungeonActorsMove)
	assert.Equal(t, byte((int(minuteBefore)+1)%60), state.DateTime.Minute)
	assert.Equal(t, 5, state.ActiveDurationEffect.TurnsRemaining, "turning must not run end-player processing")

	moveToken, moveStart := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(moveToken, DungeonFullEndPlayer)

	assert.Equal(t, 1, moveStart.ElapsedMinutes)
	assert.Equal(t, 4, state.ActiveDurationEffect.TurnsRemaining)
	assert.True(t, state.IsAtDungeonCommandBoundary())
}

func TestDungeonQuicknessAlternatesChargedAndFreeCycles(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectQuickness, TurnsRemaining: 10}

	firstToken, first := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(firstToken, DungeonFullEndPlayer)
	secondToken, second := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(secondToken, DungeonFullEndPlayer)
	thirdToken, third := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(thirdToken, DungeonFullEndPlayer)

	assert.Equal(t, 1, first.ElapsedMinutes)
	assert.True(t, first.DungeonActorsMove)
	assert.False(t, first.QuicknessCycleSkip)
	assert.Equal(t, 0, second.ElapsedMinutes)
	assert.False(t, second.DungeonActorsMove)
	assert.True(t, second.QuicknessCycleSkip)
	assert.Equal(t, 1, third.ElapsedMinutes)
	assert.Equal(t, 7, state.ActiveDurationEffect.TurnsRemaining,
		"full completion decrements duration even on a Quickness-skipped actor phase")
}

func TestDungeonTimeStopSuppressesPrechargeButExpiresOnFullCompletion(t *testing.T) {
	state := newMagicLightLifecycleState(t)
	attachDungeonStateForTest(t, state, references.DungeonCells{})
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectTimeStop, TurnsRemaining: 1}
	minuteBefore := state.DateTime.Minute

	token, start := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

	assert.Equal(t, 0, start.ElapsedMinutes)
	assert.True(t, start.TimeStopped)
	assert.False(t, start.DungeonActorsMove)
	assert.Equal(t, minuteBefore, state.DateTime.Minute)
	assert.Equal(t, DurationEffectNone, state.ActiveDurationEffect.EffectType)
}

func TestDungeonTimeStopDoesNotExpireWhenOnlyTurning(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectTimeStop, TurnsRemaining: 1}

	token, _ := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)

	assert.Equal(t, 1, state.ActiveDurationEffect.TurnsRemaining)
}

func TestDungeonCommandCycleRejectsDuplicateAndForeignLifecycle(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	token, _ := state.BeginDungeonCommandCycle()

	assert.PanicsWithValue(t, "dungeon command cycle is already active", func() {
		state.BeginDungeonCommandCycle()
	})
	assert.PanicsWithValue(t, "stale or foreign dungeon command cycle", func() {
		state.CompleteDungeonCommandCycle(DungeonCommandCycle{id: token.id + 1}, DungeonCommandOnly)
	})
	state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)
	assert.PanicsWithValue(t, "no dungeon command cycle is active", func() {
		state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)
	})
}

func TestDungeonCommandCycleRejectsPendingRoomContinuation(t *testing.T) {
	var cells references.DungeonCells
	position := references.Position{X: 1, Y: 1}
	cells.SetCell(0, position, references.DungeonCell(0xa1))
	state := newDungeonStateTestGame(t, cells)

	assert.PanicsWithValue(t, "cannot begin a dungeon command while a room continuation is pending", func() {
		state.BeginDungeonCommandCycle()
	})
}

func TestDungeonQuicknessPhaseResetsWhenEffectIsInactive(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	state.activeDungeon.quicknessSkipNext = true

	token, start := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)

	assert.True(t, start.DungeonActorsMove)
	assert.False(t, state.activeDungeon.quicknessSkipNext)
}

func TestDungeonQuicknessCommandOnlyCyclesAlternateWithoutExpiring(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectQuickness, TurnsRemaining: 10}

	firstToken, first := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(firstToken, DungeonCommandOnly)
	secondToken, second := state.BeginDungeonCommandCycle()
	state.CompleteDungeonCommandCycle(secondToken, DungeonCommandOnly)

	assert.Equal(t, 1, first.ElapsedMinutes)
	assert.Equal(t, 0, second.ElapsedMinutes)
	assert.Equal(t, 10, state.ActiveDurationEffect.TurnsRemaining)
}

func TestDungeonActionsRequireOneUnresolvedCommandCycle(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	assert.PanicsWithValue(t, "dungeon action requires an active command cycle", func() {
		state.TryDungeonAdvance()
	})

	token, _ := state.BeginDungeonCommandCycle()
	state.TurnDungeon(DungeonTurnRight)
	assert.PanicsWithValue(t, "dungeon command cycle already resolved an action", func() {
		state.TurnDungeon(DungeonTurnRight)
	})
	state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)
}

func TestDungeonResolvedActionEnforcesItsCompletionMode(t *testing.T) {
	state := newDungeonStateTestGame(t, references.DungeonCells{})
	token, _ := state.BeginDungeonCommandCycle()
	state.TurnDungeon(DungeonTurnRight)

	assert.PanicsWithValue(t, "dungeon command completion does not match resolved action", func() {
		state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	})
	state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)
}
