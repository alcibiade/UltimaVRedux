package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestShrineRuinedState_Unit tests shrine ruined state management in isolation
// These tests verify the core ruined shrine getter/setter logic without dependencies
func TestShrineRuinedState_Unit(t *testing.T) {
	t.Run("IsShrineRuined_AllShrinesDefault_NotRuined", func(t *testing.T) {
		t.Parallel()
		gs := &GameState{SystemCallbacks: journalTestCallbacks()}
		// closedShrines is [8]bool, defaults to false for all shrines

		for i := 0; i < TotalShrines; i++ {
			got := gs.IsShrineRuined(i)
			assert.False(t, got, "Shrine %d should not be ruined by default", i)
		}
	})

	t.Run("SetShrineRuined_RuinSingleShrine_IsRuined", func(t *testing.T) {
		t.Parallel()
		gs := &GameState{SystemCallbacks: journalTestCallbacks()}

		gs.SetShrineRuined(0, true)

		assert.True(t, gs.IsShrineRuined(0), "Shrine 0 should be ruined after SetShrineRuined(0, true)")
		assert.False(t, gs.IsShrineRuined(1), "Shrine 1 should still not be ruined")
		assert.False(t, gs.IsShrineRuined(7), "Shrine 7 should still not be ruined")
	})

	t.Run("SetShrineRuined_RestoreRuinedShrine_NotRuined", func(t *testing.T) {
		t.Parallel()
		gs := &GameState{SystemCallbacks: journalTestCallbacks()}

		gs.SetShrineRuined(3, true)
		assert.True(t, gs.IsShrineRuined(3), "Shrine should be ruined initially")

		gs.SetShrineRuined(3, false)
		assert.False(t, gs.IsShrineRuined(3), "Shrine 3 should be restored after SetShrineRuined(3, false)")
	})

	t.Run("SetShrineRuined_InvalidIndex_Ignored", func(t *testing.T) {
		t.Parallel()
		gs := &GameState{SystemCallbacks: journalTestCallbacks()}

		// Should not panic or crash
		gs.SetShrineRuined(-1, true)
		gs.SetShrineRuined(TotalShrines, true)
		gs.SetShrineRuined(999, true)

		// Should not affect any actual shrines
		for i := 0; i < TotalShrines; i++ {
			assert.False(t, gs.IsShrineRuined(i), "Invalid indices should not affect shrine %d", i)
		}
	})

	t.Run("IsShrineRuined_InvalidIndex_ReturnsFalse", func(t *testing.T) {
		t.Parallel()
		gs := &GameState{SystemCallbacks: journalTestCallbacks()}

		assert.False(t, gs.IsShrineRuined(-1), "IsShrineRuined(-1) should return false safely")
		assert.False(t, gs.IsShrineRuined(TotalShrines), "IsShrineRuined(8) should return false safely")
		assert.False(t, gs.IsShrineRuined(999), "IsShrineRuined(999) should return false safely")
	})

	t.Run("SetShrineRuined_AllShrines_Independent", func(t *testing.T) {
		t.Parallel()
		gs := &GameState{SystemCallbacks: journalTestCallbacks()}

		// Ruin each shrine independently
		for i := 0; i < TotalShrines; i++ {
			gs.SetShrineRuined(i, true)

			// Verify only shrine i is ruined
			for j := 0; j < TotalShrines; j++ {
				if i == j {
					assert.True(t, gs.IsShrineRuined(j), "Shrine %d should be ruined", j)
				} else {
					assert.False(t, gs.IsShrineRuined(j), "Shrine %d should not be ruined", j)
				}
			}

			// Restore before next iteration
			gs.SetShrineRuined(i, false)
		}
	})

	t.Run("SetShrineRuined_MultipleToggling_Consistent", func(t *testing.T) {
		t.Parallel()
		gs := &GameState{SystemCallbacks: journalTestCallbacks()}

		shrine := 2

		// Toggle multiple times
		for iteration := 0; iteration < 5; iteration++ {
			// Ruin
			gs.SetShrineRuined(shrine, true)
			assert.True(t, gs.IsShrineRuined(shrine), "Iteration %d: Shrine should be ruined", iteration)

			// Restore
			gs.SetShrineRuined(shrine, false)
			assert.False(t, gs.IsShrineRuined(shrine), "Iteration %d: Shrine should be restored", iteration)
		}
	})

	t.Run("SetShrineRuined_BoundaryIndices_Valid", func(t *testing.T) {
		t.Parallel()
		gs := &GameState{SystemCallbacks: journalTestCallbacks()}

		// Index 0 (first shrine)
		gs.SetShrineRuined(0, true)
		assert.True(t, gs.IsShrineRuined(0), "Shrine 0 should work")

		// Index 7 (last shrine)
		gs.SetShrineRuined(7, true)
		assert.True(t, gs.IsShrineRuined(7), "Shrine 7 should work")

		// Reset
		gs.SetShrineRuined(0, false)
		gs.SetShrineRuined(7, false)

		assert.False(t, gs.IsShrineRuined(0), "Shrine 0 should be restored")
		assert.False(t, gs.IsShrineRuined(7), "Shrine 7 should be restored")
	})
}

// TestGetShrineForWordRuined_Unit tests shrine Word of Power recognition for ruined shrines
func TestGetShrineForWordRuined_Unit(t *testing.T) {
	t.Run("GetShrineForWord_AllEightWords_CorrectMapping", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			word      string
			wantIndex int
		}{
			{"FALLAX", 0},    // Honesty
			{"VILIS", 1},     // Compassion
			{"INOPIA", 2},    // Valour
			{"MALUM", 3},     // Justice
			{"AVIDUS", 4},    // Sacrifice
			{"INFAMA", 5},    // Honor
			{"IGNAVUS", 6},   // Spirituality
			{"VERAMOCOR", 7}, // Humility
		}

		for _, tc := range testCases {
			got := GetShrineForWordResponse(tc.word)
			assert.Equal(t, tc.wantIndex, got, "Word %q should map to shrine %d", tc.word, tc.wantIndex)
		}
	})

	t.Run("GetShrineForWord_CaseInsensitive_AllWorks", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			word      string
			wantIndex int
		}{
			{"fallax", 0},
			{"FALLAX", 0},
			{"FaLlAx", 0},
			{"vilis", 1},
			{"VILIS", 1},
			{"ViLiS", 1},
		}

		for _, tc := range testCases {
			got := GetShrineForWordResponse(tc.word)
			assert.Equal(t, tc.wantIndex, got, "Word %q should be case-insensitive and map to shrine %d", tc.word, tc.wantIndex)
		}
	})

	t.Run("GetShrineForWord_WithWhitespace_Works", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			word      string
			wantIndex int
		}{
			{"  FALLAX  ", 0},
			{" FALLAX", 0},
			{"FALLAX ", 0},
			{"\tVILIS\t", 1},
		}

		for _, tc := range testCases {
			got := GetShrineForWordResponse(tc.word)
			assert.Equal(t, tc.wantIndex, got, "Word %q with whitespace should map to shrine %d", tc.word, tc.wantIndex)
		}
	})

	t.Run("GetShrineForWord_InvalidWord_Returns_1", func(t *testing.T) {
		t.Parallel()

		testCases := []string{
			"INVALID",
			"",
			"FALL",    // Incomplete word
			"FAL LAX", // Space in middle
			"123",
		}

		for _, word := range testCases {
			got := GetShrineForWordResponse(word)
			assert.Equal(t, -1, got, "Invalid word %q should return -1", word)
		}
	})

	t.Run("GetShrineForWord_NoShrineShrines_NoDuplicates", func(t *testing.T) {
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

// TestGetShrineByPosition_Unit tests shrine position lookup
func TestGetShrineByPosition_Unit(t *testing.T) {
	t.Run("GetShrineByPosition_AllEightLocations_CorrectMapping", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name      string
			x, y      byte
			wantIndex int
		}{
			{"Honesty", 0xE9, 0x42, 0},
			{"Compassion", 0x80, 0x5C, 1},
			{"Valour", 0x24, 0xE5, 2},
			{"Justice", 0x49, 0x0B, 3},
			{"Sacrifice", 0xCD, 0x2D, 4},
			{"Honor", 0x51, 0xCF, 5},
			{"Spirituality", 0x00, 0x00, 6}, // Via moongate
			{"Humility", 0xE7, 0xD8, 7},
		}

		for _, tc := range testCases {
			got := GetShrineByPosition(tc.x, tc.y)
			assert.Equal(t, tc.wantIndex, got, "%s shrine at (%#x, %#x) should return %d", tc.name, tc.x, tc.y, tc.wantIndex)
		}
	})

	t.Run("GetShrineByPosition_InvalidLocation_Returns_1", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name  string
			x, y  byte
			notes string
		}{
			{"Completely invalid", 0xFF, 0xFF, "no shrine at this location"},
			{"Almost shrine location", 0xE8, 0x42, "off by one X"},
			{"Almost shrine location", 0xE9, 0x43, "off by one Y"},
			{"Zero coordinates", 0x00, 0x01, "Spirituality is 0,0 only"},
			{"Random location", 0x50, 0x50, "middle of nowhere"},
		}

		for _, tc := range testCases {
			got := GetShrineByPosition(tc.x, tc.y)
			assert.Equal(t, -1, got, "%s (%#x, %#x): %s", tc.name, tc.x, tc.y, tc.notes)
		}
	})

	t.Run("GetShrineByPosition_NoPartialMatches_Exact", func(t *testing.T) {
		t.Parallel()

		// Verify shrine lookup requires EXACT coordinates
		// Get valid shrine coordinate
		validShrine := GetShrineData(0) // Honesty at 0xE9, 0x42

		// Test exact match works
		got := GetShrineByPosition(validShrine.WorldX, validShrine.WorldY)
		assert.Equal(t, 0, got, "Exact coordinates should match")

		// Test off-by-one fails
		got = GetShrineByPosition(validShrine.WorldX+1, validShrine.WorldY)
		assert.Equal(t, -1, got, "Off-by-one X should not match")

		got = GetShrineByPosition(validShrine.WorldX, validShrine.WorldY+1)
		assert.Equal(t, -1, got, "Off-by-one Y should not match")

		got = GetShrineByPosition(validShrine.WorldX-1, validShrine.WorldY-1)
		assert.Equal(t, -1, got, "Off-by-one diagonal should not match")
	})

	t.Run("GetShrineByPosition_AllShrinesUnique_NoConflicts", func(t *testing.T) {
		t.Parallel()

		// Verify each shrine has a unique position (no conflicts in coordinates)
		positions := make(map[string]int)

		for i := 0; i < TotalShrines; i++ {
			shrine := GetShrineData(i)
			key := shrine.VirtueName // Use virtue name as position identifier

			shrine2 := GetShrineByPosition(shrine.WorldX, shrine.WorldY)
			assert.Equal(t, i, shrine2, "Position (%#x, %#x) should uniquely identify shrine %d", shrine.WorldX, shrine.WorldY, i)

			positions[key] = i
		}

		assert.Equal(t, TotalShrines, len(positions), "All shrines should have unique positions")
	})
}

// TestShrineDataConstants_Unit tests shrine data constants and values
func TestShrineDataConstants_Unit(t *testing.T) {
	t.Run("TotalShrines_Constant_EqualsEight", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 8, TotalShrines, "TotalShrines should be 8")
	})

	t.Run("QuestStateConstants_Distinct", func(t *testing.T) {
		t.Parallel()

		constants := []int{
			QuestStateNotAssigned,
			QuestStateReadyToComplete,
			QuestStateOffering,
		}

		// Check all distinct
		for i := 0; i < len(constants); i++ {
			for j := i + 1; j < len(constants); j++ {
				assert.NotEqual(t, constants[i], constants[j], "Quest state constants should be distinct")
			}
		}

		// Check values make sense
		assert.Equal(t, 0, QuestStateNotAssigned, "NotAssigned should be 0")
		assert.Equal(t, 1, QuestStateReadyToComplete, "ReadyToComplete should be 1")
		assert.Equal(t, 2, QuestStateOffering, "Offering should be 2")
	})

	t.Run("MaxStatValue_Reasonable", func(t *testing.T) {
		t.Parallel()
		assert.Greater(t, MaxStatValue, 20, "MaxStatValue should be at least 20")
		assert.Less(t, MaxStatValue, 100, "MaxStatValue should be less than 100")
		assert.Equal(t, 30, MaxStatValue, "MaxStatValue should be 30 (capped)")
	})

	t.Run("MantraRepetitions_Matches_Original", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 3, MantraRepetitions, "Mantra should be repeated 3 times per original game")
	})
}

// TestShrineDataIntegrity_Unit tests shrine data is complete and valid
func TestShrineDataIntegrity_Unit(t *testing.T) {
	t.Run("GetShrineData_AllShrinesValid", func(t *testing.T) {
		t.Parallel()

		for i := 0; i < TotalShrines; i++ {
			shrine := GetShrineData(i)
			assert.NotNil(t, shrine, "GetShrineData(%d) should not be nil", i)

			assert.Equal(t, i, shrine.Index, "Shrine %d should have correct index", i)
			assert.NotEmpty(t, shrine.VirtueName, "Shrine %d should have virtue name", i)
			assert.NotEmpty(t, shrine.VirtueKey, "Shrine %d should have virtue key", i)
			assert.NotEmpty(t, shrine.Mantra, "Shrine %d should have mantra", i)
			assert.NotEmpty(t, shrine.WordOfPower, "Shrine %d should have Word of Power", i)

			// WorldX and WorldY should be valid (can be 0 for Spirituality)
			// but most should be non-zero
			if i < 6 || i > 6 { // Skip Spirituality (index 6)
				assert.True(t, shrine.WorldX > 0 || shrine.WorldY > 0, "Shrine %d should have valid world coordinates", i)
			}
		}
	})

	t.Run("GetShrineData_InvalidIndices_ReturnNil", func(t *testing.T) {
		t.Parallel()

		assert.Nil(t, GetShrineData(-1), "GetShrineData(-1) should return nil")
		assert.Nil(t, GetShrineData(TotalShrines), "GetShrineData(8) should return nil")
		assert.Nil(t, GetShrineData(999), "GetShrineData(999) should return nil")
	})

	t.Run("ShrineData_StatBonuses_Valid", func(t *testing.T) {
		t.Parallel()

		for i := 0; i < TotalShrines; i++ {
			shrine := GetShrineData(i)

			// Bonuses should be 0 or 1 (byte values)
			assert.True(t, shrine.StrBonus <= 1, "Shrine %d STR bonus should be 0 or 1", i)
			assert.True(t, shrine.DexBonus <= 1, "Shrine %d DEX bonus should be 0 or 1", i)
			assert.True(t, shrine.IntBonus <= 1, "Shrine %d INT bonus should be 0 or 1", i)

			// Humility should give no stat bonuses but +6 karma instead
			if i == 7 {
				assert.Equal(t, byte(0), shrine.StrBonus, "Humility should give no STR bonus")
				assert.Equal(t, byte(0), shrine.DexBonus, "Humility should give no DEX bonus")
				assert.Equal(t, byte(0), shrine.IntBonus, "Humility should give no INT bonus")
				assert.Equal(t, 6, shrine.KarmaBonus(), "Humility should give +6 karma")
			} else {
				assert.Equal(t, 3, shrine.KarmaBonus(), "Non-Humility shrines should give +3 karma")
			}
		}
	})

	t.Run("ShrineData_KarmaBonuses_Correct", func(t *testing.T) {
		t.Parallel()

		for i := 0; i < TotalShrines; i++ {
			shrine := GetShrineData(i)

			if i == 7 { // Humility
				assert.Equal(t, 6, shrine.KarmaBonus(), "Humility should give +6 karma")
			} else {
				assert.Equal(t, 3, shrine.KarmaBonus(), "Shrine %d should give +3 karma", i)
			}
		}
	})
}
