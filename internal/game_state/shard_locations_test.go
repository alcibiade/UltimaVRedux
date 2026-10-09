package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

// TestShardLocations_ActualCoordinates tests shard placement at authentic Ultima V coordinates
func TestShardLocations_ActualCoordinates(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(-1).          // Shards are on underworld floor -1
		WithPlayerAt(100, 100). // Start away from shard locations
		WithSystemCallbacks().
		Build()

	// Test data from original outdoor effect rules
	// See ref:QUEST-0002.
	testCases := []struct {
		name          string
		position      references.Position
		expectedShard references.QuestItem
	}{
		{
			name:          "Shard of Falsehood at (192,80)",
			position:      references.Position{X: 192, Y: 80},
			expectedShard: references.ShardOfFalsehood,
		},
		{
			name:          "Shard of Hatred at (130,65)",
			position:      references.Position{X: 130, Y: 65},
			expectedShard: references.ShardOfHatred,
		},
		{
			name:          "Shard of Cowardice at (176,184)",
			position:      references.Position{X: 176, Y: 184},
			expectedShard: references.ShardOfCowardice,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Shard should be visible at correct position (since it's not in inventory)
			hasShardHere, questItem := gs.IsShardAtPosition(tc.position)
			assert.True(t, hasShardHere, "Shard should be visible at position %v", tc.position)
			assert.Equal(t, tc.expectedShard, questItem, "Wrong shard type at position %v", tc.position)

			// Test visibility logic directly
			isVisible := gs.IsShardVisible(tc.position, tc.expectedShard)
			assert.True(t, isVisible, "IsShardVisible should return true for %s", tc.name)

			// After adding the shard to inventory, it should no longer be visible
			gs.PartyState.Inventory.QuestItems.Set(tc.expectedShard, 1)
			hasShardHere, questItem = gs.IsShardAtPosition(tc.position)
			assert.True(t, hasShardHere, "Shard position should still be detected after adding to inventory at %v", tc.position)
			isVisible = gs.IsShardVisible(tc.position, questItem)
			assert.False(t, isVisible, "Shard should not be visible after adding to inventory at %v", tc.position)

			// Remove from inventory for next test
			gs.PartyState.Inventory.QuestItems.Set(tc.expectedShard, 0)
		})
	}
}

// TestShardLocations_GetSpritePlacement tests that shards appear correctly in the world
func TestShardLocations_GetSpritePlacement(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(-1).         // Shards are on underworld floor -1
		WithPlayerAt(192, 80). // Start at shard location
		WithSystemCallbacks().
		Build()

	// Move player to Shard of Falsehood location
	gs.MapState.PlayerLocation.Position = references.Position{X: 192, Y: 80}

	// Verify shard is at position (should be visible since not in inventory)
	hasShardHere, questItem := gs.IsShardAtPosition(references.Position{X: 192, Y: 80})
	assert.True(t, hasShardHere, "Shard of Falsehood should be visible at authentic coordinates when not in inventory")
	assert.Equal(t, references.ShardOfFalsehood, questItem, "Should be Shard of Falsehood")

	// Test that the sprite index is correct (using JSON data)
	invRefs := references.NewInventoryItemsReferences()
	spriteIndex := invRefs.GetQuestItemSpriteIndex(questItem)
	assert.Equal(t, GetShardSpriteIndex(), spriteIndex, "Shard should use correct sprite index")

	t.Logf("Shard of Falsehood found at (192,80) with sprite index %d", spriteIndex)
}

// TestShardLocations_PlacementData tests the placement data structure
func TestShardLocations_PlacementData(t *testing.T) {
	placementData := GetShardPlacementData()

	assert.Len(t, placementData, 3, "Should have placement data for all 3 shards")

	// Verify each shard has correct data
	expectedData := map[references.QuestItem]struct {
		position references.Position
		location references.Location
	}{
		references.ShardOfFalsehood: {
			position: references.Position{X: 192, Y: 80}, // 0xc0, 0x50
			location: references.BritanniaUnderworld,
		},
		references.ShardOfHatred: {
			position: references.Position{X: 130, Y: 65}, // 0x82, 0x41
			location: references.BritanniaUnderworld,
		},
		references.ShardOfCowardice: {
			position: references.Position{X: 176, Y: 184}, // 0xb0, 0xb8
			location: references.BritanniaUnderworld,
		},
	}

	invRefs := references.NewInventoryItemsReferences()
	for _, shardData := range placementData {
		expected := expectedData[shardData.QuestItemType]
		shardName := invRefs.GetQuestItemDisplayName(shardData.QuestItemType)

		assert.Equal(t, expected.position, shardData.Position,
			"Wrong position for quest item %s", shardName)
		assert.Equal(t, expected.location, shardData.Location,
			"Wrong location for quest item %s", shardName)
		assert.Equal(t, references.FloorNumber(-1), shardData.Floor,
			"All shards should be on underworld floor (-1)")
	}

	t.Logf("All three shard placements match the reference coordinates (ref:QUEST-0002)")
}

// TestShardLocations_MultipleShards tests handling multiple shards at once
func TestShardLocations_MultipleShards(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(-1). // Shards are on underworld floor -1
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()

	// All shards should be visible at their respective locations (since not in inventory)
	shardPositions := []references.Position{
		{X: 192, Y: 80},  // Falsehood
		{X: 130, Y: 65},  // Hatred
		{X: 176, Y: 184}, // Cowardice
	}

	for _, pos := range shardPositions {
		hasShardHere, _ := gs.IsShardAtPosition(pos)
		assert.True(t, hasShardHere, "Shard should be visible at position %v when not in inventory", pos)
	}

	// Test adding one shard to inventory doesn't affect others
	gs.PartyState.Inventory.QuestItems.Set(references.ShardOfFalsehood, 1)

	// Falsehood shard should be gone (not visible but position still detected)
	hasFirstShard, questItem := gs.IsShardAtPosition(references.Position{X: 192, Y: 80})
	assert.True(t, hasFirstShard, "Shard position should still be detected")
	isFirstShardVisible := gs.IsShardVisible(references.Position{X: 192, Y: 80}, questItem)
	assert.False(t, isFirstShardVisible, "Shard in inventory should not be visible")

	// Other shards should still be there
	hasSecondShard, _ := gs.IsShardAtPosition(references.Position{X: 130, Y: 65})
	hasThirdShard, _ := gs.IsShardAtPosition(references.Position{X: 176, Y: 184})
	assert.True(t, hasSecondShard, "Uncollected shard should still be visible")
	assert.True(t, hasThirdShard, "Uncollected shard should still be visible")

	// Verify no unexpected callbacks during state changes
	assert.Equal(t, 0, len(mockCallbacks.Messages), "No messages expected during state setup")
}
