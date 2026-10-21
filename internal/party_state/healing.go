package party_state

// CountActiveMembers returns the count of party members who are not dead or sleeping
// Used for food consumption calculations
func (p *PartyState) CountActiveMembers() int {
	count := 0
	for _, player := range p.characters {
		if player != nil && player.IsInParty() && !player.IsDead() && !player.IsAsleep() {
			count++
		}
	}

	return count
}

// DamageFromStarvation applies starvation damage to all living party members
// Returns slice of character indexes who died from starvation
// Starvation uses ordered party damage; ref:PROV-0001 and ref:ENV-0001.
// Each living character takes random(1,8) damage
// NOTE: This uses GameState's RNG, caller must provide it via method on GameState
func (p *PartyState) DamageFromStarvation(rng func(min, max int) int) []int {
	var deadIndexes []int

	// Process active living members in roster order; ref:ENV-0001.
	for i := range p.characters {
		character := p.characters[i]

		// Only damage living characters (not dead)
		// Dead or absent members consume no damage draw; ref:ENV-0001.
		if character == nil || !character.IsAlive() {
			continue
		}

		oldHp := character.GetCurrentHp()

		// Each living member receives a separate inclusive 1..8 draw; ref:ENV-0001.
		// Apply random 1-8 starvation damage per character
		damage := rng(1, 8)
		character.SubtractHp(uint16(damage))

		// Check for death from starvation
		if oldHp > 0 && character.GetCurrentHp() == 0 {
			character.MarkAsDead()
			deadIndexes = append(deadIndexes, i)
		}
	}

	return deadIndexes
}

// HasPoisonedCharacter returns true if any party member is currently poisoned
func (p *PartyState) HasPoisonedCharacter() bool {
	for i := range p.characters {
		if p.characters[i] != nil && p.characters[i].IsInParty() && p.characters[i].IsPoisoned() {
			return true
		}
	}

	return false
}

// min helper function for uint16
func min[T ~uint8 | ~uint16 | ~int](a, b T) T {
	if a < b {
		return a
	}

	return b
}
