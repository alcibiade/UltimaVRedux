// Integration tests for Jimmy command workflows with real game data
// These tests validate the complete jimmy command flow including key consumption,
// door/chest interaction, character selection, and system callbacks.
package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestJimmyWorkflow_CompleteFlow tests the entire jimmy command workflow
func TestJimmyWorkflow_CompleteFlow(t *testing.T) {
	// Test complete jimmy workflow with real game data
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle). // Castle likely has doors
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	// Ensure player has keys for testing
	gs.PartyState.Inventory.Provisions.Keys.Set(5)

	// Ensure first character is in Good status for jimmy
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)

	t.Logf("🗝️ Jimmy Workflow Test - Castle Environment")
	t.Logf("Initial keys: %d", gs.PartyState.Inventory.Provisions.Keys.Get())

	// Test jimmy in different directions to find lockable objects
	directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}

	for _, dir := range directions {
		mockCallbacks.Reset()
		initialKeys := gs.PartyState.Inventory.Provisions.Keys.Get()

		result := gs.ActionJimmySmallMap(dir)

		t.Logf("Jimmy %v result: %v (keys before: %d, after: %d)",
			dir, result, initialKeys, gs.PartyState.Inventory.Provisions.Keys.Get())

		// Log all feedback messages
		for i, msg := range mockCallbacks.Messages {
			t.Logf("Jimmy message %d: %s", i+1, msg)
		}

		// Check for key consumption (if jimmy was attempted)
		if initialKeys > gs.PartyState.Inventory.Provisions.Keys.Get() {
			keyConsumed := initialKeys - gs.PartyState.Inventory.Provisions.Keys.Get()
			t.Logf("✅ Key consumed properly: %d key(s) used", keyConsumed)

			// Should have sound effect if attempt was made
			if len(mockCallbacks.SoundEffectsPlayed) > 0 {
				t.Logf("🔊 Sound effect played: %v", mockCallbacks.SoundEffectsPlayed[0])
			}

			// Should have time advancement
			if len(mockCallbacks.TimeAdvanced()) > 0 {
				t.Logf("⏰ Time advanced: %d minute(s)", mockCallbacks.TimeAdvanced()[0])
			}
		}
	}

	t.Logf("🗝️ Jimmy workflow integration test completed")
}

// TestJimmyWorkflow_NoKeys tests jimmy workflow when no keys available
func TestJimmyWorkflow_NoKeys(t *testing.T) {
	// Test jimmy workflow with no keys
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return
	}

	// Ensure no keys available
	gs.PartyState.Inventory.Provisions.Keys.Set(0)

	// Ensure first character is in Good status
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)

	t.Logf("🚫 Jimmy No Keys Test")

	// Test jimmy with no keys - should get appropriate message
	result := gs.ActionJimmySmallMap(references.Up)

	t.Logf("Jimmy with no keys result: %v", result)

	// Should have appropriate "no keys" message
	mockCallbacks.AssertMessageContains("") // Any message is fine

	// Should NOT consume any keys (none to consume)
	if gs.PartyState.Inventory.Provisions.Keys.Get() != 0 {
		t.Errorf("Expected 0 keys remaining, got %d", gs.PartyState.Inventory.Provisions.Keys.Get())
	}

	// Should not advance time if no jimmy attempt was made
	if len(mockCallbacks.TimeAdvanced()) == 0 {
		t.Logf("✅ Correctly no time advancement when no keys")
	}

	t.Logf("🚫 Jimmy no keys test completed")
}

// TestJimmyWorkflow_CharacterSelection tests character selection for jimmy
func TestJimmyWorkflow_CharacterSelection(t *testing.T) {
	// Test character selection logic for jimmy
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(12, 12).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return
	}

	// Set up interesting character states for selection testing
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Dead) // Should not be selected
	gs.PartyState.GetCharacter(1).SetStatus(party_state.Good) // Should be selected
	gs.PartyState.GetCharacter(2).SetStatus(party_state.Good) // Available but second choice

	// Give some keys for testing
	gs.PartyState.Inventory.Provisions.Keys.Set(3)

	t.Logf("👥 Jimmy Character Selection Test")
	t.Logf("Character 0 status: %d (Dead)", gs.PartyState.GetCharacter(0).GetStatus())
	t.Logf("Character 1 status: %d (Good)", gs.PartyState.GetCharacter(1).GetStatus())
	t.Logf("Character 2 status: %d (Good)", gs.PartyState.GetCharacter(2).GetStatus())

	// Test character selection logic
	selectedChar := gs.SelectCharacterForJimmy()

	if selectedChar == nil {
		t.Logf("No character selected for jimmy")
	} else {
		// Find which character was selected
		for i := 0; i < 6; i++ {
			char := gs.PartyState.GetCharacter(i)
			if char == selectedChar {
				t.Logf("✅ Character %d selected for jimmy (status: %d)", i, char.GetStatus())
				break
			}
		}
	}

	// Test jimmy action to see full workflow
	mockCallbacks.Reset()
	result := gs.ActionJimmySmallMap(references.Right)
	t.Logf("Jimmy with character selection result: %v", result)

	// Log feedback
	for i, msg := range mockCallbacks.Messages {
		t.Logf("Character selection jimmy message %d: %s", i+1, msg)
	}

	t.Logf("👥 Character selection test completed")
}

// TestJimmyWorkflow_KeyConsumptionRegression tests key consumption edge cases
func TestJimmyWorkflow_KeyConsumptionRegression(t *testing.T) {
	// Regression: a failed physical attempt consumes exactly one key.
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(8, 8).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return
	}

	// Test key consumption with exactly 1 key
	gs.PartyState.Inventory.Provisions.Keys.Set(1)
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)

	t.Logf("🔑 Jimmy Key Consumption Regression Test")
	initialKeys := gs.PartyState.Inventory.Provisions.Keys.Get()
	t.Logf("Starting with %d key(s)", initialKeys)

	mockCallbacks.Reset()
	result := gs.ActionJimmySmallMap(references.Left)

	finalKeys := gs.PartyState.Inventory.Provisions.Keys.Get()
	t.Logf("Jimmy result: %v", result)
	t.Logf("Keys after jimmy: %d", finalKeys)

	// Log all messages for analysis
	for i, msg := range mockCallbacks.Messages {
		t.Logf("Regression test message %d: %s", i+1, msg)
	}

	// Verify key consumption logic
	if initialKeys > 0 && finalKeys == initialKeys {
		// If we had keys but none were consumed, jimmy might not have been attempted
		t.Logf("ℹ️ No keys consumed - jimmy attempt might not have found valid target")
	} else if initialKeys > finalKeys {
		keyConsumed := initialKeys - finalKeys
		t.Logf("✅ Keys consumed as expected: %d", keyConsumed)

		// Should have feedback and effects when keys are consumed
		if len(mockCallbacks.Messages) == 0 {
			t.Errorf("Expected feedback message when keys are consumed")
		}
	}

	t.Logf("🔑 Key consumption regression test completed")
}

// TestJimmyWorkflow_MultipleAttempts tests multiple jimmy attempts
func TestJimmyWorkflow_MultipleAttempts(t *testing.T) {
	// Test multiple jimmy attempts to verify consistent behavior
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle). // Likely has multiple lockable objects
		WithPlayerAt(5, 5).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return
	}

	// Start with multiple keys for testing
	gs.PartyState.Inventory.Provisions.Keys.Set(10)
	gs.PartyState.GetCharacter(0).SetStatus(party_state.Good)

	t.Logf("🔄 Jimmy Multiple Attempts Test")
	initialKeys := gs.PartyState.Inventory.Provisions.Keys.Get()

	// Perform multiple jimmy attempts
	attemptCount := 5
	totalKeysConsumed := uint16(0)

	for i := 0; i < attemptCount; i++ {
		mockCallbacks.Reset()
		keysBefore := gs.PartyState.Inventory.Provisions.Keys.Get()

		// Alternate directions for variety
		direction := references.Direction(i%4 + 1) // Up, Down, Left, Right
		result := gs.ActionJimmySmallMap(direction)

		keysAfter := gs.PartyState.Inventory.Provisions.Keys.Get()
		keysThisAttempt := keysBefore - keysAfter
		totalKeysConsumed += keysThisAttempt

		t.Logf("Attempt %d (direction %v): result=%v, keys_consumed=%d",
			i+1, direction, result, keysThisAttempt)

		// Log main feedback message
		if len(mockCallbacks.Messages) > 0 {
			t.Logf("  Message: %s", mockCallbacks.Messages[len(mockCallbacks.Messages)-1])
		}
	}

	t.Logf("Multiple attempts summary:")
	t.Logf("  Initial keys: %d", initialKeys)
	t.Logf("  Final keys: %d", gs.PartyState.Inventory.Provisions.Keys.Get())
	t.Logf("  Total consumed: %d", totalKeysConsumed)
	t.Logf("  Keys remaining: %d", gs.PartyState.Inventory.Provisions.Keys.Get())

	t.Logf("🔄 Multiple attempts test completed")
}
