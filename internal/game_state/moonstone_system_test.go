package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
)

// Simple mock message functions for testing
func mockAddRowStr(message string)             {}
func mockAppendToCurrentRowStr(message string) {}
func mockShowCommandPrompt(command string)     {}

// testSearchForMoonstone is a test helper that checks if there's a buried moonstone at given position
// (moved from moonstone_system.go to keep test-only code in test files)
func testSearchForMoonstone(gs *GameState, x, y uint8) bool {
	if gs.moonstoneSystem == nil {
		gs.moonstoneSystem = NewMoonstoneSystem()
	}

	moonstoneIndex := gs.moonstoneSystem.GetMoonstoneAtPosition(x, y)
	if moonstoneIndex >= 0 {
		// Mark moonstone as found
		gs.moonstoneSystem.MarkMoonstoneFoundAtPosition(moonstoneIndex)

		// Trigger moonstone discovery message via SystemCallbacks
		gs.SystemCallbacks.Message.AddRowStr("a strange rock!")

		return true
	}

	return false
}

func TestMoonstoneSystemInitialization(t *testing.T) {
	// Create a game state and verify moonstone system is initialized
	gameConfig := &config.UltimaVConfiguration{}
	gameReferences := &references.GameReferences{}

	gs := initBlankGameState(gameReferences, gameConfig, 11, 11)

	// Initialize DateTime for testing
	gs.DateTime.Hour = 12 // Daytime

	// Verify moonstone system is not nil
	assert.NotNil(t, gs.moonstoneSystem, "Moonstone system should be initialized")

	// Verify it has the expected 8 moonstones
	moonstones := gs.GetMoonstones()
	assert.Equal(t, 8, len(moonstones), "Should have 8 moonstones")

	// Test that GetActiveMoongatePositions returns empty slice during daytime
	positions := gs.GetActiveMoongatePositions()
	assert.NotNil(t, positions, "Should return empty slice, not nil")
	assert.Equal(t, 0, len(positions), "Should have no active moongates during daytime")
}

func TestMoonstoneSearchAndMoongateVisibility(t *testing.T) {
	gameConfig := &config.UltimaVConfiguration{}
	gameReferences := &references.GameReferences{}

	gs := initBlankGameState(gameReferences, gameConfig, 11, 11)

	// Initialize DateTime for testing
	gs.DateTime.Hour = 12 // Daytime

	// Add SystemCallbacks for testing
	gs.SystemCallbacks = &SystemCallbacks{
		Message: MessageCallbacks{
			AddRowStr:             mockAddRowStr,
			AppendToCurrentRowStr: mockAppendToCurrentRowStr,
			ShowCommandPrompt:     mockShowCommandPrompt,
		},
	}

	// Get initial moonstone locations
	moonstones := gs.GetMoonstones()
	firstMoonstone := moonstones[0]

	// Initially, moonstone should not be found
	assert.False(t, firstMoonstone.IsFound, "Moonstone should not be found initially")

	// Daytime keeps moongates hidden even while their moonstones are buried.
	positions := gs.GetActiveMoongatePositions()
	assert.Equal(t, 0, len(positions), "No moongates should be active initially")

	// Search for moonstone at its location
	found := testSearchForMoonstone(gs, firstMoonstone.X, firstMoonstone.Y)
	assert.True(t, found, "Should find moonstone at its location")

	// After finding, moonstone should be marked as found and revealed in the world.
	updatedMoonstones := gs.GetMoonstones()
	updatedMoonstone := updatedMoonstones[0]
	assert.True(t, updatedMoonstone.IsFound, "Moonstone should be found after search")
	assert.True(t, updatedMoonstone.IsFoundNotCollected())
	assert.False(t, updatedMoonstone.IsInInventory(), "Moonstone should not be in inventory")

	// Revealing the stone preserves its source gate.
	// Force nighttime for testing (set hour to midnight)
	gs.DateTime.Hour = 0
	gs.initializeMoongateStage()
	positions = gs.GetActiveMoongatePositions()
	assert.Contains(t, positions, references.Position{X: references.Coordinate(firstMoonstone.X), Y: references.Coordinate(firstMoonstone.Y)})
}

func TestMoongateSourceRequiresPlacedMapFloorIdentity(t *testing.T) {
	gameConfig := &config.UltimaVConfiguration{}
	gameReferences := &references.GameReferences{}
	gs := initBlankGameState(gameReferences, gameConfig, 11, 11)
	gs.MapState.PlayerLocation = references.PlayerLocation{
		Location: references.BritanniaUnderworld,
		Floor:    0,
		Position: references.Position{X: 224, Y: 133},
	}
	gs.DateTime.Hour = 20
	gs.initializeMoongateStage()
	assert.True(t, gs.IsPlayerOnActiveMoongate(), "buried source must be eligible")

	gs.moonstoneSystem.MarkMoonstoneFoundAtPosition(0)
	assert.True(t, gs.IsPlayerOnActiveMoongate(), "revealed source remains eligible")
	gs.MapState.PlayerLocation.Location = references.Britain
	assert.False(t, gs.IsPlayerOnActiveMoongate(), "same coordinates on another map must be ineligible")
	gs.MapState.PlayerLocation.Location = references.BritanniaUnderworld
	gs.MapState.PlayerLocation.Floor = 1
	assert.False(t, gs.IsPlayerOnActiveMoongate(), "same coordinates on another floor must be ineligible")
}
