package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/pkg/input"
)

func altHeld(key ebiten.Key) bool {
	return key == ebiten.KeyAltLeft || key == ebiten.KeyG
}

func newCraftingChordScene(t *testing.T, enabled bool) (*GameScene, *game_state.MockSystemCallbacks) {
	t.Helper()
	gameState, callbacks := game_state.NewIntegrationTestBuilder(t).
		WithLocation(references.BritanniaUnderworld).
		WithPlayerAt(100, 100).
		WithSystemCallbacks().
		Build()
	configuration := *gameState.GameConfig
	gameplay := *configuration.GameplayConfiguration
	gameplay.CraftingEnabled = enabled
	configuration.GameplayConfiguration = &gameplay
	gameState.GameConfig = &configuration
	callbacks.Reset()

	return &GameScene{gameState: gameState, keyboard: input.NewKeyboardWithKeyStateSource(0, altHeld)}, callbacks
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
