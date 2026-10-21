package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKlimbLargeMapSequential_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()
	t.Run("KlimbWithoutGrapple_ExpectedBehavior", func(t *testing.T) {
		mockCallbacks.Reset()

		// Try to climb without grapple - should fail with "With what?" message
		success := gs.ActionKlimbLargeMap(references.Up)

		assert.False(t, success, "Climbing without grapple should fail")
		assert.Contains(t, mockCallbacks.Messages, "With what?", "Should show 'With what?' message")
		assert.Equal(t, 0, len(mockCallbacks.TimeAdvanced()), "Should not advance time on failure")
	})
}

func TestKlimbHelperMethods(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	// Set up predictable party
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)
	gs.PartyState.GetCharacter(0).Dexterity = 30 // High dex
	gs.PartyState.GetCharacter(1).SetStatus(party_state.Good)
	gs.PartyState.GetCharacter(1).Dexterity = 5               // Low dex
	gs.PartyState.GetCharacter(2).SetStatus(party_state.Dead) // Should be skipped

	t.Run("performClimbingDexterityChecks", func(t *testing.T) {
		gs.SetRandomSeed(12345)

		injuredMembers := gs.performClimbingDexterityChecks()

		// Should have some injured members based on low dexterity
		t.Logf("Injured members: %v", injuredMembers)

		// Dead characters should be skipped
		for _, idx := range injuredMembers {
			assert.NotEqual(t, 2, idx, "Dead character should not be in injured list")
		}
	})

	t.Run("productionClimbingFallDamage", func(t *testing.T) {
		initialHP0 := gs.PartyState.GetCharacter(0).GetCurrentHp()
		initialHP1 := gs.PartyState.GetCharacter(1).GetCurrentHp()

		gs.SetRandomSeed(12345)

		// Exercise the actual staged damage path. The enclosing command owns
		// turn completion, so this fixture isolates the fall's HP mutation.
		gs.surfaceCycleOwned = true
		gs.StartClimbingFallWorkflow([]int{0})
		for range 100 {
			gs.ElapsedMs += 100
			gs.EventQueueManager.Update()
			if !gs.EventQueueManager.HasWorkflow(climbingFallWorkflowName) {
				break
			}
		}
		require.False(t, gs.EventQueueManager.HasWorkflow(climbingFallWorkflowName))

		finalHP0 := gs.PartyState.GetCharacter(0).GetCurrentHp()
		finalHP1 := gs.PartyState.GetCharacter(1).GetCurrentHp()

		// First character should take damage
		assert.True(t, finalHP0 < initialHP0, "Injured character should lose HP")

		// Second character should be unchanged
		assert.Equal(t, initialHP1, finalHP1, "Non-injured character should keep same HP")

		assert.GreaterOrEqual(t, initialHP0-finalHP0, uint16(1))
		assert.LessOrEqual(t, initialHP0-finalHP0, uint16(5))
	})
}

func TestClimbingFallUsesMinorSemanticSound(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	gs.StartClimbingFallWorkflow([]int{0})
	gs.EventQueueManager.Update()

	assert.Equal(t, []SoundEffect{SoundFallingMinor}, mockCallbacks.SoundEffectsPlayed)
}
