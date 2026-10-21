// Integration tests for Look command sign detection with real game data.
// These tests validate that looking at sign tiles triggers the ShowSignDialog callback
// and correctly looks up sign data from SignReferences.
package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestLookAtSign_Detection tests that looking at sign tiles triggers ShowSignDialog callback
func TestLookAtSign_Detection(t *testing.T) {
	t.Run("LookAtSignTile_TriggersCallback", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		// Search for a sign tile in Britain to position the player next to it
		layeredMap := gs.GetLayeredMapByCurrentLocation()
		if layeredMap == nil {
			t.Skip("Map not loaded")
		}

		signFound := false
		var signPos references.Position
		var adjacentPos references.Position

		// Look for any sign tile in the map
		for y := 1; y < 63; y++ {
			for x := 1; x < 63; x++ {
				pos := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
				tile := layeredMap.GetTopTile(&pos)
				if tile == nil {
					continue
				}

				if tile.Index.IsSign() {
					signFound = true
					signPos = pos
					// Position player one tile above the sign
					adjacentPos = references.Position{X: references.Coordinate(x), Y: references.Coordinate(y - 1)}
					t.Logf("Found sign (sprite %d) at position (%d, %d)", tile.Index, x, y)
					break
				}
			}
			if signFound {
				break
			}
		}

		if !signFound {
			t.Logf("No sign tiles found in Britain map - testing with positioned approach")
			t.Skip("No sign found for testing")
		}

		// Move player to adjacent position
		gs.MapState.PlayerLocation.Position = adjacentPos
		mockCallbacks.Reset()

		// Look down at the sign
		result := gs.ActionLookSmallMap(references.Down)

		// Validate command succeeded
		assert.True(t, result, "Look command should succeed")

		// Validate ShowSignDialog callback was invoked
		assert.True(t, mockCallbacks.SignDialogShown,
			"ShowSignDialog callback should have been called for sign at (%d, %d)", signPos.X, signPos.Y)

		// Validate message output still includes look description
		assert.True(t, mockCallbacks.MessagesContain("Thou dost see"),
			"Should still output standard look message")

		// Validate time advancement
		assert.GreaterOrEqual(t, len(mockCallbacks.TimeAdvanced()), 1, "Should advance time")
	})
}

// TestLookAtSign_SignLookup tests that sign data is correctly looked up from SignReferences
func TestLookAtSign_SignLookup(t *testing.T) {
	t.Run("SignDataLookup_ReturnsCorrectSign", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		// Check if SignReferences is loaded
		if gs.GameReferences.SignReferences == nil {
			t.Skip("SignReferences not loaded")
		}

		layeredMap := gs.GetLayeredMapByCurrentLocation()
		if layeredMap == nil {
			t.Skip("Map not loaded")
		}

		// Find a sign that has data in SignReferences
		signWithDataFound := false
		var signPos references.Position
		var adjacentPos references.Position
		var expectedSign *references.SignRecord

		for y := 1; y < 63; y++ {
			for x := 1; x < 63; x++ {
				pos := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
				tile := layeredMap.GetTopTile(&pos)
				if tile == nil {
					continue
				}

				if tile.Index.IsSign() {
					// Check if this sign has data
					sign := gs.GameReferences.SignReferences.GetSignAt(
						references.Britain,
						int(gs.MapState.PlayerLocation.Floor),
						x, y,
					)
					if sign != nil {
						signWithDataFound = true
						signPos = pos
						expectedSign = sign
						adjacentPos = references.Position{X: references.Coordinate(x), Y: references.Coordinate(y - 1)}
						t.Logf("Found sign with data at (%d, %d)", x, y)
						break
					}
				}
			}
			if signWithDataFound {
				break
			}
		}

		if !signWithDataFound {
			t.Logf("No signs with SignReferences data found in Britain - this is acceptable")
			t.Skip("No sign with data found for testing")
		}

		// Move player to adjacent position
		gs.MapState.PlayerLocation.Position = adjacentPos
		mockCallbacks.Reset()

		// Look down at the sign
		result := gs.ActionLookSmallMap(references.Down)

		// Validate command succeeded
		assert.True(t, result, "Look command should succeed")

		// Validate ShowSignDialog callback was invoked with correct sign
		assert.True(t, mockCallbacks.SignDialogShown, "ShowSignDialog should have been called")
		assert.NotNil(t, mockCallbacks.SignDialogSign, "Sign data should be passed to callback")
		assert.Equal(t, expectedSign, mockCallbacks.SignDialogSign,
			"Callback should receive correct sign for position (%d, %d)", signPos.X, signPos.Y)
	})
}

// TestLookAtSign_DefaultMessage tests that signs without data show default message
func TestLookAtSign_DefaultMessage(t *testing.T) {
	t.Run("SignWithoutData_ShowsDefaultMessage", func(t *testing.T) {
		signPosition := references.Position{X: 10, Y: 10}
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(10, 9).
			WithSystemCallbacks().
			Build()

		layeredMap := gs.GetLayeredMapByCurrentLocation()
		require.Nil(t, gs.GameReferences.SignReferences.GetSignAt(references.Britain, int(references.MainFloor), 10, 10))
		layeredMap.SetTileByLayer(map_state.MapOverrideLayer, &signPosition, indexes.Sign)

		// Look down at the sign
		result := gs.ActionLookSmallMap(references.Down)

		// Validate command succeeded
		assert.True(t, result, "Look command should succeed")

		// Validate ShowSignDialog callback was invoked with nil sign (triggers default message)
		assert.True(t, mockCallbacks.SignDialogShown, "ShowSignDialog should have been called")
		assert.Nil(t, mockCallbacks.SignDialogSign,
			"Sign data should be nil for position (%d, %d) without SignReferences entry", signPosition.X, signPosition.Y)
	})
}

// TestLookAtSign_IsSignMethod tests the IsSign() helper method on SpriteIndex
func TestLookAtSign_IsSignMethod(t *testing.T) {
	testCases := []struct {
		index    indexes.SpriteIndex
		name     string
		expected bool
	}{
		// Original Look rules dispatches only these terrain tiles to inscription lookup.
		// See ref:SIGN-0002.
		{indexes.Sign, "Sign", true},
		{indexes.SmallSign, "SmallSign", true},
		{indexes.SignWarning, "SignWarning", true},
		{indexes.SimpleCross, "SimpleCross", true},
		{indexes.StoneHeadstone, "StoneHeadstone", true},
		// Directional structure, flags, and merchant symbols use ordinary LOOK behavior.
		{indexes.LargeSignBottom, "LargeSignBottom", false},
		{indexes.LargeSignLeft, "LargeSignLeft", false},
		{indexes.LargeSignRight, "LargeSignRight", false},
		{indexes.SignGrass, "SignGrass", false},
		{indexes.SnakeSign1, "SnakeSign1", false},
		{indexes.SnakeSign2, "SnakeSign2", false},
		{indexes.SnakeSign3, "SnakeSign3", false},
		{indexes.SnakeSign4, "SnakeSign4", false},
		{indexes.SignFood, "SignFood", false},
		{indexes.SignCourt, "SignCourt", false},
		{indexes.SignArmourer, "SignArmourer", false},
		{indexes.SignHealer, "SignHealer", false},
		{indexes.SignSmith, "SignSmith", false},
		{indexes.SignGuild, "SignGuild", false},
		{indexes.SignInn, "SignInn", false},
		{indexes.SignMagic, "SignMagic", false},
		{indexes.SignShipwright, "SignShipwright", false},
		{indexes.Grass, "Grass", false},
		{indexes.Tree, "Tree", false},
		{indexes.Fountain1, "Fountain1", false},
		{indexes.Well, "Well", false},
		{indexes.Clock1, "Clock1", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.index.IsSign()
			assert.Equal(t, tc.expected, result,
				"IsSign() for %s (index %d) should return %v", tc.name, tc.index, tc.expected)
		})
	}
}

// TestLookAtSign_NonSignTiles tests that non-sign tiles don't trigger sign dialog
func TestLookAtSign_NonSignTiles(t *testing.T) {
	t.Run("LookAtNonSignTile_NoSignDialog", func(t *testing.T) {
		gs, mockCallbacks := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		layeredMap := gs.GetLayeredMapByCurrentLocation()
		if layeredMap == nil {
			t.Skip("Map not loaded")
		}

		// Find a non-sign tile (grass, stone, etc.)
		nonSignFound := false
		var adjacentPos references.Position

		for y := 1; y < 63; y++ {
			for x := 1; x < 63; x++ {
				pos := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
				tile := layeredMap.GetTopTile(&pos)
				if tile == nil {
					continue
				}

				// Find a tile that is NOT a sign, fountain, well, moongate, or clock
				// (these have special handling)
				if !tile.Index.IsSign() &&
					tile.Index != indexes.Fountain1 &&
					tile.Index != indexes.Fountain2 &&
					tile.Index != indexes.Fountain3 &&
					tile.Index != indexes.Fountain4 &&
					tile.Index != indexes.Well &&
					tile.Index != indexes.Clock1 &&
					tile.Index != indexes.Clock2 &&
					tile.Index != indexes.Telescope &&
					tile.Index != indexes.CrystalBall {
					nonSignFound = true
					adjacentPos = references.Position{X: references.Coordinate(x), Y: references.Coordinate(y - 1)}
					t.Logf("Found non-sign tile (sprite %d) at (%d, %d)", tile.Index, x, y)
					break
				}
			}
			if nonSignFound {
				break
			}
		}

		if !nonSignFound {
			t.Skip("No suitable non-sign tile found")
		}

		// Move player to adjacent position
		gs.MapState.PlayerLocation.Position = adjacentPos
		mockCallbacks.Reset()

		// Look down
		result := gs.ActionLookSmallMap(references.Down)

		// Validate command succeeded
		assert.True(t, result, "Look command should succeed")

		// Validate ShowSignDialog was NOT invoked
		assert.False(t, mockCallbacks.SignDialogShown,
			"ShowSignDialog should NOT be called for non-sign tiles")

		// Validate normal look message was output
		assert.True(t, mockCallbacks.MessagesContain("Thou dost see"),
			"Should output standard look message")
	})
}

// TestLookAtSign_MultipleLocations tests sign lookup across different locations
func TestLookAtSign_MultipleLocations(t *testing.T) {
	locations := []references.Location{
		references.Britain,
		references.Moonglow,
		references.LordBritishsCastle,
	}

	for _, loc := range locations {
		t.Run(loc.String(), func(t *testing.T) {
			gs, mockCallbacks := NewIntegrationTestBuilder(t).
				WithLocation(loc).
				WithPlayerAt(15, 15).
				WithSystemCallbacks().
				Build()

			layeredMap := gs.GetLayeredMapByCurrentLocation()
			if layeredMap == nil {
				t.Skip("Map not loaded")
			}

			// Count signs found in this location
			signCount := 0
			for y := 0; y < 64; y++ {
				for x := 0; x < 64; x++ {
					pos := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
					tile := layeredMap.GetTopTile(&pos)
					if tile != nil && tile.Index.IsSign() {
						signCount++
					}
				}
			}

			t.Logf("Found %d sign tiles in %s", signCount, loc.String())
			_ = mockCallbacks // Used for setup, may test specific signs in future
		})
	}
}
