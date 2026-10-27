package game_state

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

var craftingFixtureParty = references.Position{X: 100, Y: 100}

// newCraftingFixture puts the party on a cleared grass square of the overworld with crafting on.
func newCraftingFixture(t *testing.T, enabled bool) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(int(craftingFixtureParty.X), int(craftingFixtureParty.Y)).
		WithSystemCallbacks().
		WithRandomSeed(77).
		Build()
	configuration := *state.GameConfig
	gameplay := *configuration.GameplayConfiguration
	gameplay.CraftingEnabled = enabled
	configuration.GameplayConfiguration = &gameplay
	state.GameConfig = &configuration
	state.DebugOptions.MonsterGen = false
	state.CurrentNPCAIController.GetNpcs().ClearAllMapUnits()
	state.ItemStacksMap = *references.NewItemStacksMap()
	for dy := -references.LandClaimSize; dy <= references.LandClaimSize; dy++ {
		for dx := -references.LandClaimSize; dx <= references.LandClaimSize; dx++ {
			setCraftingTile(state, offset(craftingFixtureParty, dx, dy), indexes.Grass)
		}
	}
	callbacks.Reset()

	return state, callbacks
}

func offset(position references.Position, dx, dy int) references.Position {
	return *(&references.Position{
		X: position.X + references.Coordinate(dx), Y: position.Y + references.Coordinate(dy),
	}).GetWrapped(references.XLargeMapTiles, references.YLargeMapTiles)
}

func setCraftingTile(state *GameState, position references.Position, tile indexes.SpriteIndex) {
	state.overworldMap().UnSetTileByLayer(map_state.MapOverrideLayer, &position)
	state.overworldMap().SetTileByLayer(map_state.MapLayer, &position, tile)
}

func craftingTile(state *GameState, position references.Position) indexes.SpriteIndex {
	return state.overworldMap().SpriteIndexByLayer(map_state.MapLayer, position)
}

func giveCraftingSupplies(state *GameState) {
	inventory := &state.PartyState.Inventory
	inventory.AddMaterials(references.MaterialCost{references.Wood: 100, references.Stone: 100})
	inventory.Gold.Set(1000)
}

func TestCraftingCommandsAnswerWhatWhenDisabled(t *testing.T) {
	state, callbacks := newCraftingFixture(t, false)
	setCraftingTile(state, offset(craftingFixtureParty, 1, 0), indexes.Tree)

	assert.False(t, state.ActionGatherLargeMap(references.Right))
	assert.False(t, state.ActionBuildLargeMap(0, references.Right))
	require.ErrorIs(t, state.ApplyCraft(0), ErrCraftingDisabled)
	assert.Equal(t, []string{"What?", "What?"}, callbacks.Messages)
	assert.Equal(t, indexes.SpriteIndex(indexes.Tree), craftingTile(state, offset(craftingFixtureParty, 1, 0)))
}

func TestGatherWithAxeFellsTreeAndThinsForest(t *testing.T) {
	state, callbacks := newCraftingFixture(t, true)
	state.PartyState.Inventory.Tools.Set(references.Axe, 1)
	tree := offset(craftingFixtureParty, 1, 0)
	forest := offset(craftingFixtureParty, -1, 0)
	setCraftingTile(state, tree, indexes.Tree)
	setCraftingTile(state, forest, indexes.Forest4)

	require.True(t, state.ActionGatherLargeMap(references.Right))
	assert.Equal(t, indexes.SpriteIndex(indexes.Grass), craftingTile(state, tree))
	wood := state.PartyState.Inventory.Materials.Get(references.Wood)
	assert.GreaterOrEqual(t, wood, uint16(2))
	assert.LessOrEqual(t, wood, uint16(4))
	assert.Equal(t, references.GatherMinutes, sumMinutes(callbacks.TimeAdvanced()))

	require.True(t, state.ActionGatherLargeMap(references.Left))
	assert.Equal(t, indexes.SpriteIndex(indexes.Forest3), craftingTile(state, forest))
}

func TestGatherStoneNeedsPickForMountains(t *testing.T) {
	state, callbacks := newCraftingFixture(t, true)
	mountain := offset(craftingFixtureParty, 0, -1)
	setCraftingTile(state, mountain, indexes.TallMountains)

	assert.False(t, state.ActionGatherLargeMap(references.Up))
	assert.Equal(t, []string{"You need a Pick!"}, callbacks.Messages)

	state.PartyState.Inventory.Tools.Set(references.Pick, 1)
	require.True(t, state.ActionGatherLargeMap(references.Up))
	assert.Equal(t, indexes.SpriteIndex(indexes.TallMountains), craftingTile(state, mountain), "mountains never deplete")
	assert.Positive(t, state.PartyState.Inventory.Materials.Get(references.Stone))
}

func TestGatherBareHandsNeverChangesTerrain(t *testing.T) {
	state, _ := newCraftingFixture(t, true)
	tree := offset(craftingFixtureParty, 1, 0)
	setCraftingTile(state, tree, indexes.Tree)

	for range 30 {
		state.ActionGatherLargeMap(references.Right)
	}
	assert.Equal(t, indexes.SpriteIndex(indexes.Tree), craftingTile(state, tree))
	wood := state.PartyState.Inventory.Materials.Get(references.Wood)
	assert.Positive(t, wood, "one in three bare-hand attempts should find a branch")
	assert.Less(t, wood, uint16(30))
}

func TestGatherRefusedOffTheSurface(t *testing.T) {
	state, callbacks := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15).
		WithSystemCallbacks().
		Build()
	configuration := *state.GameConfig
	gameplay := *configuration.GameplayConfiguration
	gameplay.CraftingEnabled = true
	configuration.GameplayConfiguration = &gameplay
	state.GameConfig = &configuration
	callbacks.Reset()

	assert.False(t, state.ActionGatherLargeMap(references.Right))
	assert.Equal(t, []string{"Not here!"}, callbacks.Messages)
}

func TestApplyCraftSpendsMaterialsForTool(t *testing.T) {
	state, _ := newCraftingFixture(t, true)
	require.ErrorIs(t, state.ApplyCraft(0), ErrNotEnoughMaterials)

	state.PartyState.Inventory.AddMaterials(references.MaterialCost{references.Wood: 3, references.Stone: 2})
	require.NoError(t, state.ApplyCraft(0))
	assert.Equal(t, uint16(1), state.PartyState.Inventory.Tools.Get(references.Axe))
	assert.Zero(t, state.PartyState.Inventory.Materials.Get(references.Wood))
	assert.Zero(t, state.PartyState.Inventory.Materials.Get(references.Stone))
	require.ErrorIs(t, state.ApplyCraft(99), ErrUnknownRecipe)
}

func TestStakeLandClaimChargesAndRejectsOverlap(t *testing.T) {
	state, _ := newCraftingFixture(t, true)
	_, err := state.StakeLandClaim("Homestead")
	require.ErrorIs(t, err, ErrNotEnoughGold)

	giveCraftingSupplies(state)
	claim, err := state.StakeLandClaim("  Homestead  ")
	require.NoError(t, err)
	assert.Equal(t, "Homestead", claim.Name)
	assert.True(t, claim.Contains(craftingFixtureParty))
	assert.Equal(t, craftingFixtureParty, claim.Center())
	assert.Equal(t, uint16(1000-references.LandClaimGoldCost), state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, uint16(100-references.LandClaimStakeCost[references.Wood]), state.PartyState.Inventory.Materials.Get(references.Wood))
	assert.True(t, state.isPartyInsideOwnClaim())
	notes := state.CaptureAtlasNotes().Records
	require.Len(t, notes, 1)
	assert.Equal(t, "Homestead", notes[0].Label)
	assert.Equal(t, int(craftingFixtureParty.X), notes[0].Position.X)

	_, err = state.StakeLandClaim("Again")
	require.ErrorIs(t, err, ErrLandAlreadyClaimed)
	_, err = state.StakeLandClaim("   ")
	require.ErrorIs(t, err, ErrInvalidLandClaimName)
}

func TestStakeLandClaimRejectsSettlementsAndWater(t *testing.T) {
	state, _ := newCraftingFixture(t, true)
	giveCraftingSupplies(state)
	setCraftingTile(state, offset(craftingFixtureParty, 3, 3), indexes.Village)
	_, err := state.StakeLandClaim("Village green")
	require.ErrorIs(t, err, ErrLandClaimHasSettlement)

	setCraftingTile(state, offset(craftingFixtureParty, 3, 3), indexes.Grass)
	for dy := -8; dy < 8; dy++ {
		for dx := -8; dx < 1; dx++ {
			setCraftingTile(state, offset(craftingFixtureParty, dx, dy), indexes.Water1)
		}
	}
	_, err = state.StakeLandClaim("Lakeside")
	require.ErrorIs(t, err, ErrLandClaimTooWet)
}

func TestLandClaimContainsWrapsAroundTheWorld(t *testing.T) {
	claim := LandClaim{Origin: references.Position{X: 250, Y: 250}}
	assert.True(t, claim.Contains(references.Position{X: 255, Y: 2}))
	assert.True(t, claim.Contains(references.Position{X: 9, Y: 9}))
	assert.False(t, claim.Contains(references.Position{X: 10, Y: 9}))
	assert.False(t, claim.Contains(references.Position{X: 249, Y: 250}))
}

func TestBuildRequiresOwnClaimAndMaterials(t *testing.T) {
	state, callbacks := newCraftingFixture(t, true)
	assert.False(t, state.ActionBuildLargeMap(2, references.Right))
	assert.Equal(t, []string{"Not your land!"}, callbacks.Messages)

	giveCraftingSupplies(state)
	_, err := state.StakeLandClaim("Homestead")
	require.NoError(t, err)
	state.PartyState.Inventory.Materials.Set(references.Stone, 0)
	callbacks.Reset()
	assert.False(t, state.ActionBuildLargeMap(2, references.Right))
	assert.Equal(t, []string{"You need 3 Stone."}, callbacks.Messages)
	assert.Equal(t, indexes.SpriteIndex(indexes.Grass), craftingTile(state, offset(craftingFixtureParty, 1, 0)))
}

func TestBuildFloorThenFurnitureAndDemolishRestoresTerrain(t *testing.T) {
	state, _ := newCraftingFixture(t, true)
	giveCraftingSupplies(state)
	_, err := state.StakeLandClaim("Homestead")
	require.NoError(t, err)
	target := offset(craftingFixtureParty, 1, 0)

	require.True(t, state.ActionBuildLargeMap(0, references.Right)) // Wood floor
	assert.Equal(t, indexes.SpriteIndex(indexes.WoodFloor1), craftingTile(state, target))
	require.True(t, state.ActionBuildLargeMap(17, references.Right)) // Dresser over the floor
	assert.Equal(t, indexes.SpriteIndex(indexes.Dresser), craftingTile(state, target))
	require.False(t, state.ActionBuildLargeMap(2, references.Right), "a dresser is not buildable ground")

	woodBefore := state.PartyState.Inventory.Materials.Get(references.Wood)
	require.True(t, state.ActionDemolishLargeMap(references.Right))
	assert.Equal(t, indexes.SpriteIndex(indexes.Grass), craftingTile(state, target), "demolition restores the original terrain")
	assert.Equal(t, woodBefore+1, state.PartyState.Inventory.Materials.Get(references.Wood), "half of 3 wood is refunded")
	assert.False(t, state.ActionDemolishLargeMap(references.Right), "nothing placed remains")
}

func TestBuiltWallBlocksWalking(t *testing.T) {
	state, _ := newCraftingFixture(t, true)
	giveCraftingSupplies(state)
	_, err := state.StakeLandClaim("Homestead")
	require.NoError(t, err)
	require.True(t, state.ActionBuildLargeMap(2, references.Right))

	wall := state.GameReferences.TileReferences.GetTile(craftingTile(state, offset(craftingFixtureParty, 1, 0)))
	assert.False(t, wall.IsPassable(references.NoPartyVehicle))
}

func TestCraftingStateSurvivesSaveAndReload(t *testing.T) {
	state, _ := newCraftingFixture(t, true)
	giveCraftingSupplies(state)
	state.PartyState.Inventory.Tools.Set(references.Pick, 1)
	_, err := state.StakeLandClaim("Homestead")
	require.NoError(t, err)
	require.True(t, state.ActionBuildLargeMap(0, references.Up))
	require.True(t, state.ActionBuildLargeMap(22, references.Up)) // Storage chest over the floor
	require.True(t, state.ActionBuildLargeMap(7, references.Down))
	chest, ok := state.HomeChestAt(offset(craftingFixtureParty, 0, -1))
	require.True(t, ok)
	chest.Materials[references.Stone] = 7

	snapshot, err := state.CaptureSaveSnapshot()
	require.NoError(t, err)
	restored, err := NewGameStateFromSaveSnapshot(snapshot, state.GameConfig, state.GameReferences,
		xTilesVisibleOnGameScreen, yTilesVisibleOnGameScreen)
	require.NoError(t, err)
	restored.SystemCallbacks = state.SystemCallbacks

	assert.Equal(t, state.LandClaims(), restored.LandClaims())
	assert.Equal(t, state.PartyState.Inventory.Materials.Get(references.Wood), restored.PartyState.Inventory.Materials.Get(references.Wood))
	assert.Equal(t, uint16(1), restored.PartyState.Inventory.Tools.Get(references.Pick))
	assert.Equal(t, indexes.SpriteIndex(indexes.Box), craftingTile(restored, offset(craftingFixtureParty, 0, -1)))
	assert.Equal(t, indexes.SpriteIndex(indexes.RegularDoor), craftingTile(restored, offset(craftingFixtureParty, 0, 1)))
	restoredChest, ok := restored.HomeChestAt(offset(craftingFixtureParty, 0, -1))
	require.True(t, ok)
	assert.Equal(t, uint16(7), restoredChest.Materials[references.Stone])

	require.False(t, restored.ActionDemolishLargeMap(references.Up), "a full chest cannot be demolished")
	restoredChest.Materials[references.Stone] = 0
	require.True(t, restored.ActionDemolishLargeMap(references.Up))
	assert.Equal(t, indexes.SpriteIndex(indexes.Grass), craftingTile(restored, offset(craftingFixtureParty, 0, -1)),
		"the chest was built over a floor, which was built over grass")
}

func TestHomeChestDepositAndWithdraw(t *testing.T) {
	state, _ := newCraftingFixture(t, true)
	giveCraftingSupplies(state)
	_, err := state.StakeLandClaim("Homestead")
	require.NoError(t, err)
	require.True(t, state.ActionBuildLargeMap(22, references.Right))
	position, ok := state.HomeChestInDirection(references.Right)
	require.True(t, ok)
	_, ok = state.HomeChestInDirection(references.Left)
	assert.False(t, ok)

	gold := state.PartyState.Inventory.Gold.Get()
	assert.Equal(t, int(gold), state.TransferHomeChest(position, HomeChestGold, true))
	assert.Zero(t, state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, "Chest: "+strconv.Itoa(int(gold))+" Gold", state.HomeChestSummary(position))

	state.PartyState.Inventory.Gold.Set(9990)
	assert.Equal(t, 9, state.TransferHomeChest(position, HomeChestGold, false), "withdrawal stops at the stack cap")
	assert.Equal(t, uint16(9999), state.PartyState.Inventory.Gold.Get())
	require.False(t, state.ActionDemolishLargeMap(references.Right), "the chest still holds gold")
}

func TestClaimedCampSiteAndSpawnGuard(t *testing.T) {
	state, _ := newCraftingFixture(t, true)
	site := state.MapState.PlayerLocation
	assert.False(t, state.isClaimedCampSite(site))
	assert.False(t, state.isClaimSpawnBlocked(craftingFixtureParty))

	giveCraftingSupplies(state)
	_, err := state.StakeLandClaim("Homestead")
	require.NoError(t, err)
	assert.True(t, state.isClaimedCampSite(site))
	assert.True(t, state.isClaimSpawnBlocked(offset(craftingFixtureParty, 7, 7)))
	assert.False(t, state.isClaimSpawnBlocked(offset(craftingFixtureParty, 8, 8)))

	state.GameConfig.GameplayConfiguration.CraftingEnabled = false
	assert.False(t, state.isClaimedCampSite(site), "claims grant nothing while the option is off")
	assert.False(t, state.isClaimSpawnBlocked(craftingFixtureParty))
}

func sumMinutes(minutes []int) int {
	total := 0
	for _, minute := range minutes {
		total += minute
	}

	return total
}
