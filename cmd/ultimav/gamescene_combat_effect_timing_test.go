package main

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestCombatTimingGateAdmitsOnePhaseAfterHitchEveryRate(t *testing.T) {
	for _, rate := range combatTimingTestRates() {
		t.Run(fmt.Sprint(rate), func(t *testing.T) {
			scene, callbacks := newCombatUseTestScene(t)
			setCombatTimingTestRate(scene, rate)
			scene.spriteSheet = sharedPresentationTestSpriteSheet()
			index, monster := signatureMonster(t, scene, references.MonsterDevil, references.Position{X: 5, Y: 5})
			scene.showCombatSignatureEffect(signatureGateEvent(index, monster))
			t.Cleanup(scene.clearCombatSignatureEffects)
			effect := scene.combatSignatureEffects[0]
			timing := newCombatPresentationTiming(rate)
			assert.Equal(t, [3]int64{timing.durationMs(125), timing.durationMs(600), timing.durationMs(125)}, effect.PhaseDurationsMs)
			setCombatTimingTestRate(scene, 4)
			for phase := combatGateArrival; phase < combatGateComplete; phase++ {
				start := effect.PhaseStartedMs
				duration := effect.PhaseDurationsMs[int(phase)]
				scene.gameState.ElapsedMs = start + duration - 1
				scene.updateCombatSignatureEffects()
				require.Equal(t, phase, effect.GatePhase)
				// A huge hitch may admit the next phase, never spend its budget too.
				scene.gameState.ElapsedMs = start + duration + 10000
				scene.updateCombatSignatureEffects()
				require.Equal(t, phase+1, effect.GatePhase)
				if phase+1 < combatGateComplete {
					require.Len(t, scene.combatSignatureEffects, 1)
					scene.updateCombatSignatureEffects()
					require.Equal(t, phase+1, effect.GatePhase)
				}
			}
			assert.Empty(t, scene.combatSignatureEffects)
			assert.Nil(t, effect.Frame)
			assert.Equal(t, []game_state.SoundEffect{game_state.SoundMonsterGate}, callbacks.SoundEffectsPlayed)
		})
	}
}

func TestCombatTimingBlinkStoredBoundariesEveryRate(t *testing.T) {
	for _, rate := range combatTimingTestRates() {
		t.Run(fmt.Sprint(rate), func(t *testing.T) {
			scene, _ := newCombatUseTestScene(t)
			setCombatTimingTestRate(scene, rate)
			scene.spriteSheet = sharedPresentationTestSpriteSheet()
			index, monster := signatureMonster(t, scene, references.MonsterWisp, references.Position{X: 5, Y: 5})
			scene.showCombatSignatureEffect(game_state.CombatEvent{Kind: game_state.CombatEventTeleport,
				Attacker: game_state.CombatFeedbackEntity{Index: index}, From: references.Position{X: 3, Y: 5}, To: monster.Position()})
			t.Cleanup(scene.clearCombatSignatureEffects)
			require.Len(t, scene.combatSignatureEffects, 1)
			effect := scene.combatSignatureEffects[0]
			setCombatTimingTestRate(scene, 0.5)
			departure, hidden, arrival := effect.PhaseDurationsMs[0], effect.PhaseDurationsMs[1], effect.PhaseDurationsMs[2]
			assert.Equal(t, departure+hidden+arrival, effect.DurationMs)
			for _, point := range []struct {
				age   int64
				phase combatTeleportPhase
			}{
				{departure - 1, combatTeleportDeparture}, {departure, combatTeleportHidden}, {departure + 1, combatTeleportHidden},
				{departure + hidden - 1, combatTeleportHidden}, {departure + hidden, combatTeleportArrival},
				{departure + hidden + 1, combatTeleportArrival},
			} {
				phase, _ := effect.blink(point.age)
				assert.Equal(t, point.phase, phase)
			}
			scene.gameState.ElapsedMs = effect.StartedMs + effect.DurationMs - 1
			assert.True(t, scene.combatSignatureEffectCurrent(effect))
			scene.gameState.ElapsedMs++
			assert.False(t, scene.combatSignatureEffectCurrent(effect))
			scene.updateCombatSignatureEffects()
			assert.Empty(t, scene.combatSignatureEffects)
		})
	}
}

func TestCombatTimingFiniteCuesDoNotShortenLabelsOrBlockInput(t *testing.T) {
	for _, rate := range combatTimingTestRates() {
		t.Run(fmt.Sprint(rate), func(t *testing.T) {
			scene, _ := newCombatUseTestScene(t)
			setCombatTimingTestRate(scene, rate)
			timing := newCombatPresentationTiming(rate)
			scene.spriteSheet = sharedPresentationTestSpriteSheet()
			scene.showCombatImpact(game_state.CombatAttackFeedback{
				Attacker:     game_state.CombatFeedbackEntity{IsPartyMember: true, Index: 0, Position: references.Position{X: 3, Y: 3}},
				Target:       game_state.CombatFeedbackEntity{Index: 0, Position: references.Position{X: 4, Y: 3}},
				Result:       game_state.CombatFeedbackDamage,
				Presentation: game_state.CombatAttackPresentation{Delivery: game_state.CombatAttackDeliveryMelee},
			})
			effect := &scene.combatImpactEffects[0]
			assert.Equal(t, timing.seconds(320), effect.motionDurations().jolt)
			assert.Equal(t, timing.seconds(300), effect.motionDurations().lunge)
			assert.Equal(t, timing.seconds(180), effect.motionDurations().strike)
			setCombatTimingTestRate(scene, 1)
			scene.gameState.ElapsedMs = effect.StartedMs + timing.durationMs(300)
			scene.updateCombatImpactEffects(0)
			x, y := scene.combatImpactEntityOffset(true, 0, 0)
			assert.Zero(t, x)
			assert.Zero(t, y)
			scene.gameState.ElapsedMs = effect.StartedMs + timing.durationMs(320)
			scene.updateCombatImpactEffects(0)
			x, y = scene.combatImpactEntityOffset(false, 0, 0)
			assert.Zero(t, x)
			assert.Zero(t, y)
			scene.updateCombatImpactEffects(0)
			require.Len(t, scene.combatImpactEffects, 1, "motion ends before the reading budget")
			setCombatTimingTestRate(scene, rate)
			scene.showCombatDivisionEffect(game_state.CombatEvent{})
			scene.showCombatVanishEffect(combatVanishTestEvent(t, scene, references.Position{X: 4, Y: 5}))
			scene.showCombatSceptreActorEffect()
			scene.showCombatFieldDispelEffect(game_state.CombatEvent{})
			start := scene.gameState.ElapsedMs
			setCombatTimingTestRate(scene, 1)
			scene.gameState.ElapsedMs = start + timing.durationMs(320)
			scene.updateCombatSceptreEffects()
			assert.Empty(t, scene.combatSceptreEffects)
			require.Len(t, scene.combatFieldDispels, 1)
			scene.gameState.ElapsedMs = start + timing.durationMs(420)
			scene.updateCombatSceptreEffects()
			assert.Empty(t, scene.combatFieldDispels)
			scene.gameState.ElapsedMs = start + timing.durationMs(720) - 1
			scene.updateCombatDivisionEffects(0)
			require.Len(t, scene.combatDivisionEffects, 1)
			scene.gameState.ElapsedMs++
			scene.updateCombatDivisionEffects(0)
			assert.Empty(t, scene.combatDivisionEffects)
			scene.gameState.ElapsedMs = start + timing.durationMs(750) - 1
			scene.updateCombatVanishEffects(0)
			require.Len(t, scene.combatVanishEffects, 1)
			scene.gameState.ElapsedMs++
			scene.updateCombatVanishEffects(0)
			assert.Empty(t, scene.combatVanishEffects)
			assert.False(t, scene.gameState.EventQueueManager.HasInputBlockingWorkflows())
		})
	}
}

func TestCombatTimingUtilityPhasesSnapshotEveryRate(t *testing.T) {
	for _, rate := range combatTimingTestRates() {
		t.Run(fmt.Sprint(rate), func(t *testing.T) {
			scene, callbacks := newCombatDoorSpellScene(t, references.AnExPor)
			setCombatTimingTestRate(scene, rate)
			state := scene.gameState
			t.Cleanup(state.EventQueueManager.Shutdown)
			target := references.Position{X: 6, Y: 5}
			setProductionCombatPushTerrain(state, target, indexes.RegularDoor)
			casterIdx := state.CombatScene.GetActiveCombatantIndex()
			require.True(t, state.BeginCombatAdjacentSpellTargeting(game_state.CastRequest{Spell: references.AnExPor, CasterIdx: casterIdx}).Valid)
			state.SetCombatAdjacentSpellDirection(references.Right)
			result := state.ConfirmCombatAdjacentSpellTarget()
			require.Equal(t, game_state.CombatAdjacentSpellTargetResolved, result.Outcome)
			start := state.ElapsedMs
			timing := newCombatPresentationTiming(rate)
			scene.startCombatUtilitySpellEffect(result)
			setCombatTimingTestRate(scene, 1)
			drainCombatPacingAt(scene, start)
			drainCombatPacingAt(scene, start+timing.durationMs(180)-1)
			assertSceneCombatDoorTerrain(t, state, target, indexes.RegularDoor)
			drainCombatPacingAt(scene, start+timing.durationMs(180))
			assertSceneCombatDoorTerrain(t, state, target, indexes.MagicLockDoor)
			require.NotNil(t, scene.combatUtilityEffect)
			assert.Equal(t, combatUtilityResult, scene.combatUtilityEffect.Phase)
			drainCombatPacingAt(scene, start+timing.durationMs(180)+timing.durationMs(220)-1)
			assert.True(t, callbacks.InputBlocked)
			drainCombatPacingAt(scene, start+timing.durationMs(180)+timing.durationMs(220))
			assert.Nil(t, scene.combatUtilityEffect)
			assert.False(t, callbacks.InputBlocked)
			require.Len(t, combatDoorSpellEvents(callbacks), 1)
			state.EventQueueManager.Shutdown()
			require.Len(t, combatDoorSpellEvents(callbacks), 1)
		})
	}
}

func TestCombatTimingSummonPhaseAdmissionEveryRate(t *testing.T) {
	for _, rate := range combatTimingTestRates() {
		t.Run(fmt.Sprint(rate), func(t *testing.T) {
			scene, _ := newCombatUseTestScene(t)
			setCombatTimingTestRate(scene, rate)
			state := scene.gameState
			t.Cleanup(state.EventQueueManager.Shutdown)
			state.SetRandomSeed(1)
			before := state.CombatScene.MonsterManager.Count()
			finished := 0
			state.EventQueueManager.Workflow("timed_summon").Priority(event_queue.PriorityHigh).
				Sequential().First(func() {}).ThenSequence(scene.thenCastCombatSummon(references.KalXen, state.CombatScene.GetActiveCombatantIndex())).
				Finally(func() { finished++; scene.clearCombatInvocationEffects() }).Start()
			setCombatTimingTestRate(scene, 1)
			at := state.ElapsedMs
			drainCombatPacingAt(scene, at)
			require.NotNil(t, scene.combatSummonEffect)
			assert.Equal(t, before+1, state.CombatScene.MonsterManager.Count())
			for phase, base := range []int64{120, 140, 300, 180} {
				duration := newCombatPresentationTiming(rate).durationMs(base)
				drainCombatPacingAt(scene, at+duration-1)
				require.NotNil(t, scene.combatSummonEffect)
				assert.Equal(t, combatSummonEffectPhase(phase), scene.combatSummonEffect.Phase)
				at += duration
				drainCombatPacingAt(scene, at)
			}
			assert.Nil(t, scene.combatSummonEffect)
			assert.Equal(t, 1, finished)
			assert.Equal(t, before+1, state.CombatScene.MonsterManager.Count())
		})
	}
}
