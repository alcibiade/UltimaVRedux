package game_state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/require"
)

func TestIntegrationBuilderDataClassification(t *testing.T) {
	if mode := os.Getenv("ULTIMA5_BUILDER_PROBE"); mode != "" {
		if mode == "legacy" {
			if getBaseGameReferences() == nil {
				originaldata.Check(t, originaldata.ErrUnavailable)
			}
			return
		}
		builder := NewIntegrationTestBuilder(t)
		if mode == "require" {
			builder.FailIfNoGameData()
		}
		builder.Build()
		return
	}
	for _, test := range []struct {
		name, selected, strict, mode, diagnostic string
		fail                                     bool
	}{
		{name: "optional absent", mode: "optional", diagnostic: "--- SKIP:"},
		{name: "strict absent", mode: "optional", strict: "1", diagnostic: "optional original game data unavailable", fail: true},
		{name: "builder required absent", mode: "require", diagnostic: "Failed to load Ultima V game data", fail: true},
		{name: "explicit missing", mode: "optional", selected: "missing", diagnostic: "resolving original data directory", fail: true},
		{name: "explicit tiny DATA.OVL", mode: "optional", selected: ".", diagnostic: "validating DATA.OVL", fail: true},
		{name: "legacy optional absent", mode: "legacy", diagnostic: "--- SKIP:"},
		{name: "legacy strict absent", mode: "legacy", strict: "1", diagnostic: "optional original game data unavailable", fail: true},
		{name: "legacy explicit tiny DATA.OVL", mode: "legacy", selected: ".", diagnostic: "validating DATA.OVL", fail: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(directory, "DATA.OVL"), []byte("tiny"), 0o600))
			output, err := originaldata.TestProcess(t, "TestIntegrationBuilderDataClassification", directory,
				"ULTIMA5_BUILDER_PROBE="+test.mode, "ULTIMA5_DATA_PATH="+test.selected,
				"ULTIMA5_DATA_DIR=", "ULTIMA5_REQUIRE_DATA="+test.strict, "HOME="+directory)
			require.Equal(t, test.fail, err != nil, "%s", output)
			require.Contains(t, string(output), test.diagnostic)
			if test.fail {
				require.NotContains(t, string(output), "--- SKIP:")
			}
		})
	}
}

func TestLegacyReferencesConcurrentCache(t *testing.T) {
	originaldata.Directory(t)
	refs := getBaseGameReferences()
	cfg := cachedIntegrationCfg
	for range 8 {
		t.Run("same source", func(t *testing.T) {
			t.Parallel()
			require.Same(t, refs, getBaseGameReferences())
			require.Same(t, cfg, cachedIntegrationCfg)
		})
	}
}
