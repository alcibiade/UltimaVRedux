package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestSetPartyKarmaUsesExactValidatedValue(t *testing.T) {
	gameState, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	gameState.PartyState.Karma.Value = 50
	refreshes := 0
	gameState.SystemCallbacks.Screen.RefreshInventoryDisplay = func() { refreshes++ }

	result, err := gameState.SetPartyKarma(99)
	require.NoError(t, err)
	assert.Equal(t, PartyKarmaAlteration{Before: 50, After: 99}, result)
	assert.Equal(t, 99, gameState.PartyState.Karma.Value)
	assert.Equal(t, 1, refreshes)
	assert.Empty(t, callbacks.TransientIconCalls)

	result, err = gameState.SetPartyKarma(99)
	require.NoError(t, err)
	assert.Equal(t, PartyKarmaAlteration{Before: 99, After: 99}, result)
	assert.Equal(t, 1, refreshes)
}

func TestAdjustPartyKarmaPublishesOnlyActualClampedChanges(t *testing.T) {
	tests := []struct {
		name      string
		before    int
		delta     int
		after     int
		wantIcons []TransientIconKind
	}{
		{name: "increase", before: 50, delta: 3, after: 53, wantIcons: []TransientIconKind{TransientIconKarmaIncrease}},
		{name: "decrease", before: 50, delta: -5, after: 45, wantIcons: []TransientIconKind{TransientIconKarmaDecrease}},
		{name: "clamped increase", before: 98, delta: 3, after: 99, wantIcons: []TransientIconKind{TransientIconKarmaIncrease}},
		{name: "clamped decrease", before: 1, delta: -5, after: 0, wantIcons: []TransientIconKind{TransientIconKarmaDecrease}},
		{name: "upper no-op", before: 99, delta: 1, after: 99},
		{name: "lower no-op", before: 0, delta: -1, after: 0},
		{name: "zero no-op", before: 50, delta: 0, after: 50},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gameState, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.Britain).
				WithSystemCallbacks().
				Build()
			gameState.PartyState.Karma.Value = test.before

			result := gameState.AdjustPartyKarma(test.delta)

			assert.Equal(t, PartyKarmaAlteration{Before: test.before, After: test.after}, result)
			assert.Equal(t, test.after, gameState.PartyState.Karma.Value)
			assert.Equal(t, test.wantIcons, callbacks.TransientIconCalls)
		})
	}
}

func TestSetPartyKarmaRejectsOutOfRangeWithoutMutation(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	gameState.PartyState.Karma.Value = 50

	for _, invalid := range []int{-1, MaxKarmaValue + 1} {
		result, err := gameState.SetPartyKarma(invalid)
		require.ErrorIs(t, err, ErrInvalidPartyKarma)
		assert.Equal(t, PartyKarmaAlteration{Before: 50, After: 50}, result)
		assert.Equal(t, 50, gameState.PartyState.Karma.Value)
	}
}

func TestCommitScriptedGoldPaymentUsesOriginalBeggarCooldown(t *testing.T) {
	tests := []struct {
		name            string
		gold            uint16
		counter         int
		amount          int
		isBeggar        bool
		wantGold        uint16
		wantKarma       int
		startingKarma   int
		wantAward       int
		wantCounter     int
		wantKarmaSignal bool
	}{
		{name: "cooldown not ready", gold: 50, counter: 99, amount: 50, isBeggar: true, wantGold: 0, wantKarma: 50, wantCounter: 99},
		{name: "qualifying payment", gold: 50, counter: 100, amount: 1, isBeggar: true, wantGold: 49, wantKarma: 51, wantAward: 1, wantCounter: 0, wantKarmaSignal: true},
		{name: "all gold bonus", gold: 50, counter: 100, amount: 50, isBeggar: true, wantGold: 0, wantKarma: 53, wantAward: 3, wantCounter: 0, wantKarmaSignal: true},
		{name: "zero gold script payment", gold: 0, counter: 100, amount: 0, isBeggar: true, wantGold: 0, wantKarma: 53, wantAward: 3, wantCounter: 0, wantKarmaSignal: true},
		{name: "maximum karma reports no award", gold: 1, counter: 100, amount: 1, isBeggar: true, startingKarma: MaxKarmaValue, wantGold: 0, wantKarma: MaxKarmaValue, wantCounter: 0},
		{name: "non beggar does not consume cooldown", gold: 50, counter: 100, amount: 50, wantGold: 0, wantKarma: 50, wantCounter: 100},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.Britain).
				WithSystemCallbacks().
				Build()
			state.PartyState.Inventory.Gold.Set(test.gold)
			startingKarma := test.startingKarma
			if startingKarma == 0 {
				startingKarma = 50
			}
			state.PartyState.Karma.Value = startingKarma
			state.beggarKarmaCounter = test.counter

			result, err := state.CommitScriptedGoldPayment(test.amount, test.isBeggar)

			require.NoError(t, err)
			assert.Equal(t, test.gold, result.GoldBefore)
			assert.Equal(t, test.wantGold, result.GoldAfter)
			assert.Equal(t, startingKarma, result.KarmaBefore)
			assert.Equal(t, test.wantKarma, result.KarmaAfter)
			assert.Equal(t, test.wantAward, result.KarmaAward)
			assert.Equal(t, test.wantGold, state.PartyState.Inventory.Gold.Get())
			assert.Equal(t, test.wantKarma, state.PartyState.Karma.Value)
			assert.Equal(t, test.wantCounter, state.beggarKarmaCounter)
			assert.Equal(t, test.wantKarmaSignal, len(callbacks.TransientIconCalls) == 1)
		})
	}
}

func TestCommitScriptedGoldPaymentFailureDoesNotConsumeBeggarCooldown(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.Gold.Set(49)
	state.PartyState.Karma.Value = 50
	state.beggarKarmaCounter = 100

	for _, amount := range []int{50, -1, int(^uint16(0)) + 1} {
		result, err := state.CommitScriptedGoldPayment(amount, true)

		require.ErrorIs(t, err, ErrInvalidScriptedGoldPayment)
		assert.Equal(t, ScriptedGoldPaymentOutcome{
			GoldBefore: 49, GoldAfter: 49, KarmaBefore: 50, KarmaAfter: 50,
		}, result)
		assert.Equal(t, uint16(49), state.PartyState.Inventory.Gold.Get())
		assert.Equal(t, 50, state.PartyState.Karma.Value)
		assert.Equal(t, 100, state.beggarKarmaCounter)
		assert.Empty(t, callbacks.TransientIconCalls)
	}
}
