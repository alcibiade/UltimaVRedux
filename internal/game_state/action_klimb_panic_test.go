package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestKlimbFencePanicRegression prevents regression of the klimbing fence panic bug.
//
// BUG FIXED: When pressing K on small map, ActionKlimbSmallMap() would crash with nil pointer
// dereference if GetNewPositionInDirection() returned nil (invalid direction or out-of-bounds).
//
// ORIGINAL ERROR: "runtime error: invalid memory address or nil pointer dereference"
// ROOT CAUSE: Line 47 in action_klimb.go didn't check if newPosition was nil before using it
// FIX: Added nil check and graceful handling before dereferencing newPosition
//
// This test ensures the panic never returns by testing edge conditions.
func TestKlimbFencePanicRegression(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.NorthBritanny).
		WithPlayerAt(0, 0). // At map edge
		WithSystemCallbacks().
		Build()

	if gameState == nil {
		t.Skip("Skipping test - real game data not available")
		return
	}

	// Set deterministic seed
	gameState.SetRandomSeed(12345)

	t.Log("Testing klimb at map boundary to reproduce panic")

	// This should NOT panic - currently causes nil pointer dereference
	// when direction.GetNewPositionInDirection returns nil for out-of-bounds
	t.Run("KlimbLeft_AtMapEdge_ShouldNotPanic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("PANIC REPRODUCED: klimbing left at map edge caused panic: %v", r)
				t.Errorf("This confirms the bug - GetNewPositionInDirection returned nil but wasn't checked")
			}
		}()

		result := gameState.ActionKlimbSmallMap(references.Left)

		// Should handle gracefully and return false
		if result {
			t.Error("Expected klimb to fail at map boundary, but it succeeded")
		}

		t.Log("SUCCESS: klimb handled map boundary gracefully without panic")
	})

	t.Run("KlimbUp_AtMapEdge_ShouldNotPanic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("PANIC REPRODUCED: klimbing up at map edge caused panic: %v", r)
			}
		}()

		result := gameState.ActionKlimbSmallMap(references.Up)

		// Should handle gracefully and return false
		if result {
			t.Error("Expected klimb to fail at map boundary, but it succeeded")
		}
	})
}

// TestKlimbDirectionZeroPanic prevents regression of the initial K key press panic.
//
// BUG FIXED: When pressing K initially, smallMapKlimb() calls ActionKlimbSmallMap(Direction(0))
// to check for ladders at current position. Direction(0) is NoneDirection which causes
// GetNewPositionInDirection() to return nil, leading to a panic.
//
// REPRODUCTION STEPS: Press K key on small map → immediate panic
// FIX: Added nil check in ActionKlimbSmallMap() before dereferencing newPosition
//
// This test ensures pressing K initially never causes a panic.
func TestKlimbDirectionZeroPanic(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.NorthBritanny).
		WithPlayerAt(15, 15). // Safe middle position
		WithSystemCallbacks().
		Build()
	gs.SetRandomSeed(12345)

	t.Run("KlimbWithDirectionZero_ShouldHandleGracefully", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("PANIC: klimbing with direction 0 caused panic: %v", r)
			}
		}()

		// This is what gets called when K is pressed initially (checking for ladders)
		result := gs.ActionKlimbSmallMap(references.Direction(0))

		// Should either succeed (if on ladder) or fail gracefully
		t.Logf("Klimb with direction 0 result: %v", result)
	})
}
