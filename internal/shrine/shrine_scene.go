package shrine

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ShrinePhase represents the current phase of shrine meditation
type ShrinePhase int

const (
	PhaseEntering   ShrinePhase = iota // Walking into shrine
	PhaseMeditating                    // At altar, answering prompts
	PhaseExiting                       // Walking out of shrine
)

// Walk animation constants
const (
	WalkStepsToAltar = 4                              // Steps from entrance to altar
	WalkDelayMs      = references.StandardWalkDelayMs // ~330ms, 6 ticks per step (original 18Hz timing)
	EntranceY        = 10                             // Y position at shrine entrance
	AltarY           = 5                              // Y position at altar (center)
	CenterX          = 5                              // X position (always centered)
)

// ShrineScene manages the shrine visual experience and state
type ShrineScene struct {
	// Shrine identification
	ShrineIndex int

	// Current phase and animation state
	Phase    ShrinePhase
	WalkStep int // Current step in walk animation (0-4)

	// Scene state
	IsActive bool

	// Original player location (to return to after shrine visit)
	OriginalPlayerLocation references.PlayerLocation

	// Shrine map reference
	ShrineMap *ShrineMap
}

// NewShrineScene creates a new shrine scene for meditation
func NewShrineScene(shrineIndex int, originalPlayerLocation references.PlayerLocation) *ShrineScene {
	scene := &ShrineScene{
		ShrineIndex:            shrineIndex,
		Phase:                  PhaseEntering,
		WalkStep:               0,
		IsActive:               true,
		OriginalPlayerLocation: originalPlayerLocation,
	}

	// Create the shrine map
	scene.ShrineMap = NewShrineMap()

	logging.Debug(logging.CategoryGameState, "Created shrine scene for shrine %d", shrineIndex)

	return scene
}

// NewShrineSceneWithMap creates a shrine scene with a specific map reference
func NewShrineSceneWithMap(shrineIndex int, originalPlayerLocation references.PlayerLocation, mapData *MiniMapData) *ShrineScene {
	scene := &ShrineScene{
		ShrineIndex:            shrineIndex,
		Phase:                  PhaseEntering,
		WalkStep:               0,
		IsActive:               true,
		OriginalPlayerLocation: originalPlayerLocation,
	}

	// Create the shrine map with loaded data
	scene.ShrineMap = NewShrineMapWithData(mapData)

	logging.Debug(logging.CategoryGameState, "Created shrine scene for shrine %d with loaded map data", shrineIndex)

	return scene
}

// StartEntering begins the walking-in animation
func (ss *ShrineScene) StartEntering() {
	ss.Phase = PhaseEntering
	ss.WalkStep = 0
	logging.Debug(logging.CategoryGameState, "Starting shrine entry animation")
}

// AdvanceWalkIn advances the walking-in animation by one step
// Returns true if more steps remain, false when at altar
func (ss *ShrineScene) AdvanceWalkIn() bool {
	if ss.Phase != PhaseEntering {
		return false
	}

	ss.WalkStep++
	logging.Debug(logging.CategoryGameState, "Shrine walk-in step %d/%d", ss.WalkStep, WalkStepsToAltar)

	if ss.WalkStep >= WalkStepsToAltar {
		ss.Phase = PhaseMeditating
		logging.Debug(logging.CategoryGameState, "Reached altar, entering meditation phase")

		return false // No more walk-in steps
	}

	return true // More steps to go
}

// StartMeditating sets the scene to meditation phase
func (ss *ShrineScene) StartMeditating() {
	ss.Phase = PhaseMeditating
	logging.Debug(logging.CategoryGameState, "Entering meditation phase at altar")
}

// StartExiting begins the walking-out animation
func (ss *ShrineScene) StartExiting() {
	ss.Phase = PhaseExiting
	ss.WalkStep = WalkStepsToAltar // Start at altar position
	logging.Debug(logging.CategoryGameState, "Starting shrine exit animation")
}

// AdvanceWalkOut advances the walking-out animation by one step
// Returns true if more steps remain, false when at entrance
func (ss *ShrineScene) AdvanceWalkOut() bool {
	if ss.Phase != PhaseExiting {
		return false
	}

	ss.WalkStep--
	logging.Debug(logging.CategoryGameState, "Shrine walk-out step %d/%d", WalkStepsToAltar-ss.WalkStep, WalkStepsToAltar)

	if ss.WalkStep <= 0 {
		ss.IsActive = false
		logging.Debug(logging.CategoryGameState, "Reached entrance, shrine visit complete")

		return false // No more walk-out steps
	}

	return true // More steps to go
}

// GetPlayerPosition returns the current player position on the shrine map
func (ss *ShrineScene) GetPlayerPosition() references.Position {
	// Calculate Y position based on walk step
	// Walk step 0 = entrance (Y=10), step 4 = altar (Y=5)
	// Each step moves 1 tile up (Y decreases by 1.25 per step for smooth animation)
	// Using integer positions: 10 -> 9 -> 8 -> 7 -> 6 -> 5
	yPos := EntranceY - ss.WalkStep

	if yPos < AltarY {
		yPos = AltarY
	}

	return references.Position{X: references.Coordinate(CenterX), Y: references.Coordinate(yPos)}
}

// GetOriginalPlayerLocation returns the player location to return to after shrine
func (ss *ShrineScene) GetOriginalPlayerLocation() references.PlayerLocation {
	return ss.OriginalPlayerLocation
}

// Complete ends the shrine scene
func (ss *ShrineScene) Complete() {
	ss.IsActive = false
	logging.Debug(logging.CategoryGameState, "Shrine scene completed for shrine %d", ss.ShrineIndex)
}

// Interrupt stops the shrine scene early (e.g., on failed validation)
func (ss *ShrineScene) Interrupt() {
	ss.IsActive = false
	logging.Debug(logging.CategoryGameState, "Shrine scene interrupted for shrine %d", ss.ShrineIndex)
}

// IsMeditating returns true if currently at the altar meditating
func (ss *ShrineScene) IsMeditating() bool {
	return ss.Phase == PhaseMeditating
}

// IsEntering returns true if walking into the shrine
func (ss *ShrineScene) IsEntering() bool {
	return ss.Phase == PhaseEntering
}

// IsExiting returns true if walking out of the shrine
func (ss *ShrineScene) IsExiting() bool {
	return ss.Phase == PhaseExiting
}
