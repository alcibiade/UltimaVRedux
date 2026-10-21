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

	skiffDeepWaterWorkflowName = "skiff_deep_water"
)

// A terminal batch stays closed even if a higher-priority recovery workflow
// revives or replaces the party before this owner's remaining steps run.
type skiffDeepWaterBatch struct {
	terminal bool
}

// ProcessSkiffDeepWaterDamage handles damage to party members when skiff or carpet
// enters DEEP water (terrain type 1)
//
// Based on original Ultima V:
// - original outdoor rules - Damage trigger in DEEP water
// - original world-creature rules - Calls party damage for non-ships
// - original party-effect rules - Each party member takes 1-8 HP damage
//
// Damage rules (the current Redux caller is successful movement only):
//   - Triggers only in DEEP water (tile index 1), NOT SEA (2) or SHOALS (3)
//   - Both skiffs and carpets are affected
//   - Damages party members directly, not the vehicle
//   - Shows "Rough seas!" message
//   - Presents water flash and shake feedback
//   - Each alive party member takes random 1-8 HP damage
//
// See ref:SHIP-0003.
func (g *GameState) ProcessSkiffDeepWaterDamage(position *references.Position) {
	if !g.isSkiffDeepWaterDamageEligible(position) {
		return
	}

	// Retain every admitted hit. Input blocking alone cannot protect this name
	// against programmatic movement, and a same-priority child cannot preempt
	// automatic sailing. One higher-priority owner drains all owed hits.
	g.pendingSkiffDeepWaterHits++
	g.startPendingSkiffDeepWaterDamage()
}

func (g *GameState) startPendingSkiffDeepWaterDamage() {
	if g.pendingSkiffDeepWaterHits == 0 || g.EventQueueManager.HasWorkflow(skiffDeepWaterWorkflowName) {
		return
	}
	completed := false
	batch := &skiffDeepWaterBatch{}
	queue := g.EventQueueManager.Workflow(skiffDeepWaterWorkflowName).
		Priority(event_queue.PriorityHigh).
		Sequential().BlockInput().First(func() {})
	g.appendPendingSkiffDeepWaterDamage(queue, batch)
	queue.Then(func() { completed = true }).Finally(func() {
		if !completed && !batch.terminal {
			g.pendingSkiffDeepWaterHits = 0
		}
		g.ClearSkiffDeepWaterEffects()
	}).OnComplete(g.startPendingSkiffDeepWaterDamage).Start()
	// A request arriving after Finally, during input-release cleanup, belongs
	// to the next owner. OnComplete runs after the fixed name is unreserved;
	// it preserves that request without bypassing duplicate-name protection.
}

func (g *GameState) isSkiffDeepWaterDamageEligible(position *references.Position) bool {
	if position == nil || g.PartyState.IsAllPartyDead() {
		return false
	}
	vehicle := g.PartyVehicle.GetVehicleDetails()

	// Only process for skiffs and carpets
	if vehicle.VehicleType != references.SkiffVehicle &&
		vehicle.VehicleType != references.CarpetVehicle {
		return false
	}

	// Check if in DEEP water (terrain type 1)
	tile := g.GetCurrentLayeredMap().GetTopTile(position)

	return tile != nil && tile.Index == indexes.Water1
}

func (g *GameState) appendPendingSkiffDeepWaterDamage(queue *event_queue.SequentialEventBuilder, batch *skiffDeepWaterBatch) {
	queue.Then(func() {
		if batch.terminal || g.pendingSkiffDeepWaterHits <= 0 {
			panic("rough-seas hit requires a live batch and an admitted request")
		}
		// Consume before execution, never after a possible death handoff. An old
		// continuation must not decrement a new post-recovery batch's request.
		g.pendingSkiffDeepWaterHits--
	}).ThenSequence(g.skiffDeepWaterDamageSequence(func() {
		batch.terminal = true
		g.pendingSkiffDeepWaterHits = 0
	})).If(func() bool { return !batch.terminal && g.pendingSkiffDeepWaterHits > 0 }, func(next *event_queue.SequentialEventBuilder) {
		g.appendPendingSkiffDeepWaterDamage(next, batch)
	})
}

func (g *GameState) skiffDeepWaterDamageSequence(onTerminal func()) event_queue.Sequence {
	return func(queue *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		var hadLivingParty bool

		return queue.Then(func() {
			hadLivingParty = !g.PartyState.IsAllPartyDead()
			g.SystemCallbacks.Message.AddRowStr("Rough seas!")
		}).ThenSequence(
			NewPartyStateBuilder(g).
				WithShake(WaterDurationMs, WaterShakePixels).
				FlashWater().
				ToAllParty().WithRandomDamage(1, 8).
				Build()).Then(func() {
			if hadLivingParty && g.PartyState.IsAllPartyDead() {
				// Establish terminal authority before invoking a callback that can
				// preempt this owner and return with a resurrected/replaced party.
				onTerminal()
				g.checkForPartyWipe()
			}
		}).Wait(25)
	}
}

// ClearSkiffDeepWaterEffects is the inline workflow owner's cancellation cleanup.
func (g *GameState) ClearSkiffDeepWaterEffects() {
	g.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
	g.SystemCallbacks.Visual.SetScreenShakeOffset(0, 0)
}
