package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
)

func TestCombatAutoPassWithoutSignatureHonorsSpeedAdjustedDelay(t *testing.T) {
	for _, test := range []struct {
		name       string
		speed      float64
		durationMs int64
	}{
		{name: "slow", speed: 0.5, durationMs: 500},
		{name: "normal", speed: 1, durationMs: 250},
		{name: "one and half", speed: 1.5, durationMs: 167},
		{name: "double", speed: 2, durationMs: 125},
		{name: "two and half", speed: 2.5, durationMs: 100},
		{name: "triple", speed: 3, durationMs: 83},
		{name: "three and half", speed: 3.5, durationMs: 71},
		{name: "fast", speed: 4, durationMs: 63},
	} {
		t.Run(test.name, func(t *testing.T) {
			scene, callbacks := newCombatUseTestScene(t)
			setCombatTimingTestRate(scene, test.speed)
			state := scene.gameState
			step := nextNonActionablePCPresentationStep(t, state)
			require.Equal(t, game_state.CombatPresentationAutoPass, step.Kind)
			require.Equal(t, step.Actor, state.CombatScene.GetCurrentTurnIdentity())
			require.Empty(t, scene.combatSignatureEffects)
			require.False(t, scene.hasCombatSignatureEffects())
			require.Equal(t, uint64(test.durationMs), scene.combatMovementDuration(combatAutoPassPCDurationMs))
			actorBefore, ok := state.CombatScene.TurnQueue.GetEntrySnapshot(step.Actor)
			require.True(t, ok)
			rngBefore := state.GetRNG().Calls()
			const startMs int64 = 900
			state.ElapsedMs = startMs
			scene.startScheduledCombatAutoPass(step)
			scene.gameConfig.GameplayConfiguration.CombatSpeedFactor = 1
			require.True(t, callbacks.InputBlocked)

			// Drain every immediately runnable stage at the same clock value.
			// A single Update can hide an accidentally discarded Wait behind
			// the queue's own callback/finalizer staging.
			for _, elapsed := range []int64{0, test.durationMs / 2, test.durationMs - 1} {
				state.ElapsedMs = startMs + elapsed
				for range 8 {
					state.EventQueueManager.Update()
					require.True(t, state.EventQueueManager.HasWorkflow("combat_pc_auto_pass"), "elapsed=%d", elapsed)
					require.True(t, state.EventQueueManager.HasInputBlockingWorkflows())
					require.True(t, callbacks.InputBlocked)
					require.True(t, state.CombatScene.HasScheduledCombatStep())
					require.Equal(t, step.Actor, state.CombatScene.GetCurrentTurnIdentity())
					actorNow, found := state.CombatScene.TurnQueue.GetEntrySnapshot(step.Actor)
					require.True(t, found)
					assert.Equal(t, actorBefore, actorNow)
					// A pending auto-pass token validates, but has no executable
					// action. This checks the exact token without completing it.
					_, err := state.ResolveScheduledCombatStep(step.Token)
					require.ErrorIs(t, err, game_state.ErrCombatPresentationNotExecutable)
					assert.Equal(t, rngBefore, state.GetRNG().Calls())
				}
			}

			state.ElapsedMs = startMs + test.durationMs
			for range 8 {
				state.EventQueueManager.Update()
			}
			assert.False(t, state.EventQueueManager.HasWorkflow("combat_pc_auto_pass"))
			assert.False(t, state.EventQueueManager.HasInputBlockingWorkflows())
			assert.False(t, callbacks.InputBlocked)
			after := snapshotBlockedCombatState(t, state)
			assert.ErrorIs(t, state.CompleteScheduledCombatStep(step.Token), game_state.ErrCombatPresentationDuplicate)
			assert.Equal(t, after, snapshotBlockedCombatState(t, state), "duplicate completion must not advance the replacement turn")
		})
	}
}
