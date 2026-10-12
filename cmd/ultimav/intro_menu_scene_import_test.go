package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestIntroImportUsesActiveFolderInsteadOfPendingOrHistory(t *testing.T) {
	active := filepath.Dir(writePlayerSaveLegacyBundle(t, false))
	pending := filepath.Dir(writePlayerSaveLegacyBundle(t, false))
	history := filepath.Dir(writePlayerSaveLegacyBundle(t, false))
	scene := newIntroInputTestScene()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	startup := config.LoadStartupConfiguration()
	require.NoError(t, startup.ConfigurationError)
	scene.config = startup.Configuration
	scene.config.SavedConfigData.DataFilePath = active
	file, err := scene.config.ConfigurationFilePath()
	require.NoError(t, err)
	contents := []byte("DataFilePath: '" + pending + "'\nLegacyImport:\n  LastSuccessfulDirectory: '" + history + "'\n")
	require.NoError(t, os.WriteFile(file, contents, 0o600))
	require.NoError(t, scene.config.RefreshPersistedConfiguration())
	game, repository := introImportTestGame(scene)
	resolved := make(chan string, 1)
	scene.prepareLegacySave = func(ctx context.Context, path string) (preparedLegacyPlayerSave, error) {
		resolved <- path
		return game.prepareLegacyPlayerSaveContext(ctx, path)
	}
	scene.startActiveLegacySaveImport(game)
	browser := awaitIntroImportDestination(t, game, scene)
	assert.Equal(t, filepath.Join(active, "SAVED.GAM"), <-resolved)
	view, selected := browser.GetSelectedView()
	require.True(t, selected)
	assert.Contains(t, view.Details, widgets.SaveSlotDetail{Label: "Source folder", Value: active})
	assert.Empty(t, repository.saves, "destination confirmation still owns the write")
	after, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, contents, after)
}

func TestIntroImportMissingPairCanRetryActiveFolder(t *testing.T) {
	legacyPath := writePlayerSaveLegacyBundle(t, false)
	directory := filepath.Dir(legacyPath)
	oolPath := filepath.Join(directory, "SAVED.OOL")
	ool, err := os.ReadFile(oolPath)
	require.NoError(t, err)
	require.NoError(t, os.Remove(oolPath))
	scene := newIntroInputTestScene()
	scene.config.SavedConfigData.DataFilePath = directory
	game, repository := introImportTestGame(scene)
	scene.startActiveLegacySaveImport(game)
	require.Eventually(t, func() bool {
		require.NoError(t, game.Update())
		return scene.dialogStack.Count() == 1
	}, time.Second, time.Millisecond)
	menu, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.ButtonListModal)
	require.True(t, ok)
	assert.Equal(t, []string{"Check Again", "Back"}, menu.ButtonTexts())
	assert.Empty(t, repository.saves)
	require.NoError(t, os.WriteFile(oolPath, ool, 0o600))
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	require.NoError(t, game.Update())
	awaitIntroImportDestination(t, game, scene)
	assert.Empty(t, repository.saves)
}

func TestIntroImportRejectsUnsetActiveFolderEvenWithWorkingDirectorySave(t *testing.T) {
	legacyPath := writePlayerSaveLegacyBundle(t, false)
	t.Chdir(filepath.Dir(legacyPath))
	scene := newIntroInputTestScene()
	game, repository := introImportTestGame(scene)
	scene.prepareLegacySave = func(context.Context, string) (preparedLegacyPlayerSave, error) {
		t.Error("an unset active path must never prepare a working-directory save")
		return preparedLegacyPlayerSave{}, nil
	}
	scene.startActiveLegacySaveImport(game)
	assert.False(t, scene.legacyPrepareActive)
	assert.IsType(t, &widgets.ButtonListModal{}, scene.dialogStack.PeekTopModalDialog())
	assert.Empty(t, repository.saves)
}

func TestIntroImportEscapeWinsOverReadyCompletion(t *testing.T) {
	scene := newIntroInputTestScene()
	scene.config.SavedConfigData.DataFilePath = filepath.Dir(writePlayerSaveLegacyBundle(t, false))
	game, repository := introImportTestGame(scene)
	scene.startActiveLegacySaveImport(game)
	require.Eventually(t, func() bool { return len(scene.legacyPrepareResults) == 1 }, time.Second, time.Millisecond)
	scene.elapsedMs = 20_000
	scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	require.NoError(t, game.Update())
	assert.False(t, scene.legacyPrepareActive)
	assert.Zero(t, scene.dialogStack.Count())
	assert.Equal(t, scene.elapsedMs, scene.menuIdleStartedMs)
	assert.Empty(t, repository.saves)
}

func TestIntroImportRetryDiscardsLatePreparationAndCapturesFunction(t *testing.T) {
	scene := newIntroInputTestScene()
	scene.config.SavedConfigData.DataFilePath = filepath.Dir(writePlayerSaveLegacyBundle(t, false))
	game, repository := introImportTestGame(scene)
	started, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	scene.prepareLegacySave = func(context.Context, string) (preparedLegacyPlayerSave, error) {
		close(started)
		<-release
		close(finished)
		return preparedLegacyPlayerSave{}, nil
	}
	scene.startActiveLegacySaveImport(game)
	<-started
	oldResults := scene.legacyPrepareResults
	scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	require.NoError(t, game.Update())
	scene.prepareLegacySave = game.prepareLegacyPlayerSaveContext
	scene.startActiveLegacySaveImport(game)
	assert.NotEqual(t, oldResults, scene.legacyPrepareResults)
	close(release)
	<-finished
	browser := awaitIntroImportDestination(t, game, scene)
	assert.Empty(t, repository.saves)
	require.NoError(t, game.Update())
	assert.Same(t, browser, scene.dialogStack.PeekTopModalDialog())
}

func TestIntroImportDisposeDiscardsReadyPreparation(t *testing.T) {
	scene := newIntroInputTestScene()
	scene.config.SavedConfigData.DataFilePath = filepath.Dir(writePlayerSaveLegacyBundle(t, false))
	game, repository := introImportTestGame(scene)
	scene.startActiveLegacySaveImport(game)
	require.Eventually(t, func() bool { return len(scene.legacyPrepareResults) == 1 }, time.Second, time.Millisecond)
	scene.Dispose()
	assert.False(t, scene.legacyPrepareActive)
	assert.Zero(t, scene.dialogStack.Count())
	assert.Empty(t, repository.saves)
}

func introImportTestGame(scene *IntroMenuScene) (*Game, *playerSaveTestRepository) {
	repository := &playerSaveTestRepository{
		records:   make(map[player_save.SlotID]player_save.Record),
		summaries: []player_save.SlotSummary{{Slot: player_save.Slot01, Status: player_save.SlotEmpty}},
	}
	game := NewGame(scene)
	game.ConfigurePlayerSaveRepository(repository, func(*config.UltimaVConfiguration, save_snapshot.SaveSnapshot) (Scene, error) {
		return &playerSaveTestScene{}, nil
	})
	return game, repository
}

func awaitIntroImportDestination(t *testing.T, game *Game, scene *IntroMenuScene) *widgets.SaveSlotBrowser {
	t.Helper()
	require.Eventually(t, func() bool {
		require.NoError(t, game.Update())
		_, ready := scene.dialogStack.PeekTopModalDialog().(*widgets.SaveSlotBrowser)
		return ready
	}, time.Second, time.Millisecond)
	return scene.dialogStack.PeekTopModalDialog().(*widgets.SaveSlotBrowser)
}
