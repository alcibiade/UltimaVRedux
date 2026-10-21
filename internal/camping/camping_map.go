package camping

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Party position constants
const (
	MaxPartyMembers       = 8  // Maximum 8 party members
	MaxMonsterEntryPoints = 16 // Maximum 16 monster entry points
)

// Camping map center position
const (
	CampfirePositionX = 5 // Center X of 11x11 map
	CampfirePositionY = 5 // Center Y of 11x11 map
)

// CampingType defines the type of camping environment
type CampingType int

//goland:noinspection GoUnusedConst
const (
	OutdoorCamping CampingType = iota
	DungeonCamping
)

// BRIT.CBT camping position constants
const (
	// CampingDirectionIndex is the direction row used for outdoor camping positions in BRIT.CBT
	// Camping party coordinates occupy row 3 of BRIT.CBT metadata (ref:CBT-0002)
	// With FirstDirectionRow=1, direction index 2 maps to row 3 (FirstDirectionRow + 2 = row 3)
	CampingDirectionIndex = 2
)

// CampingMap represents the camping scene layout
type CampingMap struct {
	// Party member positions around the campfire
	PartyPositions [MaxPartyMembers]references.Position

	// Guard position (if any)
	GuardPosition references.Position
	HasGuard      bool
	GuardIndex    int // NoGuard means no guard

	// Monster entry points for encounters
	MonsterEntryPoints [MaxMonsterEntryPoints]references.Position

	// Campfire position (center of map)
	CampfirePosition references.Position

	// Camping type (outdoor vs dungeon)
	CampingType CampingType

	// Combat map reference (contains authentic positions from BRIT.CBT)
	combatMapRef *references.CombatMapReference
}

// NewCampingMap creates a new camping map with default layout
func NewCampingMap(guardIndex int) *CampingMap {
	return NewCampingMapWithType(guardIndex, OutdoorCamping)
}

// NewCampingMapWithType creates a new camping map with specified type
func NewCampingMapWithType(guardIndex int, campingType CampingType) *CampingMap {
	campingMap := &CampingMap{
		GuardIndex:       guardIndex,
		HasGuard:         guardIndex >= 0,
		CampfirePosition: references.Position{X: CampfirePositionX, Y: CampfirePositionY},
		CampingType:      campingType,
	}

	// Set up party positions around the campfire
	campingMap.setupPartyPositions()

	// Set up monster entry points at map edges
	campingMap.setupMonsterEntryPoints()

	return campingMap
}

// NewCampingMapWithCombatRef creates a new camping map using positions from combat map reference
// This provides authentic positions loaded from BRIT.CBT for outdoor camping
func NewCampingMapWithCombatRef(guardIndex int, campingType CampingType, combatMapRef *references.CombatMapReference) *CampingMap {
	campingMap := &CampingMap{
		GuardIndex:       guardIndex,
		HasGuard:         guardIndex >= 0,
		CampfirePosition: references.Position{X: CampfirePositionX, Y: CampfirePositionY},
		CampingType:      campingType,
		combatMapRef:     combatMapRef,
	}

	// Set up party positions (will use combatMapRef if available)
	campingMap.setupPartyPositions()

	// Set up monster entry points (will use combatMapRef if available)
	campingMap.setupMonsterEntryPoints()

	return campingMap
}

// setupPartyPositions arranges party members around the campfire
func (cm *CampingMap) setupPartyPositions() {
	var partyPositions []references.Position

	// If we have a combat map reference, use its authentic positions from BRIT.CBT
	if cm.combatMapRef != nil {
		// Use positions from combat map for the first 6 players (original U5 party size)
		// CampingDirectionIndex maps to row 3 in BRIT.CBT; ref:CBT-0002.
		for i := range references.MaxPlayersPerMap {
			partyPositions = append(partyPositions, cm.combatMapRef.PlayerPositions[CampingDirectionIndex][i])
		}
		// Add extra positions for players 6-7 (Redux supports up to 8 party members)
		// Place them intelligently around remaining spaces
		partyPositions = append(partyPositions, references.Position{X: 4, Y: 6}) // Player 6: Southwest
		partyPositions = append(partyPositions, references.Position{X: 6, Y: 4}) // Player 7: Northeast
	} else if cm.CampingType == OutdoorCamping {
		// Fallback: Outdoor camping positions from extracted BRIT.CBT data
		// These are the authentic positions: {X: 6, Y: 6}, {X: 4, Y: 4}, {X: 6, Y: 4}, {X: 4, Y: 6}, {X: 5, Y: 3}, {X: 3, Y: 5}
		partyPositions = []references.Position{
			{X: 6, Y: 6}, // Player 0: Southeast of campfire
			{X: 4, Y: 4}, // Player 1: Northwest of campfire
			{X: 6, Y: 4}, // Player 2: Northeast of campfire
			{X: 4, Y: 6}, // Player 3: Southwest of campfire
			{X: 5, Y: 3}, // Player 4: North (2 tiles from campfire)
			{X: 3, Y: 5}, // Player 5: West (2 tiles from campfire)
			{X: 7, Y: 5}, // Player 6: East (2 tiles from campfire) - Redux extension
			{X: 5, Y: 7}, // Player 7: South (2 tiles from campfire) - Redux extension
		}
	} else {
		// Dungeon camping positions - exact coordinates from original Ultima V
		// Use the six central party positions for generated dungeon camping; ref:DNG-0010.
		partyPositions = []references.Position{
			{X: 5, Y: 4}, // Player 0: North of campfire
			{X: 6, Y: 5}, // Player 1: East of campfire
			{X: 5, Y: 6}, // Player 2: South of campfire
			{X: 4, Y: 5}, // Player 3: West of campfire
			{X: 4, Y: 4}, // Player 4: Northwest of campfire
			{X: 6, Y: 6}, // Player 5: Southeast of campfire
			{X: 4, Y: 6}, // Player 6: Southwest (extra) - Redux extension
			{X: 6, Y: 4}, // Player 7: Northeast (extra) - Redux extension
		}
	}

	for i := 0; i < len(cm.PartyPositions) && i < len(partyPositions); i++ {
		cm.PartyPositions[i] = partyPositions[i]
	}

	// If there's a guard, set the guard position
	if cm.HasGuard && cm.GuardIndex >= 0 && cm.GuardIndex < len(cm.PartyPositions) {
		cm.GuardPosition = cm.PartyPositions[cm.GuardIndex]
	}
}

// setupMonsterEntryPoints defines where monsters can enter during encounters
func (cm *CampingMap) setupMonsterEntryPoints() {
	// If we have a combat map reference, use its authentic monster positions from BRIT.CBT
	if cm.combatMapRef != nil {
		cm.MonsterEntryPoints = cm.combatMapRef.MonsterPositions

		return
	}

	// Fallback: Entry points along the edges of the map for monster spawning
	entryPoints := []references.Position{
		// North edge
		{X: 3, Y: 0}, {X: 5, Y: 0}, {X: 7, Y: 0},
		// East edge
		{X: 10, Y: 3}, {X: 10, Y: 5}, {X: 10, Y: 7},
		// South edge
		{X: 7, Y: 10}, {X: 5, Y: 10}, {X: 3, Y: 10},
		// West edge
		{X: 0, Y: 7}, {X: 0, Y: 5}, {X: 0, Y: 3},
		// Corner positions
		{X: 1, Y: 1}, {X: 9, Y: 1}, {X: 9, Y: 9}, {X: 1, Y: 9},
	}

	for i := 0; i < len(cm.MonsterEntryPoints) && i < len(entryPoints); i++ {
		cm.MonsterEntryPoints[i] = entryPoints[i]
	}
}

// HasGeneratedDungeonChestAt reports whether the source map still displays its
// generated center-chest marker during dungeon camping.
func (cm *CampingMap) HasGeneratedDungeonChestAt(position references.Position) bool {
	if cm.combatMapRef == nil {
		return false
	}
	chest, found := cm.combatMapRef.GeneratedDungeonChest()

	return found && chest.Position == position
}

// GetPartyMemberPosition returns the position for a specific party member
// If the member is the guard, returns the dynamic GuardPosition (updated during wandering)
// Otherwise returns the static PartyPositions array position
func (cm *CampingMap) GetPartyMemberPosition(memberIndex int) references.Position {
	if memberIndex < 0 || memberIndex >= len(cm.PartyPositions) {
		return references.Position{X: CampfirePositionX, Y: CampfirePositionY} // Default to campfire
	}

	// If this is the guard, return their current wandering position
	if cm.HasGuard && memberIndex == cm.GuardIndex {
		return cm.GuardPosition
	}

	// Otherwise return the fixed party position
	return cm.PartyPositions[memberIndex]
}
