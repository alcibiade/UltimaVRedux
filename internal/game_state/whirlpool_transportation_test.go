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

// TestWhirlpoolTransportation_TeleportToUnderworld verifies that whirlpool
// transportation moves the ship to the correct underworld position
func TestWhirlpoolTransportation_TeleportToUnderworld(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
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
	spawned := false
	for i := 0; i < 100; i++ {
		gs.SetRandomSeed(12345 + uint64(i))
		if gs.TrySpawnWhirlpool(whirlpoolPos) {
			spawned = true
			break
		}
	}
	require.True(t, spawned, "Should spawn whirlpool for test")

	// Board frigate and move to whirlpool
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		whirlpoolPos,
		0,
	)
	frigate.GetVehicleDetails().SetHullHP(100) // Start with full hull (clamped to 99)
	gs.PartyVehicle = *frigate
	gs.MapState.PlayerLocation.Position = whirlpoolPos

	// Trigger whirlpool transportation
	q := gs.EventQueueManager.Workflow("whirlpool_transportation").
		Priority(event_queue.PriorityHigh).
		Sequential()
	gs.BuildWhirlpoolTransportationQueue(q)
	q.Start()

	// Wait for event queue to complete (synchronous for testing)
	// Must advance ElapsedMs to allow animations to progress (deltaMs > 0)
	for gs.EventQueueManager.HasActiveWorkflows() {
		gs.ElapsedMs += 16 // Simulate ~60fps frame advancement
		gs.EventQueueManager.Update()
	}

	// Expect Underworld destination (34,18), floor -1. ref:SHIP-0003
	expectedPos := references.Position{X: 34, Y: 18}
	assert.Equal(t, expectedPos, gs.MapState.PlayerLocation.Position,
		"Should teleport to underworld whirlpool destination")
	assert.Equal(t, references.FloorNumber(-1), gs.MapState.PlayerLocation.Floor,
		"Should be on underworld floor (-1)")

	// TEST: Should display "WHIRLPOOL!" message
	assert.Contains(t, mockCallbacks.Messages, "WHIRLPOOL!",
		"Should display whirlpool message")

	// TEST: Whirlpool uses the shared uniform 1-30 hull hit.
	hullHP := gs.PartyVehicle.GetVehicleDetails().GetHullHP()
	assert.Less(t, hullHP, 99, "Hull should take damage from whirlpool")
	assert.GreaterOrEqual(t, hullHP, 69, "Hull damage should be 1-30 HP")
	assert.Equal(t, 1, mockCallbacks.PlaybackRequestCount(SoundWhirlpoolVortex, SoundPlaybackLoop))
	assert.Equal(t, 1, mockCallbacks.StopRequestCount(SoundWhirlpoolVortex))
}

// TestWhirlpoolTransportation_HullDamage verifies that whirlpool causes
// the shared source hit's uniform 1-30 hull damage.
func TestWhirlpoolTransportation_HullDamage(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	// Spawn whirlpool
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

	require.True(t, found, "Should find deep water")

	spawned := false
	for i := 0; i < 100; i++ {
		gs.SetRandomSeed(12345 + uint64(i))
		if gs.TrySpawnWhirlpool(whirlpoolPos) {
			spawned = true
			break
		}
	}
	require.True(t, spawned, "Should spawn whirlpool")

	// Board frigate with known hull strength
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		whirlpoolPos,
		0,
	)
	initialHull := 99 // Max hull HP
	frigate.GetVehicleDetails().SetHullHP(initialHull)
	gs.PartyVehicle = *frigate
	gs.MapState.PlayerLocation.Position = whirlpoolPos

	// Set deterministic seed for damage calculation
	gs.SetRandomSeed(99999)

	// Trigger transportation
	q := gs.EventQueueManager.Workflow("whirlpool_transportation").
		Priority(event_queue.PriorityHigh).
		Sequential()
	gs.BuildWhirlpoolTransportationQueue(q)
	q.Start()

	// Process event queue
	// Must advance ElapsedMs to allow animations to progress (deltaMs > 0)
	for gs.EventQueueManager.HasActiveWorkflows() {
		gs.ElapsedMs += 16 // Simulate ~60fps frame advancement
		gs.EventQueueManager.Update()
	}

	// TEST: Hull damage should be in range 1-30
	finalHull := gs.PartyVehicle.GetVehicleDetails().GetHullHP()
	damage := initialHull - finalHull

	assert.GreaterOrEqual(t, damage, 1, "Hull damage should be at least 1 HP")
	assert.LessOrEqual(t, damage, 30, "Hull damage should be at most 30 HP")
}

// TestWhirlpoolTransportation_MessageSequence verifies the message flow
func TestWhirlpoolTransportation_MessageSequence(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	// Setup whirlpool and frigate
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

	require.True(t, found, "Should find deep water")

	spawned := false
	for i := 0; i < 100; i++ {
		gs.SetRandomSeed(12345 + uint64(i))
		if gs.TrySpawnWhirlpool(whirlpoolPos) {
			spawned = true
			break
		}
	}
	require.True(t, spawned, "Should spawn whirlpool")

	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		whirlpoolPos,
		0,
	)
	gs.PartyVehicle = *frigate
	gs.MapState.PlayerLocation.Position = whirlpoolPos

	// Clear any existing messages
	mockCallbacks.Messages = []string{}

	// Trigger transportation
	q := gs.EventQueueManager.Workflow("whirlpool_transportation").
		Priority(event_queue.PriorityHigh).
		Sequential()
	gs.BuildWhirlpoolTransportationQueue(q)
	q.Start()

	// Process event queue
	// Must advance ElapsedMs to allow animations to progress (deltaMs > 0)
	for gs.EventQueueManager.HasActiveWorkflows() {
		gs.ElapsedMs += 16 // Simulate ~60fps frame advancement
		gs.EventQueueManager.Update()
	}

	// Expect the whirlpool announcement during the transport workflow. ref:SHIP-0003
	foundWhirlpoolMsg := false
	for _, msg := range mockCallbacks.Messages {
		if msg == "WHIRLPOOL!" {
			foundWhirlpoolMsg = true
			break
		}
	}

	assert.True(t, foundWhirlpoolMsg, "Should display WHIRLPOOL! message")
}
