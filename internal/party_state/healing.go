package party_state

import (
	"fmt"
)

// ProcessPoisonDamage applies poison damage to poisoned active party members.
// Returns slice of character indexes who died from poison
// Each poisoned member loses one HP per player update; ref:PROV-0001.
// Bed rest performs that update every 10-minute cycle; ref:CAMP-0006.
// This means poison damage happens every 10 game minutes, not just hourly like starvation
func (p *PartyState) ProcessPoisonDamage() []int {
	var deadIndexes []int

	for i, char := range p.characters {
		if char == nil || !char.IsAlive() {
			continue
		}

		// Poisoned characters take 1 damage per cycle (10 minutes)
		// Poison status causes one HP loss; ref:PROV-0001.
		if char.IsPoisoned() {
			oldHp := char.GetCurrentHp()
			char.SubtractHp(1)

			// Check for death from poison
			if oldHp > 0 && char.GetCurrentHp() == 0 {
				char.MarkAsDead()
				deadIndexes = append(deadIndexes, i)
			}
		}
	}

	return deadIndexes
}

// RegenerateSleepHealth heals sleeping characters for one 10-minute cycle
// Returns messages for any deaths that occur from poison damage
// NOTE: This is for INN REST, not bed rest. Bed rest does NOT heal in original Ultima V.
func (p *PartyState) RegenerateSleepHealth() []string {
	var messages []string

	for i := range p.characters {
		char := p.characters[i]

		// Nil or dead characters don't heal
		if char == nil || !char.IsAlive() {
			continue
		}

		// Poisoned characters take damage while sleeping instead of healing
		if char.IsPoisoned() {
			oldHp := char.GetCurrentHp()
			char.SubtractHp(1)

			// Check for death from poison
			if oldHp > 0 && char.GetCurrentHp() == 0 {
				char.MarkAsDead()
				messages = append(messages,
					fmt.Sprintf("Alas! %s has died!", char.GetNameAsString()))
			}

			continue
		}

		// Only process sleeping characters for healing
		if !char.IsAsleep() {
			continue
		}

		// Normal healing for sleeping characters: 1 HP per 10 minutes
		if !char.IsFullyHealed() {
			char.AddHp(1)
		}

		// MP restoration: 2 MP per 10 minutes
		if char.GetCurrentMp() < char.GetMaxMp() {
			char.AddMp(2)
		}
	}

	return messages
}

// SetPartySleepStatus toggles sleep status for the entire party
// sleeping=true: Good status characters become Sleep
// sleeping=false: Sleep status characters become Good
func (p *PartyState) SetPartySleepStatus(sleeping bool) {
	for i := range p.characters {
		char := p.characters[i]
		if char == nil {
			continue
		}
		if sleeping && char.IsGood() {
			char.SetStatus(Sleep)
		} else if !sleeping && char.IsAsleep() {
			char.SetStatus(Good)
		}
		// Other statuses (Dead, Poisoned) remain unchanged
	}
}

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

// CountAliveMembers returns the count of party members who are not dead
// Used for food consumption calculations
func (p *PartyState) CountAliveMembers() int {
	count := 0
	for _, player := range p.characters {
		if player != nil && player.IsAlive() {
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
