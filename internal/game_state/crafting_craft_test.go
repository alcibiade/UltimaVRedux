package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCraftAnAxeSpendsItsInputsAndItsMinutesBeyondTheTurn(t *testing.T) {
	state, callbacks := newDeedFixture(t, true)
	state.PartyState.Inventory.Materials.Set(references.Wood, 5)
	state.PartyState.Inventory.Materials.Set(references.Stone, 1)

	require.NoError(t, state.Craft(references.RecipeAxe))

	assert.Equal(t, uint16(1), state.PartyState.Inventory.Tools.Get(references.Axe))
	assert.Equal(t, uint16(3), state.PartyState.Inventory.Materials.Get(references.Wood))
	assert.Zero(t, state.PartyState.Inventory.Materials.Get(references.Stone))
	assert.Equal(t, []string{"Crafted 1 Axe."}, callbacks.Messages)
	recipe, _ := references.CraftRecipeByID(references.RecipeAxe)
	assert.Equal(t, recipe.Minutes-DefaultLargeMapMinutesPerTurn, gatherMinutes(callbacks),
		"the scene completes the ordinary two-minute overworld turn")
}

func TestCraftRefusalsSpendNothing(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		prepare func(*GameState)
		want    error
		message string
	}{
		{"lacking inputs", func(s *GameState) { s.PartyState.Inventory.Materials.Set(references.Wood, 1) },
			ErrCraftLacksInputs, "Thou dost need 2 Wood, 2 Stone."},
		{"no room for the output", func(s *GameState) {
			s.PartyState.Inventory.Materials.Set(references.Wood, 2)
			s.PartyState.Inventory.Materials.Set(references.Stone, 2)
			s.PartyState.Inventory.Tools.Set(references.Pick, party_state.ItemQuantitySmallMax)
		}, ErrCraftNoRoomToOutput, "Thou canst carry no more Pick!"},
		{"in the Underworld", func(s *GameState) {
			s.PartyState.Inventory.Materials.Set(references.Wood, 2)
			s.PartyState.Inventory.Materials.Set(references.Stone, 2)
			s.MapState.PlayerLocation.Floor = references.FloorNumber(references.UNDERWORLD)
		}, ErrCraftNotHere, "Not here!"},
	} {
		state, callbacks := newDeedFixture(t, true)
		testCase.prepare(state)
		wood := state.PartyState.Inventory.Materials.Get(references.Wood)

		err := state.Craft(references.RecipePick)

		require.ErrorIs(t, err, testCase.want, testCase.name)
		assert.Equal(t, []string{testCase.message}, callbacks.Messages, testCase.name)
		assert.Equal(t, wood, state.PartyState.Inventory.Materials.Get(references.Wood), testCase.name)
		assert.Empty(t, callbacks.TimeAdvanced(), testCase.name)
	}
}

func TestCraftRecipeOptionsMarkWhatThePartyLacks(t *testing.T) {
	state, _ := newDeedFixture(t, true)
	state.PartyState.Inventory.Materials.Set(references.Wood, 2)
	state.PartyState.Inventory.Materials.Set(references.Stone, 1)

	options := state.CraftRecipeOptions()

	require.Len(t, options, len(references.CraftRecipes()))
	assert.Equal(t, references.RecipeAxe, options[0].Recipe.ID)
	assert.True(t, options[0].Available())
	assert.False(t, options[1].Affordable, "a Pick needs 2 Stone")
	assert.True(t, options[1].StationReady, "tools need no station")
	assert.Equal(t, references.RecipePlanks, options[2].Recipe.ID)
	assert.True(t, options[2].Affordable)
	assert.False(t, options[2].StationReady, "planks need a sawmill, and the overworld has none")
}

func TestCraftWorksOnTheDeedMapWithTheSmallMapTurn(t *testing.T) {
	state, callbacks, _ := newGatherFixture(t)
	state.PartyState.Inventory.Materials.Set(references.Wood, 2)
	state.PartyState.Inventory.Materials.Set(references.Stone, 2)

	require.NoError(t, state.Craft(references.RecipePick))

	recipe, _ := references.CraftRecipeByID(references.RecipePick)
	assert.Equal(t, recipe.Minutes-DefaultSmallMapMinutesPerTurn, gatherMinutes(callbacks))
}

func TestTheCraftMenuStaysOpenOnlyWhileNothingClaimsTheParty(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	assert.True(t, state.CraftMenuMayStayOpen(), "on the quiet deed map")

	require.Positive(t, state.StartDeedRaidForTest())
	assert.False(t, state.CraftMenuMayStayOpen(), "raiders on the deed close it")
	state.dismissDeedRaiders()
	assert.True(t, state.CraftMenuMayStayOpen())

	setCraftingEnabled(state, false)
	assert.False(t, state.CraftMenuMayStayOpen(), "crafting no longer applies")
}
