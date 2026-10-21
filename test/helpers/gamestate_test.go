package helpers

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/test/originaldata"
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
