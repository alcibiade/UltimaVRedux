package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// outdoorActorContinuation retains monster_main's stack-local cursor while
// outwar or a transporting whirlpool owns control. A source pointer identifies
// a pending contact; an encounter ID binds it to the actual tactical entry.
// Explicit player attacks do not manufacture an actor-phase cursor.
type outdoorActorContinuation struct {
	beforeSlot        int
	world             references.World
	source            *map_units.NPCEnemy
	encounterID       CombatEncounterID
	ready             bool
	scanned           bool
	waterfall         *waterfallCascade
	waterfallReturned bool
}

func (g *GameState) suspendOutdoorActorPhase(enemy *map_units.NPCEnemy) *outdoorActorContinuation {
	if g.outdoorActorContinuation != nil {
		panic("outdoor actor phase already suspended")
	}
	phase := &outdoorActorContinuation{
		beforeSlot: enemy.MapUnitDetails().NPCNum,
		world:      references.World(g.MapState.PlayerLocation.Floor),
		source:     enemy,
	}
	if cascade := g.activeWaterfall; cascade != nil && cascade.awaitingActors {
		phase.waterfall = cascade
	}
	g.outdoorActorContinuation = phase

	return phase
}

func (g *GameState) discardOutdoorActorContinuation() {
	if phase := g.outdoorActorContinuation; phase != nil {
		g.cancelWaterfallCascade(phase.waterfall)
	}
	g.outdoorActorContinuation = nil
}

func (g *GameState) cancelOutdoorActorPhase() {
	if phase := g.outdoorActorContinuation; phase != nil && g.pendingOverworldEncounter == phase.source {
		g.pendingOverworldEncounter = nil
	}
	g.discardOutdoorActorContinuation()
	// These callers retain their own stack-local work around monster_main.
	// Cancellation/death must not run their next phase or command tail.
	g.activeSlowMovement = nil
	g.activeShipRepair = nil
}

func (g *GameState) finishOutdoorCombatReturn(origin *OverworldEncounterOrigin, encounterID CombatEncounterID, _ CombatEndReason) {
	// Outdoor encounter return does not depend solely on victory. A defeat
	// result can mean living PCs fled while hostiles remained; only actual party
	// death or a terminal handoff prevents outdoor continuation. ref:ENEMY-0008
	if g.PartyState.IsAllPartyDead() || g.finaleHandoffPending {
		g.cancelOutdoorActorPhase()

		return
	}
	if phase := g.outdoorActorContinuation; phase != nil {
		if origin == nil || phase.source != origin.SourceEnemy || phase.encounterID != encounterID {
			g.cancelOutdoorActorPhase()

			return
		}
		// EndCombat has restored the world and removed/replaced only the source.
		// Resume at EndCombat's stable boundary, or through the pending-contact
		// drain after an owning combat workflow has released input.
		phase.ready = true

		return
	}
	if origin != nil {
		g.resumeShipRepairAfterCombat()
	}
}

func (g *GameState) resumeOutdoorActorPhase() {
	phase := g.outdoorActorContinuation
	if phase == nil || g.ShouldDeferOverworldContact() {
		return
	}
	// Reject a replaced waterfall landing before admitting even the first
	// queued battle, not just when its lower-slot continuation returns.
	if !g.acceptWaterfallActorReturn(phase) {
		g.cancelOutdoorActorPhase()

		return
	}
	if !phase.ready {
		return
	}
	if !phase.scanned {
		g.continueOutdoorActorPhase(phase)
	}
	if g.outdoorActorContinuation == phase && phase.scanned && !g.ShouldDeferOverworldContact() {
		g.outdoorActorContinuation = nil
		g.resumeShipRepairAfterCombat()
	}
}

// continueOutdoorActorPhase also runs inside the transporting whirlpool's
// workflow, after init_out. It never repeats generation, cadence, or player work.
func (g *GameState) continueOutdoorActorPhase(phase *outdoorActorContinuation) {
	if g.outdoorActorContinuation != phase {
		return
	}
	if g.PartyState.IsAllPartyDead() || g.finaleHandoffPending ||
		g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType ||
		references.World(g.MapState.PlayerLocation.Floor) != phase.world || !g.acceptWaterfallActorReturn(phase) {
		g.cancelOutdoorActorPhase()

		return
	}
	g.outdoorActorContinuation = nil
	g.GetCurrentLargeMapNPCAIController().ContinueTurnAfterAttack(phase.beforeSlot, g.checkOutdoorActorAttack)
	if g.PartyState.IsAllPartyDead() || g.finaleHandoffPending {
		g.cancelWaterfallCascade(phase.waterfall)
		g.cancelOutdoorActorPhase()

		return
	}
	if g.outdoorActorContinuation == nil && g.activeShipRepair != nil && g.activeShipRepair.awaitingCombat {
		// Lower-slot shots can still own presentation. Finish this repair cycle
		// once, after the entire phase and its effects, rather than per battle.
		phase.scanned = true
		phase.ready = true
		g.outdoorActorContinuation = phase
	}
}
