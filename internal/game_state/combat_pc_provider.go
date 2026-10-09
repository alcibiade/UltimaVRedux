package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
)

// CombatPCProvider wraps a PlayerCharacter to implement CombatantProvider
// for the dex-based turn queue. This adapter lives in game_state (not
// party_state) to avoid cyclic imports.
//
// The wrapper holds a pointer to the PlayerCharacter, so stat changes
// (damage, status effects) are reflected immediately — no data duplication.
type CombatPCProvider struct {
	character   *party_state.PlayerCharacter
	sleeping    *bool
	underground *bool
}

// NewCombatPCProvider creates a CombatantProvider for a party member.
func NewCombatPCProvider(character *party_state.PlayerCharacter) *CombatPCProvider {
	if character == nil {
		panic("NewCombatPCProvider: nil character")
	}

	return &CombatPCProvider{character: character, sleeping: nil, underground: nil}
}

// GetDexterity implements CombatantProvider.
// Returns the PC's dexterity stat for turn ordering.
func (p *CombatPCProvider) GetDexterity() int {
	return int(p.character.Dexterity)
}

// IsAlive implements CombatantProvider.
func (p *CombatPCProvider) IsAlive() bool {
	return p.character.IsAlive()
}

// CanTakeTurn implements CombatantProvider.
// Returns false if the PC is asleep or otherwise unable to act.
func (p *CombatPCProvider) CanTakeTurn() bool {
	if p.underground != nil && *p.underground {
		return false
	}
	if p.sleeping != nil {
		return p.IsAlive() && !*p.sleeping
	}

	return p.IsAlive() && !p.character.IsAsleep()
}

// IsPC implements CombatantProvider. Always returns true for player characters.
func (p *CombatPCProvider) IsPC() bool {
	return true
}

// Character returns the underlying PlayerCharacter for direct access.
func (p *CombatPCProvider) Character() *party_state.PlayerCharacter {
	return p.character
}
