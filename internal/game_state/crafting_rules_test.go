package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/camping"
	"github.com/bradhannah/UltimaVRedux/internal/journal"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func boardCraftingFixtureHorse(state *GameState) {
	state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle,
		state.MapState.PlayerLocation.Position, state.MapState.PlayerLocation.Floor)
}

func TestCraftRefusedOffTheSurfaceAndAboardAVehicle(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		setup func(*GameState)
	}{
		{name: "underworld", setup: func(state *GameState) {
			state.MapState.PlayerLocation.Floor = references.FloorNumber(references.UNDERWORLD)
		}},
		{name: "on horseback", setup: boardCraftingFixtureHorse},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			state, callbacks := newCraftingFixture(t, true)
			state.PartyState.Inventory.AddMaterials(references.MaterialCost{references.Wood: 3, references.Stone: 2})
			testCase.setup(state)
			callbacks.Reset()

			require.ErrorIs(t, state.ApplyCraft(0), ErrCraftingNotOnSurface)
			assert.Equal(t, []string{"Not here!"}, callbacks.Messages)
			assert.Zero(t, state.PartyState.Inventory.Tools.Get(references.Axe))
			assert.Equal(t, uint16(3), state.PartyState.Inventory.Materials.Get(references.Wood), "a refused craft keeps materials")
			assert.Empty(t, callbacks.TimeAdvanced())
		})
	}
}

func TestGatherAndBuildRefusedAboardAVehicle(t *testing.T) {
	state, callbacks := newCraftingFixture(t, true)
	giveCraftingSupplies(state)
	_, err := state.StakeLandClaim("Homestead")
	require.NoError(t, err)
	setCraftingTile(state, offset(craftingFixtureParty, 1, 0), indexes.Tree)
	boardCraftingFixtureHorse(state)
	callbacks.Reset()

	assert.False(t, state.ActionGatherLargeMap(references.Right))
	assert.False(t, state.ActionBuildLargeMap(0, references.Left))
	assert.Equal(t, []string{"Not here!", "Not here!"}, callbacks.Messages)
}

// The menu path completes Craft and Stake through the surface command cycle, which pays the
// ordinary turn; the action itself spends only the rest (the scene tests check the total).
func TestCraftAndStakeSpendOnlyLaborBeyondTheOrdinaryTurn(t *testing.T) {
	state, callbacks := newCraftingFixture(t, true)
	giveCraftingSupplies(state)

	require.NoError(t, state.ApplyCraft(0))
	assert.Equal(t, references.CraftRecipes()[0].Minutes-DefaultLargeMapMinutesPerTurn, sumMinutes(callbacks.TimeAdvanced()))

	callbacks.Reset()
	_, err := state.StakeLandClaim("Homestead")
	require.NoError(t, err)
	assert.Equal(t, references.LandClaimStakeMinutes-DefaultLargeMapMinutesPerTurn, sumMinutes(callbacks.TimeAdvanced()))
}

func TestGatherRefusesASquareUnderAnItemStack(t *testing.T) {
	state, callbacks := newCraftingFixture(t, true)
	state.PartyState.Inventory.Tools.Set(references.Axe, 1)
	tree := offset(craftingFixtureParty, 1, 0)
	setCraftingTile(state, tree, indexes.Tree)
	require.NoError(t, state.ItemStacksMap.AddOrMerge(&tree, references.ItemAndQuantity{Item: references.Food, Quantity: 3}))

	assert.False(t, state.ActionGatherLargeMap(references.Right))
	assert.Equal(t, []string{"Nothing to gather!"}, callbacks.Messages)
	assert.Equal(t, indexes.SpriteIndex(indexes.Tree), craftingTile(state, tree), "the tree under the loot stands")
	assert.Zero(t, state.PartyState.Inventory.Materials.Get(references.Wood))
}

func placeCraftingMoonstone(state *GameState, position references.Position) {
	state.moonstoneSystem = NewMoonstoneSystem()
	stone := &state.moonstoneSystem.Moonstones[0]
	stone.X, stone.Y = uint8(position.X), uint8(position.Y)
	stone.Map, stone.Level = uint8(references.BritanniaUnderworld), 0
	stone.State = MoonstoneStateBuried
}

func TestMoongateSquaresAreNeverClaimedOrBuiltOn(t *testing.T) {
	state, callbacks := newCraftingFixture(t, true)
	giveCraftingSupplies(state)
	gate := offset(craftingFixtureParty, 1, 0)
	placeCraftingMoonstone(state, offset(craftingFixtureParty, 5, -6))

	_, err := state.StakeLandClaim("Gatehouse")
	require.ErrorIs(t, err, ErrLandClaimHasMoongate)
	assert.Equal(t, []string{"Too near a moongate!"}, callbacks.Messages)

	placeCraftingMoonstone(state, offset(craftingFixtureParty, 40, 40))
	_, err = state.StakeLandClaim("Homestead")
	require.NoError(t, err)
	// A moonstone buried inside the claim afterwards still keeps its square clear.
	placeCraftingMoonstone(state, gate)
	callbacks.Reset()
	assert.False(t, state.ActionBuildLargeMap(2, references.Right))
	assert.Equal(t, []string{"Not here!"}, callbacks.Messages)
	assert.Equal(t, indexes.SpriteIndex(indexes.Grass), craftingTile(state, gate))
}

func TestLandClaimNamesUseThePrintableAtlasRepertoire(t *testing.T) {
	for _, testCase := range []struct {
		name string
		ok   bool
	}{
		{name: "Homestead", ok: true},
		{name: "  Lord's Rest  ", ok: true},
		{name: "abcdefghijklmnopqrstuvwx", ok: true},
		{name: "abcdefghijklmnopqrstuvwxy"},
		{name: "   "},
		{name: "Café"},
		{name: "Tab\there"},
	} {
		_, ok := NormalizeLandClaimName(testCase.name)
		assert.Equal(t, testCase.ok, ok, "%q", testCase.name)
	}

	state, _ := newCraftingFixture(t, true)
	giveCraftingSupplies(state)
	_, err := state.StakeLandClaim("Café")
	require.ErrorIs(t, err, ErrInvalidLandClaimName)
	assert.Empty(t, state.LandClaims())
}

func craftingAtlasNote(state *GameState, id uint64, x, y int) journal.AtlasNoteRecord {
	return journal.AtlasNoteRecord{
		ID:         id,
		Position:   journal.SiteLocation{Map: references.BritanniaUnderworld, X: x, Y: y},
		Label:      "Old note",
		RecordedAt: state.sampleDungeonObservation().at.Time,
	}
}

func TestStakeLandClaimKeepsAnExistingAtlasNoteSilently(t *testing.T) {
	state, callbacks := newCraftingFixture(t, true)
	giveCraftingSupplies(state)
	existing := craftingAtlasNote(state, 1, int(craftingFixtureParty.X), int(craftingFixtureParty.Y))
	require.NoError(t, state.RestoreAtlasNotes(journal.AtlasNotesState{LastID: 1, Records: []journal.AtlasNoteRecord{existing}}))

	_, err := state.StakeLandClaim("Homestead")
	require.NoError(t, err)
	assert.Equal(t, []string{"You claim Homestead!"}, callbacks.Messages)
	assert.Equal(t, []journal.AtlasNoteRecord{existing}, state.CaptureAtlasNotes().Records)
}

func TestStakeLandClaimReportsAFullAtlas(t *testing.T) {
	state, callbacks := newCraftingFixture(t, true)
	giveCraftingSupplies(state)
	notes := journal.AtlasNotesState{LastID: journal.MaxAtlasNotes}
	for index := range journal.MaxAtlasNotes {
		notes.Records = append(notes.Records, craftingAtlasNote(state, uint64(index+1), index, 0))
	}
	require.NoError(t, state.RestoreAtlasNotes(notes))

	_, err := state.StakeLandClaim("Homestead")
	require.NoError(t, err)
	assert.Equal(t, []string{"You claim Homestead!", "Your Atlas has no room to mark it."}, callbacks.Messages)
	assert.Len(t, state.CaptureAtlasNotes().Records, journal.MaxAtlasNotes)
}

func TestHomeChestTransfersEveryCategory(t *testing.T) {
	state, _ := newCraftingFixture(t, true)
	giveCraftingSupplies(state)
	_, err := state.StakeLandClaim("Homestead")
	require.NoError(t, err)
	require.True(t, state.ActionBuildLargeMap(22, references.Right))
	position, ok := state.HomeChestInDirection(references.Right)
	require.True(t, ok)
	inventory := &state.PartyState.Inventory
	inventory.Materials.Set(references.Wood, 20)
	inventory.Materials.Set(references.Stone, 5)
	inventory.Tools.Set(references.Axe, 2)
	inventory.Provisions.Food.Set(30)
	inventory.Provisions.Torches.Set(4)
	inventory.Provisions.Gems.Set(3)
	inventory.Provisions.Keys.Set(2)
	inventory.Provisions.SkullKeys.Set(1)
	inventory.Gold.Set(0)

	assert.Equal(t, 25, state.TransferHomeChest(position, HomeChestMaterials, true))
	assert.Equal(t, 2, state.TransferHomeChest(position, HomeChestTools, true))
	assert.Equal(t, 30, state.TransferHomeChest(position, HomeChestFood, true))
	assert.Equal(t, 10, state.TransferHomeChest(position, HomeChestSupplies, true))
	assert.Equal(t, "Chest: 20 Wood, 5 Stone, 2 Axe, 4 Torches, 3 Gems, 2 Keys, 1 Skull Keys, 30 Food",
		state.HomeChestSummary(position))
	assert.Zero(t, inventory.Materials.Get(references.Wood))
	assert.Zero(t, inventory.Provisions.SkullKeys.Get())

	// A deposit stops at the chest's cap, keeping the rest with the party.
	chest, _ := state.HomeChestAt(position)
	chest.Tools[references.Axe] = 98
	inventory.Tools.Set(references.Axe, 5)
	assert.Equal(t, 1, state.TransferHomeChest(position, HomeChestTools, true))
	assert.Equal(t, uint16(4), inventory.Tools.Get(references.Axe))

	// Taking every category empties the chest back into the party's stacks.
	moved := 0
	for _, category := range HomeChestCategories() {
		moved += state.TransferHomeChest(position, category, false)
	}
	assert.Equal(t, 20+5+95+30+4+3+2+1, moved, "the party's 4 Axes and the cap of 99 limit the Axe withdrawal")

	assert.Equal(t, uint16(99), inventory.Tools.Get(references.Axe))
	assert.Equal(t, uint16(30), inventory.Provisions.Food.Get())
	assert.Equal(t, uint16(1), inventory.Provisions.SkullKeys.Get())
	assert.Equal(t, "Chest: 4 Axe", state.HomeChestSummary(position))
}

func TestCampInsideOwnClaimMakesNoAmbushRoll(t *testing.T) {
	state, _ := newCraftingFixture(t, true)
	giveCraftingSupplies(state)
	_, err := state.StakeLandClaim("Homestead")
	require.NoError(t, err)
	scene := camping.NewCampingSceneWithCombatRef(1, NoGuardAssigned, state.MapState.PlayerLocation, testCampingMap(t))
	campingState := state.newCampingState(scene, false, false)
	state.DateTime.Hour, state.DateTime.Minute = 10, 0

	cycleDraws := func(hourChanged bool) uint64 {
		campingState.AmbushCheckHour = state.DateTime.Hour
		if hourChanged {
			campingState.AmbushCheckHour = state.DateTime.Hour - 1
		}
		before := state.GetRNG().Calls()
		require.True(t, scene.AdvanceExactCycle())
		assert.False(t, state.processSingleCampingCycle(scene, campingState))

		return state.GetRNG().Calls() - before
	}

	quiet := cycleDraws(false)
	assert.Equal(t, quiet, cycleDraws(true), "a new hour inside the claim draws no ambush roll")
	assert.Equal(t, NoAmbush, campingState.AmbushedAt)
}
