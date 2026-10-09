package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatSpellMetadataAndGateStayInParity(t *testing.T) {
	combatSpells := references.GetSpellsByContext(references.ContextCombat)
	assert.Len(t, combatSpells, 38)
	for _, spell := range combatSpells {
		assert.Truef(t, IsImplementedCombatSpell(spell), "%s is combat metadata but cannot pass the cast gate", references.GetSpellName(spell))
	}
}

func TestRemainingCombatSpellMetadataMatchesOriginalTargetingAndContexts(t *testing.T) {
	tests := []struct {
		spell     references.Spell
		targeting references.TargetingType
		context   references.SpellContext
	}{
		{references.InSanct, references.TargetAutomatic, references.ContextAll},
		{references.InAn, references.TargetAutomatic, references.ContextAll},
		{references.AnTym, references.TargetAutomatic, references.ContextAll},
		{references.QuasAnWis, references.TargetAutomatic, references.ContextCombat},
		{references.InQuasCorp, references.TargetAutomatic, references.ContextCombat},
		{references.AnXenEx, references.TargetCombatantCrosshair, references.ContextCombat},
		{references.RelXenBet, references.TargetCombatantCrosshair, references.ContextCombat},
		{references.InQuasXen, references.TargetCombatantCrosshair, references.ContextCombat},
	}
	for _, test := range tests {
		info := references.GetSpellInfo(test.spell)
		assert.Equal(t, test.targeting, info.Targeting, info.Name)
		assert.Equal(t, test.context, info.Context, info.Name)
	}
}
