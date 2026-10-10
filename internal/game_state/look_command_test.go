package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestLookCommandAtLadder tests the Look command specifically at the ladder at 3:42pm
func TestLookCommandAtLadder(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(2, 26). // Adjacent to ladder so we can look at it
		WithSystemCallbacks().
		Build()
	t.Run("LookAtLadderAt3_42PM", func(t *testing.T) {
		// Set time to 3:42 PM
		gs.DateTime.Hour = byte(15)
		gs.DateTime.Minute = byte(42)

		ladderPos := references.Position{X: 1, Y: 26}

		t.Logf("Time: %d:%02d", gs.DateTime.Hour, gs.DateTime.Minute)
		t.Logf("Player at: (%d, %d)", gs.MapState.PlayerLocation.Position.X, gs.MapState.PlayerLocation.Position.Y)
		t.Logf("Looking at: (%d, %d)", ladderPos.X, ladderPos.Y)

		// Check what GetTopTile returns for the ladder position
		currentMap := gs.GetLayeredMapByCurrentLocation()
		topTile := currentMap.GetTopTile(&ladderPos)
		if topTile != nil {
			t.Logf("GetTopTile returns: %s (index=%d)", topTile.Name, topTile.Index)
		} else {
			t.Logf("GetTopTile returns: nil")
		}

		// Check what GetDisplaySpriteAtPosition returns
		displaySprite := gs.GetDisplaySpriteAtPosition(ladderPos)
		t.Logf("GetDisplaySpriteAtPosition returns: index=%d", displaySprite)

		// Check look description
		lookDescription := gs.GameReferences.LookReferences.GetTileLookDescription(topTile.Index)
		t.Logf("Look description would be: '%s'", lookDescription)

		// Actually perform the Look command
		mockCallbacks.Reset()
		success := gs.ActionLookSmallMap(references.Left) // Look west at the ladder

		t.Logf("Look command result: %t", success)
		if len(mockCallbacks.Messages) > 0 {
			t.Logf("Look message: '%s'", mockCallbacks.Messages[0])

			if mockCallbacks.Messages[0] == "Thou dost see a guard" {
				t.Logf("*** CONFIRMED: Look sees 'a guard' at ladder! ***")
			}
		} else {
			t.Logf("No messages from Look command")
		}

		// Check if there's an NPC actually at that position
		if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
			npcs := gs.CurrentNPCAIController.GetNpcs()
			if mapUnit := npcs.GetMapUnitAtPositionOrNil(ladderPos); mapUnit != nil {
				t.Logf("NPC found at ladder position!")
			} else {
				t.Logf("No NPC found at ladder position")
			}
		}

		// Also check different times to see when guard appears/disappears
		t.Logf("\n=== Checking different times ===")
		for hour := 12; hour <= 18; hour++ {
			testTime := gs.DateTime
			testTime.Hour = byte(hour)
			testTime.Minute = byte(0)

			// Simulate time change and NPC schedule update
			gs.DateTime = testTime
			if gs.CurrentNPCAIController != nil {
				// Force schedule update
				gs.CurrentNPCAIController.PopulateMapFirstLoad()
			}

			// Check display sprite
			displaySprite := gs.GetDisplaySpriteAtPosition(ladderPos)
			lookDescription := gs.GameReferences.LookReferences.GetTileLookDescription(displaySprite)
			t.Logf("At %02d:00 - display sprite=%d, description='%s'", hour, displaySprite, lookDescription)
		}
	})
}
