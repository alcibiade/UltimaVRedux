// Integration tests for Attack command workflows with real game data.
// These tests cover empty targets, mirror breaking, and callback integration.
package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestAttackCommand_SmallMap_Integration tests attack command in small maps (towns, buildings)
func TestAttackCommand_SmallMap_Integration(t *testing.T) {
	// Test attack command in Lord British's Castle (small map with potential NPCs)
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithPlayerAt(12, 12).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	t.Run("BasicAttackBehavior", func(t *testing.T) {
		mockCallbacks.Reset()

		// This fixture has no eligible target to the north.
		result := gs.ActionAttackSmallMap(references.Up)

		assert.False(t, result, "Attack should fail when the target tile is empty")

		// Validate SystemCallbacks integration
		assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have one message")
		assert.Equal(t, "Nothing to attack!", mockCallbacks.Messages[0], "Should show 'Nothing to attack!' message for small map")

		assert.Equal(t, 0, len(mockCallbacks.TimeAdvanced()), "An empty target should not advance time")
	})

	t.Run("AttackAllDirections", func(t *testing.T) {
		mockCallbacks.Reset()

		// Attack in all directions to test consistency
		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}

		for _, direction := range directions {
			result := gs.ActionAttackSmallMap(direction)
			assert.False(t, result, "Attack should return false for direction %v", direction)
		}

		// Validate all attacks were processed consistently
		assert.Equal(t, 4, len(mockCallbacks.Messages), "Should have message for each attack")

		// All messages should be consistent
		for i, message := range mockCallbacks.Messages {
			assert.Equal(t, "Nothing to attack!", message, "Message %d should be consistent", i)
		}
	})

	t.Run("AttackAtDifferentPositions", func(t *testing.T) {
		// Test attack behavior at various positions in the castle
		testPositions := []references.Position{
			{X: 5, Y: 5}, {X: 10, Y: 10}, {X: 15, Y: 15}, {X: 8, Y: 12},
		}

		originalPosition := gs.MapState.PlayerLocation.Position

		for _, pos := range testPositions {
			gs.MapState.PlayerLocation.Position = pos
			mockCallbacks.Reset()

			result := gs.ActionAttackSmallMap(references.Up)

			// Current behavior should be consistent regardless of position
			assert.False(t, result, "Attack should return false at position (%d,%d)", pos.X, pos.Y)
			assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have message at position (%d,%d)", pos.X, pos.Y)
			assert.Equal(t, "Nothing to attack!", mockCallbacks.Messages[0], "Should have consistent message at position (%d,%d)", pos.X, pos.Y)
		}

		// Restore original position
		gs.MapState.PlayerLocation.Position = originalPosition
	})
}

// TestAttackCommand_LargeMap_Integration tests attack command on large maps (overworld)
func TestAttackCommand_LargeMap_Integration(t *testing.T) {
	// Test attack command on large map (Britannia Underworld)
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	mockCallbacks.Reset()

	// Test attack on large map
	result := gs.ActionAttackLargeMap(references.Up)

	assert.False(t, result, "Large map attack should fail when the target tile is empty")

	// Validate SystemCallbacks integration
	assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have one message")
	assert.Equal(t, "Nothing to attack!", mockCallbacks.Messages[0])

	// Should not advance time
	assert.Equal(t, 0, len(mockCallbacks.TimeAdvanced()), "Large map attack should not advance time")

	t.Run("LargeMapAllDirections", func(t *testing.T) {
		mockCallbacks.Reset()

		// Test all directions on large map
		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}

		for _, direction := range directions {
			result := gs.ActionAttackLargeMap(direction)
			assert.False(t, result, "Large map attack should return false for direction %v", direction)
		}

		// Validate consistency
		assert.Equal(t, 4, len(mockCallbacks.Messages), "Should have message for each attack attempt")
		for _, message := range mockCallbacks.Messages {
			assert.Equal(t, "Nothing to attack!", message, "All large map attack messages should be consistent")
		}
	})
}

// TestAttackCommand_LargeMapAdjacentPirateStartsCombat prevents frigates from
// reporting "Not here!" when Attack targets an adjacent pirate ship.
func TestAttackCommand_LargeMapAdjacentPirateStartsCombat(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithSystemCallbacks().
		WithRandomSeed(179).
		Build()
	gs.DebugOptions.MonsterGen = false
	playerPosition, piratePosition := findAdjacentDeepWaterPositions(t, gs)
	gs.MapState.PlayerLocation.Position = playerPosition
	frigate := map_units.NewNPCFriendlyVehiceNewRef(
		references.FrigateVehicle,
		playerPosition,
		gs.MapState.PlayerLocation.Floor,
	)
	gs.PartyVehicle = *frigate
	addPirateTestEnemy(t, gs, piratePosition, indexes.PirateShipUp)
	var attackDirection references.Direction
	for _, direction := range []references.Direction{
		references.Up,
		references.Right,
		references.Down,
		references.Left,
	} {
		candidate := direction.GetNewPositionInDirection(&playerPosition).
			GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
		if *candidate == piratePosition {
			attackDirection = direction
			break
		}
	}
	require.NotEqual(t, references.NoneDirection, attackDirection)
	mockCallbacks.Reset()

	result := gs.ActionAttackLargeMap(attackDirection)

	require.True(t, result)
	require.NotNil(t, gs.CombatScene)
	require.NotNil(t, gs.CombatScene.OverworldOrigin)
	assert.Equal(t, OverworldContactPirateShip, gs.CombatScene.OverworldOrigin.ContactType)
	assert.Equal(t, references.ShipShipCombatMap, gs.CombatScene.OverworldOrigin.CombatMap)
	assert.NotContains(t, mockCallbacks.Messages, "Not here!")
	assert.Contains(t, mockCallbacks.Messages, "Attacked!")
}

// TestAttackCommand_CombatMap_NoCombatScene tests that combat attack safely returns false
// when no CombatScene is active (defensive guard).
func TestAttackCommand_CombatMap_NoCombatScene(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	mockCallbacks.Reset()

	// With no CombatScene, attack should return false immediately (nil guard)
	result := gs.ActionAttackCombatMap(references.Up)
	assert.False(t, result, "Combat attack returns false when no CombatScene active")

	// No messages produced — the nil guard exits before any feedback
	assert.Equal(t, 0, len(mockCallbacks.Messages), "No messages when CombatScene is nil")
	assert.Equal(t, 0, len(mockCallbacks.TimeAdvanced()), "No time advancement when CombatScene is nil")

	t.Run("AllDirections", func(t *testing.T) {
		mockCallbacks.Reset()
		directions := []references.Direction{references.Up, references.Down, references.Left, references.Right}
		for _, dir := range directions {
			result := gs.ActionAttackCombatMap(dir)
			assert.False(t, result, "Combat attack should return false for direction %v", dir)
		}
		assert.Equal(t, 0, len(mockCallbacks.Messages), "No messages for any direction when CombatScene is nil")
	})
}

// TestAttackCommand_DungeonMap_Integration tests an empty direct dungeon Attack.
func TestAttackCommand_DungeonMap_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithSystemCallbacks().
		Build()
	attachDungeonStateForTest(t, gs, references.DungeonCells{})
	mockCallbacks.Reset()

	token, _ := gs.BeginDungeonCommandCycle()
	result := gs.ActionAttackDungeonMap(references.NoneDirection)
	gs.CompleteDungeonCommandCycle(token, DungeonCommandOnly)

	assert.False(t, result)
	assert.Equal(t, []string{"What?"}, mockCallbacks.Messages)
	assert.Nil(t, gs.CombatScene)
	assert.True(t, gs.IsAtDungeonCommandBoundary())
}

// TestAttackCommand_SystemCallbacks_Integration tests comprehensive callback behavior
func TestAttackCommand_SystemCallbacks_Integration(t *testing.T) {
	// Test SystemCallbacks consistency across different map types
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	testCases := []struct {
		name         string
		attackFunc   func(references.Direction) bool
		expectMsg    string
		expectMsgCnt int
		expectTime   int
		expectResult bool
	}{
		{
			name:         "SmallMapAttack",
			attackFunc:   gs.ActionAttackSmallMap,
			expectMsg:    "Nothing to attack!",
			expectMsgCnt: 1,
			expectTime:   0,
			expectResult: false,
		},
		{
			name:         "LargeMapAttack",
			attackFunc:   gs.ActionAttackLargeMap,
			expectMsg:    "Not here!",
			expectMsgCnt: 1,
			expectTime:   0,
			expectResult: false,
		},
		{
			name:         "CombatMapAttack_NoCombatScene",
			attackFunc:   gs.ActionAttackCombatMap,
			expectMsg:    "",
			expectMsgCnt: 0, // No message when CombatScene is nil
			expectTime:   0,
			expectResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockCallbacks.Reset()

			result := tc.attackFunc(references.Up)

			// Validate consistent SystemCallbacks behavior
			assert.Equal(t, tc.expectResult, result, "Result should match expected")
			assert.Equal(t, tc.expectMsgCnt, len(mockCallbacks.Messages), "Should have correct message count")
			if tc.expectMsgCnt > 0 {
				assert.Equal(t, tc.expectMsg, mockCallbacks.Messages[0], "Should have correct message")
			}
			assert.Equal(t, tc.expectTime, len(mockCallbacks.TimeAdvanced()), "Should have correct time advancement")

			assert.Equal(t, 0, len(mockCallbacks.SoundEffectsPlayed), "Empty targets should not play sound effects")
		})
	}
}

// TestAttackCommand_ErrorHandling_Integration tests edge cases and error handling
func TestAttackCommand_ErrorHandling_Integration(t *testing.T) {
	// Test attack command error handling
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithPlayerAt(1, 1). // Near edge to test boundary conditions
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	t.Run("AttackAtMapEdges", func(t *testing.T) {
		// Test attack near map edges
		edgePositions := []references.Position{
			{X: 0, Y: 0}, {X: 1, Y: 1}, {X: 30, Y: 30},
		}

		originalPosition := gs.MapState.PlayerLocation.Position

		for _, pos := range edgePositions {
			gs.MapState.PlayerLocation.Position = pos
			mockCallbacks.Reset()

			result := gs.ActionAttackSmallMap(references.Up)

			// Should handle edge cases gracefully
			assert.False(t, result, "Attack should handle edge positions gracefully at (%d,%d)", pos.X, pos.Y)
			assert.Equal(t, 1, len(mockCallbacks.Messages), "Should have message at edge position (%d,%d)", pos.X, pos.Y)
			assert.Equal(t, "Nothing to attack!", mockCallbacks.Messages[0], "Should have consistent message at edge")
			assert.Equal(t, 0, len(mockCallbacks.TimeAdvanced()), "Should not advance time at edge positions")
		}

		// Restore original position
		gs.MapState.PlayerLocation.Position = originalPosition
	})
}

// TestAttackCommand_Regression_Integration tests empty targets by map type.
func TestAttackCommand_Regression_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithSystemCallbacks().
		Build()

	if gs == nil {
		return // Test was skipped due to missing game data
	}

	t.Run("EmptyTargetBaseline", func(t *testing.T) {
		mockCallbacks.Reset()

		// CombatMap is excluded: when CombatScene is nil it produces no messages,
		// so it can't be tested the same way as the other map types.
		attackFunctions := map[string]func(references.Direction) bool{
			"SmallMap": gs.ActionAttackSmallMap,
			"LargeMap": gs.ActionAttackLargeMap,
		}

		expectedMessages := map[string]string{
			"SmallMap": "Nothing to attack!",
			"LargeMap": "Not here!",
		}

		for mapType, attackFunc := range attackFunctions {
			mockCallbacks.Reset()
			result := attackFunc(references.Up)

			assert.False(t, result, "%s Attack should fail on an empty target", mapType)
			assert.Equal(t, expectedMessages[mapType], mockCallbacks.Messages[0], "%s empty-target message", mapType)
			assert.Equal(t, 0, len(mockCallbacks.TimeAdvanced()), "%s empty target should not advance time", mapType)
		}
	})
}
