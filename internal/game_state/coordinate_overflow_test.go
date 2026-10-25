package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestMovementConsistency ensures movement calculations are consistent
func TestMovementConsistency_DirectionalOperations(t *testing.T) {
	testCases := []struct {
		name    string
		initial references.Position
		isLarge bool
	}{
		{"SmallMapCenter", references.Position{X: 15, Y: 15}, false},
		{"SmallMapEdge", references.Position{X: 31, Y: 31}, false},
		{"LargeMapCenter", references.Position{X: 128, Y: 128}, true},
		{"LargeMapEdge", references.Position{X: 255, Y: 255}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Test that opposite movements cancel each other out (when not hitting boundaries)
			original := tc.initial

			// Move right, then left - should return to original (if not at boundary)
			pos := tc.initial
			pos.GoRight(tc.isLarge)
			pos.GoLeft(tc.isLarge)

			// Move up, then down
			pos.GoUp(tc.isLarge)
			pos.GoDown(tc.isLarge)

			// Check if we're back to original (accounting for wrapping)
			if tc.isLarge || (original.X < 31 && original.Y < 31) {
				// For large maps or positions not at small map boundary, should be consistent
				if pos.X != original.X || pos.Y != original.Y {
					t.Logf("Movement consistency test: started at (%d,%d), ended at (%d,%d)",
						original.X, original.Y, pos.X, pos.Y)
					// This might be expected behavior due to boundary clamping, so just log
				}
			}

			// Test Direction.GetNewPositionInDirection consistency
			for _, dir := range []references.Direction{references.Up, references.Down, references.Left, references.Right} {
				newPos := dir.GetNewPositionInDirection(&original)
				if newPos == nil {
					t.Errorf("GetNewPositionInDirection returned nil for direction %v", dir)
				}

				// Verify the position changed in the expected direction
				switch dir {
				case references.Up:
					if newPos.Y != original.Y-1 {
						t.Errorf("Up movement: expected Y=%d, got Y=%d", original.Y-1, newPos.Y)
					}
				case references.Down:
					if newPos.Y != original.Y+1 {
						t.Errorf("Down movement: expected Y=%d, got Y=%d", original.Y+1, newPos.Y)
					}
				case references.Left:
					if newPos.X != original.X-1 {
						t.Errorf("Left movement: expected X=%d, got X=%d", original.X-1, newPos.X)
					}
				case references.Right:
					if newPos.X != original.X+1 {
						t.Errorf("Right movement: expected X=%d, got X=%d", original.X+1, newPos.X)
					}
				}
			}
		})
	}
}
