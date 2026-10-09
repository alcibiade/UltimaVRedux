package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
)

type transientStatusChanges struct {
	healing     bool
	poisonOnset bool
	poisonCure  bool
	levelUp     bool
}

func (changes *transientStatusChanges) observeHealing(
	hpBefore uint16,
	character *party_state.PlayerCharacter,
) {
	changes.healing = changes.healing || character.GetCurrentHp() > hpBefore
}

func (changes *transientStatusChanges) observePoison(
	statusBefore party_state.CharacterStatus,
	character *party_state.PlayerCharacter,
) {
	statusAfter := character.GetStatus()
	changes.poisonOnset = changes.poisonOnset || statusBefore != party_state.Poisoned && statusAfter == party_state.Poisoned
	changes.poisonCure = changes.poisonCure || statusBefore == party_state.Poisoned && statusAfter == party_state.Good
}

func (g *GameState) publishTransientStatusChanges(changes transientStatusChanges) {
	if changes.healing {
		g.SystemCallbacks.UI.ShowTransientIcon(TransientIconHealing)
	}
	if changes.poisonOnset {
		g.SystemCallbacks.UI.ShowTransientIcon(TransientIconPoisonOnset)
	}
	if changes.poisonCure {
		g.SystemCallbacks.UI.ShowTransientIcon(TransientIconPoisonCure)
	}
	if changes.levelUp {
		g.SystemCallbacks.UI.ShowTransientIcon(TransientIconLevelUp)
	}
}
