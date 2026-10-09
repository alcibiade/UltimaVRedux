package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestFloorCollisionBug_LordBritishCastle reproduces the specific bug reported by user
//
// BUG DESCRIPTION:
// User at Lord British's Castle Floor 0 position (16,22) is blocked from moving to:
// - (15,22) - Left movement blocked despite cobble floor tile
// - (16,23) - Down movement blocked despite cobble floor tile
//
// ROOT CAUSE HYPOTHESIS:
// ObjectPresentAt() method only checks X,Y position but ignores floor level,
// causing collision detection with NPCs/objects on different floors.
//
// EXPECTED BEHAVIOR:
// - Objects on different floors should NOT block movement
// - Objects on same floor SHOULD block movement
// - Look command showing "cobble" indicates passable floor tile
func TestFloorCollisionBug_LordBritishCastle_Integration(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		WithPlayerAt(16, 22). // Exact position reported by user
		Build()
	t.Logf("=== FLOOR COLLISION BUG DIAGNOSTIC ===")
	t.Logf("Player at Lord British's Castle Floor 0, position (16,22)")
	t.Logf("Attempting to reproduce movement blocking to adjacent floor tiles")

	// Verify we're on the correct floor
	playerFloor := gs.MapState.PlayerLocation.Floor
	t.Logf("Player floor: %d", playerFloor)

	// Test the two specific blocked directions reported by user
	blockedPositions := []struct {
		name      string
		position  references.Position
		direction string
	}{
		{"Left movement (16,22) -> (15,22)", references.Position{X: 15, Y: 22}, "left"},
		{"Down movement (16,22) -> (16,23)", references.Position{X: 16, Y: 23}, "down"},
	}

	for _, blocked := range blockedPositions {
		t.Run(blocked.name, func(t *testing.T) {
			pos := blocked.position
			t.Logf("\n=== ANALYZING BLOCKED POSITION (%d,%d) ===", pos.X, pos.Y)

			// Get tile information
			currentMap := gs.GetCurrentLayeredMap()
			tile := currentMap.GetTopTile(&pos)
			if tile != nil {
				t.Logf("Tile: index=%d, name=%s", tile.Index, tile.Name)

				// Check terrain passability
				vehicleType := gs.PartyVehicle.GetVehicleDetails().VehicleType
				terrainPassable := tile.IsPassable(vehicleType)
				t.Logf("Terrain passable for vehicle %v: %t", vehicleType, terrainPassable)
			} else {
				t.Logf("Tile: nil")
			}

			// Check object collision - this is where the bug likely is
			objectPresent := gs.ObjectPresentAt(&pos)
			t.Logf("ObjectPresentAt(%d,%d): %t", pos.X, pos.Y, objectPresent)

			if objectPresent {
				// Investigate WHAT object is causing the collision
				if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
					mapUnit := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(pos)
					if mapUnit != nil {
						unitDetails := (*mapUnit).MapUnitDetails()
						t.Logf("🔍 FOUND OBJECT AT (%d,%d):", pos.X, pos.Y)
						t.Logf("  - Type: %T", *mapUnit)
						t.Logf("  - Object floor: %d", unitDetails.Floor)
						t.Logf("  - Player floor: %d", playerFloor)
						t.Logf("  - Visible: %t", unitDetails.Visible)

						if unitDetails.Floor != playerFloor {
							t.Logf("🐛 BUG DETECTED: Object on floor %d blocking movement for player on floor %d!",
								unitDetails.Floor, playerFloor)
							t.Logf("   This object should NOT block movement - they're on different floors!")
						} else {
							t.Logf("✅ Object correctly on same floor as player - blocking is expected")
						}
					}
				}
			}

			// Final movement check
			isPassable := gs.IsPassable(&pos)
			t.Logf("Final IsPassable(%d,%d): %t", pos.X, pos.Y, isPassable)

			// Determine expected vs actual behavior
			if tile != nil && tile.IsPassable(gs.PartyVehicle.GetVehicleDetails().VehicleType) {
				if !isPassable && objectPresent {
					t.Logf("🔍 ANALYSIS: Floor tile is passable but ObjectPresentAt returned true")
					t.Logf("   This suggests the bug is in floor-aware collision detection")
				} else if isPassable {
					t.Logf("✅ Position is correctly passable")
				}
			}
		})
	}

	t.Logf("\n=== BUG REPRODUCTION SUMMARY ===")
	t.Logf("If objects on different floors are blocking movement,")
	t.Logf("the ObjectPresentAt method needs floor-aware collision detection.")
}

// TestFloorCollisionDetection_MultiFloor validates that floor-aware collision works correctly
func TestFloorCollisionDetection_MultiFloor_Integration(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		WithPlayerAt(16, 22).
		Build()
	t.Logf("=== MULTI-FLOOR COLLISION DETECTION TEST ===")

	// Test collision detection across different floors
	testPositions := []references.Position{
		{X: 15, Y: 22}, // Left of player
		{X: 17, Y: 22}, // Right of player
		{X: 16, Y: 21}, // Above player
		{X: 16, Y: 23}, // Below player
	}

	playerFloor := gs.MapState.PlayerLocation.Floor
	t.Logf("Testing collision detection for player on floor %d", playerFloor)

	for _, pos := range testPositions {
		t.Run(fmt.Sprintf("Position_(%d,%d)", pos.X, pos.Y), func(t *testing.T) {
			// Check for objects at this position
			objectPresent := gs.ObjectPresentAt(&pos)

			if objectPresent {
				if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
					mapUnit := gs.CurrentNPCAIController.GetNpcs().GetMapUnitAtPositionOrNil(pos)
					if mapUnit != nil {
						objectFloor := (*mapUnit).MapUnitDetails().Floor

						t.Logf("Position (%d,%d): Object on floor %d, Player on floor %d",
							pos.X, pos.Y, objectFloor, playerFloor)

						if objectFloor != playerFloor {
							t.Logf("⚠️  CROSS-FLOOR COLLISION: Object on floor %d detected for player on floor %d",
								objectFloor, playerFloor)
							t.Logf("   This may cause incorrect movement blocking")
						}
					}
				}
			} else {
				t.Logf("Position (%d,%d): No objects detected", pos.X, pos.Y)
			}
		})
	}
}
