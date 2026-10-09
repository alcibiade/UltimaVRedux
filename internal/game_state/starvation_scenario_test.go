package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestRunStarvationScenarioIsDeterministicAndPreservesDirectSetupState(t *testing.T) {
	run := func() StarvationScenarioResult {
		state, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()
		state.PartyState.Inventory.Provisions.Food.Set(123)
		state.DateTime.Hour, state.DateTime.Minute = 9, 17
		for slot := range party_state.MaxActiveParty {
			character := state.PartyState.GetCharacter(slot)
			if character != nil && character.IsInParty() {
				character.SetMaxHp(100)
				character.SetHp(100)
			}
		}

		result, err := state.RunStarvationScenario(12345, false)
		require.NoError(t, err)
		return result
	}

	first, second := run(), run()
	assert.Equal(t, first.Characters, second.Characters)
	assert.Equal(t, uint16(123), first.FoodAfter)
	assert.Equal(t, byte(9), first.HourAfter)
	assert.Equal(t, byte(17), first.MinuteAfter)
	for _, character := range first.Characters {
		assert.GreaterOrEqual(t, character.HPBefore-character.HPAfter, uint16(1))
		assert.LessOrEqual(t, character.HPBefore-character.HPAfter, uint16(8))
	}
}

func TestRunStarvationScenarioForcesAuthenticHourlyCheck(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.Provisions.Food.Set(123)
	state.DateTime.Hour, state.DateTime.Minute = 9, 17

	result, err := state.RunStarvationScenario(12345, true)

	require.NoError(t, err)
	assert.Equal(t, uint16(123), result.FoodBefore)
	assert.Equal(t, uint16(0), result.FoodAfter)
	assert.Equal(t, byte(12), result.HourAfter)
	assert.Equal(t, byte(17), result.MinuteAfter)
}

func TestRunStarvationScenarioRejectsCombatWithoutMutation(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	state.MapState.PlayerLocation.Location = references.CombatLocation
	food := state.PartyState.Inventory.Provisions.Food.Get()

	_, err := state.RunStarvationScenario(12345, true)

	require.ErrorIs(t, err, ErrInvalidStarvationScenario)
	assert.Equal(t, food, state.PartyState.Inventory.Provisions.Food.Get())
}
