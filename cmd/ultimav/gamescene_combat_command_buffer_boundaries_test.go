package main

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func TestBufferedCombatBlockedCompletionFrameRejectsArrowBeforeQueueAdvance(t *testing.T) {
	scene, held := newBufferedCombatUpdateScene(t)
	held[ebiten.KeyG] = true
	require.NoError(t, scene.Update(nil))
	delete(held, ebiten.KeyG)
	// Finish presentation only as far as authoritative readiness while its
	// finalizing workflow still owns the input block.
	for range 200 {
		scene.gameState.ElapsedMs += 16
		scene.updateWindAndEffects()
		projection, eligible := scene.gameState.ProjectSoloCombatInput()
		if eligible && projection.Ready {
			break
		}
	}
	require.True(t, scene.inputBlocked)
	held[ebiten.KeyRight] = true
	require.NoError(t, scene.Update(nil))
	require.Equal(t, GetDirectionInput, scene.secondaryKeyState)
	require.True(t, scene.keyboard.IsSuppressedUntilRelease(ebiten.KeyRight))
	before := snapshotBlockedCombatState(t, scene.gameState)
	require.NoError(t, scene.Update(nil))
	require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
}

func TestBufferedCombatDoesNotCaptureOwnActionOrCarriedLetter(t *testing.T) {
	scene, held, _ := newCombatDirectionUpdateScene(t, "last_remaining")
	state := scene.gameState
	require.True(t, state.SelectCombatPlayer(state.CombatScene.GetCurrentTurnIdentity().EntityIndex))
	token := state.FinishCombatTurnDeferred()
	state.EventQueueManager.Workflow("player_action_test").Priority(event_queue.PriorityHigh).
		Sequential().BlockInput().First(func() {}).Wait(100).
		Then(func() { require.NoError(t, state.CompleteScheduledCombatStep(token)) }).Start()
	held[ebiten.KeyG] = true
	require.NoError(t, scene.Update(nil))
	require.False(t, scene.transientCommandInput.pending.active)
	for range 100 {
		state.ElapsedMs += 16
		require.NoError(t, scene.Update(nil))
		if !state.EventQueueManager.HasWorkflow("player_action_test") {
			break
		}
	}
	require.False(t, scene.transientCommandInput.pending.active)
	// Explicitly traverse ordinary paid turns to the next enemy if the first
	// deferred action happened to return directly to this high-DEX character.
	clear(held)
	require.NoError(t, scene.Update(nil))
	for range 100 {
		if state.CombatScene.HasScheduledCombatStep() {
			break
		}
		state.FinishCombatAction()
	}
	// A key edge sampled before the enemy presentation is installed is carried,
	// not a request observed during that owner.
	held[ebiten.KeyG] = true
	require.NoError(t, scene.Update(nil))
	require.False(t, scene.transientCommandInput.pending.active)
	require.True(t, scene.keyboard.IsSuppressedUntilRelease(ebiten.KeyG))
	clear(held)
	scene.ResetTransientCommandInput()
	state.EventQueueManager.Shutdown()
}

func TestBufferedCombatModalAndModifiedInputKeepPrecedence(t *testing.T) {
	scene, held := newBufferedCombatUpdateScene(t)
	held[ebiten.KeyControlLeft], held[ebiten.KeyG] = true, true
	require.NoError(t, scene.Update(nil))
	require.False(t, scene.transientCommandInput.pending.active)
	clear(held)
	require.NoError(t, scene.Update(nil))
	held[ebiten.KeyG] = true
	require.NoError(t, scene.Update(nil))
	require.True(t, scene.transientCommandInput.pending.active)
	scene.showCommandHelp()
	held[ebiten.KeyO] = true
	require.NoError(t, scene.Update(nil))
	require.False(t, scene.transientCommandInput.pending.active)
	require.Equal(t, 1, scene.dialogStack.Count(), "only Help owns a dialog")
	require.True(t, scene.dialogStack.HasModalDialog())
}

func TestBufferedCombatSameSampleSupportedTieConsumesLoser(t *testing.T) {
	scene, held := newBufferedCombatUpdateScene(t)
	held[ebiten.KeyJ], held[ebiten.KeyO] = true, true
	require.NoError(t, scene.Update(nil))
	require.Equal(t, ebiten.KeyJ, scene.transientCommandInput.pending.key)
	delete(held, ebiten.KeyJ)
	require.NoError(t, scene.Update(nil))
	require.Equal(t, ebiten.KeyJ, scene.transientCommandInput.pending.key, "held losing O cannot emerge next frame")
	require.True(t, scene.keyboard.IsSuppressedUntilRelease(ebiten.KeyO))
}
