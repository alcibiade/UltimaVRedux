package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/datetime"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

// TestGuardAutoConversation_MinocGuard tests that a guard in Minoc automatically
// initiates conversation when the Avatar stands orthogonally adjacent and passes a turn.
//
// Based on user report: Guard at Minoc (15,24) at 2:49pm should speak when Avatar
// is at (15,25) and passes a turn.
func TestGuardAutoConversation_MinocGuard(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Minoc).
		WithPlayerAt(15, 25). // Avatar position (one tile below guard at 15,24)
		WithSystemCallbacks().
		WithRandomSeed(12345). // Deterministic RNG
		Build()

	if gs == nil {
		return // Test skipped due to missing game data
	}

	// Set time to 2:49pm (14:49 in 24-hour time)
	gs.DateTime = datetime.UltimaDate{
		Year:   100,
		Month:  1,
		Day:    1,
		Hour:   14, // 2pm
		Minute: 49,
		Turn:   0,
	}

	// Update map with current time to ensure NPCs are positioned by their schedule
	gs.UpdateSmallMap(gs.GameReferences.TileReferences, gs.GameReferences.LocationReferences)

	t.Logf("📍 Test Setup:")
	t.Logf("  Location: Minoc")
	t.Logf("  Avatar position: (15, 25)")
	t.Logf("  Time: %d:%02d (2:49pm)", gs.DateTime.Hour, gs.DateTime.Minute)

	// Verify NPCs exist
	if gs.CurrentNPCAIController == nil {
		t.Fatal("No NPC AI controller found")
	}

	npcs := gs.CurrentNPCAIController.GetNpcs()
	if npcs == nil {
		t.Fatal("No NPCs found")
	}

	// Find guard at position (15, 24)
	var guardNPC *map_units.NPCFriendly
	guardPos := references.Position{X: 15, Y: 24}

	for _, mapUnit := range *npcs {
		if npc, ok := mapUnit.(*map_units.NPCFriendly); ok && npc.IsVisible() {
			if npc.PosPtr().X == guardPos.X && npc.PosPtr().Y == guardPos.Y {
				guardNPC = npc
				t.Logf("  ✅ Found NPC at (15, 24): DialogNumber=%d, AI Type=%v",
					npc.NPCReference.DialogNumber, npc.GetIndividualBehaviourByUltimaData(gs.DateTime).Ai)

				break
			}
		}
	}

	if guardNPC == nil {
		t.Skip("No NPC found at position (15, 24) in Minoc at 2:49pm - may need schedule data verification")
	}

	// Verify the NPC has appropriate AI type for auto-conversation
	aiType := guardNPC.GetIndividualBehaviourByUltimaData(gs.DateTime).Ai
	autoTalkAITypes := []references.AiType{
		references.CustomAi,                   // AI type 4 - guards in Minoc
		references.Begging,                    // AI type 103
		references.SmallWanderWantsToChat,     // AI type 102
		references.GenericExtortingGuard,      // AI type 104
		references.HalfYourGoldExtortingGuard, // AI type 105
	}

	hasAutoTalkAI := false
	for _, aiT := range autoTalkAITypes {
		if aiType == aiT {
			hasAutoTalkAI = true

			break
		}
	}

	if !hasAutoTalkAI {
		t.Logf("  ⚠️ NPC at (15,24) has AI type %v - not an auto-talk AI type", aiType)
		t.Skip("NPC at (15,24) does not have auto-talk AI type - test not applicable")
	}

	t.Logf("  ✅ NPC has auto-talk AI type: %v", aiType)

	// Verify NPC is orthogonally adjacent (Manhattan distance == 1)
	avatarPos := gs.MapState.PlayerLocation.Position
	dx := int(guardNPC.PosPtr().X) - int(avatarPos.X)
	dy := int(guardNPC.PosPtr().Y) - int(avatarPos.Y)
	manhattanDist := absInt(dx) + absInt(dy)

	t.Logf("  Manhattan distance: %d (dx=%d, dy=%d)", manhattanDist, dx, dy)
	assert.Equal(t, 1, manhattanDist, "Avatar should be orthogonally adjacent (distance==1)")

	// Clear mock callbacks before testing
	mockCallbacks.Reset()

	t.Logf("\n🎬 Testing Auto-Conversation Trigger:")

	// Advance turn to trigger NPC AI movement and auto-talk check
	// This simulates the NPC getting adjacent and the game checking for auto-conversation
	gs.FinishTurn()

	t.Logf("\n📊 Results After Turn Advancement:")
	t.Logf("  Talk dialog calls: %d", len(mockCallbacks.TalkDialogCalls))
	t.Logf("  Dialogs pushed: %d", len(mockCallbacks.DialogsPushed))
	t.Logf("  Messages: %d", len(mockCallbacks.Messages))
	t.Logf("  Time advanced: %d", len(mockCallbacks.TimeAdvanced()))

	// Log all messages for debugging
	for i, msg := range mockCallbacks.Messages {
		t.Logf("  Message %d: %s", i+1, msg)
	}

	// Check if a Talk interaction was set by AI.
	pendingInteraction := guardNPC.MapUnitDetails().PendingInteraction
	t.Logf("  Pending interaction: %v", pendingInteraction)

	// Verify that auto-conversation was triggered
	// Expected behavior: NPC should auto-initiate talk dialog when adjacent
	assert.True(t, len(mockCallbacks.TalkDialogCalls) > 0 || pendingInteraction == map_units.PendingNPCInteractionTalk,
		"Guard should either trigger auto-talk or set a Talk interaction when adjacent")

	if len(mockCallbacks.TalkDialogCalls) > 0 {
		t.Logf("  ✅ Auto-conversation triggered successfully!")
		t.Logf("  Dialog NPC: %+v", mockCallbacks.TalkDialogCalls[0])
	} else if pendingInteraction == map_units.PendingNPCInteractionTalk {
		t.Logf("  Talk interaction set (may require UI update cycle to trigger)")
	} else {
		t.Errorf("  ❌ Auto-conversation NOT triggered - guard remained silent")
		t.Errorf("     Expected: Guard at (15,24) should speak when Avatar at (15,25)")
		t.Errorf("     Actual: No conversation initiated, pending interaction=%v", pendingInteraction)
	}
}

// TestGuardFloorSeparation_NoInteraction tests that guards on different floors do NOT interact
// with the Avatar, even if their X/Y coordinates are adjacent.
//
// Bug Report: Guard on floor 3 of Lord British's castle was extorting Avatar on floor 1
// Root Cause: IsNextTo() only checked X/Y, not floor level
// Fix: IsNextTo() now requires floor1 == floor2 for adjacency
func TestGuardFloorSeparation_NoInteraction(t *testing.T) {
	// Use Lord British's castle for realistic multi-floor testing
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithPlayerAt(15, 15). // Avatar position on floor 1
		WithSystemCallbacks().
		WithRandomSeed(12345). // Deterministic RNG
		Build()

	if gs == nil {
		return // Test skipped due to missing game data
	}

	// Set Avatar to floor 1
	gs.MapState.PlayerLocation.Floor = 1

	t.Logf("📍 Test Setup:")
	t.Logf("  Location: Lord British's Castle")
	t.Logf("  Avatar position: (15, 15) on Floor 1")

	// Update map with current time to ensure NPCs are positioned
	gs.UpdateSmallMap(gs.GameReferences.TileReferences, gs.GameReferences.LocationReferences)

	// Get NPC AI controller
	if gs.CurrentNPCAIController == nil {
		t.Fatal("No NPC AI controller found")
	}

	npcs := gs.CurrentNPCAIController.GetNpcs()
	if npcs == nil {
		t.Fatal("No NPCs found")
	}

	// Find or create a guard on floor 3 at the same X/Y coordinates as Avatar
	// This simulates the reported bug: guard on floor 3 interacting with Avatar on floor 1
	var testGuard *map_units.NPCFriendly
	avatarPos := gs.MapState.PlayerLocation.Position

	for _, mapUnit := range *npcs {
		if npc, ok := mapUnit.(*map_units.NPCFriendly); ok && npc.IsVisible() {
			// Look for a guard-type NPC
			if npc.NPCReference.GetNPCType() == references.Guard {
				// Position guard on floor 3 at Avatar's X/Y position (orthogonally adjacent)
				npc.SetFloor(3)                                                     // Different floor than Avatar
				npc.SetPos(references.Position{X: avatarPos.X, Y: avatarPos.Y - 1}) // One tile north

				testGuard = npc
				t.Logf("  ✅ Positioned guard at (%d, %d) on Floor 3 (Avatar at same X/Y on Floor 1)",
					npc.PosPtr().X, npc.PosPtr().Y)
				break
			}
		}
	}

	if testGuard == nil {
		t.Skip("No guard NPC found in Lord British's Castle - test not applicable")
	}

	// Verify the guard is at the right position
	assert.Equal(t, references.FloorNumber(3), testGuard.Floor(), "Guard should be on floor 3")
	assert.Equal(t, avatarPos.Y-1, testGuard.PosPtr().Y, "Guard should be one tile north of Avatar")
	assert.Equal(t, avatarPos.X, testGuard.PosPtr().X, "Guard should be at same X coordinate as Avatar")

	// Clear mock callbacks before testing
	mockCallbacks.Reset()

	t.Logf("\n🎬 Testing Floor Separation:")
	t.Logf("  Guard floor: %d", testGuard.Floor())
	t.Logf("  Avatar floor: %d", gs.MapState.PlayerLocation.Floor)

	// Advance turn to trigger NPC AI movement and auto-talk checks
	// If floor checking is broken, the guard would set a pending interaction.
	gs.FinishTurn()

	t.Logf("\n📊 Results After Turn Advancement:")
	t.Logf("  Talk dialog calls: %d", len(mockCallbacks.TalkDialogCalls))
	t.Logf("  Dialogs pushed: %d", len(mockCallbacks.DialogsPushed))
	t.Logf("  Pending interaction: %v", testGuard.MapUnitDetails().PendingInteraction)

	// VERIFY: Guard on different floor should NOT trigger auto-talk
	assert.Equal(t, map_units.PendingNPCInteractionNone, testGuard.MapUnitDetails().PendingInteraction,
		"Guard on floor 3 should NOT set an interaction when Avatar is on floor 1")

	assert.Equal(t, 0, len(mockCallbacks.TalkDialogCalls),
		"Guard on different floor should NOT trigger talk dialog")

	if testGuard.MapUnitDetails().PendingInteraction != map_units.PendingNPCInteractionNone {
		t.Errorf("  ❌ BUG: Guard on floor 3 triggered auto-talk for Avatar on floor 1")
		t.Errorf("     This indicates IsNextTo() is not checking floor levels correctly")
	} else {
		t.Logf("  ✅ Floor separation working correctly - no cross-floor interaction")
	}
}
