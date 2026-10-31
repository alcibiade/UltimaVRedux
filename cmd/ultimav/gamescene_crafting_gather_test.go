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

	menu := topCraftingMenu(t, scene)
	assert.Equal(t, []string{"Axe: 2 Wood, 1 Stone", "Pick: 2 Wood, 2 Stone (lacking)"}, menu.ButtonTexts())
	assert.Contains(t, menu.Description(), "Thou dost carry 2 Wood, 1 Stone.")
	menu.Cancel()
	assert.Empty(t, callbacks.TimeAdvanced(), "cancelling is free")
	assert.Nil(t, scene.dialogStack.PeekTopModalDialog())
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

		require.True(t, scene.handleCraftingChord(ebiten.KeyC))
		topCraftingMenu(t, scene).ActivateButton(0)

		assert.Equal(t, uint16(1), scene.gameState.PartyState.Inventory.Tools.Get(references.Axe))
		assert.Equal(t, recipe.Minutes, sumSceneMinutes(callbacks.TimeAdvanced()), "on the deed map: %v", onDeed)
		assert.Contains(t, callbacks.Messages, "Crafted 1 Axe.")
	}
}

func TestChoosingALackingRecipeSpendsNothing(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
	scene.gameState.PartyState.Inventory.Materials.Set(references.Wood, 1)

	require.True(t, scene.handleCraftingChord(ebiten.KeyC))
	topCraftingMenu(t, scene).ActivateButton(1)

	assert.Contains(t, callbacks.Messages, "Thou dost need 2 Wood, 2 Stone.")
	assert.Equal(t, uint16(1), scene.gameState.PartyState.Inventory.Materials.Get(references.Wood))
	assert.Empty(t, callbacks.TimeAdvanced(), "a refused craft spends no turn")
	assert.Nil(t, scene.dialogStack.PeekTopModalDialog())
}
