package references

// Test support: fixtures and seams that tests in other packages use. The
// dead-code check treats this file as test support (workshop deadcode config).

// GetListOfAllSmallMaps names every small-map location, Moonglow to Serpent's Hold.
func GetListOfAllSmallMaps() []string {
	names := make([]string, 0)

	for loc := Moonglow; loc <= SerpentsHold; loc++ {
		names = append(names, loc.String())
	}

	return names
}

// MonsterSpawnPositions are 16 predefined positions for placing monsters on the 11x11 combat map.
// Original encounter-placement coordinates; ref:COMBAT-0001.
// Encounter generation uses these positions independently of creature identities.
//
//nolint:gochecknoglobals, mnd // Immutable encounter-placement coordinates; ref:COMBAT-0001.
var MonsterSpawnPositions = [16]Position{
	{X: 2, Y: 0},
	{X: 5, Y: 0},
	{X: 8, Y: 0},
	{X: 4, Y: 1},
	{X: 6, Y: 1},
	{X: 1, Y: 2},
	{X: 5, Y: 2},
	{X: 9, Y: 2},
	{X: 3, Y: 3},
	{X: 7, Y: 3},
	{X: 0, Y: 4},
	{X: 4, Y: 4},
	{X: 6, Y: 4},
	{X: 10, Y: 4},
	{X: 2, Y: 5},
	{X: 8, Y: 5},
}

// GetAllMonsterSpawnPositions returns a copy of all spawn positions.
// Useful for shuffling positions for random monster placement.
func GetAllMonsterSpawnPositions() []Position {
	positions := make([]Position, len(MonsterSpawnPositions))
	copy(positions, MonsterSpawnPositions[:])

	return positions
}

// ShuffleMonsterSpawnPositions returns shuffled spawn positions.
// Randomizes the placement order without changing the coordinate set.
func ShuffleMonsterSpawnPositions(rng RandomProvider) []Position {
	positions := GetAllMonsterSpawnPositions()

	// Fisher-Yates shuffle
	for i := len(positions) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		positions[i], positions[j] = positions[j], positions[i]
	}

	return positions
}

// IsNormalChar returns true if the byte represents a normal (IBM) character (high bit set).
func IsNormalChar(b byte) bool {
	return b&SignHighBitMask != 0
}

// GetSpellsByContext returns all spells valid in the given context.
func GetSpellsByContext(context SpellContext) []Spell {
	var spells []Spell
	for i := range TotalSpellCount {
		if spellMetadata[i].Context.HasContext(context) {
			spells = append(spells, Spell(i))
		}
	}

	return spells
}

// IsSpellValidInContext returns true if the spell can be cast in the given context.
func IsSpellValidInContext(spell Spell, context SpellContext) bool {
	info := GetSpellInfo(spell)
	if info == nil {
		return false
	}

	return info.Context.HasContext(context)
}
