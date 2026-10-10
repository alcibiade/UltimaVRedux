package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReagentCaches_Integration tests the complete reagent cache discovery workflow
func TestReagentCaches_Integration(t *testing.T) {
	// Test all 3 reagent cache locations from original Search/Get/Open/Jimmy rules reagent-cache search
	// See ref:INV-0008.
	testCases := []struct {
		name             string
		x, y             int
		expectedReagent  references.Reagent
		expectedMessage  string
		playerX, playerY int
	}{
		{
			name: "MandrakeRoot_Location1",
			x:    182, y: 54, // xreag[0] = 0xb6, yreag[0] = 0x36
			expectedReagent: references.MandrakeRoot,
			expectedMessage: "mandrake root!",
			playerX:         182, playerY: 55, // Position south of mandrake location
		},
		{
			name: "MandrakeRoot_Location2",
			x:    97, y: 165, // xreag[1] = 0x61, yreag[1] = 0xa5
			expectedReagent: references.MandrakeRoot,
			expectedMessage: "mandrake root!",
			playerX:         97, playerY: 166, // Position south of mandrake location
		},
		{
			name: "Nightshade_Location",
			x:    44, y: 137, // xreag[2] = 0x2c, yreag[2] = 0x89
			expectedReagent: references.NightShade,
			expectedMessage: "nightshade!",
			playerX:         44, playerY: 138, // Position south of nightshade location
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+"_Found_AtMidnight", func(t *testing.T) {
			gs, mockCallbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithPlayerAt(tc.playerX, tc.playerY).
				WithSystemCallbacks().
				Build()

			// Set time to midnight (hour 0)
			gs.DateTime.Hour = 0
			gs.DateTime.Minute = 0

			// Ensure no reagent in inventory initially
			reagentCount := gs.PartyState.Inventory.Reagent.Get(tc.expectedReagent)
			assert.Equal(t, uint16(0), reagentCount, "Player should start with no %s", tc.expectedReagent)

			mockCallbacks.Reset()

			// Search north to the reagent cache location
			searchSuccess := gs.ActionSearchLargeMap(references.Up)

			// Verify search succeeded
			assert.True(t, searchSuccess, "Search should succeed at midnight for %s", tc.name)

			// Verify SystemCallbacks were used
			assert.Greater(t, len(mockCallbacks.Messages), 0, "Should have generated search messages")
			assert.Contains(t, mockCallbacks.Messages[0], tc.expectedMessage, "Message should mention %s", tc.expectedMessage)

			// Verify reagent was added directly to inventory (authentic Ultima V behavior)
			reagentCount = gs.PartyState.Inventory.Reagent.Get(tc.expectedReagent)
			assert.GreaterOrEqual(t, reagentCount, uint16(2), "Should have at least 2 %s in inventory", tc.expectedReagent)
			assert.LessOrEqual(t, reagentCount, uint16(15), "Should have at most 15 %s in inventory", tc.expectedReagent)

			// Verify NO item was placed on map (reagents go directly to inventory)
			searchPos := &references.Position{X: references.Coordinate(tc.x), Y: references.Coordinate(tc.y)}
			assert.False(t, gs.ItemStacksMap.HasItemStackAtPosition(searchPos), "No %s should be on map - goes directly to inventory", tc.expectedReagent)
		})

		t.Run(tc.name+"_NotFound_NotMidnight", func(t *testing.T) {
			gs, mockCallbacks := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithPlayerAt(tc.playerX, tc.playerY).
				WithSystemCallbacks().
				Build()

			// Set time to non-midnight (hour 1)
			gs.DateTime.Hour = 1
			gs.DateTime.Minute = 0

			// Ensure no reagent in inventory initially
			initialReagentCount := gs.PartyState.Inventory.Reagent.Get(tc.expectedReagent)

			mockCallbacks.Reset()

			// Search north to reagent cache location - should fail due to time
			searchSuccess := gs.ActionSearchLargeMap(references.Up)

			// Verify search failed because it's not midnight
			assert.False(t, searchSuccess, "Should not find %s when not midnight", tc.expectedReagent)
			assert.Greater(t, len(mockCallbacks.Messages), 0, "Should have generated search messages")
			assert.NotContains(t, mockCallbacks.Messages[0], tc.expectedMessage, "Message should not mention %s", tc.expectedMessage)

			// Verify no reagent was added to inventory (failed search)
			reagentCount := gs.PartyState.Inventory.Reagent.Get(tc.expectedReagent)
			assert.Equal(t, initialReagentCount, reagentCount, "No %s should be added to inventory when search fails", tc.expectedReagent)

			// Verify no item was placed on map
			searchPos := &references.Position{X: references.Coordinate(tc.x), Y: references.Coordinate(tc.y)}
			assert.False(t, gs.ItemStacksMap.HasItemStackAtPosition(searchPos), "No %s should be on map", tc.expectedReagent)
		})
	}
}

// TestReagentCaches_DailyRegeneration tests daily regeneration mechanics
func TestReagentCaches_DailyRegeneration(t *testing.T) {
	// Test with nightshade location
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(44, 138). // Position south of nightshade location
		WithSystemCallbacks().
		Build()

	// Set time to midnight and day 15
	gs.DateTime.Hour = 0
	gs.DateTime.Day = 15

	// Record initial nightshade count
	initialNightshadeCount := gs.PartyState.Inventory.Reagent.Get(references.NightShade)

	// First search should succeed
	mockCallbacks.Reset()
	firstSearchSuccess := gs.ActionSearchLargeMap(references.Up)
	assert.True(t, firstSearchSuccess, "First search should find nightshade")

	// Verify nightshade was added to inventory
	firstSearchCount := gs.PartyState.Inventory.Reagent.Get(references.NightShade)
	assert.Greater(t, firstSearchCount, initialNightshadeCount, "First search should add nightshade to inventory")

	mockCallbacks.Reset()

	// Second search same day should fail (daily regeneration prevents)
	secondSearchSuccess := gs.ActionSearchLargeMap(references.Up)
	assert.False(t, secondSearchSuccess, "Second search same day should fail")
	assert.Greater(t, len(mockCallbacks.Messages), 0, "Should have generated search messages")
	assert.NotContains(t, mockCallbacks.Messages[0], "nightshade", "Should not find nightshade again same day")

	// Verify nightshade count unchanged (no additional nightshade added)
	secondSearchCount := gs.PartyState.Inventory.Reagent.Get(references.NightShade)
	assert.Equal(t, firstSearchCount, secondSearchCount, "Second search should not add more nightshade")

	// Advance to next day
	gs.DateTime.Day = 16

	mockCallbacks.Reset()

	// Search next day should succeed again
	thirdSearchSuccess := gs.ActionSearchLargeMap(references.Up)
	assert.True(t, thirdSearchSuccess, "Search next day should find nightshade again")
	assert.Contains(t, mockCallbacks.Messages[0], "nightshade!", "Should find nightshade next day")

	// Verify additional nightshade was added to inventory
	thirdSearchCount := gs.PartyState.Inventory.Reagent.Get(references.NightShade)
	assert.Greater(t, thirdSearchCount, secondSearchCount, "Third search should add more nightshade to inventory")
}

// TestReagentCaches_EdgeCases tests edge cases and error conditions
func TestReagentCaches_EdgeCases(t *testing.T) {
	t.Run("WrongMap_NoReagents", func(t *testing.T) {
		// Test searching in wrong map (should be overworld only)
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain). // Wrong map
			WithPlayerAt(44, 138).
			WithSystemCallbacks().
			Build()

		// Set time to midnight
		gs.DateTime.Hour = 0
		gs.DateTime.Day = 10

		// Search should not find reagents in wrong map
		searchSuccess := gs.ActionSearchSmallMap(references.Up)
		assert.False(t, searchSuccess, "Should not find reagents in wrong map")
	})

	t.Run("TimeBoundaryTesting", func(t *testing.T) {
		// Test hour boundaries around midnight
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.BritanniaUnderworld).
			WithPlayerAt(44, 138).
			WithSystemCallbacks().
			Build()

		// Test various hours
		testHours := []struct {
			hour     byte
			expected bool
			desc     string
		}{
			{23, false, "11 PM should not work"},
			{0, true, "Midnight should work"},
			{1, false, "1 AM should not work"},
		}

		for _, testCase := range testHours {
			// Set time and unique day
			gs.DateTime.Hour = testCase.hour
			gs.DateTime.Day = byte(40 + testCase.hour) // Unique day

			// Record initial nightshade count
			initialNightshadeCount := gs.PartyState.Inventory.Reagent.Get(references.NightShade)

			mockCallbacks.Reset()

			searchSuccess := gs.ActionSearchLargeMap(references.Up)

			if testCase.expected {
				assert.True(t, searchSuccess, "Hour %d (%s) should find nightshade", testCase.hour, testCase.desc)
				assert.Contains(t, mockCallbacks.Messages[0], "nightshade", "Should find nightshade at %s", testCase.desc)

				// Verify nightshade was added to inventory
				finalNightshadeCount := gs.PartyState.Inventory.Reagent.Get(references.NightShade)
				assert.Greater(t, finalNightshadeCount, initialNightshadeCount, "Nightshade should be added to inventory at %s", testCase.desc)
			} else {
				foundNightshade := len(mockCallbacks.Messages) > 0 &&
					len(mockCallbacks.Messages[0]) > 0 &&
					assert.ObjectsAreEqual("nightshade", mockCallbacks.Messages[0])
				assert.False(t, foundNightshade, "Hour %d (%s) should not find nightshade", testCase.hour, testCase.desc)

				// Verify nightshade was NOT added to inventory
				finalNightshadeCount := gs.PartyState.Inventory.Reagent.Get(references.NightShade)
				assert.Equal(t, initialNightshadeCount, finalNightshadeCount, "No nightshade should be added to inventory at %s", testCase.desc)
			}
		}
	})
}

// TestReagentCaches_MessageFormatting tests message formatting
func TestReagentCaches_MessageFormatting(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(44, 138).
		WithSystemCallbacks().
		Build()

	// Set time to midnight
	gs.DateTime.Hour = 0
	gs.DateTime.Day = 50

	// Record initial nightshade count
	initialNightshadeCount := gs.PartyState.Inventory.Reagent.Get(references.NightShade)

	// Search for nightshade
	searchSuccess := gs.ActionSearchLargeMap(references.Up)
	assert.True(t, searchSuccess, "Search should succeed")

	// Verify nightshade was added to inventory
	finalNightshadeCount := gs.PartyState.Inventory.Reagent.Get(references.NightShade)
	assert.Greater(t, finalNightshadeCount, initialNightshadeCount, "Nightshade should be added to inventory")

	// Verify message format matches expected pattern
	require.Greater(t, len(mockCallbacks.Messages), 0, "Should have search message")
	message := mockCallbacks.Messages[0]

	// Check message contains expected elements
	assert.Contains(t, message, "nightshade!", "Message should contain 'nightshade!'")

	// Verify the message has the expected format
	assert.Contains(t, message, "Thou dost find", "Should have search prefix")

	t.Logf("Nightshade search message: %s", message)
}
