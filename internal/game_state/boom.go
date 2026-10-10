package game_state

import "github.com/bradhannah/UltimaVRedux/internal/party_state"

// TrapType represents the type of trap that can be triggered.
// Used by failed-experiment trap selection for mix failures, chest traps, dungeon traps, etc.
// Original source: original party-effect rules (boom function)
// See ref:MIX-0003.
type TrapType int

const (
	TrapNone   TrapType = iota // No trap triggered
	TrapAcid                   // Acid: damage to single target (1-30)
	TrapPoison                 // Poison: poisons single target
	TrapBomb                   // Bomb: damage to all party (1-8 each)
	TrapGas                    // Gas: poisons entire party
)

// TrapResult contains the outcome of a boom() trap selection.
// Does NOT apply effects - caller uses PartyStateBuilder for damage/VFX.
type TrapResult struct {
	TrapType      TrapType // Which trap was selected
	Message       string   // Display message: "ACID!", "POISON!", "BOMB!", "GAS!"
	TargetIndex   int      // Primary target (e.g., mixer for ACID/POISON)
	TargetIndices []int    // All affected targets (for BOMB/GAS)
	DamageMin     int      // Minimum damage (for ACID/BOMB)
	DamageMax     int      // Maximum damage (for ACID/BOMB)
}

// ExecuteBoom selects a trap and returns targets without applying effects.
// The eight selection entries contain three Acid, two Poison, two Bomb and
// one Gas. Acid damages the mixer with transformed d30; Poison targets the
// mixer; Bomb damages living party members; Gas attempts party poison.
// These multiplicities do not prove original RNG distribution. The caller
// owns damage and status application. ref:MIX-0003
func (g *GameState) ExecuteBoom() *TrapResult {
	// Original mixing rules / original command rules selects the first Good or Poisoned PC.
	// See ref:MIX-0003.
	targetIndex := -1
	g.PartyState.ForEachCharacter(func(index int, character *party_state.PlayerCharacter) {
		if targetIndex == -1 && index < party_state.MaxActiveParty && character.PartyMap == party_state.PartyMapActive &&
			(character.IsGood() || character.IsPoisoned()) {
			targetIndex = index
		}
	})
	if targetIndex == -1 {
		return nil // No valid target
	}

	return g.ExecuteBoomForTarget(targetIndex)
}

// ExecuteBoomForTarget preserves the original selected character for
// single-target acid and poison traps.
func (g *GameState) ExecuteBoomForTarget(targetIndex int) *TrapResult {
	character := g.PartyState.GetCharacter(targetIndex)
	if character == nil || character.PartyMap != party_state.PartyMapActive || character.IsDead() {
		return nil
	}

	// Select among eight trap entries: Acid occupies three, Poison two,
	// Bomb two and Gas one. ref:MIX-0003
	trapTypes := []TrapType{TrapAcid, TrapAcid, TrapAcid, TrapPoison, TrapPoison, TrapBomb, TrapBomb, TrapGas}
	selectedTrap := trapTypes[g.RandomIntInRange(0, len(trapTypes)-1)]

	result := &TrapResult{
		TrapType:    selectedTrap,
		TargetIndex: targetIndex,
	}

	switch selectedTrap {
	case TrapAcid:
		result.Message = "ACID!"
		result.TargetIndices = []int{targetIndex}
		result.DamageMin = 1
		result.DamageMax = 30

	case TrapPoison:
		result.Message = "POISON!"
		result.TargetIndices = []int{targetIndex}

	case TrapBomb:
		result.Message = "BOMB!"
		result.TargetIndices = g.getAllLivingPartyIndices()
		result.DamageMin = 1
		result.DamageMax = 8

	case TrapGas:
		result.Message = "GAS!"
		result.TargetIndices = g.getAllLivingPartyIndices()
	}

	return result
}

// getAllLivingPartyIndices returns indices of all living party members.
// Used by BOMB and GAS traps that affect the entire party.
func (g *GameState) getAllLivingPartyIndices() []int {
	var indices []int
	g.PartyState.ForEachCharacter(func(i int, char *party_state.PlayerCharacter) {
		if i < party_state.MaxActiveParty && char.PartyMap == party_state.PartyMapActive && !char.IsDead() {
			indices = append(indices, i)
		}
	})

	return indices
}
