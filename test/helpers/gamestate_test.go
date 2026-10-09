package helpers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/require"
)

func TestConfigurationClassification(t *testing.T) {
	if os.Getenv("ULTIMA5_CONFIGURATION_PROBE") == "1" {
		LoadTestConfiguration(t)
		return
	}
	for _, test := range []struct {
		name, selected, strict, diagnostic string
		fail                               bool
	}{
		{name: "optional absent", diagnostic: "--- SKIP:"},
		{name: "strict absent", strict: "1", diagnostic: "optional original game data unavailable", fail: true},
		{name: "missing selected", selected: "missing", diagnostic: "resolving original data directory", fail: true},
		{name: "tiny DATA.OVL", selected: ".", diagnostic: "validating DATA.OVL", fail: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(directory, "DATA.OVL"), []byte("tiny"), 0o600))
			output, err := originaldata.TestProcess(t, "TestConfigurationClassification", directory,
				"ULTIMA5_CONFIGURATION_PROBE=1", "ULTIMA5_DATA_PATH="+test.selected,
				"ULTIMA5_DATA_DIR=", "ULTIMA5_REQUIRE_DATA="+test.strict, "HOME="+directory)
			require.Equal(t, test.fail, err != nil, "%s", output)
			require.Contains(t, string(output), test.diagnostic)
			if test.fail {
				require.NotContains(t, string(output), "--- SKIP:")
			}
		})
	}
}

func TestGameDataCacheUsesSelectedSourceAndFreshPreferences(t *testing.T) {
	directory := originaldata.Directory(t)
	t.Setenv("ULTIMA5_DATA_PATH", directory)
	first, firstRefs, err := LoadGameData()
	require.NoError(t, err)
	first.GameplayConfiguration.SmoothSurfaceMovement = true
	first.LoggingConfiguration.CategoryStates["test-isolation"] = true
	firstByte := first.RawDataOvl[0]
	first.RawDataOvl[0] ^= 0xff

	alias := filepath.Join(t.TempDir(), "alias")
	require.NoError(t, os.Symlink(directory, alias))
	t.Setenv("ULTIMA5_DATA_PATH", alias)
	second, secondRefs, err := LoadGameData()
	require.NoError(t, err)
	require.Same(t, firstRefs, secondRefs)
	require.NotSame(t, first, second)
	require.NotSame(t, first.SavedConfigData, second.SavedConfigData)
	require.NotSame(t, first.AudioConfiguration, second.AudioConfiguration)
	require.NotSame(t, first.DisplayConfiguration, second.DisplayConfiguration)
	require.NotSame(t, first.GameplayConfiguration, second.GameplayConfiguration)
	require.NotSame(t, first.LoggingConfiguration, second.LoggingConfiguration)
	require.NotSame(t, first.DebugConfiguration, second.DebugConfiguration)
	require.False(t, second.GameplayConfiguration.SmoothSurfaceMovement)
	require.Empty(t, second.LoggingConfiguration.CategoryStates)
	require.Equal(t, firstByte, second.RawDataOvl[0])
	require.ErrorIs(t, second.SaveGameplayConfiguration(), config.ErrConfigFileUnknown)

	// A different directory cannot reuse a successful cache entry. Symlinks
	// provision another source without copying copyrighted assets.
	other := t.TempDir()
	entries, err := os.ReadDir(directory)
	require.NoError(t, err)
	for _, entry := range entries {
		require.NoError(t, os.Symlink(filepath.Join(directory, entry.Name()), filepath.Join(other, entry.Name())))
	}
	t.Setenv("ULTIMA5_DATA_PATH", other)
	third, thirdRefs, err := LoadGameData()
	require.NoError(t, err)
	require.NotSame(t, firstRefs, thirdRefs)
	require.NotEqual(t, first.SavedConfigData.DataFilePath, third.SavedConfigData.DataFilePath)

	// Replace only this temporary source's symlink, never the original asset.
	dataOvl := filepath.Join(other, "DATA.OVL")
	require.NoError(t, os.Remove(dataOvl))
	_, _, err = LoadGameData()
	require.ErrorIs(t, err, os.ErrNotExist)
	require.NoError(t, os.WriteFile(dataOvl, []byte("tiny"), 0o600))
	_, _, err = LoadGameData()
	require.ErrorContains(t, err, "validating DATA.OVL")
	_, err = LoadConfiguration()
	require.ErrorContains(t, err, "validating DATA.OVL")
	require.NoError(t, os.Remove(dataOvl))
	require.NoError(t, os.Symlink(filepath.Join(directory, "DATA.OVL"), dataOvl))
	_, restoredRefs, err := LoadGameData()
	require.NoError(t, err)
	require.Same(t, thirdRefs, restoredRefs)

	broken := t.TempDir()
	t.Setenv("ULTIMA5_DATA_PATH", broken)
	_, _, err = LoadGameData()
	require.ErrorIs(t, err, os.ErrNotExist)
	require.ErrorContains(t, err, "DATA.OVL")
	require.NoError(t, os.WriteFile(filepath.Join(broken, "DATA.OVL"), []byte("tiny"), 0o600))
	_, _, err = LoadGameData()
	require.ErrorContains(t, err, "validating DATA.OVL")
	_, err = os.Stat(filepath.Join(broken, "DATA.OVL"))
	require.NoError(t, err)

	t.Setenv("ULTIMA5_DATA_PATH", directory)
	_, recovered, err := LoadGameData()
	require.NoError(t, err)
	require.Same(t, firstRefs, recovered)
}

func TestLoadGameDataPreservesReferenceLoadCause(t *testing.T) {
	directory := t.TempDir()
	directory, err := filepath.EvalSymlinks(directory)
	require.NoError(t, err)
	t.Setenv("ULTIMA5_DATA_PATH", directory)
	_, _, err = LoadGameData()
	require.ErrorIs(t, err, os.ErrNotExist)
	require.False(t, errors.Is(err, originaldata.ErrUnavailable))
	var pathError *os.PathError
	require.ErrorAs(t, err, &pathError)
	require.Equal(t, filepath.Join(directory, "DATA.OVL"), pathError.Path)

	require.NoError(t, os.WriteFile(filepath.Join(directory, "DATA.OVL"), []byte("tiny"), 0o600))
	_, _, err = LoadGameData()
	require.ErrorContains(t, err, "validating DATA.OVL")
	require.False(t, errors.Is(err, os.ErrNotExist), "parser errors must not be replaced with not-found")
	require.False(t, errors.Is(err, originaldata.ErrUnavailable))

	// Synthetic empty string tables get past DATA.OVL parsing without original
	// assets, exposing the next loader's missing-file error and wrapped cause.
	require.NoError(t, os.WriteFile(filepath.Join(directory, "DATA.OVL"), make([]byte, 0x2000), 0o600))
	_, _, err = LoadGameData()
	require.ErrorContains(t, err, "loading test references")
	require.ErrorIs(t, err, os.ErrNotExist)
	require.ErrorAs(t, err, &pathError)
	require.False(t, errors.Is(err, originaldata.ErrUnavailable))
}

func TestConfigurationDoesNotReadOrWriteHome(t *testing.T) {
	directory := originaldata.Directory(t)
	t.Setenv("ULTIMA5_DATA_PATH", directory)
	home := t.TempDir()
	t.Setenv("HOME", home)
	preferences := filepath.Join(home, ".ultima_v_redux")
	require.NoError(t, os.Mkdir(preferences, 0o700))
	filename := filepath.Join(preferences, "config.yaml")
	for _, contents := range []string{"[malformed YAML", "Gameplay:\n  Presentation:\n    SmoothSurfaceMovement: true\n"} {
		require.NoError(t, os.WriteFile(filename, []byte(contents), 0o600))
		cfg := LoadTestConfiguration(t)
		require.False(t, cfg.GameplayConfiguration.SmoothSurfaceMovement)
		require.ErrorIs(t, cfg.SaveGameplayConfiguration(), config.ErrConfigFileUnknown)
		after, err := os.ReadFile(filename)
		require.NoError(t, err)
		require.Equal(t, contents, string(after))
	}
}

func TestBootstrapSaveFixture(t *testing.T) {
	require.Len(t, LoadBootstrapSaveData(t), 4192)
}

func BenchmarkLoadGameDataWarm(b *testing.B) {
	b.Setenv("HOME", b.TempDir())
	_, warmRefs, err := LoadGameData()
	if errors.Is(err, originaldata.ErrUnavailable) && !originaldata.Required() {
		b.Skip(err)
	}
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		_, refs, err := LoadGameData()
		if err != nil {
			b.Fatal(err)
		}
		if refs != warmRefs {
			b.Fatal("warm load did not reuse references")
		}
	}
}
