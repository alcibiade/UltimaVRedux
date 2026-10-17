package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// waterfallCascade is the logical owner across physical workflow instances. The
// first fall may yield input to its owed actor/contact phase; its visited set and
// exact-once tail survive that handoff. None of this transient state is saved.
type waterfallCascade struct {
	landing        references.PlayerLocation
	visited        map[references.PlayerLocation]struct{}
	afterFirstFall func()
	actorTailDone  bool
	awaitingActors bool
	interrupted    bool
	fadeOwned      bool
	flashOwned     bool
}

func (g *GameState) isWaterfallLandingEligible() bool {
	if g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType ||
		g.CombatScene != nil || g.finaleHandoffPending || g.PartyState.IsAllPartyDead() {
		return false
	}
	tile := g.GetTileInDirectionWrapped(&g.MapState.PlayerLocation.Position, references.Down)

	return tile != nil && tile.IsWaterfall()
}

// A foreign relocation/terminal owner invalidates the remaining descent. Never
// drag the party back to a cached position or apply old-map damage after takeover.
func (g *GameState) waterfallOwnsLanding(cascade *waterfallCascade) bool {
	if g.activeWaterfall != cascade || cascade.interrupted || g.MapState.PlayerLocation != cascade.landing ||
		g.CombatScene != nil || g.finaleHandoffPending || g.PartyState.IsAllPartyDead() {
		cascade.interrupted = true

		return false
	}

	return true
}

func (g *GameState) cancelWaterfallCascade(cascade *waterfallCascade) {
	if cascade == nil {
		return
	}
	cascade.interrupted = true
	cascade.afterFirstFall = nil
	if g.activeWaterfall == cascade {
		g.activeWaterfall = nil
	}
}

// reconcileWaterfallCombatEntry runs at the validated combat commit boundary.
// A foreign encounter must discard the old waterfall cursor before its pending
// contact could intercept tactical input ahead of the scene's resume hook.
func (g *GameState) reconcileWaterfallCombatEntry(combat *CombatState) {
	cascade := g.activeWaterfall
	if cascade == nil {
		return
	}
	phase := g.outdoorActorContinuation
	if cascade.awaitingActors && phase != nil && phase.waterfall == cascade &&
		combat.OverworldOrigin != nil && phase.source == combat.OverworldOrigin.SourceEnemy &&
		g.waterfallOwnsLanding(cascade) {
		return
	}
	if phase != nil && phase.waterfall == cascade {
		g.cancelOutdoorActorPhase()
	} else {
		g.cancelWaterfallCascade(cascade)
	}
}

// ResumeWaterfallContinuation is the pre-input continuation checkpoint, AFTER
// pending outdoor contacts, tactical scheduling and NPC auto-talk. It returns
// true when surface input must wait (including the frame that resumes the fall),
// but false during owned tactical combat so its ordinary input remains usable.
// It never advances a command/player/actor cycle or resolves contact itself.
func (g *GameState) ResumeWaterfallContinuation() bool {
	if g.EventQueueManager.IsShutdown() {
		g.cancelWaterfallCascade(g.activeWaterfall)

		return false
	}
	cascade := g.activeWaterfall
	if cascade == nil {
		return g.EventQueueManager.HasWorkflow(waterfallWorkflowName)
	}
	if cascade.interrupted || g.PartyState.IsAllPartyDead() || g.finaleHandoffPending {
		g.cancelWaterfallCascade(cascade)

		return false
	}
	if g.CombatScene != nil {
		phase := g.outdoorActorContinuation
		if !cascade.awaitingActors || phase == nil || phase.waterfall != cascade ||
			phase.encounterID != g.CombatScene.EncounterID || g.CombatScene.OverworldOrigin == nil ||
			phase.source != g.CombatScene.OverworldOrigin.SourceEnemy {
			g.cancelWaterfallCascade(cascade)
		}

		return false
	}
	// A transporting actor can be between its position commit and its own return
	// boundary. Review its destination only once that workflow has released input.
	if g.EventQueueManager.HasWorkflow(waterfallWorkflowName) || g.ShouldDeferOverworldContact() {
		return true
	}
	if !g.waterfallOwnsLanding(cascade) {
		g.cancelWaterfallCascade(cascade)

		return true
	}
	if g.HasPendingOverworldContact() {
		return true
	}
	if !cascade.awaitingActors {
		panic("waterfall continuation has neither a workflow nor an actor handoff")
	}
	cascade.awaitingActors = false
	if !g.isWaterfallLandingEligible() {
		g.cancelWaterfallCascade(cascade)

		return true
	}

	return g.startWaterfallWorkflow(cascade)
}

// completeFirstWaterfallActorTail commits the initiating command's one actor
// phase AFTER the first complete fall, BEFORE any automatic next fall. Mark it
// done before callbacks can suspend the phase or re-enter completion.
func (g *GameState) completeFirstWaterfallActorTail(cascade *waterfallCascade) {
	if cascade.actorTailDone || !g.waterfallOwnsLanding(cascade) {
		return
	}
	cascade.actorTailDone = true
	complete := cascade.afterFirstFall
	cascade.afterFirstFall = nil
	if complete == nil {
		return
	}
	// New outdoor cursors bind to this exact cascade while the phase executes.
	cascade.awaitingActors = true
	complete()
	if !g.waterfallOwnsLanding(cascade) {
		g.cancelWaterfallCascade(cascade)

		return
	}
	// Release this owner instead of waiting on children of any priority. This
	// includes on-foot whirlpool presentation, which retains no actor cursor.
	// Unrelated nonblocking/background work may cause a harmless handoff here;
	// Resume uses the interaction gate, never waits for all background work.
	cascade.awaitingActors = g.HasPendingOverworldContact() ||
		!g.EventQueueManager.HasOnlyWorkflow(waterfallWorkflowName)
}

// acceptWaterfallActorReturn is called only by the validated outdoor cursor at
// its stable return boundary. An owned whirlpool may legitimately relocate the
// party; unrelated replacement cannot move a retained waterfall's landing.
func (g *GameState) acceptWaterfallActorReturn(phase *outdoorActorContinuation) bool {
	cascade := phase.waterfall
	if cascade == nil {
		return true
	}
	if g.activeWaterfall != cascade || cascade.interrupted || !cascade.awaitingActors {
		return false
	}
	if phase.ready && !phase.waterfallReturned {
		if phase.source != nil && phase.source.Variant() == map_units.EnemyVariantWhirlpool {
			cascade.landing = g.MapState.PlayerLocation
		}
		phase.waterfallReturned = true
	}

	return g.waterfallOwnsLanding(cascade)
}

// thenWaterfallCascade inserts one complete asynchronous descent at runtime.
// Its final ThenSequence reviews the UPDATED landing only after forced steps,
// damage feedback and any canonical map transition have all completed. This is
// queue continuation, not recursive ordinary turn completion or a polling loop
// that starts independent workflows for successive falls. ref:ENV-0003.
func (g *GameState) thenWaterfallCascade(cascade *waterfallCascade) event_queue.Sequence {
	return func(sequence *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		if cascade.awaitingActors || !g.waterfallOwnsLanding(cascade) || !g.isWaterfallLandingEligible() {
			return sequence
		}
		if _, visited := cascade.visited[cascade.landing]; visited {
			// The last fall has already committed. Retain that authoritative landing,
			// finish the owed cycle, and return input instead of orbiting the map.
			g.SystemCallbacks.Message.AddRowStr("The waterfall current circles back. You regain control.")

			return sequence
		}
		cascade.visited[cascade.landing] = struct{}{}

		return sequence.Then(func() {
			if g.waterfallOwnsLanding(cascade) {
				g.SystemCallbacks.UI.ShowActionFeedback("F-A-L-L-S!!!", true)
				g.advanceWaterfallSouthStep(cascade)
			}
		}).Wait(waterfallStepDelayMs).
			Then(func() { g.advanceWaterfallSouthStep(cascade) }).
			ThenSequence(g.thenWaterfallDamageCheck(cascade)).
			ThenSequence(g.thenUnderworldTransition(cascade)).
			Then(func() { g.completeFirstWaterfallActorTail(cascade) }).
			ThenSequence(g.thenWaterfallCascade(cascade))
	}
}

// advanceWaterfallSouthStep is the narrow authority boundary for each actual
// forced step. It neither dispatches input nor owns ordinary command completion.
func (g *GameState) advanceWaterfallSouthStep(cascade *waterfallCascade) {
	if !g.waterfallOwnsLanding(cascade) {
		return
	}
	g.MovePlayerOneStep(references.Down)
	cascade.landing = g.MapState.PlayerLocation
}
