package game_state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// TestActionLookLargeMap_SignDetection tests that looking at a sign on the large map
// correctly triggers the sign dialog instead of showing "Thou dost see *".
func TestActionLookLargeMap_SignDetection(t *testing.T) {
	// Get actual sign positions from SIGNS.DAT for BritanniaUnderworld (Location 0)
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(95, 149). // One tile south of sign at (95, 148)
		WithSystemCallbacks().
		Build()

	// Verify we're on the large map
	mapType := gs.MapState.PlayerLocation.Location.GetMapType()
	if mapType != references.LargeMapType {
		t.Fatalf("Expected LargeMapType, got %v", mapType)
	}

	t.Logf("Player position: (%d, %d)", gs.MapState.PlayerLocation.Position.X, gs.MapState.PlayerLocation.Position.Y)
	t.Logf("Location: %v", gs.MapState.PlayerLocation.Location)

	// Get the tile we're about to look at (north of player)
	lookPos := references.Position{X: 95, Y: 148}
	topTile := gs.GetLayeredMapByCurrentLocation().GetTopTile(&lookPos)
	if topTile != nil {
		t.Logf("Tile at look position: index=%d (0x%x), IsSign=%v",
			topTile.Index, topTile.Index, topTile.Index.IsSign())
	} else {
		t.Log("No tile at look position")
	}

	// Look north at the sign
	result := gs.ActionLookLargeMap(references.Up)
	if !result {
		t.Error("ActionLookLargeMap returned false, expected true")
	}

	// Check if ShowSignDialog was called (sign path) vs ShowActionFeedback (non-sign path)
	if mockCallbacks.SignDialogShown {
		t.Log("SUCCESS: ShowSignDialog was called (sign detected correctly)")

		// Check that a sign record was passed
		if mockCallbacks.SignDialogSign != nil {
			t.Logf("Sign record has %d bytes of raw text", len(mockCallbacks.SignDialogSign.RawText))

			// Count newlines in the sign
			newlineCount := 0
			for _, b := range mockCallbacks.SignDialogSign.RawText {
				if b == 0x8A {
					newlineCount++
				}
			}
			t.Logf("Sign contains %d newlines (0x8A), expected %d lines", newlineCount, newlineCount+1)

			if newlineCount == 0 {
				t.Log("Note: This sign has no internal newlines")
			}
		} else {
			t.Log("ShowSignDialog was called with nil sign record")
		}
	} else {
		t.Error("FAILURE: ShowSignDialog was NOT called - sign detection failed")
		// Check what messages were generated
		for i, msg := range mockCallbacks.Messages {
			t.Logf("Message %d: %s", i, msg)
		}
	}
}

// TestActionLookLargeMap_MultiLineSign tests that multi-line sign rendering works correctly.
func TestActionLookLargeMap_MultiLineSign(t *testing.T) {
	// Use a sign known to have newlines
	// From our earlier test: Sign at Location 0, pos=(233,242) has 8 newlines
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(233, 243). // One tile south of sign
		WithSystemCallbacks().
		Build()

	t.Logf("Testing multi-line sign detection")

	// Get the tile we're about to look at
	lookPos := references.Position{X: 233, Y: 242}
	topTile := gs.GetLayeredMapByCurrentLocation().GetTopTile(&lookPos)
	if topTile != nil {
		t.Logf("Tile at look position (233,242): index=%d (0x%x), IsSign=%v",
			topTile.Index, topTile.Index, topTile.Index.IsSign())
	}

	// Look north at the sign
	gs.ActionLookLargeMap(references.Up)

	if !mockCallbacks.SignDialogShown {
		t.Error("ShowSignDialog was not called")
		for i, msg := range mockCallbacks.Messages {
			t.Logf("Message %d: %s", i, msg)
		}
		return
	}

	if mockCallbacks.SignDialogSign == nil {
		t.Error("Sign record is nil")
		return
	}

	// Count newlines
	newlineCount := 0
	for _, b := range mockCallbacks.SignDialogSign.RawText {
		if b == 0x8A {
			newlineCount++
		}
	}

	t.Logf("Sign raw text: %d bytes, %d newlines", len(mockCallbacks.SignDialogSign.RawText), newlineCount)

	if newlineCount == 0 {
		t.Error("Expected sign with newlines, but found none")
	} else {
		t.Logf("SUCCESS: Multi-line sign detected with %d lines", newlineCount+1)
	}
}

func TestActionLookLargeMapAllSignsUseReadableTerrainInTheirWorld(t *testing.T) {
	for _, test := range []struct {
		name  string
		floor references.FloorNumber
	}{
		{name: "overworld", floor: references.FloorNumber(references.OVERWORLD)},
		{name: "underworld", floor: references.FloorNumber(references.UNDERWORLD)},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, _ := NewIntegrationTestBuilder(t).
				WithLocation(references.BritanniaUnderworld).
				WithFloor(test.floor).
				WithPlayerAt(0, 0).
				WithSystemCallbacks().
				Build()

			checked := 0
			for _, sign := range state.GameReferences.SignReferences.GetSignsAtLocation(references.BritanniaUnderworld) {
				if sign.Floor != int(test.floor) {
					continue
				}
				checked++
				position := references.Position{X: references.Coordinate(sign.X), Y: references.Coordinate(sign.Y)}
				tile := state.GetLayeredMapByCurrentLocation().GetTopTile(&position)
				require.NotNil(t, tile, "sign at (%d,%d) has no terrain", sign.X, sign.Y)
				require.Truef(t, tile.Index.IsSign(),
					"sign at (%d,%d) uses unreadable tile %d (0x%x)", sign.X, sign.Y, tile.Index, tile.Index)
			}
			require.Positive(t, checked)
		})
	}
}
