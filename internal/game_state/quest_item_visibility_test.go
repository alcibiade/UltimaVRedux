package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

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
