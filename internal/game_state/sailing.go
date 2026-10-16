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
// - Must be outdoors in exploration on a frigate
// - Sails must be raised
// - Not already sailing
func (g *GameState) StartAutomaticSailing(direction references.Direction) bool {
	// Check if already sailing
	if g.isSailing {
		logging.Debug(logging.CategoryGameState, "Sailing: Already in sailing mode")

		return false
	}

	if !g.automaticSailingEligible() || direction == references.NoneDirection {
		return false
	}

	// All requirements met - start sailing
	vehicle := g.PartyVehicle.GetVehicleDetails()
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
		g.automaticSailingStopEpoch++
	}
	g.isSailing = false
}

// AutomaticSailingStopEpoch identifies actual stops for presentation/input cleanup.
// Reading it neither changes gameplay nor hides a stale sailing flag.
func (g *GameState) AutomaticSailingStopEpoch() uint64 {
	return g.automaticSailingStopEpoch
}

// ContinueAutomaticSailing validates each wait/movement boundary. Invalidated
// eligibility clears authoritative state without spending time or randomness.
func (g *GameState) ContinueAutomaticSailing() bool {
	if !g.isSailing {
		return false
	}
	if !g.automaticSailingEligible() || g.PartyState.IsAllPartyDead() {
		g.StopAutomaticSailing()

		return false
	}

	return true
}

func (g *GameState) automaticSailingEligible() bool {
	if g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType ||
		g.CombatScene != nil || g.IsInCamping() || g.IsInMiniMap() ||
		g.IsFalling() || g.IsTeleportationActive() {
		return false
	}
	vehicle := g.PartyVehicle.GetVehicleDetails()

	return vehicle.VehicleType == references.FrigateVehicle && vehicle.SailState == map_units.SailRaised
}

// StopAutomaticSailingForCombat runs at the validated tactical-entry commit,
// including stationary and player-initiated entries. Normal return retains furl.
func (g *GameState) StopAutomaticSailingForCombat() {
	// REDUX DEVIATION: SHIP-0001: original tactical entry retains the party's
	// sails and sailing direction. Redux stops and furls for a manual decision
	// after combat, while preserving facing.
	g.StopAutomaticSailing()
	vehicle := g.PartyVehicle.GetVehicleDetails()
	if vehicle.VehicleType == references.FrigateVehicle {
		vehicle.SetSailState(map_units.SailFurled)
	}
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
	if !g.ContinueAutomaticSailing() {
		return
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
