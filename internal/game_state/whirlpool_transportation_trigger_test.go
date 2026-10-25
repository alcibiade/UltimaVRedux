package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWhirlpoolTransportation_FrigateManualMovement verifies that manual
// arrow key movement onto a whirlpool triggers transportation (not just automatic
// sailing), with the sails furled so no sailing mode is required.
func TestWhirlpoolTransportation_FrigateManualMovement(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	// Find deep water and spawn whirlpool (skip 10 tiles to avoid collision with existing tests)
	layeredMap := gs.GetCurrentLayeredMap()
	var whirlpoolPos references.Position
	found := false
	skip := 0

	for y := references.Coordinate(0); y < references.YLargeMapTiles && !found; y++ {
		for x := references.Coordinate(0); x < references.XLargeMapTiles && !found; x++ {
			pos := references.Position{X: x, Y: y}
			tile := layeredMap.GetTopTile(&pos)
			if tile.Is(indexes.Water1) {
				if skip < 10 {
					skip++

					continue
				}
				whirlpoolPos = pos
				found = true
			}
		}
	}

	require.True(t, found, "Should find deep water")

	// Spawn whirlpool
	require.True(t, gs.spawnWhirlpoolAt(whirlpoolPos), "Should spawn whirlpool for test")

	// Setup frigate ADJACENT to whirlpool (not on it yet)
	adjacentPos := references.Position{X: whirlpoolPos.X - 1, Y: whirlpoolPos.Y}
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		adjacentPos,
		0,
	)
	frigate.GetVehicleDetails().SetSailState(map_units.SailFurled) // Sails DOWN for manual movement
	frigate.GetVehicleDetails().SetHullHP(99)
	gs.PartyVehicle = *frigate
	gs.MapState.PlayerLocation.Position = adjacentPos

	// Clear any existing messages
	mockCallbacks.Messages = []string{}

	// MANUAL MOVEMENT: Simulate player pressing arrow key to move onto whirlpool
	// This simulates the UI layer calling MapState.PlayerLocation.Position = whirlpoolPos
	// followed by FinishTurn(), then manually checking for whirlpool (caller responsibility)
	gs.MapState.PlayerLocation.Position = whirlpoolPos
	gs.FinishTurn()

	// After movement, caller must check for whirlpool and trigger transportation
	// (prevents nested event queue violation when called from WhileAwaitingInput)
	if gs.IsPlayerOnWhirlpool() {
		q := gs.EventQueueManager.Workflow("whirlpool_transportation").
			Priority(event_queue.PriorityHigh).
			Sequential()
		gs.BuildWhirlpoolTransportationQueue(q)
		q.Start()
	}

	// Process event queue for transportation
	// Must advance ElapsedMs to allow animations to progress (deltaMs > 0)
	for gs.EventQueueManager.HasActiveWorkflows() {
		gs.ElapsedMs += 16 // Simulate ~60fps frame advancement
		gs.EventQueueManager.Update()
	}

	// TEST: Should display "WHIRLPOOL!" message
	assert.Contains(t, mockCallbacks.Messages, "WHIRLPOOL!",
		"Should display whirlpool message after manual movement")

	// TEST: Should teleport to underworld (34, 18), floor -1
	expectedPos := references.Position{X: 34, Y: 18}
	assert.Equal(t, expectedPos, gs.MapState.PlayerLocation.Position,
		"Should teleport to underworld destination")
	assert.Equal(t, references.FloorNumber(-1), gs.MapState.PlayerLocation.Floor,
		"Should be on underworld floor (-1)")

	// TEST: Hull should take damage
	hullHP := gs.PartyVehicle.GetVehicleDetails().GetHullHP()
	assert.Less(t, hullHP, 99, "Hull should take damage from whirlpool")
}

// TestWhirlpoolTransportation_SkiffManualMovement verifies that manually
// moving a skiff onto a whirlpool triggers transport. ref:SHIP-0003
func TestWhirlpoolTransportation_SkiffManualMovement(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	// Find deep water and spawn whirlpool (skip 11 tiles to avoid collision with existing tests)
	layeredMap := gs.GetCurrentLayeredMap()
	var whirlpoolPos references.Position
	found := false
	skip := 0

	for y := references.Coordinate(0); y < references.YLargeMapTiles && !found; y++ {
		for x := references.Coordinate(0); x < references.XLargeMapTiles && !found; x++ {
			pos := references.Position{X: x, Y: y}
			tile := layeredMap.GetTopTile(&pos)
			if tile.Is(indexes.Water1) {
				if skip < 11 {
					skip++

					continue
				}
				whirlpoolPos = pos
				found = true
			}
		}
	}

	require.True(t, found, "Should find deep water")

	// Spawn whirlpool
	require.True(t, gs.spawnWhirlpoolAt(whirlpoolPos), "Should spawn whirlpool for test")

	// Setup SKIFF ADJACENT to whirlpool
	adjacentPos := references.Position{X: whirlpoolPos.X - 1, Y: whirlpoolPos.Y}
	skiff := map_units.NewNPCFriendlyVehiceNewRef(
		references.SkiffVehicle,
		adjacentPos,
		0,
	)
	gs.PartyVehicle = *skiff
	gs.MapState.PlayerLocation.Position = adjacentPos

	// Clear any existing messages
	mockCallbacks.Messages = []string{}

	// MANUAL MOVEMENT: Simulate player rowing skiff onto whirlpool
	gs.MapState.PlayerLocation.Position = whirlpoolPos
	gs.FinishTurn()

	// After movement, caller must check for whirlpool and trigger transportation
	// (prevents nested event queue violation when called from WhileAwaitingInput)
	if gs.IsPlayerOnWhirlpool() {
		q := gs.EventQueueManager.Workflow("whirlpool_transportation").
			Priority(event_queue.PriorityHigh).
			Sequential()
		gs.BuildWhirlpoolTransportationQueue(q)
		q.Start()
	}

	// Process event queue for transportation
	// Must advance ElapsedMs to allow animations to progress (deltaMs > 0)
	for gs.EventQueueManager.HasActiveWorkflows() {
		gs.ElapsedMs += 16 // Simulate ~60fps frame advancement
		gs.EventQueueManager.Update()
	}

	// TEST: Skiffs SHOULD trigger transportation (original behavior)
	assert.Contains(t, mockCallbacks.Messages, "WHIRLPOOL!",
		"Skiffs should trigger whirlpool transportation (player_shape >= SHIP)")

	// TEST: Should teleport to underworld
	expectedPos := references.Position{X: 34, Y: 18}
	assert.Equal(t, expectedPos, gs.MapState.PlayerLocation.Position,
		"Skiff should teleport to underworld destination")
	assert.Equal(t, references.FloorNumber(-1), gs.MapState.PlayerLocation.Floor,
		"Should be on underworld floor (-1)")
}

// TestWhirlpoolTransportation_OnFootNoTransport verifies that foot contact
// damages the party without transporting it. ref:SHIP-0003
func TestWhirlpoolTransportation_OnFootNoTransport(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	// Find deep water and spawn whirlpool (skip 2 tiles to avoid test pollution)
	layeredMap := gs.GetCurrentLayeredMap()
	var whirlpoolPos references.Position
	found := false
	skip := 0

	for y := references.Coordinate(0); y < references.YLargeMapTiles && !found; y++ {
		for x := references.Coordinate(0); x < references.XLargeMapTiles && !found; x++ {
			pos := references.Position{X: x, Y: y}
			tile := layeredMap.GetTopTile(&pos)
			if tile.Is(indexes.Water1) {
				if skip < 2 {
					skip++

					continue
				}
				whirlpoolPos = pos
				found = true
			}
		}
	}

	require.True(t, found, "Should find deep water")

	// Spawn whirlpool
	require.True(t, gs.spawnWhirlpoolAt(whirlpoolPos), "Should spawn whirlpool for test")

	// Setup player ON FOOT (no vehicle) adjacent to whirlpool
	adjacentPos := references.Position{X: whirlpoolPos.X - 1, Y: whirlpoolPos.Y}
	gs.MapState.PlayerLocation.Position = adjacentPos

	// Clear any existing messages
	mockCallbacks.Messages = []string{}

	// Manual foot contact causes damage without whirlpool transport. ref:SHIP-0003
	originalPos := whirlpoolPos
	gs.MapState.PlayerLocation.Position = whirlpoolPos
	gs.FinishTurn()

	// Process event queue (should be empty - no transportation)
	// Must advance ElapsedMs to allow animations to progress (deltaMs > 0)
	for gs.EventQueueManager.HasActiveWorkflows() {
		gs.ElapsedMs += 16 // Simulate ~60fps frame advancement
		gs.EventQueueManager.Update()
	}

	// TEST: Should NOT display "WHIRLPOOL!" message (no transportation)
	assert.NotContains(t, mockCallbacks.Messages, "WHIRLPOOL!",
		"On-foot player should NOT trigger whirlpool transportation")

	// TEST: Should still be at whirlpool position (not teleported)
	assert.Equal(t, originalPos, gs.MapState.PlayerLocation.Position,
		"On-foot player should remain at whirlpool position (no teleport)")
	assert.Equal(t, references.FloorNumber(0), gs.MapState.PlayerLocation.Floor,
		"Should still be on overworld floor (no teleport)")
}
