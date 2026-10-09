package game_state

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestMixReceiptReportsActualStock(t *testing.T) {
	for _, before := range []uint16{0, 98, 99} {
		t.Run(fmt.Sprint(before), func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
			state.PartyState.Inventory.Spells.Set(references.InLor, before)
			state.PartyState.Inventory.Reagent.Set(references.SulfurAsh, 20)
			outcome := state.ExecuteMixing(MixRequest{
				Spell: references.InLor, Quantity: 10,
				ReagentMask: references.GetSpellReagents(references.InLor),
			})
			require.True(t, outcome.Success)
			assert.Equal(t, "Done!", outcome.Message)
			assert.Equal(t, "In Lor", outcome.SpellName)
			assert.Equal(t, 10, outcome.Quantity, "dialog must deduct the full reagent cost")
			assert.Equal(t, min(10, 99-int(before)), outcome.Added)
			assert.Equal(t, min(uint16(99), before+10), outcome.Stock)
			assert.Equal(t, outcome.Stock, state.PartyState.Inventory.Spells.Get(references.InLor))
			assert.Equal(t, uint16(10), state.PartyState.Inventory.Reagent.Get(references.SulfurAsh))
		})
	}
}

func TestMixReceiptUsesCanonicalManiNameWithRealReferences(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	for _, spell := range []references.Spell{references.Mani, references.VasMani} {
		for reagent := references.SulfurAsh; reagent <= references.MandrakeRoot; reagent++ {
			state.PartyState.Inventory.Reagent.Set(reagent, 10)
		}
		outcome := state.ExecuteMixing(MixRequest{
			Spell: spell, Quantity: 1, ReagentMask: references.GetSpellReagents(spell),
		})
		require.True(t, outcome.Success)
		assert.Equal(t, references.GetSpellName(spell), outcome.SpellName)
		assert.NotContains(t, outcome.SpellName, ",")
	}
}

func TestMixFailureRetainsTrapMessageAndConsumesReagents(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	state.SetRandomSeed(42)
	state.PartyState.Inventory.Reagent.Set(references.Garlic, 10)
	before := state.PartyState.Inventory.Spells.Get(references.InLor)
	outcome := state.ExecuteMixing(MixRequest{
		Spell: references.InLor, Quantity: 3, ReagentMask: 1 << uint(references.Garlic),
	})
	require.False(t, outcome.Success)
	require.NotNil(t, outcome.TrapResult)
	assert.Equal(t, outcome.TrapResult.Message, outcome.Message)
	assert.NotContains(t, outcome.Message, "Fizzle")
	assert.Equal(t, uint16(7), state.PartyState.Inventory.Reagent.Get(references.Garlic))
	assert.Equal(t, before, state.PartyState.Inventory.Spells.Get(references.InLor))
}

func TestMixNoReagentsFeedback(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	for reagent := references.SulfurAsh; reagent <= references.MandrakeRoot; reagent++ {
		state.PartyState.Inventory.Reagent.Set(reagent, 0)
	}
	var message string
	state.SystemCallbacks.UI.ShowActionFeedback = func(value string, _ bool) { message = value }
	for _, action := range []func() bool{state.ActionMixSmallMap, state.ActionMixLargeMap, state.ActionMixDungeonMap} {
		message = ""
		assert.False(t, action())
		assert.Equal(t, "No reagents owned!", message)
	}
}

func TestMixInQuasXenUsesSourceFiveReagentRecipe(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).WithSystemCallbacks().Build()
	mask := references.GetSpellReagents(references.InQuasXen)
	for reagent := references.SulfurAsh; reagent <= references.MandrakeRoot; reagent++ {
		state.PartyState.Inventory.Reagent.Set(reagent, 2)
	}

	outcome := state.ExecuteMixing(MixRequest{Spell: references.InQuasXen, Quantity: 1, ReagentMask: mask})

	require.True(t, outcome.Success)
	assert.Equal(t, uint16(2), state.PartyState.Inventory.Reagent.Get(references.NightShade))
	for _, reagent := range []references.Reagent{
		references.SulfurAsh, references.Ginseng, references.SpiderSilk,
		references.BloodMoss, references.MandrakeRoot,
	} {
		assert.Equal(t, uint16(1), state.PartyState.Inventory.Reagent.Get(reagent), reagent)
	}

	withNightshade := mask | 1<<uint(references.NightShade)
	outcome = state.ExecuteMixing(MixRequest{Spell: references.InQuasXen, Quantity: 1, ReagentMask: withNightshade})
	assert.False(t, outcome.Success)
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Reagent.Get(references.NightShade))
}
