package game_state

import (
	"encoding/hex"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSearchSystem_SaveDataIntegration_DeepDive performs an exhaustive analysis
// of the search system using real save data and validates all aspects of the
// authentic original Search/Get/Open/Jimmy rules implementation
// See ref:INV-0008.
func TestSearchSystem_SaveDataIntegration_DeepDive(t *testing.T) {
	// Load real game with save data
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	require.NotNil(t, gs, "Game state should be initialized")
	require.NotNil(t, gs.SearchManager, "Search manager should be initialized")

	aslm := gs.SearchManager.GetAuthenticLocationManager()
	require.NotNil(t, aslm, "Authentic search location manager should be initialized")

	t.Run("ExhaustiveLocationValidation", func(t *testing.T) {
		// Get all search locations from the system
		allLocations := references.GetAllSearchLocations()
		t.Logf("Total search locations in system: %d", len(allLocations))

		// Group locations by map for analysis
		locationsByMap := make(map[references.Location][]references.SearchLocationReference)
		for _, loc := range allLocations {
			mapLoc := references.Location(loc.MapNumber)
			locationsByMap[mapLoc] = append(locationsByMap[mapLoc], loc)
		}

		// Analyze distribution
		t.Log("=== SEARCH LOCATION DISTRIBUTION BY MAP ===")
		for mapLoc, locs := range locationsByMap {
			t.Logf("Map %d (%v): %d locations", mapLoc, mapLoc, len(locs))

			// Group by level within each map
			byLevel := make(map[int]int)
			for _, loc := range locs {
				byLevel[loc.Level]++
			}
			for level, count := range byLevel {
				t.Logf("  Level %d: %d locations", level, count)
			}
		}

		// Validate data integrity
		for _, loc := range allLocations {
			// Check that all locations have valid indices
			assert.True(t, loc.Index >= 0 && loc.Index < 113,
				"Location index %d should be in valid range [0,113)", loc.Index)

			// Check that quantities are valid (from objh[] array)
			// Note: Some original locations have quantity 0, which is authentic data
			assert.True(t, loc.Quantity >= 0 && loc.Quantity <= 255,
				"Quantity %d should be in valid range [0,255]", loc.Quantity)

			decoded := references.DecodeSearchLocationItem(loc)
			if loc.ObjectClass == references.SearchObjectDeadPC || loc.ObjectClass == references.SearchObjectDeadMonster {
				assert.Nil(t, decoded.Item)
			} else {
				assert.NotNil(t, decoded.Item)
			}
		}
	})

	t.Run("SaveDataFoundStatusIntegration", func(t *testing.T) {
		// Test save/load of found status
		originalStatus := aslm.getFoundStatus()
		t.Logf("Original found status (hex): %s", hex.EncodeToString(originalStatus[:]))

		// Mark some locations as found
		testIndices := []int{0, 5, 10, 15, 20, 42, 100}
		for _, idx := range testIndices {
			aslm.markAsFound(idx)
		}

		// Get the updated status
		modifiedStatus := aslm.getFoundStatus()
		t.Logf("Modified found status (hex): %s", hex.EncodeToString(modifiedStatus[:]))

		// Verify bits are set correctly
		for _, idx := range testIndices {
			assert.True(t, aslm.isAlreadyFound(idx),
				"Index %d should be marked as found", idx)
		}

		// Test that non-marked indices are still not found
		for idx := 0; idx < 113; idx++ {
			isTestIndex := false
			for _, testIdx := range testIndices {
				if idx == testIdx {
					isTestIndex = true
					break
				}
			}
			if !isTestIndex {
				assert.False(t, aslm.isAlreadyFound(idx),
					"Index %d should not be marked as found", idx)
			}
		}

		// Simulate loading from save file
		newAslm := NewAuthenticSearchLocationManager(gs)
		newAslm.setFoundStatus(modifiedStatus)

		// Verify loaded status matches
		for _, idx := range testIndices {
			assert.True(t, newAslm.isAlreadyFound(idx),
				"Index %d should still be marked as found after load", idx)
		}

		// Test reset functionality
		aslm.ResetAllFoundStatus()
		clearedStatus := aslm.getFoundStatus()
		for i := range clearedStatus {
			assert.Equal(t, byte(0), clearedStatus[i],
				"Byte %d should be cleared after reset", i)
		}
	})

	t.Run("ActualGameplaySearchSequence", func(t *testing.T) {
		// Simulate actual gameplay search sequence
		// Reset to clean state
		aslm.ResetAllFoundStatus()
		mockCallbacks.Reset()

		// Find all Moonglow locations for testing
		moongowLocations := []references.SearchLocationReference{}
		for _, loc := range references.GetAllSearchLocations() {
			if references.Location(loc.MapNumber) == references.Moonglow {
				moongowLocations = append(moongowLocations, loc)
			}
		}

		t.Logf("Testing %d Moonglow search locations", len(moongowLocations))

		// Test each location
		foundCount := 0
		for i, loc := range moongowLocations {
			// Move player to near the search location
			gs.MapState.PlayerLocation.Location = references.Moonglow
			gs.MapState.PlayerLocation.Floor = references.FloorNumber(loc.Level)

			// Calculate position one tile away from search location
			playerX := references.Coordinate(loc.X)
			playerY := references.Coordinate(loc.Y - 1) // Position player south of target
			if playerY < 0 {
				playerY = references.Coordinate(loc.Y + 1) // If at edge, position north
			}
			gs.MapState.PlayerLocation.Position = references.Position{X: playerX, Y: playerY}

			// Perform search action
			searchDirection := references.Up
			if playerY > references.Coordinate(loc.Y) {
				searchDirection = references.Down
			}

			// Execute the search
			result := gs.ActionSearchSmallMap(searchDirection)

			if result {
				foundCount++
				t.Logf("Location %d: Found %v (Qty: %d) at (%d,%d)",
					i, loc.ItemType, loc.Quantity, loc.X, loc.Y)

				// Verify the item was placed in ItemStacksMap at the search target position
				// The search target is the location being searched (where items are placed)
				playerPos := gs.MapState.PlayerLocation.Position
				searchTargetPos := searchDirection.GetNewPositionInDirection(&playerPos)
				assert.True(t, gs.ItemStacksMap.HasItemStackAtPosition(searchTargetPos),
					"Items should be placed at search target position (%d,%d)", searchTargetPos.X, searchTargetPos.Y)

				// Verify we can't search again at same location
				result2 := gs.ActionSearchSmallMap(searchDirection)
				assert.False(t, result2, "Second search at same location should fail")
			}
		}

		t.Logf("Successfully found %d/%d locations", foundCount, len(moongowLocations))

		// Verify messages were generated
		assert.Greater(t, len(mockCallbacks.Messages), 0, "Should have generated search messages")
	})

	t.Run("SpecialCaseValidation", func(t *testing.T) {
		// Deep test of special cases (indices 13-15)

		// Test Index 13 - Keys in Lord British's Castle
		keysLocation := references.SearchLocationReference{}
		hasKeysLocation := false
		for _, loc := range references.GetAllSearchLocations() {
			if loc.Index == 13 {
				keysLocation = loc
				hasKeysLocation = true
				break
			}
		}

		if hasKeysLocation {
			t.Log("=== Testing Keys Special Case (Index 13) ===")
			t.Logf("Location: Map=%d, Level=%d, X=%d, Y=%d",
				keysLocation.MapNumber, keysLocation.Level, keysLocation.X, keysLocation.Y)

			// Reset found status
			aslm.ResetAllFoundStatus()

			// Test with keys in inventory (should fail)
			gs.PartyState.Inventory.Provisions.Keys.Set(5)
			result := aslm.SearchAtPosition(
				references.Coordinate(keysLocation.X),
				references.Coordinate(keysLocation.Y),
				references.FloorNumber(keysLocation.Level),
				references.Location(keysLocation.MapNumber))
			assert.Nil(t, result, "Should not find keys when player has keys")

			// Test with no keys (should succeed)
			gs.PartyState.Inventory.Provisions.Keys.Set(0)
			result = aslm.SearchAtPosition(
				references.Coordinate(keysLocation.X),
				references.Coordinate(keysLocation.Y),
				references.FloorNumber(keysLocation.Level),
				references.Location(keysLocation.MapNumber))

			if keysLocation.Requirements.RequiresNoKeys {
				assert.NotNil(t, result, "Should find keys when player has no keys")
				if result != nil {
					assert.Equal(t, keysLocation.Quantity, result.Quantity,
						"Should return exact quantity from objh array")
				}
			}
		}

		// Test Index 14 - Skull Keys
		skullKeysLocation := references.SearchLocationReference{}
		hasSkullKeysLocation := false
		for _, loc := range references.GetAllSearchLocations() {
			if loc.Index == 14 {
				skullKeysLocation = loc
				hasSkullKeysLocation = true
				break
			}
		}

		if hasSkullKeysLocation {
			t.Log("=== Testing Skull Keys Special Case (Index 14) ===")
			t.Logf("Location: Map=%d, Level=%d, X=%d, Y=%d",
				skullKeysLocation.MapNumber, skullKeysLocation.Level,
				skullKeysLocation.X, skullKeysLocation.Y)

			// Reset found status
			aslm.ResetAllFoundStatus()

			// First search should succeed
			result := aslm.SearchAtPosition(
				references.Coordinate(skullKeysLocation.X),
				references.Coordinate(skullKeysLocation.Y),
				references.FloorNumber(skullKeysLocation.Level),
				references.Location(skullKeysLocation.MapNumber))

			if skullKeysLocation.Requirements.IsSkullKeys {
				assert.NotNil(t, result, "Should find skull keys on first search")

				// Second search should fail (already found)
				result2 := aslm.SearchAtPosition(
					references.Coordinate(skullKeysLocation.X),
					references.Coordinate(skullKeysLocation.Y),
					references.FloorNumber(skullKeysLocation.Level),
					references.Location(skullKeysLocation.MapNumber))
				assert.Nil(t, result2, "Should not find skull keys twice")
			}
		}
	})

	t.Run("BitmaskMathematicalValidation", func(t *testing.T) {
		// Validate the bitmask mathematics match original Search/Get/Open/Jimmy rules exactly
		// See ref:INV-0008.
		aslm.ResetAllFoundStatus()

		// Each discovered-search index sets its corresponding low-bit-first
		// bit within the byte at index divided by eight. ref:INV-0008
		testCases := []struct {
			index     int
			byteIndex int
			bitMask   byte
		}{
			{0, 0, 0x01},    // Index 0: byte 0, bit 0
			{1, 0, 0x02},    // Index 1: byte 0, bit 1
			{7, 0, 0x80},    // Index 7: byte 0, bit 7
			{8, 1, 0x01},    // Index 8: byte 1, bit 0
			{15, 1, 0x80},   // Index 15: byte 1, bit 7
			{42, 5, 0x04},   // Index 42: byte 5, bit 2
			{100, 12, 0x10}, // Index 100: byte 12, bit 4
			{112, 14, 0x01}, // Index 112: byte 14, bit 0
		}

		for _, tc := range testCases {
			// Mark as found
			aslm.markAsFound(tc.index)

			// Get the status and check the correct bit is set
			status := aslm.getFoundStatus()
			actualByte := status[tc.byteIndex]

			assert.True(t, (actualByte&tc.bitMask) != 0,
				"Index %d: byte[%d] should have bit mask 0x%02X set (actual: 0x%02X)",
				tc.index, tc.byteIndex, tc.bitMask, actualByte)

			// Verify isAlreadyFound returns true
			assert.True(t, aslm.isAlreadyFound(tc.index),
				"Index %d should be marked as found", tc.index)
		}
	})

	t.Run("QuantityAccuracyValidation", func(t *testing.T) {
		// Validate that quantities match exactly from objh[] array with NO randomization
		allLocations := references.GetAllSearchLocations()

		// Create fresh manager for clean tests
		freshAslm := NewAuthenticSearchLocationManager(gs)

		// Test a sample of locations to verify quantities
		sampleLocations := []references.SearchLocationReference{}
		if len(allLocations) > 0 {
			// Take first, middle, and last locations as samples
			sampleLocations = append(sampleLocations, allLocations[0])
			if len(allLocations) > 1 {
				sampleLocations = append(sampleLocations, allLocations[len(allLocations)/2])
				sampleLocations = append(sampleLocations, allLocations[len(allLocations)-1])
			}
		}

		for _, loc := range sampleLocations {
			// Skip special indices for this test
			if loc.Index >= 13 && loc.Index <= 15 {
				continue
			}

			result := freshAslm.SearchAtPosition(
				references.Coordinate(loc.X),
				references.Coordinate(loc.Y),
				references.FloorNumber(loc.Level),
				references.Location(loc.MapNumber))

			if result != nil {
				assert.Equal(t, loc.Quantity, result.Quantity,
					"Location index %d: quantity should match exactly from objh array (no randomization)",
					loc.Index)
				t.Logf("Verified exact quantity %d for index %d",
					result.Quantity, loc.Index)
			}
		}
	})

	t.Run("EdgeCasesAndErrorHandling", func(t *testing.T) {
		// Test boundary conditions and error cases

		// Test invalid indices
		assert.True(t, aslm.isAlreadyFound(-1), "Negative index should return true (already found)")
		assert.True(t, aslm.isAlreadyFound(113), "Index 113 should return true (out of range)")
		assert.True(t, aslm.isAlreadyFound(200), "Large index should return true (out of range)")

		// Test marking invalid indices (should not crash)
		aslm.markAsFound(-1)  // Should be no-op
		aslm.markAsFound(113) // Should be no-op
		aslm.markAsFound(200) // Should be no-op

		// Test searching at invalid coordinates
		result := aslm.SearchAtPosition(
			references.Coordinate(-1),
			references.Coordinate(-1),
			references.FloorNumber(-1),
			references.Location(255))
		assert.Nil(t, result, "Invalid coordinates should return nil")

		// Test searching at valid coordinates but wrong map
		if len(references.GetAllSearchLocations()) > 0 {
			loc := references.GetAllSearchLocations()[0]
			wrongMap := references.Location((loc.MapNumber + 1) % 256)
			result := aslm.SearchAtPosition(
				references.Coordinate(loc.X),
				references.Coordinate(loc.Y),
				references.FloorNumber(loc.Level),
				wrongMap)
			assert.Nil(t, result, "Wrong map should return nil")
		}
	})
}

// GetFoundStatus returns a copy of the objFound bitmask array for save file integration
func (aslm *AuthenticSearchLocationManager) getFoundStatus() [24]byte {
	return aslm.objFound
}

// SetFoundStatus sets the objFound bitmask array from save file data
func (aslm *AuthenticSearchLocationManager) setFoundStatus(foundStatus [24]byte) {
	aslm.objFound = foundStatus
}
