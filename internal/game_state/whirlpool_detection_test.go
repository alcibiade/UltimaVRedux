package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Frigates and skiffs on a whirlpool both qualify for its transport.
// ref:SHIP-0003
func TestWhirlpoolDetection_VesselOnWhirlpool(t *testing.T) {
	for _, vessel := range []references.VehicleType{references.FrigateVehicle, references.SkiffVehicle} {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(100, 100).
			Build()
		layeredMap := gs.GetCurrentLayeredMap()
		var whirlpoolPos references.Position
		found := false
		for y := references.Coordinate(0); y < references.YLargeMapTiles && !found; y++ {
			for x := references.Coordinate(0); x < references.XLargeMapTiles && !found; x++ {
				pos := references.Position{X: x, Y: y}
				if layeredMap.GetTopTile(&pos).Is(indexes.Water1) {
					whirlpoolPos, found = pos, true
				}
			}
		}
		require.True(t, found, "Should find deep water tile")
		require.True(t, gs.spawnWhirlpoolAt(whirlpoolPos), "Should spawn whirlpool for test")

		gs.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(vessel, whirlpoolPos, 0)
		gs.MapState.PlayerLocation.Position = whirlpoolPos

		assert.True(t, gs.IsPlayerOnWhirlpool(), "vehicle %v on the whirlpool tile", vessel)
	}
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
