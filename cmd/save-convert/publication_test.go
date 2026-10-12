package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/player_save"
	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
	"github.com/stretchr/testify/require"
)

func TestStandaloneCreateReplaceAndRefusal(t *testing.T) {
	root := t.TempDir()
	if runtime.GOOS != "windows" {
		require.NoError(t, os.Chmod(root, 0o755))
	}
	path := filepath.Join(root, "import.json")
	empty, err := inspectDestination(path, "", false)
	require.NoError(t, err)
	require.Empty(t, empty.revision)
	require.NoError(t, publishFile(context.Background(), path, empty.revision, []byte("first"), repositoryfs.SyncDirectory))
	_, err = inspectDestination(path, "", false)
	require.ErrorIs(t, err, errOccupied)
	reviewed, err := inspectDestination(path, "", true)
	require.NoError(t, err)
	require.NotEmpty(t, reviewed.revision)
	require.NoError(t, publishFile(context.Background(), path, reviewed.revision, []byte("replacement"), repositoryfs.SyncDirectory))
	require.ErrorIs(t, publishFile(context.Background(), path, reviewed.revision, []byte("stale"), repositoryfs.SyncDirectory), errStale)
	require.Equal(t, "replacement", string(readFile(t, path)))
	if runtime.GOOS != "windows" {
		info, err := os.Stat(root)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o755), info.Mode().Perm())
		info, err = os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, os.FileMode(0o600), info.Mode().Perm())
	}
	entries, err := os.ReadDir(root)
	require.NoError(t, err)
	require.Len(t, entries, 2, "only the output and stable lock file should remain")
}

func TestStandaloneConcurrentOneWinner(t *testing.T) {
	for _, existing := range []bool{false, true} {
		root := t.TempDir()
		path := filepath.Join(root, "import.json")
		if existing {
			require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))
		}
		reviewed, err := inspectDestination(path, "", true)
		require.NoError(t, err)
		start := make(chan struct{})
		results := make(chan error, 2)
		for _, content := range []string{"first", "second"} {
			go func() {
				<-start
				results <- publishFile(context.Background(), path, reviewed.revision, []byte(content), repositoryfs.SyncDirectory)
			}()
		}
		close(start)
		first, second := <-results, <-results
		if first == nil {
			require.ErrorIs(t, second, errStale)
		} else {
			require.ErrorIs(t, first, errStale)
			require.NoError(t, second)
		}
		require.Contains(t, []string{"first", "second"}, string(readFile(t, path)))
	}
}

func TestStandaloneSyncFailureReportsPossiblePublication(t *testing.T) {
	path := filepath.Join(t.TempDir(), "import.json")
	syncErr := errors.New("injected sync failure")
	err := publishFile(context.Background(), path, "", []byte("complete"), func(string) error { return syncErr })
	require.ErrorIs(t, err, syncErr)
	require.Contains(t, err.Error(), "complete output may exist")
	require.Equal(t, "complete", string(readFile(t, path)))
}

func TestDestinationRejectsSymlinksDirectoriesAndDanglingLinks(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "original")
	require.NoError(t, os.WriteFile(original, []byte("original"), 0o600))
	for _, slot := range []player_save.SlotID{"", player_save.Slot01} {
		for _, kind := range []string{"directory", "symlink", "dangling"} {
			t.Run(string(slot)+kind, func(t *testing.T) {
				parent := t.TempDir()
				path := filepath.Join(parent, "import.json")
				if slot != "" {
					path = filepath.Join(parent, string(slot)+".json")
				}
				if kind == "directory" {
					require.NoError(t, os.Mkdir(path, 0o700))
				} else {
					target := original
					if kind == "dangling" {
						target += "-missing"
					}
					if err := os.Symlink(target, path); err != nil {
						t.Skipf("symlinks unavailable: %v", err)
					}
				}
				_, err := inspectDestination(path, slot, true)
				require.ErrorIs(t, err, repositoryfs.ErrNotRegular)
			})
		}
	}
	require.Equal(t, "original", string(readFile(t, original)))
}

func TestStandaloneDisappearedRevisionIsNotRecreated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "import.json")
	require.NoError(t, os.WriteFile(path, []byte("old"), 0o600))
	reviewed, err := inspectDestination(path, "", true)
	require.NoError(t, err)
	require.NoError(t, os.Remove(path))
	require.ErrorIs(t, publishFile(context.Background(), path, reviewed.revision, []byte("replacement"), repositoryfs.SyncDirectory), errStale)
	_, err = os.Stat(path)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}
