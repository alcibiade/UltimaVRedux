package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFlags(t *testing.T) {
	base := []string{"--data-dir", "data", "--source", "source"}
	for _, test := range []struct {
		name  string
		args  []string
		valid bool
	}{
		{"file", []string{"--output", "import.json"}, true},
		{"slot", []string{"--save-dir", "saves", "--slot", "slot-08", "--overwrite"}, true},
		{"unicode note", []string{"--output", "import.json", "--description", strings.Repeat("é", 80)}, true},
		{"long note", []string{"--output", "import.json", "--description", strings.Repeat("é", 81)}, false},
		{"multiline note", []string{"--output", "import.json", "--description", "a\nb"}, false},
		{"both", []string{"--output", "import.json", "--save-dir", "saves", "--slot", "slot-01"}, false},
		{"partial output", []string{"--output", "import.json", "--slot", "slot-01"}, false},
		{"partial slot", []string{"--save-dir", "saves"}, false},
		{"bad slot", []string{"--save-dir", "saves", "--slot", "slot-09"}, false},
		{"missing destination", nil, false},
		{"positional", []string{"--output", "import.json", "extra"}, false},
		{"unknown", []string{"--automatic"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := parseOptions(append(append([]string(nil), base...), test.args...), io.Discard)
			if test.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
	for _, args := range [][]string{nil, {"--source", "s", "--output", "i.json"}, {"--data-dir", "d", "--output", "i.json"}} {
		_, err := parseOptions(args, io.Discard)
		require.ErrorIs(t, err, errArguments)
	}
	_, err := parseOptions([]string{"--help"}, io.Discard)
	require.ErrorIs(t, err, flag.ErrHelp)
	var help bytes.Buffer
	require.NoError(t, run(context.Background(), []string{"--help"}, io.Discard, &help))
	require.Contains(t, help.String(), "data-dir")
	require.Contains(t, help.String(), "Workbench named-library loading requires")
	for _, source := range []string{"  ", "~", "~/saves"} {
		_, err := parseOptions([]string{"--source", source, "--data-dir", "data", "--output", "save.json"}, io.Discard)
		require.ErrorIs(t, err, errArguments)
	}
}

func TestOccupiedDestinationRefusedBeforeConversion(t *testing.T) {
	source := copyOriginalFixture(t)
	data := t.TempDir() // Invalid installation: occupancy must be diagnosed first.
	for _, slotMode := range []bool{false, true} {
		root := t.TempDir()
		path := filepath.Join(root, "save.json")
		args := []string{"--source", source, "--data-dir", data}
		if slotMode {
			path = filepath.Join(root, "slot-01.json")
			args = append(args, "--save-dir", root, "--slot", "slot-01")
		} else {
			args = append(args, "--output", path)
		}
		require.NoError(t, os.WriteFile(path, []byte("existing destination"), 0o600))
		require.ErrorIs(t, run(context.Background(), args, io.Discard, io.Discard), errOccupied)
		require.Equal(t, "existing destination", string(readFile(t, path)))
	}
}

func TestInvalidInputNeverCreatesOutputOrSlotRoot(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	data := filepath.Join(root, "data")
	require.NoError(t, os.Mkdir(source, 0o700))
	require.NoError(t, os.Mkdir(data, 0o700))
	for _, name := range []string{"SAVED.GAM", "SAVED.OOL"} {
		require.NoError(t, os.WriteFile(filepath.Join(source, name), []byte("bad save"), 0o600))
	}
	for _, slotMode := range []bool{false, true} {
		args := []string{"--data-dir", data, "--source", source}
		target := filepath.Join(root, "output.json")
		if slotMode {
			target = filepath.Join(root, "new-saves")
			args = append(args, "--save-dir", target, "--slot", "slot-01")
		} else {
			args = append(args, "--output", target)
		}
		require.Error(t, run(context.Background(), args, io.Discard, io.Discard))
		_, err := os.Lstat(target)
		require.True(t, errors.Is(err, os.ErrNotExist), "destination appeared: %v", err)
	}
}

func TestDestinationPolicy(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	data := filepath.Join(root, "data")
	require.NoError(t, os.Mkdir(source, 0o700))
	require.NoError(t, os.Mkdir(data, 0o700))
	for _, target := range []string{filepath.Join(source, "save.json"), filepath.Join(data, "save.json")} {
		require.ErrorIs(t, checkInputOverlap(target, false, source, data), errDestination)
	}
	require.ErrorIs(t, checkInputOverlap(filepath.Join(root, "slot-01.json"), true, source, data), errDestination)
	require.NoError(t, checkInputOverlap(filepath.Join(root, "separate.json"), false, source, data))
	require.NoError(t, checkInputOverlap(filepath.Join(root, "source-other", "save.json"), false, source, data))
	for _, name := range []string{"SAVED.GAM", "SAVED.OOL", "DATA.OVL", "config.yaml", "CONFIG.JSON", "slot-01.json", "SLOT-08.JSON"} {
		require.ErrorIs(t, validateOutputName(name), errDestination)
	}
	_, err := destinationPath(options{output: filepath.Join(root, "missing", "save.json")})
	require.Error(t, err)
	_, err = canonicalSlotRoot(filepath.Join(root, "new-saves"))
	require.NoError(t, err)
	_, err = os.Lstat(filepath.Join(root, "new-saves"))
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestDestinationPolicyResolvesParentAliases(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	require.NoError(t, os.Mkdir(real, 0o700))
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(real, alias); err != nil {
		t.Logf("symlink alias unavailable on this host: %v", err)
		return
	}
	resolved, err := destinationPath(options{output: filepath.Join(alias, "save.json")})
	require.NoError(t, err)
	canonical, err := existingDirectory(real)
	require.NoError(t, err)
	require.ErrorIs(t, checkInputOverlap(resolved, false, canonical), errDestination)
	// Also exercise the identity fallback directly, before canonical path cleanup.
	info, err := os.Stat(real)
	require.NoError(t, err)
	inside, err := sameDirectoryAncestor(info, filepath.Join(alias, "missing", "save.json"))
	require.NoError(t, err)
	require.True(t, inside)
	_, err = canonicalSlotRoot(alias)
	require.ErrorIs(t, err, errDestination)
}
