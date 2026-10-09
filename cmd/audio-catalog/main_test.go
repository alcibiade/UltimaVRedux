package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/audio"
)

func TestExportUsesActualAssetsAndPreservesUnreviewedState(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "inventory.json")
	guide := filepath.Join(directory, "guide.md")
	manifestPath := filepath.Join("../..", audio.LibraryManifestRelativePath)
	before, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	require.NoError(t, run([]string{"-source", "../..", "-output", output, "-guide", guide}))
	after, err := os.ReadFile(manifestPath)
	require.NoError(t, err)
	require.Equal(t, before, after, "export must preserve manually authored entries")
	data, err := os.ReadFile(output)
	require.NoError(t, err)
	var catalog audio.Catalog
	require.NoError(t, json.Unmarshal(data, &catalog))
	require.NoError(t, catalog.Validate())
	require.Len(t, catalog.SourceTreeSHA256, 64)
	for _, moment := range catalog.Moments {
		require.Equal(t, audio.CatalogUnreviewed, moment.ListeningReview)
		if moment.ID == "sfx/SoundPushObject" {
			require.Equal(t, "playback_request_found", moment.TriggerCoverage)
			require.Equal(t, "push_pull_01", moment.AssetID)
		}
	}
	text, err := os.ReadFile(guide)
	require.NoError(t, err)
	require.Contains(t, string(text), "sfx/SoundMagicUnlock")
	require.Contains(t, string(text), "file: null")
	require.Contains(t, string(text), "sfx/SoundPushObject")
}

func TestExportRejectsBadSourceAndConflictingOutputs(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "inventory.json")
	guide := filepath.Join(directory, "guide.md")
	require.Error(t, run([]string{"-source", directory, "-output", output, "-guide", guide}))
	_, err := os.Stat(output)
	require.True(t, os.IsNotExist(err), "failed source inspection must not write a partial inventory")
	require.ErrorIs(t, run([]string{"-output", output, "-guide", output}), audio.ErrAudioCatalog)
	require.ErrorIs(t, run([]string{"unexpected"}), audio.ErrAudioCatalog)
	require.ErrorIs(t, run([]string{"-source", "../..", "-output", filepath.Join("../..", audio.LibraryManifestRelativePath),
		"-guide", guide}), audio.ErrAudioCatalog)
}
