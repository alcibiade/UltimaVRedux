package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestMovementUIIntegration_LordBritishCastle tests the full movement pipeline
// from UI input to game state validation, reproducing the exact user scenario.
//
// This test validates the entire movement flow:
// 1. UI layer direction input
// 2. GetNewPositionInDirection calculation
// 3. IsPassable validation
// 4. Movement execution or blocking
func TestMovementUIIntegration_LordBritishCastle(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		WithPlayerAt(16, 22). // Exact user-reported position
		Build()
	t.Logf("=== UI INTEGRATION MOVEMENT TEST ===")
	t.Logf("Starting position: (%d,%d) on floor %d",
		gs.MapState.PlayerLocation.Position.X,
		gs.MapState.PlayerLocation.Position.Y,
		gs.MapState.PlayerLocation.Floor)

	// Test the exact movements that are reported as blocked
	movements := []struct {
		name          string
		direction     references.Direction
		expectedPos   references.Position
		shouldSucceed bool
	}{
		{
			name:          "Move Left to (15,22)",
			direction:     references.Left,
			expectedPos:   references.Position{X: 15, Y: 22},
			shouldSucceed: true, // Based on our diagnostic test, this should work
		},
		{
			name:          "Move Down to (16,23)",
			direction:     references.Down,
			expectedPos:   references.Position{X: 16, Y: 23},
			shouldSucceed: true, // Based on our diagnostic test, this should work
		},
	}

	for _, movement := range movements {
		t.Run(movement.name, func(t *testing.T) {
			// Record initial position
			initialPos := gs.MapState.PlayerLocation.Position
			t.Logf("Initial position: (%d,%d)", initialPos.X, initialPos.Y)

			// Calculate new position using the same logic as UI layer
			newPosition := movement.direction.GetNewPositionInDirection(&initialPos)
			if newPosition == nil {
				t.Fatalf("GetNewPositionInDirection returned nil for direction %v", movement.direction)
			}

			t.Logf("Calculated new position: (%d,%d)", newPosition.X, newPosition.Y)

			// Verify the expected position calculation
			if newPosition.X != movement.expectedPos.X || newPosition.Y != movement.expectedPos.Y {
				t.Errorf("Position calculation mismatch: expected (%d,%d), got (%d,%d)",
					movement.expectedPos.X, movement.expectedPos.Y, newPosition.X, newPosition.Y)
			}

			// Test passability check - this is the critical validation
			isPassable := gs.IsPassable(newPosition)
			t.Logf("IsPassable(%d,%d): %t", newPosition.X, newPosition.Y, isPassable)

			// Check for bounds issues
			isOutOfBounds := gs.IsOutOfBounds(*newPosition)
			t.Logf("IsOutOfBounds(%d,%d): %t", newPosition.X, newPosition.Y, isOutOfBounds)

			// Detailed analysis of why it might be blocked
			if !isPassable {
				t.Logf("🔍 MOVEMENT BLOCKED - Analyzing reasons:")

				// Check terrain
				currentMap := gs.GetCurrentLayeredMap()
				tile := currentMap.GetTopTile(newPosition)
				if tile != nil {
					vehicleType := gs.PartyVehicle.GetVehicleDetails().VehicleType
					terrainPassable := tile.IsPassable(vehicleType)
					t.Logf("  - Terrain passable: %t (tile: %s, index: %d)", terrainPassable, tile.Name, tile.Index)
				}

				// Check objects
				objectPresent := gs.ObjectPresentAt(newPosition)
				t.Logf("  - Object present: %t", objectPresent)

				if objectPresent {
					if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
						mapUnit := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(*newPosition)
						if mapUnit != nil {
							unitDetails := (*mapUnit).MapUnitDetails()
							t.Logf("  - Object details: floor=%d, visible=%t, type=%T",
								unitDetails.Floor, unitDetails.Visible, *mapUnit)

							if unitDetails.Floor != gs.MapState.PlayerLocation.Floor {
								t.Logf("  🐛 BUG: Object on floor %d blocking player on floor %d!",
									unitDetails.Floor, gs.MapState.PlayerLocation.Floor)
							}
						}
					}
				}

				// Check bounds
				t.Logf("  - Out of bounds: %t", isOutOfBounds)
			}

			// Test outcome validation
			if movement.shouldSucceed && !isPassable {
				t.Errorf("❌ Movement should succeed but IsPassable returned false")
				t.Errorf("   This indicates a bug in the movement validation logic")
			} else if !movement.shouldSucceed && isPassable {
				t.Errorf("❌ Movement should be blocked but IsPassable returned true")
			} else {
				t.Logf("✅ Movement validation behaving as expected")
			}

			// Simulate the actual movement attempt
			mockCallbacks.Reset()

			// This simulates what the UI layer does - we can't easily test the UI layer directly,
			// but we can test the game logic that determines movement success/failure
			if isPassable {
				t.Logf("✅ Movement would succeed - player can move to (%d,%d)", newPosition.X, newPosition.Y)
			} else {
				t.Logf("❌ Movement would be blocked - 'Blocked!' message expected")
			}
		})
	}
}

// TestMovementStateConsistency checks for any state inconsistencies that might cause movement issues
func TestMovementStateConsistency_LordBritishCastle(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		WithPlayerAt(16, 22).
		Build()
	t.Logf("=== MOVEMENT STATE CONSISTENCY CHECK ===")

	// Check for any state inconsistencies
	t.Logf("Player location: %v", gs.MapState.PlayerLocation.Location)
	t.Logf("Player position: (%d,%d)", gs.MapState.PlayerLocation.Position.X, gs.MapState.PlayerLocation.Position.Y)
	t.Logf("Player floor: %d", gs.MapState.PlayerLocation.Floor)
	t.Logf("Vehicle type: %v", gs.PartyVehicle.GetVehicleDetails().VehicleType)

	// Check current map state
	currentMap := gs.GetCurrentLayeredMap()
	if currentMap == nil {
		t.Error("CurrentLayeredMap is nil - this could cause movement issues")
		return
	}

	playerTile := currentMap.GetTopTile(&gs.MapState.PlayerLocation.Position)
	if playerTile == nil {
		t.Error("Player tile is nil - this could cause movement issues")
		return
	}

	t.Logf("Player standing on tile: index=%d, name=%s", playerTile.Index, playerTile.Name)

	// Validate NPC controller state
	if gs.CurrentNPCAIController == nil {
		t.Error("CurrentNPCAIController is nil - this could affect object collision detection")
		return
	}

	if gs.CurrentNPCAIController.GetNpcs() == nil {
		t.Error("NPC map units is nil - this could affect object collision detection")
		return
	}

	// Check for any objects at or near the player position
	nearbyPositions := []references.Position{
		{X: 15, Y: 22}, // Left
		{X: 17, Y: 22}, // Right
		{X: 16, Y: 21}, // Up
		{X: 16, Y: 23}, // Down
		{X: 16, Y: 22}, // Player position
	}

	for _, pos := range nearbyPositions {
		objectPresent := gs.ObjectPresentAt(&pos)
		if objectPresent {
			if mapUnit := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(pos); mapUnit != nil {
				unitDetails := (*mapUnit).MapUnitDetails()
				t.Logf("Object at (%d,%d): floor=%d, visible=%t, type=%T",
					pos.X, pos.Y, unitDetails.Floor, unitDetails.Visible, *mapUnit)
			}
		}
	}

	t.Logf("✅ State consistency check complete")
}
