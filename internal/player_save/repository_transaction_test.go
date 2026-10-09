package player_save

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
	"github.com/stretchr/testify/require"
)

func TestPlayerSlotProcessContender(t *testing.T) {
	root := os.Getenv("PLAYER_SLOT_PROCESS_ROOT")
	if root == "" {
		return
	}
	repository := newTestRepository(t, root, nil)
	fmt.Println("ready")
	_, err := repository.Save(Slot01, testEnvelope(testTime(3)), os.Getenv("PLAYER_SLOT_PROCESS_REVISION"))
	if os.Getenv("PLAYER_SLOT_PROCESS_STALE") == "true" {
		require.ErrorIs(t, err, ErrStaleRevision)
	} else {
		require.NoError(t, err)
	}
}

func startSlotProcess(t *testing.T, root, revision string, stale bool) <-chan error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestPlayerSlotProcessContender$")
	command.Env = append(os.Environ(), "PLAYER_SLOT_PROCESS_ROOT="+root, "PLAYER_SLOT_PROCESS_REVISION="+revision, fmt.Sprintf("PLAYER_SLOT_PROCESS_STALE=%t", stale))
	stdout, err := command.StdoutPipe()
	require.NoError(t, err)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	require.NoError(t, command.Start())
	scanner := bufio.NewScanner(stdout)
	ready := false
	for scanner.Scan() {
		if scanner.Text() == "ready" {
			ready = true
			break
		}
	}
	require.True(t, ready, "contender did not start")
	result := make(chan error, 1)
	go func() {
		var output bytes.Buffer
		for scanner.Scan() {
			fmt.Fprintln(&output, scanner.Text())
		}
		if err := command.Wait(); err != nil {
			result <- fmt.Errorf("contender: %w: %s %s", err, output.String(), stderr.String())
			return
		}
		result <- nil
	}()
	return result
}

func TestPlayerSlotProcessRevisionCheckWaitsForPublication(t *testing.T) {
	repository := newTestRepository(t, t.TempDir(), nil)
	var contender <-chan error
	repository.syncDirectory = func(root string) error {
		contender = startSlotProcess(t, root, "", true)
		select {
		case err := <-contender:
			t.Fatalf("contender escaped transaction lock: %v", err)
		case <-time.After(100 * time.Millisecond):
		}
		return repositoryfs.SyncDirectory(root)
	}
	_, err := repository.Save(Slot01, testEnvelope(testTime(1)), "")
	require.NoError(t, err)
	require.NoError(t, <-contender)
}

func TestPlayerSlotProcessWaitsThroughFailureRollback(t *testing.T) {
	for _, operation := range []string{"create", "overwrite", "delete", "restore"} {
		t.Run(operation, func(t *testing.T) {
			repository := newTestRepository(t, t.TempDir(), nil)
			original, err := repository.Save(Slot01, testEnvelope(testTime(1)), "")
			require.NoError(t, err)
			// Noncanonical bytes must survive rollback exactly, not re-encoding.
			raw := append([]byte(" \n"), original.rawData...)
			require.NoError(t, os.WriteFile(repository.path(Slot01), raw, fileMode))
			original, err = repository.Load(Slot01)
			require.NoError(t, err)
			if operation == "create" {
				require.NoError(t, repository.Delete(Slot01, original.Revision))
				original.Revision = ""
			}
			lockPath := filepath.Join(repository.root, repositoryfs.ReferenceLockName)
			lockBefore, err := os.Stat(lockPath)
			require.NoError(t, err)
			var contender <-chan error
			calls := 0
			repository.syncDirectory = func(root string) error {
				calls++
				if calls == 1 {
					contender = startSlotProcess(t, root, original.Revision, false)
					select {
					case err := <-contender:
						t.Fatalf("contender escaped rollback lock: %v", err)
					case <-time.After(100 * time.Millisecond):
					}
					return errors.New("injected directory sync failure")
				}
				if operation == "create" {
					_, err := os.Stat(repository.path(Slot01))
					require.ErrorIs(t, err, os.ErrNotExist)
				} else {
					restored, err := os.ReadFile(repository.path(Slot01))
					require.NoError(t, err)
					require.Equal(t, raw, restored)
				}
				return repositoryfs.SyncDirectory(root)
			}
			switch operation {
			case "delete":
				err = repository.Delete(Slot01, original.Revision)
			case "restore":
				err = repository.Restore(original, original.Revision)
			default:
				_, err = repository.Save(Slot01, testEnvelope(testTime(2)), original.Revision)
			}
			require.ErrorContains(t, err, "injected directory sync failure")
			require.Equal(t, 2, calls)
			require.NoError(t, <-contender)
			final, err := repository.Load(Slot01)
			require.NoError(t, err)
			require.Equal(t, testTime(3), final.Envelope.Metadata.CreatedAt)
			lockAfter, err := os.Stat(lockPath)
			require.NoError(t, err)
			require.True(t, os.SameFile(lockBefore, lockAfter), "lock inode changed")
		})
	}
}
