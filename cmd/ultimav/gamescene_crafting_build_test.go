package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// newBuildScene enters a deed through a real scene and stands the party in the clear yard.
func newBuildScene(t *testing.T) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()
	scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
	enterCraftingSceneDeed(t, scene)
	scene.gameState.MapState.PlayerLocation.Position = references.DeedMapCentre
	callbacks.Reset()

	return scene, callbacks
}

func sceneTileNorth(scene *GameScene) indexes.SpriteIndex {
	position := scene.gameState.MapState.PlayerLocation.Position
	position.Y--

	return scene.gameState.GetLayeredMapByCurrentLocation().SpriteIndexByLayer(map_state.MapLayer, position)
}

// answerDirection presses a direction for the pending small-map prompt.
func answerDirection(scene *GameScene, key ebiten.Key) {
	holdCraftingKeys(scene)
	scene.keyboard.SimulateKeyPress(key)
	scene.smallMapHandleSecondaryInput()
}

// buildThroughScene answers Alt+B, picks the first category and the first piece, then a direction.
func buildThroughScene(t *testing.T, scene *GameScene, direction ebiten.Key) {
	t.Helper()
	holdCraftingKeys(scene, ebiten.KeyAltLeft)
	require.True(t, scene.handleCraftingChord(ebiten.KeyB))
	topCraftingMenu(t, scene).ActivateButton(0)
	topCraftingMenu(t, scene).ActivateButton(0)
	require.Equal(t, BuildDirectionInput, scene.secondaryKeyState)
	answerDirection(scene, direction)
}

func demolishThroughScene(t *testing.T, scene *GameScene, direction ebiten.Key) {
	t.Helper()
	holdCraftingKeys(scene, ebiten.KeyAltLeft)
	require.True(t, scene.handleCraftingChord(ebiten.KeyB))
	build := topCraftingMenu(t, scene)
	texts := build.ButtonTexts()
	build.ActivateButton(len(texts) - 1)
	require.Equal(t, DemolishDirectionInput, scene.secondaryKeyState)
	answerDirection(scene, direction)
}

func TestBuildMenuListsCarriedPiecesByCategory(t *testing.T) {
	scene, callbacks := newBuildScene(t)
	inventory := &scene.gameState.PartyState.Inventory
	inventory.Pieces.Set(references.PieceStoneWall, 6)
	inventory.Pieces.Set(references.PieceDoor, 1)
	inventory.Pieces.Set(references.PieceWorkbench, 1)

	require.True(t, scene.handleCraftingChord(ebiten.KeyB))
	build := topCraftingMenu(t, scene)
	assert.Equal(t, []string{"Stations (1 kind)", "Walls and doors (2 kinds)", "Demolish"}, build.ButtonTexts())
	build.ActivateButton(1)
	pieces := topCraftingMenu(t, scene)
	assert.Equal(t, []string{"Stone wall (6 carried)", "Door (1 carried)"}, pieces.ButtonTexts())
	pieces.Cancel()

	assert.Equal(t, []string{"Build-"}, callbacks.CommandPrompts)
	assert.Empty(t, callbacks.TimeAdvanced(), "closing the menus is free")
	assert.Nil(t, scene.dialogStack.PeekTopModalDialog())
}

func TestBuildAndDemolishThroughTheSceneSpendExactlyTheirMinutes(t *testing.T) {
	scene, callbacks := newBuildScene(t)
	scene.gameState.PartyState.Inventory.Pieces.Set(references.PieceStoneWall, 1)

	buildThroughScene(t, scene, ebiten.KeyUp)

	assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
	assert.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall), sceneTileNorth(scene))
	assert.Equal(t, references.BuildPlaceMinutes, sumSceneMinutes(callbacks.TimeAdvanced()))
	assert.Equal(t, []string{"Build-", "Place-"}, callbacks.CommandPrompts)

	callbacks.Reset()
	demolishThroughScene(t, scene, ebiten.KeyUp)

	assert.NotEqual(t, indexes.SpriteIndex(indexes.StoneBrickWall), sceneTileNorth(scene))
	assert.Equal(t, uint16(1), scene.gameState.PartyState.Inventory.Pieces.Get(references.PieceStoneWall))
	assert.Equal(t, references.BuildDemolishMinutes, sumSceneMinutes(callbacks.TimeAdvanced()))
	assert.Equal(t, []string{"Build-", "Demolish-"}, callbacks.CommandPrompts)
}

func TestARefusedOrCancelledBuildIsFree(t *testing.T) {
	scene, callbacks := newBuildScene(t)
	scene.gameState.PartyState.Inventory.Pieces.Set(references.PieceStoneWall, 1)

	demolishThroughScene(t, scene, ebiten.KeyUp)
	assert.Contains(t, callbacks.Messages, "Thou didst not build that!")

	holdCraftingKeys(scene, ebiten.KeyAltLeft)
	require.True(t, scene.handleCraftingChord(ebiten.KeyB))
	topCraftingMenu(t, scene).ActivateButton(0)
	topCraftingMenu(t, scene).ActivateButton(0)
	answerDirection(scene, ebiten.KeyEscape)

	assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
	assert.Equal(t, uint16(1), scene.gameState.PartyState.Inventory.Pieces.Get(references.PieceStoneWall))
	assert.Empty(t, callbacks.TimeAdvanced())
}

// openNorthThroughScene answers Open and the direction north, as the routed scene does.
func openNorthThroughScene(scene *GameScene) {
	scene.secondaryKeyState = OpenDirectionInput
	answerDirection(scene, ebiten.KeyUp)
}

func TestOpenOnTheStorageChestStoresAndTakesGoods(t *testing.T) {
	scene, callbacks := newBuildScene(t)
	inventory := &scene.gameState.PartyState.Inventory
	inventory.Pieces.Set(references.PieceStorageChest, 1)
	buildThroughScene(t, scene, ebiten.KeyUp)
	inventory.Materials.Set(references.Wood, 20)
	inventory.Gold.Set(300)
	karma := scene.gameState.PartyState.Karma
	callbacks.Reset()

	openNorthThroughScene(scene)
	chest := topCraftingMenu(t, scene)
	assert.Equal(t, "The chest is empty.", chest.Description())
	assert.Equal(t, []string{"Store...", "Take..."}, chest.ButtonTexts())
	chest.ActivateButton(0)
	store := topCraftingMenu(t, scene)
	assert.Equal(t, []string{"Everything", "Materials", "Tools", "Pieces", "Gold", "Food, torches, gems and keys"},
		store.ButtonTexts())
	store.ActivateButton(1)
	assert.Contains(t, callbacks.Messages, "Store materials: 20.")
	assert.Contains(t, topCraftingMenu(t, scene).Description(), "The chest holds 20 Wood.")
	topCraftingMenu(t, scene).Cancel()

	assert.Zero(t, inventory.Materials.Get(references.Wood))
	assert.Equal(t, uint16(300), inventory.Gold.Get(), "only the chosen category moved")
	assert.Equal(t, karma, scene.gameState.PartyState.Karma, "the party's own chest costs no karma")
	assert.Equal(t, game_state.DefaultSmallMapMinutesPerTurn, sumSceneMinutes(callbacks.TimeAdvanced()),
		"opening the chest and moving goods costs the ordinary turn")
	assert.Nil(t, scene.dialogStack.PeekTopModalDialog())

	callbacks.Reset()
	openNorthThroughScene(scene)
	topCraftingMenu(t, scene).Cancel()
	assert.Empty(t, callbacks.TimeAdvanced(), "closing the chest without moving anything is free")
}

func TestHoleUpOnTheOwnBedRestsLikeAnInnThroughTheScene(t *testing.T) {
	scene, callbacks := newBuildScene(t)
	state := scene.gameState
	head := references.Position{X: references.DeedMapCentre.X, Y: references.DeedMapCentre.Y - 1}
	state.PartyState.Inventory.Pieces.Set(references.PieceBedHead, 1)
	state.PartyState.Inventory.Pieces.Set(references.PieceBedFoot, 1)
	require.True(t, state.BuildDeedPiece(references.PieceBedHead, references.Up))
	state.MapState.PlayerLocation.Position = references.Position{X: head.X + 1, Y: head.Y + 1}
	require.True(t, state.BuildDeedPiece(references.PieceBedFoot, references.Up))
	state.MapState.PlayerLocation.Position = head
	avatar := state.PartyState.GetCharacter(0)
	avatar.SetHp(1)
	gold := state.PartyState.Inventory.Gold.Get()
	state.DateTime.Hour, state.DateTime.Minute = 21, 0
	callbacks.Reset()

	scene.smallMapInputHandlerWithGame(nil, ebiten.KeyH)
	for range 400 {
		if !state.EventQueueManager.HasActiveWorkflows() {
			break
		}
		state.ElapsedMs += 250
		state.EventQueueManager.Update()
	}

	require.False(t, state.EventQueueManager.HasActiveWorkflows())
	assert.Contains(t, callbacks.Messages, "Rested!")
	assert.Equal(t, avatar.GetMaxHp(), avatar.GetCurrentHp())
	assert.Equal(t, byte(6), state.DateTime.Hour)
	assert.Equal(t, gold, state.PartyState.Inventory.Gold.Get())
	assert.Equal(t, references.Position{X: head.X + 1, Y: head.Y}, state.MapState.PlayerLocation.Position,
		"the party gets out of bed to the east, as after any bed rest")
}

func TestHoleUpOnTheOwnBedWarnsThePoisonedAndEscIsFree(t *testing.T) {
	scene, callbacks := newBuildScene(t)
	state := scene.gameState
	state.PartyState.Inventory.Pieces.Set(references.PieceBedHead, 1)
	require.True(t, state.BuildDeedPiece(references.PieceBedHead, references.Up))
	state.MapState.PlayerLocation.Position.Y--
	avatar := state.PartyState.GetCharacter(0)
	avatar.SetStatus(party_state.Poisoned)
	callbacks.Reset()

	scene.smallMapInputHandlerWithGame(nil, ebiten.KeyH)
	warning := topCraftingMenu(t, scene)
	assert.Contains(t, warning.Description(), avatar.GetNameAsString()+" is poisoned.")
	warning.Cancel()

	assert.False(t, state.EventQueueManager.HasWorkflow(deedBedRestWorkflowName))
	assert.Equal(t, party_state.Poisoned, avatar.GetStatus())
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestCraftHelpMentionsBuildingOnTheDeed(t *testing.T) {
	scene, _ := newBuildScene(t)
	for _, entry := range scene.craftingCommandHelpEntries() {
		assert.Empty(t, entry.Unavailable, entry.Key)
		if entry.Key == "Alt+B" {
			assert.Contains(t, entry.NextInput, "On your deed map: place a crafted piece, or Demolish one")
		}
	}
}
