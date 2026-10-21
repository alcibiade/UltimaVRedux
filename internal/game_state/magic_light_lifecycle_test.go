package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func newMagicLightLifecycleState(t *testing.T) *GameState {
	t.Helper()
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()
	state.SystemCallbacks.UI.ShowActionFeedback = func(string, bool) {}

	return state
}

func TestMagicLightAuthenticDurations(t *testing.T) {
	state := newMagicLightLifecycleState(t)

	state.ActivateLightEffect(InLorLightDuration)
	assert.Equal(t, 100, state.MagicLightMinutesRemaining)
	state.ActivateLightEffect(VasLorLightDuration)
	assert.Equal(t, 255, state.MagicLightMinutesRemaining)
	state.AdvanceGameplayMinutes(1)
	assert.Equal(t, 254, state.MagicLightMinutesRemaining, "255 is not permanent for magical light")

	scrollDuration := state.GameReferences.InventoryItemReferences.GetScrollEffectDuration(references.ScrollVasLor)
	require.Equal(t, 240, scrollDuration)
	state.ActivateLightEffect(scrollDuration)
	assert.Equal(t, 240, state.MagicLightMinutesRemaining)
}

func TestMagicLightAppearsInIndependentPersistentEffectStatus(t *testing.T) {
	state := newMagicLightLifecycleState(t)

	state.ActivateLightEffect(1)
	effects := state.GetPersistentEffectStatuses()
	require.Len(t, effects, 1)
	assert.Equal(t, PersistentEffectMagicLight, effects[0].ID)

	state.AdvanceGameplayMinutes(1)
	assert.Empty(t, state.GetPersistentEffectStatuses())
}

func TestMagicLightCoexistsWithAndDoesNotReplaceActiveSpell(t *testing.T) {
	tests := []struct {
		name       string
		effectType DurationEffectType
		activate   func(*GameState)
	}{
		{name: "Protection", effectType: DurationEffectProtection, activate: func(state *GameState) { state.ActivateProtectionEffect(20) }},
		{name: "Quickness", effectType: DurationEffectQuickness, activate: func(state *GameState) { state.ActivateQuicknessEffect(20) }},
		{name: "NegateMagic", effectType: DurationEffectNegateMagic, activate: func(state *GameState) { state.ActivateNegateMagicEffect(20) }},
		{name: "Confusion", effectType: DurationEffectConfusion, activate: func(state *GameState) { state.ActivateConfusionEffect(20) }},
		{name: "TimeStop", effectType: DurationEffectTimeStop, activate: func(state *GameState) { state.ActivateTimeStopEffect(20) }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := newMagicLightLifecycleState(t)
			test.activate(state)
			state.ActivateLightEffect(100)

			assert.True(t, state.HasActiveDurationEffect(test.effectType))
			assert.Equal(t, 100, state.MagicLightMinutesRemaining)
			assert.True(t, state.MapState.Lighting.HasMagicLight())

			state.ActivateLightEffect(255)
			assert.True(t, state.HasActiveDurationEffect(test.effectType))
			assert.Equal(t, 255, state.MagicLightMinutesRemaining)
		})
	}

	state := newMagicLightLifecycleState(t)
	state.ActivateProtectionEffect(20)
	state.ActivateLightEffect(100)
	state.ActivateNegateMagicEffect(20)
	assert.True(t, state.HasActiveDurationEffect(DurationEffectNegateMagic))
	assert.Equal(t, 100, state.MagicLightMinutesRemaining, "active_spell replacement must not replace magic_light")
}

func TestAdvanceGameplayMinutesUsesActualElapsedMinutes(t *testing.T) {
	state := newMagicLightLifecycleState(t)
	state.MapState.Lighting.LightTorch()
	state.ActivateLightEffect(20)
	minuteBefore := state.DateTime.Minute

	elapsed := state.AdvanceGameplayMinutes(7)

	assert.Equal(t, 7, elapsed)
	assert.Equal(t, byte((int(minuteBefore)+7)%60), state.DateTime.Minute)
	assert.Equal(t, 13, state.MagicLightMinutesRemaining)
	assert.Equal(t, map_state.DefaultTorchDurationMinutes-7, state.MapState.Lighting.TorchMinutesRemaining())
}

func TestAdvanceGameplayMinutesAppliesQuicknessAndMinimumOne(t *testing.T) {
	state := newMagicLightLifecycleState(t)
	state.ActivateLightEffect(20)
	state.ActivateQuicknessEffect(20)

	assert.Equal(t, 2, state.AdvanceGameplayMinutes(5))
	assert.Equal(t, 18, state.MagicLightMinutesRemaining)
	assert.Equal(t, 1, state.AdvanceGameplayMinutes(1))
	assert.Equal(t, 17, state.MagicLightMinutesRemaining)
}

func TestAdvanceGameplayMinutesTimeStopFreezesCalendarAndLight(t *testing.T) {
	state := newMagicLightLifecycleState(t)
	state.MapState.Lighting.LightTorch()
	state.ActivateLightEffect(20)
	state.ActivateTimeStopEffect(1)
	minuteBefore := state.DateTime.Minute

	assert.Equal(t, 0, state.AdvanceGameplayMinutes(10))
	assert.Equal(t, minuteBefore, state.DateTime.Minute)
	assert.Equal(t, 20, state.MagicLightMinutesRemaining)
	assert.Equal(t, map_state.DefaultTorchDurationMinutes, state.MapState.Lighting.TorchMinutesRemaining())

	state.FinishTurn()
	assert.False(t, state.HasActiveDurationEffect(DurationEffectTimeStop))
	assert.Equal(t, minuteBefore, state.DateTime.Minute, "Time Stop's final action must remain frozen")
	assert.Equal(t, 20, state.MagicLightMinutesRemaining)
	assert.Equal(t, map_state.DefaultTorchDurationMinutes, state.MapState.Lighting.TorchMinutesRemaining())
}

func TestCombatCalendarAdvanceConsumesMagicLightOnlyWhenMinuteAdvances(t *testing.T) {
	state := newMagicLightLifecycleState(t)
	state.ActivateLightEffect(3)
	state.CombatScene = &CombatState{TurnQueue: NewCombatTurnQueue()}

	state.advanceCombatQueueTime()
	assert.Equal(t, 3, state.MagicLightMinutesRemaining)

	state.CombatScene.TurnQueue.actionCount = actionsPerTimeAdvance
	state.advanceCombatQueueTime()
	assert.Equal(t, 2, state.MagicLightMinutesRemaining)
}

func TestDeathCleanupClearsActiveSpellAndIndependentLight(t *testing.T) {
	state := newMagicLightLifecycleState(t)
	state.ActivateProtectionEffect(20)
	state.ActivateLightEffect(100)
	state.MapState.Lighting.LightTorch()

	state.applyFullPartyDeathConsequences()

	assert.False(t, state.HasAnyDurationEffect())
	assert.Zero(t, state.MagicLightMinutesRemaining)
	assert.False(t, state.MapState.Lighting.HasAnyLight())
}
