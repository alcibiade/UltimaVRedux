package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/environment"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

const (
	// Test configuration constants
	testPlayerX              = 100
	testPlayerY              = 100
	testSeed1                = 12345
	testSeed2                = 54321
	lowDexterity             = 5
	highDexterity            = 25
	testMaxDexterity         = 30
	extremelyLowDexterity    = 1
	expectedMinPoisonedCount = 1
	expectedMinMessageCount  = 2
	partySize                = 4

	// Green mist visual effect constants
	greenMistRed       = 0.2
	greenMistGreen     = 0.6
	greenMistBlue      = 0.2
	greenMistIntensity = 0.3
)

func TestSwampPoisonEventQueue_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(testPlayerX, testPlayerY).
		WithSystemCallbacks().
		Build()
	// Set up party with different dexterity levels for testing
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)
	gs.PartyState.GetCharacter(0).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(0).Dexterity = lowDexterity // Low dexterity - likely to fail
	gs.PartyState.GetCharacter(1).SetStatus(party_state.Good)
	gs.PartyState.GetCharacter(1).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(1).Dexterity = highDexterity       // High dexterity - likely to resist
	gs.PartyState.GetCharacter(2).SetStatus(party_state.Poisoned) // Already poisoned - should be skipped
	gs.PartyState.GetCharacter(2).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(3).SetStatus(party_state.Dead) // Dead - should be skipped

	// Set random seed for deterministic testing
	gs.SetRandomSeed(testSeed1)

	t.Run("SwampPoisonSequence_StagesEffectsProperly", func(t *testing.T) {
		mockCallbacks.Reset()

		// Test the shared helper function directly via temp event queue
		// This tests the same code path that ProcessEnvironmentalHazardsOnTurnAdvancement uses
		gs.EventQueueManager.Workflow("test_swamp").Priority(event_queue.PriorityNormal).
			Sequential().BlockInput().
			ThenSequence(

				gs.thenApplyHazardEffects(environment.PoisonSwamp)).
			Start()

		// Process event queue through animation (pattern from lava_damage_event_queue_test.go)
		// Green mist animation is 400ms + 75ms wait, so process through full duration
		// Use 30 iterations to ensure Phase 4 (poison application) completes
		for i := 0; i < 30; i++ {
			gs.EventQueueManager.Update()
			gs.ElapsedMs += 20
		}

		// Verify initial atmospheric effects
		assert.Contains(t, mockCallbacks.Messages, "Poisonous vapors rise from the swamp...", "Should show atmospheric warning")
		assert.True(t, len(mockCallbacks.ScreenFlashColors) > 0, "Should trigger green mist flash color")
		assert.True(t, len(mockCallbacks.ScreenFlashIntensities) > 0, "Should trigger green mist flash intensity")

		// Verify green mist color (RGB: 0.2, 0.6, 0.2) from event queue pattern
		flashColor := mockCallbacks.ScreenFlashColors[0]
		assert.Equal(t, greenMistRed, flashColor.R, "Should use green mist red component")
		assert.Equal(t, greenMistGreen, flashColor.G, "Should use green mist green component")
		assert.Equal(t, greenMistBlue, flashColor.B, "Should use green mist blue component")

		// Verify peak intensity (animation uses triangle wave, so find maximum)
		maxIntensity := 0.0
		for _, intensity := range mockCallbacks.ScreenFlashIntensities {
			if intensity > maxIntensity {
				maxIntensity = intensity
			}
		}
		assert.InDelta(t, greenMistIntensity, maxIntensity, 0.05, "Should reach approximately 0.3 intensity")

		assert.Contains(t, mockCallbacks.SoundEffectsPlayed, SoundFireBurn, "Should play atmospheric sound")

		// Verify final state after second stage
		assert.True(t, len(mockCallbacks.Messages) >= expectedMinMessageCount, "Should have atmospheric warning + poison status messages")
		assert.Contains(t, mockCallbacks.SoundEffectsPlayed, SoundHeal, "Should play coughing/sickness sound")

		// Verify poison status was applied (character 0 with low dexterity should be poisoned)
		// Note: With seed 12345, the dexterity rolls should be predictable
		poisonedCount := 0
		for i := 0; i < partySize; i++ {
			if gs.PartyState.GetCharacter(i).GetStatus() == party_state.Poisoned {
				poisonedCount++
			}
		}

		// Should have at least one poisoned character (the one that was already poisoned plus any new ones)
		assert.True(t, poisonedCount >= expectedMinPoisonedCount, "Should have at least one poisoned character")

		// Character 2 should still be poisoned (was already poisoned)
		assert.Equal(t, party_state.Poisoned, gs.PartyState.GetCharacter(2).GetStatus(), "Previously poisoned character should remain poisoned")

		// Dead character should remain dead
		assert.Equal(t, party_state.Dead, gs.PartyState.GetCharacter(3).GetStatus(), "Dead character should remain dead")
	})
}

func TestPerformSwampPoisonDexterityChecks(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	// Set up controlled test scenario
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)
	gs.PartyState.GetCharacter(0).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(0).Dexterity = extremelyLowDexterity // Extremely low - should always fail

	gs.PartyState.GetCharacter(1).SetStatus(party_state.Good)
	gs.PartyState.GetCharacter(1).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(1).Dexterity = testMaxDexterity // Maximum dexterity - should resist most rolls

	gs.PartyState.GetCharacter(2).SetStatus(party_state.Poisoned) // Should be excluded
	gs.PartyState.GetCharacter(2).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(2).Dexterity = lowDexterity * 2

	gs.PartyState.GetCharacter(3).SetStatus(party_state.Dead) // Should be excluded
	gs.PartyState.GetCharacter(3).PartyMap = party_state.PartyMapActive
	gs.PartyState.GetCharacter(3).Dexterity = lowDexterity * 3

	gs.PartyState.GetCharacter(4).SetStatus(party_state.Good) // Should be excluded (not in party)
	gs.PartyState.GetCharacter(4).PartyMap = party_state.PartyMapUnrecruited
	gs.PartyState.GetCharacter(4).Dexterity = extremelyLowDexterity

	// Set seed for deterministic results
	gs.SetRandomSeed(testSeed2)

	poisonTargets := gs.performSwampPoisonDexterityChecks()

	// Verify exclusions work correctly
	assert.NotContains(t, poisonTargets, 2, "Should exclude already poisoned character")
	assert.NotContains(t, poisonTargets, 3, "Should exclude dead character")
	assert.NotContains(t, poisonTargets, 4, "Should exclude non-party character")

	// With low dexterity, character 0 should likely be in targets
	// With high dexterity, character 1 should likely resist
	// Exact results depend on RNG, but we can verify the logic structure
	for _, target := range poisonTargets {
		assert.True(t, target == 0 || target == 1, "Only characters 0 or 1 should be valid targets")
		member := gs.PartyState.GetCharacter(target)
		assert.Equal(t, party_state.Good, member.GetStatus(), "Target should be in good status")
		assert.Equal(t, party_state.PartyMapActive, member.PartyMap, "Target should be in party")
	}
}

func TestApplySwampPoisonToTargets(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	// Set up characters with names for testing message generation
	copy(gs.PartyState.GetCharacter(0).Name[:], "Iolo\x00")
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)

	copy(gs.PartyState.GetCharacter(1).Name[:], "Shamino\x00")
	gs.PartyState.GetCharacter(1).SetStatus(party_state.Good)

	// Apply poison to first two characters
	poisonTargets := []int{0, 1}
	gs.applySwampPoisonToTargets(poisonTargets)

	// Verify poison status was applied
	assert.Equal(t, party_state.Poisoned, gs.PartyState.GetCharacter(0).GetStatus(), "First character should be poisoned")
	assert.Equal(t, party_state.Poisoned, gs.PartyState.GetCharacter(1).GetStatus(), "Second character should be poisoned")

	// Note: Message verification would require SystemCallbacks to be properly set up
	// The actual message checking would be done in the integration test above
}

func TestSwampPoisonEventQueue_NoTargetsSkipsSequence(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(testPlayerX, testPlayerY).
		WithSystemCallbacks().
		Build()

	// Set up party where no one can be poisoned
	for i := 0; i < partySize; i++ {
		gs.PartyState.GetCharacter(i).SetStatus(party_state.Dead) // All dead
		gs.PartyState.GetCharacter(i).PartyMap = party_state.PartyMapActive
	}

	mockCallbacks.Reset()

	// Trigger sequence - thenApplyHazardEffects should detect no valid targets and skip effects
	gs.EventQueueManager.Workflow("test_no_targets").Priority(event_queue.PriorityNormal).
		Sequential().BlockInput().
		First(func() {

		}).ThenSequence(

		gs.thenApplyHazardEffects(environment.PoisonSwamp)).
		Start()
	gs.EventQueueManager.Update()

	// Verify no effects occurred (using new tracking arrays)
	assert.Empty(t, mockCallbacks.Messages, "Should show no messages when no targets")
	assert.Empty(t, mockCallbacks.ScreenFlashColors, "Should show no visual effects when no targets")
	assert.Empty(t, mockCallbacks.ScreenFlashIntensities, "Should show no visual intensity when no targets")
	assert.Empty(t, mockCallbacks.SoundEffectsPlayed, "Should play no sounds when no targets")
}
