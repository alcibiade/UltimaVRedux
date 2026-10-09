package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

func TestIntegrationIntroMenuLoadsOriginalGraphicsAndCompletesPresentation(t *testing.T) {
	dataDir := originaldata.Directory(t)
	gameConfig := &config.UltimaVConfiguration{
		SavedConfigData: &config.UltimaVConfigurationFlags{DataFilePath: dataDir},
		SoundDisabled:   true,
	}
	scene, err := CreateIntroMenuScene(gameConfig)
	require.NoError(t, err)
	scene.keyboard = newPresentationTestKeyboard()
	require.NotNil(t, scene.titleFizz)
	require.NotNil(t, scene.flameFizz)

	for updates := 0; updates < 1000 && !scene.menuVisible; updates++ {
		if scene.returnView != nil {
			scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
		}
		require.NoError(t, scene.Update(nil))
	}

	assert.True(t, scene.menuVisible)
	screen := newPresentationRaster(1280, 720)
	t.Cleanup(screen.Deallocate)
	assert.NotPanics(t, func() { scene.Draw(screen) })
}

func TestIntegrationMainMenuSceneSkipsPresentation(t *testing.T) {
	dataDir := originaldata.Directory(t)
	gameConfig := &config.UltimaVConfiguration{
		SavedConfigData: &config.UltimaVConfigurationFlags{DataFilePath: dataDir},
		SoundDisabled:   true,
	}

	scene, err := CreateMainMenuScene(gameConfig)
	require.NoError(t, err)
	scene.keyboard = newPresentationTestKeyboard()

	assert.True(t, scene.menuVisible)
	assert.True(t, scene.presentation.complete)
	assert.False(t, scene.eventQueueManager.HasActiveWorkflows())
}

func TestIntegrationNarrativeIntroductionLoadsAdvancesAndReturnsToMenu(t *testing.T) {
	dataDir := originaldata.Directory(t)
	gameConfig := &config.UltimaVConfiguration{
		SavedConfigData: &config.UltimaVConfigurationFlags{DataFilePath: dataDir},
		SoundDisabled:   true,
	}
	scene, err := CreateMainMenuScene(gameConfig)
	require.NoError(t, err)
	scene.keyboard = newPresentationTestKeyboard()
	scene.nCurrentSelection = introChoiceNarrative
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	require.NoError(t, scene.Update(nil))
	require.NotNil(t, scene.narrative)
	screen := newPresentationRaster(1280, 720)
	t.Cleanup(screen.Deallocate)
	seenFrames := make(map[string]bool)
	fizzMidpointDrawn, fizzCompleteDrawn := false, false

	for updates := 0; updates < 5000 && !scene.menuVisible; updates++ {
		narrative := scene.narrative
		// Synchronization establishes the dissolve gate even between render checkpoints.
		narrative.syncFrame(scene.elapsedMs)
		frame := string(narrative.currentFrame)
		if !seenFrames[frame] {
			scene.Draw(screen)
			seenFrames[frame] = true
		}
		if narrative.fizz != nil {
			age := scene.elapsedMs - narrative.frameStartedMs
			if age >= introNarrativeFizzDuration/2 && !fizzMidpointDrawn {
				scene.Draw(screen)
				fizzMidpointDrawn = true
			}
			if age >= introNarrativeFizzDuration && !fizzCompleteDrawn {
				scene.Draw(screen)
				fizzCompleteDrawn = true
			}
			require.Equal(t, age >= introNarrativeFizzDuration, narrative.canAdvance(scene.elapsedMs))
		}
		if scene.narrative.executor.IsWaitingForAdvance() && scene.narrative.canAdvance(scene.elapsedMs) {
			scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
		}
		require.NoError(t, scene.Update(nil))
	}

	assert.True(t, scene.menuVisible)
	assert.Nil(t, scene.narrative)
	require.Len(t, seenFrames, len(introNarrativeFrames))
	require.True(t, fizzMidpointDrawn)
	require.True(t, fizzCompleteDrawn)
}
