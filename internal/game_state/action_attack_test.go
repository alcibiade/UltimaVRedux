package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestActionAttackSmallMap_MirrorBreaking tests the mirror breaking functionality
func TestActionAttackSmallMap_MirrorBreaking(t *testing.T) {
	// Test successful mirror breaking using the real mirror in Britain at (23,1)
	t.Run("AttackMirror_Success", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(23, 2). // Position south of the real mirror at (23,1)
			WithSystemCallbacks().
			Build()

		// Verify there's actually a mirror at the expected position
		mirrorPos := &references.Position{X: 23, Y: 1}
		currentMap := gs.GetLayeredMapByCurrentLocation()
		tile := currentMap.GetTopTile(mirrorPos)
		t.Logf("Expected Mirror (%d), got tile index %d at position (%d,%d)", indexes.Mirror, tile.Index, mirrorPos.X, mirrorPos.Y)

		// Verify there's actually a mirror there
		assert.Equal(t, indexes.Mirror, int(tile.Index), "Should have a mirror at the test position")

		// Attack the mirror (north from our position)
		beforeTime := gs.DateTime
		result := gs.ActionAttackSmallMap(references.Up)

		// Verify attack succeeds
		assert.True(t, result, "Attack should succeed when targeting a mirror")

		// Verify mirror is broken at the composite level
		brokenTile := currentMap.GetTopTile(mirrorPos)
		assert.Equal(t, indexes.MirrorBroken, int(brokenTile.Index), "Mirror should be broken after attack")

		// Verify the layered map structure is correct
		// The base MapLayer should still have the original Mirror tile
		baseTile := currentMap.GetTileByLayer(map_state.MapLayer, mirrorPos)
		assert.Equal(t, indexes.Mirror, int(baseTile.Index), "Base layer should still have original Mirror tile")

		// The MapOverrideLayer should have the broken mirror tile
		overrideTile := currentMap.GetTileByLayer(map_state.MapOverrideLayer, mirrorPos)
		assert.NotNil(t, overrideTile, "Override layer should have a tile at mirror position")
		assert.Equal(t, indexes.MirrorBroken, int(overrideTile.Index), "Override layer should have MirrorBroken tile")

		// Verify proper user feedback
		assert.Len(t, mockCallbacks.Messages, 1, "Should have exactly one message")
		assert.Equal(t, "Broken!", mockCallbacks.Messages[0], "Should show 'Broken!' message")

		// Verify sound effect
		assert.Len(t, mockCallbacks.SoundEffectsPlayed, 1, "Should play exactly one sound effect")
		assert.Equal(t, SoundGlassBreak, mockCallbacks.SoundEffectsPlayed[0], "Should play glass breaking sound")

		expectedTime := beforeTime
		expectedTime.Advance(DefaultSmallMapMinutesPerTurn)
		assert.Equal(t, expectedTime, gs.DateTime)
	})

	// Test attacking non-mirror objects shows "Nothing to attack!"
	t.Run("AttackNonMirror_ShowsNothing", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(10, 10). // Position with empty terrain to the north
			WithSystemCallbacks().
			Build()

		// Attack in any direction (should hit non-mirror terrain)
		result := gs.ActionAttackSmallMap(references.Up)

		// Verify attack returns false
		assert.False(t, result, "Attack should fail when targeting non-mirror objects")

		// Verify appropriate message
		assert.Len(t, mockCallbacks.Messages, 1, "Should have one message")
		assert.Equal(t, "Nothing to attack!", mockCallbacks.Messages[0], "Should show 'Nothing to attack!' for non-mirror")
		assert.Len(t, mockCallbacks.SoundEffectsPlayed, 0, "Should not play sound for non-mirror")
	})
}
