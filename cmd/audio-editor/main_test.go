package main

import (
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/stretchr/testify/require"
)

func TestCheckValidatesWithoutWritingOrOpeningAudio(t *testing.T) {
	const manifest = "../../internal/audio/assets/library.json"
	before, err := os.ReadFile(manifest)
	require.NoError(t, err)
	require.NoError(t, run([]string{"-library", manifest, "-check"}))
	after, err := os.ReadFile(manifest)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Nil(t, audio.CurrentContext(), "read-only validation must not open an audio device")
}

func TestEditorFlagsAndMissingLibrary(t *testing.T) {
	require.NoError(t, run([]string{"-help"}))
	require.Error(t, run([]string{"-unsupported"}))
	require.Error(t, run([]string{"extra"}))
	require.Error(t, run([]string{"-library", t.TempDir() + "/missing.json", "-check"}))
}
