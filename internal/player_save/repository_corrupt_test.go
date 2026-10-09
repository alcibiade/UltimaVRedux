package player_save

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/repositoryfs"
	"github.com/bradhannah/UltimaVRedux/internal/save_snapshot"
	"github.com/stretchr/testify/require"
)

func corruptSlotFixtures(t *testing.T) map[string][]byte {
	t.Helper()
	canonical, err := save_snapshot.EncodeCanonical(testEnvelope(testTime(1)))
	require.NoError(t, err)
	invalid := testEnvelope(testTime(1))
	invalid.Snapshot.World.Calendar.Month = -1
	invalidSnapshot, err := json.Marshal(invalid)
	require.NoError(t, err)
	return map[string][]byte{
		"old envelope":     bytes.Replace(canonical, []byte(`"envelopeVersion": 2`), []byte(`"envelopeVersion": 1`), 1),
		"malformed JSON":   []byte("not JSON"),
		"truncated":        canonical[:len(canonical)/2],
		"zero bytes":       {},
		"invalid snapshot": invalidSnapshot,
	}
}

func TestRepositoryCorruptRecordsAreInspectableAndRevisionSafe(t *testing.T) {
	for name, data := range corruptSlotFixtures(t) {
		t.Run(name, func(t *testing.T) {
			repository := newTestRepository(t, t.TempDir(), func(save_snapshot.SaveSnapshot) error {
				t.Fatal("corrupt snapshot reached compatibility validation")
				return nil
			})
			path := repository.path(Slot01)
			require.NoError(t, os.WriteFile(path, data, fileMode))
			before, err := os.Stat(path)
			require.NoError(t, err)
			record, err := repository.Load(Slot01)
			wantError := save_snapshot.ErrInvalidEnvelope
			if name == "invalid snapshot" {
				wantError = save_snapshot.ErrInvalidSnapshot
			}
			require.ErrorIs(t, err, wantError)
			require.Equal(t, Slot01, record.Slot)
			require.Equal(t, repositoryfs.ContentRevision(data), record.Revision)
			require.Equal(t, record.Revision, record.rawRevision)
			require.True(t, bytes.Equal(data, record.rawData))
			require.Empty(t, record.Envelope)
			summary := repository.List()[0]
			require.Equal(t, SlotCorrupt, summary.Status)
			require.Equal(t, record.Revision, summary.Revision)
			require.Empty(t, summary.Metadata)
			require.NotEmpty(t, summary.Problem)
			_, err = repository.Latest()
			require.ErrorIs(t, err, ErrSlotNotFound)
			after, err := os.Stat(path)
			require.NoError(t, err)
			require.True(t, os.SameFile(before, after))
			require.Equal(t, before.ModTime(), after.ModTime())
			entries, err := os.ReadDir(repository.root)
			require.NoError(t, err)
			require.Equal(t, []string{"slot-01.json"}, entryNames(entries))
			for _, revision := range []string{"", "stale"} {
				_, err = repository.Save(Slot01, testEnvelope(testTime(2)), revision)
				require.ErrorIs(t, err, ErrStaleRevision)
				require.ErrorIs(t, repository.Delete(Slot01, revision), ErrStaleRevision)
			}
			unchanged, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, data, unchanged)
			changed := append(append([]byte{}, data...), ' ')
			require.NoError(t, os.WriteFile(path, changed, fileMode))
			_, err = repository.Save(Slot01, testEnvelope(testTime(2)), record.Revision)
			require.ErrorIs(t, err, ErrStaleRevision)
			require.ErrorIs(t, repository.Delete(Slot01, record.Revision), ErrStaleRevision)
			require.NoError(t, os.WriteFile(path, data, fileMode))
			replacement, err := repository.Save(Slot01, testEnvelope(testTime(2)), record.Revision)
			require.NoError(t, err)
			require.ErrorIs(t, repository.Restore(record, record.Revision), ErrStaleRevision)
			require.NoError(t, repository.Restore(record, replacement.Revision))
			restored, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, data, restored)
			require.NoError(t, repository.Delete(Slot01, record.Revision))
			_, err = os.Stat(path)
			require.ErrorIs(t, err, os.ErrNotExist)
		})
	}
}

func TestRepositoryCorruptMutationSyncFailureRestoresExactBytes(t *testing.T) {
	for name, data := range corruptSlotFixtures(t) {
		for _, operation := range []string{"save", "delete", "restore"} {
			t.Run(name+"/"+operation, func(t *testing.T) {
				repository := newTestRepository(t, t.TempDir(), nil)
				require.NoError(t, os.WriteFile(repository.path(Slot01), data, fileMode))
				record, err := repository.Load(Slot01)
				require.Error(t, err)
				syncError := errors.New("injected sync failure")
				calls := 0
				repository.syncDirectory = func(string) error {
					calls++
					if calls == 1 {
						return syncError
					}
					return nil
				}
				switch operation {
				case "save":
					_, err = repository.Save(Slot01, testEnvelope(testTime(2)), record.Revision)
				case "delete":
					err = repository.Delete(Slot01, record.Revision)
				case "restore":
					err = repository.Restore(record, record.Revision)
				}
				require.ErrorIs(t, err, syncError)
				require.Equal(t, 2, calls)
				restored, err := os.ReadFile(repository.path(Slot01))
				require.NoError(t, err)
				require.Equal(t, data, restored)
			})
		}
	}
}

func TestRepositoryRestoreRejectsMissingOrMismatchedPrivateRevision(t *testing.T) {
	repository := newTestRepository(t, t.TempDir(), nil)
	for _, record := range []Record{
		{Slot: Slot01, Revision: repositoryfs.ContentRevision(nil)},
		{Slot: Slot01, rawData: []byte("corrupt"), rawRevision: repositoryfs.ContentRevision(nil)},
	} {
		require.ErrorIs(t, repository.Restore(record, ""), errInvalidRestoreRecord)
	}
	entries, err := os.ReadDir(repository.root)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestRepositoryUnavailableSlotsHaveNoMutationRevision(t *testing.T) {
	for _, kind := range []string{"directory", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			repository := newTestRepository(t, t.TempDir(), nil)
			saved, err := repository.Save(Slot02, testEnvelope(testTime(1)), "")
			require.NoError(t, err)
			if kind == "directory" {
				require.NoError(t, os.Mkdir(repository.path(Slot01), rootMode))
			} else {
				require.NoError(t, os.Symlink(repository.path(Slot02), repository.path(Slot01)))
			}
			summary := repository.List()[0]
			require.Equal(t, SlotUnavailable, summary.Status)
			require.Empty(t, summary.Revision)
			record, err := repository.Load(Slot01)
			require.Error(t, err)
			require.Empty(t, record)
			for _, revision := range []string{"", saved.Revision} {
				_, err = repository.Save(Slot01, testEnvelope(testTime(2)), revision)
				require.Error(t, err)
				require.Error(t, repository.Delete(Slot01, revision))
				saved.Slot = Slot01
				require.Error(t, repository.Restore(saved, revision))
			}
			require.Equal(t, SlotUnavailable, repository.List()[0].Status)
		})
	}
}

func TestRepositoryConcurrentCorruptOverwritesHaveOneWinner(t *testing.T) {
	repository := newTestRepository(t, t.TempDir(), nil)
	require.NoError(t, os.WriteFile(repository.path(Slot01), nil, fileMode))
	revision := repository.List()[0].Revision
	require.NotEmpty(t, revision)
	start := make(chan struct{})
	results := make(chan error, 2)
	for day := 1; day <= 2; day++ {
		contender := newTestRepository(t, repository.root, nil)
		envelope := testEnvelope(testTime(day))
		go func() {
			<-start
			_, err := contender.Save(Slot01, envelope, revision)
			results <- err
		}()
	}
	close(start)
	first, second := <-results, <-results
	if first != nil {
		first, second = second, first
	}
	require.NoError(t, first)
	require.ErrorIs(t, second, ErrStaleRevision)
	_, err := repository.Load(Slot01)
	require.NoError(t, err)
}
