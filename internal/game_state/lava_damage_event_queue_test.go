package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/environment"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

func TestLavaDamageEventQueue_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()
	// Set up party with healthy characters
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)
	gs.PartyState.GetCharacter(0).SetHp(50)
	gs.PartyState.GetCharacter(1).SetStatus(party_state.Good)
	gs.PartyState.GetCharacter(1).SetHp(40)
	gs.PartyState.GetCharacter(2).SetStatus(party_state.Dead) // Should be excluded

	// Set random seed for deterministic damage
	gs.SetRandomSeed(12345)

	t.Run("LavaDamageSequence_StagesEffectsProperly_PartyStateBuilder", func(t *testing.T) {
		mockCallbacks.Reset()

		// Test the shared helper function directly via temp event queue
		// This tests the same code path that ProcessEnvironmentalHazardsOnTurnAdvancement uses
		gs.EventQueueManager.Workflow("test_lava").Priority(event_queue.PriorityNormal).
			Sequential().BlockInput().
			ThenSequence(

				gs.thenApplyHazardEffects(environment.LavaBurn)).
			Start()

		// Process the event queue through animation
		for i := 0; i < 20; i++ {
			gs.EventQueueManager.Update()
			gs.ElapsedMs += 20
		}

		// Verify warning message shown
		assert.Contains(t, mockCallbacks.Messages, "Ouch! Hot!", "Should show initial warning")

		// Verify visual effects (PartyStateBuilder uses new tracking arrays)
		assert.True(t, len(mockCallbacks.ScreenFlashColors) > 0, "Should trigger flash color")
		assert.Equal(t, 1.0, mockCallbacks.ScreenFlashColors[0].R, "Flash should be red")
		assert.True(t, len(mockCallbacks.ScreenFlashIntensities) > 0, "Should animate flash intensity")

		// Verify audio effects
		assert.Contains(t, mockCallbacks.SoundEffectsPlayed, SoundFireBurn, "Should play fire burn sound")

		// Verify damage was applied to living party members
		assert.True(t, gs.PartyState.GetCharacter(0).GetCurrentHp() < 50, "First character should take damage")
		assert.True(t, gs.PartyState.GetCharacter(1).GetCurrentHp() < 40, "Second character should take damage")
		assert.Equal(t, party_state.Dead, gs.PartyState.GetCharacter(2).GetStatus(), "Dead character should remain dead")
	})
}

// Note: Party member filtering is now handled by PartyStateBuilder.applyDamage()
// and is tested in party_state_builder_test.go (TestPartyStateBuilder_DeadCharactersSkipped)
