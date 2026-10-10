package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestFloorAwareCollision_Regression ensures the floor-aware collision fix works correctly
// and doesn't break existing collision detection for same-floor objects
func TestFloorAwareCollision_Regression_Integration(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		WithPlayerAt(16, 22).
		Build()
	t.Logf("=== FLOOR-AWARE COLLISION REGRESSION TEST ===")
	playerFloor := gs.MapState.PlayerLocation.Floor
	t.Logf("Player on floor: %d", playerFloor)

	// Test scenarios that should validate floor-aware collision detection
	testScenarios := []struct {
		name            string
		position        references.Position
		expectCollision bool
		reason          string
	}{
		{
			name:            "Empty floor tile (15,22)",
			position:        references.Position{X: 15, Y: 22},
			expectCollision: false,
			reason:          "No objects on same floor",
		},
		{
			name:            "Empty floor tile (16,23)",
			position:        references.Position{X: 16, Y: 23},
			expectCollision: false,
			reason:          "No objects on same floor",
		},
		{
			name:            "Same-floor object (15,24)",
			position:        references.Position{X: 15, Y: 24},
			expectCollision: true,
			reason:          "NPC on same floor should block movement",
		},
	}

	for _, scenario := range testScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("Testing position (%d,%d)", scenario.position.X, scenario.position.Y)

			// Check object collision with floor awareness
			objectPresent := gs.ObjectPresentAt(&scenario.position)
			t.Logf("ObjectPresentAt: %t", objectPresent)

			// Investigate what's actually there
			if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
				// Check using old method (position only)
				oldMethodUnit := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(scenario.position)

				// Check using new method (position + floor)
				newMethodUnit := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionAndFloorOrNil(scenario.position, playerFloor)

				if oldMethodUnit != nil {
					unitDetails := (*oldMethodUnit).MapUnitDetails()
					t.Logf("Old method found object: floor=%d, same_floor=%t",
						unitDetails.Floor, unitDetails.Floor == playerFloor)
				} else {
					t.Logf("Old method: no object found")
				}

				if newMethodUnit != nil {
					unitDetails := (*newMethodUnit).MapUnitDetails()
					t.Logf("New method found object: floor=%d", unitDetails.Floor)
				} else {
					t.Logf("New method: no object found")
				}
			}

			// Validate expected behavior
			if scenario.expectCollision && !objectPresent {
				t.Errorf("❌ Expected collision but ObjectPresentAt returned false")
				t.Errorf("   Reason: %s", scenario.reason)
			} else if !scenario.expectCollision && objectPresent {
				t.Errorf("❌ Expected no collision but ObjectPresentAt returned true")
				t.Errorf("   Reason: %s", scenario.reason)
			} else {
				t.Logf("✅ Collision detection behaving correctly: %s", scenario.reason)
			}

			// Test final movement validation
			isPassable := gs.IsPassable(&scenario.position)
			t.Logf("IsPassable: %t", isPassable)

			// For floor tiles without same-floor objects, should be passable
			if !scenario.expectCollision {
				tile := gs.GetCurrentLayeredMap().GetTopTile(&scenario.position)
				if tile != nil && tile.IsPassable(gs.PartyVehicle.GetVehicleDetails().VehicleType) {
					if !isPassable {
						t.Errorf("❌ Floor tile should be passable but IsPassable returned false")
					} else {
						t.Logf("✅ Floor tile correctly passable")
					}
				}
			}
		})
	}
}

// TestFloorAwareCollision_MultiFloorBuilding tests collision across multiple floors
func TestFloorAwareCollision_MultiFloorBuilding_Integration(t *testing.T) {
	// Test in a building with multiple floors
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		WithPlayerAt(16, 22).
		Build()
	t.Logf("=== MULTI-FLOOR COLLISION TEST ===")

	// Find all objects in the area and categorize by floor
	objectsByFloor := make(map[references.FloorNumber][]references.Position)

	for x := 14; x <= 18; x++ {
		for y := 20; y <= 24; y++ {
			pos := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}

			if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
				if mapUnit := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(pos); mapUnit != nil {
					floor := (*mapUnit).MapUnitDetails().Floor
					objectsByFloor[floor] = append(objectsByFloor[floor], pos)
				}
			}
		}
	}

	t.Logf("Objects found by floor:")
	for floor, positions := range objectsByFloor {
		t.Logf("  Floor %d: %d objects at positions %v", floor, len(positions), positions)
	}

	// Test collision detection for each floor
	for floor, positions := range objectsByFloor {
		for _, pos := range positions {
			// Test with player on same floor
			originalFloor := gs.MapState.PlayerLocation.Floor
			gs.MapState.PlayerLocation.Floor = floor

			objectPresentSameFloor := gs.ObjectPresentAt(&pos)

			// Test with player on different floor
			if floor != 0 {
				gs.MapState.PlayerLocation.Floor = 0
				objectPresentDifferentFloor := gs.ObjectPresentAt(&pos)

				t.Logf("Position (%d,%d) object on floor %d:", pos.X, pos.Y, floor)
				t.Logf("  Same floor collision: %t", objectPresentSameFloor)
				t.Logf("  Different floor collision: %t", objectPresentDifferentFloor)

				if objectPresentDifferentFloor {
					t.Errorf("❌ Object on floor %d should not block player on floor 0", floor)
				} else {
					t.Logf("✅ Cross-floor collision correctly prevented")
				}
			}

			// Restore original floor
			gs.MapState.PlayerLocation.Floor = originalFloor
		}
	}
}

// TestFloorAwareCollision_EdgeCases tests edge cases for floor-aware collision
func TestFloorAwareCollision_EdgeCases(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		WithPlayerAt(16, 22).
		Build()
	t.Logf("=== FLOOR-AWARE COLLISION EDGE CASES ===")

	t.Run("NilPosition", func(t *testing.T) {
		// Test nil position handling
		result := gs.ObjectPresentAt(nil)
		if result {
			t.Errorf("ObjectPresentAt(nil) should return false")
		} else {
			t.Logf("✅ Nil position handled correctly")
		}
	})

	t.Run("NilNPCController", func(t *testing.T) {
		// Test with nil NPC controller (temporarily)
		originalController := gs.CurrentNPCAIController
		gs.CurrentNPCAIController = nil

		defer func() {
			if r := recover(); r != nil {
				t.Errorf("ObjectPresentAt panicked with nil NPC controller: %v", r)
			}
			gs.CurrentNPCAIController = originalController
		}()

		pos := references.Position{X: 15, Y: 22}
		gs.ObjectPresentAt(&pos)
		t.Logf("✅ Nil NPC controller handled without panic")
	})

	t.Run("ExtremeFloorNumbers", func(t *testing.T) {
		// Test with extreme floor numbers
		originalFloor := gs.MapState.PlayerLocation.Floor
		defer func() { gs.MapState.PlayerLocation.Floor = originalFloor }()

		extremeFloors := []references.FloorNumber{-10, -1, 0, 1, 10}
		pos := references.Position{X: 15, Y: 22}

		for _, floor := range extremeFloors {
			gs.MapState.PlayerLocation.Floor = floor
			result := gs.ObjectPresentAt(&pos)
			t.Logf("Floor %d collision check: %t", floor, result)
		}

		t.Logf("✅ Extreme floor numbers handled correctly")
	})
}
