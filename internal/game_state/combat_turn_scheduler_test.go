package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestChaosInterceptionImmediateAndPresentedResolveIdentically(t *testing.T) {
	for _, alreadyPossessed := range []bool{false, true} {
		for _, condition := range []string{"awake", "sleeping", "underground"} {
			t.Run(fmt.Sprintf("possessed=%t/%s", alreadyPossessed, condition), func(t *testing.T) {
				setup := func() (*GameState, *MockSystemCallbacks) {
					gs, mock := setupCombatStatusTest(t)
					require.NotNil(t, gs)
					actor := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
					for index := range party_state.MaxActiveParty {
						character := gs.PartyState.GetCharacter(index)
						character.SetStatus(party_state.Good)
						character.SetMaxHp(1000)
						character.SetHp(1000)
					}
					gs.CombatScene.SetPartyMemberPosition(0, references.Position{X: 5, Y: 5})
					gs.CombatScene.SetPartyMemberPosition(1, references.Position{X: 6, Y: 5})
					gs.PartyState.GetCharacter(0).SetEquippedItem(references.SlotWeapon, references.ChaosSword)
					gs.CombatScene.setCombatPCPossessed(actor, alreadyPossessed)
					gs.setCombatPCSleeping(actor, condition == "sleeping")
					gs.CombatScene.setCombatPCUnderground(actor, condition == "underground")
					addControlledTestMonster(gs, references.MonsterOrc, references.Position{X: 0, Y: 0}, CombatMonsterLoyaltyMonster)
					require.True(t, gs.CombatScene.AddField(NewCombatField(CombatFieldFire, gs.CombatScene.GetCombatantPosition(actor))))
					gs.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 4}
					forceNextCombatActor(gs.CombatScene.TurnQueue, 0, true)
					gs.CombatScene.TurnQueue.GetEntry(1, true).turnCountdown = 1
					gs.SetRandomSeed(42)
					mock.Reset()
					return gs, mock
				}
				immediate, immediateMock := setup()
				presented, presentedMock := setup()

				immediate.BeginCombatTurns(CombatTurnsImmediate)
				presented.BeginCombatTurns(CombatTurnsPresented)
				drainPresentedCombatToPlayer(t, presented)

				wantDuration := 3
				if alreadyPossessed {
					wantDuration = 4
				}
				assert.Equal(t, wantDuration, immediate.ActiveDurationEffect.TurnsRemaining,
					"the established Redux tail depends on source dispatch, not post-Chaos loyalty")
				assert.Equal(t, immediate.ActiveDurationEffect, presented.ActiveDurationEffect)
				assert.Equal(t, CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1}, immediate.CombatScene.GetCurrentTurnIdentity())
				assert.Equal(t, immediate.CombatScene.GetCurrentTurnIdentity(), presented.CombatScene.GetCurrentTurnIdentity())
				assert.Equal(t, immediate.DateTime, presented.DateTime)
				assert.Equal(t, immediate.CombatScene.PartyPositions, presented.CombatScene.PartyPositions)
				assert.Equal(t, immediate.CombatScene.PartyPossessed, presented.CombatScene.PartyPossessed)
				assert.Equal(t, immediate.CombatScene.PartySleeping, presented.CombatScene.PartySleeping)
				assert.Equal(t, immediate.CombatScene.PartyUnderground, presented.CombatScene.PartyUnderground)
				for index := range party_state.MaxActiveParty {
					assert.Equal(t, *immediate.PartyState.GetCharacter(index), *presented.PartyState.GetCharacter(index))
				}
				assert.Equal(t, immediateMock.CombatEvents, presentedMock.CombatEvents)
				fieldTriggers := 0
				for _, event := range immediateMock.CombatEvents {
					if event.Kind == CombatEventFieldTriggered {
						fieldTriggers++
						assert.Equal(t, 0, event.Target.Index)
					}
				}
				assert.Equal(t, 1, fieldTriggers, "the actor's standing field resolves exactly once")
				assert.Equal(t, immediate.CombatScene.Fields(), presented.CombatScene.Fields())
				assert.Len(t, immediate.CombatScene.Fields(), 1, "undefined player tail must not invent field expiration")
				immediateForecast := make([]CombatTurnForecastEntry, 8)
				presentedForecast := make([]CombatTurnForecastEntry, 8)
				assert.Equal(t, immediate.CombatScene.TurnQueue.FillForecast(immediateForecast), presented.CombatScene.TurnQueue.FillForecast(presentedForecast))
				assert.Equal(t, immediateForecast, presentedForecast)
				assert.Equal(t, immediate.GetRNG().Calls(), presented.GetRNG().Calls())
				assert.Equal(t, immediate.RandomIntInRange(0, 10000), presented.RandomIntInRange(0, 10000))
			})
		}
	}
}

func TestCombatPresentationStepRequiresExactResolutionAndCompletion(t *testing.T) {
	gs, _, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	forceNextMonsterTurnForTest(t, gs, 0)
	positionBefore := monster.Position()

	gs.BeginCombatTurns(CombatTurnsPresented)
	step, ok := gs.ReviewScheduledCombatStep()
	require.True(t, ok)
	assert.Equal(t, CombatPresentationMonsterTurn, step.Kind)
	assert.False(t, step.Actor.IsPartyMember)
	assert.Equal(t, 0, step.Actor.EntityIndex)
	require.True(t, step.HasTarget)
	assert.Equal(t, step.Target, gs.CombatScene.ActiveTargetIdentity)
	assert.True(t, gs.CombatScene.ActiveTargetIdentitySet)
	assert.Equal(t, step.Target.EntityIndex, gs.CombatScene.ActiveTargetPCIndex)
	_, reviewedAgain := gs.ReviewScheduledCombatStep()
	assert.False(t, reviewedAgain, "a scheduled step can be claimed only once")
	assert.Equal(t, positionBefore, monster.Position(), "review must not execute monster AI")

	assert.ErrorIs(t, gs.CompleteScheduledCombatStep(step.Token), ErrCombatPresentationUnresolved)
	_, err := gs.ResolveScheduledCombatStep(step.Token)
	require.NoError(t, err)
	_, err = gs.ResolveScheduledCombatStep(step.Token)
	assert.ErrorIs(t, err, ErrCombatPresentationDuplicate)
	require.NoError(t, gs.CompleteScheduledCombatStep(step.Token))
	assert.ErrorIs(t, gs.CompleteScheduledCombatStep(step.Token), ErrCombatPresentationDuplicate)
}

func TestCombatPresentationCompletionRejectsStaleAndPostCombatTokens(t *testing.T) {
	gs, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	forceNextMonsterTurnForTest(t, gs, 0)

	gs.BeginCombatTurns(CombatTurnsPresented)
	first, ok := gs.ReviewScheduledCombatStep()
	require.True(t, ok)
	_, err := gs.ResolveScheduledCombatStep(first.Token)
	require.NoError(t, err)
	require.NoError(t, gs.CompleteScheduledCombatStep(first.Token))

	completeNextPresentationStep(t, gs)
	pendingToken := nextPendingPresentationToken(t, gs)
	assert.ErrorIs(t, gs.CompleteScheduledCombatStep(first.Token), ErrCombatPresentationStale)
	assert.True(t, gs.CombatScene.HasScheduledCombatStep(), "stale completion must preserve the replacement step")

	gs.TransitionFromCombatMap()
	assert.ErrorIs(t, gs.CompleteScheduledCombatStep(pendingToken), ErrCombatPresentationInactive)

	require.NoError(t, gs.TransitionToCombatMap(
		0, references.BritanniaCombatTerritory, CombatEntryNorth,
	))
	gs.BeginCombatTurns(CombatTurnsPresented)
	replacementToken := gs.FinishCombatTurnDeferred()
	assert.ErrorIs(t, gs.CompleteScheduledCombatStep(pendingToken), ErrCombatPresentationStale)
	assert.True(t, gs.CombatScene.HasScheduledCombatStep(),
		"a prior encounter token must not complete the replacement encounter")
	require.NoError(t, gs.CompleteScheduledCombatStep(replacementToken))
}

func TestPresentedCombatMatchesImmediateDEXTransition(t *testing.T) {
	immediate, _, immediateMonster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 9182)
	presented, _, presentedMonster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 9182)
	if immediate == nil || presented == nil {
		return
	}
	forceNextMonsterTurnForTest(t, immediate, 0)
	forceNextMonsterTurnForTest(t, presented, 0)
	immediate.SetRandomSeed(9182)
	presented.SetRandomSeed(9182)

	immediate.AdvanceCombatTurn()
	presented.BeginCombatTurns(CombatTurnsPresented)
	drainPresentedCombatToPlayer(t, presented)

	assert.Equal(t, immediate.CombatScene.GetCurrentTurnIdentity(), presented.CombatScene.GetCurrentTurnIdentity())
	assert.Equal(t, immediateMonster.Position(), presentedMonster.Position())
	for index := range immediate.PartyState.GetCharacterCount() {
		assert.Equal(t,
			immediate.PartyState.GetCharacter(index).GetCurrentHp(),
			presented.PartyState.GetCharacter(index).GetCurrentHp(),
			"party HP %d", index,
		)
	}
	immediateForecast := make([]CombatTurnForecastEntry, 8)
	presentedForecast := make([]CombatTurnForecastEntry, 8)
	assert.Equal(t,
		immediate.CombatScene.TurnQueue.FillForecast(immediateForecast),
		presented.CombatScene.TurnQueue.FillForecast(presentedForecast),
	)
	assert.Equal(t, immediateForecast, presentedForecast)
	assert.Equal(t, immediate.RandomIntInRange(0, 10000), presented.RandomIntInRange(0, 10000),
		"presentation timing must not consume gameplay RNG")
}

func TestCombatPresentationTokenCannotResolveAnotherGameState(t *testing.T) {
	first, _, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	second, _, secondMonster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if first == nil || second == nil {
		return
	}
	forceNextMonsterTurnForTest(t, first, 0)
	forceNextMonsterTurnForTest(t, second, 0)
	first.BeginCombatTurns(CombatTurnsPresented)
	second.BeginCombatTurns(CombatTurnsPresented)
	firstStep, ok := first.ReviewScheduledCombatStep()
	require.True(t, ok)
	secondStep, ok := second.ReviewScheduledCombatStep()
	require.True(t, ok)
	secondPosition := secondMonster.Position()

	_, err := second.ResolveScheduledCombatStep(firstStep.Token)
	assert.ErrorIs(t, err, ErrCombatPresentationStale)
	assert.Equal(t, secondPosition, secondMonster.Position())
	assert.True(t, second.CombatScene.HasScheduledCombatStep())

	_, err = second.ResolveScheduledCombatStep(secondStep.Token)
	require.NoError(t, err)
	require.NoError(t, second.CompleteScheduledCombatStep(secondStep.Token))
}

func completeNextPresentationStep(t *testing.T, gs *GameState) {
	t.Helper()
	if step, ok := gs.ReviewScheduledCombatStep(); ok {
		if step.Kind == CombatPresentationMonsterTurn {
			result, err := gs.ResolveScheduledCombatStep(step.Token)
			require.NoError(t, err)
			if result.ProjectileLaunch != nil {
				gs.ResolvePendingMonsterMissile()
			}
		}
		require.NoError(t, gs.CompleteScheduledCombatStep(step.Token))

		return
	}

	token := gs.FinishCombatTurnDeferred()
	require.NoError(t, gs.CompleteScheduledCombatStep(token))
}

func nextPendingPresentationToken(t *testing.T, gs *GameState) CombatPresentationToken {
	t.Helper()
	if step, ok := gs.ReviewScheduledCombatStep(); ok {
		return step.Token
	}

	return gs.FinishCombatTurnDeferred()
}

func drainPresentedCombatToPlayer(t *testing.T, gs *GameState) {
	t.Helper()
	for range 64 {
		if !gs.CombatScene.HasScheduledCombatStep() {
			return
		}
		step, ok := gs.ReviewScheduledCombatStep()
		require.True(t, ok)
		if step.Kind == CombatPresentationMonsterTurn {
			result, err := gs.ResolveScheduledCombatStep(step.Token)
			require.NoError(t, err)
			if result.ProjectileLaunch != nil {
				gs.ResolvePendingMonsterMissile()
			}
		}
		require.NoError(t, gs.CompleteScheduledCombatStep(step.Token))
	}
	t.Fatal("presented combat did not reach a player turn")
}
