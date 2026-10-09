package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
)

func TestConfigureSnapshotRepositoryUsesDefaultPersonalRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support the repository permission policy")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	projectRoot := filepath.Join(t.TempDir(), "project")
	game := NewGame(&snapshotCommandTestScene{snapshot: validCommandTestSnapshot()})
	require.NoError(t, configureSnapshotRepository(game, snapshotStartupOptions{projectRoot: projectRoot}))

	result := executeSnapshotTestCommand(t, game, debugging.CommandSaveSnapshotCapture, map[string]any{
		"source": "personal", "id": "default-root", "name": "Default Root",
	})
	require.True(t, result.Success, result.ErrorMessage)
	_, err := os.Stat(filepath.Join(home, ".ultima_v_redux", "snapshots", "default-root.json"))
	require.NoError(t, err)
}
