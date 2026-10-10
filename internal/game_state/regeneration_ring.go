package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const regenerationRingRollMaximum = 7

// RegenerationRingResult reports one living wearer's roll and resulting HP.
type RegenerationRingResult struct {
	CharacterIndex int
	CharacterName  string
	HPBefore       uint16
	HPAfter        uint16
	RollSucceeded  bool
}

// ProcessRegenerationRingSweep applies check ring in active-party order.
// See ref:EQUIP-0018.
func (g *GameState) ProcessRegenerationRingSweep() []RegenerationRingResult {
	results := make([]RegenerationRingResult, 0, party_state.MaxActiveParty)
	for index := range party_state.MaxActiveParty {
		if !g.PartyState.HasCharacter(index) {
			continue
		}
		character := g.PartyState.GetCharacter(index)
		if !character.IsInParty() || character.IsDead() ||
			character.GetEquippedItem(references.SlotRing) != references.RingRegeneration {
			continue
		}

		before := character.GetCurrentHp()
		status := character.GetStatus()
		succeeded := g.RandomIntInRange(0, regenerationRingRollMaximum) == regenerationRingRollMaximum
		if succeeded {
			character.AddHp(1)
			character.SetStatus(status)
		}
		results = append(results, RegenerationRingResult{
			CharacterIndex: index,
			CharacterName:  character.GetNameAsString(),
			HPBefore:       before,
			HPAfter:        character.GetCurrentHp(),
			RollSucceeded:  succeeded,
		})
	}

	return results
}
