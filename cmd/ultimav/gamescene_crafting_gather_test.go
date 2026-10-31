package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// sceneDeedNode returns the first node of a kind on the scene's active deed map and puts the
// party just south of it.
func sceneDeedNode(t *testing.T, scene *GameScene, kind references.DeedResourceKind) game_state.DeedResourceNodeState {
	t.Helper()
	deed, ok := scene.gameState.ActiveDeed()
	require.True(t, ok)
	for _, node := range scene.gameState.DeedResourceNodes(deed.ID) {
		if node.Kind == kind {
			scene.gameState.MapState.PlayerLocation.Position = references.Position{X: node.Position.X, Y: node.Position.Y + 1}

			return node
		}
	}
	t.Fatalf("no %s node", kind.Name())

	return game_state.DeedResourceNodeState{}
}

// gatherThroughScene answers Alt+G and its direction prompt as the routed scene does.
func gatherThroughScene(scene *GameScene, direction ebiten.Key) {
	holdCraftingKeys(scene, ebiten.KeyAltLeft)
	scene.handleCraftingChord(ebiten.KeyG)
	holdCraftingKeys(scene)
	scene.keyboard.SimulateKeyPress(direction)
	scene.smallMapHandleSecondaryInput()
}

func TestGatherThroughTheSceneSpendsExactlyItsMinutes(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
	enterCraftingSceneDeed(t, scene)
	sceneDeedNode(t, scene, references.DeedResourceForage)
	callbacks.Reset()

	gatherThroughScene(scene, ebiten.KeyUp)

	assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
	assert.Equal(t, []string{"Gather-"}, callbacks.CommandPrompts)
	assert.Positive(t, scene.gameState.PartyState.Inventory.Materials.Get(references.Forage))
	assert.Equal(t, references.DeedNodeRuleFor(references.DeedResourceForage).Minutes, sumSceneMinutes(callbacks.TimeAdvanced()),
		"the attempt costs its stated minutes, the ordinary turn included")
}

func TestARefusedOrCancelledGatherIsFree(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
	enterCraftingSceneDeed(t, scene)
	sceneDeedNode(t, scene, references.DeedResourceOre)
	callbacks.Reset()

	gatherThroughScene(scene, ebiten.KeyUp)
	assert.Contains(t, callbacks.Messages, "Thou dost need a Pick!")

	holdCraftingKeys(scene, ebiten.KeyAltLeft)
	scene.handleCraftingChord(ebiten.KeyG)
	holdCraftingKeys(scene)
	scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	scene.smallMapHandleSecondaryInput()

	assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestCraftMenuMarksLackingRowsInText(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
	scene.gameState.PartyState.Inventory.Materials.Set(references.Wood, 2)
	scene.gameState.PartyState.Inventory.Materials.Set(references.Stone, 1)

	require.True(t, scene.handleCraftingChord(ebiten.KeyC))

	categories := topCraftingMenu(t, scene)
	assert.Equal(t, "Tools (1 ready)", categories.ButtonTexts()[0])
	assert.Equal(t, "Refined materials (none ready)", categories.ButtonTexts()[1])
	assert.Contains(t, categories.Description(), "Thou dost carry 2 Wood, 1 Stone.")
	categories.ActivateButton(0)
	menu := topCraftingMenu(t, scene)
	assert.Equal(t, []string{"Axe: 2 Wood, 1 Stone", "Pick: 2 Wood, 2 Stone (lacking)"}, menu.ButtonTexts())
	menu.Cancel()
	topCraftingMenu(t, scene).Cancel()
	assert.Empty(t, callbacks.TimeAdvanced(), "cancelling is free")
	assert.Nil(t, scene.dialogStack.PeekTopModalDialog())
}

// craftThroughScene answers Alt+C and picks a category row, then a recipe row.
func craftThroughScene(t *testing.T, scene *GameScene, category references.CraftCategory, recipe references.CraftRecipeID) {
	t.Helper()
	require.True(t, scene.handleCraftingChord(ebiten.KeyC))
	categories := topCraftingMenu(t, scene)
	row := 0
	for index, candidate := range references.CraftCategories() {
		if candidate == category {
			row = index
		}
	}
	categories.ActivateButton(row)
	recipes := topCraftingMenu(t, scene)
	index := 0
	for _, candidate := range references.CraftRecipes() {
		if candidate.Category != category {
			continue
		}
		if candidate.ID == recipe {
			recipes.ActivateButton(index)

			return
		}
		index++
	}
	t.Fatalf("recipe %d is not in category %s", recipe, category.Name())
}

func TestCraftingThroughTheSceneSpendsExactlyTheRecipeMinutes(t *testing.T) {
	recipe, _ := references.CraftRecipeByID(references.RecipeAxe)
	for _, onDeed := range []bool{false, true} {
		scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
		if onDeed {
			enterCraftingSceneDeed(t, scene)
			holdCraftingKeys(scene, ebiten.KeyAltLeft)
		}
		scene.gameState.PartyState.Inventory.Materials.Set(references.Wood, 2)
		scene.gameState.PartyState.Inventory.Materials.Set(references.Stone, 1)
		callbacks.Reset()

		craftThroughScene(t, scene, references.CraftCategoryTools, references.RecipeAxe)

		assert.Equal(t, uint16(1), scene.gameState.PartyState.Inventory.Tools.Get(references.Axe))
		assert.Equal(t, recipe.Minutes, sumSceneMinutes(callbacks.TimeAdvanced()), "on the deed map: %v", onDeed)
		assert.Contains(t, callbacks.Messages, "Crafted 1 Axe.")
	}
}

func TestChoosingALackingRecipeSpendsNothing(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
	scene.gameState.PartyState.Inventory.Materials.Set(references.Wood, 1)

	craftThroughScene(t, scene, references.CraftCategoryTools, references.RecipePick)

	assert.Contains(t, callbacks.Messages, "Thou dost need 2 Wood, 2 Stone.")
	assert.Equal(t, uint16(1), scene.gameState.PartyState.Inventory.Materials.Get(references.Wood))
	assert.Empty(t, callbacks.TimeAdvanced(), "a refused craft spends no turn")
	assert.Nil(t, scene.dialogStack.PeekTopModalDialog())
}

func TestEveryCraftRowFitsTheMenu(t *testing.T) {
	for _, recipe := range references.CraftRecipes() {
		for _, option := range []game_state.CraftRecipeOption{
			{Recipe: recipe, Affordable: true, StationReady: true},
			{Recipe: recipe, Affordable: false, StationReady: true},
			{Recipe: recipe, Affordable: false, StationReady: false},
		} {
			assert.LessOrEqual(t, len(craftMenuRowLabel(option)), craftMenuRowMaxChars, craftMenuRowLabel(option))
		}
	}
	scroll, _ := references.CraftRecipeByID(references.RecipeScrollInManiCorp)
	assert.Equal(t, "In Mani Corp: SA Gi Ga SS BM MR, 1 Planks (lacking)",
		craftMenuRowLabel(game_state.CraftRecipeOption{Recipe: scroll, StationReady: true}))
	assert.Equal(t, "In Mani Corp (needs Alchemy table)", craftMenuRowLabel(game_state.CraftRecipeOption{Recipe: scroll}),
		"a row too long away from its station drops its inputs")
	sword, _ := references.CraftRecipeByID(references.RecipeLongSword)
	assert.Equal(t, "Long Sword: 4 Ingots, 1 Planks (needs Forge)", craftMenuRowLabel(game_state.CraftRecipeOption{Recipe: sword}))
}

func TestAlchemyMenusExplainTheReagentShortNames(t *testing.T) {
	scene, _ := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
	inventory := &scene.gameState.PartyState.Inventory
	inventory.Materials.Set(references.Planks, 4)
	inventory.Materials.Set(references.Wood, 9)
	for _, reagent := range references.ReagentCatalog() {
		inventory.Reagent.Set(reagent, 0)
	}
	inventory.Reagent.Set(references.Garlic, 3)

	craftThroughScene(t, scene, references.CraftCategoryPotions, references.RecipeBluePotion)

	// Away from an Alchemy table the craft is refused; reopen the menu to read it.
	require.True(t, scene.handleCraftingChord(ebiten.KeyC))
	categories := topCraftingMenu(t, scene)
	for index, category := range references.CraftCategories() {
		if category == references.CraftCategoryPotions {
			categories.ActivateButton(index)
		}
	}
	menu := topCraftingMenu(t, scene)
	assert.Contains(t, menu.Description(), "Thou dost carry 4 Planks, 3 Ga.")
	assert.Contains(t, menu.Description(), "Key: SA Sulfurous Ash, Gi Ginseng, Ga Garlic")
	assert.Equal(t, "Blue Potion: SA Gi Ga (needs Alchemy table)", menu.ButtonTexts()[0])
}
