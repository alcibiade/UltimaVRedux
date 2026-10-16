package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/require"
)

func newBufferedCombatUpdateScene(t *testing.T) (*GameScene, map[ebiten.Key]bool) {
	t.Helper()
	scene, held, _ := newCombatDirectionUpdateScene(t, "last_remaining")
	state := scene.gameState
	state.SystemCallbacks.Visual.IsAnyKeyPressed = scene.IsAnyKeyPressed
	scene.keyboard.SetElapsedMillisecondsSource(scene.getElapsedMs)
	require.True(t, state.SelectCombatPlayer(state.CombatScene.GetCurrentTurnIdentity().EntityIndex))
	for range 100 {
		state.FinishCombatAction()
		if state.CombatScene.HasScheduledCombatStep() {
			break
		}
	}
	require.True(t, state.CombatScene.HasScheduledCombatStep())
	require.NoError(t, scene.Update(nil))
	require.True(t, scene.inputBlocked)
	require.True(t, state.EventQueueManager.HasWorkflow("monster_turn_delay"))
	t.Cleanup(func() {
		clear(held)
		scene.ResetTransientCommandInput()
		state.EventQueueManager.Shutdown()
	})
	return scene, held
}

func advanceBufferedCombatToPrompt(t *testing.T, scene *GameScene, want InputState) {
	t.Helper()
	for range 200 {
		scene.gameState.ElapsedMs += 16
		require.NoError(t, scene.Update(nil))
		if scene.secondaryKeyState == want {
			return
		}
	}
	t.Fatalf("command never became active: want %v, got %v", want, scene.secondaryKeyState)
}

func TestBufferedCombatVerbsEnterNormalPromptsThroughUpdate(t *testing.T) {
	for _, test := range []struct {
		key   ebiten.Key
		state InputState
	}{
		{ebiten.KeyG, GetDirectionInput}, {ebiten.KeyO, OpenDirectionInput},
		{ebiten.KeyJ, JimmyDoorDirectionInput}, {ebiten.KeyP, PushDirectionInput},
		{ebiten.KeyS, SearchDirectionInput},
	} {
		t.Run(test.key.String(), func(t *testing.T) {
			scene, held := newBufferedCombatUpdateScene(t)
			before := snapshotBlockedCombatState(t, scene.gameState)
			outputBefore := scene.output.GetOutputStr(true)
			held[test.key] = true
			require.NoError(t, scene.Update(nil))
			require.True(t, scene.transientCommandInput.pending.active)
			require.Zero(t, scene.dialogStack.Count(), "enqueue has no status dialog")
			require.Equal(t, outputBefore, scene.output.GetOutputStr(true), "enqueue is silent")
			require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState), "recording intent is not an action")
			advanceBufferedCombatToPrompt(t, scene, test.state)
			require.False(t, scene.transientCommandInput.pending.active)
			require.Zero(t, scene.dialogStack.Count(), "only the ordinary direction prompt is needed")
			actor := scene.gameState.CombatScene.GetCurrentTurnIdentity()
			position := scene.gameState.CombatScene.GetCombatantPosition(actor)
			beforeCancel := snapshotBlockedCombatState(t, scene.gameState)
			held[ebiten.KeyEscape] = true
			require.NoError(t, scene.Update(nil))
			require.Equal(t, PrimaryInput, scene.secondaryKeyState)
			require.Equal(t, beforeCancel, snapshotBlockedCombatState(t, scene.gameState))
			require.Equal(t, position, scene.gameState.CombatScene.GetCombatantPosition(actor))
			require.NoError(t, scene.Update(nil))
			require.Equal(t, PrimaryInput, scene.secondaryKeyState, "held original verb cannot refill")
		})
	}
}

func TestBufferedCombatReplacementHeldFirstKeyAndEarlyDirections(t *testing.T) {
	for _, direction := range []ebiten.Key{ebiten.KeyRight, ebiten.KeyNumpad6} {
		t.Run(direction.String(), func(t *testing.T) {
			scene, held := newBufferedCombatUpdateScene(t)
			outputBefore := scene.output.GetOutputStr(true)
			held[ebiten.KeyG] = true
			require.NoError(t, scene.Update(nil))
			held[ebiten.KeyO] = true
			require.NoError(t, scene.Update(nil))
			require.Equal(t, ebiten.KeyO, scene.transientCommandInput.pending.key)
			held[ebiten.KeyJ], held[direction] = true, true
			require.NoError(t, scene.Update(nil))
			require.Equal(t, ebiten.KeyJ, scene.transientCommandInput.pending.key)
			require.Zero(t, scene.dialogStack.Count(), "replacement has no status dialog")
			require.Equal(t, outputBefore, scene.output.GetOutputStr(true), "replacement is silent")
			require.True(t, scene.keyboard.IsSuppressedUntilRelease(direction))
			advanceBufferedCombatToPrompt(t, scene, JimmyDoorDirectionInput)
			before := snapshotBlockedCombatState(t, scene.gameState)
			for range 3 {
				require.NoError(t, scene.Update(nil))
			}
			require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
			delete(held, direction)
			require.NoError(t, scene.Update(nil))
			held[direction] = true
			require.NoError(t, scene.Update(nil))
			require.Equal(t, PrimaryInput, scene.secondaryKeyState, "fresh direction reaches Jimmy, never primary movement")
			require.False(t, scene.transientCommandInput.pending.active)
		})
	}
}

func TestBufferedCombatCancellationUnsupportedAndSameSampleEscapeWin(t *testing.T) {
	for _, clearKey := range []ebiten.Key{ebiten.KeyEscape, ebiten.KeySpace, ebiten.KeyTab, ebiten.KeyK} {
		t.Run(clearKey.String(), func(t *testing.T) {
			scene, held := newBufferedCombatUpdateScene(t)
			held[ebiten.KeyG], held[ebiten.KeyRight] = true, true
			require.NoError(t, scene.Update(nil))
			held[clearKey], held[ebiten.KeyO] = true, true
			require.NoError(t, scene.Update(nil))
			require.False(t, scene.transientCommandInput.pending.active)
			require.Zero(t, scene.dialogStack.Count(), "clearing has no status dialog")
			require.True(t, scene.keyboard.IsSuppressedUntilRelease(clearKey))
			require.True(t, scene.keyboard.IsSuppressedUntilRelease(ebiten.KeyRight))
			for range 100 {
				scene.gameState.ElapsedMs += 16
				require.NoError(t, scene.Update(nil))
			}
			require.Equal(t, PrimaryInput, scene.secondaryKeyState)
			require.Zero(t, scene.dialogStack.Count(), "held Escape cannot open combat exit")
			require.False(t, scene.keyboard.TryToRegisterRepeatableKeyPress(ebiten.KeyRight))
		})
	}
}

func TestBufferedCombatReadyHandoffDirectionUsesNewPromptInSameUpdate(t *testing.T) {
	scene, held := newBufferedCombatUpdateScene(t)
	held[ebiten.KeyG] = true
	require.NoError(t, scene.Update(nil))
	delete(held, ebiten.KeyG)
	// Advance the real scheduled owner without running command routing. This
	// leaves a retained verb at an already-ready entry boundary for the next Update.
	for range 200 {
		scene.gameState.ElapsedMs += 16
		scene.updateWindAndEffects()
		projection, eligible := scene.gameState.ProjectSoloCombatInput()
		if eligible && projection.Ready && !scene.inputBlocked {
			break
		}
	}
	require.False(t, scene.inputBlocked)
	require.True(t, scene.transientCommandInput.pending.active)
	actor := scene.gameState.CombatScene.GetCurrentTurnIdentity()
	position := scene.gameState.CombatScene.GetCombatantPosition(actor)
	held[ebiten.KeyRight] = true
	require.NoError(t, scene.Update(nil))
	require.Equal(t, PrimaryInput, scene.secondaryKeyState, "Get consumes the fresh direction in the acceptance frame")
	require.Equal(t, position, scene.gameState.CombatScene.GetCombatantPosition(actor))
	require.False(t, scene.transientCommandInput.pending.active)
}

func TestBufferedCombatGetThenOpenNeedsOnlyOrdinaryPrompt(t *testing.T) {
	scene, held := newBufferedCombatUpdateScene(t)
	outputBefore := scene.output.GetOutputStr(true)
	held[ebiten.KeyG] = true
	require.NoError(t, scene.Update(nil))
	held[ebiten.KeyO] = true
	require.NoError(t, scene.Update(nil))
	require.Equal(t, ebiten.KeyO, scene.transientCommandInput.pending.key)
	require.Zero(t, scene.dialogStack.Count())
	require.Equal(t, outputBefore, scene.output.GetOutputStr(true))
	advanceBufferedCombatToPrompt(t, scene, OpenDirectionInput)
	require.False(t, scene.transientCommandInput.pending.active)
	require.Zero(t, scene.dialogStack.Count(), "Open is ready without dismissing a queue card")
	before := snapshotBlockedCombatState(t, scene.gameState)
	for range 3 {
		require.NoError(t, scene.Update(nil))
	}
	require.Equal(t, OpenDirectionInput, scene.secondaryKeyState)
	require.Equal(t, before, snapshotBlockedCombatState(t, scene.gameState))
}

func TestBufferedCombatFocusAndActorInvalidationCannotRevive(t *testing.T) {
	for _, invalidate := range []string{"focus", "sleep", "selection", "encounter"} {
		t.Run(invalidate, func(t *testing.T) {
			scene, held := newBufferedCombatUpdateScene(t)
			focused := true
			scene.transientCommandInput.focused = func() bool { return focused }
			held[ebiten.KeyG] = true
			require.NoError(t, scene.Update(nil))
			actor := scene.transientCommandInput.pending.opportunity.Actor
			switch invalidate {
			case "focus":
				focused = false
			case "sleep":
				scene.gameState.CombatScene.PartySleeping[actor.EntityIndex] = true
			case "selection":
				scene.gameState.CombatScene.ClearSoloMoveMode()
			case "encounter":
				scene.gameState.CombatScene.EncounterID++
			}
			require.NoError(t, scene.Update(nil))
			require.False(t, scene.transientCommandInput.pending.active)
			require.Zero(t, scene.dialogStack.Count(), "invalidation has no status dialog")
			focused = true
			scene.gameState.CombatScene.PartySleeping[actor.EntityIndex] = false
			scene.gameState.CombatScene.SetSoloMoveCharacter(actor.EntityIndex)
			require.NoError(t, scene.Update(nil))
			require.False(t, scene.transientCommandInput.pending.active)
		})
	}
}
