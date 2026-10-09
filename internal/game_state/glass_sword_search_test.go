package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGlassSwordSearch_Integration tests the complete glass sword discovery workflow
func TestGlassSwordSearch_Integration(t *testing.T) {
	// Glass sword location: BritanniaUnderworld at (64, 80), level 0
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(64, 81). // Position south of glass sword location
		WithSystemCallbacks().
		Build()

	t.Run("GlassSwordFound_WhenPlayerHasNone", func(t *testing.T) {
		// Ensure player has no glass swords initially
		glassSwordCount := gs.PartyState.Inventory.Equipment.Get(references.GlassSword)
		assert.Equal(t, uint16(0), glassSwordCount, "Player should start with no glass swords")

		mockCallbacks.Reset()

		// Search north to position (64, 80) where glass sword should be
		searchSuccess := gs.ActionSearchSmallMap(references.Up)

		// Verify search succeeded
		assert.True(t, searchSuccess, "Search should succeed")

		// Verify SystemCallbacks were used
		assert.Greater(t, len(mockCallbacks.Messages), 0, "Should have generated search messages")
		assert.Equal(t, "Thou dost find\na weapon!", mockCallbacks.Messages[0])

		// Verify item was placed on map for Get command
		searchPos := &references.Position{X: 64, Y: 80}
		assert.True(t, gs.ItemStacksMap.HasItemStackAtPosition(searchPos), "Glass sword should be on map")

		// Get the actual item from the map to verify details
		if gs.ItemStacksMap.HasItemStackAtPosition(searchPos) {
			topItem := gs.ItemStacksMap.Peek(searchPos)
			assert.NotNil(t, topItem, "Should have an item")
			assert.Equal(t, references.Equipment(references.GlassSword), topItem.Item, "Should find GlassSword equipment")
			assert.Equal(t, uint16(1), topItem.Quantity, "Should find 1 glass sword")
		}
	})

	t.Run("GlassSwordNotFound_WhenPlayerHasOne", func(t *testing.T) {
		// Give player a glass sword
		gs.PartyState.Inventory.Equipment.Set(references.GlassSword, 1)
		glassSwordCount := gs.PartyState.Inventory.Equipment.Get(references.GlassSword)
		assert.Equal(t, uint16(1), glassSwordCount, "Player should have 1 glass sword")

		// Clear any items from previous test
		searchPos := &references.Position{X: 64, Y: 80}
		for gs.ItemStacksMap.HasItemStackAtPosition(searchPos) {
			gs.ItemStacksMap.Pop(searchPos)
		}

		mockCallbacks.Reset()

		// Search north to position (64, 80) where glass sword should be
		searchSuccess := gs.ActionSearchSmallMap(references.Up)

		// Verify search failed because player already has glass sword
		assert.False(t, searchSuccess, "Should not find glass sword when player has one")
		assert.Greater(t, len(mockCallbacks.Messages), 0, "Should have generated search messages")
		assert.NotContains(t, mockCallbacks.Messages[0], "glass sword", "Message should not mention glass sword")

		// Verify no item was placed on map
		assert.False(t, gs.ItemStacksMap.HasItemStackAtPosition(searchPos), "No glass sword should be on map")
	})

	t.Run("GlassSwordSecondSearch_Fails", func(t *testing.T) {
		// Reset player inventory
		gs.PartyState.Inventory.Equipment.Set(references.GlassSword, 0)

		// Clear any items from previous tests
		searchPos := &references.Position{X: 64, Y: 80}
		for gs.ItemStacksMap.HasItemStackAtPosition(searchPos) {
			gs.ItemStacksMap.Pop(searchPos)
		}

		// First search should succeed
		firstSearchSuccess := gs.ActionSearchSmallMap(references.Up)
		assert.True(t, firstSearchSuccess, "First search should find glass sword")

		mockCallbacks.Reset()

		// Second search should fail (location already searched)
		secondSearchSuccess := gs.ActionSearchSmallMap(references.Up)
		assert.False(t, secondSearchSuccess, "Second search should fail")
		assert.Greater(t, len(mockCallbacks.Messages), 0, "Should have generated search messages")
		assert.Contains(t, mockCallbacks.Messages[0], MessageAlreadyFoundItems, "Should indicate items already found")
	})

	t.Run("GlassSwordLocation_Validation", func(t *testing.T) {
		// Verify the glass sword search location exists in the game data
		allLocations := references.GetAllSearchLocations()
		var glassSwordLocation *references.SearchLocationReference
		for _, loc := range allLocations {
			if loc.Index == 15 { // Glass sword index
				glassSwordLocation = &loc
				break
			}
		}

		require.NotNil(t, glassSwordLocation, "Glass sword search location should exist in game data")
		assert.Equal(t, int(references.BritanniaUnderworld), glassSwordLocation.MapNumber, "Glass sword should be in overworld")
		assert.Equal(t, 64, glassSwordLocation.X, "Glass sword X coordinate should be 64")
		assert.Equal(t, 80, glassSwordLocation.Y, "Glass sword Y coordinate should be 80")
		assert.Equal(t, 0, glassSwordLocation.Level, "Glass sword should be on surface level")
		assert.True(t, glassSwordLocation.Requirements.RequiresNoGlassSwords, "Glass sword should require no glass swords in inventory")
	})
}

// TestGlassSwordSearch_GetCommand_Integration tests the complete workflow of search → get
func TestGlassSwordSearch_GetCommand_Integration(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(64, 81). // Position south of glass sword location
		WithSystemCallbacks().
		Build()

	t.Run("SearchAndGet_Complete_Workflow", func(t *testing.T) {
		// Ensure player has no glass swords initially
		gs.PartyState.Inventory.Equipment.Set(references.GlassSword, 0)

		// Clear any existing items
		searchPos := &references.Position{X: 64, Y: 80}
		for gs.ItemStacksMap.HasItemStackAtPosition(searchPos) {
			gs.ItemStacksMap.Pop(searchPos)
		}

		// Search for glass sword
		searchSuccess := gs.ActionSearchSmallMap(references.Up)
		assert.True(t, searchSuccess, "Search should find glass sword")

		// Verify glass sword is on map at search position
		assert.True(t, gs.ItemStacksMap.HasItemStackAtPosition(searchPos), "Glass sword should be on map")

		// Player is at (64, 81), glass sword is at (64, 80) which is North (Up direction)
		mockCallbacks.Reset()

		// Get the glass sword (use ActionGetLargeMap since BritanniaUnderworld is a large map)
		getResult := gs.ActionGetLargeMap(references.Up) // Get from the north where item is
		assert.True(t, getResult, "Get should succeed")

		// Verify glass sword was added to inventory
		glassSwordCount := gs.PartyState.Inventory.Equipment.Get(references.GlassSword)
		assert.Equal(t, uint16(1), glassSwordCount, "Player should now have 1 glass sword")

		// Verify glass sword no longer on map
		assert.False(t, gs.ItemStacksMap.HasItemStackAtPosition(searchPos), "Glass sword should no longer be on map")

		// Verify proper messages were generated
		assert.Greater(t, len(mockCallbacks.Messages), 0, "Get command should generate messages")
	})
}

// TestGlassSwordSearch_EdgeCases tests various edge cases and error conditions
func TestGlassSwordSearch_EdgeCases(t *testing.T) {
	t.Run("WrongLocation_NoGlassSword", func(t *testing.T) {
		// Test searching at wrong location
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(50, 50). // Different location
			WithSystemCallbacks().
			Build()

		// Ensure player has no glass swords
		gs.PartyState.Inventory.Equipment.Set(references.GlassSword, 0)

		// Search should not find glass sword at wrong location
		searchSuccess := gs.ActionSearchSmallMap(references.Up)
		assert.False(t, searchSuccess, "Should not find glass sword at wrong location")
	})

	t.Run("WrongMap_NoGlassSword", func(t *testing.T) {
		// Test searching in wrong map
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain). // Wrong map
			WithPlayerAt(64, 81).
			WithSystemCallbacks().
			Build()

		// Ensure player has no glass swords
		gs.PartyState.Inventory.Equipment.Set(references.GlassSword, 0)

		// Search should not find glass sword in wrong map
		searchSuccess := gs.ActionSearchSmallMap(references.Up)
		assert.False(t, searchSuccess, "Should not find glass sword in wrong map")
	})

	t.Run("MultipleGlassSwords_StillBlocked", func(t *testing.T) {
		// Test that having multiple glass swords still blocks finding more
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(64, 81).
			WithSystemCallbacks().
			Build()

		// Give player multiple glass swords
		gs.PartyState.Inventory.Equipment.Set(references.GlassSword, 5)

		// Search should still fail
		searchSuccess := gs.ActionSearchSmallMap(references.Up)
		assert.False(t, searchSuccess, "Should not find glass sword when player has multiple")
	})
}

// TestGlassSwordSearch_AuthenticManager_UnitTests tests the authentic manager logic directly
func TestGlassSwordSearch_AuthenticManager_UnitTests(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithSystemCallbacks().
		Build()

	aslm := gs.SearchManager.GetAuthenticLocationManager()

	t.Run("meetsGlassSwordsRequirement_NoGlassSwords", func(t *testing.T) {
		// Player has no glass swords
		gs.PartyState.Inventory.Equipment.Set(references.GlassSword, 0)

		// Should meet requirements and find glass sword
		result := aslm.SearchAtPosition(64, 80, 0, references.BritanniaUnderworld)
		assert.NotNil(t, result, "Should find glass sword when player has none")
	})

	t.Run("meetsGlassSwordsRequirement_HasGlassSwords", func(t *testing.T) {
		// Player has glass swords
		gs.PartyState.Inventory.Equipment.Set(references.GlassSword, 1)

		// Should not meet requirements
		result := aslm.SearchAtPosition(64, 80, 0, references.BritanniaUnderworld)
		assert.Nil(t, result, "Should not find glass sword when player has one")
	})

	t.Run("meetsGlassSwordsRequirement_NoRequirement", func(t *testing.T) {
		// This tests the general logic that non-glass-sword locations aren't affected
		// We can't test this directly without modifying the search data, but the
		// authentic manager has the logic to handle RequiresNoGlassSwords=false
		t.Log("RequiresNoGlassSwords=false locations should work regardless of inventory")
	})
}
