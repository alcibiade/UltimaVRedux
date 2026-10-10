package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

func TestQuestItemVisibility_Sceptre_Stonegate(t *testing.T) {
	// Test the specific Sceptre location at Stonegate x=15,y=15,floor=0
	// Player positioned safely on carpet at x=15,y=14 to avoid trap at x=15,y=16

	sceptrePos := &references.Position{X: 15, Y: 15}
	playerSafePos := &references.Position{X: 15, Y: 14} // One tile north of sceptre, safe from trap

	t.Run("Sceptre_NOT_in_inventory_should_be_visible", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Stonegate).
			WithPlayerAt(int(playerSafePos.X), int(playerSafePos.Y)).
			WithSystemCallbacks().
			Build()

		// Manually place Sceptre at the expected position since it's not there by default
		mapLayers := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0)
		mapLayers.SetTileByLayer(map_state.MapLayer, sceptrePos, indexes.Sceptre)

		// Ensure Sceptre is NOT in inventory
		assert.Equal(t, uint16(0), gs.PartyState.Inventory.QuestItems.Get(references.Sceptre),
			"Sceptre should not be in inventory initially")

		// Check what tile is at the Sceptre position
		actualTile := gs.GetEffectiveTile(sceptrePos)

		t.Logf("Sceptre position (%d,%d) shows tile index: %d", sceptrePos.X, sceptrePos.Y, actualTile)
		t.Logf("Expected Sceptre sprite index: %d", indexes.Sceptre)
		t.Logf("Sceptre in inventory: %t", gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre))

		// When Sceptre is NOT in inventory, it should be visible at Stonegate
		assert.Equal(t, int(actualTile), int(indexes.Sceptre),
			"Sceptre should be visible at Stonegate when not in inventory")
	})

	t.Run("Sceptre_IN_inventory_should_be_hidden", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Stonegate).
			WithPlayerAt(int(playerSafePos.X), int(playerSafePos.Y)).
			WithSystemCallbacks().
			Build()

		// Manually place Sceptre at the expected position since it's not there by default
		mapLayers := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0)
		mapLayers.SetTileByLayer(map_state.MapLayer, sceptrePos, indexes.Sceptre)

		// Add Sceptre to inventory using quest-add functionality
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

		// Check what tile is at the Sceptre position
		actualTile := gs.GetEffectiveTile(sceptrePos)

		t.Logf("Sceptre position (%d,%d) shows tile index: %d", sceptrePos.X, sceptrePos.Y, actualTile)
		t.Logf("Expected BrickFloor sprite index: %d", indexes.BrickFloor)
		t.Logf("Sceptre in inventory: %t", gs.PartyState.Inventory.QuestItems.HasSome(references.Sceptre))

		// When Sceptre IS in inventory, it should be hidden (replaced with BrickFloor)
		assert.Equal(t, int(actualTile), int(indexes.BrickFloor),
			"Sceptre should be hidden (replaced with BrickFloor) when in inventory")
	})
}

func TestQuestItemVisibility_AllQuestItems(t *testing.T) {
	// Test all three quest items for conditional visibility

	t.Run("Crown_conditional_visibility", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain). // Use any location for testing
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		testPos := &references.Position{X: 16, Y: 15}

		// Manually set the tile to Crown for testing
		mapLayers := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0)
		mapLayers.SetTileByLayer(map_state.MapLayer, testPos, indexes.Crown)

		// Test without Crown in inventory
		actualTile := gs.GetEffectiveTile(testPos)
		assert.Equal(t, indexes.SpriteIndex(indexes.Crown), actualTile, "Crown should be visible when not owned")

		// Add Crown to inventory
		crownItem := &references.ItemAndQuantity{Item: references.Crown, Quantity: 1}
		err := gs.PartyState.Inventory.PutItemInInventory(crownItem)
		assert.NoError(t, err)

		// Test with Crown in inventory
		actualTile = gs.GetEffectiveTile(testPos)
		assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), actualTile, "Crown should be hidden when owned")
	})

	t.Run("Amulet_conditional_visibility", func(t *testing.T) {
		gs, _ := NewIntegrationTestBuilder(t).
			WithLocation(references.Britain).
			WithPlayerAt(15, 15).
			WithSystemCallbacks().
			Build()

		testPos := &references.Position{X: 16, Y: 15}

		// Manually set the tile to Amulet for testing
		mapLayers := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0)
		mapLayers.SetTileByLayer(map_state.MapLayer, testPos, indexes.Amulet)

		// Test without Amulet in inventory
		actualTile := gs.GetEffectiveTile(testPos)
		assert.Equal(t, indexes.SpriteIndex(indexes.Amulet), actualTile, "Amulet should be visible when not owned")

		// Add Amulet to inventory
		amuletItem := &references.ItemAndQuantity{Item: references.Amulet, Quantity: 1}
		err := gs.PartyState.Inventory.PutItemInInventory(amuletItem)
		assert.NoError(t, err)

		// Test with Amulet in inventory
		actualTile = gs.GetEffectiveTile(testPos)
		assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), actualTile, "Amulet should be hidden when owned")
	})
}
