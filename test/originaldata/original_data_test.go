package originaldata

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDirectorySelection(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	t.Setenv("ULTIMA5_DATA_PATH", first)
	t.Setenv("ULTIMA5_DATA_DIR", second)
	got, err := ResolveDirectory()
	require.NoError(t, err)
	want, err := filepath.EvalSymlinks(first)
	require.NoError(t, err)
	require.Equal(t, want, got)

	t.Setenv("ULTIMA5_DATA_PATH", "")
	got, err = ResolveDirectory()
	require.NoError(t, err)
	want, err = filepath.EvalSymlinks(second)
	require.NoError(t, err)
	require.Equal(t, want, got)

	t.Setenv("ULTIMA5_DATA_PATH", filepath.Join(first, "missing"))
	_, err = ResolveDirectory()
	require.ErrorIs(t, err, os.ErrNotExist)
	require.False(t, errors.Is(err, ErrUnavailable), "selected absence must not become an optional skip")
	var pathError *os.PathError
	require.ErrorAs(t, err, &pathError)
}

func TestDirectoryCanonicalSource(t *testing.T) {
	directory := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	require.NoError(t, os.Symlink(directory, alias))
	t.Setenv("ULTIMA5_DATA_PATH", alias)
	got, err := ResolveDirectory()
	require.NoError(t, err)
	want, err := filepath.EvalSymlinks(directory)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestDirectoryClassification(t *testing.T) {
	if os.Getenv("ULTIMA5_DIRECTORY_PROBE") == "1" {
		Directory(t)
		return
	}
	for _, test := range []struct {
		name, selected, strict, diagnostic string
		fail                               bool
	}{
		{name: "optional absent", diagnostic: "--- SKIP:"},
		{name: "required absent", strict: "1", diagnostic: "optional original game data unavailable", fail: true},
		{name: "explicit missing", selected: "missing", diagnostic: "resolving original data directory", fail: true},
		{name: "explicit incomplete", selected: ".", diagnostic: "DATA.OVL", fail: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			directory := t.TempDir()
			output, err := TestProcess(t, "TestDirectoryClassification", directory,
				"ULTIMA5_DIRECTORY_PROBE=1", "ULTIMA5_DATA_PATH="+test.selected,
				"ULTIMA5_DATA_DIR=", "ULTIMA5_REQUIRE_DATA="+test.strict, "HOME="+directory)
			require.Equal(t, test.fail, err != nil, "%s", output)
			require.Contains(t, string(output), test.diagnostic)
			if test.fail {
				require.NotContains(t, string(output), "--- SKIP:")
			}
		})
	}
}
