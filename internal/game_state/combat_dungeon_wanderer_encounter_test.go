package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	beradrng "github.com/bradhannah/berad-sdk/modules/berad-rng"
)

func TestDirectDungeonAttackStartsGeneratedCombatWithoutEndPlayerProcessing(t *testing.T) {
	state, callbacks := newDungeonWandererEncounterTestState(t, 0)
	if state == nil {
		return
	}
	state.MapState.Lighting.RestoreTorchMinutes(0)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 5}
	state.activeDungeon.wanderer = &DungeonWanderer{
		Monster: references.MonsterSpider, Floor: 0,
		Position: references.Position{X: 2, Y: 1}, PreviousPosition: references.Position{X: 3, Y: 1},
		OnCeiling: true,
	}
	callbacks.Reset()

	token, _ := state.BeginDungeonCommandCycle()
	require.True(t, state.ActionAttackDungeonMap(references.NoneDirection))
	state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)

	require.NotNil(t, state.CombatScene)
	require.NotNil(t, state.CombatScene.DungeonWandererOrigin)
	assert.Equal(t, DungeonWandererEncounterDirect, state.CombatScene.DungeonWandererOrigin.Kind)
	assert.Equal(t, CombatExitDungeonEncounter, state.CombatScene.ExitContext)
	assert.Equal(t, 5, state.ActiveDurationEffect.TurnsRemaining)
	assert.True(t, state.IsAtDungeonCommandBoundary(), "direct Attack completes before tactical combat")
	assert.NotContains(t, callbacks.Messages, "What?")
}

func TestDirectDungeonAttackRequiresWandererInWrappedFacingCell(t *testing.T) {
	state, callbacks := newDungeonWandererEncounterTestState(t, 0)
	if state == nil {
		return
	}
	state.MapState.PlayerLocation.Position = references.Position{X: 7, Y: 1}
	state.activeDungeon.wanderer = &DungeonWanderer{
		Monster: references.MonsterRat, Floor: 0,
		Position: references.Position{X: 0, Y: 1}, PreviousPosition: references.Position{X: 0, Y: 2},
	}
	callbacks.Reset()

	token, _ := state.BeginDungeonCommandCycle()
	require.True(t, state.ActionAttackDungeonMap(references.Left), "input direction is ignored in favor of facing")
	state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)
	require.NotNil(t, state.CombatScene)

	state.EndCombat(CombatVictory)
	state.activeDungeon.wanderer = nil
	callbacks.Reset()
	token, _ = state.BeginDungeonCommandCycle()
	assert.False(t, state.ActionAttackDungeonMap(references.NoneDirection))
	state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)
	assert.Equal(t, []string{"What?"}, callbacks.Messages)
}

func TestDirectDungeonAttackReturnIgnoresCardinalExitAndRegenerates(t *testing.T) {
	state, _ := newDungeonWandererEncounterTestState(t, 3)
	if state == nil {
		return
	}
	originalPosition := state.MapState.PlayerLocation.Position
	state.activeDungeon.wanderer = &DungeonWanderer{
		Monster: references.MonsterGhost, Floor: 3,
		Position: references.Position{X: 2, Y: 1}, PreviousPosition: references.Position{X: 2, Y: 2},
	}

	token, _ := state.BeginDungeonCommandCycle()
	require.True(t, state.ActionAttackDungeonMap(references.NoneDirection))
	state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)
	beforeReturnRNG := state.GetRNG().Calls()
	state.CombatScene.ExitRoute = CombatExitWest
	state.EndCombat(CombatVictory)

	assert.Equal(t, originalPosition, state.MapState.PlayerLocation.Position)
	assert.Equal(t, references.FloorNumber(3), state.MapState.PlayerLocation.Floor)
	assert.Greater(t, state.GetRNG().Calls(), beforeReturnRNG, "return must attempt one replacement generation")
}

func TestDirectDungeonAttackReturnAppliesVerticalExitOnly(t *testing.T) {
	state, _ := newDungeonWandererEncounterTestState(t, 3)
	if state == nil {
		return
	}
	state.activeDungeon.wanderer = &DungeonWanderer{
		Monster: references.MonsterGhost, Floor: 3,
		Position: references.Position{X: 2, Y: 1}, PreviousPosition: references.Position{X: 2, Y: 2},
	}
	position := state.MapState.PlayerLocation.Position
	_, err := state.TransformDungeonCell(2, position, func(references.DungeonCell) references.DungeonCell {
		return 0xa0
	})
	require.NoError(t, err)

	token, _ := state.BeginDungeonCommandCycle()
	require.True(t, state.ActionAttackDungeonMap(references.NoneDirection))
	state.CompleteDungeonCommandCycle(token, DungeonCommandOnly)
	state.CombatScene.ExitRoute = CombatExitUp
	state.EndCombat(CombatVictory)

	assert.Equal(t, references.FloorNumber(2), state.MapState.PlayerLocation.Floor)
	assert.Equal(t, references.DungeonArrivedFromBelow, state.activeDungeon.lastDirection)
	assert.False(t, state.HasPendingDungeonRoomEncounter(), "direct Attack must not enter destination rooms")
	assert.True(t, state.IsAtDungeonCommandBoundary())
}

func TestAutomaticDungeonContactSuspendsAndResumesFullEndPlayer(t *testing.T) {
	state, callbacks, party, attackOrigin := prepareDungeonWandererContactTest(t, 0)
	if state == nil {
		return
	}
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 5}
	callbacks.Reset()

	token, _ := state.BeginDungeonCommandCycle()
	result := state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)

	require.True(t, result.WandererMovement.Contact)
	assert.Equal(t, attackOrigin, result.WandererMovement.AttackOrigin)
	assert.Nil(t, state.CombatScene)
	assert.False(t, state.IsAtDungeonCommandBoundary())
	assert.True(t, state.EventQueueManager.HasWorkflow(dungeonWandererContactWorkflowName))
	assert.Equal(t, 5, state.ActiveDurationEffect.TurnsRemaining)
	err := state.ValidateSaveSnapshotEligibility()
	require.Error(t, err)

	drainDungeonWandererContactWorkflow(t, state)
	require.NotNil(t, state.CombatScene)
	require.NotNil(t, state.CombatScene.DungeonWandererOrigin)
	assert.Equal(t, DungeonWandererEncounterContact, state.CombatScene.DungeonWandererOrigin.Kind)
	expectedFacing, ok := dungeonWandererAttackFacing(party, attackOrigin)
	require.True(t, ok)
	assert.Equal(t, expectedFacing, state.activeDungeon.facing)
	assert.Contains(t, callbacks.Messages, "Attacked from the "+dungeonWandererFacingName(expectedFacing)+"!")

	state.CombatScene.ExitRoute = CombatExitWest
	state.EndCombat(CombatVictory)

	expectedPosition, moved := applyDungeonCardinalExit(party, CombatExitWest)
	require.True(t, moved)
	assert.Equal(t, expectedPosition, state.MapState.PlayerLocation.Position)
	assert.Equal(t, references.DungeonFacingWest, state.activeDungeon.facing)
	assert.Equal(t, 4, state.ActiveDurationEffect.TurnsRemaining)
	assert.True(t, state.IsAtDungeonCommandBoundary())
}

func TestAutomaticDungeonContactExitOntoHiddenPitFallsBeforeCommandCompletes(t *testing.T) {
	state, _, party, _ := prepareDungeonWandererContactTest(t, 0)
	if state == nil {
		return
	}
	pitPosition, moved := applyDungeonCardinalExit(party, CombatExitWest)
	require.True(t, moved)
	_, err := state.TransformDungeonCell(0, pitPosition, func(references.DungeonCell) references.DungeonCell {
		return 0x61
	})
	require.NoError(t, err)

	token, _ := state.BeginDungeonCommandCycle()
	result := state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	require.True(t, result.WandererMovement.Contact)
	drainDungeonWandererContactWorkflow(t, state)
	state.CombatScene.ExitRoute = CombatExitWest
	state.EndCombat(CombatVictory)

	assert.True(t, state.EventQueueManager.HasWorkflow(dungeonHiddenPitWorkflowName))
	assert.False(t, state.IsAtDungeonCommandBoundary())
	drainDungeonHiddenPitWorkflow(t, state)
	assert.Equal(t, references.FloorNumber(1), state.MapState.PlayerLocation.Floor)
	assert.Equal(t, pitPosition, state.MapState.PlayerLocation.Position)
	source, ok := state.activeDungeon.Cell(0, pitPosition)
	require.True(t, ok)
	assert.Equal(t, byte(0x60), source.Raw())
	assert.True(t, state.IsAtDungeonCommandBoundary())
}

func TestAutomaticDungeonContactVerticalExitResumesOnDestinationFloor(t *testing.T) {
	state, _, _, _ := prepareDungeonWandererContactTest(t, 3)
	if state == nil {
		return
	}
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 5}

	token, _ := state.BeginDungeonCommandCycle()
	result := state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	require.True(t, result.WandererMovement.Contact)
	drainDungeonWandererContactWorkflow(t, state)
	state.CombatScene.ExitRoute = CombatExitUp
	state.EndCombat(CombatVictory)

	assert.Equal(t, references.FloorNumber(2), state.MapState.PlayerLocation.Floor)
	assert.Equal(t, references.DungeonArrivedFromBelow, state.activeDungeon.lastDirection)
	assert.Equal(t, 4, state.ActiveDurationEffect.TurnsRemaining)
	assert.True(t, state.IsAtDungeonCommandBoundary())
}

func TestAutomaticDungeonContactBoundaryExitReleasesCommandBeforeLeaving(t *testing.T) {
	tests := []struct {
		name      string
		floor     references.FloorNumber
		route     CombatExitRoute
		wantFloor references.FloorNumber
	}{
		{name: "up to Britannia", floor: 0, route: CombatExitUp, wantFloor: references.FloorNumber(references.OVERWORLD)},
		{name: "down to Underworld", floor: references.DungeonFloorCount - 1, route: CombatExitDown, wantFloor: references.FloorNumber(references.UNDERWORLD)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, _, _, _ := prepareDungeonWandererContactTest(t, test.floor)
			if state == nil {
				return
			}
			state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 5}

			token, _ := state.BeginDungeonCommandCycle()
			result := state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
			require.True(t, result.WandererMovement.Contact)
			drainDungeonWandererContactWorkflow(t, state)
			state.CombatScene.ExitRoute = test.route

			assert.NotPanics(t, func() { state.EndCombat(CombatVictory) })
			assert.Equal(t, references.BritanniaUnderworld, state.MapState.PlayerLocation.Location)
			assert.Equal(t, test.wantFloor, state.MapState.PlayerLocation.Floor)
			assert.Nil(t, state.activeDungeon)
			assert.Equal(t, 5, state.ActiveDurationEffect.TurnsRemaining)
			assert.True(t, state.IsAtDungeonCommandBoundary())
		})
	}
}

func TestAutomaticDungeonContactDefersFinalEffectsThroughReturnedRoom(t *testing.T) {
	state, _, party, _ := prepareDungeonWandererContactTest(t, 0)
	if state == nil {
		return
	}
	roomPosition, moved := applyDungeonCardinalExit(party, CombatExitEast)
	require.True(t, moved)
	_, err := state.TransformDungeonCell(0, roomPosition, func(references.DungeonCell) references.DungeonCell {
		return 0xa0
	})
	require.NoError(t, err)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 5}

	token, _ := state.BeginDungeonCommandCycle()
	result := state.CompleteDungeonCommandCycle(token, DungeonFullEndPlayer)
	require.True(t, result.WandererMovement.Contact)
	drainDungeonWandererContactWorkflow(t, state)
	state.CombatScene.ExitRoute = CombatExitEast
	state.EndCombat(CombatVictory)

	assert.Equal(t, roomPosition, state.MapState.PlayerLocation.Position)
	assert.Equal(t, 5, state.ActiveDurationEffect.TurnsRemaining)
	assert.True(t, state.HasPendingDungeonRoomEncounter())
	assert.False(t, state.IsAtDungeonCommandBoundary())

	require.NoError(t, state.ResolvePendingDungeonRoomEncounter())
	require.NotNil(t, state.CombatScene)
	state.CombatScene.ExitRoute = CombatExitEast
	state.EndCombat(CombatVictory)

	assert.Equal(t, 4, state.ActiveDurationEffect.TurnsRemaining)
	assert.Nil(t, state.activeDungeon.wanderer, "the completed returned room clears the replacement wanderer")
	assert.True(t, state.IsAtDungeonCommandBoundary())
}

func newDungeonWandererEncounterTestState(
	t *testing.T,
	floor references.FloorNumber,
) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithSystemCallbacks().
		WithRandomSeed(268).
		Build()
	if state == nil {
		return nil, callbacks
	}
	attachDungeonStateForTest(t, state, references.DungeonCells{})
	state.MapState.PlayerLocation.Floor = floor
	callbacks.Reset()

	return state, callbacks
}

func prepareDungeonWandererContactTest(
	t *testing.T,
	floor references.FloorNumber,
) (*GameState, *MockSystemCallbacks, references.Position, references.Position) {
	t.Helper()
	const seed = 3265
	state, callbacks := newDungeonWandererEncounterTestState(t, floor)
	if state == nil {
		return nil, callbacks, references.Position{}, references.Position{}
	}
	attackOrigin := references.Position{X: 3, Y: 3}
	control := beradrng.New(seed)
	direction := dungeonWandererDirection(control.Intn(4))
	party := direction.GetNewPositionInDirection(&attackOrigin).GetWrapped(
		references.XDungeonMapTiles,
		references.YDungeonMapTiles,
	)
	state.MapState.PlayerLocation.Position = *party
	state.activeDungeon.wanderer = &DungeonWanderer{
		Monster: references.MonsterGhost, Floor: floor,
		Position: attackOrigin, PreviousPosition: attackOrigin,
	}
	state.SetRandomSeed(seed)

	return state, callbacks, *party, attackOrigin
}

func drainDungeonWandererContactWorkflow(t *testing.T, state *GameState) {
	t.Helper()
	for range 10 {
		state.EventQueueManager.Update()
		state.ElapsedMs += 100
		if !state.EventQueueManager.HasWorkflow(dungeonWandererContactWorkflowName) {
			return
		}
	}
	require.False(t, state.EventQueueManager.HasWorkflow(dungeonWandererContactWorkflowName))
}
