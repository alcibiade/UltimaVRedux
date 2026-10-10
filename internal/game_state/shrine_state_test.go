package game_state

import (
	"testing"
)

// createTestGameStateForShrine creates a minimal GameState for shrine testing
func createTestGameStateForShrine() *GameState {
	gs := &GameState{SystemCallbacks: journalTestCallbacks()} //nolint:exhaustruct
	// Fields are zero-initialized, which is correct for testing
	return gs
}

// TestShrineGettersSetters_RoundTrip verifies set then get returns same value
func TestShrineGettersSetters_RoundTrip(t *testing.T) {
	t.Parallel()

	gs := createTestGameStateForShrine()

	// Test all 8 shrines
	for i := 0; i < TotalShrines; i++ {
		// Test completed flag
		gs.SetShrineQuestCompleted(i, true)
		if !gs.IsShrineQuestCompleted(i) {
			t.Errorf("Shrine %d: SetShrineQuestCompleted(true) -> IsShrineQuestCompleted() = false", i)
		}

		gs.SetShrineQuestCompleted(i, false)
		if gs.IsShrineQuestCompleted(i) {
			t.Errorf("Shrine %d: SetShrineQuestCompleted(false) -> IsShrineQuestCompleted() = true", i)
		}

		// Test active flag
		gs.SetShrineQuestActive(i, true)
		if !gs.IsShrineQuestActive(i) {
			t.Errorf("Shrine %d: SetShrineQuestActive(true) -> IsShrineQuestActive() = false", i)
		}

		gs.SetShrineQuestActive(i, false)
		if gs.IsShrineQuestActive(i) {
			t.Errorf("Shrine %d: SetShrineQuestActive(false) -> IsShrineQuestActive() = true", i)
		}

		// Test ruined flag
		gs.SetShrineRuined(i, true)
		if !gs.IsShrineRuined(i) {
			t.Errorf("Shrine %d: SetShrineRuined(true) -> IsShrineRuined() = false", i)
		}

		gs.SetShrineRuined(i, false)
		if gs.IsShrineRuined(i) {
			t.Errorf("Shrine %d: SetShrineRuined(false) -> IsShrineRuined() = true", i)
		}
	}
}

// TestShrineGettersSetters_InvalidIndex verifies out-of-bounds handling
func TestShrineGettersSetters_InvalidIndex(t *testing.T) {
	t.Parallel()

	gs := createTestGameStateForShrine()

	invalidIndices := []int{-1, -10, TotalShrines, TotalShrines + 10, 100}

	for _, idx := range invalidIndices {
		// Getters should return false for invalid indices
		if gs.IsShrineQuestCompleted(idx) {
			t.Errorf("IsShrineQuestCompleted(%d) should return false for invalid index", idx)
		}

		if gs.IsShrineQuestActive(idx) {
			t.Errorf("IsShrineQuestActive(%d) should return false for invalid index", idx)
		}

		if gs.IsShrineRuined(idx) {
			t.Errorf("IsShrineRuined(%d) should return false for invalid index", idx)
		}

		// Setters should not panic for invalid indices
		gs.SetShrineQuestCompleted(idx, true)
		gs.SetShrineQuestActive(idx, true)
		gs.SetShrineRuined(idx, true)
	}
}

// TestGetShrineQuestState_AllStates verifies correct state for all bit combinations
func TestGetShrineQuestState_AllStates(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name      string
		completed bool
		active    bool
		wantState int
	}{
		{"Not assigned (completed=false, active=false)", false, false, QuestStateNotAssigned},
		{"Not assigned (completed=false, active=true)", false, true, QuestStateNotAssigned},
		{"Ready to complete (completed=true, active=true)", true, true, QuestStateReadyToComplete},
		{"Offering mode (completed=true, active=false)", true, false, QuestStateOffering},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gs := createTestGameStateForShrine()

			// Set up state for shrine 0
			gs.SetShrineQuestCompleted(0, tc.completed)
			gs.SetShrineQuestActive(0, tc.active)

			got := gs.GetShrineQuestState(0)
			if got != tc.wantState {
				t.Errorf("GetShrineQuestState() = %d, want %d", got, tc.wantState)
			}
		})
	}
}

// TestGetShrineQuestState_InvalidIndex verifies invalid index handling
func TestGetShrineQuestState_InvalidIndex(t *testing.T) {
	t.Parallel()

	gs := createTestGameStateForShrine()

	if gs.GetShrineQuestState(-1) != QuestStateNotAssigned {
		t.Error("GetShrineQuestState(-1) should return QuestStateNotAssigned")
	}

	if gs.GetShrineQuestState(TotalShrines) != QuestStateNotAssigned {
		t.Error("GetShrineQuestState(TotalShrines) should return QuestStateNotAssigned")
	}
}

// TestGetFirstActiveQuest verifies finding first active quest
func TestGetFirstActiveQuest(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		activeFlags []bool
		wantIndex   int
	}{
		{"No active quests", []bool{false, false, false, false, false, false, false, false}, -1},
		{"First active", []bool{true, false, false, false, false, false, false, false}, 0},
		{"Third active", []bool{false, false, true, false, false, false, false, false}, 2},
		{"Last active", []bool{false, false, false, false, false, false, false, true}, 7},
		{"Multiple active - returns first", []bool{false, true, false, true, false, false, false, false}, 1},
		{"All active - returns first", []bool{true, true, true, true, true, true, true, true}, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gs := createTestGameStateForShrine()

			for i, active := range tc.activeFlags {
				gs.SetShrineQuestActive(i, active)
			}

			got := gs.GetFirstActiveQuest()
			if got != tc.wantIndex {
				t.Errorf("GetFirstActiveQuest() = %d, want %d", got, tc.wantIndex)
			}
		})
	}
}

// TestAreAllQuestsComplete verifies all-complete check
func TestAreAllQuestsComplete(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		completedFlags []bool
		want           bool
	}{
		{"None complete", []bool{false, false, false, false, false, false, false, false}, false},
		{"Some complete", []bool{true, true, true, false, false, false, false, false}, false},
		{"All but one", []bool{true, true, true, true, true, true, true, false}, false},
		{"All complete", []bool{true, true, true, true, true, true, true, true}, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			gs := createTestGameStateForShrine()

			for i, completed := range tc.completedFlags {
				gs.SetShrineQuestCompleted(i, completed)
			}

			got := gs.AreAllQuestsComplete()
			if got != tc.want {
				t.Errorf("AreAllQuestsComplete() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestShrineStateIndependence verifies shrines are independent of each other
func TestShrineStateIndependence(t *testing.T) {
	t.Parallel()

	gs := createTestGameStateForShrine()

	// Set all flags for shrine 3
	gs.SetShrineQuestCompleted(3, true)
	gs.SetShrineQuestActive(3, true)
	gs.SetShrineRuined(3, true)

	// Verify other shrines are not affected
	for i := 0; i < TotalShrines; i++ {
		if i == 3 {
			continue
		}

		if gs.IsShrineQuestCompleted(i) {
			t.Errorf("Shrine %d IsShrineQuestCompleted should be false (only shrine 3 was set)", i)
		}

		if gs.IsShrineQuestActive(i) {
			t.Errorf("Shrine %d IsShrineQuestActive should be false (only shrine 3 was set)", i)
		}

		if gs.IsShrineRuined(i) {
			t.Errorf("Shrine %d IsShrineRuined should be false (only shrine 3 was set)", i)
		}
	}

	// Verify shrine 3 has all flags set
	if !gs.IsShrineQuestCompleted(3) || !gs.IsShrineQuestActive(3) || !gs.IsShrineRuined(3) {
		t.Error("Shrine 3 should have all flags set")
	}
}

// TestQuestStateMachine_FullCycle tests the complete quest lifecycle
func TestQuestStateMachine_FullCycle(t *testing.T) {
	t.Parallel()

	gs := createTestGameStateForShrine()
	shrineIdx := 0 // Test with Honesty shrine

	// Step 1: Initial state - not assigned
	if gs.GetShrineQuestState(shrineIdx) != QuestStateNotAssigned {
		t.Error("Initial state should be QuestStateNotAssigned")
	}

	// Step 2: Visit shrine - quest assigned (active becomes true)
	gs.SetShrineQuestActive(shrineIdx, true)
	// Still not assigned because completed is false
	if gs.GetShrineQuestState(shrineIdx) != QuestStateNotAssigned {
		t.Error("After assigning quest, state should still be QuestStateNotAssigned (completed=false)")
	}

	// Step 3: Visit Codex - quest answered (completed becomes true)
	gs.SetShrineQuestCompleted(shrineIdx, true)
	// Now ready to complete (completed=true, active=true)
	if gs.GetShrineQuestState(shrineIdx) != QuestStateReadyToComplete {
		t.Error("After Codex visit, state should be QuestStateReadyToComplete")
	}

	// Step 4: Visit shrine to complete - active cleared
	gs.SetShrineQuestActive(shrineIdx, false)
	// Now in offering mode (completed=true, active=false)
	if gs.GetShrineQuestState(shrineIdx) != QuestStateOffering {
		t.Error("After completing quest, state should be QuestStateOffering")
	}
}

// TestShrineRuinedState_BlocksEntry verifies ruined shrines are properly detected
func TestShrineRuinedState_BlocksEntry(t *testing.T) {
	t.Parallel()

	gs := &GameState{SystemCallbacks: journalTestCallbacks()}

	// Initially no shrines are ruined
	for i := 0; i < TotalShrines; i++ {
		if gs.IsShrineRuined(i) {
			t.Errorf("Shrine %d should not be ruined initially", i)
		}
	}

	// Mark shrine 0 (Honesty) as ruined
	gs.SetShrineRuined(0, true)

	if !gs.IsShrineRuined(0) {
		t.Error("Shrine 0 should be ruined after SetShrineRuined(0, true)")
	}

	// Other shrines should not be affected
	for i := 1; i < TotalShrines; i++ {
		if gs.IsShrineRuined(i) {
			t.Errorf("Shrine %d should not be ruined", i)
		}
	}

	// Restore the shrine
	gs.SetShrineRuined(0, false)

	if gs.IsShrineRuined(0) {
		t.Error("Shrine 0 should not be ruined after SetShrineRuined(0, false)")
	}
}

// TestShrinePositionLookup_AllShrines verifies all shrine positions can be found
func TestShrinePositionLookup_AllShrines(t *testing.T) {
	t.Parallel()

	for i := 0; i < TotalShrines; i++ {
		shrine := GetShrineData(i)

		// Skip Spirituality (index 6) which has special coordinates 0,0
		if i == 6 {
			continue
		}

		idx := GetShrineByPosition(shrine.WorldX, shrine.WorldY)
		if idx != i {
			t.Errorf("GetShrineByPosition(%d, %d) = %d, want %d (%s)",
				shrine.WorldX, shrine.WorldY, idx, i, shrine.VirtueName)
		}
	}
}

// TestShrinePositionLookup_InvalidPosition returns -1 for invalid positions
func TestShrinePositionLookup_InvalidPosition(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		x, y byte
	}{
		{0, 0}, // Spirituality position (special case)
		{100, 100},
		{255, 255},
		{1, 1},
	}

	for _, tc := range testCases {
		idx := GetShrineByPosition(tc.x, tc.y)
		// Should return -1 for invalid/non-shrine positions
		// (except 0,0 which is Spirituality but accessed via moongate)
		if tc.x == 0 && tc.y == 0 {
			// Spirituality is at 0,0 but returns index 6
			if idx != 6 {
				t.Errorf("GetShrineByPosition(0, 0) = %d, want 6 (Spirituality)", idx)
			}
		} else if idx != -1 {
			t.Errorf("GetShrineByPosition(%d, %d) = %d, want -1", tc.x, tc.y, idx)
		}
	}
}
