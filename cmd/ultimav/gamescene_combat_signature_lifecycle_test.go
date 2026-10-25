package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestCombatSignatureSchedulerWaitsForFullEffectAndOwnsCleanup(t *testing.T) {
	for _, interrupted := range []bool{false, true} {
		scene, _ := newCombatUseTestScene(t)
		scene.spriteSheet = sharedPresentationTestSpriteSheet()
		state := scene.gameState
		step := nextNonActionablePCPresentationStep(t, state)
		index, monster := signatureMonster(t, scene, references.MonsterDevil, references.Position{X: 5, Y: 5})
		scene.recordCombatEvent(signatureGateEvent(index, monster))
		effect := scene.combatSignatureEffects[0]
		before := snapshotBlockedCombatState(t, state)
		scene.startScheduledCombatAutoPass(step)
		state.EventQueueManager.Update()
		state.ElapsedMs += combatGateDurationMs / 2
		state.EventQueueManager.Update()
		require.True(t, state.EventQueueManager.HasWorkflow("combat_pc_auto_pass"), "owner must retain the turn while cue is visible")
		assert.Equal(t, before.RNGCalls, state.GetRNG().Calls())
		if interrupted {
			state.EventQueueManager.Shutdown()
		} else {
			// A hitch cannot consume the not-yet-admitted materialization/settle.
			advanceSignatureForTest(scene, state.ElapsedMs+effect.DurationMs)
			for range 3 {
				state.ElapsedMs += DialogPollingRateMs
				state.EventQueueManager.Update()
			}
		}
		assert.False(t, state.EventQueueManager.HasWorkflow("combat_pc_auto_pass"))
		assert.Empty(t, scene.combatSignatureEffects)
		assert.False(t, effect.Dissolve.IsActive())
		assert.Nil(t, effect.Frame)
		assert.Nil(t, effect.BodyFrame)
		assert.Nil(t, effect.HealthFrame)
		assert.Error(t, state.CompleteScheduledCombatStep(step.Token), "the original token was completed exactly once")
	}
}

func TestCombatDaemonScrollCancellationCleansInvocationAtEveryPhase(t *testing.T) {
	for _, phase := range []combatSummonEffectPhase{combatSummonInvocation, combatSummonSeal, combatSummonManifestation, combatSummonAllegiance} {
		scene, _ := newCombatUseTestScene(t)
		clearCombatUseTestInventory(scene.gameState)
		clearCombatUseQuestItems(scene.gameState)
		scene.gameState.PartyState.Inventory.Scrolls.Set(references.ScrollKalXenCorp, 1)
		scene.StartCombatUseWorkflow()
		dialog := waitForTopModal[*widgets.ItemSelectionDialog](t, scene)
		dialog.HandleEnter()
		for range 150 {
			scene.gameState.ElapsedMs += 10
			scene.gameState.EventQueueManager.Update()
			if scene.combatSummonEffect != nil && scene.combatSummonEffect.Phase == phase {
				break
			}
		}
		require.NotNil(t, scene.combatSummonEffect)
		require.Equal(t, phase, scene.combatSummonEffect.Phase)
		monsterCount := scene.gameState.CombatScene.MonsterManager.Count()
		scene.gameState.EventQueueManager.Shutdown()
		assert.Nil(t, scene.combatSummonEffect)
		assert.Equal(t, monsterCount, scene.gameState.CombatScene.MonsterManager.Count(), "cleanup must not summon twice")
		assert.Zero(t, scene.gameState.PartyState.Inventory.Scrolls.Get(references.ScrollKalXenCorp))
		assert.Zero(t, scene.dialogStack.Count())
	}
}

func TestCombatInPorProductionCancellationCleansEveryPhase(t *testing.T) {
	for _, phase := range []combatTeleportPhase{combatTeleportDeparture, combatTeleportHidden, combatTeleportArrival} {
		state, mock := newProductionCombatCastState(t)
		scene := newProductionCastTestScene(state)
		state.SetRandomSeed(1)
		state.PartyState.Inventory.Spells.Set(references.InPor, 1)
		state.ConfirmSpellKnowledge(references.InPor, game_state.ExperimentationKnowledgeSource())
		scene.combatMapInputHandler(ebiten.KeyC)
		dialog := waitForTopModal[*widgets.SpellCastingDialog](t, scene)
		for _, key := range []ebiten.Key{ebiten.KeyI, ebiten.KeyP, ebiten.KeyEnter} {
			scene.keyboard.SimulateKeyPress(key)
			dialog.Update()
		}
		for range 150 {
			state.ElapsedMs += 10
			state.EventQueueManager.Update()
			if scene.combatTeleportEffect != nil && scene.combatTeleportEffect.Phase == phase {
				break
			}
		}
		require.NotNil(t, scene.combatTeleportEffect)
		require.Equal(t, phase, scene.combatTeleportEffect.Phase)
		position := state.CombatScene.GetCombatantPosition(scene.combatTeleportEffect.Caster)
		caster := scene.combatTeleportEffect.Caster
		state.EventQueueManager.Shutdown()
		assert.Nil(t, scene.combatTeleportEffect)
		assert.Equal(t, position, state.CombatScene.GetCombatantPosition(caster), "cleanup cannot re-apply teleport")
		require.NotEmpty(t, mock.ScreenFlashIntensities)
		assert.Zero(t, mock.ScreenFlashIntensities[len(mock.ScreenFlashIntensities)-1])
		assert.Zero(t, scene.dialogStack.Count())
	}
}
