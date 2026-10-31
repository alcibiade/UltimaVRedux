package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

// craftingSceneSquare is open overworld land far from settlements and moongates.
var craftingSceneSquare = references.Position{X: 130, Y: 100}

// newCraftingChordScene builds a scene whose keyboard holds the given keys, with the party at
// 130,100 on grass on the overworld, or 15,15 in a town, and 1000 gold.
func newCraftingChordScene(t *testing.T, location references.Location, enabled bool, held ...ebiten.Key) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()
	x, y := int(craftingSceneSquare.X), int(craftingSceneSquare.Y)
	if location != references.BritanniaUnderworld {
		x, y = 15, 15
	}
	gameState, callbacks := game_state.NewIntegrationTestBuilder(t).
		WithLocation(location).
		WithPlayerAt(x, y).
		WithRandomSeed(41).
		WithSystemCallbacks().
		Build()
	configuration := *gameState.GameConfig
	gameplay := *configuration.GameplayConfiguration
	gameplay.CraftingEnabled = enabled
	configuration.GameplayConfiguration = &gameplay
	gameState.GameConfig = &configuration
	gameState.DebugOptions.MonsterGen = false
	gameState.CurrentNPCAIController.GetNpcs().ClearAllMapUnits()
	gameState.DateTime.Hour, gameState.DateTime.Minute = 10, 0
	gameState.PartyState.Inventory.Gold.Set(1000)
	if location == references.BritanniaUnderworld {
		overworld := gameState.MapState.LayeredMaps.GetLayeredMap(references.LargeMapType, 0)
		for dy := -2; dy <= 2; dy++ {
			for dx := -2; dx <= 2; dx++ {
				square := references.Position{X: craftingSceneSquare.X + references.Coordinate(dx), Y: craftingSceneSquare.Y + references.Coordinate(dy)}
				overworld.SetTileByLayer(map_state.MapLayer, &square, indexes.Grass)
			}
		}
	}
	callbacks.Reset()
	scene := newSurfaceCompletionTestScene(gameState)
	holdCraftingKeys(scene, held...)

	return scene, callbacks
}

func holdCraftingKeys(scene *GameScene, held ...ebiten.Key) {
	keys := map[ebiten.Key]bool{}
	for _, key := range held {
		keys[key] = true
	}
	scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return keys[key] })
}

func topCraftingMenu(t *testing.T, scene *GameScene) *widgets.ButtonListModal {
	t.Helper()
	menu, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.ButtonListModal)
	require.True(t, ok, "no crafting menu is open")

	return menu
}

func sumSceneMinutes(minutes []int) int {
	total := 0
	for _, minute := range minutes {
		total += minute
	}

	return total
}

// enterCraftingSceneDeed stakes a deed on the party's square and enters its map.
func enterCraftingSceneDeed(t *testing.T, scene *GameScene) {
	t.Helper()
	_, err := scene.gameState.StakeDeed("Home")
	require.NoError(t, err)
	require.Equal(t, game_state.EnterLargeMapTransition, scene.gameState.ResolveEnterLargeMap())
	for range 200 {
		if !scene.gameState.EventQueueManager.HasActiveWorkflows() {
			break
		}
		scene.gameState.ElapsedMs += 1000
		scene.gameState.EventQueueManager.Update()
	}
	require.True(t, scene.gameState.IsOnDeedMap())
}

func TestCraftingChordsAnswerWhatWhileTheOptionIsOff(t *testing.T) {
	for _, location := range []references.Location{references.BritanniaUnderworld, references.Britain} {
		for _, key := range []ebiten.Key{ebiten.KeyG, ebiten.KeyC, ebiten.KeyB} {
			scene, callbacks := newCraftingChordScene(t, location, false, ebiten.KeyAltLeft)

			require.True(t, scene.handleCraftingChord(key))

			assert.Equal(t, []string{"What?"}, callbacks.Messages, "%s %s", location, key)
			assert.Empty(t, callbacks.CommandPrompts, "never Get, Cast or Board")
			assert.Nil(t, scene.dialogStack.PeekTopModalDialog())
			assert.Empty(t, callbacks.TimeAdvanced())
		}
	}
}

func TestCraftingChordNeedsAltAlone(t *testing.T) {
	for _, extra := range []ebiten.Key{ebiten.KeyControlLeft, ebiten.KeyShiftLeft, ebiten.KeyMetaLeft} {
		scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft, extra)

		require.True(t, scene.handleCraftingChord(ebiten.KeyB), "the chord is still owned, so B never boards")

		assert.Equal(t, []string{"What?"}, callbacks.Messages)
		assert.Nil(t, scene.dialogStack.PeekTopModalDialog())
	}
}

func TestPlainKeysAndOtherAltKeysKeepTheirMeaning(t *testing.T) {
	scene, _ := newCraftingChordScene(t, references.Britain, true)
	for _, key := range []ebiten.Key{ebiten.KeyG, ebiten.KeyC, ebiten.KeyB} {
		assert.False(t, scene.handleCraftingChord(key), "plain %s is Get, Cast or Board", key)
	}
	holdCraftingKeys(scene, ebiten.KeyAltLeft)
	assert.False(t, scene.handleCraftingChord(ebiten.KeyM))
}

func TestCraftingChordsInTownAnswerNotHere(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyG, ebiten.KeyC, ebiten.KeyB} {
		scene, callbacks := newCraftingChordScene(t, references.Britain, true, ebiten.KeyAltLeft)

		require.True(t, scene.handleCraftingChord(key))

		assert.Equal(t, []string{"Not here!"}, callbacks.Messages)
		assert.Empty(t, callbacks.CommandPrompts)
		assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
		assert.Empty(t, callbacks.TimeAdvanced())
	}
}

func TestCraftingChordsOnTheOverworld(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)

	require.True(t, scene.handleCraftingChord(ebiten.KeyG))
	require.True(t, scene.handleCraftingChord(ebiten.KeyC))

	assert.Equal(t, []string{"Not here!"}, callbacks.Messages, "gathering is deed-map only")
	assert.Equal(t, []string{"Craft-"}, callbacks.CommandPrompts, "crafting tools works on the overworld")
	topCraftingMenu(t, scene)
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestCraftingChordsOffTheSurfaceOrAboardAVehicleAnswerNotHere(t *testing.T) {
	for name, setup := range map[string]func(*game_state.GameState){
		"underworld": func(state *game_state.GameState) {
			state.MapState.PlayerLocation.Floor = references.FloorNumber(references.UNDERWORLD)
		},
		"on horseback": func(state *game_state.GameState) {
			state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle,
				state.MapState.PlayerLocation.Position, state.MapState.PlayerLocation.Floor)
		},
	} {
		scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
		setup(scene.gameState)

		require.True(t, scene.handleCraftingChord(ebiten.KeyB))

		assert.Equal(t, []string{"Not here!"}, callbacks.Messages, name)
		assert.Nil(t, scene.dialogStack.PeekTopModalDialog(), name)
	}
}

func TestAltBOffersADeedAndEscIsFree(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)

	require.True(t, scene.handleCraftingChord(ebiten.KeyB))

	menu := topCraftingMenu(t, scene)
	assert.Equal(t, []string{"Stake a deed here (200 gold)"}, menu.ButtonTexts())
	assert.Contains(t, menu.Description(), "A deed costs 200 gold; you carry 1000.")
	assert.Equal(t, []string{"Build-"}, callbacks.CommandPrompts)
	menu.Cancel()
	assert.Empty(t, callbacks.TimeAdvanced(), "cancelling the menu is free")
	assert.Empty(t, scene.gameState.Deeds())
}

func TestStakingThroughTheSceneSpendsExactlyTheOrdinaryTurn(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
	finish := scene.craftingCompletion(references.LargeMapType)

	state := scene.resolveDeedName("Greenhollow", true)
	finish(state)

	require.Equal(t, surfaceCommandCompleted, state)
	require.Len(t, scene.gameState.Deeds(), 1)
	assert.Equal(t, game_state.DefaultLargeMapMinutesPerTurn, sumSceneMinutes(callbacks.TimeAdvanced()),
		"a stake costs the ordinary 2-minute surface turn and nothing more")
	assert.Equal(t, uint16(1000-references.DeedGoldCost), scene.gameState.PartyState.Inventory.Gold.Get())
}

func TestRefusedOrCancelledNamesAreFree(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
	finish := scene.craftingCompletion(references.LargeMapType)

	finish(scene.resolveDeedName("Home", false))
	finish(scene.resolveDeedName("   ", true))

	assert.Equal(t, []string{"Cancelled", "A deed needs a name."}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())
	assert.Empty(t, scene.gameState.Deeds())
}

func TestAltBRefusesUnsuitableSquaresWithTheirReason(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
	scene.gameState.PartyState.Inventory.Gold.Set(10)

	require.True(t, scene.handleCraftingChord(ebiten.KeyB))

	assert.Equal(t, []string{"A deed costs 200 gold!"}, callbacks.Messages)
	assert.Nil(t, scene.dialogStack.PeekTopModalDialog())
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestAltBOnTheOwnHutPointsToEnter(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
	_, err := scene.gameState.StakeDeed("Home")
	require.NoError(t, err)
	callbacks.Reset()

	require.True(t, scene.handleCraftingChord(ebiten.KeyB))

	assert.Equal(t, []string{"Home is yours. Enter to go in."}, callbacks.Messages)
	assert.Nil(t, scene.dialogStack.PeekTopModalDialog())
}

func TestCraftingChordsOnADeedMap(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft)
	enterCraftingSceneDeed(t, scene)
	callbacks.Reset()

	require.True(t, scene.handleCraftingChord(ebiten.KeyG))
	assert.Equal(t, GatherDirectionInput, scene.secondaryKeyState, "Gather asks for a direction")
	scene.secondaryKeyState = PrimaryInput
	require.True(t, scene.handleCraftingChord(ebiten.KeyC))
	topCraftingMenu(t, scene).Cancel()
	require.True(t, scene.handleCraftingChord(ebiten.KeyB))

	assert.Equal(t, []string{"Gather-", "Craft-"}, callbacks.CommandPrompts)
	assert.Equal(t, []string{"Nothing to build yet!"}, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())
	assert.Equal(t, "Home", scene.mapBorderLabel())
}

func TestCraftingHelpExplainsAvailability(t *testing.T) {
	scene, _ := newCraftingChordScene(t, references.BritanniaUnderworld, false)
	for _, entry := range scene.craftingCommandHelpEntries() {
		assert.Equal(t, "Enable Crafting and Deeds in Configure.", entry.Unavailable, entry.Key)
	}

	scene, _ = newCraftingChordScene(t, references.BritanniaUnderworld, true)
	entries := scene.craftingCommandHelpEntries()
	require.Len(t, entries, 3)
	assert.Equal(t, "Alt+G", entries[0].Key)
	assert.Equal(t, "Not available here.", entries[0].Unavailable)
	assert.Empty(t, entries[1].Unavailable)
	assert.Empty(t, entries[2].Unavailable)
}

func TestPlayerSaveLabelNamesDeedMaps(t *testing.T) {
	assert.Equal(t, "Your land", playerSaveLocationLabel(int(references.DeedLocation), 0))
}
