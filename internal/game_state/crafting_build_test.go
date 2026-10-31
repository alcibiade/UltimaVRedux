package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// yard is the clear square at the centre of every deed map; nothing generated stands within
// four squares of it.
var yard = references.DeedMapCentre

func northOfYard() references.Position {
	return references.Position{X: yard.X, Y: yard.Y - 1}
}

// newBuildFixture enters a grassland deed and stands the party in the clear yard.
func newBuildFixture(t *testing.T) (*GameState, *MockSystemCallbacks, Deed) {
	t.Helper()
	state, callbacks, deed := newGatherFixture(t)
	state.MapState.PlayerLocation.Position = yard

	return state, callbacks, deed
}

// buildNorth gives the party one piece and places it north of the party.
func buildNorth(t *testing.T, state *GameState, piece references.BuildPieceID) {
	t.Helper()
	state.PartyState.Inventory.Pieces.IncrementBy(piece, 1)
	require.True(t, state.BuildDeedPiece(piece, references.Up))
}

// buildAt stands the party south of a square, builds a piece there and returns to the yard.
func buildAt(t *testing.T, state *GameState, piece references.BuildPieceID, position references.Position) {
	t.Helper()
	state.MapState.PlayerLocation.Position = references.Position{X: position.X, Y: position.Y + 1}
	buildNorth(t, state, piece)
	state.MapState.PlayerLocation.Position = yard
}

func liveTile(state *GameState, position references.Position) indexes.SpriteIndex {
	return state.GetLayeredMapByCurrentLocation().SpriteIndexByLayer(map_state.MapLayer, position)
}

func TestBuildPlacesAPieceAndDemolishRestoresTheGround(t *testing.T) {
	state, callbacks, deed := newBuildFixture(t)
	target := northOfYard()
	ground, _ := state.DeedTileAt(deed.ID, target)
	state.PartyState.Inventory.Pieces.Set(references.PieceStoneWall, 2)

	require.True(t, state.BuildDeedPiece(references.PieceStoneWall, references.Up))

	assert.Equal(t, uint16(1), state.PartyState.Inventory.Pieces.Get(references.PieceStoneWall))
	assert.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall), liveTile(state, target))
	placement, built := state.deedByID(deed.ID).Placement(target)
	require.True(t, built)
	assert.Equal(t, DeedPlacement{Piece: references.PieceStoneWall, Under: ground}, placement)
	assert.Contains(t, callbacks.Messages, "Built: Stone wall.")
	assert.Equal(t, references.BuildPlaceMinutes-DefaultSmallMapMinutesPerTurn, gatherMinutes(callbacks),
		"building spends its minutes beyond the ordinary turn")
	assert.False(t, state.IsPassable(&target), "a built wall blocks like any wall")

	callbacks.Reset()
	require.True(t, state.DemolishDeedPiece(references.Up))

	assert.Equal(t, uint16(2), state.PartyState.Inventory.Pieces.Get(references.PieceStoneWall), "the piece returns to stock")
	assert.Equal(t, ground, liveTile(state, target))
	tile, _ := state.DeedTileAt(deed.ID, target)
	assert.Equal(t, ground, tile)
	_, overridden := state.deedByID(deed.ID).TileOverride(target)
	assert.False(t, overridden, "the restored ground is no longer a difference")
	_, built = state.deedByID(deed.ID).Placement(target)
	assert.False(t, built)
	assert.Equal(t, []string{"Demolished: Stone wall."}, callbacks.Messages)
	assert.Equal(t, references.BuildDemolishMinutes-DefaultSmallMapMinutesPerTurn, gatherMinutes(callbacks))
}

func TestBuildRefusalsSpendNothing(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		prepare func(*testing.T, *GameState, Deed) references.BuildPieceID
		message string
	}{
		{"no piece carried", func(_ *testing.T, _ *GameState, _ Deed) references.BuildPieceID {
			return references.PieceStoneWall
		}, "Thou hast no Stone wall!"},
		{"an item stack in the way", func(t *testing.T, state *GameState, _ Deed) references.BuildPieceID {
			target := northOfYard()
			require.NoError(t, state.ItemStacksMap.AddOrMerge(&target, references.ItemAndQuantity{Item: references.Food, Quantity: 1}))

			return references.PieceStoneWall
		}, "Something is in the way!"},
		{"a creature in the way", func(t *testing.T, state *GameState, _ Deed) references.BuildPieceID {
			reference := state.enemyReference(references.MonsterOrc)
			require.NotNil(t, reference)
			orc := map_units.NewEnemyNPC(*reference, references.DynamicSpawnIndex)
			orc.SetPos(northOfYard())
			orc.SetFloor(references.MainFloor)
			orc.SetVisible(true)
			require.True(t, state.CurrentNPCAIController.GetNpcs().AddEnemy(orc))

			return references.PieceStoneWall
		}, "Something is in the way!"},
		{"the decorative ring", func(_ *testing.T, state *GameState, _ Deed) references.BuildPieceID {
			state.MapState.PlayerLocation.Position = references.Position{X: 10, Y: references.DeedInteriorDepth}

			return references.PieceStoneWall
		}, "Thou canst not build there!"},
		{"beyond the map edge", func(_ *testing.T, state *GameState, _ Deed) references.BuildPieceID {
			state.MapState.PlayerLocation.Position = references.Position{X: 10, Y: 0}

			return references.PieceStoneWall
		}, "Thou canst not build there!"},
		{"a resource node", func(_ *testing.T, state *GameState, deed Deed) references.BuildPieceID {
			standSouthOf(state, state.DeedResourceNodes(deed.ID)[0])

			return references.PieceStoneWall
		}, "Thou canst not build there!"},
		{"a used-up resource node", func(_ *testing.T, state *GameState, deed Deed) references.BuildPieceID {
			node := state.DeedResourceNodes(deed.ID)[0]
			depleteNode(state, deed.ID, node)
			standSouthOf(state, node)

			return references.PieceStoneWall
		}, "Thou canst not build there!"},
		{"ground that is not buildable", func(_ *testing.T, state *GameState, deed Deed) references.BuildPieceID {
			state.SetDeedTile(deed.ID, northOfYard(), indexes.Water1)

			return references.PieceStoneWall
		}, "Thou canst not build there!"},
		{"another piece", func(t *testing.T, state *GameState, _ Deed) references.BuildPieceID {
			buildNorth(t, state, references.PieceBarrel)

			return references.PieceStoneWall
		}, "Something is built there!"},
		{"a floor on a floor", func(t *testing.T, state *GameState, _ Deed) references.BuildPieceID {
			buildNorth(t, state, references.PieceWoodFloor)

			return references.PieceStoneFloor
		}, "Something is built there!"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state, callbacks, deed := newBuildFixture(t)
			piece := testCase.prepare(t, state, deed)
			if testCase.name != "no piece carried" {
				state.PartyState.Inventory.Pieces.IncrementBy(piece, 1)
			}
			before := state.PartyState.Inventory.Pieces.Get(piece)
			callbacks.Reset()

			assert.False(t, state.BuildDeedPiece(piece, references.Up))

			assert.Equal(t, []string{testCase.message}, callbacks.Messages)
			assert.Empty(t, callbacks.TimeAdvanced())
			assert.Equal(t, before, state.PartyState.Inventory.Pieces.Get(piece))
		})
	}
}

func TestBuildAndDemolishAreRefusedOffTheDeedMap(t *testing.T) {
	state, callbacks := newDeedFixture(t, true)
	state.PartyState.Inventory.Pieces.Set(references.PieceStoneWall, 1)

	assert.False(t, state.BuildDeedPiece(references.PieceStoneWall, references.Up))
	assert.False(t, state.DemolishDeedPiece(references.Up))

	assert.Equal(t, []string{"Not here!", "Not here!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestDemolishRefusalsSpendNothing(t *testing.T) {
	state, callbacks, _ := newBuildFixture(t)
	assert.False(t, state.DemolishDeedPiece(references.Up), "nothing built there")

	buildNorth(t, state, references.PieceDresser)
	state.PartyState.Inventory.Pieces.Set(references.PieceDresser, party_state.ItemQuantitySmallMax)
	callbacks.Reset()
	assert.False(t, state.DemolishDeedPiece(references.Up), "the stack is full")

	assert.Equal(t, []string{"Thou canst carry no more Dresser!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())
	assert.Equal(t, indexes.SpriteIndex(indexes.Dresser), liveTile(state, northOfYard()))
}

func TestAPieceBuiltOnAFloorUncoversTheFloor(t *testing.T) {
	state, _, deed := newBuildFixture(t)
	target := northOfYard()
	ground, _ := state.DeedTileAt(deed.ID, target)
	buildNorth(t, state, references.PieceWoodFloor)
	buildNorth(t, state, references.PieceChairForward)

	placement, _ := state.deedByID(deed.ID).Placement(target)
	assert.Equal(t, DeedPlacement{Piece: references.PieceChairForward, Under: ground, OnFloor: true, Floor: references.PieceWoodFloor},
		placement)

	require.True(t, state.DemolishDeedPiece(references.Up))
	assert.Equal(t, indexes.SpriteIndex(indexes.WoodFloor1), liveTile(state, target), "the floor stays")
	require.True(t, state.DemolishDeedPiece(references.Up))
	assert.Equal(t, ground, liveTile(state, target))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Pieces.Get(references.PieceWoodFloor))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Pieces.Get(references.PieceChairForward))
}

func TestABuiltDoorOpensAndDemolishingItOpenLeavesNoDoor(t *testing.T) {
	state, _, deed := newBuildFixture(t)
	target := northOfYard()
	ground, _ := state.DeedTileAt(deed.ID, target)
	buildNorth(t, state, references.PieceDoor)
	live := state.GetLayeredMapByCurrentLocation()
	require.False(t, state.IsPassable(&target), "a closed door blocks")

	require.True(t, state.ActionOpenSmallMap(references.Up))
	assert.True(t, state.IsPassable(&target), "an opened built door lets the party through")
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), live.SpriteIndexByLayer(map_state.MapLayer, target),
		"the door itself stays in the base layer")

	require.True(t, state.DemolishDeedPiece(references.Up))
	for range map_state.DefaultOpenDoorTurns + 2 {
		state.MapState.SmallMapProcessTurnDoors(func(references.Position, references.FloorNumber) bool { return false })
	}

	assert.Nil(t, state.MapState.TrackedOpenDoor())
	assert.Equal(t, ground, live.GetTopTile(&target).Index, "no door closes again on the cleared square")
	tile, _ := state.DeedTileAt(deed.ID, target)
	assert.Equal(t, ground, tile)
}

func TestAClosedBuiltDoorSurvivesItsTimerAndARebuild(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	target := northOfYard()
	buildNorth(t, state, references.PieceDoor)
	require.True(t, state.ActionOpenSmallMap(references.Up))
	for range map_state.DefaultOpenDoorTurns + 1 {
		state.MapState.SmallMapProcessTurnDoors(func(references.Position, references.FloorNumber) bool { return false })
	}

	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), state.GetLayeredMapByCurrentLocation().GetTopTile(&target).Index)
	restored := saveAndReload(t, state)
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), restored.GetLayeredMapByCurrentLocation().GetTopTile(&target).Index)
	require.True(t, restored.ActionOpenSmallMap(references.Up), "the reloaded door still opens")
	require.True(t, restored.DemolishDeedPiece(references.Up), "an open door demolishes like any piece")
	assert.True(t, restored.IsPassable(&target))
	require.True(t, state.DemolishDeedPiece(references.Up), "a closed door demolishes like any piece")
	assert.True(t, state.IsPassable(&target))
}

func TestStationsAreWithinReachOnlyWithinOneSquare(t *testing.T) {
	for _, station := range references.CraftStations() {
		piece, ok := references.BuildPieceForStation(station)
		require.True(t, ok)
		state, _, _ := newBuildFixture(t)
		require.False(t, state.craftStationWithinReach(station), "%s: nothing built yet", station.Name())
		buildNorth(t, state, piece.ID)

		for _, testCase := range []struct {
			position references.Position
			reach    bool
		}{
			{yard, true},
			{references.Position{X: yard.X - 1, Y: yard.Y}, true},
			{references.Position{X: yard.X + 1, Y: yard.Y - 2}, true},
			{references.Position{X: yard.X, Y: yard.Y + 1}, false},
			{references.Position{X: yard.X + 2, Y: yard.Y - 1}, false},
		} {
			state.MapState.PlayerLocation.Position = testCase.position
			assert.Equal(t, testCase.reach, state.craftStationWithinReach(station), "%s from %v", station.Name(), testCase.position)
		}
	}
	overworld, _ := newDeedFixture(t, true)
	assert.False(t, overworld.craftStationWithinReach(references.CraftStationWorkbench), "the overworld has no stations")
}

func TestARecipeAwayFromItsStationNamesTheStation(t *testing.T) {
	state, callbacks, _ := newBuildFixture(t)
	state.PartyState.Inventory.Materials.Set(references.Wood, 10)

	require.ErrorIs(t, state.Craft(references.RecipePlanks), ErrCraftNeedsStation)

	assert.Equal(t, []string{"Thou dost need a Sawmill nearby!"}, callbacks.Messages)
	assert.Equal(t, uint16(10), state.PartyState.Inventory.Materials.Get(references.Wood))
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestRefinedMaterialsAreMadeAtTheirStations(t *testing.T) {
	state, callbacks, _ := newBuildFixture(t)
	buildAt(t, state, references.PieceSawmill, references.Position{X: yard.X - 1, Y: yard.Y - 1})
	buildAt(t, state, references.PieceWorkbench, references.Position{X: yard.X, Y: yard.Y - 1})
	buildAt(t, state, references.PieceForge, references.Position{X: yard.X + 1, Y: yard.Y - 1})
	inventory := &state.PartyState.Inventory
	inventory.Materials.Set(references.Wood, 3)
	inventory.Materials.Set(references.Stone, 3)
	inventory.Materials.Set(references.Ore, 2)

	for _, testCase := range []struct {
		recipe   references.CraftRecipeID
		material references.Material
		made     uint16
	}{
		{references.RecipePlanks, references.Planks, 4},
		{references.RecipeCutStone, references.CutStone, 2},
		{references.RecipeIngots, references.Ingots, 1},
	} {
		callbacks.Reset()
		require.NoError(t, state.Craft(testCase.recipe))
		recipe, _ := references.CraftRecipeByID(testCase.recipe)
		assert.Equal(t, testCase.made, inventory.Materials.Get(testCase.material), recipe.Name)
		assert.Equal(t, recipe.Minutes-DefaultSmallMapMinutesPerTurn, gatherMinutes(callbacks), recipe.Name)
	}
	assert.Zero(t, inventory.Materials.Get(references.Wood))
	assert.Zero(t, inventory.Materials.Get(references.Stone))
	assert.Zero(t, inventory.Materials.Get(references.Ore))
}

func TestCraftingAPieceAddsItToStockAndZtats(t *testing.T) {
	state, callbacks, _ := newBuildFixture(t)
	buildNorth(t, state, references.PieceWorkbench)
	state.PartyState.Inventory.Materials.Set(references.Planks, 4)

	require.NoError(t, state.Craft(references.RecipeBed))

	assert.Equal(t, uint16(1), state.PartyState.Inventory.Pieces.Get(references.PieceBedHead))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Pieces.Get(references.PieceBedFoot))
	assert.Contains(t, callbacks.Messages, "Crafted 1 Bed (head), 1 Bed (foot).")
	rows := state.projectZStatsCraftingItems()
	names := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Category == "Piece" {
			names = append(names, row.Name)
		}
	}
	assert.Equal(t, []string{"Bed (head)", "Bed (foot)"}, names)

	state.PartyState.Inventory.Pieces.Set(references.PieceBedHead, party_state.ItemQuantitySmallMax)
	state.PartyState.Inventory.Materials.Set(references.Planks, 4)
	callbacks.Reset()
	require.ErrorIs(t, state.Craft(references.RecipeBed), ErrCraftNoRoomToOutput, "piece stacks are capped at 99")
	assert.Equal(t, []string{"Thou canst carry no more Bed (head)!"}, callbacks.Messages)
}

func TestLookNamesBuiltStationsBedsAndChests(t *testing.T) {
	for _, testCase := range []struct {
		piece references.BuildPieceID
		look  string
	}{
		{references.PieceWorkbench, "Look-Thou dost see a sturdy workbench with a vise"},
		{references.PieceSawmill, "Look-Thou dost see a saw frame for cutting planks"},
		{references.PieceForge, "Look-Thou dost see a forge and anvil"},
		{references.PieceAlchemyTable, "Look-Thou dost see an alchemist's table with flask and retort"},
		{references.PieceFireplace, "Look-Thou dost see a hearth fit for cooking"},
		{references.PieceBedHead, "Look-Thou dost see thy own bed"},
		{references.PieceStorageChest, "Look-Thou dost see thy storage chest (empty)"},
	} {
		state, callbacks, _ := newBuildFixture(t)
		buildNorth(t, state, testCase.piece)
		callbacks.Reset()
		require.True(t, state.ActionLookSmallMap(references.Up))
		assert.Contains(t, callbacks.Messages, testCase.look)
	}
}

func TestPlacementsSurviveLeavingTheDeedAndASaveInsideOrOutside(t *testing.T) {
	state, _, deed := newBuildFixture(t)
	wall := northOfYard()
	chest := references.Position{X: yard.X + 1, Y: yard.Y - 1}
	chair := references.Position{X: yard.X - 1, Y: yard.Y - 1}
	buildAt(t, state, references.PieceStoneWall, wall)
	buildAt(t, state, references.PieceStorageChest, chest)
	buildAt(t, state, references.PieceWoodFloor, chair)
	buildAt(t, state, references.PieceChairBack, chair)
	state.PartyState.Inventory.Materials.Set(references.Wood, 30)
	require.Positive(t, state.TransferStorageChest(chest, StorageChestMaterials, true))
	state.PartyState.Inventory.Pieces.Set(references.PieceDoor, 3)

	check := func(t *testing.T, restored *GameState) {
		t.Helper()
		restoredDeed := restored.deedByID(deed.ID)
		original := state.deedByID(deed.ID)
		assert.Equal(t, original.placements, restoredDeed.placements)
		contents, ok := restored.StorageChestContents(chest)
		if restored.IsOnDeedMap() {
			require.True(t, ok)
			assert.Equal(t, uint16(30), contents.Materials[references.Wood])
		}
		assert.Equal(t, *original.chests[chest], *restoredDeed.chests[chest])
		assert.Equal(t, uint16(3), restored.PartyState.Inventory.Pieces.Get(references.PieceDoor))
		tile, _ := restored.DeedTileAt(deed.ID, chair)
		assert.Equal(t, indexes.SpriteIndex(indexes.ChairBackBack), tile)
	}

	inside := saveAndReload(t, state)
	require.True(t, inside.IsOnDeedMap())
	check(t, inside)
	assert.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall), liveTile(inside, wall))
	assert.Equal(t, indexes.SpriteIndex(indexes.Box), liveTile(inside, chest))

	state.MapState.PlayerLocation.Position = references.DeedMapEntry
	leaveDeedMap(t, state)
	require.False(t, state.IsOnDeedMap())
	outside := saveAndReload(t, state)
	check(t, outside)

	enterFixtureDeed(t, state)
	assert.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall), liveTile(state, wall), "re-entering rebuilds the placements")
	assert.Equal(t, indexes.SpriteIndex(indexes.ChairBackBack), liveTile(state, chair))
	state.MapState.PlayerLocation.Position = references.Position{X: chest.X, Y: chest.Y + 1}
	_, found := state.StorageChestInDirection(references.Up)
	assert.True(t, found, "the chest still opens")
}

func TestStorageChestTransfersEachCategoryUpToTheCaps(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	buildNorth(t, state, references.PieceStorageChest)
	chest := northOfYard()
	inventory := &state.PartyState.Inventory
	inventory.Materials.Set(references.Wood, 12)
	inventory.Tools.Set(references.Axe, 1)
	inventory.Pieces.Set(references.PieceDoor, 2)
	inventory.Gold.Set(500)
	inventory.Provisions.Food.Set(40)
	inventory.Provisions.Torches.Set(3)
	inventory.Provisions.Keys.Set(2)

	for _, testCase := range []struct {
		category StorageChestCategory
		moved    int
	}{
		{StorageChestMaterials, 12}, {StorageChestTools, 1}, {StorageChestPieces, 2}, {StorageChestGold, 500},
		{StorageChestProvisionsCategory, 45},
	} {
		assert.Equal(t, testCase.moved, state.TransferStorageChest(chest, testCase.category, true), testCase.category.Name())
	}
	contents, _ := state.StorageChestContents(chest)
	assert.Equal(t, uint16(12), contents.Materials[references.Wood])
	assert.Equal(t, uint16(1), contents.Tools[references.Axe])
	assert.Equal(t, uint16(2), contents.Pieces[references.PieceDoor])
	assert.Equal(t, uint16(500), contents.Gold)
	assert.Equal(t, [NumberOfChestProvisions]uint16{3, 0, 2, 0, 40}, contents.Provisions)
	assert.Zero(t, inventory.Gold.Get())
	assert.Zero(t, inventory.Provisions.Food.Get())
	assert.Equal(t, "The chest holds 12 Wood, 1 Axe, 2 Door, 3 Torches, 2 Keys, 40 Food, 500 Gold.",
		state.StorageChestSummary(chest))

	// Taking stops at the party's cap; storing stops at the chest's.
	inventory.Gold.Set(party_state.ItemQuantityLargeMax - 100)
	assert.Equal(t, 100, state.TransferStorageChest(chest, StorageChestGold, false))
	contents, _ = state.StorageChestContents(chest)
	assert.Equal(t, uint16(400), contents.Gold, "the rest stays in the chest")
	inventory.Tools.Set(references.Axe, party_state.ItemQuantitySmallMax)
	assert.Zero(t, state.TransferStorageChest(chest, StorageChestTools, false), "a full stack takes nothing")
	inventory.Gold.Set(party_state.ItemQuantityLargeMax)
	assert.Equal(t, int(party_state.ItemQuantityLargeMax-400), state.TransferStorageChest(chest, StorageChestGold, true))
	assert.Equal(t, uint16(400), inventory.Gold.Get(), "storing stops at the chest's cap")

	inventory.Gold.Set(0)
	inventory.Tools.Set(references.Axe, 0)
	moved := 0
	for _, category := range StorageChestCategories() {
		moved += state.TransferStorageChest(chest, category, false)
	}
	assert.Equal(t, 12+1+2+int(party_state.ItemQuantityLargeMax)+45, moved, "everything comes back out")
	contents, _ = state.StorageChestContents(chest)
	assert.True(t, contents.IsEmpty())
	assert.Equal(t, "The chest is empty.", state.StorageChestSummary(chest))
}

func TestStoringEverythingKeepsSurvivalGoodsWithTheParty(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	buildNorth(t, state, references.PieceStorageChest)
	chest := northOfYard()
	inventory := &state.PartyState.Inventory
	inventory.Materials.Set(references.Wood, 12)
	inventory.Tools.Set(references.Pick, 2)
	inventory.Gold.Set(100)
	inventory.Provisions.Food.Set(40)
	inventory.Provisions.Torches.Set(3)
	inventory.Provisions.Keys.Set(2)
	inventory.Provisions.SkullKeys.Set(1)
	inventory.Provisions.Gems.Set(5)

	assert.Equal(t, 12+100+5, state.TransferStorageChestEverything(chest, true))

	contents, _ := state.StorageChestContents(chest)
	assert.Equal(t, [NumberOfChestProvisions]uint16{0, 5, 0, 0, 0}, contents.Provisions, "only the gems go in")
	assert.Zero(t, contents.Tools[references.Pick])
	assert.Equal(t, uint16(40), inventory.Provisions.Food.Get())
	assert.Equal(t, uint16(3), inventory.Provisions.Torches.Get())
	assert.Equal(t, uint16(2), inventory.Provisions.Keys.Get())
	assert.Equal(t, uint16(1), inventory.Provisions.SkullKeys.Get())
	assert.Equal(t, uint16(2), inventory.Tools.Get(references.Pick))

	// Their own rows still store them, and taking everything brings them all back.
	assert.Equal(t, 40+3+2+1, state.TransferStorageChest(chest, StorageChestProvisionsCategory, true))
	assert.Equal(t, 2, state.TransferStorageChest(chest, StorageChestTools, true))
	assert.Equal(t, 12+100+5+40+3+2+1+2, state.TransferStorageChestEverything(chest, false))
	contents, _ = state.StorageChestContents(chest)
	assert.True(t, contents.IsEmpty())
}

func TestDemolishRefusesANonEmptyChest(t *testing.T) {
	state, callbacks, _ := newBuildFixture(t)
	buildNorth(t, state, references.PieceStorageChest)
	chest := northOfYard()
	state.PartyState.Inventory.Gold.Set(1)
	require.Equal(t, 1, state.TransferStorageChest(chest, StorageChestGold, true))
	callbacks.Reset()

	assert.False(t, state.DemolishDeedPiece(references.Up))
	assert.Equal(t, []string{"Empty the chest first!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())

	require.Equal(t, 1, state.TransferStorageChest(chest, StorageChestGold, false))
	require.True(t, state.DemolishDeedPiece(references.Up))
	_, found := state.StorageChestInDirection(references.Up)
	assert.False(t, found)
	assert.Empty(t, state.deedByID(state.crafting.active.ID).chests)
}

func TestStorageChestOpensOnlyOnTheDeedWithTheOptionOn(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	buildNorth(t, state, references.PieceStorageChest)
	_, found := state.StorageChestInDirection(references.Up)
	require.True(t, found)

	setCraftingEnabled(state, false)
	_, found = state.StorageChestInDirection(references.Up)
	assert.False(t, found, "with the option off the chest is plain scenery")
}

// standOnBed builds a bed north of the yard (head west, foot east) and puts the party on its head.
func standOnBed(t *testing.T, state *GameState) references.Position {
	t.Helper()
	head := northOfYard()
	buildAt(t, state, references.PieceBedHead, head)
	buildAt(t, state, references.PieceBedFoot, references.Position{X: head.X + 1, Y: head.Y})
	state.MapState.PlayerLocation.Position = head

	return head
}

func TestOwnBedRestsLikeAnInnForFree(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	head := standOnBed(t, state)
	require.True(t, state.IsOnOwnDeedBed())
	avatar := state.PartyState.GetCharacter(0)
	avatar.SetHp(1)
	avatar.SetMp(0)
	gold, food := state.PartyState.Inventory.Gold.Get(), state.PartyState.Inventory.Provisions.Food.Get()
	state.DateTime.Hour, state.DateTime.Minute = 22, 0

	result := state.RestInOwnDeedBed()

	assert.Equal(t, avatar.GetMaxHp(), avatar.GetCurrentHp(), "the morning restores every hit point")
	assert.Equal(t, avatar.Intelligence, avatar.GetCurrentMp(), "the Avatar's magic returns as at an inn")
	assert.Equal(t, party_state.Good, avatar.GetStatus())
	assert.Equal(t, byte(6), state.DateTime.Hour, "the party sleeps until six")
	assert.Equal(t, byte(6), result.DateTimeAfter.Hour)
	assert.Equal(t, gold, state.PartyState.Inventory.Gold.Get(), "the party's own bed is free")
	assert.Equal(t, food, state.PartyState.Inventory.Provisions.Food.Get(), "no meals are replayed, as at an inn")
	assert.Empty(t, result.PoisonDeaths)
	assert.Equal(t, head, state.MapState.PlayerLocation.Position)
}

func TestOwnBedRestIsFatalToThePoisonedAsAtAnInn(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	standOnBed(t, state)
	avatar := state.PartyState.GetCharacter(0)
	avatar.SetStatus(party_state.Poisoned)

	result := state.RestInOwnDeedBed()

	assert.Equal(t, []string{avatar.GetNameAsString()}, result.PoisonDeaths)
	assert.Equal(t, party_state.Dead, avatar.GetStatus())
}

func TestOnlyTheHeadOfAnOwnBuiltBedRestsLikeAnInn(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	head := standOnBed(t, state)
	state.MapState.PlayerLocation.Position = references.Position{X: head.X + 1, Y: head.Y}
	assert.False(t, state.IsOnOwnDeedBed(), "the foot of the bed")
	state.MapState.PlayerLocation.Position = head
	setCraftingEnabled(state, false)
	assert.False(t, state.IsOnOwnDeedBed(), "the option is off")
	setCraftingEnabled(state, true)
	assert.True(t, state.IsOnOwnDeedBed())
}

func TestTheFireplaceCooksForageIntoFood(t *testing.T) {
	state, callbacks, _ := newBuildFixture(t)
	buildNorth(t, state, references.PieceFireplace)
	inventory := &state.PartyState.Inventory
	inventory.Materials.Set(references.Forage, 2)
	inventory.Materials.Set(references.Wood, 1)
	inventory.Provisions.Food.Set(10)
	callbacks.Reset()

	require.NoError(t, state.Craft(references.RecipeCookFood))

	assert.Equal(t, uint16(13), inventory.Provisions.Food.Get())
	assert.Zero(t, inventory.Materials.Get(references.Forage))
	assert.Contains(t, callbacks.Messages, "Crafted 3 Food.")
	recipe, _ := references.CraftRecipeByID(references.RecipeCookFood)
	assert.Equal(t, recipe.Minutes-DefaultSmallMapMinutesPerTurn, gatherMinutes(callbacks))

	inventory.Materials.Set(references.Forage, 2)
	inventory.Materials.Set(references.Wood, 1)
	inventory.Provisions.Food.Set(party_state.ItemQuantityLargeMax - 2)
	callbacks.Reset()
	require.ErrorIs(t, state.Craft(references.RecipeCookFood), ErrCraftNoRoomToOutput)
	assert.Equal(t, []string{"Thou canst carry no more Food!"}, callbacks.Messages)
	assert.Equal(t, uint16(2), inventory.Materials.Get(references.Forage), "a refusal spends nothing")
}

// gatherFrom works the nodes of a kind, in order, until the party carries want of the material.
func gatherFrom(t *testing.T, state *GameState, deed Deed, kind references.DeedResourceKind, material references.Material, want uint16) {
	t.Helper()
	for range 400 {
		if state.PartyState.Inventory.Materials.Get(material) >= want {
			return
		}
		worked := false
		for _, node := range state.DeedResourceNodes(deed.ID) {
			if node.Kind != kind || node.Depleted || node.Covered {
				continue
			}
			standSouthOf(state, node)
			require.True(t, state.GatherDeedNode(references.Up))
			worked = true

			break
		}
		require.True(t, worked, "no %s node left", kind.Name())
	}
	require.GreaterOrEqual(t, state.PartyState.Inventory.Materials.Get(material), want)
}

func craftOrFail(t *testing.T, state *GameState, recipe references.CraftRecipeID) {
	t.Helper()
	require.NoError(t, state.Craft(recipe))
}

func TestABarehandedPartyCanReachAForge(t *testing.T) {
	// Foothills are the ore terrain (ref:CRAFT-0003); a grassland deed has one ore vein and may
	// have to wait for Trammel to restore it.
	state, _ := newDeedFixture(t, true)
	setOverworldTile(state, deedFixtureSquare, indexes.Hills1)
	deed := stakeFixtureDeed(t, state, "Quarry")
	require.Equal(t, references.DeedTerrainHills, deed.Terrain)
	enterFixtureDeed(t, state)
	inventory := &state.PartyState.Inventory
	require.Zero(t, inventory.Materials.Get(references.Wood))

	// Bare hands win the first Wood and Stone for the tools.
	gatherFrom(t, state, deed, references.DeedResourceWood, references.Wood, 4)
	gatherFrom(t, state, deed, references.DeedResourceStone, references.Stone, 3)
	craftOrFail(t, state, references.RecipeAxe)
	craftOrFail(t, state, references.RecipePick)

	// Tools gather enough for a workbench, a sawmill, cut stone and a forge.
	gatherFrom(t, state, deed, references.DeedResourceWood, references.Wood, 17)
	gatherFrom(t, state, deed, references.DeedResourceStone, references.Stone, 18)
	gatherFrom(t, state, deed, references.DeedResourceOre, references.Ore, 5)
	state.MapState.PlayerLocation.Position = yard
	craftOrFail(t, state, references.RecipeWorkbench)
	craftOrFail(t, state, references.RecipeSawmill)
	require.True(t, state.BuildDeedPiece(references.PieceWorkbench, references.Up))
	require.True(t, state.BuildDeedPiece(references.PieceSawmill, references.Left))
	craftOrFail(t, state, references.RecipePlanks)
	for range 4 {
		craftOrFail(t, state, references.RecipeCutStone)
	}
	craftOrFail(t, state, references.RecipeForge)
	require.True(t, state.BuildDeedPiece(references.PieceForge, references.Right))
	craftOrFail(t, state, references.RecipeIngots)

	assert.Equal(t, uint16(1), inventory.Materials.Get(references.Ingots))
	assert.Equal(t, uint16(4), inventory.Materials.Get(references.Planks))
}

func TestBuildPieceStocksListCarriedPiecesInCatalogueOrder(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	state.PartyState.Inventory.Pieces.Set(references.PieceFireplace, 1)
	state.PartyState.Inventory.Pieces.Set(references.PieceStoneWall, 7)

	stocks := state.BuildPieceStocks()

	require.Len(t, stocks, 2)
	assert.Equal(t, references.PieceStoneWall, stocks[0].Piece.ID)
	assert.Equal(t, uint16(7), stocks[0].Quantity)
	assert.Equal(t, references.PieceFireplace, stocks[1].Piece.ID)
}

// TestBuiltPiecesKeepTheirOriginalTileRules checks walls, windows and fences block, and floors,
// chairs and beds can be walked on, as their original tiles.
func TestBuiltPiecesKeepTheirOriginalTileRules(t *testing.T) {
	for _, testCase := range []struct {
		piece    references.BuildPieceID
		passable bool
	}{
		{references.PieceStoneWall, false}, {references.PieceWindow, false}, {references.PieceFenceEastWest, false},
		{references.PieceWoodFloor, true}, {references.PieceChairForward, true}, {references.PieceBedHead, true},
		{references.PieceTableMiddle, false}, {references.PieceWorkbench, false},
	} {
		state, _, _ := newBuildFixture(t)
		buildNorth(t, state, testCase.piece)
		target := northOfYard()
		assert.Equal(t, testCase.passable, state.IsPassable(&target), testCase.piece.Piece().Name)
	}
}

func TestAnOpenedBuiltDoorSurvivesASaveOnTheDeedMap(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	target := northOfYard()
	buildNorth(t, state, references.PieceDoor)
	require.True(t, state.ActionOpenSmallMap(references.Up))

	restored := saveAndReload(t, state)

	door := restored.MapState.TrackedOpenDoor()
	require.NotNil(t, door, "the open door's timer is restored")
	assert.Equal(t, target, door.Position.Position)
	assert.True(t, restored.IsPassable(&target))
	for range map_state.DefaultOpenDoorTurns + 1 {
		restored.MapState.SmallMapProcessTurnDoors(func(references.Position, references.FloorNumber) bool { return false })
	}
	assert.False(t, restored.IsPassable(&target), "it closes again as a door")
}

func TestBuiltPiecesCannotBePushedOrPulled(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		state, callbacks, _ := newBuildFixture(t)
		chest := northOfYard()
		beyond := references.Position{X: chest.X, Y: chest.Y - 1}
		buildAt(t, state, references.PieceStoneFloor, yard)
		buildAt(t, state, references.PieceStoneFloor, chest)
		buildAt(t, state, references.PieceStoneFloor, beyond)
		buildAt(t, state, references.PieceStorageChest, chest)
		state.MapState.PlayerLocation.Position = yard
		setCraftingEnabled(state, enabled)
		callbacks.Reset()

		assert.False(t, state.ActionPushSmallMap(references.Up), "a built chest on a brick floor stays put")
		assert.True(t, callbacks.MessagesContain("Won't budge!"))
		assert.Equal(t, indexes.SpriteIndex(indexes.Box), liveTile(state, chest))
		assert.Equal(t, indexes.SpriteIndex(indexes.BrickFloor), liveTile(state, beyond))
		assert.Equal(t, yard, state.MapState.PlayerLocation.Position)
		callbacks.AssertTimeAdvanced(0)
	}
}

func TestAnYlemLeavesBuiltPiecesStanding(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	chest := northOfYard()
	buildNorth(t, state, references.PieceStorageChest)

	assert.Equal(t, AnYlemNoEffect, state.CastAnYlem(&chest))
	assert.Equal(t, indexes.SpriteIndex(indexes.Box), state.GetLayeredMapByCurrentLocation().GetTileTopMapOnlyTile(&chest).Index)
	assert.False(t, state.IsPassable(&chest))
}

func TestAnExPorBeyondTheDeedMapEdgeIsNoDoor(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	state.MapState.PlayerLocation.Position = references.Position{X: 0, Y: 15}
	beyond := references.Position{X: -1, Y: 15}

	assert.Equal(t, MagicLockNotADoor, state.MagicLockDoor(&beyond))
}
