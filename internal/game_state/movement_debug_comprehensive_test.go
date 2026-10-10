package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestComprehensiveMovementDebug provides exhaustive debugging for the movement issue
// This test attempts to reproduce the EXACT conditions that might cause blocking
func TestComprehensiveMovementDebug_LordBritishCastle(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		WithPlayerAt(16, 22).
		Build()
	t.Logf("=== COMPREHENSIVE MOVEMENT DEBUG ===")

	// First, let's examine the EXACT game state
	t.Logf("Player Details:")
	t.Logf("  Position: (%d,%d)", gs.MapState.PlayerLocation.Position.X, gs.MapState.PlayerLocation.Position.Y)
	t.Logf("  Floor: %d", gs.MapState.PlayerLocation.Floor)
	t.Logf("  Location: %v", gs.MapState.PlayerLocation.Location)

	vehicleType := gs.PartyVehicle.GetVehicleDetails().VehicleType
	t.Logf("  Vehicle: %v", vehicleType)

	// Check debug options that might affect movement
	t.Logf("Debug Options:")
	t.Logf("  FreeMove: %t", gs.DebugOptions.FreeMove)

	// Now test each blocked direction with exhaustive analysis
	testDirections := []struct {
		name      string
		direction references.Direction
		targetPos references.Position
	}{
		{"Left (16,22) -> (15,22)", references.Left, references.Position{X: 15, Y: 22}},
		{"Down (16,22) -> (16,23)", references.Down, references.Position{X: 16, Y: 23}},
		{"Right (16,22) -> (17,22)", references.Right, references.Position{X: 17, Y: 22}},
		{"Up (16,22) -> (16,21)", references.Up, references.Position{X: 16, Y: 21}},
	}

	for _, test := range testDirections {
		t.Run(test.name, func(t *testing.T) {
			t.Logf("\n=== ANALYZING %s ===", test.name)

			// Step 1: Direction calculation
			calculatedPos := test.direction.GetNewPositionInDirection(&gs.MapState.PlayerLocation.Position)
			if calculatedPos == nil {
				t.Logf("❌ GetNewPositionInDirection returned nil for %v", test.direction)
				return
			}
			t.Logf("Calculated position: (%d,%d)", calculatedPos.X, calculatedPos.Y)

			// Step 2: Bounds checking
			isOutOfBounds := gs.IsOutOfBounds(*calculatedPos)
			t.Logf("IsOutOfBounds: %t", isOutOfBounds)
			if isOutOfBounds {
				t.Logf("❌ BLOCKED: Out of bounds")
				return
			}

			// Step 3: Map and tile analysis
			currentMap := gs.GetCurrentLayeredMap()
			if currentMap == nil {
				t.Logf("❌ CurrentLayeredMap is nil")
				return
			}

			topTile := currentMap.GetTopTile(calculatedPos)
			if topTile == nil {
				t.Logf("❌ TopTile is nil at (%d,%d)", calculatedPos.X, calculatedPos.Y)
				return
			}

			t.Logf("Tile Analysis:")
			t.Logf("  Index: %d", topTile.Index)
			t.Logf("  Name: %s", topTile.Name)
			t.Logf("  IsPassable: %t", topTile.IsPassable(vehicleType))

			// Step 4: Check if it's a door tile
			isDoor := topTile.Is(indexes.RegularDoor) || topTile.Is(indexes.LockedDoor) ||
				topTile.Is(indexes.MagicLockDoor) || topTile.Is(indexes.RegularDoorView)
			t.Logf("  Is door: %t", isDoor)

			// Step 5: Object collision analysis
			objectPresent := gs.ObjectPresentAt(calculatedPos)
			t.Logf("Object Analysis:")
			t.Logf("  ObjectPresentAt: %t", objectPresent)

			if objectPresent && gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
				mapUnit := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(*calculatedPos)
				if mapUnit != nil {
					unitDetails := (*mapUnit).MapUnitDetails()
					t.Logf("  Object Details:")
					t.Logf("    Type: %T", *mapUnit)
					t.Logf("    Floor: %d (Player on floor: %d)", unitDetails.Floor, gs.MapState.PlayerLocation.Floor)
					t.Logf("    Visible: %t", unitDetails.Visible)
					t.Logf("    Position: (%d,%d)", unitDetails.Position.X, unitDetails.Position.Y)

					// Check for floor mismatch
					if unitDetails.Floor != gs.MapState.PlayerLocation.Floor {
						t.Logf("  🐛 POTENTIAL BUG: Object on different floor causing collision!")
					}
				} else {
					t.Logf("  ⚠️  ObjectPresentAt=true but GetMapUnitAtPositionOrNil=nil")
				}
			}

			// Step 6: Auto-open door check
			canAutoOpen := gs.CanAutoOpenDoor(calculatedPos)
			t.Logf("Auto-open door check: %t", canAutoOpen)

			// Step 7: Final passability test
			finalPassable := gs.IsPassable(calculatedPos)
			t.Logf("FINAL IsPassable: %t", finalPassable)

			// Step 8: Movement outcome simulation
			if finalPassable || gs.DebugOptions.FreeMove {
				t.Logf("✅ OUTCOME: Movement should SUCCEED")
			} else if canAutoOpen {
				t.Logf("🚪 OUTCOME: Auto-open door sequence should start")
			} else {
				t.Logf("❌ OUTCOME: 'Blocked!' message should appear")

				// Determine why it's blocked
				if !topTile.IsPassable(vehicleType) {
					t.Logf("  Reason: Terrain not passable")
				} else if objectPresent {
					t.Logf("  Reason: Object collision")
				} else {
					t.Logf("  Reason: Unknown - potential bug!")
				}
			}
		})
	}

	// Additional debugging: Check all nearby objects
	t.Logf("\n=== NEARBY OBJECTS ANALYSIS ===")
	for x := 14; x <= 18; x++ {
		for y := 20; y <= 24; y++ {
			pos := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
			if gs.ObjectPresentAt(&pos) {
				if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
					if mapUnit := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(pos); mapUnit != nil {
						unitDetails := (*mapUnit).MapUnitDetails()
						t.Logf("Object at (%d,%d): floor=%d, type=%T, visible=%t",
							x, y, unitDetails.Floor, *mapUnit, unitDetails.Visible)
					}
				}
			}
		}
	}
}
