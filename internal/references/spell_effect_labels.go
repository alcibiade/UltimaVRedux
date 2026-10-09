package references

// SpellEffectLabel gives the spell browsers a short, shared plain-language name.
// Full mechanical descriptions remain in SpellInfo.
func SpellEffectLabel(spell Spell) string {
	labels := [...]string{
		"Light", "Energy missile", "Awaken", "Cure poison", "Heal", "Destroy matter",
		"Unlock / disarm", "Repel undead", "Change wind", "Magic vision", "Summon creature", "Create food",
		"Great light", "Fireball", "Fire field", "Poison field", "Sleep field", "Blink",
		"Dispel field", "Protect party", "Protection field", "Ascend", "Descend", "Reveal hidden",
		"Insect swarm", "Lock door", "Unlock magic", "Great healing", "Mass sleep", "Quickness",
		"Earthquake", "Confuse", "Negate magic", "X-ray vision", "Charm", "Transform to rat",
		"Invisibility", "Death", "Clone", "View surroundings", "Poison wind", "Fear",
		"Resurrect", "Summon daemon", "Energy vortex", "Flame wind", "Moongate", "Stop time",
	}
	if spell < 0 || int(spell) >= len(labels) {
		return "Unknown spell"
	}

	return labels[spell]
}
