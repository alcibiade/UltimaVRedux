package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTileSnapshotKeepsLookupAnnotationsOutsideRuntimeTiles(t *testing.T) {
	data, err := json.Marshal(tileSnapshot(references.NewTileReferences()))
	require.NoError(t, err)
	var records map[string]map[string]any
	require.NoError(t, json.Unmarshal(data, &records))
	assert.Equal(t, "largest forest", records["10"]["Description"])
	assert.Equal(t, "Forest4", records["10"]["Name"])
	assert.Equal(t, float64(6), records["10"]["SpeedFactor"])
	assert.Equal(t, true, records["1"]["IsPartOfAnimation"])
	for _, record := range records {
		assert.NotContains(t, record, "AnimationIndex")
		assert.NotContains(t, record, "DontDraw")
		assert.NotContains(t, record, "IsGuessableFloor")
	}
}

func TestRemoveRetiredDumpFilesPreservesConsumedSnapshots(t *testing.T) {
	outputDir := t.TempDir()
	retired := []string{
		"DataOvl.json",
		"EnemyReferences.json",
		"EnemyReferences.yaml",
		"TalkReferences.json",
		"TileReferences.yaml",
		"dump_log.txt",
		"npc_data_index.json",
	}
	for _, name := range retired {
		require.NoError(t, os.WriteFile(filepath.Join(outputDir, name), []byte("stale"), 0600))
	}
	retained := []string{"LocationReferences.json", "NPCReferences.json", "TileReferences.json", "ConversationCatalog.json"}
	for _, name := range retained {
		require.NoError(t, os.WriteFile(filepath.Join(outputDir, name), []byte("retained"), 0600))
	}

	require.NoError(t, removeRetiredDumpFiles(outputDir))

	for _, name := range retired {
		_, err := os.Stat(filepath.Join(outputDir, name))
		assert.ErrorIs(t, err, os.ErrNotExist)
	}
	for _, name := range retained {
		data, err := os.ReadFile(filepath.Join(outputDir, name))
		require.NoError(t, err)
		assert.Equal(t, []byte("retained"), data)
	}
}

func TestDumpReferencesWritesOnlyConsumedSnapshots(t *testing.T) {
	directory := originaldata.Directory(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	configurationDirectory := filepath.Join(home, ".ultima_v_redux")
	require.NoError(t, os.Mkdir(configurationDirectory, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(configurationDirectory, "config.yaml"),
		[]byte(fmt.Sprintf("DataFilePath: %q\n", directory)), 0o600))
	outputDir := t.TempDir()

	require.NoError(t, dumpReferences(outputDir))

	entries, err := os.ReadDir(outputDir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	assert.ElementsMatch(t, []string{
		"ConversationCatalog.json",
		"LocationReferences.json",
		"NPCReferences.json",
		"TileReferences.json",
	}, names)
	data, err := os.ReadFile(filepath.Join(outputDir, "ConversationCatalog.json"))
	require.NoError(t, err)
	var documents []map[string]any
	require.NoError(t, json.Unmarshal(data, &documents))
	require.Len(t, documents, 135)
	assert.Equal(t, "original.castle.001", documents[0]["id"])
	assert.Equal(t, map[string]any{"kind": "original", "source": "castle", "dialog": float64(1)}, documents[0]["origin"])
	assert.NotContains(t, string(data), "TalkScript")
}
