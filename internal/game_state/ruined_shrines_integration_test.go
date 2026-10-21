package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

// TestGetEffectiveTileRuined_Integration tests tile display for ruined shrines
func TestGetEffectiveTileRuined_Integration(t *testing.T) {
	t.Run("GetEffectiveTile_RuinedShrine_GateTest", func(t *testing.T) {
		t.Parallel()

		// This test documents the expected behavior for shrine tile display
		// When GetEffectiveTile() is enhanced to check shrine ruined status:
		//   - Ruined shrines should return BrokenShrine sprite (index 26)
		//   - Active shrines should return Shrine sprite (index 25)
		//
		// NOTE: This test GATE will FAIL until GetEffectiveTile() enhancement
		// This documents the expected implementation point

		// Verify sprite indices exist (these are from original Ultima V)
		assert.NotEqual(t, indexes.Shrine, indexes.BrokenShrine, "Shrine and BrokenShrine should be different sprites")
		// Shrine = 25, BrokenShrine = 26 per internal/sprites/indexes/sprites.go
		assert.True(t, indexes.Shrine == 25, "Shrine sprite should be index 25")
		assert.True(t, indexes.BrokenShrine == 26, "BrokenShrine sprite should be index 26")

		// Shrine ruined state tracking works
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()

		assert.False(t, gs.IsShrineRuined(0), "Shrine should not be ruined initially")
		gs.SetShrineRuined(0, true)
		assert.True(t, gs.IsShrineRuined(0), "Shrine should be ruined after SetShrineRuined(true)")
		gs.SetShrineRuined(0, false)
		assert.False(t, gs.IsShrineRuined(0), "Shrine should be restored after SetShrineRuined(false)")

		// GetEffectiveTile() enhancement is now implemented!
		// Verify GetEffectiveTile() checks shrine ruined status and returns BrokenShrine when ruined
		// Use default BritanniaUnderworld location which is LargeMapType
		shrineData := GetShrineData(0) // Honesty shrine
		gs2, _ := NewIntegrationTestBuilder(t).
			WithPlayerAt(int(shrineData.WorldX), int(shrineData.WorldY)). // Position at shrine
			WithSystemCallbacks().
			Build()

		pos := &references.Position{
			X: references.Coordinate(shrineData.WorldX),
			Y: references.Coordinate(shrineData.WorldY),
		}

		// Active shrine should show Shrine sprite
		tile := gs2.GetEffectiveTile(pos)
		assert.Equal(t, indexes.SpriteIndex(indexes.Shrine), tile, "Active shrine should return Shrine sprite")

		// Ruin the shrine and verify it shows BrokenShrine
		gs2.SetShrineRuined(0, true)
		tile = gs2.GetEffectiveTile(pos)
		assert.Equal(t, indexes.SpriteIndex(indexes.BrokenShrine), tile, "Ruined shrine should return BrokenShrine sprite")

		// Restore and verify it returns to Shrine
		gs2.SetShrineRuined(0, false)
		tile = gs2.GetEffectiveTile(pos)
		assert.Equal(t, indexes.SpriteIndex(indexes.Shrine), tile, "Restored shrine should return Shrine sprite")
	})

	t.Run("GetEffectiveTile_AllEightShrines_PositionLookup", func(t *testing.T) {
		t.Parallel()

		// This test verifies that shrine positions are correctly defined in data
		// (Not testing tile rendering, just position data integrity)
		for i := 0; i < TotalShrines; i++ {
			shrine := GetShrineData(i)
			if shrine == nil {
				t.Errorf("Shrine %d data is nil", i)
				continue
			}

			// Skip Spirituality (position 0,0) for now - special moongate handling
			if i == 6 {
				continue
			}

			// Verify shrine position lookup works
			foundIndex := GetShrineByPosition(shrine.WorldX, shrine.WorldY)
			assert.Equal(t, i, foundIndex, "Shrine %d (%s) position (%#x, %#x) should lookup correctly",
				i, shrine.VirtueName, shrine.WorldX, shrine.WorldY)
		}
	})
}

// TestShrineRestorationWorkflow_Integration tests the complete restoration workflow
func TestShrineRestorationWorkflow_Integration(t *testing.T) {
	t.Run("Restoration_RuinedShrine_StateChange", func(t *testing.T) {
		t.Parallel()

		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()

		shrine := GetShrineData(0)
		if shrine == nil {
			t.Fatal("Shrine 0 data is nil")
		}

		// Initially not ruined
		assert.False(t, gs.IsShrineRuined(0), "Shrine should not be ruined initially")

		// Ruin it
		gs.SetShrineRuined(0, true)
		assert.True(t, gs.IsShrineRuined(0), "Shrine should be ruined after SetShrineRuined(true)")

		// Restore it
		gs.SetShrineRuined(0, false)
		assert.False(t, gs.IsShrineRuined(0), "Shrine should be restored after SetShrineRuined(false)")
	})

	t.Run("Restoration_AllEightShrines_Independent", func(t *testing.T) {
		t.Parallel()

		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()

		// Ruin each shrine one at a time
		for i := 0; i < TotalShrines; i++ {
			gs.SetShrineRuined(i, true)

			// Verify only shrine i is ruined
			for j := 0; j < TotalShrines; j++ {
				expected := (i == j)
				assert.Equal(t, expected, gs.IsShrineRuined(j),
					"After ruining shrine %d, shrine %d ruined status should be %v", i, j, expected)
			}

			// Restore before next iteration
			gs.SetShrineRuined(i, false)
		}
	})

	t.Run("Restoration_WordToShrineMapping_Complete", func(t *testing.T) {
		t.Parallel()

		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()

		// Verify each word maps to its shrine
		words := []string{"FALLAX", "VILIS", "INOPIA", "MALUM", "AVIDUS", "INFAMA", "IGNAVUS", "VERAMOCOR"}

		for i, word := range words {
			shrine := GetShrineData(i)
			assert.NotNil(t, shrine, "Shrine %d should exist", i)
			assert.Equal(t, word, shrine.WordOfPower, "Shrine %d should have word %s", i, word)

			// Verify word lookup
			idx := GetShrineForWordResponse(word)
			assert.Equal(t, i, idx, "Word %s should map to shrine %d", word, i)

			// Verify shrine is not ruined initially
			assert.False(t, gs.IsShrineRuined(i), "Shrine %d should not be ruined initially", i)
		}
	})

	t.Run("Restoration_RuineAllShrines_StateIndependence", func(t *testing.T) {
		t.Parallel()

		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithSystemCallbacks().
			Build()

		// Ruin all shrines
		for i := 0; i < TotalShrines; i++ {
			gs.SetShrineRuined(i, true)
		}

		// Verify all ruined
		for i := 0; i < TotalShrines; i++ {
			assert.True(t, gs.IsShrineRuined(i), "Shrine %d should be ruined", i)
		}

		// Restore all
		for i := 0; i < TotalShrines; i++ {
			gs.SetShrineRuined(i, false)
		}

		// Verify all restored
		for i := 0; i < TotalShrines; i++ {
			assert.False(t, gs.IsShrineRuined(i), "Shrine %d should be restored", i)
		}
	})
}

// TestProcessWordOfPowerRecognition_Integration tests Word of Power recognition
func TestProcessWordOfPowerRecognition_Integration(t *testing.T) {
	t.Run("GetShrineForWord_AllEightWords_IntegrationTest", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			word     string
			expected int
			virtue   string
		}{
			{"FALLAX", 0, "Honesty"},
			{"VILIS", 1, "Compassion"},
			{"INOPIA", 2, "Valour"},
			{"MALUM", 3, "Justice"},
			{"AVIDUS", 4, "Sacrifice"},
			{"INFAMA", 5, "Honor"},
			{"IGNAVUS", 6, "Spirituality"},
			{"VERAMOCOR", 7, "Humility"},
		}

		for _, tc := range testCases {
			shrine := GetShrineData(tc.expected)
			assert.NotNil(t, shrine, "Shrine %d (%s) should exist", tc.expected, tc.virtue)
			assert.Equal(t, tc.word, shrine.WordOfPower, "Shrine %d should have Word %s", tc.expected, tc.word)

			// Test lookup
			idx := GetShrineForWordResponse(tc.word)
			assert.Equal(t, tc.expected, idx, "Word %s should map to shrine %d (%s)", tc.word, tc.expected, tc.virtue)
		}
	})

	t.Run("GetShrineForWord_CaseInsensitive_IntegrationTest", func(t *testing.T) {
		t.Parallel()

		variations := []string{"FALLAX", "fallax", "FaLlAx", "  FALLAX  "}
		expectedShrine := 0

		for _, word := range variations {
			idx := GetShrineForWordResponse(word)
			assert.Equal(t, expectedShrine, idx, "Word %q should map to shrine %d", word, expectedShrine)
		}
	})

	t.Run("GetShrineForWord_InvalidWords_ReturnNegativeOne", func(t *testing.T) {
		t.Parallel()

		invalidWords := []string{
			"INVALID",
			"",
			"FALL",    // Incomplete
			"FAL LAX", // Space in middle
			"123",
		}

		for _, word := range invalidWords {
			idx := GetShrineForWordResponse(word)
			assert.Equal(t, -1, idx, "Invalid word %q should return -1", word)
		}
	})

	t.Run("GetShrineForWord_NoduplicatesMappings", func(t *testing.T) {
		t.Parallel()

		// Verify each word maps to exactly one unique shrine
		wordToShrine := make(map[int]string)

		words := []string{"FALLAX", "VILIS", "INOPIA", "MALUM", "AVIDUS", "INFAMA", "IGNAVUS", "VERAMOCOR"}
		expectedShrines := []int{0, 1, 2, 3, 4, 5, 6, 7}

		for i, word := range words {
			shrine := GetShrineForWordResponse(word)
			assert.Equal(t, expectedShrines[i], shrine, "Word %q should map to shrine %d", word, expectedShrines[i])

			// Check for duplicates
			if prevWord, exists := wordToShrine[shrine]; exists {
				t.Errorf("Shrine %d maps to multiple words: %q and %q", shrine, prevWord, word)
			}
			wordToShrine[shrine] = word
		}

		// Should have exactly 8 unique mappings
		assert.Equal(t, TotalShrines, len(wordToShrine), "Should have exactly %d word mappings", TotalShrines)
	})
}

// TestShrineProximityDetection_Integration tests proximity detection near shrine locations
func TestShrineProximityDetection_Integration(t *testing.T) {
	t.Run("ShrineLocation_AllEightPositions_Documented", func(t *testing.T) {
		t.Parallel()

		// This test documents all 8 shrine locations for reference
		testCases := []struct {
			name  string
			index int
			x     byte
			y     byte
		}{
			{"Honesty", 0, 0xE9, 0x42},
			{"Compassion", 1, 0x80, 0x5C},
			{"Valour", 2, 0x24, 0xE5},
			{"Justice", 3, 0x49, 0x0B},
			{"Sacrifice", 4, 0xCD, 0x2D},
			{"Honor", 5, 0x51, 0xCF},
			{"Spirituality", 6, 0x00, 0x00},
			{"Humility", 7, 0xE7, 0xD8},
		}

		for _, tc := range testCases {
			shrine := GetShrineData(tc.index)
			assert.NotNil(t, shrine, "%s shrine data should exist", tc.name)
			assert.Equal(t, tc.x, shrine.WorldX, "%s shrine X coordinate should match", tc.name)
			assert.Equal(t, tc.y, shrine.WorldY, "%s shrine Y coordinate should match", tc.name)
		}
	})

	t.Run("ShrinePosition_RNG_NoRandomization", func(t *testing.T) {
		t.Parallel()

		gs := &GameState{}
		gs.SetRandomSeed(12345)

		// Shrines have fixed world positions that never change
		for i := 0; i < TotalShrines; i++ {
			shrine := GetShrineData(i)
			assert.NotNil(t, shrine, "Shrine %d should have data", i)

			// Position should not be random
			expectedIdx := GetShrineByPosition(shrine.WorldX, shrine.WorldY)
			assert.Equal(t, i, expectedIdx, "Shrine %d position should be exact", i)
		}
	})
}

// TestShrineDataConsistency_Integration tests data consistency across lookups
func TestShrineDataConsistency_Integration(t *testing.T) {
	t.Run("ShrineData_CircularLookups_Consistent", func(t *testing.T) {
		t.Parallel()

		// Verify circular lookups work correctly:
		// Index → Data → Word → Index should return same index
		// Index → Data → Position → Index should return same index

		for i := 0; i < TotalShrines; i++ {
			// Index → Data
			shrine := GetShrineData(i)
			assert.NotNil(t, shrine, "GetShrineData(%d) should return data", i)

			// Data → Word → Index
			idx := GetShrineForWordResponse(shrine.WordOfPower)
			assert.Equal(t, i, idx, "Word-based lookup should return shrine %d", i)

			// Data → Position → Index (skip Spirituality which has special handling)
			if i != 6 {
				idx := GetShrineByPosition(shrine.WorldX, shrine.WorldY)
				assert.Equal(t, i, idx, "Position-based lookup should return shrine %d", i)
			}
		}
	})

	t.Run("ShrineData_AllFields_NonEmpty", func(t *testing.T) {
		t.Parallel()

		for i := 0; i < TotalShrines; i++ {
			shrine := GetShrineData(i)
			assert.NotNil(t, shrine, "Shrine %d should have data", i)

			// Check all string fields are populated
			assert.NotEmpty(t, shrine.VirtueName, "Shrine %d should have virtue name", i)
			assert.NotEmpty(t, shrine.VirtueKey, "Shrine %d should have virtue key", i)
			assert.NotEmpty(t, shrine.Mantra, "Shrine %d should have mantra", i)
			assert.NotEmpty(t, shrine.WordOfPower, "Shrine %d should have Word of Power", i)

			// Check index matches
			assert.Equal(t, i, shrine.Index, "Shrine %d should have correct index", i)
		}
	})
}

// TestActionEnterRuinedShrine_Integration tests the Enter command behavior for ruined shrines
func TestActionEnterRuinedShrine_Integration(t *testing.T) {
	t.Run("Enter_RuinedShrine_ShowsEnterRuins", func(t *testing.T) {
		t.Parallel()

		// Set up player at the Honesty shrine position on the large map
		shrineData := GetShrineData(0) // Honesty shrine
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithPlayerAt(int(shrineData.WorldX), int(shrineData.WorldY)).
			WithSystemCallbacks().
			Build()

		// Ruin the shrine
		gs.SetShrineRuined(0, true)

		// Attempt to enter - should append "ruins" to the Enter command prompt.
		result := gs.ActionEnterLargeMap()

		// Enter should return false (did not enter)
		assert.False(t, result, "ActionEnterLargeMap should return false for ruined shrine")

		assert.Equal(t, []string{"Enter "}, mockCallbacks.CommandPrompts)
		assert.Equal(t, "ruins", mockCallbacks.CurrentRowContent)
	})

	t.Run("Enter_ActiveShrine_ReturnsTrue", func(t *testing.T) {
		t.Parallel()

		// Set up player at the Honesty shrine position on the large map
		shrineData := GetShrineData(0) // Honesty shrine
		gs, _ := NewIntegrationTestBuilder(t).
			WithPlayerAt(int(shrineData.WorldX), int(shrineData.WorldY)).
			WithSystemCallbacks().
			Build()

		// Shrine is NOT ruined (default state)
		assert.False(t, gs.IsShrineRuined(0), "Shrine should not be ruined initially")

		// Attempt to enter - should return true (StartShrineWorkflow is called)
		result := gs.ActionEnterLargeMap()

		// Enter should return true (entered shrine - workflow started)
		assert.True(t, result, "ActionEnterLargeMap should return true for active shrine")
	})

	t.Run("Enter_RestoredShrine_ReturnsTrue", func(t *testing.T) {
		t.Parallel()

		// Set up player at the Honesty shrine position
		shrineData := GetShrineData(0)
		gs, _ := NewIntegrationTestBuilder(t).
			WithPlayerAt(int(shrineData.WorldX), int(shrineData.WorldY)).
			WithSystemCallbacks().
			Build()

		// Ruin then restore the shrine
		gs.SetShrineRuined(0, true)
		gs.SetShrineRuined(0, false)

		// Attempt to enter - should return true
		result := gs.ActionEnterLargeMap()

		// Enter should return true (entered restored shrine)
		assert.True(t, result, "ActionEnterLargeMap should return true for restored shrine")
	})
}

// TestGetAdjacentRuinedShrine_Integration tests proximity detection for ruined shrine restoration
func TestGetAdjacentRuinedShrine_Integration(t *testing.T) {
	t.Run("GetAdjacentRuinedShrine_CorrectWordAdjacentShrine_ReturnsShrineData", func(t *testing.T) {
		t.Parallel()

		// Set up player adjacent to Honesty shrine (one tile south)
		shrineData := GetShrineData(0) // Honesty shrine at (0xE9, 0x42)
		gs, _ := NewIntegrationTestBuilder(t).
			WithPlayerAt(int(shrineData.WorldX), int(shrineData.WorldY)+1). // One tile south
			WithSystemCallbacks().
			Build()

		// Ruin the shrine
		gs.SetShrineRuined(0, true)

		// Player yells "FALLAX" (Word of Power for Honesty)
		result := gs.GetAdjacentRuinedShrine("FALLAX")

		assert.NotNil(t, result, "Should return ShrineData when adjacent to ruined shrine with correct word")
		assert.Equal(t, 0, result.Index, "Should return Honesty shrine (index 0)")
		assert.Equal(t, "Honesty", result.VirtueName, "Should return correct virtue name")
	})

	t.Run("GetAdjacentRuinedShrine_CorrectWordAllDirections_ReturnsShrineData", func(t *testing.T) {
		t.Parallel()

		shrineData := GetShrineData(0) // Honesty shrine
		directions := []struct {
			name string
			dx   int
			dy   int
		}{
			{"North", 0, -1},
			{"South", 0, 1},
			{"East", 1, 0},
			{"West", -1, 0},
		}

		for _, dir := range directions {
			gs, _ := NewIntegrationTestBuilder(t).
				WithPlayerAt(int(shrineData.WorldX)+dir.dx, int(shrineData.WorldY)+dir.dy).
				WithSystemCallbacks().
				Build()

			gs.SetShrineRuined(0, true)

			result := gs.GetAdjacentRuinedShrine("FALLAX")

			assert.NotNil(t, result, "Should return ShrineData when adjacent %s", dir.name)
			assert.Equal(t, 0, result.Index, "Should return correct shrine index from %s", dir.name)
		}
	})

	t.Run("GetAdjacentRuinedShrine_WrongWord_ReturnsNil", func(t *testing.T) {
		t.Parallel()

		// Set up player adjacent to Honesty shrine
		shrineData := GetShrineData(0)
		gs, _ := NewIntegrationTestBuilder(t).
			WithPlayerAt(int(shrineData.WorldX), int(shrineData.WorldY)+1).
			WithSystemCallbacks().
			Build()

		// Ruin the shrine
		gs.SetShrineRuined(0, true)

		// Try different incorrect words
		wrongWords := []string{"VILIS", "INOPIA", "MALUM", "INVALID", ""}

		for _, word := range wrongWords {
			result := gs.GetAdjacentRuinedShrine(word)
			assert.Nil(t, result, "Should return nil for word %q (not Honesty's word)", word)
		}
	})

	t.Run("GetAdjacentRuinedShrine_NotAdjacent_ReturnsNil", func(t *testing.T) {
		t.Parallel()

		// Set up player NOT adjacent to any shrine
		gs, _ := NewIntegrationTestBuilder(t).
			WithPlayerAt(100, 100). // Far from any shrine
			WithSystemCallbacks().
			Build()

		// Ruin the Honesty shrine
		gs.SetShrineRuined(0, true)

		// Try the correct word but player is not adjacent
		result := gs.GetAdjacentRuinedShrine("FALLAX")

		assert.Nil(t, result, "Should return nil when player not adjacent to ruined shrine")
	})

	t.Run("GetAdjacentRuinedShrine_ShrineNotRuined_ReturnsNil", func(t *testing.T) {
		t.Parallel()

		// Set up player adjacent to Honesty shrine
		shrineData := GetShrineData(0)
		gs, _ := NewIntegrationTestBuilder(t).
			WithPlayerAt(int(shrineData.WorldX), int(shrineData.WorldY)+1).
			WithSystemCallbacks().
			Build()

		// Shrine is NOT ruined (default state)
		assert.False(t, gs.IsShrineRuined(0), "Shrine should not be ruined")

		// Try the correct word but shrine is not ruined
		result := gs.GetAdjacentRuinedShrine("FALLAX")

		assert.Nil(t, result, "Should return nil when adjacent shrine is not ruined")
	})

	t.Run("GetAdjacentRuinedShrine_CaseInsensitive_ReturnsShrineData", func(t *testing.T) {
		t.Parallel()

		// Set up player adjacent to Honesty shrine
		shrineData := GetShrineData(0)
		gs, _ := NewIntegrationTestBuilder(t).
			WithPlayerAt(int(shrineData.WorldX), int(shrineData.WorldY)+1).
			WithSystemCallbacks().
			Build()

		gs.SetShrineRuined(0, true)

		// Try different case variations
		variations := []string{"FALLAX", "fallax", "FaLlAx", "  fallax  ", "speak FALLAX now"}

		for _, word := range variations {
			result := gs.GetAdjacentRuinedShrine(word)
			assert.NotNil(t, result, "Should return ShrineData for word %q (case-insensitive)", word)
			assert.Equal(t, 0, result.Index, "Should return correct shrine index for word %q", word)
		}
	})

	t.Run("GetAdjacentRuinedShrine_PartialWord_ReturnsNil", func(t *testing.T) {
		t.Parallel()

		shrineData := GetShrineData(0)
		gs, _ := NewIntegrationTestBuilder(t).
			WithPlayerAt(int(shrineData.WorldX), int(shrineData.WorldY)+1).
			WithSystemCallbacks().
			Build()
		gs.SetShrineRuined(0, true)

		for _, response := range []string{"FALL", "ALLAX", ""} {
			assert.Nil(t, gs.GetAdjacentRuinedShrine(response))
		}
	})

	t.Run("GetAdjacentRuinedShrine_DiagonalNotAdjacent_ReturnsNil", func(t *testing.T) {
		t.Parallel()

		// Set up player DIAGONAL to shrine (not adjacent in original game logic)
		shrineData := GetShrineData(0)
		gs, _ := NewIntegrationTestBuilder(t).
			WithPlayerAt(int(shrineData.WorldX)+1, int(shrineData.WorldY)+1). // Diagonal (SE)
			WithSystemCallbacks().
			Build()

		gs.SetShrineRuined(0, true)

		result := gs.GetAdjacentRuinedShrine("FALLAX")

		assert.Nil(t, result, "Should return nil for diagonal position (not cardinal adjacent)")
	})

	t.Run("GetAdjacentRuinedShrine_AllEightShrines_CorrectWord", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			shrineIdx int
			word      string
			virtue    string
		}{
			{0, "FALLAX", "Honesty"},
			{1, "VILIS", "Compassion"},
			{2, "INOPIA", "Valour"},
			{3, "MALUM", "Justice"},
			{4, "AVIDUS", "Sacrifice"},
			{5, "INFAMA", "Honor"},
			{6, "IGNAVUS", "Spirituality"},
			{7, "VERAMOCOR", "Humility"},
		}

		for _, tc := range testCases {
			shrineData := GetShrineData(tc.shrineIdx)
			if shrineData == nil {
				t.Errorf("Shrine %d (%s) data is nil", tc.shrineIdx, tc.virtue)
				continue
			}

			// Skip Spirituality (index 6) which has special position (0,0)
			if tc.shrineIdx == 6 {
				continue
			}

			gs, _ := NewIntegrationTestBuilder(t).
				WithPlayerAt(int(shrineData.WorldX), int(shrineData.WorldY)+1).
				WithSystemCallbacks().
				Build()

			gs.SetShrineRuined(tc.shrineIdx, true)

			result := gs.GetAdjacentRuinedShrine(tc.word)

			assert.NotNil(t, result, "%s shrine: should return ShrineData for word %s", tc.virtue, tc.word)
			if result != nil {
				assert.Equal(t, tc.shrineIdx, result.Index, "%s shrine: should return correct index", tc.virtue)
			}
		}
	})
}
