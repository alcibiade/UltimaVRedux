package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const combatInSanctDefenseBonus = 3

func (g *GameState) combatPartyDefense(character *party_state.PlayerCharacter) int {
	defense := 0
	// Doom gates the complete equipment sum on body armour; spell Protection remains independent.
	if g.CombatScene == nil || !g.CombatScene.isDoomInteriorCombat() ||
		character.GetEquippedItem(references.SlotArmour) == references.MysticArmour {
		// REDUX DEVIATION: damage calculation uses persisted armor_class, which
		// Ready/Unready never synchronize with equipment-derived armor. Redux treats
		// that apparent source defect as non-authoritative and uses current equipment.
		// See ref:EQUIP-0014.
		defense = character.GetTotalDefense()
	}
	if g.HasActiveDurationEffect(DurationEffectProtection) {
		defense += combatInSanctDefenseBonus
	}

	return defense
}
