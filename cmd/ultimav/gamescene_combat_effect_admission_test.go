package main

import (
	"fmt"
	"image"
	"slices"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/display"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestCombatFiniteEffectsProductionAdmissionAfterHitch(t *testing.T) {
	for _, rate := range []float64{1, 4} {
		for _, hitchMs := range []int64{50, 200, 3000} {
			t.Run(fmt.Sprintf("%gx/hitch=%dms", rate, hitchMs), func(t *testing.T) {
				testCombatFiniteEffectAdmission(t, rate, hitchMs)
			})
		}
	}
}

func TestCombatKapowAdmissionClockIsScopedToCombat(t *testing.T) {
	for _, combat := range []bool{false, true} {
		t.Run(fmt.Sprintf("combat=%t", combat), func(t *testing.T) {
			scene, _ := newCombatUseTestScene(t)
			if !combat {
				scene.gameState.TransitionFromCombatMap()
			}
			setCombatTimingTestRate(scene, 4)
			now := time.Unix(100, 0)
			scene.lastEffectsUpdate, scene.now = now, func() time.Time { return now }
			scene.gameState.ElapsedMs = 4000
			scene.playKapowAtPosition(references.Position{X: 4, Y: 5})
			now = now.Add(200 * time.Millisecond)
			// Keep the published timestamp fixed to distinguish admission time
			// from the independent cosmetic clock of a noncombat effect.
			scene.updateVisualEffects()
			require.Len(t, scene.visualEffects, 1)
			if combat {
				assert.Zero(t, scene.visualEffects[0].Elapsed)
				assert.Equal(t, 0.125, scene.visualEffects[0].Duration)
			} else {
				assert.Equal(t, 0.2, scene.visualEffects[0].Elapsed)
				assert.Equal(t, 0.5, scene.visualEffects[0].Duration)
			}
		})
	}
}

func testCombatFiniteEffectAdmission(t *testing.T, rate float64, hitchMs int64) {
	t.Helper()
	manager := display.GetManager()
	savedManager := *manager
	t.Cleanup(func() { *manager = savedManager })
	manager.PublishLogicalSize(1280, 720)
	scene, _ := newCombatUseTestScene(t)
	state := scene.gameState
	setCombatTimingTestRate(scene, rate)
	scene.spriteSheet = sharedPresentationTestSpriteSheet()
	scene.mapImage = ebiten.NewImage(304, 208)
	t.Cleanup(scene.mapImage.Deallocate)
	screen := newPresentationRaster(1280, 720)
	t.Cleanup(screen.Deallocate)
	t.Cleanup(state.EventQueueManager.Shutdown)
	t.Cleanup(scene.clearCombatVanishEffects)
	scene.windDisplay = &WindDisplay{}
	scene.moongateVisualSystem = NewMoongateVisualSystem(state)
	state.SystemCallbacks.Combat = game_state.NewCombatCallbacks(scene.recordCombatEvent, scene.recordCombatConclusion)
	state.SystemCallbacks.Visual.ShowCombatImpact = scene.showCombatImpact
	state.SystemCallbacks.Visual.PlayKapowAtPosition = scene.playKapowAtPosition
	parentIndex, parent := signatureMonster(t, scene, references.MonsterSlime, references.Position{X: 4, Y: 5})
	childIndex, child := signatureMonster(t, scene, references.MonsterSlime, references.Position{X: 5, Y: 5})
	vanished := combatVanishTestEvent(t, scene, references.Position{X: 6, Y: 5})
	attacker := game_state.CombatFeedbackEntity{Index: parentIndex, Position: parent.Position()}
	target := game_state.CombatFeedbackEntity{Index: childIndex, Position: child.Position()}
	now := time.Unix(100, 0)
	scene.startTime, scene.lastEffectsUpdate, scene.now = now, now, func() time.Time { return now }
	scene.updateWindAndEffects()
	admissions := 0
	state.EventQueueManager.Workflow("finite_effect_admission").Priority(event_queue.PriorityNormal).
		Sequential().First(func() {
		admissions++
		calls := state.GetRNG().Calls() // Production wind has already advanced this frame.
		state.SystemCallbacks.Visual.ShowCombatImpact(game_state.CombatAttackFeedback{
			Attacker: attacker, Target: target, Result: game_state.CombatFeedbackDamage, HPLost: 1,
			Presentation: game_state.CombatAttackPresentation{Delivery: game_state.CombatAttackDeliveryMelee},
		})
		state.SystemCallbacks.Combat.PublishEvent(game_state.CombatEvent{
			Kind: game_state.CombatEventSpell, Attacker: attacker, Target: target, Result: game_state.CombatFeedbackMiss,
			SpellTarget: &game_state.CombatAreaSpellTargetResult{
				Target: target, CombatantID: child.CombatantID(), Disclosed: true, Outcome: game_state.CombatSpellResisted,
			},
		})
		state.SystemCallbacks.Combat.PublishEvent(game_state.CombatEvent{
			Kind: game_state.CombatEventDivision, Attacker: attacker, Target: target, From: parent.Position(), To: child.Position(),
		})
		state.SystemCallbacks.Combat.PublishEvent(vanished)
		state.SystemCallbacks.Combat.PublishEvent(game_state.CombatEvent{Kind: game_state.CombatEventFieldDispelled, To: child.Position()})
		scene.showCombatSceptreActorEffect()
		state.SystemCallbacks.Visual.PlayKapowAtPosition(child.Position())
		assert.Equal(t, calls, state.GetRNG().Calls(), "admitting resolved presentation must not consume gameplay RNG")
	}).Start()

	// Actual production ordering: advance elapsed/wind, publish from the queue,
	// then updateVisualEffects with the entire preceding frame's delta.
	now = scene.startTime.Add(time.Duration(hitchMs) * time.Millisecond)
	scene.updateWindAndEffects()
	require.Equal(t, 1, admissions)
	timing := newCombatPresentationTiming(rate)
	assertCombatFiniteEffectAge(t, scene, timing, hitchMs, 0, childIndex)
	setCombatTimingTestRate(scene, 0.5) // Clock repair must also retain admitted durations.

	// At 1ms the mathematically live streak can still be subpixel. Keep that
	// lifetime/geometry boundary below, and require pixels on the first ordinary
	// 16ms frame using a logical display that actually matches this render target.
	const firstDrawAgeMs int64 = 16
	labelMs := max(timing.durationMs(combatImpactDurationMs), combatImpactLabelFloorMs)
	ages := []int64{0, 1, firstDrawAgeMs, labelMs - 1, labelMs, labelMs + 1}
	for _, base := range []int64{
		combatStrikeDurationMs, combatDivisionTetherDurationMs, combatLungeDurationMs, combatJoltDurationMs,
		combatSceptreActorDurationMs, combatFieldDispelDurationMs, 500, combatDivisionDurationMs, combatVanishDurationMs,
	} {
		duration := timing.durationMs(base)
		ages = append(ages, duration-1, duration, duration+1)
	}
	slices.Sort(ages)
	for _, age := range slices.Compact(ages) {
		now = scene.startTime.Add(time.Duration(hitchMs+age) * time.Millisecond)
		scene.updateWindAndEffects()
		assertCombatFiniteEffectAge(t, scene, timing, hitchMs, age, childIndex)
		if age == firstDrawAgeMs {
			for _, entity := range []game_state.CombatFeedbackEntity{attacker, target} {
				options := scene.combatEntityDrawOptions(entity.Position, entity.IsPartyMember, entity.Index, entity.Slot)
				x, y := options.GeoM.Apply(combatEntityCenterSourcePixels, combatEntityCenterSourcePixels)
				require.True(t, image.Pt(int(x), int(y)).In(screen.Bounds()), "strike endpoint must be on the fixture's render target")
			}
			screen.Clear()
			scene.drawCombatStrikeEffects(screen)
			assert.Positive(t, opaquePixelCount(screen), "the first 16ms advancing frame must still contain the scaled strike")
		}
		if age == timing.durationMs(combatStrikeDurationMs) {
			screen.Clear()
			scene.drawCombatStrikeEffects(screen)
			assert.Zero(t, opaquePixelCount(screen), "strike pixels must expire at their captured deadline while labels remain live")
		}
		calls := state.GetRNG().Calls()
		scene.updateWindAndEffects()
		assertCombatFiniteEffectAge(t, scene, timing, hitchMs, age, childIndex)
		assert.Equal(t, calls, state.GetRNG().Calls(), "same-time redraw/update cannot age effects or draw more wind rolls")
		assert.Equal(t, 1, admissions)
	}
}

func assertCombatFiniteEffectAge(
	t *testing.T, scene *GameScene, timing combatPresentationTiming, startedMs, ageMs int64, childIndex int,
) {
	t.Helper()
	if ageMs < max(timing.durationMs(combatImpactDurationMs), combatImpactLabelFloorMs) {
		require.Len(t, scene.combatImpactEffects, 2, "physical and area-spell labels retain their reading lifetime")
		for _, effect := range scene.combatImpactEffects {
			assert.Equal(t, startedMs, effect.StartedMs)
			assert.Equal(t, float64(ageMs)/1000, effect.Elapsed)
		}
		effect := scene.combatImpactEffects[0]
		visual, ok := combatStrikeVisualFor(effect.Feedback.Presentation)
		require.True(t, ok)
		visual.Duration = effect.motionDurations().strike
		_, visible := combatStrikeLineAtProgress(0, 0, 18, 0, effect.Elapsed, visual)
		assert.Equal(t, ageMs < timing.durationMs(combatStrikeDurationMs), visible)
	} else {
		assert.Empty(t, scene.combatImpactEffects)
	}
	if ageMs < timing.durationMs(combatDivisionDurationMs) {
		require.Len(t, scene.combatDivisionEffects, 1)
		effect := scene.combatDivisionEffects[0]
		assert.Equal(t, startedMs, effect.StartedMs)
		assert.Equal(t, float64(ageMs)/1000, effect.Elapsed)
		_, highlighted := scene.combatDivisionChildOutlinePhase(childIndex)
		assert.Equal(t, ageMs >= timing.durationMs(combatDivisionTetherDurationMs), highlighted)
	} else {
		assert.Empty(t, scene.combatDivisionEffects)
	}
	if ageMs < timing.durationMs(combatVanishDurationMs) {
		require.Len(t, scene.combatVanishEffects, 1)
		effect := &scene.combatVanishEffects[0]
		assert.Equal(t, startedMs, effect.StartedMs)
		assert.Equal(t, float64(ageMs)/1000, effect.Elapsed)
		assert.InDelta(t, float64(ageMs)/float64(timing.durationMs(combatVanishDurationMs)), effect.Dissolve.GetLevel(), 0.00001)
	} else {
		assert.Empty(t, scene.combatVanishEffects)
	}
	if ageMs < timing.durationMs(500) {
		require.Len(t, scene.visualEffects, 1)
		assert.Equal(t, startedMs, scene.visualEffects[0].combatStartedMs)
		assert.Equal(t, float64(ageMs)/1000, scene.visualEffects[0].Elapsed)
	} else {
		assert.Empty(t, scene.visualEffects)
	}
	assert.Equal(t, ageMs < timing.durationMs(combatSceptreActorDurationMs), len(scene.combatSceptreEffects) == 1)
	assert.Equal(t, ageMs < timing.durationMs(combatFieldDispelDurationMs), len(scene.combatFieldDispels) == 1)
}
