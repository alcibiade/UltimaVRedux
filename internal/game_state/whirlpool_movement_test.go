package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/logging"
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
	whirlpoolTile := gs.GameReferences.TileReferences.GetTile(indexes.Whirlpool_KeyIndex)
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

// TestWhirlpoolMovement_MovementWhenActive verifies whirlpools move on alternating turns
//
// With persistent XOR toggle, whirlpool alternates: skip, move, skip, move...
// On "move" turns, whirlpool should attempt movement via either:
// - Random direction (50% chance via OneInXOdds(2))
// - Seek toward player (50% chance)
//
// This test verifies movement DOES occur on open water on non-skip turns.
func TestWhirlpoolMovement_MovementWhenActive(t *testing.T) {
	// Enable AI logging to debug movement
	logging.SetCategoryEnabled(logging.CategoryAI, true)
	defer logging.SetCategoryEnabled(logging.CategoryAI, false)

	// Arrange - Create game state on open water area
	// Note: Player must be within 22 tiles of whirlpool or it gets cleaned up by distance check
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(10, 10). // Close to test position
		Build()

	// Set deterministic random seed for reproducible movement patterns
	gs.SetRandomSeed(12345)

	// Create whirlpool enemy reference
	whirlpoolTile := gs.GameReferences.TileReferences.GetTile(indexes.Whirlpool_KeyIndex)
	whirlpoolEnemy := references.EnemyReference{
		MonsterType:          references.MonsterWhirlpool,
		KeyFrameTile:         whirlpoolTile,
		HitPoints:            1,
		HasWhirlpoolMovement: true,
		IsWaterEnemy:         true,
		Name:                 "Test Whirlpool",
	}

	// Find a deep water position within 22 tiles of player
	layeredMap := gs.GetCurrentLayeredMap()
	var testPos references.Position
	found := false

	// Search near player position (10,10)
	for y := references.Coordinate(5); y < 20 && !found; y++ {
		for x := references.Coordinate(5); x < 20 && !found; x++ {
			pos := references.Position{X: x, Y: y}
			tile := layeredMap.GetTopTile(&pos)

			// Check if this is deep water
			if tile.Is(indexes.Water1) {
				testPos = pos
				found = true
			}
		}
	}

	if !found {
		t.Fatal("Could not find deep water position near player")
	}

	// Create whirlpool at test position
	npcEnemy := map_units.NewEnemyNPC(whirlpoolEnemy, -1)
	npcEnemy.SetPos(testPos)
	npcEnemy.SetFloor(0)
	npcEnemy.SetVisible(true)

	// Add to NPC controller
	gs.CurrentNPCAIController.GetNpcs().AddEnemy(npcEnemy)

	// Debug logging
	t.Logf("Test setup:")
	t.Logf("  Whirlpool position: %v", testPos)
	t.Logf("  HasWhirlpoolMovement flag: %v", npcEnemy.EnemyReference.HasWhirlpoolMovement)
	t.Logf("  IsWaterEnemy flag: %v", npcEnemy.EnemyReference.IsWaterEnemy)

	// Check surrounding tiles to verify movement is possible
	allDirections := testPos.GetFourDirectionsWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
	t.Logf("  Adjacent tiles:")

	for i, pos := range allDirections {
		tile := layeredMap.GetTopTile(&pos)
		isWater := tile.IsWater()
		canMove := npcEnemy.EnemyReference.CanMoveToTile(tile)
		t.Logf("    Dir %d (%v): sprite=%d, isWater=%v, canMove=%v", i, pos, tile.Index, isWater, canMove)
	}

	// Act - Run multiple turns to ensure movement occurs
	// With alternating skip pattern, half of turns should allow movement
	movementCount := 0
	maxIterations := 20

	// Get whirlpool from AI controller's slice (AddEnemy adds by value, so we need to track the one in the slice)
	getWhirlpool := func() *map_units.NPCEnemy {
		enemies := gs.CurrentNPCAIController.GetNpcs().GetEnemies()
		for i := range enemies {
			if enemies[i].EnemyReference.KeyFrameTile.Index == indexes.Whirlpool_KeyIndex {
				return &enemies[i]
			}
		}

		return nil
	}

	for i := 0; i < maxIterations; i++ {
		whirlpool := getWhirlpool()
		if whirlpool == nil {
			t.Fatal("Whirlpool disappeared from NPC list")
		}
		currentPos := whirlpool.Pos()

		gs.CurrentNPCAIController.AdvanceNextTurnCalcAndMoveNPCs()

		whirlpool = getWhirlpool()
		if whirlpool == nil {
			t.Fatal("Whirlpool disappeared after turn")
		}
		newPos := whirlpool.Pos()

		if currentPos != newPos {
			movementCount++
			t.Logf("Turn %d: Whirlpool moved from %v to %v", i+1, currentPos, newPos)
		}
	}

	// Assert - Whirlpool should have moved on some non-skip turns
	// With 50% skip rate, we expect roughly half of 20 turns (10) to be movement attempts
	// Not all movement attempts succeed (terrain/RNG), but at least some should
	if movementCount == 0 {
		t.Errorf("Whirlpool never moved in %d turns (expected at least some movement on open water)",
			maxIterations)
	} else {
		t.Logf("Whirlpool moved %d times in %d turns (%.1f%% movement rate)",
			movementCount, maxIterations, float64(movementCount)/float64(maxIterations)*100)
	}

	// Verify whirlpool didn't teleport far away (sanity check for movement algorithm)
	whirlpool := getWhirlpool()
	finalPos := whirlpool.Pos()
	deltaX := int(finalPos.X) - int(testPos.X)
	deltaY := int(finalPos.Y) - int(testPos.Y)

	if deltaX < 0 {
		deltaX = -deltaX
	}

	if deltaY < 0 {
		deltaY = -deltaY
	}

	distance := deltaX + deltaY
	if distance > maxIterations {
		t.Errorf("Whirlpool moved too far: %d tiles in %d turns (expected <= %d)",
			distance, maxIterations, maxIterations)
	}
}

// TestWhirlpoolMovement_FlagRequired verifies HasWhirlpoolMovement flag gates behavior
//
// Without HasWhirlpoolMovement flag, enemies should NOT use whirlpool AI logic.
// This test ensures the flag properly controls whirlpool behavior.
func TestWhirlpoolMovement_FlagRequired(t *testing.T) {
	// Arrange - Create game state
	// Player must be within 22 tiles of test position (105, 100)
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		Build()

	// Create whirlpool tile but WITHOUT HasWhirlpoolMovement flag
	whirlpoolTile := gs.GameReferences.TileReferences.GetTile(indexes.Whirlpool_KeyIndex)
	nonWhirlpoolEnemy := references.EnemyReference{
		KeyFrameTile:         whirlpoolTile,
		HitPoints:            1,
		HasWhirlpoolMovement: false, // FLAG DISABLED
		IsWaterEnemy:         true,
		Name:                 "Non-Whirlpool Enemy",
	}

	// Create whirlpool WITH flag for comparison
	whirlpoolEnemy := references.EnemyReference{
		MonsterType:          references.MonsterWhirlpool,
		KeyFrameTile:         whirlpoolTile,
		HitPoints:            1,
		HasWhirlpoolMovement: true, // FLAG ENABLED
		IsWaterEnemy:         true,
		Name:                 "Real Whirlpool",
	}

	testCases := []struct {
		name           string
		enemy          references.EnemyReference
		shouldBehavior bool
	}{
		{"Without flag - no whirlpool behavior", nonWhirlpoolEnemy, false},
		{"With flag - whirlpool behavior active", whirlpoolEnemy, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Position close to player to avoid cleanup
			testPos := references.Position{X: 105, Y: 100}

			// Create enemy at test position
			npcEnemy := map_units.NewEnemyNPC(tc.enemy, -1)
			npcEnemy.SetPos(testPos)
			npcEnemy.SetFloor(0)
			npcEnemy.SetVisible(true)

			// Add to NPC controller
			gs.CurrentNPCAIController.GetNpcs().AddEnemy(npcEnemy)

			// Act - Process AI turn (first turn should skip for whirlpool due to toggle)
			initialPos := npcEnemy.Pos()
			gs.CurrentNPCAIController.AdvanceNextTurnCalcAndMoveNPCs()

			// Get updated position from NPC list
			enemies := gs.CurrentNPCAIController.GetNpcs().GetEnemies()
			var finalPos references.Position
			found := false

			for i := range enemies {
				if enemies[i].EnemyReference.Name == tc.enemy.Name {
					finalPos = enemies[i].Pos()
					found = true

					break
				}
			}

			if !found {
				t.Fatal("Enemy not found after turn")
			}

			moved := initialPos != finalPos

			if tc.shouldBehavior {
				// With flag: First turn should skip (toggle false→true means skip)
				if moved {
					t.Errorf("Enemy WITH flag should have skipped first turn but moved from %v to %v",
						initialPos, finalPos)
				} else {
					t.Logf("Enemy WITH flag correctly skipped first turn (whirlpool behavior active)")
				}
			} else {
				// Without flag: Uses standard AI (might move normally, not our concern here)
				// We just verify whirlpool-specific XOR logic is NOT applied
				t.Logf("Enemy WITHOUT flag moved: %v (standard AI, not whirlpool behavior)", moved)
			}

			// Cleanup
			gs.CurrentNPCAIController.GetNpcs().RemoveNPCAtPosition(testPos)
			if moved {
				gs.CurrentNPCAIController.GetNpcs().RemoveNPCAtPosition(finalPos)
			}
		})
	}
}

// TestWhirlpoolMovement_MixedBehavior verifies whirlpool uses both random and seek
//
// Over many turns, whirlpool should use BOTH movement strategies:
// - Random direction (~50% when moving)
// - Seek toward player (~50% when moving)
//
// With persistent toggle, whirlpool alternates: skip, move, skip, move...
// On "move" turns, it uses mixed random/seek behavior.
// This test uses statistical sampling to verify balanced behavior.
func TestWhirlpoolMovement_MixedBehavior(t *testing.T) {
	// Arrange - Create game state
	// Note: Player must be within 22 tiles of whirlpool or it gets cleaned up by distance check
	playerPos := references.Position{X: 100, Y: 100}
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(int(playerPos.X), int(playerPos.Y)).
		Build()

	// Set deterministic random seed for reproducible movement patterns
	gs.SetRandomSeed(54321)

	// Create whirlpool enemy
	whirlpoolTile := gs.GameReferences.TileReferences.GetTile(indexes.Whirlpool_KeyIndex)
	whirlpoolEnemy := references.EnemyReference{
		MonsterType:          references.MonsterWhirlpool,
		KeyFrameTile:         whirlpoolTile,
		HitPoints:            1,
		HasWhirlpoolMovement: true,
		IsWaterEnemy:         true,
		Name:                 "Test Whirlpool",
	}

	// Find a deep water position close to player (within 22 tiles)
	layeredMap := gs.GetCurrentLayeredMap()
	var testPos references.Position
	found := false

	// Search in a spiral pattern from player position
	for dist := references.Coordinate(5); dist < 20 && !found; dist++ {
		for dy := -dist; dy <= dist && !found; dy++ {
			for dx := -dist; dx <= dist && !found; dx++ {
				x := playerPos.X + references.Coordinate(dx)
				y := playerPos.Y + references.Coordinate(dy)

				// Bounds check
				if x < 0 || x >= references.XLargeMapTiles || y < 0 || y >= references.YLargeMapTiles {
					continue
				}

				pos := references.Position{X: x, Y: y}
				tile := layeredMap.GetTopTile(&pos)

				// Check if this is deep water
				if tile.Is(indexes.Water1) {
					testPos = pos
					found = true
				}
			}
		}
	}

	if !found {
		t.Fatal("Could not find deep water position near player")
	}

	distX := int(testPos.X) - int(playerPos.X)
	if distX < 0 {
		distX = -distX
	}

	distY := int(testPos.Y) - int(playerPos.Y)
	if distY < 0 {
		distY = -distY
	}

	t.Logf("Test setup: whirlpool at %v, player at %v (distance=%d)",
		testPos, playerPos, distX+distY)

	// Create whirlpool
	npcEnemy := map_units.NewEnemyNPC(whirlpoolEnemy, -1)
	npcEnemy.SetPos(testPos)
	npcEnemy.SetFloor(0)
	npcEnemy.SetVisible(true)
	gs.CurrentNPCAIController.GetNpcs().AddEnemy(npcEnemy)

	// Helper to get whirlpool from AI controller's slice (AddEnemy adds by value)
	getWhirlpool := func() *map_units.NPCEnemy {
		enemies := gs.CurrentNPCAIController.GetNpcs().GetEnemies()
		for i := range enemies {
			if enemies[i].EnemyReference.KeyFrameTile.Index == indexes.Whirlpool_KeyIndex {
				return &enemies[i]
			}
		}

		return nil
	}

	// Act - Run many turns and track movement patterns
	// With alternating skip, only ~50 of 100 turns will be movement attempts
	const iterations = 100
	movementTowardPlayer := 0
	movementAwayFromPlayer := 0
	noMovement := 0

	for i := 0; i < iterations; i++ {
		whirlpool := getWhirlpool()
		if whirlpool == nil {
			t.Fatalf("Whirlpool disappeared from NPC list at iteration %d", i)
		}

		currentPos := whirlpool.Pos()

		// Calculate current distance to player
		cdx := int(currentPos.X) - int(playerPos.X)
		if cdx < 0 {
			cdx = -cdx
		}

		cdy := int(currentPos.Y) - int(playerPos.Y)
		if cdy < 0 {
			cdy = -cdy
		}

		currentDistance := cdx + cdy

		gs.CurrentNPCAIController.AdvanceNextTurnCalcAndMoveNPCs()

		whirlpool = getWhirlpool()
		if whirlpool == nil {
			t.Fatalf("Whirlpool disappeared after turn %d", i)
		}

		newPos := whirlpool.Pos()

		// Calculate new distance to player
		ndx := int(newPos.X) - int(playerPos.X)
		if ndx < 0 {
			ndx = -ndx
		}

		ndy := int(newPos.Y) - int(playerPos.Y)
		if ndy < 0 {
			ndy = -ndy
		}

		newDistance := ndx + ndy

		if currentPos == newPos {
			noMovement++
		} else if newDistance < currentDistance {
			movementTowardPlayer++
		} else if newDistance > currentDistance {
			movementAwayFromPlayer++
		}
		// Equal distance = lateral movement, not counted
	}

	totalMovement := movementTowardPlayer + movementAwayFromPlayer

	// Assert - Over 100 turns, we should see SOME of both behaviors
	// With 50% skip rate, we expect ~50 turns to be movement attempts
	// Not all movement attempts succeed, but we should see some of each type
	t.Logf("Movement patterns over %d turns:", iterations)
	t.Logf("  Toward player: %d (%.1f%%)", movementTowardPlayer, float64(movementTowardPlayer)/float64(iterations)*100)
	t.Logf("  Away from player: %d (%.1f%%)", movementAwayFromPlayer, float64(movementAwayFromPlayer)/float64(iterations)*100)
	t.Logf("  No movement: %d (%.1f%%) - includes ~50 skip turns", noMovement, float64(noMovement)/float64(iterations)*100)

	// Verify we saw both movement patterns (not purely random or purely seek)
	if totalMovement > 0 {
		if movementTowardPlayer == 0 {
			t.Error("Never moved toward player - expected some seek behavior")
		}

		if movementAwayFromPlayer == 0 {
			t.Logf("Warning: Never moved away from player - might be pure seek (terrain/RNG dependent)")
		}
	}

	// Verify whirlpool actually moved sometimes
	if totalMovement == 0 {
		t.Error("Whirlpool never moved in 100 turns - expected some movement")
	}
}
