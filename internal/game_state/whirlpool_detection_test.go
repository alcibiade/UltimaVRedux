package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWhirlpoolDetection_ShipOnWhirlpool verifies that the system can detect
// when a ship/frigate is on a whirlpool tile
func TestWhirlpoolDetection_ShipOnWhirlpool(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		Build()

	// Find a deep water position and spawn a whirlpool
	layeredMap := gs.GetCurrentLayeredMap()
	var whirlpoolPos references.Position
	found := false

	for y := references.Coordinate(0); y < references.YLargeMapTiles && !found; y++ {
		for x := references.Coordinate(0); x < references.XLargeMapTiles && !found; x++ {
			pos := references.Position{X: x, Y: y}
			tile := layeredMap.GetTopTile(&pos)
			if tile.Is(indexes.Water1) {
				whirlpoolPos = pos
				found = true
			}
		}
	}

	require.True(t, found, "Should find deep water tile")

	// Force spawn whirlpool
	require.True(t, gs.spawnWhirlpoolAt(whirlpoolPos), "Should spawn whirlpool for test")

	// Board a frigate and move to whirlpool position
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		whirlpoolPos,
		0,
	)
	gs.PartyVehicle = *frigate
	gs.MapState.PlayerLocation.Position = whirlpoolPos

	// TEST: Should detect ship on whirlpool
	isOnWhirlpool := gs.IsPlayerOnWhirlpool()
	assert.True(t, isOnWhirlpool, "Should detect frigate on whirlpool tile")
}

// TestWhirlpoolDetection_ShipNotOnWhirlpool verifies that detection correctly
// returns false when ship is not on a whirlpool
func TestWhirlpoolDetection_ShipNotOnWhirlpool(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		Build()

	// Find deep water without whirlpool
	layeredMap := gs.GetCurrentLayeredMap()
	var deepWaterPos references.Position
	found := false

	for y := references.Coordinate(0); y < references.YLargeMapTiles && !found; y++ {
		for x := references.Coordinate(0); x < references.XLargeMapTiles && !found; x++ {
			pos := references.Position{X: x, Y: y}
			tile := layeredMap.GetTopTile(&pos)
			if tile.Is(indexes.Water1) && gs.GetWhirlpoolAt(pos) == nil {
				deepWaterPos = pos
				found = true
			}
		}
	}

	require.True(t, found, "Should find deep water without whirlpool")

	// Board frigate on non-whirlpool water
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		deepWaterPos,
		0,
	)
	gs.PartyVehicle = *frigate
	gs.MapState.PlayerLocation.Position = deepWaterPos

	// TEST: Should NOT detect whirlpool
	isOnWhirlpool := gs.IsPlayerOnWhirlpool()
	assert.False(t, isOnWhirlpool, "Should not detect whirlpool when not present")
}

// TestWhirlpoolDetection_SkiffOnWhirlpool verifies that skiffs are NOT
// affected by whirlpools (original: only ships/frigates)
func TestWhirlpoolDetection_SkiffOnWhirlpool(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		Build()

	// Find and spawn whirlpool
	layeredMap := gs.GetCurrentLayeredMap()
	var whirlpoolPos references.Position
	found := false

	for y := references.Coordinate(0); y < references.YLargeMapTiles && !found; y++ {
		for x := references.Coordinate(0); x < references.XLargeMapTiles && !found; x++ {
			pos := references.Position{X: x, Y: y}
			tile := layeredMap.GetTopTile(&pos)
			if tile.Is(indexes.Water1) {
				whirlpoolPos = pos
				found = true
			}
		}
	}

	require.True(t, found, "Should find deep water tile")

	// Spawn whirlpool
	require.True(t, gs.spawnWhirlpoolAt(whirlpoolPos), "Should spawn whirlpool for test")

	// Board SKIFF (not frigate) and move to whirlpool
	skiff := map_units.NewNPCFriendlyVehiceNewRef(
		references.SkiffVehicle,
		whirlpoolPos,
		0,
	)
	gs.PartyVehicle = *skiff
	gs.MapState.PlayerLocation.Position = whirlpoolPos

	// Skiffs and frigates both qualify for whirlpool transport. ref:SHIP-0003
	isOnWhirlpool := gs.IsPlayerOnWhirlpool()
	assert.True(t, isOnWhirlpool, "Skiffs should be affected by whirlpools (original behavior)")
}

// TestWhirlpoolDetection_OnFootOnWhirlpool verifies that player on foot
// is NOT affected by whirlpools
func TestWhirlpoolDetection_OnFootOnWhirlpool(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		Build()

	// Find and spawn whirlpool on land tile (hypothetical)
	// In practice whirlpools only spawn on water, but test the logic
	layeredMap := gs.GetCurrentLayeredMap()
	var whirlpoolPos references.Position
	found := false

	for y := references.Coordinate(0); y < references.YLargeMapTiles && !found; y++ {
		for x := references.Coordinate(0); x < references.XLargeMapTiles && !found; x++ {
			pos := references.Position{X: x, Y: y}
			tile := layeredMap.GetTopTile(&pos)
			if tile.Is(indexes.Water1) {
				whirlpoolPos = pos
				found = true
			}
		}
	}

	require.True(t, found, "Should find water tile")

	// Spawn whirlpool
	require.True(t, gs.spawnWhirlpoolAt(whirlpoolPos), "Should spawn whirlpool for test")

	// Player on foot (no vehicle)
	gs.PartyVehicle = map_units.NewNPCFriendlyVehiceNoVehicle()
	gs.MapState.PlayerLocation.Position = whirlpoolPos

	// TEST: On foot should NOT trigger whirlpool
	isOnWhirlpool := gs.IsPlayerOnWhirlpool()
	assert.False(t, isOnWhirlpool, "Player on foot should not be affected by whirlpools")
}
