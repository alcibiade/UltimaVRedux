package mainscreen

import (
	"image/color"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

const characterPossessedLabel = "Possessed"

// CharacterCombatText is a detached text-only view of the original party actor.
// InCombat makes Sleeping authoritative even when another copy has put the shared
// character record to sleep. Portraits continue to use persistent roster status.
type CharacterCombatText struct {
	InCombat  bool
	Possessed bool
	Sleeping  bool
}

// SetCombatText replaces the roster overlay; pass a zero array on combat exit.
// Effective labels and selection-adjusted colours participate in the text cache,
// so applying or clearing an effect needs no stats callback or menu refresh.
func (c *CharacterSummary) SetCombatText(overlay [party_state.MaxCharactersInParty]CharacterCombatText) {
	c.combatText = overlay
}

func (c *CharacterSummary) characterTextStatus(
	index int, character *party_state.PlayerCharacter,
) (string, color.Color) {
	status := character.GetStatus()
	overlay := c.combatText[index]
	if !overlay.InCombat {
		return characterSummaryStatusName(character), c.getStatusColor(status)
	}
	// Main-text precedence: Dead > Possessed/Charmed > Poisoned > Sleep > Good.
	// Secondary tactical conditions remain in the exact-actor HUD.
	if status != party_state.Dead {
		switch {
		case overlay.Possessed:
			return characterPossessedLabel, ucolor.CombatPossessed
		case status == party_state.Charmed, status == party_state.Poisoned:
			// Persistent charm and poison outrank tactical sleep.
		case overlay.Sleeping:
			status = party_state.Sleep
		case status == party_state.Sleep:
			status = party_state.Good
		}
	}

	return characterStatusName(status), c.getStatusColor(status)
}

func (c *CharacterSummary) characterTextColor(
	index int, character *party_state.PlayerCharacter,
	isActivePlayer, hasActivePlayer, isSoloMode, isSoloCharacter bool,
) color.Color {
	if !c.combatText[index].InCombat {
		return c.getEffectiveTextColor(
			character.GetStatus(), isActivePlayer, hasActivePlayer, isSoloMode, isSoloCharacter,
		)
	}
	_, statusColor := c.characterTextStatus(index, character)

	return selectedCharacterTextColor(statusColor, isActivePlayer, hasActivePlayer, isSoloMode, isSoloCharacter)
}
