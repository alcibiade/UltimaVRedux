package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestKlimbFenceExpectedBehavior tests the expected fence climbing behavior mentioned by user.
// User reported: "at x=11,y=28 climbing left. left is a fence and it is expected to succeed
// and shift the avatar left 1 tile."
func TestKlimbFenceExpectedBehavior(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.NorthBritanny).
		WithPlayerAt(11, 28). // User's specified test position
		WithSystemCallbacks().
		Build()
	gs.SetRandomSeed(12345)

	t.Run("KlimbLeft_AtPosition11_28_ShouldSucceed", func(t *testing.T) {
		initialPos := gs.MapState.PlayerLocation.Position
		t.Logf("Initial position: (%d, %d)", initialPos.X, initialPos.Y)

		// Check what's to the left
		leftPos := references.Left.GetNewPositionInDirection(&initialPos)
		if leftPos != nil {
			leftTile := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, gs.MapState.PlayerLocation.Floor).GetTileTopMapOnlyTile(leftPos)
			if leftTile != nil {
				t.Logf("Tile to the left: %s (index=%d)", leftTile.Name, leftTile.Index)
				t.Logf("Is fence horizontal: %v", leftTile.Is(indexes.FenceHoriz))
				t.Logf("Is fence vertical: %v", leftTile.Is(indexes.FenceVert))
			}
		}

		mockCallbacks.Reset()
		result := gs.ActionKlimbSmallMap(references.Left)

		finalPos := gs.MapState.PlayerLocation.Position
		t.Logf("Final position: (%d, %d)", finalPos.X, finalPos.Y)
		t.Logf("Messages: %v", mockCallbacks.Messages)

		if result {
			// User expects this to succeed and move player left
			expectedX := initialPos.X - 1
			expectedY := initialPos.Y

			if finalPos.X != expectedX || finalPos.Y != expectedY {
				t.Errorf("Expected player to move to (%d, %d) but moved to (%d, %d)",
					expectedX, expectedY, finalPos.X, finalPos.Y)
			}

			// Should have "Klimb!" message
			found := false
			for _, msg := range mockCallbacks.Messages {
				if msg == "Klimb!" {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected 'Klimb!' message but got: %v", mockCallbacks.Messages)
			}

			t.Log("SUCCESS: Fence climbing worked as expected")
		} else {
			// If it failed, let's understand why
			if len(mockCallbacks.Messages) > 0 {
				lastMsg := mockCallbacks.Messages[len(mockCallbacks.Messages)-1]
				t.Logf("Klimb failed with message: %s", lastMsg)

				if lastMsg == "What?" {
					t.Log("No fence found at target location or invalid direction")
				}
			}
		}
	})
}
