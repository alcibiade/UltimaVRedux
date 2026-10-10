package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	ucolor "github.com/bradhannah/UltimaVRedux/pkg/color"
)

func newTestCombatFeedback(result game_state.CombatFeedbackResult) game_state.CombatAttackFeedback {
	return game_state.CombatAttackFeedback{
		Attacker: game_state.CombatFeedbackEntity{
			IsPartyMember: false,
			Index:         0,
			Position:      references.Position{X: 3, Y: 5},
		},
		Target: game_state.CombatFeedbackEntity{
			IsPartyMember: true,
			Index:         2,
			Position:      references.Position{X: 4, Y: 5},
		},
		Result: result,
		HPLost: 7,
		Fatal:  false,
		Presentation: game_state.CombatAttackPresentation{
			Delivery: game_state.CombatAttackDeliveryMelee,
			Source:   references.LongSword,
		},
	}
}

func TestShowCombatImpact_TargetJoltsAwayFromAttacker(t *testing.T) {
	g := &GameScene{}
	g.showCombatImpact(newTestCombatFeedback(game_state.CombatFeedbackDamage))
	g.combatImpactEffects[0].Elapsed = combatJoltDurationSeconds / 6

	x, y := g.combatEntityOffset(true, 2, 0)

	assert.Greater(t, x, 0.0, "an attack from the left should jolt the target right")
	assert.Greater(t, x, 2.5, "target recoil should be visually pronounced")
	assert.Zero(t, y)
	assert.LessOrEqual(t, x, combatJoltDistancePixels)
}

func TestCombatEntityOffset_AttackerLungesTowardTarget(t *testing.T) {
	g := &GameScene{}
	g.showCombatImpact(newTestCombatFeedback(game_state.CombatFeedbackMiss))
	g.combatImpactEffects[0].Elapsed = combatLungeDurationSeconds / 2

	x, y := g.combatEntityOffset(false, 0, 0)

	assert.Greater(t, x, 0.0, "an attacker left of its target should lunge right")
	assert.InDelta(t, combatLungeDistancePixels, x, 0.001,
		"attacker should reach the full lunge distance at the midpoint")
	assert.Zero(t, y)
}

func TestCombatEntityOffset_AttackerLungesOnlyForMelee(t *testing.T) {
	for _, delivery := range []game_state.CombatAttackDelivery{
		game_state.CombatAttackDeliveryUnknown,
		game_state.CombatAttackDeliveryProjectile,
		game_state.CombatAttackDeliverySpell,
		game_state.CombatAttackDeliveryEnvironmental,
	} {
		g := &GameScene{}
		feedback := newTestCombatFeedback(game_state.CombatFeedbackMiss)
		feedback.Presentation.Delivery = delivery
		g.showCombatImpact(feedback)
		g.combatImpactEffects[0].Elapsed = combatLungeDurationSeconds / 2

		x, y := g.combatEntityOffset(false, 0, 0)

		assert.Zero(t, x, "delivery %d should not lunge", delivery)
		assert.Zero(t, y, "delivery %d should not lunge", delivery)
	}
}

func TestCombatEntityOffset_TargetJoltsOnlyWhenDamageLands(t *testing.T) {
	for _, result := range []game_state.CombatFeedbackResult{
		game_state.CombatFeedbackMiss,
		game_state.CombatFeedbackGraze,
		game_state.CombatFeedbackPoison,
		game_state.CombatFeedbackSleep,
		game_state.CombatFeedbackTheft,
	} {
		g := &GameScene{}
		g.showCombatImpact(newTestCombatFeedback(result))
		g.combatImpactEffects[0].Elapsed = combatJoltDurationSeconds / 6

		x, y := g.combatEntityOffset(true, 2, 0)
		assert.Zero(t, x, "result %d should not move the target", result)
		assert.Zero(t, y, "result %d should not move the target", result)
	}
}

// TestCombatImpactShadowUsesScreenPixels prevents the shadow from scaling into
// a visibly duplicated MISS/GRAZE label at higher display resolutions.
func TestCombatImpactShadowUsesScreenPixels(t *testing.T) {
	shadowX, shadowY := combatImpactShadowPosition(400, 250)

	assert.Equal(t, 401.0, shadowX)
	assert.Equal(t, 251.0, shadowY)
}

func TestCombatImpactLaneOffsetSeparatesRepeatedTargetFeedback(t *testing.T) {
	assert.Equal(t, 0.0, combatImpactLaneOffset(0))
	assert.Less(t, combatImpactLaneOffset(1), 0.0)
	assert.Greater(t, combatImpactLaneOffset(2), 0.0)
	assert.Less(t, combatImpactLaneOffset(3), combatImpactLaneOffset(1))
}

func TestCombatImpactLanePositionFansInwardAtEdges(t *testing.T) {
	assert.Equal(t, 10.0, combatImpactLanePosition(0, 1, 0, 100))
	assert.Equal(t, 20.0, combatImpactLanePosition(0, 2, 0, 100))
	assert.Equal(t, 90.0, combatImpactLanePosition(100, 1, 0, 100))
	assert.Equal(t, 80.0, combatImpactLanePosition(100, 2, 0, 100))
	assert.Equal(t, 0.0, combatImpactLanePosition(-20, 0, 0, 100))
	assert.Equal(t, 100.0, combatImpactLanePosition(120, 0, 0, 100))
}

func TestCombatEntityOffset_IsDeterministicAndReturnsToZero(t *testing.T) {
	g := &GameScene{}
	g.showCombatImpact(newTestCombatFeedback(game_state.CombatFeedbackDamage))
	g.combatImpactEffects[0].Elapsed = 0.04

	firstX, firstY := g.combatEntityOffset(true, 2, 0)
	secondX, secondY := g.combatEntityOffset(true, 2, 0)
	assert.Equal(t, firstX, secondX)
	assert.Equal(t, firstY, secondY)

	g.combatImpactEffects[0].Elapsed = combatJoltDurationSeconds
	finishedX, finishedY := g.combatEntityOffset(true, 2, 0)
	assert.Zero(t, finishedX)
	assert.Zero(t, finishedY)
}

func TestCombatImpactOffset_UsesExactCloneSlot(t *testing.T) {
	feedback := newTestCombatFeedback(game_state.CombatFeedbackDamage)
	feedback.Target.Slot = 1
	g := &GameScene{}
	g.showCombatImpact(feedback)
	g.combatImpactEffects[0].Elapsed = combatJoltDurationSeconds / 6

	baseX, baseY := g.combatImpactEntityOffset(true, 2, 0)
	cloneX, _ := g.combatImpactEntityOffset(true, 2, 1)

	assert.Zero(t, baseX)
	assert.Zero(t, baseY)
	assert.NotZero(t, cloneX)
}

func TestCombatStrikeVisualUsesOnlyMeleePresentation(t *testing.T) {
	melee := game_state.CombatAttackPresentation{
		Delivery: game_state.CombatAttackDeliveryMelee,
		Source:   references.LongSword,
	}
	visual, ok := combatStrikeVisualFor(melee)
	require.True(t, ok)
	assert.Equal(t, combatStrikeDurationSeconds, visual.Duration)

	for _, delivery := range []game_state.CombatAttackDelivery{
		game_state.CombatAttackDeliveryUnknown,
		game_state.CombatAttackDeliveryProjectile,
		game_state.CombatAttackDeliverySpell,
		game_state.CombatAttackDeliveryEnvironmental,
	} {
		_, enabled := combatStrikeVisualFor(game_state.CombatAttackPresentation{
			Delivery: delivery,
			Source:   references.NoEquipment,
		})
		assert.False(t, enabled, "delivery %d should not select a strike preset", delivery)
	}
}

func TestCombatStrikeVisualIsSharedAcrossMeleeSourcesAndResults(t *testing.T) {
	sources := []references.Equipment{references.NoEquipment, references.LongSword, references.Mace}
	results := []game_state.CombatFeedbackResult{
		game_state.CombatFeedbackMiss,
		game_state.CombatFeedbackGraze,
		game_state.CombatFeedbackDamage,
		game_state.CombatFeedbackPoison,
		game_state.CombatFeedbackSleep,
		game_state.CombatFeedbackTheft,
	}
	for _, source := range sources {
		for _, result := range results {
			feedback := newTestCombatFeedback(result)
			feedback.Presentation.Source = source

			visual, ok := combatStrikeVisualFor(feedback.Presentation)

			assert.True(t, ok, "source %d result %d should use the shared melee preset", source, result)
			assert.Equal(t, ucolor.EGAWhite, visual.FaceColor)
		}
	}
}

func TestCombatStrikeLineCoversAllAdjacentDirections(t *testing.T) {
	visual, ok := combatStrikeVisualFor(game_state.CombatAttackPresentation{
		Delivery: game_state.CombatAttackDeliveryMelee,
		Source:   references.NoEquipment,
	})
	require.True(t, ok)
	directions := []struct {
		x, y float64
	}{
		{x: -1, y: -1}, {x: 0, y: -1}, {x: 1, y: -1},
		{x: -1, y: 0}, {x: 1, y: 0},
		{x: -1, y: 1}, {x: 0, y: 1}, {x: 1, y: 1},
	}
	for _, direction := range directions {
		line, visible := combatStrikeLineAtProgress(
			100, 100,
			100+direction.x*18, 100+direction.y*18,
			visual.Duration*visual.RevealFraction,
			visual,
		)
		require.True(t, visible)
		assert.Equal(t, compareCoordinate(references.Coordinate(direction.x), 0),
			compareCoordinate(references.Coordinate(line.ToX-line.FromX), 0))
		assert.Equal(t, compareCoordinate(references.Coordinate(direction.y), 0),
			compareCoordinate(references.Coordinate(line.ToY-line.FromY), 0))
	}
}

func TestCombatStrikeLineExpiresAtFixedBoundary(t *testing.T) {
	visual, ok := combatStrikeVisualFor(game_state.CombatAttackPresentation{
		Delivery: game_state.CombatAttackDeliveryMelee,
		Source:   references.NoEquipment,
	})
	require.True(t, ok)

	_, visibleBeforeBoundary := combatStrikeLineAtProgress(0, 0, 18, 0, visual.Duration-0.001, visual)
	_, visibleAtBoundary := combatStrikeLineAtProgress(0, 0, 18, 0, visual.Duration, visual)

	assert.True(t, visibleBeforeBoundary)
	assert.False(t, visibleAtBoundary)
}

func TestShowCombatImpactKeepsConcurrentMeleeStrikesIndependent(t *testing.T) {
	g := &GameScene{}
	first := newTestCombatFeedback(game_state.CombatFeedbackMiss)
	second := newTestCombatFeedback(game_state.CombatFeedbackDamage)
	second.Attacker.Index = 1
	second.Target.Index = 3
	second.Presentation.Source = references.Mace

	g.showCombatImpact(first)
	g.showCombatImpact(second)
	g.combatImpactEffects[0].Elapsed = 0.02
	g.combatImpactEffects[1].Elapsed = 0.08

	require.Len(t, g.combatImpactEffects, 2)
	for index := range g.combatImpactEffects {
		visual, ok := combatStrikeVisualFor(g.combatImpactEffects[index].Feedback.Presentation)
		require.True(t, ok)
		_, visible := combatStrikeLineAtProgress(0, 0, 18, 0, g.combatImpactEffects[index].Elapsed, visual)
		assert.True(t, visible)
	}
}

func TestCombatImpactEffectsClearOutsideCombat(t *testing.T) {
	g := &GameScene{
		gameState: &game_state.GameState{},
		combatImpactEffects: []CombatImpactEffect{
			{Feedback: newTestCombatFeedback(game_state.CombatFeedbackMiss)},
			{Feedback: newTestCombatFeedback(game_state.CombatFeedbackDamage)},
		},
	}

	g.updateCombatImpactEffects(0.01)

	assert.Empty(t, g.combatImpactEffects)
}

func TestCombatFeedbackText_ShowsAttackResult(t *testing.T) {
	label, _ := combatFeedbackText(newTestCombatFeedback(game_state.CombatFeedbackMiss))
	assert.Equal(t, "MISS", label)

	label, _ = combatFeedbackText(newTestCombatFeedback(game_state.CombatFeedbackGraze))
	assert.Equal(t, "GRAZE", label)

	label, _ = combatFeedbackText(newTestCombatFeedback(game_state.CombatFeedbackDamage))
	assert.Equal(t, "-7", label)

	label, _ = combatFeedbackText(newTestCombatFeedback(game_state.CombatFeedbackSleep))
	assert.Equal(t, "SLEEP", label)

	label, _ = combatFeedbackText(newTestCombatFeedback(game_state.CombatFeedbackTheft))
	assert.Equal(t, "STOLEN", label)
}

func TestCombatDamageAlpha_FadesOnlyNearEnd(t *testing.T) {
	assert.Equal(t, 1.0, combatDamageAlpha(0))
	assert.Equal(t, 1.0, combatDamageAlpha(combatDamageFadeStart))
	assert.Less(t, combatDamageAlpha(0.8), 1.0)
	assert.Zero(t, combatDamageAlpha(1))
}

func TestCombatFireImpactTintIsBriefAndTargeted(t *testing.T) {
	g := &GameScene{}
	g.showCombatFireImpact(game_state.CombatEvent{
		Kind: game_state.CombatEventFieldTriggered,
		Target: game_state.CombatFeedbackEntity{
			IsPartyMember: true,
			Index:         2,
			Position:      references.Position{X: 4, Y: 5},
		},
		HPLost:    7,
		FieldKind: game_state.CombatFieldFire,
	})

	assert.Equal(t, combatFireTintMaximumAlpha, g.combatFireImpactAlpha(true, 2, 0))
	assert.Equal(t, game_state.CombatAttackDeliveryEnvironmental,
		g.combatImpactEffects[0].Feedback.Presentation.Delivery)
	_, strike := combatStrikeVisualFor(g.combatImpactEffects[0].Feedback.Presentation)
	assert.False(t, strike)
	assert.Zero(t, g.combatFireImpactAlpha(true, 1, 0), "other party members must not be tinted")
	g.combatImpactEffects[0].Elapsed = combatFireTintDurationSeconds
	assert.Zero(t, g.combatFireImpactAlpha(true, 2, 0))
}

func TestCombatFireImpactUsesFloatingDamageAmount(t *testing.T) {
	g := &GameScene{}
	g.showCombatFireImpact(game_state.CombatEvent{
		Target: game_state.CombatFeedbackEntity{
			IsPartyMember: true,
			Index:         2,
			Position:      references.Position{X: 4, Y: 5},
		},
		HPLost: 7,
	})

	label, _ := combatFeedbackText(g.combatImpactEffects[0].Feedback)
	assert.Equal(t, "-7", label)
}
