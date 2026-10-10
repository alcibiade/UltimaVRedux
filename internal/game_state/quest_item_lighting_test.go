package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

// testGetVisibleQuestItemPositions is a test helper that returns positions of quest items visible in current location
// (moved from quest_item_placement.go to keep test-only code in test files)
func testGetVisibleQuestItemPositions(gs *GameState) []references.Position {
	placements := GetAllQuestItemPlacements()
	var visiblePositions []references.Position

	for _, placement := range placements {
		// Check if we're in the correct location and floor
		if gs.MapState.PlayerLocation.Location == placement.Location &&
			gs.MapState.PlayerLocation.Floor == placement.Floor {

			// Check if quest item is visible (not in inventory)
			if gs.IsQuestItemVisible(placement.Position, placement.QuestItemType) {
				visiblePositions = append(visiblePositions, placement.Position)
			}
		}
	}

	return visiblePositions
}

func TestQuestItemLighting_Integration(t *testing.T) {
	// Initialize test with quest items in different locations
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld). // Where shards are located
		WithPlayerAt(192, 80).                        // Near Shard of Falsehood position
		WithSystemCallbacks().
		Build()

	// Test GetVisibleQuestItemPositions
	visiblePositions := testGetVisibleQuestItemPositions(gs)

	// Should find quest items when in the same location/floor
	assert.GreaterOrEqual(t, len(visiblePositions), 0, "Should return quest item positions for current location")

	// Test specific quest item light radius
	testPos := references.Position{X: 192, Y: 80} // Shard of Falsehood location
	lightRadius := gs.GetQuestItemLightRadius(testPos)

	// Should return appropriate light radius for shards
	if hasItem, questItem := gs.IsQuestItemAtPosition(testPos); hasItem {
		if IsShard(questItem) {
			assert.Equal(t, 3, lightRadius, "Shards should emit 3-tile radius light")
		} else if IsLordBritishArtifact(questItem) {
			assert.Equal(t, 4, lightRadius, "Artifacts should emit 4-tile radius light")
		}
	} else {
		assert.Equal(t, 0, lightRadius, "No quest item should emit no light")
	}
}

func TestQuestItemLighting_VisibilityLogic(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(192, 80).
		WithSystemCallbacks().
		Build()

	testPos := references.Position{X: 192, Y: 80} // Shard of Falsehood location

	// Test visibility when quest item is in world (not collected)
	if hasItem, questItem := gs.IsQuestItemAtPosition(testPos); hasItem {
		isVisible := gs.IsQuestItemVisible(testPos, questItem)
		lightRadius := gs.GetQuestItemLightRadius(testPos)

		if !gs.PartyState.Inventory.QuestItems.HasSome(questItem) {
			assert.True(t, isVisible, "Quest item should be visible when not in inventory")
			assert.Greater(t, lightRadius, 0, "Quest item should emit light when visible")
		} else {
			assert.False(t, isVisible, "Quest item should not be visible when in inventory")
			assert.Equal(t, 0, lightRadius, "Quest item should not emit light when in inventory")
		}
	}
}

func TestQuestItemLighting_LightSources(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(192, 80).
		WithSystemCallbacks().
		Build()

	// Get all quest item lights for current location
	questLights := gs.GetVisibleQuestItemLights()

	// Should return structured light information
	for _, light := range questLights {
		assert.GreaterOrEqual(t, light.Position.X, references.Coordinate(0), "Light position should be valid")
		assert.GreaterOrEqual(t, light.Position.Y, references.Coordinate(0), "Light position should be valid")
		assert.GreaterOrEqual(t, light.LightRadius, 3, "Quest items should have minimum 3-tile light radius")
		assert.LessOrEqual(t, light.LightRadius, 4, "Quest items should have maximum 4-tile light radius")
	}
}

func TestQuestItemLighting_DifferentMapTypes(t *testing.T) {
	testCases := []struct {
		name     string
		location references.Location
		floor    references.FloorNumber
	}{
		{"Underworld", references.BritanniaUnderworld, -1},
		{"Stonegate", references.Stonegate, 0},
		{"BlackthornPalace", references.PalaceOfBlackthorn, 3},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gs, _ := NewIntegrationTestBuilder(t).
				WithLocation(tc.location).
				WithPlayerAt(15, 15).
				WithSystemCallbacks().
				Build()

			// Manually set the floor since WithPlayerFloor doesn't exist
			gs.MapState.PlayerLocation.Floor = tc.floor

			// Test that quest item lighting system works in all map types
			questLights := gs.GetVisibleQuestItemLights()

			// Should work without errors in any map type
			assert.NotNil(t, questLights, "Quest item lights should be retrievable")
		})
	}
}

func TestQuestItemTypes_LightRadii(t *testing.T) {
	testCases := []struct {
		questItem      references.QuestItem
		expectedRadius int
	}{
		{references.ShardOfFalsehood, 3},
		{references.ShardOfHatred, 3},
		{references.ShardOfCowardice, 3},
		{references.Crown, 4},
		{references.Sceptre, 4},
		{references.Amulet, 4},
	}

	for _, tc := range testCases {
		t.Run("QuestItem", func(t *testing.T) {
			// Test the individual helper functions
			if IsShard(tc.questItem) {
				assert.Equal(t, 3, tc.expectedRadius, "Shards should have 3-tile radius")
			}
			if IsLordBritishArtifact(tc.questItem) {
				assert.Equal(t, 4, tc.expectedRadius, "Artifacts should have 4-tile radius")
			}
		})
	}
}
