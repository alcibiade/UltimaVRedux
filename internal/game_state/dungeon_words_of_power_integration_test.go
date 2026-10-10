package game_state

import (
	"strings"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWordOfPowerSurveyRequiresKnownIdentityAndSuccessfulTarget(t *testing.T) {
	for _, dungeon := range []references.Location{references.Deceit, references.Shame, references.Doom} {
		t.Run(dungeon.String(), func(t *testing.T) {
			state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.BritanniaUnderworld).
				WithFloor(dungeonEntranceFloor(dungeon)).WithRandomSeed(42).WithSystemCallbacks().Build()
			entrance := state.GetDungeonEntranceCoordinates()[dungeon]
			state.MapState.PlayerLocation.Position = entrance.Right()
			state.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &entrance, dungeonOpenEntranceSprite(dungeon))
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
			state.DateTime.Hour = 19
			word := GetWordForDungeon(dungeon)
			require.True(t, state.ProcessWordOfPower(word), "blind valid Yell still changes the seal")
			require.Empty(t, state.journalDungeons(), "matching an internal identity does not disclose it")
			if dungeon == references.Shame || dungeon == references.Doom {
				require.True(t, state.DiscoverWordOfPowerKnowledge(strings.ToLower(dungeon.String()), DialogueKnowledgeSource("Speaker")))
				require.True(t, state.ProcessWordOfPower(word))
				require.Empty(t, state.journalDungeons(), "unassociated word still cannot identify Shame or Doom")
			}
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 0)
			require.True(t, state.ActionLookLargeMap(references.Left))
			siteIndex := journal.DungeonSurfaceSite
			if dungeon == references.Doom {
				siteIndex = journal.DungeonUnderworldSite
			}
			require.False(t, state.journalDungeons()[0].Sites[siteIndex].Survey.Measured)
			state.PartyState.Inventory.QuestItems.Set(references.Sextant, 1)
			state.MapState.PlayerLocation.Position = entrance.Right().Right()
			require.False(t, state.ProcessWordOfPower(word))
			require.False(t, state.journalDungeons()[0].Sites[siteIndex].Survey.Measured, "rejected Yell does not measure")
			state.MapState.PlayerLocation.Position = entrance.Right()
			clock, calls := state.DateTime, state.GetRNG().Calls()
			callbacks.Reset()
			require.True(t, state.ProcessWordOfPower(word))
			entry := state.journalDungeons()[0]
			require.Equal(t, word, entry.Word)
			require.Equal(t, state.IsDungeonOpen(dungeon), entry.Sites[siteIndex].Open)
			require.Equal(t, dungeon != references.Doom, entry.Sites[siteIndex].Survey.Measured)
			if dungeon != references.Doom {
				require.Equal(t, int(entrance.X), entry.Sites[siteIndex].Survey.Location.X)
				require.Equal(t, int(entrance.Y), entry.Sites[siteIndex].Survey.Location.Y)
			}
			require.Equal(t, clock, state.DateTime)
			require.Equal(t, calls, state.GetRNG().Calls())
			require.Empty(t, callbacks.Messages)
		})
	}
}

// TestWordRecognitionWorkflow_Integration tests the complete workflow of word recognition
// with simplified game state to avoid integration builder issues
func TestWordRecognitionWorkflow_Integration(t *testing.T) {
	// Create minimal game state for testing
	gs := &GameState{}

	// Initialize RNG to avoid nil pointer
	gs.SetRandomSeed(12345)

	// Initialize player position near Deceit entrance
	gs.MapState.PlayerLocation.Position = references.Position{X: 240, Y: 73}

	t.Run("Player near Deceit entrance yells FALLAX", func(t *testing.T) {
		// Verify initial state - dungeon should be closed
		assert.False(t, gs.IsDungeonOpen(references.Deceit))

		// Test proximity detection
		dungeon, isNear := gs.IsPlayerNearDungeonEntrance(3)
		assert.True(t, isNear, "Player should be near Deceit entrance")
		assert.Equal(t, references.Deceit, dungeon)

		// Test word recognition
		recognizedDungeon, exists := GetDungeonForWord("FALLAX")
		assert.True(t, exists, "FALLAX should be recognized")
		assert.Equal(t, references.Deceit, recognizedDungeon)

		// Test state toggle
		newState := gs.ToggleDungeon(references.Deceit)
		assert.True(t, newState, "Dungeon should now be open")
		assert.True(t, gs.IsDungeonOpen(references.Deceit))
	})

	t.Run("Player too far from entrance", func(t *testing.T) {
		// Move player far from any entrance
		gs.MapState.PlayerLocation.Position = references.Position{X: 0, Y: 0}

		// Test proximity detection
		dungeon, isNear := gs.IsPlayerNearDungeonEntrance(3)
		assert.False(t, isNear, "Player should not be near any entrance")
		assert.Equal(t, references.Location(0), dungeon)
	})

	t.Run("Case insensitive word recognition", func(t *testing.T) {
		testCases := []string{"fallax", "FALLAX", "FaLlAx", " FALLAX ", "vilis", "VERAMOCOR"}
		expectedDungeons := []references.Location{
			references.Deceit, references.Deceit, references.Deceit,
			references.Deceit, references.Despise, references.Doom,
		}

		for i, word := range testCases {
			dungeon, exists := GetDungeonForWord(word)
			assert.True(t, exists, "Word %s should be recognized", word)
			assert.Equal(t, expectedDungeons[i], dungeon, "Word %s should map to correct dungeon", word)
		}
	})

	t.Run("Invalid word recognition", func(t *testing.T) {
		invalidWords := []string{"INVALID", "", "FALL", "VERAMOCORX", "123"}

		for _, word := range invalidWords {
			dungeon, exists := GetDungeonForWord(word)
			assert.False(t, exists, "Word %s should not be recognized", word)
			assert.Equal(t, references.Location(0), dungeon)
		}
	})
}

// TestProximityDetectionAtAllDungeons_Integration tests proximity detection
// at each dungeon entrance with realistic distances
func TestProximityDetectionAtAllDungeons_Integration(t *testing.T) {
	gs := &GameState{}
	gs.SetRandomSeed(12345)

	dungeonTests := []struct {
		dungeon         references.Location
		word            string
		entrancePos     references.Position
		floor           references.FloorNumber
		proximityRange  int
		testPositions   []references.Position
		expectedResults []bool
	}{
		{
			dungeon:        references.Deceit,
			word:           "FALLAX",
			entrancePos:    references.Position{X: 240, Y: 73},
			floor:          0,
			proximityRange: 3,
			testPositions: []references.Position{
				{X: 240, Y: 73}, // Exact position
				{X: 242, Y: 75}, // Within range (2,2)
				{X: 237, Y: 70}, // Within range (3,3)
				{X: 236, Y: 69}, // Outside range (4,4)
				{X: 240, Y: 65}, // Too far Y (0,8)
			},
			expectedResults: []bool{true, true, true, false, false},
		},
		{
			dungeon:        references.Despise,
			word:           "VILIS",
			entrancePos:    references.Position{X: 91, Y: 67},
			floor:          0,
			proximityRange: 2,
			testPositions: []references.Position{
				{X: 91, Y: 67}, // Exact position
				{X: 93, Y: 69}, // Within range (2,2)
				{X: 94, Y: 70}, // Outside range (3,3)
			},
			expectedResults: []bool{true, true, false},
		},
		{
			dungeon:        references.Doom,
			word:           "VERAMOCOR",
			entrancePos:    references.Position{X: 128, Y: 128},
			floor:          -1,
			proximityRange: 1,
			testPositions: []references.Position{
				{X: 128, Y: 128}, // Exact position
				{X: 129, Y: 129}, // Within range (1,1)
				{X: 130, Y: 130}, // Outside range (2,2)
			},
			expectedResults: []bool{true, true, false},
		},
	}

	for _, tt := range dungeonTests {
		t.Run(tt.dungeon.String()+" proximity detection", func(t *testing.T) {
			gs.MapState.PlayerLocation.Floor = tt.floor
			assert.Equal(t, tt.entrancePos, gs.GetDungeonEntranceCoordinates()[tt.dungeon])
			// Verify word maps to correct dungeon
			recognizedDungeon, exists := GetDungeonForWord(tt.word)
			assert.True(t, exists, "Word should be recognized")
			assert.Equal(t, tt.dungeon, recognizedDungeon)

			// Test proximity at various positions
			for i, testPos := range tt.testPositions {
				gs.MapState.PlayerLocation.Position = testPos
				dungeon, isNear := gs.IsPlayerNearDungeonEntrance(tt.proximityRange)

				if tt.expectedResults[i] {
					assert.True(t, isNear, "Position (%d,%d) should be near %s", testPos.X, testPos.Y, tt.dungeon)
					assert.Equal(t, tt.dungeon, dungeon)
				} else {
					// If not near this dungeon, could be near another or none
					if isNear {
						assert.NotEqual(t, tt.dungeon, dungeon, "Should not detect %s at position (%d,%d)", tt.dungeon, testPos.X, testPos.Y)
					}
				}
			}
		})
	}
}

// TestDungeonStateManagement_Integration tests complete state management
// with save/load simulation
func TestDungeonStateManagement_Integration(t *testing.T) {
	gs := &GameState{}
	gs.SetRandomSeed(12345)

	t.Run("Initial state from save data", func(t *testing.T) {
		// Create mock save data with some dungeons open
		saveData := make([]byte, 0x0400) // Large enough buffer
		saveData[0x032A+0] = 0x80        // Deceit open
		saveData[0x032A+2] = 0x80        // Destard open
		saveData[0x032A+7] = 0x80        // Doom open

		gs.InitializeDungeonStatesFromSave(saveData)

		// Verify states loaded correctly
		assert.True(t, gs.IsDungeonOpen(references.Deceit))
		assert.False(t, gs.IsDungeonOpen(references.Despise))
		assert.True(t, gs.IsDungeonOpen(references.Destard))
		assert.False(t, gs.IsDungeonOpen(references.Wrong))
		assert.False(t, gs.IsDungeonOpen(references.Covetous))
		assert.False(t, gs.IsDungeonOpen(references.Shame))
		assert.False(t, gs.IsDungeonOpen(references.Hythloth))
		assert.True(t, gs.IsDungeonOpen(references.Doom))
	})

	t.Run("State manipulation workflow", func(t *testing.T) {
		// Test all state manipulation methods
		allDungeons := []references.Location{
			references.Deceit, references.Despise, references.Destard, references.Wrong,
			references.Covetous, references.Shame, references.Hythloth, references.Doom,
		}

		// Set all dungeons to specific states
		for i, dungeon := range allDungeons {
			isOpen := i%2 == 0 // Alternate open/closed
			gs.SetDungeonState(dungeon, isOpen)
			assert.Equal(t, isOpen, gs.IsDungeonOpen(dungeon))
		}

		// Toggle all dungeons
		for i, dungeon := range allDungeons {
			expectedNewState := i%2 == 1 // Should be opposite of above
			newState := gs.ToggleDungeon(dungeon)
			assert.Equal(t, expectedNewState, newState)
			assert.Equal(t, expectedNewState, gs.IsDungeonOpen(dungeon))
		}
	})

	t.Run("Word-to-dungeon reverse lookup", func(t *testing.T) {
		allDungeons := []references.Location{
			references.Deceit, references.Despise, references.Destard, references.Wrong,
			references.Covetous, references.Shame, references.Hythloth, references.Doom,
		}

		expectedWords := []string{
			"FALLAX", "VILIS", "INOPIA", "MALUM",
			"AVIDUS", "INFAMA", "IGNAVUS", "VERAMOCOR",
		}

		for i, dungeon := range allDungeons {
			word := GetWordForDungeon(dungeon)
			assert.Equal(t, expectedWords[i], word, "Dungeon %s should map to word %s", dungeon, expectedWords[i])

			// Verify round-trip consistency
			reverseDungeon, exists := GetDungeonForWord(word)
			assert.True(t, exists)
			assert.Equal(t, dungeon, reverseDungeon)
		}

		// Test invalid dungeon
		invalidWord := GetWordForDungeon(references.Britain)
		assert.Equal(t, "", invalidWord, "Non-dungeon location should return empty word")
	})
}

// TestCompleteWorkflow_Integration simulates the complete user workflow
// of approaching a dungeon and yelling a Word of Power
func TestCompleteWorkflow_Integration(t *testing.T) {
	gs := &GameState{}
	gs.SetRandomSeed(54321) // Different seed for variety

	// Initialize player position near Deceit but not too close
	gs.MapState.PlayerLocation.Position = references.Position{X: 237, Y: 70}

	t.Run("Approach dungeon and yell correct word", func(t *testing.T) {
		// Initial state - all dungeons closed
		assert.False(t, gs.IsDungeonOpen(references.Deceit))

		// Player approaches Deceit entrance
		gs.MapState.PlayerLocation.Position = references.Position{X: 241, Y: 74}

		// Check proximity
		dungeon, isNear := gs.IsPlayerNearDungeonEntrance(3)
		assert.True(t, isNear)
		assert.Equal(t, references.Deceit, dungeon)

		// Player yells correct Word of Power
		recognizedDungeon, exists := GetDungeonForWord("FALLAX")
		assert.True(t, exists)
		assert.Equal(t, references.Deceit, recognizedDungeon)
		assert.Equal(t, dungeon, recognizedDungeon, "Nearby dungeon should match word")

		// Toggle dungeon state (simulating successful yell)
		newState := gs.ToggleDungeon(recognizedDungeon)
		assert.True(t, newState, "Dungeon should now be open")

		// Verify state change
		assert.True(t, gs.IsDungeonOpen(references.Deceit))

		// Player yells same word again (should close)
		finalState := gs.ToggleDungeon(recognizedDungeon)
		assert.False(t, finalState, "Dungeon should now be closed")
		assert.False(t, gs.IsDungeonOpen(references.Deceit))
	})

	t.Run("Wrong word at dungeon entrance", func(t *testing.T) {
		// Player at Deceit entrance yells word for different dungeon
		gs.MapState.PlayerLocation.Position = references.Position{X: 240, Y: 73}

		// Check proximity to Deceit
		dungeon, isNear := gs.IsPlayerNearDungeonEntrance(2)
		assert.True(t, isNear)
		assert.Equal(t, references.Deceit, dungeon)

		// Player yells wrong word (for Doom)
		recognizedDungeon, exists := GetDungeonForWord("VERAMOCOR")
		assert.True(t, exists)
		assert.Equal(t, references.Doom, recognizedDungeon)

		// Words don't match location
		assert.NotEqual(t, dungeon, recognizedDungeon, "Word should be for different dungeon")

		// In actual implementation, this would result in "Nothing happens" message
		// For now, we just verify the logic works correctly
	})
}
