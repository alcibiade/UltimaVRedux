// Package camping lays out the camping scene: party positions around the
// campfire, the wandering guard, and monster entry points from BRIT.CBT.
package camping

import (
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Party position constants
const (
	MaxPartyMembers       = references.MaxPlayersPerMap // The original six-member party
	MaxMonsterEntryPoints = 16                          // Maximum 16 monster entry points
)

// Camping map center position
const (
	CampfirePositionX = 5 // Center X of 11x11 map
	CampfirePositionY = 5 // Center Y of 11x11 map
)

// Environment defines the type of camping environment
type Environment int

// Camping environments: outdoors on the surface, or inside a dungeon.
//
//goland:noinspection GoUnusedConst
const (
	OutdoorCamping Environment = iota
	DungeonCamping
)

// BRIT.CBT camping position constants
const (
	// CampingDirectionIndex is the direction row used for outdoor camping positions in BRIT.CBT
	// Camping party coordinates occupy row 3 of BRIT.CBT metadata (ref:CBT-0002)
	// With FirstDirectionRow=1, direction index 2 maps to row 3 (FirstDirectionRow + 2 = row 3)
	CampingDirectionIndex = 2
)

// Map represents the camping scene layout
type Map struct {
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
	CampingType Environment

	// Combat map reference (contains authentic positions from BRIT.CBT)
	combatMapRef *references.CombatMapReference
}

// NewCampingMapWithCombatRef creates a new camping map using positions from combat map reference
// This provides authentic positions loaded from BRIT.CBT for outdoor camping
func NewCampingMapWithCombatRef(
	guardIndex int, campingType Environment, combatMapRef *references.CombatMapReference,
) *Map {
	if combatMapRef == nil {
		panic("camping.NewCampingMapWithCombatRef: combat map reference is required")
	}
	campingMap := &Map{
		PartyPositions:     [MaxPartyMembers]references.Position{}, // Set by setupPartyPositions
		GuardPosition:      references.Position{X: 0, Y: 0},        // Set when a guard is posted
		HasGuard:           guardIndex >= 0,
		GuardIndex:         guardIndex,
		MonsterEntryPoints: [MaxMonsterEntryPoints]references.Position{}, // Set by setupMonsterEntryPoints
		CampfirePosition:   references.Position{X: CampfirePositionX, Y: CampfirePositionY},
		CampingType:        campingType,
		combatMapRef:       combatMapRef,
	}

	// Set up party positions (will use combatMapRef if available)
	campingMap.setupPartyPositions()

	// Set up monster entry points (will use combatMapRef if available)
	campingMap.setupMonsterEntryPoints()

	return campingMap
}

// setupPartyPositions arranges party members around the campfire
func (cm *Map) setupPartyPositions() {
	// Party positions come from BRIT.CBT; CampingDirectionIndex maps to row 3. ref:CBT-0002.
	cm.PartyPositions = cm.combatMapRef.PlayerPositions[CampingDirectionIndex]
	if cm.HasGuard && cm.GuardIndex >= 0 && cm.GuardIndex < len(cm.PartyPositions) {
		cm.GuardPosition = cm.PartyPositions[cm.GuardIndex]
	}
}

// setupMonsterEntryPoints defines where monsters can enter during encounters
func (cm *Map) setupMonsterEntryPoints() {
	cm.MonsterEntryPoints = cm.combatMapRef.MonsterPositions
}

// HasGeneratedDungeonChestAt reports whether the source map still displays its
// generated center-chest marker during dungeon camping.
func (cm *Map) HasGeneratedDungeonChestAt(position references.Position) bool {
	if cm.combatMapRef == nil {
		return false
	}
	chest, found := cm.combatMapRef.GeneratedDungeonChest()

	return found && chest.Position == position
}

// GetPartyMemberPosition returns the position for a specific party member
// If the member is the guard, returns the dynamic GuardPosition (updated during wandering)
// Otherwise returns the static PartyPositions array position
func (cm *Map) GetPartyMemberPosition(memberIndex int) references.Position {
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
