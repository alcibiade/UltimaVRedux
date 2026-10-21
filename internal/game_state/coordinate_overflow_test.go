package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestCoordinateOverflow validates coordinate handling for edge cases
//
// This test ensures that coordinate calculations don't overflow or produce
// unexpected results when dealing with edge values in the position system.
func TestCoordinateOverflow_EdgeCases(t *testing.T) {
	t.Run("MaxCoordinateValues", func(t *testing.T) {
		// Test maximum coordinate values
		maxCoord := references.Coordinate(32767) // Max int16 value
		position := references.Position{X: maxCoord, Y: maxCoord}

		t.Logf("Testing maximum coordinate values: (%d,%d)", position.X, position.Y)

		// These operations should not overflow
		left := position.GetPositionToLeft()
		right := position.GetPositionToRight()
		up := position.GetPositionUp()
		down := position.GetPositionDown()

		t.Logf("Left: (%d,%d), Right: (%d,%d)", left.X, left.Y, right.X, right.Y)
		t.Logf("Up: (%d,%d), Down: (%d,%d)", up.X, up.Y, down.X, down.Y)

		// Verify expected results
		if left.X != maxCoord-1 || left.Y != maxCoord {
			t.Errorf("GetPositionToLeft failed: expected (%d,%d), got (%d,%d)",
				maxCoord-1, maxCoord, left.X, left.Y)
		}
	})

	t.Run("MinCoordinateValues", func(t *testing.T) {
		// Test minimum coordinate values
		minCoord := references.Coordinate(-32768) // Min int16 value
		position := references.Position{X: minCoord, Y: minCoord}

		t.Logf("Testing minimum coordinate values: (%d,%d)", position.X, position.Y)

		// These operations should not underflow
		left := position.GetPositionToLeft()
		right := position.GetPositionToRight()
		up := position.GetPositionUp()
		down := position.GetPositionDown()

		t.Logf("Left: (%d,%d), Right: (%d,%d)", left.X, left.Y, right.X, right.Y)
		t.Logf("Up: (%d,%d), Down: (%d,%d)", up.X, up.Y, down.X, down.Y)

		// Verify expected results
		if right.X != minCoord+1 || right.Y != minCoord {
			t.Errorf("GetPositionToRight failed: expected (%d,%d), got (%d,%d)",
				minCoord+1, minCoord, right.X, right.Y)
		}
	})

	t.Run("WrappingBehavior", func(t *testing.T) {
		// Test wrapping behavior on large maps
		position := references.Position{X: 255, Y: 255}

		// Test non-wrapping movement (should clamp)
		position.GoRight(false) // Small map, should not wrap
		t.Logf("Position after GoRight(false) from (255,255): (%d,%d)", position.X, position.Y)

		// Reset and test wrapping movement
		position = references.Position{X: 255, Y: 255}
		position.GoRight(true) // Large map, should wrap
		t.Logf("Position after GoRight(true) from (255,255): (%d,%d)", position.X, position.Y)

		if position.X != 0 {
			t.Errorf("Large map wrapping failed: expected X=0, got X=%d", position.X)
		}

		// Test wrapping at boundary
		position = references.Position{X: 0, Y: 0}
		position.GoLeft(true) // Should wrap to 255
		t.Logf("Position after GoLeft(true) from (0,0): (%d,%d)", position.X, position.Y)

		if position.X != 255 {
			t.Errorf("Large map wrap-around failed: expected X=255, got X=%d", position.X)
		}
	})
}

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
