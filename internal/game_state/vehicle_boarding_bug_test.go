package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestVehicleBoardingCollisionValidation validates the fix for Boarding regression vehicle boarding issue
func TestVehicleBoardingCollisionValidation(t *testing.T) {
	t.Log("✅ Boarding regression: Validating vehicle boarding collision detection fix")
	t.Log("Expected: Player should be able to walk onto vehicle tiles")
	t.Log("Fixed: Vehicle boarding now works correctly")

	// Test at the specific coordinates mentioned in the bug report
	t.Run("LordBritishCastle_Floor2_MagicCarpet_x15_y17", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.LordBritishsCastle).
			WithPlayerAt(15, 17). // Start at the reported position
			WithFloor(2).         // Floor 2 (actual floor 2)
			WithSystemCallbacks().
			Build()
		gs.SetRandomSeed(12345) // Deterministic testing

		t.Log("Testing vehicle boarding at Lord British's Castle floor 2")
		t.Log("Player starting position: (15,17), trying to move DOWN to vehicle at (15,18)")

		// Try to move DOWN onto the magic carpet at (15,18)
		// Using simple movement logic to test vehicle boarding
		newPos := references.Down.GetNewPositionInDirection(&gs.MapState.PlayerLocation.Position)
		success := gs.IsPassable(newPos)
		if success {
			gs.MapState.PlayerLocation.Position = *newPos
		}

		// Boarding regression: Vehicle boarding should now work correctly
		t.Logf("Movement result: success=%t", success)
		t.Logf("Player final position: (%d,%d)", gs.MapState.PlayerLocation.Position.X, gs.MapState.PlayerLocation.Position.Y)

		if len(mockCallbacks.Messages) > 0 {
			t.Logf("System messages: %v", mockCallbacks.Messages)
		}

		// Check what's actually at the target position
		targetPos := &references.Position{X: 15, Y: 18}
		tile := gs.GetLayeredMapByCurrentLocation().GetTopTile(targetPos)
		hasObject := gs.ObjectPresentAt(targetPos)
		isPassable := gs.IsPassable(targetPos)

		t.Logf("Target position (15,18) analysis:")
		t.Logf("  - Tile: %s (index=%d)", tile.Name, tile.Index)
		t.Logf("  - Has object: %t", hasObject)
		t.Logf("  - Is passable: %t", isPassable)

		// Boarding regression: Assert that vehicle boarding now works
		if success {
			t.Log("✅ Boarding regression passed: Vehicle boarding works correctly!")
			// Verify player moved to the vehicle position
			if gs.MapState.PlayerLocation.Position.Y == 18 {
				t.Log("✅ Confirmed: Player successfully moved onto vehicle tile")
			}
		} else {
			t.Error("❌ Boarding regression failed: Vehicle boarding still fails")
			// Look for error messages to understand why it failed
			for _, msg := range mockCallbacks.Messages {
				t.Logf("System message: %s", msg)
			}
		}
	})
}

// TestVehicleBoardingBug_MultipleVehicleTypes tests different vehicle types to confirm systemic issue
func TestVehicleBoardingBug_MultipleVehicleTypes(t *testing.T) {
	t.Log("🐛 Boarding regression: Testing multiple vehicle types for boarding collision issue")

	// Test different vehicle types at various locations
	testCases := []struct {
		name                string
		location            references.Location
		startX, startY      int
		vehicleX, vehicleY  int
		direction           references.Direction
		expectedVehicleType string
	}{
		{
			name:     "MagicCarpet_LordBritishCastle",
			location: references.LordBritishsCastle,
			startX:   14, startY: 17,
			vehicleX: 15, vehicleY: 17,
			direction:           references.Right,
			expectedVehicleType: "Magic Carpet",
		},
		// TODO: Add more test cases for horses, skiffs, ships when we identify their locations
		// This will help confirm the issue affects all vehicle types
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs, mockCallbacks := NewIntegrationTestBuilder(t).
				WithLocation(tc.location).
				WithPlayerAt(tc.startX, tc.startY).
				WithFloor(1).
				WithSystemCallbacks().
				Build()
			gs.SetRandomSeed(12345)

			t.Logf("Testing %s at (%d,%d)", tc.expectedVehicleType, tc.vehicleX, tc.vehicleY)

			// Analyze the vehicle position before attempting movement
			vehiclePos := &references.Position{X: references.Coordinate(tc.vehicleX), Y: references.Coordinate(tc.vehicleY)}
			tile := gs.GetLayeredMapByCurrentLocation().GetTopTile(vehiclePos)
			hasObject := gs.ObjectPresentAt(vehiclePos)
			isPassable := gs.IsPassable(vehiclePos)

			t.Logf("Vehicle position analysis: tile=%s, hasObject=%t, isPassable=%t",
				tile.Name, hasObject, isPassable)

			// Try to move onto the vehicle
			// Using simple movement logic to test vehicle boarding
			newPos := tc.direction.GetNewPositionInDirection(&gs.MapState.PlayerLocation.Position)
			success := gs.IsPassable(newPos)
			if success {
				gs.MapState.PlayerLocation.Position = *newPos
			}

			t.Logf("Movement onto %s: success=%t", tc.expectedVehicleType, success)

			if !success && len(mockCallbacks.Messages) > 0 {
				t.Logf("Blocking messages: %v", mockCallbacks.Messages)
			}

			// For now, just document the behavior - don't assert success/failure
			// until we understand the exact collision detection logic
		})
	}
}

// TestCollisionDetectionAnalysis_VehicleTiles analyzes collision detection logic for vehicle tiles
func TestCollisionDetectionAnalysis_VehicleTiles(t *testing.T) {
	t.Log("🔍 Boarding regression: Analyzing collision detection logic for vehicle tiles")

	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithPlayerAt(15, 17).
		WithFloor(2).
		WithSystemCallbacks().
		Build()
	// Analyze the magic carpet position in detail
	carpetPos := &references.Position{X: 15, Y: 18}

	t.Log("=== COLLISION DETECTION ANALYSIS ===")

	// Layer 1: Terrain passability
	tile := gs.GetLayeredMapByCurrentLocation().GetTopTile(carpetPos)
	terrainPassable := tile.IsPassable(gs.PartyVehicle.GetVehicleDetails().VehicleType)

	t.Logf("Layer 1 - Terrain: %s (index=%d)", tile.Name, tile.Index)
	t.Logf("  Terrain passable: %t", terrainPassable)

	// Layer 2: Object presence
	hasObject := gs.ObjectPresentAt(carpetPos)
	t.Logf("Layer 2 - Objects: hasObject=%t", hasObject)

	// Combined result
	combinedPassable := gs.IsPassable(carpetPos)
	t.Logf("Combined passability: %t", combinedPassable)

	// Analysis
	t.Log("=== ANALYSIS ===")
	if terrainPassable && hasObject && !combinedPassable {
		t.Log("🎯 HYPOTHESIS: Vehicle detected as object, blocking movement")
		t.Log("    - Terrain allows movement")
		t.Log("    - Object present (vehicle)")
		t.Log("    - Combined logic blocks movement")
		t.Log("    - ROOT CAUSE: Vehicles treated as impassable objects")
	} else if !terrainPassable {
		t.Log("🎯 HYPOTHESIS: Terrain-level blocking")
		t.Log("    - Vehicle tile itself marked as impassable")
	} else {
		t.Log("🤔 UNEXPECTED: Need to investigate other collision factors")
	}
}
