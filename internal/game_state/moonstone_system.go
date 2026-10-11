package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// MoonstoneSystem manages the tracking and state of all 8 moonstones for moongate travel
type MoonstoneSystem struct {
	Moonstones []MoonstoneLocation // All 8 moonstone locations and states
}

// NewMoonstoneSystem creates a new moonstone system with initial buried locations
func NewMoonstoneSystem() *MoonstoneSystem {
	// Create moonstones from initial reference data
	moonstones := make([]MoonstoneLocation, len(references.InitialMoonstoneLocations))
	for i, initial := range references.InitialMoonstoneLocations {
		moonstones[i] = MoonstoneLocation{
			X:       initial.X,
			Y:       initial.Y,
			Map:     initial.Map,
			Level:   initial.Level,
			State:   MoonstoneStateBuried, // Start buried
			IsFound: false,                // Start undiscovered
		}
	}

	return &MoonstoneSystem{
		Moonstones: moonstones,
	}
}

// LoadFromSaveData creates moonstone system from SAVED.GAM data
// Loads from stones_x, stones_y, stones_m, stones_l arrays at offsets 0x028A-0x02A9
func (ms *MoonstoneSystem) LoadFromSaveData(stonesX, stonesY, stonesM, stonesL []byte) {
	if len(stonesX) != 8 || len(stonesY) != 8 || len(stonesM) != 8 || len(stonesL) != 8 {
		return // Invalid data, keep defaults
	}

	ms.Moonstones = make([]MoonstoneLocation, 8)
	for i := range 8 {
		ms.Moonstones[i] = MoonstoneLocationFromSaveFormat(
			stonesX[i], stonesY[i], stonesM[i], stonesL[i])
	}
}

// ToSaveData converts moonstone system to SAVED.GAM format
// Returns stones_x, stones_y, stones_m, stones_l arrays for saving
func (ms *MoonstoneSystem) ToSaveData() (stonesX, stonesY, stonesM, stonesL []byte) {
	stonesX = make([]byte, 8)
	stonesY = make([]byte, 8)
	stonesM = make([]byte, 8)
	stonesL = make([]byte, 8)

	for i, moonstone := range ms.Moonstones {
		stonesX[i], stonesY[i], stonesM[i], stonesL[i] = moonstone.ToSaveFormat()
	}

	return stonesX, stonesY, stonesM, stonesL
}

// GetMoonstoneByIndex returns moonstone at specific index (0-7)
func (ms *MoonstoneSystem) GetMoonstoneByIndex(index int) *MoonstoneLocation {
	if index < 0 || index >= len(ms.Moonstones) {
		return nil
	}

	return &ms.Moonstones[index]
}

// MarkMoonstoneFound marks a moonstone as discovered at the given index
// The moonstone remains buried at its location - it doesn't automatically go to inventory
// Note: This only updates the local IsFound flag. For save game compatibility,
// the obj_found array should be updated separately when saving.
func (ms *MoonstoneSystem) MarkMoonstoneFound(index int) {
	if index >= 0 && index < len(ms.Moonstones) {
		ms.Moonstones[index].IsFound = true
		// Keep moonstone buried at current location - don't put in inventory yet
		// Player must explicitly take it or it can create a moongate when buried
	}
}

// MarkMoonstoneFoundAtPosition marks a moonstone as found and transitions to FoundNotCollected state
// This is used when moonstone is discovered through search at moongate positions
func (ms *MoonstoneSystem) MarkMoonstoneFoundAtPosition(index int) {
	if index >= 0 && index < len(ms.Moonstones) {
		ms.Moonstones[index].IsFound = true
		ms.Moonstones[index].State = MoonstoneStateFoundNotCollected
	}
}

// TakeMoonstoneToInventory moves a found moonstone from its buried location to inventory
func (ms *MoonstoneSystem) TakeMoonstoneToInventory(index int) {
	if index >= 0 && index < len(ms.Moonstones) && ms.Moonstones[index].IsFound {
		ms.Moonstones[index].State = MoonstoneStateInInventory
		ms.Moonstones[index].Map = 0xFF // Update save format field too
	}
}

// BuryMoonstone places a moonstone from inventory at the specified location
// Based on original item-use rules moonstone burial logic
//
// See ref:MOON-0004.
//
//nolint:varnamelen // x,y are standard coordinate parameter names
func (ms *MoonstoneSystem) BuryMoonstone(index int, x, y uint8, mapNum uint8, floor uint8) {
	if index < 0 || index >= len(ms.Moonstones) {
		return
	}

	ms.Moonstones[index].X = x
	ms.Moonstones[index].Y = y
	ms.Moonstones[index].Map = mapNum
	ms.Moonstones[index].Level = floor
	ms.Moonstones[index].State = MoonstoneStateBuried
	ms.Moonstones[index].IsFound = false // Reset found state - moonstone is now hidden
}

// GetMoonstoneAtPosition finds moonstone buried at specific coordinates
// Returns moonstone index if found, -1 if no moonstone at position
// Only returns unfound moonstones (for search discovery)
// NOTE: This does NOT check map - use GetMoonstoneAtPositionOnMap for proper search
func (ms *MoonstoneSystem) GetMoonstoneAtPosition(x, y uint8) int {
	for i, moonstone := range ms.Moonstones {
		if moonstone.X == x && moonstone.Y == y && !moonstone.IsFound && moonstone.IsBuried() {
			return i
		}
	}

	return -1
}

// GetMoonstoneAtPositionOnMap finds moonstone buried at specific coordinates on specific map
// Based on original Search/Get/Open/Jimmy rules buried-moonstone search which checks: x, y, level, AND map
// Returns moonstone index if found, -1 if no moonstone at position
// Only returns unfound moonstones (for search discovery)
// See ref:MOON-0004.
func (ms *MoonstoneSystem) GetMoonstoneAtPositionOnMap(x, y, level, mapNum uint8) int {
	for i, moonstone := range ms.Moonstones {
		if moonstone.X == x && moonstone.Y == y &&
			moonstone.Level == level && moonstone.Map == mapNum &&
			!moonstone.IsFound && moonstone.IsBuried() {
			return i
		}
	}

	return -1
}

// GetAnyMoonstoneAtPosition finds any moonstone at specific coordinates (found or unfound)
// Returns moonstone index if found, -1 if no moonstone at position
func (ms *MoonstoneSystem) GetAnyMoonstoneAtPosition(x, y uint8) int {
	for i, moonstone := range ms.Moonstones {
		if moonstone.X == x && moonstone.Y == y && moonstone.IsBuried() {
			return i
		}
	}

	return -1
}

// GetFoundMoonstoneAtPosition finds a moonstone in FoundNotCollected state at specific coordinates
// Returns moonstone index if found, -1 if no found moonstone at position
func (ms *MoonstoneSystem) GetFoundMoonstoneAtPosition(x, y uint8) int {
	for i, moonstone := range ms.Moonstones {
		if moonstone.X == x && moonstone.Y == y && moonstone.IsFoundNotCollected() {
			return i
		}
	}

	return -1
}

// GameState moonstone integration methods

// GetMoonstoneSystem returns the moonstone system for direct manipulation
func (g *GameState) GetMoonstoneSystem() *MoonstoneSystem {
	if g.moonstoneSystem == nil {
		g.moonstoneSystem = NewMoonstoneSystem()
	}

	return g.moonstoneSystem
}

// GetMoonstones returns all moonstone locations and states - delegation to MoonstoneSystem
func (g *GameState) GetMoonstones() []MoonstoneLocation {
	if g.moonstoneSystem == nil {
		g.moonstoneSystem = NewMoonstoneSystem()
	}

	return g.moonstoneSystem.GetMoonstonesFromSystem()
}

// GetMoonstoneIndexForMoonPhase returns the moonstone index for current moon phase - delegation to MoonstoneSystem
func (g *GameState) GetMoonstoneIndexForMoonPhase(phase int) int {
	if g.moonstoneSystem == nil {
		g.moonstoneSystem = NewMoonstoneSystem()
	}

	return g.moonstoneSystem.GetMoonstoneIndexForMoonPhase(phase)
}

// MoongateDestination represents where a moongate will take you
type MoongateDestination struct {
	Phase    datetime.MoonPhase  // Captured selected destination identity, never recomputed on arrival.
	X, Y     uint8               // Destination coordinates
	Map      uint8               // Destination map (0=overworld, 1-32=towns, 0xFF=inventory)
	Level    uint8               // Destination level/floor
	Location references.Location // Destination location reference
}

// MoongateTravelKind distinguishes spatial moonstone travel from the Spirituality workflow.
type MoongateTravelKind uint8

const (
	// MoongateTravelToMoonstone selects normal spatial moonstone travel.
	MoongateTravelToMoonstone MoongateTravelKind = iota
	// MoongateTravelToSpirituality selects the midnight shrine workflow.
	MoongateTravelToSpirituality
)

// MoongateTravelRoute is the detached current-time route selected before travel begins.
type MoongateTravelRoute struct {
	Kind        MoongateTravelKind
	Destination MoongateDestination
	ShrineIndex int
}

// GetMoongateTravelRoute returns the current semantic moongate route.
func (g *GameState) GetMoongateTravelRoute() (MoongateTravelRoute, error) {
	if g.moonstoneSystem == nil {
		g.moonstoneSystem = NewMoonstoneSystem()
	}

	return g.moonstoneSystem.GetMoongateTravelRoute(&g.DateTime)
}

// GetMoongateDestinationForPhase resolves a Vas Rel Por destination from the
// selected phase without applying the current moons or midnight shrine override.
func (g *GameState) GetMoongateDestinationForPhase(phase datetime.MoonPhase) (MoongateDestination, error) {
	if g.moonstoneSystem == nil {
		g.moonstoneSystem = NewMoonstoneSystem()
	}

	return g.moonstoneSystem.GetMoongateDestinationForPhase(phase)
}

// AreMoongatesVisible reports whether the authoritative runtime stage is nonzero.
func (g *GameState) AreMoongatesVisible() bool {
	return g.moongateStage > MoongateStageHidden
}

// GetMoongateVisibilityLevel converts the authoritative stage to 0.0 through 1.0.
func (g *GameState) GetMoongateVisibilityLevel() float32 {
	return float32(g.moongateStage) / float32(MoongateStageFull)
}

// GetActiveMoongatePositions returns light-source positions while stage is nonzero.
func (g *GameState) GetActiveMoongatePositions() []references.Position {
	if g.moonstoneSystem == nil {
		g.moonstoneSystem = NewMoonstoneSystem()
	}
	if g.moongateStage == MoongateStageHidden {
		return []references.Position{}
	}

	location := g.MapState.PlayerLocation

	return g.moonstoneSystem.GetActiveMoongatePositions(location.Location, location.Floor)
}

// HasActiveMoongateAtPosition checks stage and placed moonstone source state.
func (g *GameState) HasActiveMoongateAtPosition(pos *references.Position) bool {
	if g.moonstoneSystem == nil {
		g.moonstoneSystem = NewMoonstoneSystem()
	}

	location := g.MapState.PlayerLocation

	return g.moongateStage > MoongateStageHidden &&
		g.moonstoneSystem.CanCreateMoongateAt(location.Location, location.Floor, *pos)
}

// Pure logic methods - moved from GameState for better separation of concerns

// GetMoonstonesFromSystem returns all moonstone locations and states
func (ms *MoonstoneSystem) GetMoonstonesFromSystem() []MoonstoneLocation {
	return ms.Moonstones
}

// GetMoonstoneIndexForMoonPhase returns the moonstone index for given moon phase
func (ms *MoonstoneSystem) GetMoonstoneIndexForMoonPhase(phase int) int {
	if phase >= 0 && phase <= 7 {
		return phase // Direct 1:1 mapping in original Ultima V
	}

	return 0 // Default to first moonstone
}

// GetActiveMoongatePositions returns placed source gates on one map and floor.
func (ms *MoonstoneSystem) GetActiveMoongatePositions(location references.Location, floor references.FloorNumber) []references.Position {
	var activePositions []references.Position

	for _, moonstone := range ms.Moonstones {
		if moonstone.Map == uint8(location) && moonstone.Level == uint8(floor) && moonstone.CanCreateMoongate() {
			position := references.Position{X: references.Coordinate(moonstone.X), Y: references.Coordinate(moonstone.Y)}
			activePositions = append(activePositions, position)
		}
	}

	return activePositions
}

// MoongateSourceIndex returns a placed source gate matching full world identity.
func (ms *MoonstoneSystem) MoongateSourceIndex(location references.Location, floor references.FloorNumber, pos references.Position) int {
	for index, moonstone := range ms.Moonstones {
		if moonstone.Map == uint8(location) && moonstone.Level == uint8(floor) &&
			moonstone.X == uint8(pos.X) && moonstone.Y == uint8(pos.Y) && moonstone.CanCreateMoongate() {
			return index
		}
	}

	return -1
}

// CanCreateMoongateAt reports whether a placed moonstone can source a gate at the full identity.
func (ms *MoonstoneSystem) CanCreateMoongateAt(location references.Location, floor references.FloorNumber, pos references.Position) bool {
	return ms.MoongateSourceIndex(location, floor, pos) >= 0
}

// GetMoongateTravelRoute returns the semantic current-time route for moongate travel.
// Based on original gate travel and gate entry functions
// The midnight Spirituality path is not a spatial destination.
// See ref:MOON-0004.
func (ms *MoonstoneSystem) GetMoongateTravelRoute(dateTime *datetime.UltimaDate) (MoongateTravelRoute, error) {
	currentPhase := dateTime.GetMoongateMoonPhase()
	hour := int(dateTime.Hour)
	minute := int(dateTime.Minute)

	// The current midnight route selects Spirituality during hour zero
	// before minute ten; original coordinate precedence is separately scoped
	// in the public travel rule. ref:MOON-0005
	if hour == 0 && minute < 10 {
		return MoongateTravelRoute{
			Kind:        MoongateTravelToSpirituality,
			ShrineIndex: SpiritualityShrineIndex,
		}, nil
	}

	// Normal gate travel selects the first moon before noon and the second
	// moon afterward. ref:MOON-0005
	destination, err := ms.GetMoongateDestinationForPhase(currentPhase)
	if err != nil {
		return MoongateTravelRoute{}, err
	}

	return MoongateTravelRoute{
		Kind:        MoongateTravelToMoonstone,
		Destination: destination,
		ShrineIndex: -1,
	}, nil
}

// GetMoongateDestinationForPhase resolves the current movable moonstone record
// for an explicitly selected phase.
func (ms *MoonstoneSystem) GetMoongateDestinationForPhase(phase datetime.MoonPhase) (MoongateDestination, error) {
	phaseIndex := phase.ToMoonstoneIndex()

	// Check if the destination moonstone has been found
	if phaseIndex < 0 || phaseIndex >= len(ms.Moonstones) {
		return MoongateDestination{}, fmt.Errorf("invalid moon phase index: %d", phaseIndex)
	}

	moonstone := ms.Moonstones[phaseIndex]

	// A carried moonstone has no world destination. Reject travel to it
	// without a displayed failure in ExecuteMoongateTravel. ref:MOON-0005
	if moonstone.Map == 0xFF || moonstone.IsInInventory() {
		return MoongateDestination{}, fmt.Errorf("moonstone %d is in inventory, not accessible for travel", phaseIndex)
	}

	// Determine destination location based on map number
	var location references.Location
	if moonstone.Map == 0 {
		location = references.BritanniaUnderworld // Overworld
	} else {
		// Town maps (1-32) - map to appropriate town location
		// This would need expansion for specific town mappings
		location = references.Location(moonstone.Map)
	}
	if location.GetMapType() == references.DungeonMapType && !references.IsDungeonCoordinate(
		references.FloorNumber(moonstone.Level),
		references.Position{X: references.Coordinate(moonstone.X), Y: references.Coordinate(moonstone.Y)},
	) {
		return MoongateDestination{}, fmt.Errorf(
			"moonstone %d has invalid dungeon destination floor %d at (%d,%d)",
			phaseIndex,
			moonstone.Level,
			moonstone.X,
			moonstone.Y,
		)
	}

	return MoongateDestination{
		Phase:    phase,
		X:        moonstone.X,
		Y:        moonstone.Y,
		Map:      moonstone.Map,
		Level:    moonstone.Level,
		Location: location,
	}, nil
}
