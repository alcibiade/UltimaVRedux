package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

func TestQuestItemPickup_Crown_Success(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Place Crown sprite at position (16, 15) - one tile east of player
	crownPos := &references.Position{X: 16, Y: 15}
	mapLayers := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0)
	mapLayers.SetTileByLayer(map_state.MapLayer, crownPos, indexes.Crown)

	// Verify quest item not in inventory initially
	assert.Equal(t, uint16(0), (*gs.PartyState.Inventory.QuestItems.GetQuantity(references.Crown)).Get())

	// Execute Get command to the east
	success := gs.ActionGetSmallMap(references.Right)

	// Verify success
	assert.True(t, success, "Crown pickup should succeed")

	// Verify quest item added to inventory
	assert.Equal(t, uint16(1), (*gs.PartyState.Inventory.QuestItems.GetQuantity(references.Crown)).Get())

	// Verify special message displayed
	assert.True(t, len(mockCallbacks.Messages) > 0, "Should have messages")
	assert.Equal(t, "The Crown of Lord British!", mockCallbacks.Messages[0])

	// Verify time advanced
	assert.Equal(t, []int{1}, mockCallbacks.TimeAdvanced())

	// Verify tile replaced with floor
	currentTile := gs.GetEffectiveTile(crownPos)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), currentTile)
}

func TestQuestItemPickup_Sceptre_Success(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Place Sceptre sprite at position (15, 14) - one tile north of player
	sceptrePos := &references.Position{X: 15, Y: 14}
	mapLayers := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0)
	mapLayers.SetTileByLayer(map_state.MapLayer, sceptrePos, indexes.Sceptre)

	// Execute Get command to the north
	success := gs.ActionGetSmallMap(references.Up)

	// Verify success
	assert.True(t, success, "Sceptre pickup should succeed")

	// Verify quest item added to inventory
	assert.Equal(t, uint16(1), (*gs.PartyState.Inventory.QuestItems.GetQuantity(references.Sceptre)).Get())

	// Verify special message displayed
	assert.True(t, len(mockCallbacks.Messages) > 0, "Should have messages")
	assert.Equal(t, "The Sceptre of Lord British!", mockCallbacks.Messages[0])
}

func TestQuestItemPickup_Amulet_Success(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Place Amulet sprite at position (14, 15) - one tile west of player
	amuletPos := &references.Position{X: 14, Y: 15}
	mapLayers := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0)
	mapLayers.SetTileByLayer(map_state.MapLayer, amuletPos, indexes.Amulet)

	// Execute Get command to the west
	success := gs.ActionGetSmallMap(references.Left)

	// Verify success
	assert.True(t, success, "Amulet pickup should succeed")

	// Verify quest item added to inventory
	assert.Equal(t, uint16(1), (*gs.PartyState.Inventory.QuestItems.GetQuantity(references.Amulet)).Get())

	// Verify special message displayed
	assert.True(t, len(mockCallbacks.Messages) > 0, "Should have messages")
	assert.Equal(t, "The Amulet of Lord British!", mockCallbacks.Messages[0])
}

func TestQuestItemPickup_AlreadyHave_Failure(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Add Crown to inventory first
	crownItem := &references.ItemAndQuantity{
		Item:     references.Crown,
		Quantity: 1,
	}
	err := gs.PartyState.Inventory.PutItemInInventory(crownItem)
	assert.NoError(t, err)

	// Place Crown sprite at position (16, 15)
	crownPos := &references.Position{X: 16, Y: 15}
	mapLayers := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0)
	mapLayers.SetTileByLayer(map_state.MapLayer, crownPos, indexes.Crown)

	// Execute Get command to the east
	success := gs.ActionGetSmallMap(references.Right)

	// Verify failure
	assert.False(t, success, "Crown pickup should fail when already owned")

	// Verify message displayed
	assert.True(t, len(mockCallbacks.Messages) > 0, "Should have messages")
	assert.Equal(t, "Already have that!", mockCallbacks.Messages[0])

	// Verify tile unchanged (Crown hidden due to quest item conditional visibility)
	currentTile := gs.GetEffectiveTile(crownPos)
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), currentTile)

	// Verify no time advancement
	assert.Empty(t, mockCallbacks.TimeAdvanced())
}

func TestQuestItemPickup_NothingThere_Failure(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	// Execute Get command to the east (where there's no quest item)
	success := gs.ActionGetSmallMap(references.Right)

	// Verify failure
	assert.False(t, success, "Get should fail when nothing there")

	// Verify message displayed
	assert.True(t, len(mockCallbacks.Messages) > 0, "Should have messages")
	assert.Equal(t, "Nothing to get!", mockCallbacks.Messages[0])

	// Verify no quest items added
	assert.Equal(t, uint16(0), (*gs.PartyState.Inventory.QuestItems.GetQuantity(references.Crown)).Get())
	assert.Equal(t, uint16(0), (*gs.PartyState.Inventory.QuestItems.GetQuantity(references.Sceptre)).Get())
	assert.Equal(t, uint16(0), (*gs.PartyState.Inventory.QuestItems.GetQuantity(references.Amulet)).Get())
}

func TestQuestItemPickup_MultipleItems_AllSuccess(t *testing.T) {
	gs, mockCallbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()

	mapLayers := gs.MapState.LayeredMaps.GetLayeredMap(references.SmallMapType, 0)

	// Place all three quest items around the player
	crownPos := &references.Position{X: 16, Y: 15}   // East
	sceptrePos := &references.Position{X: 15, Y: 14} // North
	amuletPos := &references.Position{X: 14, Y: 15}  // West

	mapLayers.SetTileByLayer(map_state.MapLayer, crownPos, indexes.Crown)
	mapLayers.SetTileByLayer(map_state.MapLayer, sceptrePos, indexes.Sceptre)
	mapLayers.SetTileByLayer(map_state.MapLayer, amuletPos, indexes.Amulet)

	// Pick up Crown
	mockCallbacks.Reset()
	success1 := gs.ActionGetSmallMap(references.Right)
	assert.True(t, success1)
	assert.Equal(t, uint16(1), (*gs.PartyState.Inventory.QuestItems.GetQuantity(references.Crown)).Get())
	assert.Equal(t, "The Crown of Lord British!", mockCallbacks.Messages[0])

	// Pick up Sceptre
	mockCallbacks.Reset()
	success2 := gs.ActionGetSmallMap(references.Up)
	assert.True(t, success2)
	assert.Equal(t, uint16(1), (*gs.PartyState.Inventory.QuestItems.GetQuantity(references.Sceptre)).Get())
	assert.Equal(t, "The Sceptre of Lord British!", mockCallbacks.Messages[0])

	// Pick up Amulet
	mockCallbacks.Reset()
	success3 := gs.ActionGetSmallMap(references.Left)
	assert.True(t, success3)
	assert.Equal(t, uint16(1), (*gs.PartyState.Inventory.QuestItems.GetQuantity(references.Amulet)).Get())
	assert.Equal(t, "The Amulet of Lord British!", mockCallbacks.Messages[0])

	// Verify all items in inventory
	assert.Equal(t, uint16(1), (*gs.PartyState.Inventory.QuestItems.GetQuantity(references.Crown)).Get())
	assert.Equal(t, uint16(1), (*gs.PartyState.Inventory.QuestItems.GetQuantity(references.Sceptre)).Get())
	assert.Equal(t, uint16(1), (*gs.PartyState.Inventory.QuestItems.GetQuantity(references.Amulet)).Get())

	// Verify all tiles replaced with floor - use GetEffectiveTile to get what player actually sees
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), gs.GetEffectiveTile(crownPos))
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), gs.GetEffectiveTile(sceptrePos))
	assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), gs.GetEffectiveTile(amuletPos))
}

func TestQuestItemMethods_GetSpriteIndex(t *testing.T) {
	// Test quest item sprite index from JSON data
	invRefs := references.NewInventoryItemsReferences()
	assert.Equal(t, indexes.SpriteIndex(indexes.Crown), invRefs.GetQuestItemSpriteIndex(references.Crown))
	assert.Equal(t, indexes.SpriteIndex(indexes.Sceptre), invRefs.GetQuestItemSpriteIndex(references.Sceptre))
	assert.Equal(t, indexes.SpriteIndex(indexes.Amulet), invRefs.GetQuestItemSpriteIndex(references.Amulet))
}

func TestQuestItemMethods_GetDisplayName(t *testing.T) {
	invRefs := references.NewInventoryItemsReferences()
	assert.Equal(t, "Crown of Lord British", invRefs.GetQuestItemDisplayName(references.Crown))
	assert.Equal(t, "Sceptre of Lord British", invRefs.GetQuestItemDisplayName(references.Sceptre))
	assert.Equal(t, "Amulet of Lord British", invRefs.GetQuestItemDisplayName(references.Amulet))
}

func TestQuestItemMethods_GetPickupMessage(t *testing.T) {
	invRefs := references.NewInventoryItemsReferences()
	assert.Equal(t, "The Crown of Lord British!", invRefs.GetQuestItemPickupMessage(references.Crown))
	assert.Equal(t, "The Sceptre of Lord British!", invRefs.GetQuestItemPickupMessage(references.Sceptre))
	assert.Equal(t, "The Amulet of Lord British!", invRefs.GetQuestItemPickupMessage(references.Amulet))
}
