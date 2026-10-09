package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestSurfaceCommandCycleIsAdditiveToSpecialElapsedTime(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(50, 50).
		WithRandomSeed(387).
		WithSystemCallbacks().
		Build()
	state.DebugOptions.MonsterGen = false
	state.ActiveDurationEffect = DurationEffect{
		EffectType: DurationEffectProtection, TurnsRemaining: 3,
	}
	before := state.DateTime
	callbacks.Reset()

	state.AdvanceGameplayMinutes(ShipRepairCycles * ShipRepairMinutesPerCycle)
	state.CompleteSurfaceCommandCycle(references.LargeMapType)

	want := before
	want.Advance(ShipRepairCycles * ShipRepairMinutesPerCycle)
	want.Advance(DefaultLargeMapMinutesPerTurn)
	assert.Equal(t, want, state.DateTime)
	assert.Equal(t, 2, state.ActiveDurationEffect.TurnsRemaining)
	callbacks.AssertTimeAdvanced(ShipRepairCycles*ShipRepairMinutesPerCycle + DefaultLargeMapMinutesPerTurn)
}

func TestSurfaceMovementCycleMinutesTreatsMovementCostAsTotal(t *testing.T) {
	tests := []struct {
		name         string
		movementCost int
		wantTotal    int
	}{
		{name: "ordinary", movementCost: 2, wantTotal: 2},
		{name: "slow", movementCost: 4, wantTotal: 4},
		{name: "very slow", movementCost: 6, wantTotal: 6},
		{name: "special one-minute movement", movementCost: 1, wantTotal: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithSystemCallbacks().
				Build()
			state.DebugOptions.MonsterGen = false
			callbacks.Reset()
			state.completeSurfaceCycle(references.LargeMapType, test.movementCost)
			callbacks.AssertTimeAdvanced(test.wantTotal)
		})
	}
}

func TestSurfacePlayerUpdateOrdersPoisonAndDurationBeforeRegenerationRing(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	character := state.PartyState.GetAvatar()
	character.SetStatus(party_state.Poisoned)
	character.SetHp(10)
	character.SetEquippedItem(references.SlotRing, references.RingRegeneration)
	state.ActiveDurationEffect = DurationEffect{EffectType: DurationEffectProtection, TurnsRemaining: 2}
	state.SetRandomSeed(regenerationSeedForRolls(t, regenerationRingRollMaximum))

	state.completeSurfaceCycleWithActorPhase(references.SmallMapType, DefaultSmallMapMinutesPerTurn, false)

	assert.Equal(t, uint16(10), character.GetCurrentHp(), "poison damage must precede ring healing")
	assert.Equal(t, 1, state.ActiveDurationEffect.TurnsRemaining)
	assert.Equal(t, uint64(1), state.GetRNG().Calls())
}
