package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Presentation delays do not charge gameplay minutes. ref:ENV-0003.
const (
	waterfallWorkflowName   = "waterfall_hazard"
	waterfallStepDelayMs    = 255
	waterfallFadeDurationMs = 500

	// Underworld entry point after the original two southward moves.
	waterfallUnderworldEntryX = 54  // 0x36
	waterfallUnderworldEntryY = 138 // 0x8a; the waterfall tile itself is at Y=137.
)

// IsWaterfallHazardActive includes the actor/contact interlude between workflow
// instances as well as pending, deferred and finalizing queue ownership.
func (g *GameState) IsWaterfallHazardActive() bool {
	return g.activeWaterfall != nil || g.EventQueueManager.HasWorkflow(waterfallWorkflowName)
}

func (g *GameState) startWaterfallHazard(afterFirstFall func()) bool {
	// Accepted deferred commands still complete during queue shutdown. Reject
	// only new hazard ownership: a rejected Start would not run its finalizer,
	// and the manager's transient-owner cleanup has already happened.
	if g.EventQueueManager.IsShutdown() {
		return false
	}
	if g.IsWaterfallHazardActive() {
		return true
	}
	if !g.isWaterfallLandingEligible() {
		return false
	}
	cascade := &waterfallCascade{
		landing:        g.MapState.PlayerLocation,
		visited:        make(map[references.PlayerLocation]struct{}),
		afterFirstFall: afterFirstFall,
	}
	g.activeWaterfall = cascade
	// Hazard ownership ends automatic travel, not the already owed command tail.
	g.StopAutomaticSailing()

	return g.startWaterfallWorkflow(cascade)
}

func (g *GameState) startWaterfallWorkflow(cascade *waterfallCascade) bool {
	if g.EventQueueManager.IsShutdown() {
		g.cancelWaterfallCascade(cascade)

		return false
	}
	if cascade == nil || g.activeWaterfall != cascade || g.EventQueueManager.HasWorkflow(waterfallWorkflowName) {
		panic("waterfall workflow requires its sole retained cascade and an available queue name")
	}
	// A real handoff must reach the sequence's normal end before its finalizer is
	// allowed to retain the cascade. Shutdown never resumes an abandoned owner.
	retainForActors := false
	// Successful movement may already own a high-priority rough-sea hit. This
	// normal-priority workflow waits behind that owner; never request the hit again
	// or poll an equal-priority child from inside this sequence. ref:ENV-0003.
	g.EventQueueManager.Workflow(waterfallWorkflowName).
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {}).
		ThenSequence(g.thenWaterfallCascade(cascade)).
		Then(func() {
			// If eligibility disappeared before the first step without a takeover,
			// the already consumed command still owes its ordinary actor phase.
			g.completeFirstWaterfallActorTail(cascade)
			retainForActors = g.activeWaterfall == cascade && !cascade.interrupted && cascade.awaitingActors
		}).
		Finally(func() {
			// Each physical workflow releases presentation/input once. The logical
			// owner survives only an explicit actor handoff, never cancellation.
			if cascade.fadeOwned {
				g.SystemCallbacks.Visual.SetMapFadeLevel(0)
				cascade.fadeOwned = false
			}
			if cascade.flashOwned {
				g.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
				cascade.flashOwned = false
			}
			if !retainForActors {
				g.cancelWaterfallCascade(cascade)
			}
		}).
		Start()
	// Queue construction reads the injected clock, and activation can invoke
	// callbacks. Reentrant shutdown must not report a rejected/cancelled Start
	// as an accepted continuation or leave its logical state published.
	if g.EventQueueManager.IsShutdown() {
		g.cancelWaterfallCascade(cascade)

		return false
	}

	return true
}

// thenWaterfallDamageCheck builds feedback for the Dexterity check.
// A transformed d30 at least Dexterity fails. ref:ENV-0003
func (g *GameState) thenWaterfallDamageCheck(cascade *waterfallCascade) event_queue.Sequence {
	return func(sequence *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		var effects *PartyStateBuilder

		return sequence.Then(func() {
			if !g.waterfallOwnsLanding(cascade) {
				return
			}
			damageTargets := g.performWaterfallDexterityChecks()
			if len(damageTargets) == 0 {
				return
			}
			// Keep the existing ordered checks and fixed-damage builder RNG calls.
			// Check and damage commit together, with no interruptible gap between them.
			effects = NewPartyStateBuilder(g).WithFixedDamage(1).ToCharacters(damageTargets).FlashGray()
			effects.applyDamage()
			effects.setupEffects()
			cascade.flashOwned = true
			if g.PartyState.IsAllPartyDead() {
				cascade.interrupted = true
				g.checkForPartyWipe()
			}
		}).If(func() bool {
			return effects != nil && g.waterfallOwnsLanding(cascade)
		}, func(feedback *event_queue.SequentialEventBuilder) {
			feedback.Animate(flashStandardDurationMs, func(progress float32) {
				if g.waterfallOwnsLanding(cascade) {
					effects.animateEffects(progress, flashStandardDurationMs)
				}
			}).Then(func() {
				g.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
				cascade.flashOwned = false
			})
		})
	}
}

// thenUnderworldTransition is owned by the waterfall workflow and installs no
// Finally. Its canonical transition updates the landing before the next review.
func (g *GameState) thenUnderworldTransition(cascade *waterfallCascade) event_queue.Sequence {
	return func(sequence *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		if !g.waterfallOwnsLanding(cascade) || g.MapState.PlayerLocation.Floor != references.MainFloor ||
			g.MapState.PlayerLocation.Position.X != waterfallUnderworldEntryX ||
			g.MapState.PlayerLocation.Position.Y != waterfallUnderworldEntryY {
			return sequence
		}

		return sequence.Then(func() {
			if g.waterfallOwnsLanding(cascade) {
				g.SystemCallbacks.UI.ShowActionFeedback("Falling into underworld!!", true)
			}
		}).Animate(waterfallFadeDurationMs, func(progress float32) {
			if g.waterfallOwnsLanding(cascade) {
				cascade.fadeOwned = true
				g.SystemCallbacks.Visual.SetMapFadeLevel(float64(progress))
			}
		}).Then(func() {
			if !g.waterfallOwnsLanding(cascade) {
				return
			}
			g.TransitionToLocation(references.BritanniaUnderworld, references.Basement,
				g.MapState.PlayerLocation.Position, TransitionFullLocation, false)
			cascade.landing = g.MapState.PlayerLocation
		}).Animate(waterfallFadeDurationMs, func(progress float32) {
			if g.waterfallOwnsLanding(cascade) {
				g.SystemCallbacks.Visual.SetMapFadeLevel(1.0 - float64(progress))
			}
		}).Then(func() {
			if cascade.fadeOwned {
				g.SystemCallbacks.Visual.SetMapFadeLevel(0)
				cascade.fadeOwned = false
			}
		})
	}
}

// performWaterfallDexterityChecks returns characters whose transformed
// d30 meets or exceeds Dexterity; each failed check costs one HP. ref:ENV-0003
func (g *GameState) performWaterfallDexterityChecks() []int {
	var damageTargets []int

	g.PartyState.ForEachCharacter(func(i int, member *party_state.PlayerCharacter) {
		// Skip dead or inactive party members (original: status != 'D')
		if member.IsDead() || member.PartyMap != party_state.PartyMapActive {
			return
		}

		// Waterfall damage check calls transformed d30; fail if its nonuniform result is >= DEX.
		// Note: This is OPPOSITE of swamp poison which uses roll > DEX
		// See ref:RNG-0001.
		dexRoll := g.RollOriginalD30()
		if dexRoll >= int(member.Dexterity) {
			damageTargets = append(damageTargets, i)
		}
	})

	return damageTargets
}
