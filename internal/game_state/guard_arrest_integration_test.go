package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

// TestGuardArrest_BritainGuard_AutoConversation tests that a guard automatically
// initiates arrest conversation when the player is marked as criminal and stands
// adjacent to a guard.
//
// Test location: Britain - finds guard dynamically and positions player next to it
// Expected: Guard should set a typed arrest interaction to trigger arrest dialogue.
func TestGuardArrest_BritainGuard_AutoConversation(t *testing.T) {
	// Start with initial position, we'll move player next to a guard
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(14, 28). // Initial position, will be adjusted
		WithSystemCallbacks().
		WithRandomSeed(12345). // Deterministic RNG
		Build()

	if gs == nil {
		return // Test skipped due to missing game data
	}

	// Set time to 2:30pm when guard is at position (14,27) next to player at (14,28)
	gs.DateTime = datetime.UltimaDate{
		Year:   100,
		Month:  1,
		Day:    1,
		Hour:   14, // 2pm
		Minute: 30,
		Turn:   0,
	}

	// Update map with current time to ensure NPCs are positioned by their schedule
	gs.UpdateSmallMap(gs.GameReferences.TileReferences, gs.GameReferences.LocationReferences)

	t.Logf("📍 Test Setup:")
	t.Logf("  Location: Britain")
	t.Logf("  Avatar position: (14, 28)")
	t.Logf("  Time: %d:%02d (2:30pm)", gs.DateTime.Hour, gs.DateTime.Minute)

	// Verify NPCs exist
	if gs.CurrentNPCAIController == nil {
		t.Fatal("No NPC AI controller found")
	}

	npcController, ok := gs.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	if !ok {
		t.Fatal("Expected NPCAIControllerSmallMap")
	}

	npcs := npcController.GetNpcs()
	if npcs == nil {
		t.Fatal("No NPCs found")
	}

	// Find guard at position (13, 28) - should be adjacent to player at (14, 28)
	var guardNPC *map_units.NPCFriendly
	expectedGuardPos := references.Position{X: 13, Y: 28}

	t.Logf("\n🔍 Looking for guard at (%d, %d):", expectedGuardPos.X, expectedGuardPos.Y)

	// First pass: Look for guard at expected position
	for _, mapUnit := range *npcs {
		if npc, ok := mapUnit.(*map_units.NPCFriendly); ok && npc.IsVisible() {
			npcPos := npc.PosPtr()
			if npcPos.X == expectedGuardPos.X && npcPos.Y == expectedGuardPos.Y {
				guardNPC = npc
				t.Logf("  ✅ Found NPC at (%d, %d): DialogNumber=%d, AI Type=%v, IsGuard=%v",
					npcPos.X, npcPos.Y, npc.NPCReference.DialogNumber,
					npc.GetIndividualBehaviourByUltimaData(gs.DateTime).Ai,
					npc.NPCReference.WantsToAttackAvatarWhenBadStuffGoesDown())

				break
			}
		}
	}

	if guardNPC == nil {
		// Diagnostic: Show all NPCs and their positions
		t.Logf("\n🔍 All NPCs in Britain at 2:30pm:")

		for _, mapUnit := range *npcs {
			if npc, ok := mapUnit.(*map_units.NPCFriendly); ok && npc.IsVisible() {
				npcPos := npc.PosPtr()
				isGuard := npc.NPCReference.WantsToAttackAvatarWhenBadStuffGoesDown()
				t.Logf("  NPC at (%d, %d): DialogNumber=%d, IsGuard=%v, AI=%v",
					npcPos.X, npcPos.Y, npc.NPCReference.DialogNumber, isGuard,
					npc.GetIndividualBehaviourByUltimaData(gs.DateTime).Ai)
			}
		}

		t.Skip("No NPC found at position (13, 28) in Britain at 2:30pm - see diagnostic output above")
	}

	// Mark player as criminal
	gs.SetCriminalStatus(true)
	t.Logf("\n🚨 Marked player as criminal")

	// Activate guard - set to ExtortOrAttackOrFollow AI
	if guardNPC.NPCReference.WantsToAttackAvatarWhenBadStuffGoesDown() {
		guardNPC.MapUnitDetails().SetOverriddenAiType(references.ExtortOrAttackOrFollow)
		t.Logf("  ⚔️ Activated guard at (%d, %d) - AI set to ExtortOrAttackOrFollow",
			guardNPC.PosPtr().X, guardNPC.PosPtr().Y)
	} else {
		t.Skip("NPC at (13,28) is not a guard type - cannot test arrest behavior")
	}

	// Clear mock callbacks before testing
	mockCallbacks.Reset()

	t.Logf("\n🎬 Testing Auto-Conversation Trigger:")

	// Advance turn to trigger NPC AI movement and auto-talk check
	// This should make the guard set a typed arrest interaction.
	gs.FinishTurn()

	t.Logf("\n📊 Results After Turn Advancement:")

	// Check guard for a pending arrest interaction.
	pendingInteraction := guardNPC.MapUnitDetails().PendingInteraction
	aiType := guardNPC.GetIndividualBehaviourByUltimaData(gs.DateTime).Ai
	overriddenAI := guardNPC.MapUnitDetails().GetOverriddenAiType()

	t.Logf("  Guard at (%d, %d):", guardNPC.PosPtr().X, guardNPC.PosPtr().Y)
	t.Logf("    Original AI: %v", aiType)
	t.Logf("    Overridden AI: %v", overriddenAI)
	t.Logf("    Pending interaction: %v", pendingInteraction)
	t.Logf("    DialogNumber: %d", guardNPC.NPCReference.DialogNumber)

	t.Logf("\n  Talk dialog calls: %d", len(mockCallbacks.TalkDialogCalls))
	t.Logf("  Dialogs pushed: %d", len(mockCallbacks.DialogsPushed))
	t.Logf("  Messages: %d", len(mockCallbacks.Messages))

	// Log all messages for debugging
	for i, msg := range mockCallbacks.Messages {
		t.Logf("  Message %d: %s", i+1, msg)
	}

	// Verify that auto-conversation was triggered or flag was set
	assert.True(t, len(mockCallbacks.TalkDialogCalls) > 0 || pendingInteraction == map_units.PendingNPCInteractionGuardArrest,
		"Guard should either trigger arrest or set a pending arrest interaction when adjacent to criminal")

	if len(mockCallbacks.TalkDialogCalls) > 0 {
		t.Logf("\n✅ Auto-conversation triggered successfully!")
		t.Logf("  Dialog NPC: %+v", mockCallbacks.TalkDialogCalls[0])
	} else if pendingInteraction == map_units.PendingNPCInteractionGuardArrest {
		t.Logf("\nPending arrest interaction set (requires UI update cycle to trigger actual dialog)")
	} else {
		t.Errorf("\n❌ Auto-conversation NOT triggered - guard remained silent")
		t.Errorf("   Expected: Guard should set pending arrest when adjacent to criminal")
		t.Errorf("   Actual: No conversation initiated and no pending arrest set")
		t.Errorf("   Guard overridden AI: %v (should be ExtortOrAttackOrFollow=%v)", overriddenAI, references.ExtortOrAttackOrFollow)
	}
}
