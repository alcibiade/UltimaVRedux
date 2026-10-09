package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestIsOutOfBounds_InconsistentLargeMapBounds reproduces a bounds checking bug
//
// BUG DESCRIPTION:
// The IsOutOfBounds function has inconsistent bounds checking for large maps:
// - X axis: position.X > references.XLargeMapTiles-1 (uses > with -1)
// - Y axis: position.Y >= references.YLargeMapTiles    (uses >= without -1)
//
// This means:
// - X coordinate 255 is considered out of bounds (should be valid for 0-255 range)
// - Y coordinate 255 triggers log.Fatal (inconsistent with X coordinate handling)
//
// For a 256x256 map (coordinates 0-255), both axes should use the same bounds logic.
//
// ROOT CAUSE:
// Inconsistent comparison operators and -1 usage between X and Y bounds checking.
//
// EXPECTED BEHAVIOR:
// Both should either use:
// - position.X >= references.XLargeMapTiles && position.Y >= references.YLargeMapTiles
// OR
// - position.X > references.XLargeMapTiles-1 && position.Y > references.YLargeMapTiles-1
func TestIsOutOfBounds_InconsistentLargeMapBounds_Regression(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld). // Large map
		WithSystemCallbacks().
		Build()
	t.Logf("=== LARGE MAP BOUNDS CONSISTENCY TEST ===")
	t.Logf("Large map size: %dx%d (coordinates 0-%d, 0-%d)",
		references.XLargeMapTiles, references.YLargeMapTiles,
		references.XLargeMapTiles-1, references.YLargeMapTiles-1)

	// Test edge cases that should have consistent behavior
	testCases := []struct {
		name     string
		position references.Position
		expected string
	}{
		{
			name:     "Max valid X coordinate (255,0)",
			position: references.Position{X: references.XLargeMapTiles - 1, Y: 0},
			expected: "valid (both coordinates within range)",
		},
		{
			name:     "Max valid Y coordinate (0,255)",
			position: references.Position{X: 0, Y: references.YLargeMapTiles - 1},
			expected: "valid (both coordinates within range)",
		},
		{
			name:     "Max valid both coordinates (255,255)",
			position: references.Position{X: references.XLargeMapTiles - 1, Y: references.YLargeMapTiles - 1},
			expected: "valid (both coordinates within range)",
		},
		{
			name:     "Out of bounds X (256,0)",
			position: references.Position{X: references.XLargeMapTiles, Y: 0},
			expected: "invalid (X coordinate out of range)",
		},
		{
			name:     "Out of bounds Y (0,256)",
			position: references.Position{X: 0, Y: references.YLargeMapTiles},
			expected: "invalid (Y coordinate out of range)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("Testing position (%d,%d) - expecting %s", tc.position.X, tc.position.Y, tc.expected)

			// This may trigger log.Fatal due to the inconsistency bug
			isOutOfBounds := gs.IsOutOfBounds(tc.position)

			if tc.position.X < references.XLargeMapTiles && tc.position.Y < references.YLargeMapTiles {
				// Should be valid
				if isOutOfBounds {
					t.Errorf("BUG: Position (%d,%d) should be valid but IsOutOfBounds returned true",
						tc.position.X, tc.position.Y)
					t.Errorf("This indicates inconsistent bounds checking logic")
				} else {
					t.Logf("✅ Position (%d,%d) correctly identified as valid", tc.position.X, tc.position.Y)
				}
			} else {
				// Should be invalid
				if !isOutOfBounds {
					t.Errorf("BUG: Position (%d,%d) should be invalid but IsOutOfBounds returned false",
						tc.position.X, tc.position.Y)
				} else {
					t.Logf("✅ Position (%d,%d) correctly identified as invalid", tc.position.X, tc.position.Y)
				}
			}
		})
	}

	t.Logf("\n=== BOUNDS CHECKING LOGIC ANALYSIS ===")
	t.Logf("Current buggy logic in IsOutOfBounds:")
	t.Logf("  X check: position.X > references.XLargeMapTiles-1")
	t.Logf("  Y check: position.Y >= references.YLargeMapTiles")
	t.Logf("")
	t.Logf("INCONSISTENCY: X uses '> -1' while Y uses '>='")
	t.Logf("CONSEQUENCE: Edge coordinate (255,255) has different validation behavior")
	t.Logf("")
	t.Logf("RECOMMENDED FIX:")
	t.Logf("  Use consistent logic: position.X >= XLargeMapTiles || position.Y >= YLargeMapTiles")
	t.Logf("  This makes (255,255) valid and (256,256) invalid for both axes")
}

// TestIsOutOfBounds_EdgeCaseValidation validates bounds checking edge cases
func TestIsOutOfBounds_EdgeCaseValidation_SmallMap(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle). // Small map
		WithSystemCallbacks().
		Build()
	// Small map bounds are determined by location reference
	maxX := gs.GetCurrentSmallLocationReference().GetMaxX()
	maxY := gs.GetCurrentSmallLocationReference().GetMaxY()

	t.Logf("=== SMALL MAP BOUNDS VALIDATION ===")
	t.Logf("Small map bounds: 0 to %d (X), 0 to %d (Y)", maxX, maxY)

	testCases := []struct {
		name          string
		position      references.Position
		shouldBeValid bool
	}{
		{
			name:          "Origin (0,0)",
			position:      references.Position{X: 0, Y: 0},
			shouldBeValid: true,
		},
		{
			name:          "Negative X (-1,0)",
			position:      references.Position{X: -1, Y: 0},
			shouldBeValid: false,
		},
		{
			name:          "Negative Y (0,-1)",
			position:      references.Position{X: 0, Y: -1},
			shouldBeValid: false,
		},
		{
			name:          "Max valid coordinates",
			position:      references.Position{X: maxX, Y: maxY},
			shouldBeValid: true,
		},
		{
			name:          "X out of bounds",
			position:      references.Position{X: maxX + 1, Y: 0},
			shouldBeValid: false,
		},
		{
			name:          "Y out of bounds",
			position:      references.Position{X: 0, Y: maxY + 1},
			shouldBeValid: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			isOutOfBounds := gs.IsOutOfBounds(tc.position)
			isValid := !isOutOfBounds

			if isValid != tc.shouldBeValid {
				t.Errorf("Position (%d,%d): expected valid=%t, got valid=%t (bounds: 0-%d, 0-%d)",
					tc.position.X, tc.position.Y, tc.shouldBeValid, isValid, maxX, maxY)
			} else {
				t.Logf("✅ Position (%d,%d): correctly identified as valid=%t",
					tc.position.X, tc.position.Y, isValid)
			}
		})
	}
}
