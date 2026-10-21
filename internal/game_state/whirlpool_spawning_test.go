package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestWhirlpool_RemoveWhirlpoolAt(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		Build()

	// Find deep water and spawn whirlpool
	layeredMap := gs.GetCurrentLayeredMap()
	var deepWaterPos references.Position
	found := false

	for y := references.Coordinate(0); y < references.YLargeMapTiles; y++ {
		for x := references.Coordinate(0); x < references.XLargeMapTiles; x++ {
			pos := references.Position{X: x, Y: y}
			tile := layeredMap.GetTopTile(&pos)
			if tile.Is(indexes.Water1) {
				deepWaterPos = pos
				found = true

				break
			}
		}
		if found {
			break
		}
	}

	require.True(t, found, "Should find deep water tile")

	require.True(t, gs.spawnWhirlpoolAt(deepWaterPos), "Should spawn whirlpool for test")

	// Verify whirlpool exists
	whirlpool := gs.GetWhirlpoolAt(deepWaterPos)
	require.NotNil(t, whirlpool, "Whirlpool should exist before removal")

	// Remove whirlpool
	removed := gs.RemoveWhirlpoolAt(deepWaterPos)
	assert.True(t, removed, "Should successfully remove whirlpool")

	// Verify whirlpool no longer exists
	whirlpool = gs.GetWhirlpoolAt(deepWaterPos)
	assert.Nil(t, whirlpool, "Whirlpool should not exist after removal")

	// Try to remove again (should fail)
	removedAgain := gs.RemoveWhirlpoolAt(deepWaterPos)
	assert.False(t, removedAgain, "Should not remove whirlpool that doesn't exist")
}
