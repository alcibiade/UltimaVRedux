package camping

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Time cycle constants
const (
	MinutesPerCycle        = 5
	CyclesPerHour          = 12
	DungeonMinutesPerCycle = 5
	DungeonCyclesPerHour   = 12
)

// Hour message tracking
const (
	NoHourMessageYet = -1 // Initial value for LastHourMessage
)

// Minutes per hour for time calculations
const (
	MinutesPerHour = 60
)

// CampingScene manages the camping visual experience and state
type CampingScene struct {
	// Current camping map
	CampingMap *CampingMap

	// Camping state
	Hours        int
	GuardIndex   int
	CurrentCycle int
	TotalCycles  int
	CycleMinutes int

	// Scene state
	IsActive        bool
	TimeElapsed     int // Minutes elapsed
	LastHourMessage int // Last hour we showed a time message

	// Original player location (to return to after camping)
	OriginalPlayerLocation references.PlayerLocation

	// Wanderer visual state

	WandererPresent       bool    // Whether wanderer should be visible on camping map
	WandererDissolveLevel float64 // 0.0 = fully visible, 1.0 = fully dissolved
}

// NewCampingScene creates a new camping scene
func NewCampingScene(hours int, guardIndex int, originalPlayerLocation references.PlayerLocation) *CampingScene {
	scene := &CampingScene{
		Hours:                  hours,
		GuardIndex:             guardIndex,
		TotalCycles:            hours * CyclesPerHour,
		CycleMinutes:           MinutesPerCycle,
		CurrentCycle:           0,
		IsActive:               true,
		TimeElapsed:            0,
		LastHourMessage:        NoHourMessageYet,
		OriginalPlayerLocation: originalPlayerLocation,
	}

	// Create the camping map with guard configuration
	scene.CampingMap = NewCampingMap(guardIndex)

	logging.Debug(logging.CategoryGameState, "Created camping scene for %d hours with guard %d", hours, guardIndex)

	return scene
}

// NewCampingSceneWithCombatRef creates a new camping scene with authentic positions from combat map
func NewCampingSceneWithCombatRef(hours int, guardIndex int, originalPlayerLocation references.PlayerLocation, combatMapRef *references.CombatMapReference) *CampingScene {
	scene := &CampingScene{
		Hours:                  hours,
		GuardIndex:             guardIndex,
		TotalCycles:            hours * CyclesPerHour,
		CycleMinutes:           MinutesPerCycle,
		CurrentCycle:           0,
		IsActive:               true,
		TimeElapsed:            0,
		LastHourMessage:        NoHourMessageYet,
		OriginalPlayerLocation: originalPlayerLocation,
	}

	// Create the camping map with combat map reference for authentic positions
	scene.CampingMap = NewCampingMapWithCombatRef(guardIndex, OutdoorCamping, combatMapRef)

	logging.Debug(logging.CategoryGameState, "Created camping scene with authentic BRIT.CBT positions for %d hours with guard %d", hours, guardIndex)

	return scene
}

// NewDungeonCampingScene creates a dungeon camp using a generated dungeon map.
func NewDungeonCampingScene(
	hours int,
	guardIndex int,
	originalPlayerLocation references.PlayerLocation,
	combatMapRef *references.CombatMapReference,
) *CampingScene {
	return &CampingScene{
		CampingMap:             NewCampingMapWithCombatRef(guardIndex, DungeonCamping, combatMapRef),
		Hours:                  hours,
		GuardIndex:             guardIndex,
		CurrentCycle:           0,
		TotalCycles:            hours * DungeonCyclesPerHour,
		CycleMinutes:           DungeonMinutesPerCycle,
		IsActive:               true,
		TimeElapsed:            0,
		LastHourMessage:        NoHourMessageYet,
		OriginalPlayerLocation: originalPlayerLocation,
		WandererPresent:        false,
		WandererDissolveLevel:  0,
	}
}

// StartCamping begins the camping scene
func (cs *CampingScene) StartCamping() {
	cs.IsActive = true
	cs.CurrentCycle = 0
	cs.TimeElapsed = 0

	logging.Debug(logging.CategoryGameState, "Starting camping scene for %d cycles", cs.TotalCycles)
}

// AdvanceCycle advances the camping by one cycle
func (cs *CampingScene) AdvanceCycle() bool {
	if !cs.IsActive || cs.CurrentCycle >= cs.TotalCycles {
		return false // Camping complete or not active
	}

	cs.CurrentCycle++
	cs.TimeElapsed += cs.CycleMinutes

	// Check if we should show an hour message
	currentHour := cs.TimeElapsed / MinutesPerHour
	if currentHour > cs.LastHourMessage {
		cs.LastHourMessage = currentHour
		logging.Debug(logging.CategoryGameState, "Camping: %d hour(s) completed", currentHour)
	}

	// Check if camping is complete
	if cs.CurrentCycle >= cs.TotalCycles {
		cs.CompleteCamping()

		return false
	}

	return true // Continue camping
}

// AdvanceExactCycle advances and processes the final configured cycle.
func (cs *CampingScene) AdvanceExactCycle() bool {
	if !cs.IsActive || cs.CurrentCycle >= cs.TotalCycles {
		return false
	}
	cs.CurrentCycle++
	cs.TimeElapsed += cs.CycleMinutes
	if cs.CurrentCycle >= cs.TotalCycles {
		cs.CompleteCamping()
	}

	return true
}

// CompleteCamping ends the camping scene
func (cs *CampingScene) CompleteCamping() {
	cs.IsActive = false
	logging.Debug(logging.CategoryGameState, "Camping scene completed after %d cycles", cs.CurrentCycle)
}

// InterruptCamping stops camping early (due to encounters or death)
func (cs *CampingScene) InterruptCamping() {
	cs.IsActive = false
	logging.Debug(logging.CategoryGameState, "Camping scene interrupted at cycle %d/%d", cs.CurrentCycle, cs.TotalCycles)
}

// GetPartyMemberPosition returns the position for a specific party member
func (cs *CampingScene) GetPartyMemberPosition(memberIndex int) references.Position {
	return cs.CampingMap.GetPartyMemberPosition(memberIndex)
}

// GetOriginalPlayerLocation returns the player location to return to after camping
func (cs *CampingScene) GetOriginalPlayerLocation() references.PlayerLocation {
	return cs.OriginalPlayerLocation
}

// SetWandererPresent sets whether the wanderer should be visible
func (cs *CampingScene) SetWandererPresent(present bool) {
	cs.WandererPresent = present
	logging.Debug(logging.CategoryGameState, "Wanderer presence set to: %t", present)
}

// SetWandererDissolveLevel sets the wanderer's dissolve level for animation
func (cs *CampingScene) SetWandererDissolveLevel(level float64) {
	cs.WandererDissolveLevel = level
	logging.Debug(logging.CategoryGameState, "Wanderer dissolve level set to: %.2f", level)
}

// GetWandererPosition returns the wanderer's position on the camping map
func (cs *CampingScene) GetWandererPosition() references.Position {
	// Position wanderer on the opposite side of campfire from party
	// This puts the wanderer in a visually prominent position
	return references.Position{X: 5, Y: 5} // Top side of camping area
}

// UpdateGuardPosition updates the guard's visual position during camping
// This is called when the guard wanders around the campsite
// Guard movement updates the selected watch character's position; ref:CAMP-0003.
func (cs *CampingScene) UpdateGuardPosition(newPosition references.Position) {
	// Update the camping map's guard position for rendering
	// This tells the visual system where to draw the guard sprite
	if cs.CampingMap != nil {
		cs.CampingMap.GuardPosition = newPosition
		logging.Debug(logging.CategoryGameState,
			"Updated guard position to (%d,%d) in camping scene",
			newPosition.X, newPosition.Y)
	}
}
