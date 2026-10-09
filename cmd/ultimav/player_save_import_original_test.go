package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/ui/widgets"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
)

func TestIntegrationLegacyImportCreatesSelectedSlotAndStartsGame(t *testing.T) {
	directory := originaldata.Directory(t, "SAVED.GAM", "SAVED.OOL")
	source := os.Getenv("ULTIMA5_IMPORT_SOURCE")
	if source == "" {
		source = filepath.Join(directory, "SAVED.GAM")
	}
	configuration, err := config.NewUltimaVConfigurationWithPath(directory)
	require.NoError(t, err)
	configuration.SoundDisabled = true
	menu := newIntroInputTestScene()
	menu.config = configuration
	game := NewGame(menu)
	game.playthroughIDProvider = func() (string, error) { return testPlaythroughID, nil }
	require.NoError(t, configurePlayerSaveRepository(game, playerSaveStartupOptions{root: t.TempDir()}))
	browser := openIntroImportBrowser(t, game, menu, source)
	for range 6 {
		menu.keyboard.SimulateKeyPress(ebiten.KeyDown)
		require.NoError(t, game.Update())
	}
	selected, ok := browser.GetSelectedView()
	require.True(t, ok)
	require.Equal(t, string(player_save.Slot07), selected.ID)
	menu.keyboard.SimulateKeyPress(ebiten.KeyEnter)
	require.NoError(t, game.Update())
	editor, ok := menu.dialogStack.PeekTopModalDialog().(*widgets.LegacyPathEntryModal)
	require.True(t, ok, "dialog %T, selection %d", menu.dialogStack.PeekTopModalDialog(), browser.GetSelectedIndex())
	editor.SetText("b_skiff")
	editor.Submit()
	gameplay, ok := game.currentScene.(*GameScene)
	require.True(t, ok)
	t.Cleanup(gameplay.Dispose)
	record, err := game.playerSaveRepository.Load(player_save.Slot07)
	require.NoError(t, err)
	assert.Equal(t, "b_skiff", record.Envelope.Metadata.UserDescription)
	assert.Equal(t, playthroughMetadata{id: testPlaythroughID, userDescription: "b_skiff"}, gameplay.playthroughMetadata())
	assert.Zero(t, menu.dialogStack.Count())
	assertCanonicalPreparedPlayerSave(t, record.Envelope.Snapshot)
}
