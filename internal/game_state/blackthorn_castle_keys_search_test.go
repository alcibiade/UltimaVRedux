package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestBlackthornCastleBasementBrazierSpawnsNineKeys(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.PalaceOfBlackthorn).
		WithFloor(-1).
		WithPlayerAt(8, 7).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.Provisions.Keys.Set(0)

	result := state.SearchManager.ExecuteSearch(references.Up)

	require.Equal(t, SearchResultFoundItem, result.Result)
	require.Equal(t, []references.ItemAndQuantity{{Item: references.Key, Quantity: 9}}, result.Items)
	assert.Equal(t, "In the brazier\nthou dost find\na ring of keys!", result.Message)
	item := state.ItemStacksMap.Peek(&references.Position{X: 8, Y: 6})
	require.NotNil(t, item)
	assert.Equal(t, references.Key, item.Item)
	assert.Equal(t, uint16(9), item.Quantity)
	foodBefore := state.PartyState.Inventory.Provisions.Food.Get()
	require.True(t, state.ActionGetSmallMap(references.Up))
	assert.Equal(t, uint16(9), state.PartyState.Inventory.Provisions.Keys.Get())
	assert.Equal(t, foodBefore, state.PartyState.Inventory.Provisions.Food.Get())

	assert.Equal(t, SearchResultNotFound, state.SearchManager.ExecuteSearch(references.Up).Result)
	state.PartyState.Inventory.Provisions.Keys.Set(0)
	assert.Equal(t, SearchResultFoundItem, state.SearchManager.ExecuteSearch(references.Up).Result)
}

func TestBlackthornCastleKeysRequireEmptyDynamicObjectSlot(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.PalaceOfBlackthorn).
		WithFloor(-1).
		WithPlayerAt(8, 7).
		WithSystemCallbacks().
		Build()
	state.PartyState.Inventory.Provisions.Keys.Set(0)
	position := references.Position{X: 8, Y: 6}
	occupant := map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle, position, -1)
	require.True(t, state.CurrentNPCAIController.GetNpcs().AddVehicle(*occupant))

	result := state.SearchManager.ExecuteSearch(references.Up)

	assert.Equal(t, SearchResultNotFound, result.Result)
	assert.False(t, state.ItemStacksMap.HasItemStackAtPosition(&position))
}
