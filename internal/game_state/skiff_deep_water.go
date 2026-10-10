//nolint:mnd // Existing original-game damage die size is domain-specific.
package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// Visual effect parameters for rough seas damage
const (
	WaterShakePixels = 6.0 // Moderate camera shake
	WaterDurationMs  = 200 // Standard impact duration
)

// ProcessSkiffDeepWaterDamage handles damage to party members when skiff or carpet
// enters DEEP water (terrain type 1)
//
// Based on original Ultima V:
// - original outdoor rules - Damage trigger in DEEP water
// - original world-creature rules - Calls party damage for non-ships
// - original party-effect rules - Each party member takes 1-8 HP damage
//
// Original behavior:
//   - Triggers only in DEEP water (tile index 1), NOT SEA (2) or SHOALS (3)
//   - Both skiffs and carpets are affected
//   - Damages party members directly, not the vehicle
//   - Shows "Rough seas!" message
//   - Displays kapow visual effect
//   - Each alive party member takes random 1-8 HP damage
//
// See ref:SHIP-0003.
func (g *GameState) ProcessSkiffDeepWaterDamage(position *references.Position) {
	vehicle := g.PartyVehicle.GetVehicleDetails()

	// Only process for skiffs and carpets
	if vehicle.VehicleType != references.SkiffVehicle &&
		vehicle.VehicleType != references.CarpetVehicle {
		return
	}

	// Check if in DEEP water (terrain type 1)
	tile := g.GetCurrentLayeredMap().GetTopTile(position)
	if tile == nil || tile.Index != indexes.Water1 {
		return // Not DEEP water - no damage
	}

	// The successful-movement route calls this once per step, and this workflow
	// blocks further movement input until the sequence completes.
	// Trigger damage sequence via event queue
	// Pattern: message → visual effect → damage (original Ultima V sequence)
	g.EventQueueManager.Workflow("skiff_deep_water").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {
			g.SystemCallbacks.Message.AddRowStr("Rough seas!")
		}).ThenSequence(

		NewPartyStateBuilder(g).
			WithShake(WaterDurationMs, WaterShakePixels).
			FlashWater().
			ToAllParty().WithRandomDamage(1, 8).
			Build()).Then(func() {
		g.checkForPartyWipe()
	}).
		Wait(25). // Brief pause for impact realism
		Finally(func() {
			g.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
			g.SystemCallbacks.Visual.SetScreenShakeOffset(0, 0)
		}).
		Start()
}
