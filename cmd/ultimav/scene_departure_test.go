package main

import (
	"errors"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/player_save"
)

type playerSaveCatalogTestScene struct {
	playerSaveTestScene
	summaries []player_save.SlotSummary
}

type updateErrorScene struct {
	playerSaveTestScene
	err error
}

func (scene *updateErrorScene) Update(*Game) error { return scene.err }

func (scene *playerSaveCatalogTestScene) applyPlayerSaveCatalog(summaries []player_save.SlotSummary) {
	scene.summaries = append([]player_save.SlotSummary(nil), summaries...)
}

func TestGameReturnToMainMenuUsesFreshSceneAndRepublishesSaveCatalog(t *testing.T) {
	configuration := &config.UltimaVConfiguration{}
	oldScene := &playerSaveTestScene{
		configuration: configuration, playthrough: playthroughMetadata{id: testPlaythroughID},
	}
	newScene := &playerSaveCatalogTestScene{playerSaveTestScene: playerSaveTestScene{configuration: configuration}}
	summaries := []player_save.SlotSummary{{Slot: player_save.Slot03, Status: player_save.SlotReady}}
	game := NewGame(oldScene)
	game.ConfigurePlayerSaveRepository(&playerSaveTestRepository{summaries: summaries}, nil)
	game.mainMenuSceneBuilder = func(got *config.UltimaVConfiguration) (Scene, error) {
		assert.Same(t, configuration, got)

		return newScene, nil
	}

	require.NoError(t, game.returnToMainMenu(configuration))

	assert.Same(t, newScene, game.currentScene)
	assert.Equal(t, 1, oldScene.disposals)
	assert.Equal(t, 1, newScene.activations)
	assert.Equal(t, summaries, newScene.summaries)
	assert.Empty(t, newScene.playthrough, "the fresh menu fixture must remain identity-free after activation")
}

func TestGameExitRequestTerminatesAfterSceneUpdate(t *testing.T) {
	game := NewGame(&playerSaveTestScene{})
	game.requestExit()

	assert.ErrorIs(t, game.Update(), ebiten.Termination)
}

func TestGameSceneErrorPrecedesExitRequest(t *testing.T) {
	updateErr := errors.New("scene update failed")
	game := NewGame(&updateErrorScene{err: updateErr})
	game.requestExit()

	assert.ErrorIs(t, game.Update(), updateErr)
}
