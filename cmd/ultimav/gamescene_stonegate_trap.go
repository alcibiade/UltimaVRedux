package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

// =============================================================================
// Stonegate Volcano Trap Visual Sequence
// Stonegate's trap sequence follows ref:ENV-0002.
// =============================================================================

// Stonegate trap timing constants
const (
	stonegateTrapMsgMs      = 800  // "A TRAPDOOR!" message duration
	stonegateFadeMs         = 1000 // Fade to/from black (concurrent with message)
	stonegateDeathFlashMs   = 300  // Flash duration per character death
	stonegateDeathDelayMs   = 500  // Delay between character deaths
	stonegateDeathIntensity = 0.7  // Flash intensity for death
)

// StartStonegateTrapWorkflow presents a detached, already committed consequence.
// REDUX DEVIATION: ENV-0002: original trap processing interleaves blocking effects and damage.
// Redux commits at the town hazard checkpoint, then blanks the viewport and
// gently reveals the lava with per-member flashes before the death handoff.
func (g *GameScene) StartStonegateTrapWorkflow(outcome game_state.StonegateTrapOutcome) {
	// Prevent double-triggering
	if g.gameState.EventQueueManager.HasWorkflow("stonegate_trap_workflow") {
		return
	}
	if len(outcome.KilledMembers) == 0 {
		panic("Stonegate presentation requires a committed party-wipe outcome")
	}

	logging.Debugf(logging.CategoryGameState, "Starting Stonegate volcano trap workflow")

	hideMapUnits, hideQuestItems, hideItemStacks := g.hideMapUnits, g.hideQuestItems, g.hideItemStacks
	var overlay *widgets.OverlayDialog
	// Hide the committed terrain immediately, including the first draw before
	// the workflow's next update. The cinematic never needs a mutable old map.
	g.gameState.SystemCallbacks.Visual.SetMapFadeLevel(1)

	g.gameState.EventQueueManager.Workflow("stonegate_trap_workflow").
		Priority(event_queue.PriorityCritical).
		Sequential().BlockInput().
		First(func() {
			overlay = widgets.NewOverlayDialog("A TRAPDOOR!").
				AsOverlayFeedback().
				WithExternallyManagedDuration(stonegateTrapMsgMs).
				MirrorToOutput(g.output).
				Build()

			g.installTimedOverlay(overlay)

			g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundTrapTrigger)
		}).
		Wait(stonegateFadeMs).
		Then(func() {
			g.HideNonAvatarEntities()
			logging.Debugf(
				logging.CategoryGameState,
				"Stonegate: Floor flooded with lava, killing %d characters",
				len(outcome.KilledMembers),
			)
		}).Animate(stonegateFadeMs, func(progress float32) {
		g.gameState.SystemCallbacks.Visual.SetMapFadeLevel(float64(1.0 - progress))
	}).
		Then(func() {
			g.gameState.SystemCallbacks.Visual.SetMapFadeLevel(0.0)
		}).
		Wait(300).ThenSequence(
		func(deathQ *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
			for _, member := range outcome.KilledMembers {
				deathQ.Then(func() {
					g.gameState.SystemCallbacks.Audio.PlaySoundEffect(game_state.SoundDeath)
					logging.Debugf(logging.CategoryGameState, "Stonegate: Character %d killed", member.After.Slot)
				}).ThenSequence(g.thenVolcanoDeathFlash()).Wait(stonegateDeathDelayMs)
			}

			return deathQ
		}).
		Wait(300).
		Then(func() {
			g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
			logging.Debugf(logging.CategoryGameState, "Stonegate: Chaining to death resurrection workflow")
		}).If(
		func() bool {
			return !g.gameState.EventQueueManager.HasWorkflow(deathResurrectionWorkflowName)
		},
		func(deathQ *event_queue.SequentialEventBuilder) {
			deathQ.HandoffTo(
				deathResurrectionWorkflowName,
				event_queue.PriorityCritical,
				g.buildDeathResurrectionWorkflow,
			)
		},
	).Finally(func() {
		g.gameState.SystemCallbacks.Visual.SetMapFadeLevel(0)
		g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
		g.hideMapUnits, g.hideQuestItems, g.hideItemStacks = hideMapUnits, hideQuestItems, hideItemStacks
		if overlay != nil && g.timedOverlayDialog == overlay {
			g.clearTimedOverlay()
		}
	}).
		Start()
}

// thenVolcanoDeathFlash creates an animated white flash effect for volcano deaths.
// Uses triangle wave pattern: intensity rises to peak then fades back to 0.
// Each character death receives its own noise burst (ref:ENV-0002).
func (g *GameScene) thenVolcanoDeathFlash() func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		return q.
			Then(func() {
				// Set flash color to white for death effect
				g.gameState.SystemCallbacks.Visual.SetScreenFlashColor(1.0, 1.0, 1.0)
			}).
			Animate(stonegateDeathFlashMs, func(progress float32) {
				// Triangle wave: 0 -> max -> 0
				var intensity float64
				if progress < 0.5 {
					intensity = stonegateDeathIntensity * float64(progress*2.0)
				} else {
					intensity = stonegateDeathIntensity * float64(2.0-progress*2.0)
				}
				g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(intensity)
			}).
			Then(func() {
				// Ensure flash is cleared after animation
				g.gameState.SystemCallbacks.Visual.SetScreenFlashIntensity(0)
			})
	}
}
