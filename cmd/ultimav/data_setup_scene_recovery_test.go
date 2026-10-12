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
	"github.com/bradhannah/UltimaVRedux/internal/originaldata"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
)

func TestDataSetupSavedSettingsNeedFreshConfirmationAndUseExactDisplayedFile(t *testing.T) {
	scene, configuration := newDataSetupTestScene(t, dataSetupFirstRun)
	file, err := configuration.ConfigurationFilePath()
	require.NoError(t, err)
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "DATA.OVL"), nil, 0o600))
	contents := []byte("# keep formatting\nDataFilePath: '" + directory + "'\nAudio:\n  MusicVolume: 2\n")
	require.NoError(t, os.WriteFile(file, contents, 0o600))
	otherHome := t.TempDir()
	t.Setenv("HOME", otherHome)
	t.Setenv("USERPROFILE", otherHome)
	require.NoError(t, os.Mkdir(filepath.Join(otherHome, ".ultima_v_redux"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(otherHome, ".ultima_v_redux", "config.yaml"), []byte("invalid: ["), 0o600))
	var ready *config.UltimaVConfiguration
	scene.onReady = func(_ *Game, loaded *config.UltimaVConfiguration) error { ready = loaded; return nil }
	game := &Game{}
	scene.checkSavedSettings(game)
	updateDataSetupUntil(t, scene, game, func() bool { return scene.dialogStack.Count() == 1 })
	require.Nil(t, ready)
	menu, ok := scene.dialogStack.PeekTopModalDialog().(*widgets.ButtonListModal)
	require.True(t, ok)
	assert.Contains(t, menu.ButtonTexts(), "Use This Installation")
	scene.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	require.NoError(t, scene.Update(game))
	updateDataSetupUntil(t, scene, game, func() bool { return ready != nil })
	assert.Equal(t, directory, ready.SavedConfigData.DataFilePath)
	assert.Equal(t, 2, ready.AudioConfiguration.MusicVolume)
	loadedFile, err := ready.ConfigurationFilePath()
	require.NoError(t, err)
	assert.Equal(t, file, loadedFile)
	after, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, contents, after, "accepting an already saved path must not rewrite the file")
}

func TestDataSetupInvalidTypedPreferencesBlockCandidateUntilRepaired(t *testing.T) {
	scene, configuration := newDataSetupTestScene(t, dataSetupFirstRun)
	file, err := configuration.ConfigurationFilePath()
	require.NoError(t, err)
	directory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(directory, "DATA.OVL"), nil, 0o600))
	invalid := []byte("Audio:\n  MusicVolume: [1, 2]\n")
	require.NoError(t, os.WriteFile(file, invalid, 0o600))
	candidate := originaldata.Candidate{Path: directory, Source: originaldata.SourceGOG, Label: "GOG"}
	game := &Game{}
	var ready *config.UltimaVConfiguration
	scene.onReady = func(_ *Game, loaded *config.UltimaVConfiguration) error { ready = loaded; return nil }
	scene.useCandidate(game, candidate)
	assert.False(t, scene.validating)
	require.Error(t, scene.configurationError)
	assert.Nil(t, ready)
	after, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, invalid, after)
	scene.checkSavedSettings(game)
	updateDataSetupUntil(t, scene, game, func() bool { return !scene.validating })
	assert.Nil(t, ready)
	require.NoError(t, os.WriteFile(file, []byte("Audio:\n  MusicVolume: 3\nUnknown: retained\n"), 0o600))
	scene.useCandidate(game, candidate)
	updateDataSetupUntil(t, scene, game, func() bool { return ready != nil })
	assert.NoError(t, scene.configurationError)
	assert.Equal(t, 3, ready.AudioConfiguration.MusicVolume)
	assert.Equal(t, directory, configuration.PersistedDataFilePath())
	after, err = os.ReadFile(file)
	require.NoError(t, err)
	assert.Contains(t, string(after), "retained")
}

func TestDataSetupEscapeWinsOverReadyLoadWithoutPersisting(t *testing.T) {
	scene, configuration := newDataSetupTestScene(t, dataSetupFirstRun)
	file, err := configuration.ConfigurationFilePath()
	require.NoError(t, err)
	before, err := os.ReadFile(file)
	require.NoError(t, err)
	scene.onReady = func(*Game, *config.UltimaVConfiguration) error {
		t.Error("cancelled load must not continue startup")
		return nil
	}
	scene.beginValidation(dataSetupLoadSelected, "Loading...", func(context.Context) dataSetupValidationResult {
		return dataSetupValidationResult{candidate: originaldata.Candidate{Path: t.TempDir()}, playable: configuration}
	})
	require.Eventually(t, func() bool { return len(scene.validationResults) == 1 }, time.Second, time.Millisecond)
	scene.keyboard.SimulateKeyPress(ebiten.KeyEscape)
	require.NoError(t, scene.Update(&Game{}))
	assert.False(t, scene.validating)
	after, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestDataSetupSearchRetryUsesCapturedFunctionAndChannel(t *testing.T) {
	scene, _ := newDataSetupTestScene(t, dataSetupFirstRun)
	started, release, finished := make(chan struct{}), make(chan struct{}), make(chan struct{})
	scene.discover = func(context.Context, originaldata.DiscoveryOptions) originaldata.Report {
		close(started)
		<-release
		close(finished)
		return originaldata.Report{Candidates: []originaldata.Candidate{{Path: "stale", Label: "stale"}}}
	}
	scene.startSearch()
	<-started
	oldResults := scene.searchResults
	scene.discover = func(context.Context, originaldata.DiscoveryOptions) originaldata.Report {
		return originaldata.Report{Candidates: []originaldata.Candidate{{Path: "fresh", Label: "fresh"}}}
	}
	scene.startSearch()
	assert.NotEqual(t, oldResults, scene.searchResults)
	close(release)
	<-finished
	updateDataSetupUntil(t, scene, &Game{}, func() bool { return !scene.searching })
	require.Len(t, scene.report.Candidates, 1)
	assert.Equal(t, "fresh", scene.report.Candidates[0].Path)
	scene.Dispose()
}
