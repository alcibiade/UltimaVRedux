package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestGetSearchedHMSCapeUsesPickupMessage(t *testing.T) {
	location := references.GetAllSearchLocations()[12]
	require.Equal(t, references.HMSCape, location.ItemType)
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Location(location.MapNumber)).
		WithFloor(references.FloorNumber(location.Level)).
		WithPlayerAt(location.X-1, location.Y).
		WithSystemCallbacks().
		Build()
	position := references.Position{X: references.Coordinate(location.X), Y: references.Coordinate(location.Y)}
	state.PartyState.Inventory.QuestItems.Set(references.HMSCape, 0)
	require.True(t, state.ActionSearchSmallMap(references.Right))
	item := state.ItemStacksMap.Peek(&position)
	require.NotNil(t, item)
	require.Equal(t, references.HMSCape, item.Item)
	require.Equal(t, uint16(1), item.Quantity)
	callbacks.Reset()

	require.True(t, state.ActionGetSmallMap(references.Right))

	assert.Equal(t, []string{"Plans for the H.M.S. Cape!"}, callbacks.Messages)
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.HMSCape))
	assert.False(t, state.ItemStacksMap.HasItemStackAtPosition(&position))
	assert.Equal(t, "HMSCape", state.GameReferences.InventoryItemReferences.GetReferenceByItem(references.HMSCape).ItemName)
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
	callbacks.Reset()
	assert.False(t, state.ActionGetSmallMap(references.Right))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.QuestItems.Get(references.HMSCape))
	assert.Equal(t, []string{"Nothing to get!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestGetLargeMapQuestStackUsesPickupMessageAndPreservesQuantity(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithFloor(0).
		WithPlayerAt(50, 50).
		WithSystemCallbacks().
		Build()
	position := state.MapState.PlayerLocation.Position.Right()
	state.PartyState.Inventory.QuestItems.Set(references.Carpet, 1)
	require.NoError(t, state.ItemStacksMap.Push(&position, &references.ItemStack{Items: references.ItemStacks{
		{Item: references.Carpet, Quantity: 2},
	}}))
	callbacks.Reset()

	require.True(t, state.ActionGetLargeMap(references.Right))

	assert.Equal(t, []string{"A Magic Carpet!"}, callbacks.Messages)
	assert.Equal(t, uint16(3), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
	assert.False(t, state.ItemStacksMap.HasItemStackAtPosition(&position))
	assert.Equal(t, []int{1}, callbacks.TimeAdvanced())
	callbacks.Reset()
	assert.False(t, state.ActionGetLargeMap(references.Right))
	assert.Equal(t, uint16(3), state.PartyState.Inventory.QuestItems.Get(references.Carpet))
	assert.Empty(t, callbacks.TimeAdvanced())
}
