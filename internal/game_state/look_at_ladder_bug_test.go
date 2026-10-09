package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestLookAtLadderGuardBug tests the specific user report:
// Position player adjacent to ladder, use Look command to look AT the ladder position (1,26)
func TestLookAtLadderGuardBug(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(2, 26). // Position ADJACENT to ladder so we can look AT it
		WithSystemCallbacks().
		Build()
	t.Run("LookAtLadderFrom_East", func(t *testing.T) {
		// Set time to exactly 3:42 PM as user reported
		gs.DateTime.Hour = byte(15)
		gs.DateTime.Minute = byte(42)

		t.Logf("=== Testing Look AT ladder position (1,26) ===")
		t.Logf("Time: %d:%02d", gs.DateTime.Hour, gs.DateTime.Minute)
		t.Logf("Player at: (%d, %d) floor %d",
			gs.MapState.PlayerLocation.Position.X, gs.MapState.PlayerLocation.Position.Y,
			gs.MapState.PlayerLocation.Floor)

		ladderPos := references.Position{X: 1, Y: 26}
		t.Logf("Looking AT: (%d, %d)", ladderPos.X, ladderPos.Y)

		// Use Look command to look WEST at the ladder (player at 2,26 looking at 1,26)
		mockCallbacks.Reset()
		lookSuccess := gs.ActionLookSmallMap(references.Left)

		t.Logf("Look result: %t", lookSuccess)
		if len(mockCallbacks.Messages) > 0 {
			message := mockCallbacks.Messages[0]
			t.Logf("Look message: '%s'", message)

			// Check if we found the user's bug
			if message == "Thou dost see a guard" {
				t.Logf("*** REPRODUCED USER BUG! Look shows guard at ladder! ***")

				// Get more details about what's causing this
				displaySprite := gs.GetDisplaySpriteAtPosition(ladderPos)
				t.Logf("Display sprite at ladder: %d", displaySprite)

				// Check if it's a guard sprite
				if displaySprite >= 368 && displaySprite <= 371 {
					t.Logf("Display sprite IS a guard sprite!")
				}
			} else if message == "Thou dost see a ladder" {
				t.Logf("Look correctly shows ladder (no bug)")
			} else {
				t.Logf("Look shows something else: %s", message)
			}
		} else {
			t.Logf("No messages from Look command")
		}
	})

	// Test from other adjacent positions too
	testPositions := []struct {
		name      string
		playerPos references.Position
		direction references.Direction
	}{
		{"North", references.Position{X: 1, Y: 25}, references.Down}, // North of ladder, look south
		{"South", references.Position{X: 1, Y: 27}, references.Up},   // South of ladder, look north
		{"East", references.Position{X: 2, Y: 26}, references.Left},  // East of ladder, look west
		{"West", references.Position{X: 0, Y: 26}, references.Right}, // West of ladder, look east
	}

	for _, testCase := range testPositions {
		t.Run("LookAtLadderFrom_"+testCase.name, func(t *testing.T) {
			// Position player
			gs.MapState.PlayerLocation.Position = testCase.playerPos

			t.Logf("Player at (%d,%d) looking %v at ladder",
				testCase.playerPos.X, testCase.playerPos.Y, testCase.direction)

			// Look at ladder
			mockCallbacks.Reset()
			_ = gs.ActionLookSmallMap(testCase.direction)

			if len(mockCallbacks.Messages) > 0 {
				message := mockCallbacks.Messages[0]
				t.Logf("Message: '%s'", message)

				if message == "Thou dost see a guard" {
					t.Logf("*** BUG REPRODUCED from %s! ***", testCase.name)
				}
			}
		})
	}
}
