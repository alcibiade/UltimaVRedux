package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/appicon"
)

const atlasPath = "../../internal/sprites/assets/transparent_u5_tiles.png"

func TestRunWritesOneResourcePerWindowsTarget(t *testing.T) {
	output := t.TempDir()
	require.NoError(t, run(atlasPath, output))

	atlas, err := os.ReadFile(atlasPath)
	require.NoError(t, err)
	entries, err := os.ReadDir(output)
	require.NoError(t, err)
	require.Len(t, entries, len(appicon.WindowsTargets()))
	for _, target := range appicon.WindowsTargets() {
		want, err := appicon.WindowsResource(atlas, target.Machine)
		require.NoError(t, err)
		got, err := os.ReadFile(filepath.Join(output, target.FileName()))
		require.NoError(t, err)
		assert.Equal(t, want, got, target.FileName())
	}
}

func TestRunReportsAMissingAtlas(t *testing.T) {
	err := run(filepath.Join(t.TempDir(), "missing.png"), t.TempDir())
	require.ErrorContains(t, err, "reading sprite atlas")
}
