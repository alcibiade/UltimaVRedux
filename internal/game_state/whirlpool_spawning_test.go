package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestWhirlpool_TrySpawnWhirlpool_DeepWater(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		Build()

	gs.SetRandomSeed(12345) // Deterministic testing

	// Find a deep water tile on the map
	layeredMap := gs.GetCurrentLayeredMap()
	var deepWaterPos references.Position
	found := false

	// Scan for deep water tile (sprite 1)
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

	require.True(t, found, "Should find at least one deep water tile")

	// Try to spawn whirlpool (with deterministic seed, should eventually spawn)
	spawned := false
	for i := 0; i < 100; i++ {
		if gs.TrySpawnWhirlpool(deepWaterPos) {
			spawned = true

			break
		}
		gs.SetRandomSeed(12345 + uint64(i)) // Try different seeds
	}

	if spawned {
		// Verify whirlpool exists at position
		whirlpool := gs.GetWhirlpoolAt(deepWaterPos)
		assert.NotNil(t, whirlpool, "Whirlpool should exist at spawned position")
		assert.True(t, whirlpool.IsVisible(), "Whirlpool should be visible")
		assert.Equal(t, deepWaterPos, whirlpool.Pos(), "Whirlpool position should match")
		assert.True(t, whirlpool.EnemyReference.KeyFrameTile.Is(indexes.WhirlpoolKeyIndex),
			"Whirlpool should have correct sprite")

		// Verify cannot spawn another whirlpool at same position
		secondSpawn := gs.TrySpawnWhirlpool(deepWaterPos)
		assert.False(t, secondSpawn, "Should not spawn duplicate whirlpool at same position")
	}
}

func TestWhirlpool_TrySpawnWhirlpool_NonDeepWater(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		Build()

	// Find a non-deep-water tile
	layeredMap := gs.GetCurrentLayeredMap()
	var landPos references.Position
	found := false

	for y := references.Coordinate(0); y < references.YLargeMapTiles; y++ {
		for x := references.Coordinate(0); x < references.XLargeMapTiles; x++ {
			pos := references.Position{X: x, Y: y}
			tile := layeredMap.GetTopTile(&pos)
			if !tile.Is(indexes.Water1) {
				landPos = pos
				found = true

				break
			}
		}
		if found {
			break
		}
	}

	require.True(t, found, "Should find at least one non-deep-water tile")

	// Try to spawn on non-deep-water (should always fail)
	gs.SetRandomSeed(77777) // Seed that would normally succeed
	spawned := gs.TrySpawnWhirlpool(landPos)
	assert.False(t, spawned, "Should not spawn whirlpool on non-deep-water tile")
}

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

	// Force spawn by using favorable seed
	spawned := false
	for i := 0; i < 100; i++ {
		gs.SetRandomSeed(12345 + uint64(i))
		if gs.TrySpawnWhirlpool(deepWaterPos) {
			spawned = true

			break
		}
	}

	if spawned {
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
}

func TestWhirlpool_SpawnProbability(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		Build()

	// Find deep water tiles
	layeredMap := gs.GetCurrentLayeredMap()
	deepWaterPositions := make([]references.Position, 0, 100)

	for y := references.Coordinate(0); y < references.YLargeMapTiles && len(deepWaterPositions) < 100; y++ {
		for x := references.Coordinate(0); x < references.XLargeMapTiles && len(deepWaterPositions) < 100; x++ {
			pos := references.Position{X: x, Y: y}
			tile := layeredMap.GetTopTile(&pos)
			if tile.Is(indexes.Water1) {
				deepWaterPositions = append(deepWaterPositions, pos)
			}
		}
	}

	require.GreaterOrEqual(t, len(deepWaterPositions), 100, "Should find at least 100 deep water tiles")

	// Test spawn probability over many trials using different positions
	// This avoids the issue of SetRandomSeed resetting to same state
	gs.SetRandomSeed(98765) // Single seed at start
	spawnCount := 0
	const trialsPerPosition = 8

	for _, pos := range deepWaterPositions {
		for trial := 0; trial < trialsPerPosition; trial++ {
			// Clear any existing whirlpool at this position
			gs.RemoveWhirlpoolAt(pos)

			if gs.TrySpawnWhirlpool(pos) {
				spawnCount++
			}
		}

		// IMPORTANT: Clean up after each position to prevent single-whirlpool constraint
		// from blocking subsequent positions. If the last trial spawned a whirlpool,
		// it would block all trials at the next position.
		gs.RemoveWhirlpoolAt(pos)
	}

	totalTrials := len(deepWaterPositions) * trialsPerPosition

	// Expected: 1 in 8 chance = ~12.5%
	// Allow reasonable variance (±30%)
	expectedSpawns := totalTrials / 8
	lowerBound := expectedSpawns - (expectedSpawns * 30 / 100)
	upperBound := expectedSpawns + (expectedSpawns * 30 / 100)

	assert.GreaterOrEqual(t, spawnCount, lowerBound,
		"Spawn rate should be approximately 1 in 8 (lower bound)")
	assert.LessOrEqual(t, spawnCount, upperBound,
		"Spawn rate should be approximately 1 in 8 (upper bound)")

	t.Logf("Whirlpool spawn rate: %d/%d (%.1f%%, expected ~12.5%%)",
		spawnCount, totalTrials, float64(spawnCount)/float64(totalTrials)*100)
}
