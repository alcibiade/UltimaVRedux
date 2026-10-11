package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// IsSailing returns true if currently in automatic sailing mode
func (g *GameState) IsSailing() bool {
	return g.isSailing
}

// StartAutomaticSailing begins automatic sailing mode for frigates with raised sails.
// The initiating direction becomes the ship's heading before the first cadence phase.
// Returns true if sailing started successfully, false if requirements not met
//
// Requirements:
// - Must be on a frigate
// - Sails must be raised
// - Not already sailing
func (g *GameState) StartAutomaticSailing(direction references.Direction) bool {
	// Check if already sailing
	if g.isSailing {
		logging.Debug(logging.CategoryGameState, "Sailing: Already in sailing mode")

		return false
	}

	// Get vehicle details
	vehicle := g.PartyVehicle.GetVehicleDetails()

	// Validate frigate requirement
	if vehicle.VehicleType != references.FrigateVehicle {
		logging.Debug(logging.CategoryGameState, "Sailing: Not on a frigate (vehicle type: %v)", vehicle.VehicleType)

		return false
	}

	// Validate sails raised requirement
	if vehicle.SailState != map_units.SailRaised {
		logging.Debug(logging.CategoryGameState, "Sailing: Sails not raised (sail state: %v)", vehicle.SailState)

		return false
	}
	if direction == references.NoneDirection {
		return false
	}

	// All requirements met - start sailing
	vehicle.SetPartyVehicleDirection(direction)
	g.isSailing = true
	logging.Debug(logging.CategoryGameState, "Sailing: Started automatic sailing mode")

	return true
}

// StopAutomaticSailing ends automatic sailing mode without changing sail state.
// Safe to call even if not currently sailing (idempotent)
func (g *GameState) StopAutomaticSailing() {
	if g.isSailing {
		logging.Debug(logging.CategoryGameState, "Sailing: Stopped automatic sailing mode")
	}
	g.isSailing = false
}

// CompleteAutomaticSailingInterruption returns to command input without an extra
// player update. The next command owns provision and duration effects. ref:PROV-0002.
func (g *GameState) CompleteAutomaticSailingInterruption() {
	g.StopAutomaticSailing()
}

// AutomaticSailingWaitPhases returns the source n%3 wait threshold. Calm has no
// movement threshold, so its second return value is false.
func (g *GameState) AutomaticSailingWaitPhases(direction references.Direction) (int, bool) {
	if g.IsWindCalm() {
		return 0, false
	}

	return g.windSystem.GetFrigateSailingSpeed(direction) % 3, true
}

// CompleteAutomaticSailingWait owns one source-style wait phase. This seam is
// intentionally separate from ordinary movement completion because HMS Cape
// changes wait timing and actor cadence rather than movement completion.
func (g *GameState) CompleteAutomaticSailingWait() {
	if !g.isSailing {
		panic("automatic sailing wait requires active sailing")
	}

	minutes := DefaultLargeMapMinutesPerTurn
	processWorldActors := true
	if g.IsHMSCapeActivated() {
		minutes = 1
		// Outdoor command input toggles static fast before deciding whether
		// this Cape wait receives outdoor actor phase, so the first wait skips it.
		// See ref:SHIP-0001.
		g.hmsCapeAutomaticWaitActorSkip = !g.hmsCapeAutomaticWaitActorSkip
		processWorldActors = !g.hmsCapeAutomaticWaitActorSkip
	}

	g.AdvanceGameplayMinutes(minutes)
	if processWorldActors {
		g.processNonCombatWorldActorPhase()
	}
}
