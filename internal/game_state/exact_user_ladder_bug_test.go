// Exact reproduction of user's ladder bug scenario
// This test follows the precise multistep plan provided by the user
package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestExactUserLadderBugScenario follows the user's exact 10-step plan
func TestExactUserLadderBugScenario(t *testing.T) {
	// User's exact plan:
	// 1. start at x=1,y=27 britain floor 0
	// 2. move up one tile to ladder
	// 3. confirm tile is correct
	// 4. klimb the ladder
	// 5. confirm you are on floor 1
	// 6. confirm ladder tile is correct
	// 7. move up one tile
	// 8. confirm ladder tile is now LadderDown sprite
	// 9. move avatar back down one tile and confirm tile is correct KlimbLadderDown
	// 10. klimb back down, confirm you are on floor 0

	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(1, 27). // Step 1: start at x=1,y=27 britain floor 0
		WithSystemCallbacks().
		Build()
	gs.SetRandomSeed(12345) // Deterministic behavior

	ladderPos := references.Position{X: 1, Y: 26}

	t.Logf("=== EXACT USER LADDER BUG REPRODUCTION ===")
	t.Logf("Step 1: Starting at position (1,27) floor 0")

	// Step 2: move up one tile to ladder
	t.Logf("Step 2: Moving up one tile to ladder")
	gs.MapState.PlayerLocation.Position = ladderPos
	t.Logf("Player now at (%d,%d)", gs.MapState.PlayerLocation.Position.X, gs.MapState.PlayerLocation.Position.Y)

	// Step 3: confirm tile is correct
	t.Logf("Step 3: Confirming tile is correct")
	baseTile := gs.GetLayeredMapByCurrentLocation().GetTileTopMapOnlyTile(&ladderPos)
	if baseTile == nil || !baseTile.Is(indexes.LadderUp) {
		t.Fatalf("Expected LadderUp tile, got: %v", baseTile)
	}
	t.Logf("✅ Confirmed: Base tile is LadderUp (%d)", baseTile.Index)

	// Check what sprite shows when avatar is ON the ladder
	spriteOnLadder := gs.GetDisplaySpriteAtPosition(ladderPos)
	t.Logf("Sprite when avatar ON ladder: %d", spriteOnLadder)

	// Step 4: klimb the ladder
	t.Logf("Step 4: Klimbing the ladder")
	mockCallbacks.Reset()
	success := gs.ActionKlimbSmallMap(references.Up)
	if !success {
		t.Fatalf("Failed to klimb ladder")
	}
	// Process the event queue to complete the ENTIRE floor transition workflow
	// The workflow includes: fade out (250ms), floor change, fade in (250ms), time advance, cleanup
	// Need to advance ElapsedMs to simulate time passing for animations
	// Must wait for queue to fully complete, not just floor change
	for range 100 {
		gs.EventQueueManager.Update()
		gs.ElapsedMs += 20 // Simulate ~60fps (16.7ms per frame), rounded up
		if !gs.EventQueueManager.HasWorkflow("klimb_floor") && gs.MapState.PlayerLocation.Floor == 1 {
			break // Workflow fully completed
		}
	}
	t.Logf("✅ Klimb successful. Messages: %v", mockCallbacks.Messages)

	// Step 5: confirm you are on floor 1
	t.Logf("Step 5: Confirming on floor 1")
	if gs.MapState.PlayerLocation.Floor != 1 {
		t.Fatalf("Expected floor 1, got floor %d", gs.MapState.PlayerLocation.Floor)
	}
	t.Logf("✅ Confirmed: On floor 1")

	// Step 6: confirm ladder tile is correct
	t.Logf("Step 6: Confirming ladder tile is correct on floor 1")
	newBaseTile := gs.GetLayeredMapByCurrentLocation().GetTileTopMapOnlyTile(&ladderPos)
	if newBaseTile == nil {
		t.Fatalf("No base tile found at ladder position on floor 1")
	}
	t.Logf("Base tile on floor 1: %s (%d)", newBaseTile.Name, newBaseTile.Index)

	// What sprite shows when avatar is on the ladder on floor 1?
	spriteOnLadderFloor1 := gs.GetDisplaySpriteAtPosition(ladderPos)
	t.Logf("Sprite when avatar ON ladder floor 1: %d", spriteOnLadderFloor1)

	// Step 7: move up one tile
	t.Logf("Step 7: Moving up one tile (away from ladder)")
	northPos := references.Position{X: 1, Y: 25}
	gs.MapState.PlayerLocation.Position = northPos
	t.Logf("Player now at (%d,%d)", gs.MapState.PlayerLocation.Position.X, gs.MapState.PlayerLocation.Position.Y)

	// Step 8: confirm ladder tile is now LadderDown sprite
	t.Logf("Step 8: Confirming ladder tile shows LadderDown sprite")
	spriteWhenAway := gs.GetDisplaySpriteAtPosition(ladderPos)
	t.Logf("🔍 CRITICAL: Sprite when avatar AWAY from ladder: %d", spriteWhenAway)

	// The user reports this should show LadderDown (201) but instead shows avatar sprite
	if spriteWhenAway == indexes.LadderDown {
		t.Logf("✅ CORRECT: Ladder shows LadderDown (%d) when avatar away", spriteWhenAway)
	} else if spriteWhenAway == indexes.KlimbLadderUp || spriteWhenAway == indexes.KlimbLadderDown {
		t.Errorf("🐛 USER BUG CONFIRMED: Ladder shows climbing sprite (%d) when avatar away!", spriteWhenAway)
		t.Errorf("   Expected: LadderDown (%d)", indexes.LadderDown)
		t.Errorf("   This prevents avatar from accessing the ladder again!")
	} else {
		t.Logf("⚠️  Unexpected sprite when away: %d", spriteWhenAway)
	}

	// Step 9: move avatar back down one tile and confirm tile is correct KlimbLadderDown
	t.Logf("Step 9: Moving avatar back down to ladder")
	gs.MapState.PlayerLocation.Position = ladderPos
	t.Logf("Player back on ladder at (%d,%d)", gs.MapState.PlayerLocation.Position.X, gs.MapState.PlayerLocation.Position.Y)

	spriteWhenReturn := gs.GetDisplaySpriteAtPosition(ladderPos)
	t.Logf("Sprite when avatar returns to ladder: %d", spriteWhenReturn)

	if spriteWhenReturn == indexes.KlimbLadderDown {
		t.Logf("✅ CORRECT: Avatar on ladder shows KlimbLadderDown (%d)", spriteWhenReturn)
	} else {
		t.Logf("⚠️  Unexpected sprite when avatar returns: %d (expected KlimbLadderDown %d)",
			spriteWhenReturn, indexes.KlimbLadderDown)
	}

	// Step 10: klimb back down, confirm you are on floor 0
	t.Logf("Step 10: Klimbing back down to floor 0")
	mockCallbacks.Reset()
	successDown := gs.ActionKlimbSmallMap(references.Down)
	if !successDown {
		t.Fatalf("Failed to klimb down ladder")
	}
	// Process the event queue to complete the ENTIRE floor transition workflow
	// Must wait for queue to fully complete, not just floor change
	for range 100 {
		gs.EventQueueManager.Update()
		gs.ElapsedMs += 20 // Simulate ~60fps
		if !gs.EventQueueManager.HasWorkflow("klimb_floor") && gs.MapState.PlayerLocation.Floor == 0 {
			break // Workflow fully completed
		}
	}
	t.Logf("✅ Klimb down successful. Messages: %v", mockCallbacks.Messages)

	if gs.MapState.PlayerLocation.Floor != 0 {
		t.Fatalf("Expected floor 0, got floor %d", gs.MapState.PlayerLocation.Floor)
	}
	t.Logf("✅ Confirmed: Back on floor 0")

	t.Logf("=== USER SCENARIO COMPLETE ===")
}
