package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestMovementDiagnostic_Britain_15_30 diagnoses the specific movement issue
// where player cannot move from position (15,30) in Britain floor 0
// despite BrickFloor being walkable according to passability data
func TestMovementDiagnostic_Britain_15_30_Integration(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		WithPlayerAt(15, 30).
		Build()

	if gameState == nil {
		t.Skip("Skipping test - real game data not available")
		return
	}

	// Target position we're trying to diagnose
	problemPos := references.Position{X: 15, Y: 30}

	t.Logf("=== MOVEMENT DIAGNOSTIC FOR BRITAIN (15,30) ===")
	t.Logf("Current player location: %v", gameState.MapState.PlayerLocation.Location)
	t.Logf("Current floor: %d", gameState.MapState.PlayerLocation.Floor)
	playerPos := gameState.MapState.PlayerLocation.Position
	t.Logf("Player position: (%d,%d)", playerPos.X, playerPos.Y)

	// Get the tile at the problematic position
	currentMap := gameState.GetCurrentLayeredMap()
	if currentMap == nil {
		t.Fatal("Current layered map is nil")
	}

	problemTile := currentMap.GetTopTile(&problemPos)
	if problemTile == nil {
		t.Errorf("Tile at (%d,%d) is nil - this could be the problem", problemPos.X, problemPos.Y)
		return
	}

	t.Logf("\n=== TILE ANALYSIS ===")
	t.Logf("Tile at (%d,%d): index=%d", problemPos.X, problemPos.Y, problemTile.Index)
	t.Logf("Tile name: %s", problemTile.Name)

	// Check the current vehicle type
	vehicleType := gameState.PartyVehicle.GetVehicleDetails().VehicleType
	t.Logf("\n=== VEHICLE ANALYSIS ===")
	t.Logf("Current vehicle type: %v", vehicleType)

	// Test terrain passability directly
	terrainPassable := problemTile.IsPassable(vehicleType)
	t.Logf("\n=== TERRAIN PASSABILITY ===")
	t.Logf("Tile.IsPassable(%v) = %t", vehicleType, terrainPassable)

	// If it's walking, check the specific walking passability
	if vehicleType == references.NoPartyVehicle {
		walkingPassable := problemTile.IsWalkingPassable()
		t.Logf("Tile.IsWalkingPassable() = %t", walkingPassable)
	}

	// Check object collision
	objectPresent := gameState.ObjectPresentAt(&problemPos)
	t.Logf("\n=== OBJECT COLLISION ===")
	t.Logf("ObjectPresentAt(%d,%d) = %t", problemPos.X, problemPos.Y, objectPresent)

	if objectPresent && gameState.CurrentNPCAIController != nil && gameState.CurrentNPCAIController.GetNpcs() != nil {
		mapUnit := gameState.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(problemPos)
		if mapUnit != nil {
			t.Logf("Object at position: type=%T", *mapUnit)
		}
	}

	// Check bounds
	outOfBounds := gameState.IsOutOfBounds(problemPos)
	t.Logf("\n=== BOUNDS CHECK ===")
	t.Logf("IsOutOfBounds(%d,%d) = %t", problemPos.X, problemPos.Y, outOfBounds)

	// Final IsPassable test
	finalPassable := gameState.IsPassable(&problemPos)
	t.Logf("\n=== FINAL PASSABILITY ===")
	t.Logf("GameState.IsPassable(%d,%d) = %t", problemPos.X, problemPos.Y, finalPassable)

	// Determine the root cause
	t.Logf("\n=== ROOT CAUSE ANALYSIS ===")
	if !terrainPassable {
		t.Logf("❌ ROOT CAUSE: Terrain is not passable for vehicle type %v", vehicleType)
		t.Logf("   Tile index %d (%s) is not marked as passable in passability data", problemTile.Index, problemTile.Name)
	} else if objectPresent {
		t.Logf("❌ ROOT CAUSE: Object collision - there's an NPC/object blocking the position")
	} else if outOfBounds {
		t.Logf("❌ ROOT CAUSE: Position is out of map bounds")
	} else if finalPassable {
		t.Logf("✅ POSITION IS PASSABLE: The position should be walkable")
		t.Logf("   If movement is still blocked, check UI layer or action methods")
	} else {
		t.Logf("❌ ROOT CAUSE: Unknown - IsPassable returned false but all checks passed")
	}

	// Test adjacent positions for comparison
	t.Logf("\n=== ADJACENT POSITIONS COMPARISON ===")
	adjacent := []references.Position{
		{X: 14, Y: 30}, // Left
		{X: 16, Y: 30}, // Right
		{X: 15, Y: 29}, // Up
		{X: 15, Y: 31}, // Down
	}

	for _, pos := range adjacent {
		adjTile := currentMap.GetTopTile(&pos)
		adjPassable := gameState.IsPassable(&pos)
		if adjTile != nil {
			t.Logf("Adjacent (%d,%d): index=%d, passable=%t", pos.X, pos.Y, adjTile.Index, adjPassable)
		} else {
			t.Logf("Adjacent (%d,%d): tile=nil, passable=%t", pos.X, pos.Y, adjPassable)
		}
	}
}
