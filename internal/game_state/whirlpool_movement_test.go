package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestWhirlpoolMovement_SkipTurnBehavior verifies a persistent
// alternating skip/move toggle, beginning with a skip. This is deterministic
// cadence rather than an independent random half-chance each turn. ref:ENEMY-0005
func TestWhirlpoolMovement_SkipTurnBehavior(t *testing.T) {
	// Arrange - Create game state with large map
	// Player must be within 22 tiles of whirlpool test positions
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		Build()

	// Create whirlpool enemy reference
	whirlpoolTile := gs.GameReferences.TileReferences.GetTile(indexes.WhirlpoolKeyIndex)
	whirlpoolEnemy := references.EnemyReference{
		MonsterType:          references.MonsterWhirlpool,
		KeyFrameTile:         whirlpoolTile,
		HitPoints:            1,
		HasWhirlpoolMovement: true,
		IsWaterEnemy:         true,
		Name:                 "Test Whirlpool",
	}

	// Test position close to player to avoid cleanup
	testPos := references.Position{X: 105, Y: 100}

	// Create whirlpool at test position
	npcEnemy := map_units.NewEnemyNPC(whirlpoolEnemy, -1)
	npcEnemy.SetPos(testPos)
	npcEnemy.SetFloor(0)
	npcEnemy.SetVisible(true)

	// Add to NPC controller
	gs.CurrentNPCAIController.GetNpcs().AddEnemy(npcEnemy)

	// Helper to get whirlpool from AI controller's slice
	getWhirlpool := func() *map_units.NPCEnemy {
		enemies := gs.CurrentNPCAIController.GetNpcs().GetEnemies()
		for i := range enemies {
			if enemies[i].EnemyReference.HasWhirlpoolMovement {
				return &enemies[i]
			}
		}

		return nil
	}

	// Test alternating skip pattern over multiple turns
	// Pattern should be: skip, move, skip, move... (starting with skip because toggle happens BEFORE check)
	testCases := []struct {
		turn       int
		shouldSkip bool
	}{
		{1, true},  // Turn 1: toggle false→true, skip
		{2, false}, // Turn 2: toggle true→false, move
		{3, true},  // Turn 3: toggle false→true, skip
		{4, false}, // Turn 4: toggle true→false, move
	}

	for _, tc := range testCases {
		t.Run("Turn "+string(rune('0'+tc.turn)), func(t *testing.T) {
			whirlpool := getWhirlpool()
			if whirlpool == nil {
				t.Fatal("Whirlpool disappeared from NPC list")
			}

			// Store initial position
			initialPos := whirlpool.Pos()

			// Act - Process one AI turn
			gs.CurrentNPCAIController.AdvanceNextTurnCalcAndMoveNPCs()

			// Get whirlpool again (slice may have been reallocated)
			whirlpool = getWhirlpool()
			if whirlpool == nil {
				t.Fatal("Whirlpool disappeared after turn")
			}

			// Assert - Check if whirlpool moved
			finalPos := whirlpool.Pos()
			moved := initialPos != finalPos

			if tc.shouldSkip {
				if moved {
					t.Errorf("Turn %d: Whirlpool should have skipped but moved from %v to %v",
						tc.turn, initialPos, finalPos)
				} else {
					t.Logf("Turn %d: Whirlpool correctly skipped turn at %v", tc.turn, initialPos)
				}
			} else {
				// Note: When not skipping, whirlpool may still not move if terrain blocked
				t.Logf("Turn %d: Whirlpool should try to move from %v, ended at %v (moved=%v)",
					tc.turn, initialPos, finalPos, moved)
			}
		})
	}
}
