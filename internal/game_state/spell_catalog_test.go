package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/require"
)

func TestSpellCatalogKnownRecipesAndInventoryLimits(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	state.PartyState.Inventory.Reagent.Set(references.Ginseng, 6)
	state.PartyState.Inventory.Reagent.Set(references.Garlic, 8)
	catalog := state.CurrentSpellCatalog()
	var cure *ZStatsSpell
	for index := range catalog {
		if catalog[index].ID == references.AnNox.ID() {
			cure = &catalog[index]
		}
	}
	require.NotNil(t, cure)
	require.Equal(t, 6, cure.CanMix)
	require.Equal(t, references.GetSpellReagents(references.AnNox), cure.RecipeMask)
	require.Len(t, cure.RecipeInventory, 2)
	cure.RecipeInventory[0].Quantity = 0
	state.PartyState.Inventory.Reagent.Set(references.Ginseng, 0)
	for _, spell := range state.CurrentSpellCatalog() {
		if spell.ID == references.AnNox.ID() {
			require.Zero(t, spell.CanMix)
			require.Contains(t, spell.MixMissing, "Ginseng")
		}
	}
}

func TestMixRejectsInvalidQuantityAndStaleInventoryBeforeMutation(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	state.PartyState.Inventory.Reagent.Set(references.SulfurAsh, 6)
	before := state.PartyState.Inventory.Spells.Get(references.InLor)
	for _, quantity := range []int{-1, 0, 7, 100, 65536} {
		outcome := state.ExecuteMixing(MixRequest{Spell: references.InLor,
			Quantity: quantity, ReagentMask: references.GetSpellReagents(references.InLor)})
		require.False(t, outcome.Success)
		require.Nil(t, outcome.TrapResult)
		require.Equal(t, uint16(6), state.PartyState.Inventory.Reagent.Get(references.SulfurAsh))
		require.Equal(t, before, state.PartyState.Inventory.Spells.Get(references.InLor))
	}
}
