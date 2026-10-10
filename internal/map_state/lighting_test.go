package map_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

// mockGameDimensions provides a minimal GameDimensions implementation for testing
type mockGameDimensions struct {
	tilesX, tilesY int
	wrapped        bool
}

func (m mockGameDimensions) GetTilesVisibleOnScreen() (int, int) {
	return m.tilesX, m.tilesY
}

func (m mockGameDimensions) GetTopLeftExtent() references.Position {
	return references.Position{X: 0, Y: 0}
}

func (m mockGameDimensions) GetBottomRightExtent() references.Position {
	return references.Position{X: references.Coordinate(m.tilesX), Y: references.Coordinate(m.tilesY)}
}

func (m mockGameDimensions) GetBottomRightWithoutOverflow() references.Position {
	return references.Position{X: references.Coordinate(m.tilesX), Y: references.Coordinate(m.tilesY)}
}

func (m mockGameDimensions) IsWrappedMap() bool {
	return m.wrapped
}

// newTestLighting creates a Lighting instance with reasonable test defaults
func newTestLighting() Lighting {
	return NewLighting(mockGameDimensions{tilesX: 11, tilesY: 11, wrapped: false})
}

// TestLightDistanceConstants verifies the light distance constants are properly defined
// and follow the expected hierarchy from original Ultima V.
//
// Original magical-light fuel raises sight to at least 18; torch fuel raises it
// to at least 10 (ref:LIGHT-0001). Redux instead uses presentation-oriented
// tile distances, documented as a deliberate deviation in ref:LIGHT-0006.
func TestLightDistanceConstants(t *testing.T) {
	t.Run("ConstantsAreDefined", func(t *testing.T) {
		assert.Equal(t, 3, TorchTileDistance, "TorchTileDistance should be 3")
		assert.Equal(t, 5, InLorTileDistance, "InLorTileDistance should be 5")
		assert.Equal(t, MagicLightTileDistance, VasLorTileDistance,
			"Vas Lor should use the common magical-light radius")
	})

	t.Run("LightHierarchyIsCorrect", func(t *testing.T) {
		assert.Equal(t, VasLorTileDistance, InLorTileDistance,
			"prepared light spells differ by duration, not radius")

		// In Lor should be brighter than torch
		assert.Greater(t, InLorTileDistance, TorchTileDistance,
			"In Lor should have greater range than torch")

		// Torch should still provide meaningful light
		assert.Greater(t, TorchTileDistance, 0,
			"Torch should provide some light")
	})

	t.Run("QuestItemLightDistances", func(t *testing.T) {
		assert.Equal(t, 3, ShardLightDistance, "Shards should have 3-tile radius")
		assert.Equal(t, 4, ArtifactLightDistance, "Artifacts should have 4-tile radius")

		// Artifacts should be brighter than shards
		assert.Greater(t, ArtifactLightDistance, ShardLightDistance,
			"Artifacts should have greater light range than shards")
	})
}

// TestLightingStructMagicLight tests the magic light functionality
func TestLightingStructMagicLight(t *testing.T) {
	t.Run("MagicLightInitiallyOff", func(t *testing.T) {
		lighting := newTestLighting()
		assert.False(t, lighting.HasMagicLight(), "Magic light should be off initially")
		assert.False(t, lighting.HasAnyLight(), "HasAnyLight should be false initially")
	})

	t.Run("SetMagicLightOn", func(t *testing.T) {
		lighting := newTestLighting()
		lighting.SetMagicLight(true)

		assert.True(t, lighting.HasMagicLight(), "Magic light should be on after SetMagicLight(true)")
		assert.True(t, lighting.HasAnyLight(), "HasAnyLight should be true when magic light is on")
	})

	t.Run("SetMagicLightOff", func(t *testing.T) {
		lighting := newTestLighting()
		lighting.SetMagicLight(true)
		lighting.SetMagicLight(false)

		assert.False(t, lighting.HasMagicLight(), "Magic light should be off after SetMagicLight(false)")
	})

	t.Run("TorchAndMagicLightIndependent", func(t *testing.T) {
		lighting := newTestLighting()

		// Light torch
		lighting.LightTorch()
		assert.True(t, lighting.HasTorchLit(), "Torch should be lit")
		assert.False(t, lighting.HasMagicLight(), "Magic light should still be off")
		assert.True(t, lighting.HasAnyLight(), "HasAnyLight should be true with torch")

		// Add magic light
		lighting.SetMagicLight(true)
		assert.True(t, lighting.HasTorchLit(), "Torch should still be lit")
		assert.True(t, lighting.HasMagicLight(), "Magic light should be on")
		assert.True(t, lighting.HasAnyLight(), "HasAnyLight should be true with both")

		// Extinguish torch - magic light remains
		lighting.ExtinguishTorch()
		assert.False(t, lighting.HasTorchLit(), "Torch should be extinguished")
		assert.True(t, lighting.HasMagicLight(), "Magic light should still be on")
		assert.True(t, lighting.HasAnyLight(), "HasAnyLight should be true with just magic light")
	})
}

// TestTorchBurnout tests that torches burn out over time
func TestTorchBurnout(t *testing.T) {
	t.Run("TorchBurnsOutAfterDefaultTurns", func(t *testing.T) {
		lighting := newTestLighting()
		lighting.LightTorch()

		assert.True(t, lighting.HasTorchLit(), "Torch should be lit initially")

		// Advance turns until torch burns out
		for i := 0; i < DefaultTorchDurationMinutes; i++ {
			lighting.AdvanceTurn()
		}

		assert.False(t, lighting.HasTorchLit(), "Torch should be extinguished after default turns")
	})

	t.Run("TorchDoesNotGoNegative", func(t *testing.T) {
		lighting := newTestLighting()

		// Advance turn without lighting torch - should not panic or go negative
		lighting.AdvanceTurn()
		lighting.AdvanceTurn()

		assert.False(t, lighting.HasTorchLit(), "Torch should remain unlit")
	})
}
