package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// newWorkshopFixture stands the party in the yard of a grassland deed among a Workbench, a
// Forge and an Alchemy table, all within reach.
func newWorkshopFixture(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks, _ := newBuildFixture(t)
	buildAt(t, state, references.PieceWorkbench, references.Position{X: yard.X - 1, Y: yard.Y - 1})
	buildAt(t, state, references.PieceForge, references.Position{X: yard.X, Y: yard.Y - 1})
	buildAt(t, state, references.PieceAlchemyTable, references.Position{X: yard.X + 1, Y: yard.Y - 1})
	callbacks.Reset()

	return state, callbacks
}

// requireCraft makes a recipe and checks it cost exactly its minutes with the deed-map turn.
func requireCraft(t *testing.T, state *GameState, callbacks *MockSystemCallbacks, id references.CraftRecipeID) {
	t.Helper()
	callbacks.Reset()
	require.NoError(t, state.Craft(id))
	recipe, _ := references.CraftRecipeByID(id)
	assert.Equal(t, recipe.Minutes-DefaultSmallMapMinutesPerTurn, gatherMinutes(callbacks), recipe.Name)
}

func TestTheForgeMakesALongSwordThatCanBeReadied(t *testing.T) {
	state, callbacks := newWorkshopFixture(t)
	inventory := &state.PartyState.Inventory
	inventory.Materials.Set(references.Ingots, 5)
	inventory.Materials.Set(references.Planks, 1)
	swords := inventory.Equipment.Get(references.LongSword)

	requireCraft(t, state, callbacks, references.RecipeLongSword)

	assert.Equal(t, swords+1, inventory.Equipment.Get(references.LongSword))
	assert.Equal(t, uint16(1), inventory.Materials.Get(references.Ingots))
	assert.Zero(t, inventory.Materials.Get(references.Planks))
	assert.Contains(t, callbacks.Messages, "Crafted 1 Long Sword.")
	avatar := state.PartyState.GetCharacter(0)
	avatar.SetEquippedItem(references.SlotWeapon, references.LongSword)
	assert.Equal(t, references.LongSword, avatar.GetEquippedItem(references.SlotWeapon))
}

func TestEquipmentRecipesSample(t *testing.T) {
	state, callbacks := newWorkshopFixture(t)
	inventory := &state.PartyState.Inventory
	for _, testCase := range []struct {
		recipe    references.CraftRecipeID
		equipment references.Equipment
		made      uint16
	}{
		{references.RecipeDagger, references.Dagger, 1},
		{references.RecipeHalberd, references.Halberd, 1},
		{references.RecipeBow, references.Bow, 1},
		{references.RecipeArrows, references.Arrows, 10},
		{references.RecipeQuarrels, references.Quarrels, 10},
		{references.RecipeLeatherArmour, references.LeatherArmour, 1},
		{references.RecipePlateMail, references.PlateMail, 1},
		{references.RecipeIronHelm, references.IronHelm, 1},
		{references.RecipeLargeShield, references.LargeShield, 1},
	} {
		inventory.Materials.Set(references.Ingots, 20)
		inventory.Materials.Set(references.Planks, 20)
		inventory.Materials.Set(references.Forage, 20)
		before := inventory.Equipment.Get(testCase.equipment)
		requireCraft(t, state, callbacks, testCase.recipe)
		assert.Equal(t, before+testCase.made, inventory.Equipment.Get(testCase.equipment), testCase.equipment.String())
	}
}

func TestEquipmentOutputStopsAtTheSmallStackCap(t *testing.T) {
	state, callbacks := newWorkshopFixture(t)
	inventory := &state.PartyState.Inventory
	inventory.Materials.Set(references.Ingots, 2)
	inventory.Materials.Set(references.Planks, 2)
	inventory.Equipment.Set(references.Arrows, party_state.ItemQuantitySmallMax-9)

	require.ErrorIs(t, state.Craft(references.RecipeArrows), ErrCraftNoRoomToOutput)

	assert.Equal(t, []string{"Thou canst carry no more Arrows!"}, callbacks.Messages)
	assert.Equal(t, uint16(2), inventory.Materials.Get(references.Ingots), "a refusal spends nothing")
	assert.Empty(t, callbacks.TimeAdvanced())
	inventory.Equipment.Set(references.Arrows, party_state.ItemQuantitySmallMax-10)
	requireCraft(t, state, callbacks, references.RecipeArrows)
	assert.Equal(t, party_state.ItemQuantitySmallMax, inventory.Equipment.Get(references.Arrows))
}

func TestItemRecipesNeedTheirStations(t *testing.T) {
	for _, testCase := range []struct {
		recipe  references.CraftRecipeID
		message string
	}{
		{references.RecipeLongSword, "Thou dost need a Forge nearby!"},
		{references.RecipeBow, "Thou dost need a Workbench nearby!"},
		{references.RecipeKey, "Thou dost need a Forge nearby!"},
		{references.RecipeBluePotion, "Thou dost need an Alchemy table nearby!"},
		{references.RecipeScrollVasLor, "Thou dost need an Alchemy table nearby!"},
		{references.RecipeRefineBlackPearl, "Thou dost need an Alchemy table nearby!"},
	} {
		state, callbacks, _ := newBuildFixture(t)
		inventory := &state.PartyState.Inventory
		inventory.Materials.Set(references.Ingots, 20)
		inventory.Materials.Set(references.Planks, 20)
		for _, reagent := range references.ReagentCatalog() {
			inventory.Reagent.Set(reagent, 10)
		}

		require.ErrorIs(t, state.Craft(testCase.recipe), ErrCraftNeedsStation)

		assert.Equal(t, []string{testCase.message}, callbacks.Messages)
		assert.Equal(t, uint16(20), inventory.Materials.Get(references.Ingots))
		assert.Equal(t, uint16(10), inventory.Reagent.Get(references.SulfurAsh))
		assert.Empty(t, callbacks.TimeAdvanced())
	}
}

func TestTorchesAndKeysAreProvisions(t *testing.T) {
	state, callbacks := newWorkshopFixture(t)
	inventory := &state.PartyState.Inventory
	inventory.Materials.Set(references.Wood, 1)
	inventory.Materials.Set(references.Forage, 1)
	inventory.Materials.Set(references.Ingots, 1)
	inventory.Provisions.Torches.Set(0)
	inventory.Provisions.Keys.Set(0)

	requireCraft(t, state, callbacks, references.RecipeTorches)
	requireCraft(t, state, callbacks, references.RecipeKey)

	assert.Equal(t, uint16(2), inventory.Provisions.Torches.Get())
	assert.Equal(t, uint16(1), inventory.Provisions.Keys.Get())
	assert.Zero(t, inventory.Materials.Get(references.Ingots))
}

func TestTheAlchemyTableMixesAPotion(t *testing.T) {
	state, callbacks := newWorkshopFixture(t)
	inventory := &state.PartyState.Inventory
	inventory.Reagent.Set(references.Ginseng, 1)
	inventory.Reagent.Set(references.SpiderSilk, 2)
	inventory.Potions.Set(references.Yellow, 0)

	requireCraft(t, state, callbacks, references.RecipeYellowPotion)

	assert.Equal(t, uint16(1), inventory.Potions.Get(references.Yellow))
	assert.Zero(t, inventory.Reagent.Get(references.Ginseng))
	assert.Equal(t, uint16(1), inventory.Reagent.Get(references.SpiderSilk))
	assert.Contains(t, callbacks.Messages, "Crafted 1 Yellow Potion.")

	callbacks.Reset()
	require.ErrorIs(t, state.Craft(references.RecipeYellowPotion), ErrCraftLacksInputs)
	assert.Equal(t, []string{"Thou dost need 1 Ginseng, 1 Spider Silk."}, callbacks.Messages)

	inventory.Reagent.Set(references.Ginseng, 1)
	inventory.Potions.Set(references.Yellow, party_state.ItemQuantitySmallMax)
	callbacks.Reset()
	require.ErrorIs(t, state.Craft(references.RecipeYellowPotion), ErrCraftNoRoomToOutput)
	assert.Equal(t, []string{"Thou canst carry no more Yellow Potion!"}, callbacks.Messages)
	assert.Equal(t, uint16(1), inventory.Reagent.Get(references.Ginseng))
}

func TestTheAlchemyTableScribesAScroll(t *testing.T) {
	state, callbacks := newWorkshopFixture(t)
	inventory := &state.PartyState.Inventory
	for _, reagent := range references.ReagentCatalog() {
		inventory.Reagent.Set(reagent, 1)
	}
	inventory.Materials.Set(references.Planks, 1)
	inventory.Scrolls.Set(references.ScrollInManiCorp, 0)

	requireCraft(t, state, callbacks, references.RecipeScrollInManiCorp)

	assert.Equal(t, uint16(1), inventory.Scrolls.Get(references.ScrollInManiCorp))
	assert.Zero(t, inventory.Materials.Get(references.Planks))
	for _, reagent := range []references.Reagent{
		references.SulfurAsh, references.Ginseng, references.Garlic, references.SpiderSilk, references.BloodMoss,
		references.MandrakeRoot,
	} {
		assert.Zero(t, inventory.Reagent.Get(reagent), reagent.Name())
	}
	assert.Equal(t, uint16(1), inventory.Reagent.Get(references.BlackPearl), "In Mani Corp needs no Black Pearl")
	assert.Contains(t, callbacks.Messages, "Crafted 1 In Mani Corp Scroll.")

	// Scrolls are large stacks.
	inventory.Materials.Set(references.Planks, 1)
	inventory.Reagent.Set(references.SulfurAsh, 1)
	inventory.Reagent.Set(references.MandrakeRoot, 1)
	inventory.Scrolls.Set(references.ScrollVasLor, party_state.ItemQuantityLargeMax)
	callbacks.Reset()
	require.ErrorIs(t, state.Craft(references.RecipeScrollVasLor), ErrCraftNoRoomToOutput)
	inventory.Scrolls.Set(references.ScrollVasLor, party_state.ItemQuantitySmallMax)
	requireCraft(t, state, callbacks, references.RecipeScrollVasLor)
	assert.Equal(t, party_state.ItemQuantitySmallMax+1, inventory.Scrolls.Get(references.ScrollVasLor))
}

func TestRefiningTurnsThreeCommonReagentsIntoOneRarer(t *testing.T) {
	state, callbacks := newWorkshopFixture(t)
	inventory := &state.PartyState.Inventory
	inventory.Reagent.Set(references.SulfurAsh, 7)
	inventory.Reagent.Set(references.BlackPearl, 0)

	requireCraft(t, state, callbacks, references.RecipeRefineBlackPearl)
	requireCraft(t, state, callbacks, references.RecipeRefineBlackPearl)

	assert.Equal(t, uint16(2), inventory.Reagent.Get(references.BlackPearl))
	assert.Equal(t, uint16(1), inventory.Reagent.Get(references.SulfurAsh))
	callbacks.Reset()
	require.ErrorIs(t, state.Craft(references.RecipeRefineBlackPearl), ErrCraftLacksInputs)
	assert.Equal(t, []string{"Thou dost need 3 Sulfurous Ash."}, callbacks.Messages)

	inventory.Reagent.Set(references.SulfurAsh, 3)
	inventory.Reagent.Set(references.BlackPearl, party_state.ItemQuantitySmallMax)
	callbacks.Reset()
	require.ErrorIs(t, state.Craft(references.RecipeRefineBlackPearl), ErrCraftNoRoomToOutput)
	assert.Equal(t, uint16(3), inventory.Reagent.Get(references.SulfurAsh))

	inventory.Reagent.Set(references.BloodMoss, 3)
	inventory.Reagent.Set(references.Ginseng, 3)
	requireCraft(t, state, callbacks, references.RecipeRefineNightShade)
	requireCraft(t, state, callbacks, references.RecipeRefineMandrakeRoot)
	assert.Equal(t, uint16(1), inventory.Reagent.Get(references.NightShade))
	assert.Equal(t, uint16(1), inventory.Reagent.Get(references.MandrakeRoot))
}

func TestStorageChestKeepsArmsPotionsAndScrollsButNotReadiedArms(t *testing.T) {
	state, _, _ := newBuildFixture(t)
	buildNorth(t, state, references.PieceStorageChest)
	chest := northOfYard()
	inventory := &state.PartyState.Inventory
	avatar := state.PartyState.GetCharacter(0)
	avatar.SetEquippedItem(references.SlotWeapon, references.LongSword)
	inventory.Equipment.Set(references.LongSword, 3)
	readied := state.readiedArmsQuantity(references.LongSword)
	require.Equal(t, uint16(1), readied)
	for equipment := references.LeatherHelm; equipment <= references.Ankh; equipment++ {
		if equipment != references.LongSword {
			inventory.Equipment.Set(equipment, state.readiedArmsQuantity(equipment))
		}
	}
	inventory.Potions.Set(references.Red, 4)
	inventory.Scrolls.Set(references.ScrollAnTym, 2)

	assert.Equal(t, 2, state.TransferStorageChest(chest, StorageChestEquipment, true), "the readied sword stays")
	assert.Equal(t, 4, state.TransferStorageChest(chest, StorageChestPotions, true))
	assert.Equal(t, 2, state.TransferStorageChest(chest, StorageChestScrolls, true))
	assert.Equal(t, uint16(1), inventory.Equipment.Get(references.LongSword))
	assert.Equal(t, references.LongSword, avatar.GetEquippedItem(references.SlotWeapon))
	contents, _ := state.StorageChestContents(chest)
	assert.Equal(t, uint16(2), contents.Equipment[references.LongSword])
	assert.Equal(t, uint16(4), contents.Potions[references.Red])
	assert.Equal(t, uint16(2), contents.Scrolls[references.ScrollAnTym])
	assert.Equal(t, "The chest holds 2 Long Sword, 4 Red Potion, 2 An Tym Scroll.", state.StorageChestSummary(chest))

	restored := saveAndReload(t, state)
	saved, ok := restored.StorageChestContents(chest)
	require.True(t, ok)
	assert.Equal(t, contents, saved, "a save keeps the chest's arms, potions and scrolls")

	assert.Equal(t, 2, state.TransferStorageChest(chest, StorageChestEquipment, false))
	assert.Equal(t, uint16(3), inventory.Equipment.Get(references.LongSword))
}
