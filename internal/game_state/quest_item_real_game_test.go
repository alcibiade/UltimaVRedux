package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

func TestQuestItemVisibility_RealGame_Stonegate(t *testing.T) {
	// Test the REAL game scenario - check if Sceptre actually exists at Stonegate
	// and if our conditional visibility works with the actual game data

	sceptrePos := &references.Position{X: 15, Y: 15}
	playerSafePos := &references.Position{X: 15, Y: 14} // One tile north of sceptre, safe from trap

	t.Run("Real_Game_Without_Sceptre_In_Inventory", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Stonegate).
			WithPlayerAt(int(playerSafePos.X), int(playerSafePos.Y)).
			WithSystemCallbacks().
			Build()

		// DO NOT manually place anything - test the real game data
		// Verify Sceptre is NOT in inventory
		assert.Equal(t, uint16(0), gs.PartyState.Inventory.QuestItems.Get(references.Sceptre),
			"Sceptre should not be in inventory initially")
		assert.False(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre),
			"HasSome should return false for Sceptre initially")

		// Check what's actually at the Sceptre position in the real game data
		actualTile := gs.GetEffectiveTile(sceptrePos)

		t.Logf("REAL GAME: Sceptre position (%d,%d) shows tile index: %d", sceptrePos.X, sceptrePos.Y, actualTile)
		t.Logf("REAL GAME: Expected Sceptre sprite index: %d", indexes.Sceptre)
		t.Logf("REAL GAME: Sceptre in inventory: %t", gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre))

		// Check if there's actually a Sceptre in the real game data
		if int(actualTile) == int(indexes.Sceptre) {
			t.Logf("✅ REAL GAME: Sceptre EXISTS at Stonegate when not in inventory")
		} else {
			t.Logf("❌ REAL GAME: Sceptre NOT FOUND at Stonegate - actual tile: %d, expected: %d", actualTile, indexes.Sceptre)
			t.Logf("This means the Sceptre is not placed at (15,15) in the real game data")
		}
	})

	t.Run("Real_Game_With_Sceptre_In_Inventory", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Stonegate).
			WithPlayerAt(int(playerSafePos.X), int(playerSafePos.Y)).
			WithSystemCallbacks().
			Build()

		// Add Sceptre to inventory first
		sceptreItem := &references.ItemAndQuantity{
			Item:     references.Sceptre,
			Quantity: 1,
		}
		err := gs.PartyState.Inventory.PutItemInInventory(sceptreItem)
		assert.NoError(t, err, "Should be able to add Sceptre to inventory")

		// Verify Sceptre is in inventory
		assert.Equal(t, uint16(1), gs.PartyState.Inventory.QuestItems.Get(references.Sceptre),
			"Sceptre should be in inventory")
		assert.True(t, gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre),
			"HasSome should return true for Sceptre")

		// Check what tile is at the Sceptre position with Sceptre in inventory
		actualTile := gs.GetEffectiveTile(sceptrePos)

		t.Logf("REAL GAME WITH INVENTORY: Sceptre position (%d,%d) shows tile index: %d", sceptrePos.X, sceptrePos.Y, actualTile)
		t.Logf("REAL GAME WITH INVENTORY: Expected hidden result (if Sceptre was there): %d", indexes.BrickFloor)
		t.Logf("REAL GAME WITH INVENTORY: Sceptre in inventory: %t", gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre))

		// Test our conditional visibility logic
		// The key question: Does having the Sceptre in inventory change what we see?
		if int(actualTile) == int(indexes.BrickFloor) {
			t.Logf("✅ CONDITIONAL VISIBILITY: Sceptre is being hidden because it's in inventory")
		} else {
			t.Logf("❌ CONDITIONAL VISIBILITY: Sceptre hiding NOT WORKING - still shows tile: %d", actualTile)
		}
	})

	t.Run("Debug_Explore_Stonegate_Area", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Stonegate).
			WithPlayerAt(int(playerSafePos.X), int(playerSafePos.Y)).
			WithSystemCallbacks().
			Build()

		// Explore a small area around the expected Sceptre position to see what's actually there
		t.Logf("=== EXPLORING STONEGATE AREA ===")
		for y := 14; y <= 16; y++ {
			for x := 14; x <= 16; x++ {
				pos := &references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
				tile := gs.GetEffectiveTile(pos)
				t.Logf("Position (%d,%d): tile=%d", x, y, tile)

				// Check if this could be the Sceptre
				if int(tile) == int(indexes.Sceptre) {
					t.Logf("🎯 FOUND SCEPTRE at position (%d,%d)!", x, y)
				}
			}
		}
	})
}
