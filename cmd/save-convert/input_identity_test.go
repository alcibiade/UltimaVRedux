package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/stretchr/testify/require"
)

func TestCLIPreservesBackingFilesOfSelectedSaveSymlinks(t *testing.T) {
	for _, selected := range []string{"SAVED.GAM", "SAVED.OOL"} {
		for _, slotMode := range []bool{false, true} {
			t.Run(selected+map[bool]string{false: "/file", true: "/slot"}[slotMode], func(t *testing.T) {
				source := copyOriginalFixture(t)
				original := readFile(t, filepath.Join(source, selected))
				backup := t.TempDir()
				name := "original.json"
				if slotMode {
					name = "slot-01.json"
				}
				backing := filepath.Join(backup, name)
				require.NoError(t, os.WriteFile(backing, original, 0o600))
				require.NoError(t, os.Remove(filepath.Join(source, selected)))
				linkName := selected
				if selected == "SAVED.OOL" {
					linkName = "sAvEd.oOl"
				}
				if err := os.Symlink(backing, filepath.Join(source, linkName)); err != nil {
					t.Logf("symlink creation unavailable on this host: %v; hardlink identity coverage runs separately", err)
					return
				}
				args := []string{"--source", source, "--data-dir", t.TempDir(), "--overwrite"}
				if slotMode {
					args = append(args, "--save-dir", backup, "--slot", "slot-01")
				} else {
					args = append(args, "--output", backing)
				}
				err := run(context.Background(), args, io.Discard, io.Discard)
				require.ErrorIs(t, err, errDestination)
				require.ErrorContains(t, err, "aliases a selected original save file")
				require.Equal(t, original, readFile(t, backing))
				require.Equal(t, original, readFile(t, filepath.Join(source, linkName)))
				entries, err := os.ReadDir(backup)
				require.NoError(t, err)
				require.Len(t, entries, 1, "rejection must precede lock and temporary-file creation")
				// Symlink inputs remain usable when output is a distinct file, even
				// when that file contains exactly the same bytes as the selected input.
				separate := filepath.Join(backup, "separate.json")
				require.NoError(t, os.WriteFile(separate, original, 0o600))
				_, _, err = prepareDestination(context.Background(), options{
					source: source, dataDirectory: t.TempDir(), output: separate, overwrite: true,
				})
				require.NoError(t, err)
			})
		}
	}
}

func TestPrepareDestinationRejectsHardlinksToEitherInput(t *testing.T) {
	source := copyOriginalFixture(t)
	// Resolve the exact case-insensitive OOL sibling, never a guessed uppercase path.
	require.NoError(t, os.Rename(filepath.Join(source, "SAVED.OOL"), filepath.Join(source, "saved.ool")))
	for _, selected := range []string{"SAVED.GAM", "saved.ool"} {
		t.Run(selected, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "alias.json")
			require.NoError(t, os.Link(filepath.Join(source, selected), path))
			_, _, err := prepareDestination(context.Background(), options{
				source: source, dataDirectory: t.TempDir(), output: path, overwrite: true,
			})
			require.ErrorIs(t, err, errDestination)
			require.ErrorContains(t, err, "aliases a selected original save file")
		})
	}
}

func TestPublicationRechecksCapturedInputIdentities(t *testing.T) {
	source := copyOriginalFixture(t)
	input := filepath.Join(source, "SAVED.GAM")
	original := readFile(t, input)
	for _, slotMode := range []bool{false, true} {
		root := t.TempDir()
		parsed := options{source: source, dataDirectory: t.TempDir(), overwrite: true}
		path := filepath.Join(root, "original.json")
		if slotMode {
			path = filepath.Join(root, "slot-01.json")
			parsed.saveDirectory, parsed.slot = root, "slot-01"
		} else {
			parsed.output = path
		}
		// Equal bytes ensure a content-revision check alone cannot detect this swap.
		require.NoError(t, os.WriteFile(path, original, 0o600))
		target, _, err := prepareDestination(context.Background(), parsed)
		require.NoError(t, err)
		require.NoError(t, os.Remove(path))
		require.NoError(t, os.Link(input, path))
		err = target.publish(context.Background(), save_snapshot.Envelope{})
		require.ErrorIs(t, err, errDestination, "input identity must be checked before encoding or persisting")
		if !slotMode {
			// Standalone publication repeats the identity check under its lock.
			err = publishFile(context.Background(), path, target.revision, []byte("replacement"),
				repositoryfs.SyncDirectory, target.inputs...)
			require.ErrorIs(t, err, errDestination)
		}
		require.Equal(t, original, readFile(t, input))
		require.Equal(t, original, readFile(t, path))
	}
}

func TestSelectedObjectsSiblingRejectsMissingAndAmbiguousNames(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "SAVED.GAM"), []byte("gam"), 0o600))
	_, err := selectedObjectsSibling(context.Background(), root)
	require.ErrorIs(t, err, errDestination)
	first := filepath.Join(root, "SAVED.OOL")
	second := filepath.Join(root, "saved.ool")
	require.NoError(t, os.WriteFile(first, []byte("ool"), 0o600))
	require.NoError(t, os.WriteFile(second, []byte("ool"), 0o600))
	firstInfo, err := os.Stat(first)
	require.NoError(t, err)
	secondInfo, err := os.Stat(second)
	require.NoError(t, err)
	_, err = selectedObjectsSibling(context.Background(), root)
	if os.SameFile(firstInfo, secondInfo) {
		require.NoError(t, err)
	} else {
		require.ErrorIs(t, err, errDestination)
		require.ErrorContains(t, err, "ambiguous")
	}
}
