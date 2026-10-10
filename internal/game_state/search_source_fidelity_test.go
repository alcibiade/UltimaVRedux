package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestSourceSearchRecordsTransferExactDecodedItems(t *testing.T) {
	locations := references.GetAllSearchLocations()
	tests := []struct {
		name  string
		index int
	}{
		{name: "gold provision", index: firstSearchRecordOfClass(t, references.SearchObjectGold)},
		{name: "potion subtype", index: firstSearchRecordOfClass(t, references.SearchObjectPotion)},
		{name: "scroll subtype", index: firstSearchRecordOfClass(t, references.SearchObjectScroll, 12)},
		{name: "equipment identity", index: firstSearchRecordOfClass(t, references.SearchObjectWeapon, 15)},
		{name: "HMS Cape plans", index: 12},
		{name: "castle keys", index: 13},
		{name: "skull keys", index: 14},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			state := searchStateAtLocation(t, locations[test.index])
			location := locations[test.index]
			decoded := references.DecodeSearchLocationItem(location)
			before := searchItemInventoryQuantity(&state.PartyState, decoded.Item)
			position := references.Position{X: references.Coordinate(location.X), Y: references.Coordinate(location.Y)}

			response := state.SearchManager.checkSpecialAreas(&position)
			require.NotNil(t, response)
			assert.Equal(t, SearchResultFoundItem, response.Result)
			placed := state.ItemStacksMap.Peek(&position)
			require.NotNil(t, placed)
			assert.Equal(t, decoded.Item.Type(), placed.Item.Type())
			assert.Equal(t, decoded.Item.ID(), placed.Item.ID())
			assert.Equal(t, uint16(decoded.PickupQuantity), placed.Quantity)

			require.True(t, state.getFromItemStack(&position))
			assert.Equal(t, before+uint16(decoded.PickupQuantity), searchItemInventoryQuantity(&state.PartyState, decoded.Item))
			assert.False(t, state.ItemStacksMap.HasItemStackAtPosition(&position))
		})
	}
}

func TestSharedCoordinateSearchPreservesSourceDiscoveryAndPickupOrder(t *testing.T) {
	locations := references.GetAllSearchLocations()
	first, second := locations[111], locations[112]
	require.Equal(t, first.MapNumber, second.MapNumber)
	require.Equal(t, first.Level, second.Level)
	require.Equal(t, first.X, second.X)
	require.Equal(t, first.Y, second.Y)
	state := searchStateAtLocation(t, first)
	position := references.Position{X: references.Coordinate(first.X), Y: references.Coordinate(first.Y)}

	for _, location := range []references.SearchLocationReference{first, second} {
		response := state.SearchManager.checkSpecialAreas(&position)
		require.NotNil(t, response)
		assert.Contains(t, response.Message, references.DecodeSearchLocationItem(location).DiscoveryText)
	}
	stack, ok := state.ItemStacksMap.Snapshot(&position)
	require.True(t, ok)
	require.Len(t, stack.Items, 2)
	assert.Equal(t, references.DecodeSearchLocationItem(second).Item, stack.Items[0].Item)
	assert.Equal(t, references.DecodeSearchLocationItem(first).Item, stack.Items[1].Item)

	require.True(t, state.getFromItemStack(&position))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Potions.Get(references.Black))
	require.True(t, state.getFromItemStack(&position))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Scrolls.Get(references.ScrollAnTym))
}

func TestFailedSearchPlacementDoesNotCommitFoundState(t *testing.T) {
	location := references.GetAllSearchLocations()[18]
	state := searchStateAtLocation(t, location)
	result := state.SearchManager.authenticLocationManager.FindAtPosition(
		references.Coordinate(location.X), references.Coordinate(location.Y),
		references.FloorNumber(location.Level), references.Location(location.MapNumber),
	)
	require.NotNil(t, result)

	response := state.SearchManager.handleNormalSearchLocation(nil, result)

	require.NotNil(t, response)
	assert.Equal(t, SearchResultNotFound, response.Result)
	assert.False(t, state.SearchManager.authenticLocationManager.isAlreadyFound(location.Index))
}

func TestSurfaceChestSearchUsesSelectedIntelligenceWithoutMutation(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.LordBritishsCastle).
		WithFloor(references.MainFloor).
		WithSystemCallbacks().
		Build()
	chest, direction := positionBesideSurfaceChest(t, state)
	chest.RestoreSurfaceChestNumber(lockedSurfaceChestNumber)
	target := direction.GetNewPositionInDirection(&state.MapState.PlayerLocation.Position)
	require.NotNil(t, target)
	low := state.PartyState.GetCharacter(0)
	high := state.PartyState.GetCharacter(1)
	require.NotNil(t, low)
	require.NotNil(t, high)
	low.Intelligence = 0
	high.Intelligence = 255
	seed := seedWithFirstRollBelow(t, combatTreasureDiceSides)

	state.rng.Reset(seed)
	lowResponse := state.SearchManager.checkChests(target, 0)
	calls := state.GetRNG().Calls()
	highResponse := state.SearchManager.checkChests(target, 1)
	require.NotNil(t, lowResponse)
	require.NotNil(t, highResponse)
	assert.Equal(t, chestUncertainMessage, lowResponse.Message)
	assert.Equal(t, lowResponse.Message, highResponse.Message, "switching inspectors cannot reroll")
	assert.Equal(t, calls, state.GetRNG().Calls())
	assert.Equal(t, references.ChestUncertain, chest.SurfaceChestAssessment())
	number, ok := chest.SurfaceChestNumber()
	require.True(t, ok)
	assert.Equal(t, byte(lockedSurfaceChestNumber), number)
	assert.NotNil(t, state.surfaceChestAt(target))
}

func TestSurfaceDeadMonsterPlaguePoisonsActorAndRemovesCorpse(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Yew).
		WithFloor(references.MainFloor).
		WithSystemCallbacks().
		Build()
	callbacks := NewMockSystemCallbacks(t)
	state.SystemCallbacks = callbacks.ToSystemCallbacks()
	location := references.GetAllSearchLocations()[40]
	state.MapState.PlayerLocation.Location = references.Location(location.MapNumber)
	state.MapState.PlayerLocation.Floor = references.FloorNumber(location.Level)
	position := references.Position{X: references.Coordinate(location.X), Y: references.Coordinate(location.Y)}
	result := &AuthenticSearchResult{Index: location.Index, ObjectClass: location.ObjectClass, Position: position}
	require.True(t, state.addSurfaceCorpse(result))
	actor := state.PartyState.GetCharacter(0)
	actor.SetStatus(party_state.Good)
	state.rng.Reset(seedForSurfaceCorpsePlague(t))

	response := state.SearchManager.checkCorpses(&position, 0)
	require.NotNil(t, response)
	assert.Equal(t, "Thou dost find\nPlague!", response.Message)
	assert.True(t, actor.IsPoisoned())
	assert.Empty(t, state.SurfaceCorpses)
	assert.Contains(t, callbacks.SoundEffectsPlayed, SoundPoison)
}

func TestSurfaceDeadPCIsPassablePersistentAndNotGettable(t *testing.T) {
	location := references.GetAllSearchLocations()[42]
	state := searchStateAtLocation(t, location)
	position := references.Position{X: references.Coordinate(location.X), Y: references.Coordinate(location.Y)}
	result := state.SearchManager.authenticLocationManager.FindAtPosition(
		position.X, position.Y, references.FloorNumber(location.Level), references.Location(location.MapNumber),
	)
	require.NotNil(t, result)
	response := state.SearchManager.handleNormalSearchLocation(&position, result)
	require.NotNil(t, response)
	assert.Contains(t, response.Message, "a rotting body!")
	assert.True(t, state.IsPassable(&position))
	assert.False(t, state.ItemStacksMap.HasItemStackAtPosition(&position))
	state.MapState.PlayerLocation.Position = references.Position{X: position.X - 1, Y: position.Y}
	assert.False(t, state.ActionGetSmallMap(references.Right))
	require.Len(t, state.SurfaceCorpses, 1)
}

func TestSurfaceDeadMonsterLootTransformsCorpseIntoCarriableObject(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Yew).
		WithFloor(references.MainFloor).
		WithSystemCallbacks().
		Build()
	location := references.GetAllSearchLocations()[40]
	position := references.Position{X: references.Coordinate(location.X), Y: references.Coordinate(location.Y)}
	result := &AuthenticSearchResult{Index: location.Index, ObjectClass: location.ObjectClass, Position: position}
	require.True(t, state.addSurfaceCorpse(result))
	state.rng.Reset(seedForSurfaceCorpseLoot(t))

	response := state.SearchManager.checkCorpses(&position, 0)
	require.NotNil(t, response)
	assert.Empty(t, state.SurfaceCorpses)
	item := state.ItemStacksMap.Peek(&position)
	require.NotNil(t, item)
	assert.Contains(t, []references.Item{references.Gold, references.Food}, item.Item)
	assert.GreaterOrEqual(t, item.Quantity, uint16(1))
	assert.LessOrEqual(t, item.Quantity, uint16(combatCorpseQuantityMaximum))
}

func TestCombatCorpseSearchRetainsOriginalOutcomeFamilies(t *testing.T) {
	sequence := func(values ...int) func(int, int) int {
		index := 0

		return func(_, _ int) int {
			value := values[index]
			index++

			return value
		}
	}
	gold := rollCombatCorpseSearch(sequence(0, 1, 2))
	assert.True(t, gold.hasItem)
	assert.Equal(t, references.Gold, gold.item)
	assert.Equal(t, uint16(2), gold.quantity)
	food := rollCombatCorpseSearch(sequence(0, 0, 3))
	assert.Equal(t, references.Food, food.item)
	assert.Equal(t, uint16(3), food.quantity)
	plague := rollCombatCorpseSearch(sequence(1, combatCorpsePlagueResult))
	assert.True(t, plague.poisoned)
	assert.Equal(t, "Plague!", plague.message)
	description := rollCombatCorpseSearch(sequence(1, 0, 3, 3))
	assert.False(t, description.hasItem)
	assert.Equal(t, "a bloody pulp!", description.message)
}

func firstSearchRecordOfClass(t *testing.T, class references.SearchObjectClass, excluded ...int) int {
	t.Helper()
	for _, location := range references.GetAllSearchLocations() {
		if location.ObjectClass != class {
			continue
		}
		skip := false
		for _, index := range excluded {
			skip = skip || location.Index == index
		}
		if !skip {
			return location.Index
		}
	}
	t.Fatalf("no source Search record for class %#x", class)

	return -1
}

func searchStateAtLocation(t *testing.T, location references.SearchLocationReference) *GameState {
	t.Helper()
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Location(location.MapNumber)).
		WithFloor(references.FloorNumber(location.Level)).
		WithPlayerAt(location.X, location.Y).
		WithSystemCallbacks().
		Build()
	state.DateTime.Day = 1
	if location.Index == SpecialItemCastleKeys {
		state.PartyState.Inventory.Provisions.Keys.Set(0)
	}
	for _, earlier := range references.GetAllSearchLocations()[:location.Index] {
		if earlier.MapNumber == location.MapNumber && earlier.Level == location.Level &&
			earlier.X == location.X && earlier.Y == location.Y {
			state.SearchManager.authenticLocationManager.markAsFound(earlier.Index)
		}
	}

	return state
}

func searchItemInventoryQuantity(state *party_state.PartyState, item references.Item) uint16 {
	switch item := item.(type) {
	case references.Provision:
		return state.Inventory.LockerSnapshot().Provisions[item]
	case references.Potion:
		return state.Inventory.Potions.Get(item)
	case references.Scroll:
		return state.Inventory.Scrolls.Get(item)
	case references.Equipment:
		return state.Inventory.Equipment.Get(item)
	case references.QuestItem:
		return state.Inventory.QuestItems.Get(item)
	default:
		panic("unsupported Search inventory identity")
	}
}

func seedWithFirstRollBelow(t *testing.T, sides int) uint64 {
	t.Helper()
	for seed := uint64(1); seed < 1000; seed++ {
		if deterministic_random.New(seed).Intn(sides)+1 < sides {
			return seed
		}
	}
	t.Fatal("no deterministic seed found")

	return 0
}

func seedForSurfaceCorpsePlague(t *testing.T) uint64 {
	t.Helper()
	for seed := uint64(1); seed < 10000; seed++ {
		rng := deterministic_random.New(seed)
		if rng.Intn(combatCorpseLootMaxRoll+1) != 0 && rng.Intn(combatCorpsePlagueMaxRoll+1) == combatCorpsePlagueResult {
			return seed
		}
	}
	t.Fatal("no deterministic plague seed found")

	return 0
}

func seedForSurfaceCorpseLoot(t *testing.T) uint64 {
	t.Helper()
	for seed := uint64(1); seed < 10000; seed++ {
		if deterministic_random.New(seed).Intn(combatCorpseLootMaxRoll+1) == 0 {
			return seed
		}
	}
	t.Fatal("no deterministic corpse-loot seed found")

	return 0
}
