package references

// FlameOfPrinciple represents one of the three Flames of Principle in Ultima V.
// These are the eternal flames that embody the three Principles of Virtue.
type FlameOfPrinciple int

const (
	// FlameOfTruth is located in Lycaeum.
	// Use the Shard of Falsehood here to destroy Faulinei.
	FlameOfTruth FlameOfPrinciple = 0

	// FlameOfLove is located in Empath Abbey.
	// Use the Shard of Hatred here to destroy Astaroth.
	FlameOfLove FlameOfPrinciple = 1

	// FlameOfCourage is located in Serpent's Hold.
	// Use the Shard of Cowardice here to destroy Nosfentor.
	FlameOfCourage FlameOfPrinciple = 2

	// FlameCount is the total number of Flames of Principle.
	FlameCount = 3
)

// String returns the name of the Flame.
func (f FlameOfPrinciple) String() string {
	switch f {
	case FlameOfTruth:
		return "Truth"
	case FlameOfLove:
		return "Love"
	case FlameOfCourage:
		return "Courage"
	default:
		return "Unknown"
	}
}

// GetPrinciple returns the full principle name.
func (f FlameOfPrinciple) GetPrinciple() string {
	return "Flame of " + f.String()
}

// GetShadowlord returns the Shadowlord that can be defeated at this Flame.
func (f FlameOfPrinciple) GetShadowlord() Shadowlord {
	return Shadowlord(f)
}

// GetRequiredShard returns the QuestItem shard required at this Flame.
func (f FlameOfPrinciple) GetRequiredShard() QuestItem {
	switch f {
	case FlameOfTruth:
		return ShardOfFalsehood
	case FlameOfLove:
		return ShardOfHatred
	case FlameOfCourage:
		return ShardOfCowardice
	default:
		return ShardOfFalsehood
	}
}

// FlameLocation contains the exact position of a Flame of Principle.
// Ritual locations follow ref:SHADOW-0005.
type FlameLocation struct {
	Location Location
	X        int
	Y        int
	Floor    int // -1 represents basement (0xFF in original)
}

// FlameLocations maps each Flame to its exact position in the world.
// Truth: Lycaeum (15,9), floor 2; Love: Empath Abbey (15,3), floor 1;
// Courage: Serpent's Hold (15,16), basement. See ref:SHADOW-0005.
//
//nolint:gochecknoglobals // Fixed original-game reference data, read-only after init.
var FlameLocations = map[FlameOfPrinciple]FlameLocation{
	FlameOfTruth: {
		Location: Lycaeum,
		X:        15,
		Y:        9,
		Floor:    2,
	},
	FlameOfLove: {
		Location: EmpathAbbey,
		X:        15,
		Y:        3,
		Floor:    1,
	},
	FlameOfCourage: {
		Location: SerpentsHold,
		X:        15,
		Y:        16,
		Floor:    -1, // Basement (0xFF in original)
	},
}

// GetFlameAtLocation returns the FlameOfPrinciple for a given location, or -1 if none.
func GetFlameAtLocation(loc Location) FlameOfPrinciple {
	for flame, flameLocation := range FlameLocations {
		if flameLocation.Location == loc {
			return flame
		}
	}

	return -1
}

// IsKeepLocation returns true if the location is one of the three Keeps
// where Shadowlords can be summoned.
func IsKeepLocation(loc Location) bool {
	return loc == Lycaeum || loc == EmpathAbbey || loc == SerpentsHold
}

// IsAtFlamePosition checks if the given position matches the exact Flame coordinates.
func IsAtFlamePosition(loc Location, x, y, floor int) (FlameOfPrinciple, bool) {
	for flame, flameLocation := range FlameLocations {
		if flameLocation.Location == loc &&
			flameLocation.X == x &&
			flameLocation.Y == y &&
			flameLocation.Floor == floor {
			return flame, true
		}
	}

	return -1, false
}

// IsStonegateLocation returns true if the location is Stonegate.
// Stonegate has unique mechanics: all 3 Shadowlords present, magic always absorbed.
func IsStonegateLocation(loc Location) bool {
	return loc == Stonegate
}

// IsMagicBlockedLocation returns true if magic is always absorbed at this location.
// Original prepared-cast absorption gates (ref:SPELL-0001):
//   - Stonegate (0x1d): Magic ALWAYS absorbed, no bypass
//   - Palace of Blackthorn (0x12): Magic absorbed UNLESS player has Crown
//
// This function only checks Stonegate since it has no bypass condition.
// Palace of Blackthorn requires separate Crown check.
func IsMagicBlockedLocation(loc Location) bool {
	return loc == Stonegate
}
