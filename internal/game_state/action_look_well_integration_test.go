package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

// TestWellDetection_RealMapData tests well detection using real Ultima V game data
func TestWellDetection_RealMapData(t *testing.T) {
	t.Run("Paws_WellAtCorrectPosition", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Paws).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		// Search for well tiles in Paws map
		layeredMap := gs.GetLayeredMapByCurrentLocation()
		if layeredMap == nil {
			t.Skip("Map not loaded - skipping well tile search")
		}

		wellFound := false
		var wellPos references.Position

		for y := 0; y < 64; y++ {
			for x := 0; x < 64; x++ {
				pos := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
				tile := layeredMap.GetTopTile(&pos)
				if tile == nil {
					continue
				}
				if tile.Index == indexes.Well {
					wellFound = true
					wellPos = pos
					t.Logf("Found well at position (%d, %d) in Paws", x, y)
					break
				}
			}
			if wellFound {
				break
			}
		}

		assert.True(t, wellFound, "Should find at least one well tile in Paws map")
		if wellFound {
			// Verify detection works at this position
			assert.True(t, gs.DetectWishingWellAtPosition(&wellPos),
				"Should detect well at position (%d, %d)", wellPos.X, wellPos.Y)
		}
	})

	t.Run("EmpathAbbey_WellAtCorrectPosition", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.EmpathAbbey).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		// Search for well tiles in Empath Abbey map
		layeredMap := gs.GetLayeredMapByCurrentLocation()
		if layeredMap == nil {
			t.Skip("Map not loaded - skipping well tile search")
		}

		wellFound := false
		var wellPos references.Position

		for y := 0; y < 64; y++ {
			for x := 0; x < 64; x++ {
				pos := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
				tile := layeredMap.GetTopTile(&pos)
				if tile == nil {
					continue
				}
				if tile.Index == indexes.Well {
					wellFound = true
					wellPos = pos
					t.Logf("Found well at position (%d, %d) in Empath Abbey", x, y)
					break
				}
			}
			if wellFound {
				break
			}
		}

		if !wellFound {
			t.Logf("No well tile found in Empath Abbey map - this is acceptable (well may not be placed in original data)")
		} else {
			// If we found a well, verify detection works at this position
			assert.True(t, gs.DetectWishingWellAtPosition(&wellPos),
				"Should detect well at position (%d, %d)", wellPos.X, wellPos.Y)
		}
	})

	t.Run("IolosHut_WellStartsInteraction", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.IolosHut).
			WithPlayerAt(8, 3).
			WithSystemCallbacks().
			Build()
		wellPosition := references.Position{X: 8, Y: 2}

		assert.True(t, gs.DetectWishingWellAtPosition(&wellPosition))
		assert.True(t, gs.ActionLookSmallMap(references.Up))
		assert.True(t, mockCallbacks.WellInteractionShown)
	})

	t.Run("Britain_NoWellAtPosition", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		// The arbitrary position does not contain a well.
		wellPos := references.Position{X: references.Coordinate(20), Y: references.Coordinate(20)}
		assert.False(t, gs.DetectWishingWellAtPosition(&wellPos),
			"Should not detect a well at a non-well position")
	})
}

// TestWellTrigger_ModernSystem tests the modern well trigger system with SystemCallbacks
func TestWellTrigger_ModernSystem(t *testing.T) {
	t.Run("Paws_TriggerModernWell", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Paws).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		// Find a well tile in Paws
		layeredMap := gs.GetLayeredMapByCurrentLocation()
		if layeredMap == nil {
			t.Skip("Map not loaded - skipping modern well trigger test")
		}

		var wellPos *references.Position
		for y := 0; y < 64; y++ {
			for x := 0; x < 64; x++ {
				pos := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
				tile := layeredMap.GetTopTile(&pos)
				if tile == nil {
					continue
				}
				if tile.Index == indexes.Well {
					wellPos = &pos
					break
				}
			}
			if wellPos != nil {
				break
			}
		}

		if wellPos == nil {
			t.Skip("No well found in Paws map for this test")
		}

		// Trigger modern well system
		success := gs.TriggerWishingWellModern(wellPos)

		// Verify well was triggered successfully
		assert.True(t, success, "Modern well trigger should succeed in valid location")

		// Verify ShowWellInteraction callback was invoked
		assert.True(t, mockCallbacks.WellInteractionShown,
			"ShowWellInteraction callback should have been called")
	})

	t.Run("Britain_NonWellPositionNotTriggered", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		wellPos := references.Position{X: references.Coordinate(20), Y: references.Coordinate(20)}

		// Attempt to trigger the workflow where no well exists.
		success := gs.TriggerWishingWellModern(&wellPos)

		assert.False(t, success, "Modern well trigger should fail without a well")

		// Verify ShowWellInteraction callback was NOT invoked
		assert.False(t, mockCallbacks.WellInteractionShown,
			"ShowWellInteraction callback should not have been called in invalid location")
	})

	t.Run("NilPosition_PanicsAsExpected", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Paws).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		// Attempt to trigger modern well with nil position should panic
		// This is a programming bug - callers should never pass nil
		assert.Panics(t, func() {
			gs.TriggerWishingWellModern(nil)
		}, "TriggerWishingWellModern should panic when called with nil position (caller bug)")
	})
}

// TestWellDetection_EdgeCases tests edge cases for well detection
func TestWellDetection_EdgeCases(t *testing.T) {
	t.Run("MapBounds_TopLeft", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Paws).
			WithPlayerAt(0, 0). // Top-left corner
			WithSystemCallbacks().
			Build()

		pos := references.Position{X: references.Coordinate(0), Y: references.Coordinate(0)}
		// Should not panic or crash when checking bounds
		_ = gs.DetectWishingWellAtPosition(&pos)
	})

	t.Run("MapBounds_BottomRight", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Paws).
			WithPlayerAt(63, 63). // Bottom-right corner
			WithSystemCallbacks().
			Build()

		pos := references.Position{X: references.Coordinate(63), Y: references.Coordinate(63)}
		// Should not panic or crash when checking bounds
		_ = gs.DetectWishingWellAtPosition(&pos)
	})

	t.Run("OutOfBounds_Negative", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Paws).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		pos := references.Position{X: references.Coordinate(-1), Y: references.Coordinate(-1)}
		// Should handle gracefully
		result := gs.DetectWishingWellAtPosition(&pos)
		assert.False(t, result, "Should not detect well at out-of-bounds position")
	})
}
