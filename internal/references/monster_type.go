package references

import (
	"sort"
	"strings"
)

// MonsterType represents the index into the enemies array (0-47).
// This is the authoritative identity for each monster type,
// matching the creature catalog order in ref:ENEMY-0001.
// MonsterNone (-1) indicates no monster or invalid monster.
type MonsterType int

// Monster type constants matching the creature catalog indices; ref:ENEMY-0001.
// These values directly index into EnemyReferences slice.
const (
	MonsterNone MonsterType = -1 // No monster / invalid

	// Player classes and NPCs (indices 0-15)
	MonsterMage        MonsterType = 0  // Wizard/Mage
	MonsterBard        MonsterType = 1  // Bard
	MonsterFighter     MonsterType = 2  // Fighter
	MonsterAvatar      MonsterType = 3  // Avatar
	MonsterVillager    MonsterType = 4  // TownsPerson
	MonsterMerchant    MonsterType = 5  // Merchant
	MonsterJester      MonsterType = 6  // Jester
	MonsterBanjoPlayer MonsterType = 7  // Bard playing banjo
	MonsterStocks      MonsterType = 8  // Person in stocks (also used for pirates)
	MonsterManacles    MonsterType = 9  // Person in manacles
	MonsterChild       MonsterType = 10 // Child
	MonsterBeggar      MonsterType = 11 // Beggar
	MonsterGuard       MonsterType = 12 // Guard
	MonsterWanderer    MonsterType = 13 // Wanderer
	MonsterBlackthorn  MonsterType = 14 // Blackthorn
	MonsterLordBritish MonsterType = 15 // Lord British

	// Water monsters (indices 16-19)
	MonsterSeaHorse   MonsterType = 16 // Sea Horse
	MonsterSquid      MonsterType = 17 // Squid
	MonsterSeaSerpent MonsterType = 18 // Sea Serpent
	MonsterShark      MonsterType = 19 // Shark

	// Common monsters (indices 20-31)
	MonsterRat      MonsterType = 20 // Giant Rat
	MonsterBat      MonsterType = 21 // Bat
	MonsterSpider   MonsterType = 22 // Spider
	MonsterGhost    MonsterType = 23 // Ghost
	MonsterSlime    MonsterType = 24 // Slime
	MonsterGremlin  MonsterType = 25 // Gremlin
	MonsterMimic    MonsterType = 26 // Mimic
	MonsterReaper   MonsterType = 27 // Reaper
	MonsterGazer    MonsterType = 28 // Gazer
	MonsterCrawler  MonsterType = 29 // Crawler
	MonsterGargoyle MonsterType = 30 // Gargoyle
	MonsterInsect   MonsterType = 31 // Insect Swarm

	// Humanoid monsters (indices 32-39)
	MonsterOrc      MonsterType = 32 // Orc
	MonsterSkeleton MonsterType = 33 // Skeleton
	MonsterSnake    MonsterType = 34 // Snake
	MonsterEttin    MonsterType = 35 // Ettin
	MonsterHeadless MonsterType = 36 // Headless
	MonsterWisp     MonsterType = 37 // Wisp
	MonsterDevil    MonsterType = 38 // Daemon/Devil
	MonsterDragon   MonsterType = 39 // Dragon

	// Special monsters and hazards (indices 40-47)
	MonsterSandtrap   MonsterType = 40 // Sand Trap
	MonsterTroll      MonsterType = 41 // Troll
	MonsterForceField MonsterType = 42 // Force Field (spell-created, not encountered)
	MonsterWhirlpool  MonsterType = 43 // Whirlpool (environmental hazard)
	MonsterMongbat    MonsterType = 44 // Mongbat
	MonsterCorpser    MonsterType = 45 // Corpser
	MonsterRotWorm    MonsterType = 46 // Rot Worm
	MonsterShadowlord MonsterType = 47 // Shadowlord
)

// TotalMonsterTypes is the total number of monster types in the game.
const TotalMonsterTypes = 48

// monsterNameNone is the name returned for invalid monster types.
const monsterNameNone = "None"

// Index returns the integer index of the monster type.
func (m MonsterType) Index() int {
	return int(m)
}

// IsValid returns true if this is a valid monster type (not MonsterNone).
func (m MonsterType) IsValid() bool {
	return m >= 0 && m < TotalMonsterTypes
}

// MonsterBaseHitPoints returns the original undamaged hit points for a monster.
func MonsterBaseHitPoints(monster MonsterType) int {
	if !monster.IsValid() {
		return 0
	}

	return monStatsData[monster][5]
}

// Name returns the human-readable name of the monster type.
func (m MonsterType) Name() string {
	if !m.IsValid() {
		return monsterNameNone
	}

	return monsterTypeNames[m]
}

// String implements fmt.Stringer.
func (m MonsterType) String() string {
	return m.Name()
}

// IsEnvironmentalHazard returns true if this monster is an environmental hazard
// that has special handling (ForceField, Whirlpool).
func (m MonsterType) IsEnvironmentalHazard() bool {
	return m == MonsterForceField || m == MonsterWhirlpool
}

// IsWaterMonster returns true if this monster is a water-dwelling creature
// that appears in water encounters (indices 16-19).
func (m MonsterType) IsWaterMonster() bool {
	return m >= MonsterSeaHorse && m <= MonsterShark
}

// IsUndead returns true if this monster is undead (takes half damage from non-magic weapons).
// Uses the catalog's undead ability flag; ref:ENEMY-0001.
func (m MonsterType) IsUndead() bool {
	return m == MonsterGhost || m == MonsterSkeleton || m == MonsterDevil || m == MonsterShadowlord
}

// IsFixedGroup returns true if this monster type has a fixed group size
// (Guards=8, Swarms=16, Solo=1) rather than random.
func (m MonsterType) IsFixedGroup() bool {
	return m == MonsterGuard || m.IsSwarm() || m.IsSolo()
}

// IsSwarm returns true if this monster appears in swarms of 16.
// Uses the fixed appearance bound 16; ref:ENEMY-0003.
func (m MonsterType) IsSwarm() bool {
	return m == MonsterBat || m == MonsterMongbat || m == MonsterSlime
}

// IsSolo returns true if this monster always appears alone.
// Uses the fixed appearance bound 1; ref:ENEMY-0003.
//
//nolint:exhaustive // Default case handles all other monster types
func (m MonsterType) IsSolo() bool {
	switch m {
	case MonsterAvatar, MonsterWanderer, MonsterBlackthorn, MonsterLordBritish,
		MonsterSeaSerpent, MonsterMimic, MonsterGargoyle, MonsterDragon,
		MonsterSandtrap, MonsterShadowlord:
		return true
	default:
		return false
	}
}

// IsGuard returns true if this is a guard (appears in groups of 8).
func (m MonsterType) IsGuard() bool {
	return m == MonsterGuard
}

// monsterTypeNames maps MonsterType to human-readable names.
// Matches the creature display-name order in ref:ENEMY-0001.
//
//nolint:gochecknoglobals // Static lookup table for monster names
var monsterTypeNames = map[MonsterType]string{
	MonsterMage:        "Mage",
	MonsterBard:        "Bard",
	MonsterFighter:     "Fighter",
	MonsterAvatar:      "Avatar",
	MonsterVillager:    "Villager",
	MonsterMerchant:    "Merchant",
	MonsterJester:      "Jester",
	MonsterBanjoPlayer: "Bard",
	MonsterStocks:      "Pirate",
	MonsterManacles:    "Prisoner",
	MonsterChild:       "Child",
	MonsterBeggar:      "Beggar",
	MonsterGuard:       "Guard",
	MonsterWanderer:    "Wanderer",
	MonsterBlackthorn:  "Blackthorn",
	MonsterLordBritish: "Lord British",
	MonsterSeaHorse:    "Sea Horse",
	MonsterSquid:       "Squid",
	MonsterSeaSerpent:  "Sea Serpent",
	MonsterShark:       "Shark",
	MonsterRat:         "Rat",
	MonsterBat:         "Bat",
	MonsterSpider:      "Spider",
	MonsterGhost:       "Ghost",
	MonsterSlime:       "Slime",
	MonsterGremlin:     "Gremlin",
	MonsterMimic:       "Mimic",
	MonsterReaper:      "Reaper",
	MonsterGazer:       "Gazer",
	MonsterCrawler:     "Crawler",
	MonsterGargoyle:    "Gargoyle",
	MonsterInsect:      "Insect",
	MonsterOrc:         "Orc",
	MonsterSkeleton:    "Skeleton",
	MonsterSnake:       "Snake",
	MonsterEttin:       "Ettin",
	MonsterHeadless:    "Headless",
	MonsterWisp:        "Wisp",
	MonsterDevil:       "Daemon",
	MonsterDragon:      "Dragon",
	MonsterSandtrap:    "Sand Trap",
	MonsterTroll:       "Troll",
	MonsterForceField:  "Force Field",
	MonsterWhirlpool:   "Whirlpool",
	MonsterMongbat:     "Mongbat",
	MonsterCorpser:     "Corpser",
	MonsterRotWorm:     "Rot Worm",
	MonsterShadowlord:  "Shadowlord",
}

// ParseMonsterType converts a string to MonsterType.
// Accepts both spaces and hyphens (e.g., "sea horse" or "sea-horse").
// Returns MonsterNone if not found.
func ParseMonsterType(s string) MonsterType {
	s = strings.ToLower(strings.TrimSpace(s))
	for mt, name := range monsterTypeNames {
		lowerName := strings.ToLower(name)
		// Try exact match first
		if lowerName == s {
			return mt
		}
		// Try hyphenated form (sea-horse matches "Sea Horse")
		if strings.ReplaceAll(lowerName, " ", "-") == s {
			return mt
		}
	}

	return MonsterNone
}

// GetMonsterNames returns a slice of all monster names in hyphenated form.
// Used for debug command autocomplete.
func GetMonsterNames() []string {
	names := make([]string, 0, len(monsterTypeNames))
	seen := make(map[string]bool, len(monsterTypeNames))

	for _, name := range monsterTypeNames {
		// Convert to hyphenated form for autocomplete
		hyphenated := strings.ReplaceAll(strings.ToLower(name), " ", "-")
		if !seen[hyphenated] {
			seen[hyphenated] = true
			names = append(names, hyphenated)
		}
	}

	sort.Strings(names)

	return names
}
