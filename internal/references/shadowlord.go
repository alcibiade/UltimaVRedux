package references

// Shadowlord represents one of the three Shadowlords in Ultima V.
// Each Shadowlord is associated with a Shard of Evil and an anti-virtue.
type Shadowlord int

const (
	// NoShadowlord indicates no Shadowlord is present at a location.
	// Used as the return value when no Shadowlord is assigned.
	NoShadowlord Shadowlord = -1

	// Faulinei is the Shadowlord of Falsehood (anti-Truth).
	// Defeated using Shard of Falsehood at the Flame of Truth in Lycaeum.
	Faulinei Shadowlord = 0

	// Astaroth is the Shadowlord of Hatred (anti-Love).
	// Defeated using Shard of Hatred at the Flame of Love in Empath Abbey.
	Astaroth Shadowlord = 1

	// Nosfentor is the Shadowlord of Cowardice (anti-Courage).
	// Defeated using Shard of Cowardice at the Flame of Courage in Serpent's Hold.
	Nosfentor Shadowlord = 2

	// ShadowlordCount is the total number of Shadowlords.
	ShadowlordCount = 3
)

// String returns the name of the Shadowlord.
func (s Shadowlord) String() string {
	switch s {
	case Faulinei:
		return "Faulinei"
	case Astaroth:
		return "Astaroth"
	case Nosfentor:
		return "Nosfentor"
	case NoShadowlord:
	}

	return "Unknown"
}

// GetAntiVirtue returns the anti-virtue this Shadowlord represents.
func (s Shadowlord) GetAntiVirtue() string {
	switch s {
	case Faulinei:
		return "Falsehood"
	case Astaroth:
		return "Hatred"
	case Nosfentor:
		return "Cowardice"
	case NoShadowlord:
	}

	return "Unknown"
}

// GetPresenceMessage returns the atmospheric message displayed when entering a town
// where this Shadowlord is present; ref:SHADOW-0001.
func (s Shadowlord) GetPresenceMessage() string {
	switch s {
	case Faulinei:
		return "An air of falsehood\ndoth surround thee..."
	case Astaroth:
		return "An air of hatred\ndoth surround thee..."
	case Nosfentor:
		return "An air of cowardice\ndoth surround thee..."
	case NoShadowlord:
	}

	return "An air of darkness\ndoth surround thee..."
}

// ShadowlordFromName returns a Shadowlord by name (case insensitive substring match).
// Returns -1 if no match found.
func ShadowlordFromName(name string) Shadowlord {
	upperName := toUpperASCII(name)
	if containsSubstring(upperName, "FAULINEI") {
		return Faulinei
	}
	if containsSubstring(upperName, "ASTAROTH") {
		return Astaroth
	}
	if containsSubstring(upperName, "NOSFENTOR") {
		return Nosfentor
	}

	return -1
}

// toUpperASCII converts a string to uppercase (ASCII only).
func toUpperASCII(s string) string {
	result := make([]byte, len(s))
	for i := range len(s) {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		result[i] = c
	}

	return string(result)
}

// containsSubstring checks if haystack contains needle.
func containsSubstring(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}

	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}

	return false
}

// ShadowlordState tracks the state of all three Shadowlords.
type ShadowlordState struct {
	// Alive indicates whether each Shadowlord is still alive (not yet permanently defeated).
	Alive [ShadowlordCount]bool

	// LastCalled tracks which Shadowlord was most recently summoned via yell.
	// This is used for shard validation - the shard must match the summoned Shadowlord.
	LastCalled Shadowlord

	// Summoned indicates whether a Shadowlord is currently summoned at a Keep via yell.
	// Only one Shadowlord can be summoned at a time.
	// Note: Yell summoning does NOT affect NPCs - it's for combat (destroy with Shard).
	Summoned bool

	// TownAssignments stores the persistent town assigned to each living Shadowlord.
	// Zero means unassigned in the original new-game Shadowlord destinations; ref:SHADOW-0001.
	TownAssignments [ShadowlordCount]Location

	// AutomaticPresence tracks which Shadowlord is automatically present on the loaded map.
	// Set to NoShadowlord (-1) when no automatic presence, or 0-2 for the present Shadowlord.
	// Unlike yell summoning, automatic presence DOES affect NPCs and should persist
	// across floor changes within the same location.
	// Original town entry establishes presence before applying NPC influence; ref:SHADOW-0002 and ref:SHADOW-0003.
	//
	AutomaticPresence Shadowlord
}

// NewShadowlordState creates a new ShadowlordState with all Shadowlords alive.
func NewShadowlordState() *ShadowlordState {
	return &ShadowlordState{
		Alive:             [ShadowlordCount]bool{true, true, true},
		TownAssignments:   [ShadowlordCount]Location{},
		LastCalled:        NoShadowlord,
		Summoned:          false,
		AutomaticPresence: NoShadowlord, // No automatic presence initially
	}
}

// IsAlive returns true if the specified Shadowlord is still alive.
func (s *ShadowlordState) IsAlive(lord Shadowlord) bool {
	if lord < 0 || lord >= ShadowlordCount {
		return false
	}

	return s.Alive[lord]
}

// Kill marks a Shadowlord as permanently defeated.
func (s *ShadowlordState) Kill(lord Shadowlord) {
	if lord >= 0 && lord < ShadowlordCount {
		s.Alive[lord] = false
		s.TownAssignments[lord] = 0
	}
	// Clear summoned state if this was the summoned Shadowlord
	if s.LastCalled == lord {
		s.Summoned = false
		s.LastCalled = NoShadowlord
	}
}

// Summon marks a Shadowlord as summoned at a Keep.
func (s *ShadowlordState) Summon(lord Shadowlord) bool {
	if lord < 0 || lord >= ShadowlordCount {
		return false
	}

	if !s.Alive[lord] {
		return false
	}

	if s.Summoned {
		return false
	}

	s.Summoned = true
	s.LastCalled = lord

	return true
}

// ClearSummoned clears the yell-summoned state.
func (s *ShadowlordState) ClearSummoned() {
	s.Summoned = false
	s.LastCalled = NoShadowlord
}

// ClearAutomaticPresence clears the automatic (moon-based) Shadowlord presence.
// This should be called when leaving a town/small map entirely.
func (s *ShadowlordState) ClearAutomaticPresence() {
	s.AutomaticPresence = NoShadowlord
}

// HasAutomaticPresence returns true if a Shadowlord is automatically present.
func (s *ShadowlordState) HasAutomaticPresence() bool {
	return s.AutomaticPresence != NoShadowlord
}

// AllDefeated returns true if all three Shadowlords have been defeated.
func (s *ShadowlordState) AllDefeated() bool {
	return !s.Alive[Faulinei] && !s.Alive[Astaroth] && !s.Alive[Nosfentor]
}

// CountAlive returns the number of Shadowlords still alive.
func (s *ShadowlordState) CountAlive() int {
	count := 0
	for _, alive := range s.Alive {
		if alive {
			count++
		}
	}

	return count
}

// GetLivingShadowlords returns a slice of all living Shadowlords.
// Order matches original Ultima V: Nosfentor (2), Astaroth (1), Faulinei (0) - reverse order.
// Original identity resolution checks indices 2 down to 0; ref:SHADOW-0001.
func (s *ShadowlordState) GetLivingShadowlords() []Shadowlord {
	lords := make([]Shadowlord, 0, ShadowlordCount)

	// Original order: 2, 1, 0 (Nosfentor, Astaroth, Faulinei)
	for i := ShadowlordCount - 1; i >= 0; i-- {
		if s.Alive[i] {
			lords = append(lords, Shadowlord(i))
		}
	}

	return lords
}
