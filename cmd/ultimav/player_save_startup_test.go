package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/player_save"
)

func TestConfigurePlayerSaveRepositoryUsesDefaultRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support the repository permission policy")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	scene := &playerSaveTestScene{
		snapshot: validCommandTestSnapshot(), playthrough: playthroughMetadata{id: testPlaythroughID},
	}
	game := NewGame(scene)
	require.NoError(t, configurePlayerSaveRepository(game, playerSaveStartupOptions{}))

	_, err := game.savePlayerSaveSlot(player_save.Slot01, "", "")
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(home, ".ultima_v_redux", "saves", "slot-01.json"))
}
