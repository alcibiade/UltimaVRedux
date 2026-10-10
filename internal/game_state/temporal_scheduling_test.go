package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type countingNPCController struct {
	turns int
	units map_units.MapUnits
}

func (c *countingNPCController) PopulateMapFirstLoad()           {}
func (c *countingNPCController) AdvanceNextTurnCalcAndMoveNPCs() { c.turns++ }
func (c *countingNPCController) FreshenExistingNPCsOnMap()       {}
func (c *countingNPCController) RemoveAllEnemies()               { c.units.RemoveAllEnemies() }
func (c *countingNPCController) HardResetNPCs(_ bool)            {}
func (c *countingNPCController) GetNpcs() *map_units.MapUnits    { return &c.units }
func (c *countingNPCController) ClearEnemies() int               { return c.units.RemoveAllEnemies() }

func TestQuicknessAlternatesNonCombatNPCProcessing(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	if gs == nil {
		return
	}
	controller := &countingNPCController{}
	gs.CurrentNPCAIController = controller
	require.True(t, gs.ActivateQuicknessEffect(4))

	for range 4 {
		gs.ProcessTurnEffects()
	}
	assert.Equal(t, 2, controller.turns, "Rel Tym processes the first and third world-actor updates")
	assert.False(t, gs.HasActiveDurationEffect(DurationEffectQuickness))

	gs.ProcessTurnEffects()
	assert.Equal(t, 3, controller.turns, "normal processing resumes immediately after expiry")
}

func TestCombatQuicknessForecastMarksMonsterLoyalActionsUncertainWithoutRNG(t *testing.T) {
	q := NewCombatTurnQueue()
	q.AddCombatant(newMockPC(35), 0)
	q.AddCombatant(newMockMonster(35), 0)
	q.SetTemporalEffectsResolver(func() CombatTemporalEffects {
		return CombatTemporalEffects{
			EffectType:     DurationEffectQuickness,
			TurnsRemaining: 6,
			IsMonsterLoyal: func(identity CombatTurnIdentity) bool { return !identity.IsPartyMember },
		}
	})

	forecast := make([]CombatTurnForecastEntry, 6)
	require.Equal(t, len(forecast), q.FillForecast(forecast))
	assert.Equal(t, []bool{true, true, true, true, true, true}, forecastCanAct(forecast))
	assert.Equal(t, []bool{false, true, false, true, false, true}, forecastActionUncertain(forecast))

	for index, predicted := range forecast {
		actual := q.AdvanceToNextActor()
		require.NotNil(t, actual)
		assert.Equal(t, predicted.CombatTurnIdentity, turnQueueIdentity(actual), "actor %d", index)
		assert.Equal(t, predicted.CanTakeTurn, actual.Combatant.CanTakeTurn(), "actionability %d", index)
	}
}

func TestCombatQuicknessMonsterGateConsumesOneRollWithoutExpiringDuration(t *testing.T) {
	gs, mock, _ := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	require.True(t, gs.ActivateQuicknessEffect(1))
	queue := gs.CombatScene.TurnQueue
	forceNextCombatActor(queue, 0, false)
	gs.CombatScene.presentTurns = true
	var seed uint64
	for candidate := uint64(1); ; candidate++ {
		gs.SetRandomSeed(candidate)
		if gs.RandomIntInRange(0, 1) == 0 {
			seed = candidate

			break
		}
	}
	gs.SetRandomSeed(seed)
	mock.Reset()

	gs.AdvanceCombatTurn()
	step, scheduled := gs.ReviewScheduledCombatStep()
	require.True(t, scheduled)
	_, err := gs.ResolveScheduledCombatStep(step.Token)
	require.NoError(t, err)
	actualNextRoll := gs.RandomIntInRange(0, 10000)

	gs.SetRandomSeed(seed)
	assert.Zero(t, gs.RandomIntInRange(0, 1), "selected seed must suppress this monster-loyal action")
	expectedNextRoll := gs.RandomIntInRange(0, 10000)
	assert.Equal(t, expectedNextRoll, actualNextRoll)
	assert.True(t, gs.HasActiveDurationEffect(DurationEffectQuickness))
	assert.Equal(t, 1, gs.ActiveDurationEffect.TurnsRemaining)
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, CombatConditionQuicknessSuppressed, mock.CombatEvents[0].Condition)
}

func TestAnTymEffectiveDexRespectsLoyaltyAndPCIdentity(t *testing.T) {
	gs, _, hostile := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
	if gs == nil {
		return
	}
	hostileDexterity := hostile.GetDexterity()
	require.True(t, gs.ActivateTimeStopEffect(10))
	hostileIdentity := CombatTurnIdentity{EntityIndex: 0}
	hostileSnapshot, found := gs.CombatScene.TurnQueue.GetEntrySnapshot(hostileIdentity)
	require.True(t, found)
	assert.Equal(t, hostileDexterity, hostileSnapshot.Dexterity)
	assert.False(t, hostileSnapshot.CanTakeTurn)
	assert.Equal(t, hostileDexterity, hostile.GetDexterity(), "An Tym must not mutate permanent/current monster DEX")

	hostile.SetLoyalty(CombatMonsterLoyaltyParty)
	loyalSnapshot, found := gs.CombatScene.TurnQueue.GetEntrySnapshot(hostileIdentity)
	require.True(t, found)
	assert.Equal(t, hostile.GetDexterity(), loyalSnapshot.Dexterity)

	pcIndex := firstActiveCombatPC(gs)
	gs.CombatScene.PartyPossessed[pcIndex] = true
	pcSnapshot, found := gs.CombatScene.TurnQueue.GetEntrySnapshot(CombatTurnIdentity{
		IsPartyMember: true,
		EntityIndex:   pcIndex,
	})
	require.True(t, found)
	assert.Equal(t, int(gs.PartyState.GetCharacter(pcIndex).Dexterity), pcSnapshot.Dexterity,
		"Time Stop must not replace stored PC DEX")
	assert.False(t, pcSnapshot.CanTakeTurn, "possessed PCs are monster-loyal for the Time Stop action gate")
}

func TestAnTymSuppressesMonsterLoyalActionBeforeSleepProcessing(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(*GameState, *CombatMonster) CombatTurnIdentity
		asleep  func(*GameState, *CombatMonster, CombatTurnIdentity) bool
	}{
		{
			name: "hostile monster",
			prepare: func(gs *GameState, monster *CombatMonster) CombatTurnIdentity {
				monster.SetAsleep(true)
				forceNextCombatActor(gs.CombatScene.TurnQueue, 0, false)

				return CombatTurnIdentity{EntityIndex: 0}
			},
			asleep: func(_ *GameState, monster *CombatMonster, _ CombatTurnIdentity) bool {
				return monster.IsAsleep()
			},
		},
		{
			name: "possessed PC",
			prepare: func(gs *GameState, _ *CombatMonster) CombatTurnIdentity {
				identity := CombatTurnIdentity{IsPartyMember: true, EntityIndex: firstActiveCombatPC(gs)}
				gs.CombatScene.PartyPossessed[identity.EntityIndex] = true
				gs.setCombatPCSleeping(identity, true)
				forceNextCombatActor(gs.CombatScene.TurnQueue, identity.EntityIndex, true)

				return identity
			},
			asleep: func(gs *GameState, _ *CombatMonster, identity CombatTurnIdentity) bool {
				return gs.CombatScene.isCombatPCSleeping(identity)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gs, mock, monster := setupCombatWithMonster(t, references.MonsterOrc, references.Down, 42)
			if gs == nil {
				return
			}
			identity := test.prepare(gs, monster)
			require.True(t, gs.ActivateTimeStopEffect(10))
			gs.CombatScene.presentTurns = true
			gs.SetRandomSeed(73)
			expectedNextRoll := gs.RandomIntInRange(0, 10000)
			gs.SetRandomSeed(73)
			mock.Reset()

			gs.AdvanceCombatTurn()
			step, scheduled := gs.ReviewScheduledCombatStep()
			require.True(t, scheduled)
			assert.Equal(t, CombatPresentationMonsterTurn, step.Kind)
			assert.Equal(t, identity, step.Actor)
			assert.False(t, step.HasTarget, "Time Stop must suppress target planning")
			_, err := gs.ResolveScheduledCombatStep(step.Token)
			require.NoError(t, err)

			assert.True(t, test.asleep(gs, monster, identity), "Time Stop must precede wake processing")
			assert.Empty(t, mock.CombatEvents)
			assert.Equal(t, expectedNextRoll, gs.RandomIntInRange(0, 10000),
				"Time Stop suppression must not consume gameplay RNG")
		})
	}
}

func TestAnTymForecastExpiresAtScheduledActionBoundary(t *testing.T) {
	q := NewCombatTurnQueue()
	q.AddCombatant(newMockPC(35), 0)
	monster := newMockMonster(35)
	q.AddCombatant(monster, 0)
	remaining := 1
	q.SetTemporalEffectsResolver(func() CombatTemporalEffects {
		effectType := DurationEffectNone
		if remaining > 0 {
			effectType = DurationEffectTimeStop
		}
		return CombatTemporalEffects{
			EffectType:     effectType,
			TurnsRemaining: remaining,
			IsMonsterLoyal: func(identity CombatTurnIdentity) bool { return !identity.IsPartyMember },
		}
	})

	forecast := make([]CombatTurnForecastEntry, 2)
	require.Equal(t, 2, q.FillForecast(forecast))
	assert.True(t, forecast[0].IsPartyMember)
	assert.False(t, forecast[1].IsPartyMember)
	assert.Equal(t, 35, forecast[1].Dexterity,
		"the effect expires after the first scheduled action, before the hostile reset boundary")

	first := q.AdvanceToNextActor()
	require.True(t, first.Combatant.IsPC())
	remaining--
	second := q.AdvanceToNextActor()
	require.False(t, second.Combatant.IsPC())
	assert.Equal(t, resetTurnCountdown(35), second.turnCountdown)
	assert.Equal(t, 35, monster.GetDexterity())
}

func TestAnTymDoesNotRewriteCountdownWhenHostileJoinsQueue(t *testing.T) {
	q := NewCombatTurnQueue()
	q.SetTemporalEffectsResolver(func() CombatTemporalEffects {
		return CombatTemporalEffects{
			EffectType:     DurationEffectTimeStop,
			TurnsRemaining: 10,
			IsMonsterLoyal: func(identity CombatTurnIdentity) bool { return !identity.IsPartyMember },
		}
	})
	monster := newMockMonster(35)

	q.AddCombatant(monster, 0)

	assert.Equal(t, resetTurnCountdown(35), q.GetEntry(0, false).turnCountdown)
	snapshot, found := q.GetEntrySnapshot(CombatTurnIdentity{EntityIndex: 0})
	require.True(t, found)
	assert.False(t, snapshot.CanTakeTurn)
	assert.Equal(t, 35, monster.GetDexterity(), "queue initialization must not mutate provider DEX")
}

func TestAnTymPreservesStoredDEXOrderingAndForecastsSuppression(t *testing.T) {
	q := NewCombatTurnQueue()
	pc := newMockPC(35)
	hostile := newMockMonster(5)
	q.AddCombatant(pc, 0)
	q.AddCombatant(hostile, 0)
	remaining := 40
	q.SetTemporalEffectsResolver(func() CombatTemporalEffects {
		effectType := DurationEffectNone
		if remaining > 0 {
			effectType = DurationEffectTimeStop
		}
		return CombatTemporalEffects{
			EffectType:     effectType,
			TurnsRemaining: remaining,
			IsMonsterLoyal: func(identity CombatTurnIdentity) bool { return !identity.IsPartyMember },
		}
	})

	hostileIdentity := CombatTurnIdentity{EntityIndex: 0}
	snapshot, found := q.GetEntrySnapshot(hostileIdentity)
	require.True(t, found)
	assert.Equal(t, 5, snapshot.Dexterity)
	assert.False(t, snapshot.CanTakeTurn)
	assert.Equal(t, 5, hostile.GetDexterity(), "Time Stop must not mutate permanent monster DEX")

	forecast := make([]CombatTurnForecastEntry, 40)
	require.Equal(t, len(forecast), q.FillForecast(forecast))
	hostileResolved := false
	for index, predicted := range forecast {
		actual := q.AdvanceToNextActor()
		require.NotNil(t, actual)
		assert.Equal(t, predicted.CombatTurnIdentity, turnQueueIdentity(actual), "actor %d", index)
		assert.Equal(t, predicted.Countdown, actual.turnCountdown, "reset countdown %d", index)
		if !predicted.IsPartyMember {
			hostileResolved = true
			assert.Equal(t, 5, predicted.Dexterity)
			assert.False(t, predicted.CanTakeTurn)
			assert.Equal(t, resetTurnCountdown(5), actual.turnCountdown)
		}
		if predicted.IsPartyMember {
			remaining--
		}
	}
	assert.True(t, hostileResolved, "the hostile still reaches its stored-DEX scheduling boundary")
	assert.Equal(t, 5, hostile.GetDexterity())
}

func forecastCanAct(forecast []CombatTurnForecastEntry) []bool {
	result := make([]bool, len(forecast))
	for index, entry := range forecast {
		result[index] = entry.CanTakeTurn
	}

	return result
}

func forecastActionUncertain(forecast []CombatTurnForecastEntry) []bool {
	result := make([]bool, len(forecast))
	for index, entry := range forecast {
		result[index] = entry.ActionUncertain
	}

	return result
}
