//nolint:mnd // Existing original-game waterfall timing is domain-specific.
package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// Waterfall effect timing constants (based on original outdoor effect rules)
// See ref:ENV-0003.
const (
	// glide(2500,800,1,300) from original - timing placeholders
	waterfallGlideDurationMs   = 2500 // glide() first parameter
	waterfallMessageDurationMs = 1500 // "F-A-L-L-S!!!" display time

	// Underworld entry point after the original two southward moves.
	waterfallUnderworldEntryX = 54  // 0x36
	waterfallUnderworldEntryY = 138 // 0x8a; the waterfall tile itself is at Y=137.
)

// ProcessWaterfallHazard handles waterfall encounter with DEX-based damage
// Includes underworld transition at specific coordinates (54, 138)
//
// Triggered on waterfall tiles (0xd4-0xd7 = 212-215)
// Applies to all movement types (on foot, carpet, skiff)
// See ref:ENV-0003.
func (g *GameState) ProcessWaterfallHazard(position *references.Position) {
	// Waterfalls only exist on large maps (overworld/underworld)
	if g.MapState.PlayerLocation.Location.GetMapType() != references.LargeMapType {
		return
	}

	// Check tile BELOW the passed position (waterfall is below triggering tile)
	tile := g.GetTileInDirectionWrapped(position, references.Down)
	if tile == nil || !tile.IsWaterfall() {
		return // Not a waterfall
	}

	// Local helper to move player down one tile with large map wrapping
	movePlayerDownOneTile := func() {
		downPosition := g.MapState.PlayerLocation.Position.GetPositionDown()
		wrappedPosition := downPosition.GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
		g.SetPlayerPosition(*wrappedPosition)
	}

	// The successful-movement route calls this once per step, and this workflow
	// blocks further movement input until the sequence completes.
	// Start waterfall sequence via event queue
	g.EventQueueManager.Workflow("waterfall_hazard").
		Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {

		}).
		Then(func() {
			g.SystemCallbacks.UI.ShowActionFeedback(
				"F-A-L-L-S!!!", true,
			)

			movePlayerDownOneTile()
		},
		).
		Wait(255).
		Then(func() {
			movePlayerDownOneTile()
		}).ThenSequence(

		g.thenWaterfallGlideDuration()).
		Then(func() {

		}).ThenSequence(

		g.thenWaterfallDamageCheck()).ThenSequence(

		g.thenUnderworldTransition()).Finally(

		func() {
			// Guaranteed cleanup: ensure fade is cleared
			// g.SystemCallbacks.Visual.SetMapFadeLevel(0.0)
		}).
		Start()
}

// thenWaterfallGlideDuration handles glide timing (screen black)
// Original: glide(2500,800,1,300) - fade completed, now glide
// TODO Phase 4: Add smooth camera/position interpolation during glide
func (g *GameState) thenWaterfallGlideDuration() func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		// Wait while screen is black - "falling" duration
		q = q.Wait(waterfallGlideDurationMs)

		return q
	}
}

// thenWaterfallDamageCheck builds feedback for the Dexterity check.
// A transformed d30 at least Dexterity fails. ref:ENV-0003
func (g *GameState) thenWaterfallDamageCheck() func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		// Perform DEX checks to determine damage targets
		damageTargets := g.performWaterfallDexterityChecks()

		// Apply damage + gray flash if anyone took damage
		if len(damageTargets) == 0 {
			return q
		}

		// Apply 1 HP damage to failed DEX checks with gray flash
		// FlashGray() uses standard 200ms duration, 0.7 intensity (fall/impact pattern)
		q = NewPartyStateBuilder(g).
			WithFixedDamage(1).          // 1 HP damage (original)
			ToCharacters(damageTargets). // Only those who failed DEX check
			FlashGray().                 // Gray fall/impact flash
			Build()(q)

		return q
	}
}

// thenUnderworldTransition returns event queue modifier for underworld transition with visual effects
func (g *GameState) thenUnderworldTransition() func(*event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
	return func(q *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		// Check if we're at the underworld entry coordinates
		if g.MapState.PlayerLocation.Position.X != waterfallUnderworldEntryX ||
			g.MapState.PlayerLocation.Position.Y != waterfallUnderworldEntryY {
			return q // Not at underworld entry, continue normal flow
		}

		// Show underworld transition message
		q = q.Then(func() {
			g.SystemCallbacks.UI.ShowActionFeedback("Falling into underworld!!", true)
		})

		// Fade out with Animate (500ms)
		q = q.Animate(500, func(progress float32) {
			// Fade from visible (0.0) to black (1.0)
			g.SystemCallbacks.Visual.SetMapFadeLevel(float64(progress))
		})

		// Underworld transition (happens during black screen)
		q = q.Then(func() {
			g.TransitionToLocation(
				references.BritanniaUnderworld,
				references.Basement,
				g.MapState.PlayerLocation.Position,
				TransitionFullLocation,
				false, // No entry music
			)
		})

		// Fade in with Animate (500ms)
		q = q.Animate(500, func(progress float32) {
			// Fade from black (1.0) to visible (0.0)
			g.SystemCallbacks.Visual.SetMapFadeLevel(1.0 - float64(progress))
		})

		// Final cleanup
		q = q.Finally(func() {
			g.SystemCallbacks.Visual.SetMapFadeLevel(0.0)
		})

		return q
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
