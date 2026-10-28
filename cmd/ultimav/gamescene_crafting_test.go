package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func newCraftingChordScene(t *testing.T, enabled bool) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()

	return newCraftingChordSceneAt(t, references.BritanniaUnderworld, enabled, ebiten.KeyAltLeft)
}

// newCraftingChordSceneAt builds a scene whose keyboard holds the given keys, with the party at
// 130,100 on the overworld, or 15,15 in a town.
func newCraftingChordSceneAt(t *testing.T, location references.Location, enabled bool, held ...ebiten.Key) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()
	x, y := 130, 100
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
	callbacks.Reset()
	scene := newSurfaceCompletionTestScene(gameState)
	keys := map[ebiten.Key]bool{}
	for _, key := range held {
		keys[key] = true
	}
	scene.keyboard = input.NewKeyboardWithKeyStateSource(0, func(key ebiten.Key) bool { return keys[key] })

	return scene, callbacks
}

func topCraftingMenu(t *testing.T, scene *GameScene) *widgets.ButtonListModal {
	t.Helper()
	menu, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.ButtonListModal)
	require.True(t, ok, "no crafting menu is open")

	return menu
}

func TestAltCCraftThroughTheMenuSpendsTheDocumentedThirtyMinutes(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, true)
	scene.gameState.PartyState.Inventory.AddMaterials(references.MaterialCost{references.Wood: 3, references.Stone: 2})

	scene.largeMapInputHandler(ebiten.KeyC)
	menu := topCraftingMenu(t, scene)
	assert.Contains(t, menu.Description(), "Not enough for: Pick.")
	menu.ActivateButton(0)

	assert.Equal(t, uint16(1), scene.gameState.PartyState.Inventory.Tools.Get(references.Axe))
	assert.Equal(t, references.CraftRecipes()[0].Minutes, sumSceneMinutes(callbacks.TimeAdvanced()))
}

func TestAltCRefusedRecipeSpendsNothing(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, true)

	scene.largeMapInputHandler(ebiten.KeyC)
	menu := topCraftingMenu(t, scene)
	assert.Contains(t, menu.Description(), "Not enough for: Axe, Pick.")
	menu.ActivateButton(0)

	assert.Contains(t, callbacks.Messages, "You need 3 Wood, 2 Stone.")
	assert.Zero(t, scene.gameState.PartyState.Inventory.Tools.Get(references.Axe))
	assert.Empty(t, callbacks.TimeAdvanced(), "an unaffordable recipe spends no time")
}

func TestCraftingChordsAnswerNotHereOffTheSurfaceOrAboardAVehicle(t *testing.T) {
	for _, testCase := range []struct {
		name  string
		setup func(*game_state.GameState)
	}{
		{name: "underworld", setup: func(state *game_state.GameState) {
			state.MapState.PlayerLocation.Floor = references.FloorNumber(references.UNDERWORLD)
		}},
		{name: "on horseback", setup: func(state *game_state.GameState) {
			state.PartyVehicle = *map_units.NewNPCFriendlyVehiceNewRef(references.HorseVehicle,
				state.MapState.PlayerLocation.Position, state.MapState.PlayerLocation.Floor)
		}},
	} {
		for _, key := range []ebiten.Key{ebiten.KeyG, ebiten.KeyC, ebiten.KeyB} {
			t.Run(testCase.name+"/"+key.String(), func(t *testing.T) {
				scene, callbacks := newCraftingChordScene(t, true)
				testCase.setup(scene.gameState)

				scene.largeMapInputHandler(key)

				assert.Equal(t, []string{"Not here!"}, callbacks.Messages)
				assert.Empty(t, callbacks.CommandPrompts)
				assert.Nil(t, scene.dialogStack.PeekTopModalDialog(), "no menu opens off the surface")
				assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
				assert.Empty(t, callbacks.TimeAdvanced())
			})
		}
	}
}

func TestCraftingChordNeedsAltAlone(t *testing.T) {
	scene, callbacks := newCraftingChordSceneAt(t, references.BritanniaUnderworld, true, ebiten.KeyAltLeft, ebiten.KeyControlLeft)

	scene.largeMapInputHandler(ebiten.KeyG)

	assert.Equal(t, []string{"What?"}, callbacks.Messages)
	assert.Empty(t, callbacks.CommandPrompts, "Ctrl+Alt+G is neither Gather nor Get")
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestCraftingChordsInTownNeverBecomeGetCastOrBoard(t *testing.T) {
	for _, enabled := range []bool{true, false} {
		scene, callbacks := newCraftingChordSceneAt(t, references.Britain, enabled, ebiten.KeyAltLeft)
		want := "Not here!"
		if !enabled {
			want = "What?"
		}
		for _, key := range []ebiten.Key{ebiten.KeyG, ebiten.KeyC, ebiten.KeyB} {
			callbacks.Reset()
			assert.True(t, scene.craftingChordOffLargeMap(key))
			assert.Equal(t, []string{want}, callbacks.Messages)
			assert.Empty(t, callbacks.CommandPrompts)
		}
		assert.False(t, scene.craftingChordOffLargeMap(ebiten.KeyM), "other Alt keys keep their town meaning")
	}
}

func sumSceneMinutes(minutes []int) int {
	total := 0
	for _, minute := range minutes {
		total += minute
	}

	return total
}

func TestAltGOpensGatherPromptWhenCraftingEnabled(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, true)

	scene.largeMapInputHandler(ebiten.KeyG)

	assert.Equal(t, []string{"Gather-"}, callbacks.CommandPrompts)
	assert.Equal(t, GatherDirectionInput, scene.secondaryKeyState)
	assert.True(t, scene.hasUncommittedSurfaceDirection(references.LargeMapType), "Esc cancels Gather for free")
}

func TestAltGAnswersWhatWhenCraftingDisabled(t *testing.T) {
	scene, callbacks := newCraftingChordScene(t, false)

	scene.largeMapInputHandler(ebiten.KeyG)

	assert.Empty(t, callbacks.CommandPrompts, "Alt+G must not fall through to Get")
	assert.Equal(t, []string{"What?"}, callbacks.Messages)
	assert.Equal(t, PrimaryInput, scene.secondaryKeyState)
}

func TestCraftingChordsListedInSurfaceHelpOnly(t *testing.T) {
	for _, context := range []references.GeneralMapType{references.LargeMapType, references.SmallMapType} {
		for _, entry := range commandHelpEntries(context) {
			if entry.Key != "Alt+G" {
				continue
			}
			assert.Equal(t, context != references.LargeMapType, entry.Unavailable != "")
		}
	}
}
